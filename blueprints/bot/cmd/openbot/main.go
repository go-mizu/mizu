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
		var err error
		switch os.Args[1] {
		case "agent":
			err = runAgent()
		case "sessions":
			err = runSessions()
		case "history":
			sessionID := ""
			if len(os.Args) > 2 {
				sessionID = os.Args[2]
			}
			err = runHistory(sessionID)
		case "message":
			if len(os.Args) > 2 && os.Args[2] == "send" {
				err = runMessageSend()
			} else {
				printUsage()
			}
		case "status":
			err = runStatus()
		case "send":
			// Legacy compat: redirect to agent command.
			if len(os.Args) < 3 {
				log.Fatal("Usage: openbot send <message>")
			}
			msg := strings.Join(os.Args[2:], " ")
			os.Args = []string{os.Args[0], "agent", "-m", msg}
			err = runAgent()

		// --- OpenClaw-compatible commands ---
		case "config":
			if len(os.Args) > 2 {
				switch os.Args[2] {
				case "get":
					err = runConfigGet()
				case "set":
					err = runConfigSet()
				case "unset":
					err = runConfigUnset()
				default:
					fmt.Println("Usage: openbot config <get|set|unset>")
				}
			} else {
				fmt.Println("Usage: openbot config <get|set|unset>")
			}
		case "doctor":
			err = runDoctor()
		case "memory":
			err = runMemory()
		case "skills":
			err = runSkills()
		case "agents":
			err = runAgents()
		case "models":
			err = runModels()
		case "channels":
			err = runChannels()
		case "gateway":
			err = runGatewayCmd()
		case "cron":
			err = runCronCmd()
		case "plugins":
			err = runPlugins()
		case "hooks":
			err = runHooks()
		case "logs":
			err = runLogs()

		case "help", "--help", "-h":
			printUsage()
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
			printUsage()
			os.Exit(1)
		}
		if err != nil {
			log.Fatal(err)
		}
		return
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
	fmt.Println("OpenBot - Telegram Bot with AI Agent (OpenClaw-compatible)")
	fmt.Println()
	fmt.Println("Usage: openbot [command]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  (default)                   Run the Telegram bot")
	fmt.Println()
	fmt.Println("  agent -m <msg>              Send a message through the AI agent")
	fmt.Println("  sessions                    List stored conversation sessions")
	fmt.Println("  history [id]                Show session transcript")
	fmt.Println("  message send                Send a message to a channel")
	fmt.Println("  status                      Show bot status")
	fmt.Println()
	fmt.Println("  config get <path>           Get a config value by dot path")
	fmt.Println("  config set <path> <value>   Set a config value by dot path")
	fmt.Println("  config unset <path>         Remove a config value by dot path")
	fmt.Println("  doctor                      Health checks + quick fixes")
	fmt.Println()
	fmt.Println("  memory status               Show memory search index status")
	fmt.Println("  memory index                Reindex memory files")
	fmt.Println("  memory search -q <query>    Search memory files")
	fmt.Println()
	fmt.Println("  skills list                 List available skills")
	fmt.Println("  skills info <name>          Show skill details")
	fmt.Println("  skills check                Check skill requirements")
	fmt.Println()
	fmt.Println("  agents list                 List configured agents")
	fmt.Println("  models list                 List supported models")
	fmt.Println("  models status               Show current model")
	fmt.Println("  models set <model>          Set default model")
	fmt.Println()
	fmt.Println("  channels list               List configured channels")
	fmt.Println("  gateway                     Gateway control (stub)")
	fmt.Println("  cron                        Cron scheduler (stub)")
	fmt.Println("  plugins list                List plugins")
	fmt.Println("  hooks                       List hooks")
	fmt.Println("  logs                        Show gateway logs")
	fmt.Println()
	fmt.Println("  help                        Show this help")
	fmt.Println()
	fmt.Println("Agent flags:")
	fmt.Println("  -m, --message <text>  Message body (required)")
	fmt.Println("  -t, --to <id>         Peer ID for session routing")
	fmt.Println("  --agent <id>          Agent ID (default: main)")
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
