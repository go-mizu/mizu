package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/bot"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/channel/telegram"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/session"
	"github.com/go-mizu/mizu/blueprints/bot/store/sqlite"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

// runAgent sends a message through the bot engine and prints the response.
// This is the main "send a message to the AI" command.
func runAgent() error {
	fs := flag.NewFlagSet("agent", flag.ExitOnError)
	message := fs.String("m", "", "Message body (required)")
	messageLong := fs.String("message", "", "Message body (required)")
	to := fs.String("t", "", "Peer ID to derive session key")
	fs.String("to", "", "Peer ID")
	fs.String("agent", "main", "Agent ID")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	fs.Bool("local", true, "Run locally (default)")
	fs.Parse(os.Args[2:])

	// Resolve message from -m or --message.
	msg := *message
	if msg == "" {
		msg = *messageLong
	}
	if msg == "" {
		return fmt.Errorf("message (-m) is required")
	}

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

	peerID := "cli-user"
	if *to != "" {
		peerID = *to
	}

	ctx := context.Background()
	inMsg := &types.InboundMessage{
		ChannelType: "webhook",
		ChannelID:   "cli",
		PeerID:      peerID,
		PeerName:    "CLI",
		Content:     msg,
		Origin:      "dm",
	}

	resp, err := b.HandleMessage(ctx, inMsg)
	if err != nil {
		return fmt.Errorf("handle message: %w", err)
	}

	if *jsonOut {
		result := map[string]any{
			"status": "ok",
			"result": map[string]any{
				"payloads": []map[string]any{
					{"text": resp},
				},
			},
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	} else {
		fmt.Println(resp)
	}
	return nil
}

// runSessions lists sessions from the file-based session store with fallback to SQLite.
func runSessions() error {
	fs := flag.NewFlagSet("sessions", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	active := fs.Int("active", 0, "Only show sessions updated within N minutes")
	fs.Parse(os.Args[2:])

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Try file-based session store first.
	sessDir := filepath.Join(cfg.DataDir, "agents", "main", "sessions")
	store, err := session.NewFileStore(sessDir)
	if err != nil {
		// Fallback to SQLite.
		return runSessionsSQLite(cfg, *jsonOut)
	}

	sessions, err := store.ListSessions()
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	// Filter by active minutes if specified.
	if *active > 0 {
		cutoff := time.Now().Add(-time.Duration(*active) * time.Minute).UnixMilli()
		var filtered []session.SessionInfo
		for _, s := range sessions {
			if s.Entry.UpdatedAt >= cutoff {
				filtered = append(filtered, s)
			}
		}
		sessions = filtered
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(sessions)
		return nil
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	// Table output matching OpenClaw format.
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KIND\tKEY\tAGE\tMODEL\tTOKENS\tSTATUS")
	for _, s := range sessions {
		kind := s.Entry.ChatType
		if kind == "" {
			kind = "direct"
		}
		age := formatAge(s.Entry.UpdatedAt)
		model := s.Entry.Model
		if model == "" {
			model = "-"
		}
		tokens := fmt.Sprintf("%dk/%dk", s.Entry.InputTokens/1000, s.Entry.OutputTokens/1000)
		if s.Entry.InputTokens == 0 && s.Entry.OutputTokens == 0 {
			tokens = "-"
		}
		status := s.Entry.Status
		if status == "" {
			status = "active"
		}
		key := s.Key
		if len(key) > 40 {
			key = key[:40] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", kind, key, age, model, tokens, status)
	}
	w.Flush()
	return nil
}

// runSessionsSQLite lists sessions from the SQLite store (fallback).
func runSessionsSQLite(cfg *config.Config, jsonOut bool) error {
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

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(sessions)
		return nil
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

// runHistory shows transcript from JSONL file.
func runHistory(sessionID string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	sessDir := filepath.Join(cfg.DataDir, "agents", "main", "sessions")
	store, err := session.NewFileStore(sessDir)
	if err != nil {
		return fmt.Errorf("open session store: %w", err)
	}

	// If no session ID, find most recent.
	if sessionID == "" {
		sessions, err := store.ListSessions()
		if err != nil {
			return fmt.Errorf("list sessions: %w", err)
		}
		if len(sessions) == 0 {
			fmt.Println("No sessions found.")
			return nil
		}
		sessionID = sessions[0].Entry.SessionID
		fmt.Printf("Showing history for session %s (%s)\n\n", shortID(sessionID), sessions[0].Key)
	}

	entries, err := store.ReadTranscript(sessionID)
	if err != nil {
		return fmt.Errorf("read transcript: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No entries in this session.")
		return nil
	}

	for _, entry := range entries {
		switch entry.Type {
		case "session":
			fmt.Printf("[Session] %s (v%d)\n", entry.ID, entry.Version)
		case "message":
			if entry.Message != nil {
				ts := entry.Timestamp
				if len(ts) > 16 {
					ts = ts[11:16] // Extract HH:MM
				}
				content := fmt.Sprintf("%v", entry.Message.Content)
				if len(content) > 500 {
					content = content[:500] + "..."
				}
				fmt.Printf("[%s] %s: %s\n", ts, entry.Message.Role, content)
			}
		case "model_change":
			fmt.Printf("[Model] Changed to: %s\n", entry.Model)
		}
	}
	return nil
}

// runMessageSend sends a message directly via a channel (Telegram).
func runMessageSend() error {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	target := fs.String("t", "", "Delivery target (required)")
	targetLong := fs.String("target", "", "Delivery target")
	message := fs.String("m", "", "Message body (required)")
	messageLong := fs.String("message", "", "Message body")
	ch := fs.String("channel", "telegram", "Delivery channel")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	fs.Parse(os.Args[3:]) // skip "openbot message send"

	t := *target
	if t == "" {
		t = *targetLong
	}
	m := *message
	if m == "" {
		m = *messageLong
	}

	if t == "" {
		return fmt.Errorf("target (-t) is required")
	}
	if m == "" {
		return fmt.Errorf("message (-m) is required")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if *ch == "telegram" {
		if cfg.Telegram.BotToken == "" {
			return fmt.Errorf("no Telegram bot token configured")
		}
		// Send directly via Telegram API.
		tgCfg := types.TelegramConfig{BotToken: cfg.Telegram.BotToken}
		tgJSON, _ := json.Marshal(tgCfg)
		drv, err := telegram.NewDriver(string(tgJSON), nil)
		if err != nil {
			return fmt.Errorf("create telegram driver: %w", err)
		}
		outMsg := &types.OutboundMessage{
			ChannelType: types.ChannelTelegram,
			ChannelID:   t,
			PeerID:      t,
			Content:     m,
		}
		if err := drv.Send(context.Background(), outMsg); err != nil {
			return fmt.Errorf("send: %w", err)
		}
		if *jsonOut {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			enc.Encode(map[string]any{"status": "sent", "target": t, "channel": *ch})
		} else {
			fmt.Printf("Message sent to %s via %s\n", t, *ch)
		}
	} else {
		return fmt.Errorf("unsupported channel: %s", *ch)
	}
	return nil
}

// runStatus shows bot status including file store info.
func runStatus() error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	fs.Parse(os.Args[2:])

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Count sessions from file store.
	sessDir := filepath.Join(cfg.DataDir, "agents", "main", "sessions")
	store, _ := session.NewFileStore(sessDir)
	var sessionCount int
	if store != nil {
		sessions, _ := store.ListSessions()
		sessionCount = len(sessions)
	}

	// Count messages from SQLite.
	sqlStore, err := openStore(cfg)
	var msgCount int
	if err == nil {
		defer sqlStore.Close()
		stats, err := sqlStore.Stats(context.Background())
		if err == nil {
			msgCount = stats.Messages
		}
	}

	if *jsonOut {
		result := map[string]any{
			"config":       config.DefaultConfigPath(),
			"database":     filepath.Join(cfg.DataDir, "bot.db"),
			"sessions":     filepath.Join(cfg.DataDir, "agents", "main", "sessions"),
			"workspace":    cfg.Workspace,
			"sessionCount": sessionCount,
			"messageCount": msgCount,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
		return nil
	}

	fmt.Println("OpenBot Status:")
	fmt.Printf("  Config:    %s\n", config.DefaultConfigPath())
	fmt.Printf("  Database:  %s\n", filepath.Join(cfg.DataDir, "bot.db"))
	fmt.Printf("  Sessions:  %s\n", filepath.Join(cfg.DataDir, "agents", "main", "sessions"))
	fmt.Printf("  Workspace: %s\n", cfg.Workspace)
	fmt.Printf("  Sessions:  %d\n", sessionCount)
	fmt.Printf("  Messages:  %d\n", msgCount)
	return nil
}

// formatAge returns a human-readable age string from a millisecond timestamp.
func formatAge(updatedAtMs int64) string {
	if updatedAtMs == 0 {
		return "unknown"
	}
	d := time.Since(time.UnixMilli(updatedAtMs))
	if d < time.Minute {
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
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
