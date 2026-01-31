package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/channel"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func init() {
	channel.Register(types.ChannelTelegram, func(config string, handler channel.MessageHandler) (channel.Driver, error) {
		return NewDriver(config, handler)
	})
}

// NewDriver creates a Telegram driver directly (for standalone use).
func NewDriver(config string, handler channel.MessageHandler) (*Driver, error) {
	var cfg types.TelegramConfig
	if err := json.Unmarshal([]byte(config), &cfg); err != nil {
		return nil, fmt.Errorf("parse telegram config: %w", err)
	}
	return &Driver{config: cfg, handler: handler, client: &http.Client{Timeout: 30 * time.Second}}, nil
}

// Driver implements channel.Driver for Telegram via Bot API.
type Driver struct {
	config  types.TelegramConfig
	handler channel.MessageHandler
	client  *http.Client
	status  string
	cancel  context.CancelFunc
}

func (d *Driver) Type() types.ChannelType { return types.ChannelTelegram }
func (d *Driver) Status() string {
	if d.status == "" {
		return "disconnected"
	}
	return d.status
}

// Connect starts long-polling for updates from the Telegram Bot API.
func (d *Driver) Connect(ctx context.Context) error {
	ctx, d.cancel = context.WithCancel(ctx)
	d.status = "connected"

	go d.poll(ctx)
	return nil
}

func (d *Driver) Disconnect(_ context.Context) error {
	if d.cancel != nil {
		d.cancel()
	}
	d.status = "disconnected"
	return nil
}

const telegramMaxLength = 4096

// Send sends a text message via Telegram Bot API.
// It prefers ChannelID (chat ID) over PeerID for targeting, and splits
// messages that exceed Telegram's 4096-character limit.
func (d *Driver) Send(ctx context.Context, msg *types.OutboundMessage) error {
	target := msg.ChannelID
	if target == "" {
		target = msg.PeerID
	}

	chatID, err := strconv.ParseInt(target, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid telegram chat ID %q: %w", target, err)
	}

	chunks := splitMessage(msg.Content)
	for _, chunk := range chunks {
		if err := d.sendText(ctx, chatID, chunk); err != nil {
			return err
		}
	}
	return nil
}

// sendText sends a single text chunk with Markdown parse mode.
// If the Telegram API rejects the Markdown, it retries as plain text.
func (d *Driver) sendText(ctx context.Context, chatID int64, text string) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", d.config.BotToken)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram sendMessage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		// If Markdown parse fails, retry without parse_mode.
		if resp.StatusCode == 400 && bytes.Contains(respBody, []byte("parse")) {
			return d.sendPlain(ctx, chatID, text)
		}
		return fmt.Errorf("telegram sendMessage %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

// sendPlain sends a single text chunk without any parse mode.
func (d *Driver) sendPlain(ctx context.Context, chatID int64, text string) error {
	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}

	body, _ := json.Marshal(payload)
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", d.config.BotToken)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram sendMessage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram sendMessage %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

// splitMessage splits content into chunks that fit within Telegram's limit.
func splitMessage(content string) []string {
	if len(content) <= telegramMaxLength {
		return []string{content}
	}

	var chunks []string
	for len(content) > 0 {
		if len(content) <= telegramMaxLength {
			chunks = append(chunks, content)
			break
		}
		// Find a good split point (newline, space, or hard limit).
		splitAt := telegramMaxLength
		if idx := strings.LastIndex(content[:splitAt], "\n"); idx > telegramMaxLength/2 {
			splitAt = idx + 1
		} else if idx := strings.LastIndex(content[:splitAt], " "); idx > telegramMaxLength/2 {
			splitAt = idx + 1
		}
		chunks = append(chunks, content[:splitAt])
		content = content[splitAt:]
	}
	return chunks
}

func (d *Driver) poll(ctx context.Context) {
	var offset int64

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", d.config.BotToken, offset)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := d.client.Do(req)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		var result struct {
			OK     bool `json:"ok"`
			Result []struct {
				UpdateID int64 `json:"update_id"`
				Message  *struct {
					MessageID int64 `json:"message_id"`
					From      struct {
						ID        int64  `json:"id"`
						FirstName string `json:"first_name"`
						Username  string `json:"username"`
					} `json:"from"`
					Chat struct {
						ID   int64  `json:"id"`
						Type string `json:"type"`
					} `json:"chat"`
					Text string `json:"text"`
				} `json:"message"`
			} `json:"result"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		for _, update := range result.Result {
			offset = update.UpdateID + 1
			if update.Message == nil || update.Message.Text == "" {
				continue
			}

			origin := "dm"
			if update.Message.Chat.Type == "group" || update.Message.Chat.Type == "supergroup" {
				origin = "group"
			}

			peerName := update.Message.From.FirstName
			if update.Message.From.Username != "" {
				peerName = update.Message.From.Username
			}

			var groupID string
			if origin == "group" {
				groupID = strconv.FormatInt(update.Message.Chat.ID, 10)
			}

			msg := &types.InboundMessage{
				ChannelType: types.ChannelTelegram,
				ChannelID:   strconv.FormatInt(update.Message.Chat.ID, 10),
				PeerID:      strconv.FormatInt(update.Message.From.ID, 10),
				PeerName:    peerName,
				Content:     update.Message.Text,
				Origin:      origin,
				GroupID:     groupID,
			}

			d.handler(ctx, msg)
		}
	}
}
