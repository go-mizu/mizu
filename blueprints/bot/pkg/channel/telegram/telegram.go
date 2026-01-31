package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/channel"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func init() {
	channel.Register(types.ChannelTelegram, func(config string, handler channel.MessageHandler) (channel.Driver, error) {
		var cfg types.TelegramConfig
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return nil, fmt.Errorf("parse telegram config: %w", err)
		}
		return &Driver{config: cfg, handler: handler, client: &http.Client{Timeout: 30 * time.Second}}, nil
	})
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

// Send sends a text message via Telegram Bot API.
func (d *Driver) Send(ctx context.Context, msg *types.OutboundMessage) error {
	chatID, err := strconv.ParseInt(msg.PeerID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid telegram peer ID %q: %w", msg.PeerID, err)
	}

	payload := map[string]any{
		"chat_id": chatID,
		"text":    msg.Content,
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

			msg := &types.InboundMessage{
				ChannelType: types.ChannelTelegram,
				PeerID:      strconv.FormatInt(update.Message.From.ID, 10),
				PeerName:    peerName,
				Content:     update.Message.Text,
				Origin:      origin,
			}

			d.handler(ctx, msg)
		}
	}
}
