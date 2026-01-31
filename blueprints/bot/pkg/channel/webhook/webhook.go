package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/channel"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func init() {
	channel.Register(types.ChannelWebhook, func(config string, handler channel.MessageHandler) (channel.Driver, error) {
		var cfg types.WebhookConfig
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return nil, fmt.Errorf("parse webhook config: %w", err)
		}
		return &Driver{config: cfg, handler: handler, client: &http.Client{Timeout: 30 * time.Second}}, nil
	})
}

// Driver implements channel.Driver for generic webhooks.
type Driver struct {
	config  types.WebhookConfig
	handler channel.MessageHandler
	client  *http.Client
	status  string
}

func (d *Driver) Type() types.ChannelType { return types.ChannelWebhook }
func (d *Driver) Status() string {
	if d.status == "" {
		return "disconnected"
	}
	return d.status
}

func (d *Driver) Connect(_ context.Context) error {
	d.status = "connected"
	return nil
}

func (d *Driver) Disconnect(_ context.Context) error {
	d.status = "disconnected"
	return nil
}

// Send posts the outbound message to the configured endpoint.
func (d *Driver) Send(ctx context.Context, msg *types.OutboundMessage) error {
	if d.config.Endpoint == "" {
		return nil // No outbound endpoint configured
	}

	payload := map[string]any{
		"peerId":  msg.PeerID,
		"content": msg.Content,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", d.config.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if d.config.Secret != "" {
		req.Header.Set("X-Webhook-Secret", d.config.Secret)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook send failed: %d", resp.StatusCode)
	}
	return nil
}

// HandleInbound processes an inbound webhook request.
// This is called by the HTTP handler when a webhook request arrives.
func (d *Driver) HandleInbound(ctx context.Context, peerID, peerName, content string) error {
	msg := &types.InboundMessage{
		ChannelType: types.ChannelWebhook,
		PeerID:      peerID,
		PeerName:    peerName,
		Content:     content,
		Origin:      "webhook",
	}
	return d.handler(ctx, msg)
}
