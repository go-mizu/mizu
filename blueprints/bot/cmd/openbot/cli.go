package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/bot"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/store/sqlite"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// runSessions lists all active sessions from the database.
func runSessions() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer s.Close()

	ctx := context.Background()
	sessions, err := s.ListSessions(ctx)
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPeer\tChannel\tOrigin\tStatus\tLast Active")
	for _, sess := range sessions {
		ago := time.Since(sess.UpdatedAt).Truncate(time.Second)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s ago\n",
			shortID(sess.ID), sess.DisplayName, sess.ChannelType, sess.Origin, sess.Status, ago)
	}
	w.Flush()
	return nil
}

// runHistory shows messages for a session.
// If sessionID is empty, uses the most recent active session.
func runHistory(sessionID string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer s.Close()

	ctx := context.Background()

	// If no session ID given, find the most recent active session.
	if sessionID == "" {
		sessions, err := s.ListSessions(ctx)
		if err != nil {
			return fmt.Errorf("list sessions: %w", err)
		}
		var latest *types.Session
		for i := range sessions {
			if sessions[i].Status == "active" {
				if latest == nil || sessions[i].UpdatedAt.After(latest.UpdatedAt) {
					latest = &sessions[i]
				}
			}
		}
		if latest == nil {
			fmt.Println("No active sessions found.")
			return nil
		}
		sessionID = latest.ID
		fmt.Printf("Showing history for session %s (%s)\n\n", shortID(sessionID), latest.DisplayName)
	}

	messages, err := s.ListMessages(ctx, sessionID, 50)
	if err != nil {
		return fmt.Errorf("list messages: %w", err)
	}

	if len(messages) == 0 {
		fmt.Println("No messages in this session.")
		return nil
	}

	for _, msg := range messages {
		timeStr := msg.CreatedAt.Format("15:04")
		role := msg.Role
		content := msg.Content
		// Truncate long messages for display.
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		fmt.Printf("[%s] %s: %s\n", timeStr, role, content)
	}
	return nil
}

// runSend sends a message through the bot engine and prints the response.
func runSend(message string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	provider := llm.NewClaude()

	b, err := bot.New(cfg, provider)
	if err != nil {
		return fmt.Errorf("create bot: %w", err)
	}
	defer b.Close()

	ctx := context.Background()
	msg := &types.InboundMessage{
		ChannelType: "webhook",
		ChannelID:   "cli",
		PeerID:      "cli-user",
		PeerName:    "CLI",
		Content:     message,
		Origin:      "dm",
	}

	resp, err := b.HandleMessage(ctx, msg)
	if err != nil {
		return fmt.Errorf("handle message: %w", err)
	}

	fmt.Println(resp)
	return nil
}

// runStatus shows bot status information.
func runStatus() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	s, err := openStore(cfg)
	if err != nil {
		return err
	}
	defer s.Close()

	ctx := context.Background()
	stats, err := s.Stats(ctx)
	if err != nil {
		return fmt.Errorf("get stats: %w", err)
	}

	fmt.Println("OpenBot Status:")
	fmt.Printf("  Config:    %s\n", config.DefaultConfigPath())
	fmt.Printf("  Database:  %s\n", filepath.Join(cfg.DataDir, "bot.db"))
	fmt.Printf("  Workspace: %s\n", cfg.Workspace)
	fmt.Printf("  Sessions:  %d\n", stats.Sessions)
	fmt.Printf("  Messages:  %d\n", stats.Messages)
	fmt.Printf("  Agents:    %d\n", stats.Agents)
	return nil
}

// --- helpers ---

func loadConfig() (*config.Config, error) {
	openbotDir := config.DefaultConfigDir()
	openclawDir := filepath.Join(os.Getenv("HOME"), ".openclaw")

	if err := config.EnsureConfig(openbotDir, openclawDir); err != nil {
		return nil, fmt.Errorf("config init: %w", err)
	}

	return config.LoadFromFile(config.DefaultConfigPath())
}

func openStore(cfg *config.Config) (*sqlite.Store, error) {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	dbPath := filepath.Join(cfg.DataDir, "bot.db")
	s, err := sqlite.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	ctx := context.Background()
	if err := s.Ensure(ctx); err != nil {
		s.Close()
		return nil, fmt.Errorf("ensure schema: %w", err)
	}
	return s, nil
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
