package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/go-mizu/mizu/blueprints/bot/pkg/bot"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/channel/telegram"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/config"
	"github.com/go-mizu/mizu/blueprints/bot/pkg/llm"
	"github.com/go-mizu/mizu/blueprints/bot/types"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Subcommand dispatch.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "agent":
			if err := runAgent(); err != nil {
				log.Fatal(err)
			}
			return
		case "sessions":
			if err := runSessions(); err != nil {
				log.Fatal(err)
			}
			return
		case "history":
			sessionID := ""
			if len(os.Args) > 2 {
				sessionID = os.Args[2]
			}
			if err := runHistory(sessionID); err != nil {
				log.Fatal(err)
			}
			return
		case "message":
			if len(os.Args) > 2 && os.Args[2] == "send" {
				if err := runMessageSend(); err != nil {
					log.Fatal(err)
				}
				return
			}
			printUsage()
			return
		case "status":
			if err := runStatus(); err != nil {
				log.Fatal(err)
			}
			return
		case "send":
			// Legacy compat: redirect to agent command.
			if len(os.Args) < 3 {
				log.Fatal("Usage: openbot send <message>")
			}
			msg := strings.Join(os.Args[2:], " ")
			// Rewrite args for agent command.
			os.Args = []string{os.Args[0], "agent", "-m", msg}
			if err := runAgent(); err != nil {
				log.Fatal(err)
			}
			return
		case "help":
			printUsage()
			return
		}
	}

	// Default: run the Telegram bot.

	// 1. Ensure config exists (clone from OpenClaw if needed).
	openbotDir := config.DefaultConfigDir()
	openclawDir := filepath.Join(os.Getenv("HOME"), ".openclaw")

	if err := config.EnsureConfig(openbotDir, openclawDir); err != nil {
		log.Printf("Config init: %v", err)
		log.Printf("Create %s/openbot.json manually or install OpenClaw first.", openbotDir)
		os.Exit(1)
	}

	// 2. Load config.
	cfg, err := config.LoadFromFile(config.DefaultConfigPath())
	if err != nil {
		log.Fatalf("Load config: %v", err)
	}

	if !cfg.Telegram.Enabled {
		log.Fatal("Telegram channel is disabled in config.")
	}
	if cfg.Telegram.BotToken == "" {
		log.Fatal("No Telegram bot token. Set TELEGRAM_API_KEY env var or configure channels.telegram.botToken in openbot.json.")
	}

	// 3. Create LLM provider.
	provider := llm.NewClaude()

	// 4. Create bot engine.
	b, err := bot.New(cfg, provider)
	if err != nil {
		log.Fatalf("Create bot: %v", err)
	}
	defer b.Close()

	// 5. Create Telegram driver.
	tgCfg := types.TelegramConfig{BotToken: cfg.Telegram.BotToken}
	tgJSON, _ := json.Marshal(tgCfg)

	// We need to reference the driver in the handler closure, so declare it first.
	var drv *telegram.Driver

	handler := func(ctx context.Context, msg *types.InboundMessage) error {
		resp, err := b.HandleMessage(ctx, msg)
		if err != nil {
			log.Printf("Handle message from %s (%s): %v", msg.PeerName, msg.PeerID, err)
			return nil // don't crash on message errors
		}

		if resp == "" {
			return nil
		}

		// Send response back via Telegram.
		outMsg := &types.OutboundMessage{
			ChannelType: types.ChannelTelegram,
			ChannelID:   msg.ChannelID,
			PeerID:      msg.PeerID,
			Content:     resp,
		}
		if err := drv.Send(ctx, outMsg); err != nil {
			log.Printf("Send reply to %s: %v", msg.PeerName, err)
		}
		return nil
	}

	drv, err = telegram.NewDriver(string(tgJSON), handler)
	if err != nil {
		log.Fatalf("Create Telegram driver: %v", err)
	}

	// 6. Start polling.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("OpenBot starting...")
	fmt.Printf("  Workspace: %s\n", cfg.Workspace)
	fmt.Printf("  DM Policy: %s\n", cfg.Telegram.DMPolicy)
	if len(cfg.Telegram.AllowFrom) > 0 {
		fmt.Printf("  Allow From: %v\n", cfg.Telegram.AllowFrom)
	}
	fmt.Println("  Connecting to Telegram...")

	if err := drv.Connect(ctx); err != nil {
		log.Fatalf("Connect to Telegram: %v", err)
	}
	fmt.Println("  Connected! Listening for messages...")
	fmt.Println("  Press Ctrl+C to stop.")

	// 7. Wait for shutdown signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh
	fmt.Println("\nShutting down...")
	cancel()
	drv.Disconnect(context.Background())
	fmt.Println("OpenBot stopped.")
}

func printUsage() {
	fmt.Println("OpenBot - Telegram Bot with AI Agent")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  openbot                     Run the Telegram bot (default)")
	fmt.Println("  openbot agent -m <msg>      Send a message through the AI agent")
	fmt.Println("  openbot sessions            List stored conversation sessions")
	fmt.Println("  openbot history [id]        Show session transcript")
	fmt.Println("  openbot message send        Send a message to a channel")
	fmt.Println("  openbot status              Show bot status")
	fmt.Println("  openbot help                Show this help")
	fmt.Println()
	fmt.Println("Agent flags:")
	fmt.Println("  -m, --message <text>  Message body (required)")
	fmt.Println("  -t, --to <id>         Peer ID for session routing")
	fmt.Println("  --agent <id>          Agent ID (default: default)")
	fmt.Println("  --json                Output as JSON")
	fmt.Println()
	fmt.Println("Sessions flags:")
	fmt.Println("  --json                Output as JSON")
	fmt.Println("  --active <minutes>    Filter by recent activity")
	fmt.Println()
	fmt.Println("Message send flags:")
	fmt.Println("  -t, --target <dest>   Delivery target (required)")
	fmt.Println("  -m, --message <text>  Message body (required)")
	fmt.Println("  --channel <channel>   Channel type (default: telegram)")
}
