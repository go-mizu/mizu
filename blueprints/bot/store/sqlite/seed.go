package sqlite

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func (s *Store) SeedData(ctx context.Context) error {
	// Seed agents
	fmt.Println("  Creating agents...")
	agents := []types.Agent{
		{
			ID:           "main",
			Name:         "Main Assistant",
			Model:        "claude-sonnet-4-20250514",
			SystemPrompt: "You are a helpful AI assistant. Be concise, friendly, and accurate. Help users with their questions and tasks.",
			Workspace:    "~/.bot/workspace/main",
			MaxTokens:    4096,
			Temperature:  0.7,
			Status:       "active",
		},
		{
			ID:           "coder",
			Name:         "Code Assistant",
			Model:        "claude-sonnet-4-20250514",
			SystemPrompt: "You are a coding assistant specialized in helping with programming tasks. Write clean, well-documented code. Explain your solutions clearly.",
			Workspace:    "~/.bot/workspace/coder",
			MaxTokens:    8192,
			Temperature:  0.3,
			Status:       "active",
		},
		{
			ID:           "writer",
			Name:         "Writing Assistant",
			Model:        "claude-sonnet-4-20250514",
			SystemPrompt: "You are a writing assistant. Help users with drafting, editing, and improving their written content. Focus on clarity, style, and tone.",
			Workspace:    "~/.bot/workspace/writer",
			MaxTokens:    4096,
			Temperature:  0.8,
			Status:       "active",
		},
	}

	for _, a := range agents {
		if err := s.CreateAgent(ctx, &a); err != nil {
			return fmt.Errorf("create agent %s: %w", a.ID, err)
		}
	}

	// Seed channels
	fmt.Println("  Creating channels...")

	tgConfig, _ := json.Marshal(types.TelegramConfig{
		BotToken:     "YOUR_TELEGRAM_BOT_TOKEN",
		DMPolicy:     "pairing",
		HistoryLimit: 50,
	})
	dcConfig, _ := json.Marshal(types.DiscordConfig{
		BotToken: "YOUR_DISCORD_BOT_TOKEN",
		DMPolicy: "pairing",
	})
	mmConfig, _ := json.Marshal(types.MattermostConfig{
		ServerURL: "https://mattermost.example.com",
		BotToken:  "YOUR_MATTERMOST_TOKEN",
		TeamID:    "team-1",
	})
	whConfig, _ := json.Marshal(types.WebhookConfig{
		Secret:   "webhook-secret-key",
		Endpoint: "/api/webhook/incoming",
	})

	channels := []types.Channel{
		{ID: "tg-1", Type: types.ChannelTelegram, Name: "Telegram Bot", Config: string(tgConfig), Status: "disconnected"},
		{ID: "dc-1", Type: types.ChannelDiscord, Name: "Discord Bot", Config: string(dcConfig), Status: "disconnected"},
		{ID: "mm-1", Type: types.ChannelMattermost, Name: "Mattermost Bot", Config: string(mmConfig), Status: "disconnected"},
		{ID: "wh-1", Type: types.ChannelWebhook, Name: "Webhook Endpoint", Config: string(whConfig), Status: "connected"},
	}

	for _, c := range channels {
		if err := s.CreateChannel(ctx, &c); err != nil {
			return fmt.Errorf("create channel %s: %w", c.ID, err)
		}
	}

	// Seed bindings (routing rules)
	fmt.Println("  Creating bindings...")

	bindings := []types.Binding{
		// Default: all messages go to main agent
		{ID: "bind-default", AgentID: "main", ChannelType: "*", ChannelID: "*", PeerID: "*", Priority: 0},
		// Telegram messages go to main agent
		{ID: "bind-tg", AgentID: "main", ChannelType: "telegram", ChannelID: "*", PeerID: "*", Priority: 10},
		// Discord messages go to coder agent
		{ID: "bind-dc", AgentID: "coder", ChannelType: "discord", ChannelID: "*", PeerID: "*", Priority: 10},
		// Mattermost goes to writer agent
		{ID: "bind-mm", AgentID: "writer", ChannelType: "mattermost", ChannelID: "*", PeerID: "*", Priority: 10},
		// Webhook goes to main agent
		{ID: "bind-wh", AgentID: "main", ChannelType: "webhook", ChannelID: "*", PeerID: "*", Priority: 10},
	}

	for _, b := range bindings {
		if err := s.CreateBinding(ctx, &b); err != nil {
			return fmt.Errorf("create binding %s: %w", b.ID, err)
		}
	}

	// Seed sample sessions and messages
	fmt.Println("  Creating sample sessions...")

	sessions := []struct {
		session  types.Session
		messages []types.Message
	}{
		{
			session: types.Session{
				ID: "sess-tg-1", AgentID: "main", ChannelID: "tg-1", ChannelType: "telegram",
				PeerID: "tg:123456789", DisplayName: "Alice", Origin: "dm", Status: "active",
			},
			messages: []types.Message{
				{ID: "msg-1", SessionID: "sess-tg-1", AgentID: "main", ChannelID: "tg-1", PeerID: "tg:123456789", Role: "user", Content: "Hello! What can you help me with?"},
				{ID: "msg-2", SessionID: "sess-tg-1", AgentID: "main", ChannelID: "tg-1", PeerID: "tg:123456789", Role: "assistant", Content: "Hi Alice! I can help you with questions, writing, research, coding, and more. What would you like to work on?"},
				{ID: "msg-3", SessionID: "sess-tg-1", AgentID: "main", ChannelID: "tg-1", PeerID: "tg:123456789", Role: "user", Content: "Can you explain how Go channels work?"},
				{ID: "msg-4", SessionID: "sess-tg-1", AgentID: "main", ChannelID: "tg-1", PeerID: "tg:123456789", Role: "assistant", Content: "Go channels are typed conduits for sending and receiving values between goroutines. They provide synchronization without explicit locks.\n\nBasic usage:\n```go\nch := make(chan int)    // unbuffered\nch <- 42               // send\nval := <-ch            // receive\n```\n\nBuffered channels allow a fixed number of values without blocking:\n```go\nch := make(chan int, 5) // buffer size 5\n```\n\nChannels are commonly used for goroutine coordination, fan-out/fan-in patterns, and pipeline processing."},
			},
		},
		{
			session: types.Session{
				ID: "sess-dc-1", AgentID: "coder", ChannelID: "dc-1", ChannelType: "discord",
				PeerID: "dc:987654321", DisplayName: "Bob", Origin: "dm", Status: "active",
			},
			messages: []types.Message{
				{ID: "msg-5", SessionID: "sess-dc-1", AgentID: "coder", ChannelID: "dc-1", PeerID: "dc:987654321", Role: "user", Content: "Help me write a function to reverse a linked list in Go"},
				{ID: "msg-6", SessionID: "sess-dc-1", AgentID: "coder", ChannelID: "dc-1", PeerID: "dc:987654321", Role: "assistant", Content: "Here's a function to reverse a singly linked list:\n\n```go\ntype ListNode struct {\n    Val  int\n    Next *ListNode\n}\n\nfunc reverseList(head *ListNode) *ListNode {\n    var prev *ListNode\n    curr := head\n    for curr != nil {\n        next := curr.Next\n        curr.Next = prev\n        prev = curr\n        curr = next\n    }\n    return prev\n}\n```\n\nThis iterates through the list, reversing each pointer. Time complexity is O(n), space is O(1)."},
			},
		},
		{
			session: types.Session{
				ID: "sess-wh-1", AgentID: "main", ChannelID: "wh-1", ChannelType: "webhook",
				PeerID: "webhook:api-client", DisplayName: "API Client", Origin: "webhook", Status: "active",
			},
			messages: []types.Message{
				{ID: "msg-7", SessionID: "sess-wh-1", AgentID: "main", ChannelID: "wh-1", PeerID: "webhook:api-client", Role: "user", Content: "Summarize the latest changes in Go 1.24"},
				{ID: "msg-8", SessionID: "sess-wh-1", AgentID: "main", ChannelID: "wh-1", PeerID: "webhook:api-client", Role: "assistant", Content: "Go 1.24 introduced several key changes including generic type aliases, improved test coverage tooling with `go test -covermode`, weak pointers via `runtime/weak`, and new `os.Root` for restricted filesystem operations. The release also brought FIPS 140-3 compliant cryptography and `tool` directives in `go.mod` for managing development tools."},
			},
		},
	}

	for _, ss := range sessions {
		s.db.ExecContext(ctx,
			`INSERT INTO sessions (id, agent_id, channel_id, channel_type, peer_id, display_name, origin, status, metadata, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, '{}', datetime('now'), datetime('now'))`,
			ss.session.ID, ss.session.AgentID, ss.session.ChannelID, ss.session.ChannelType, ss.session.PeerID, ss.session.DisplayName, ss.session.Origin, ss.session.Status,
		)
		for _, m := range ss.messages {
			if err := s.CreateMessage(ctx, &m); err != nil {
				return fmt.Errorf("create message %s: %w", m.ID, err)
			}
		}
	}

	return nil
}
