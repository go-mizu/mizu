package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/channel"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func init() {
	channel.Register(types.ChannelDiscord, func(config string, handler channel.MessageHandler) (channel.Driver, error) {
		var cfg types.DiscordConfig
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return nil, fmt.Errorf("parse discord config: %w", err)
		}
		return &Driver{config: cfg, handler: handler, client: &http.Client{Timeout: 30 * time.Second}}, nil
	})
}

const discordAPI = "https://discord.com/api/v10"

// Driver implements channel.Driver for Discord via HTTP API.
type Driver struct {
	config  types.DiscordConfig
	handler channel.MessageHandler
	client  *http.Client
	status  string
	cancel  context.CancelFunc
}

func (d *Driver) Type() types.ChannelType { return types.ChannelDiscord }
func (d *Driver) Status() string {
	if d.status == "" {
		return "disconnected"
	}
	return d.status
}

func (d *Driver) Connect(ctx context.Context) error {
	// Verify bot token by fetching current user
	req, err := http.NewRequestWithContext(ctx, "GET", discordAPI+"/users/@me", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+d.config.BotToken)

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord auth failed %d: %s", resp.StatusCode, body)
	}

	_, d.cancel = context.WithCancel(ctx)
	d.status = "connected"
	return nil
}

func (d *Driver) Disconnect(_ context.Context) error {
	if d.cancel != nil {
		d.cancel()
	}
	d.status = "disconnected"
	return nil
}

// Send sends a message to a Discord channel.
func (d *Driver) Send(ctx context.Context, msg *types.OutboundMessage) error {
	payload := map[string]any{
		"content": msg.Content,
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/channels/%s/messages", discordAPI, msg.PeerID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+d.config.BotToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("discord send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord send %d: %s", resp.StatusCode, respBody)
	}
	return nil
}
