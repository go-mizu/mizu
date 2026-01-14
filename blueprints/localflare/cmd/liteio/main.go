// Command liteio starts a local S3-compatible storage server.
//
// Usage:
//
//	liteio [flags]
//
// Flags:
//
//	-p, --port int           Port to listen on (default 9000)
//	-h, --host string        Host to bind to (default "0.0.0.0")
//	-d, --data-dir string    Data directory (default "$HOME/data/liteio")
//	--driver string          Storage driver DSN (overrides data-dir)
//	--access-key string      Access key ID (default "liteio")
//	--secret-key string      Secret access key (default "liteio123")
//	--region string          S3 region (default "us-east-1")
//	--version                Print version
//	--help                   Print help
//
// Examples:
//
//	# Start with default settings
//	liteio
//
//	# Custom port and data directory
//	liteio -p 8000 -d /tmp/storage
//
//	# Use memory driver (ephemeral)
//	liteio --driver "memory://"
//
//	# Custom credentials
//	liteio --access-key admin --secret-key admin123
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-mizu/blueprints/localflare/pkg/storage/server"
	"github.com/spf13/cobra"
)

// Build variables - set via ldflags
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = ""
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	cfg := server.DefaultConfig()

	cmd := &cobra.Command{
		Use:   "liteio",
		Short: "Local S3-compatible storage server",
		Long: `LiteIO is a lightweight, local S3-compatible object storage server.

It provides a drop-in replacement for S3 during local development and testing,
with full support for the standard S3 API including multipart uploads.

Examples:
  # Start with default settings (port 9000)
  liteio

  # Custom port and data directory
  liteio --port 8000 --data-dir /tmp/storage

  # Use memory driver (ephemeral, data lost on restart)
  liteio --driver "memory://"

  # Custom credentials
  liteio --access-key admin --secret-key admin123

Environment variables:
  LITEIO_PORT         Port to listen on
  LITEIO_HOST         Host to bind to
  LITEIO_DATA_DIR     Data directory path
  LITEIO_DRIVER       Storage driver DSN
  LITEIO_ACCESS_KEY   Access key ID
  LITEIO_SECRET_KEY   Secret access key
  LITEIO_REGION       S3 region`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, BuildTime),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cfg)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	flags := cmd.Flags()
	flags.IntVarP(&cfg.Port, "port", "p", cfg.Port, "Port to listen on")
	flags.StringVar(&cfg.Host, "host", cfg.Host, "Host to bind to")
	flags.StringVarP(&cfg.DSN, "data-dir", "d", "", "Data directory (local driver)")
	flags.StringVar(&cfg.DSN, "driver", "", "Storage driver DSN (overrides data-dir)")
	flags.StringVar(&cfg.AccessKeyID, "access-key", cfg.AccessKeyID, "Access key ID")
	flags.StringVar(&cfg.SecretAccessKey, "secret-key", cfg.SecretAccessKey, "Secret access key")
	flags.StringVar(&cfg.Region, "region", cfg.Region, "S3 region")

	// Environment variable bindings
	if v := os.Getenv("LITEIO_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Port)
	}
	if v := os.Getenv("LITEIO_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("LITEIO_DATA_DIR"); v != "" {
		cfg.DSN = "local://" + v
	}
	if v := os.Getenv("LITEIO_DRIVER"); v != "" {
		cfg.DSN = v
	}
	if v := os.Getenv("LITEIO_ACCESS_KEY"); v != "" {
		cfg.AccessKeyID = v
	}
	if v := os.Getenv("LITEIO_SECRET_KEY"); v != "" {
		cfg.SecretAccessKey = v
	}
	if v := os.Getenv("LITEIO_REGION"); v != "" {
		cfg.Region = v
	}

	return cmd
}

func runServer(cfg *server.Config) error {
	// Set up logger
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	cfg.Logger = slog.New(handler)

	// Handle data-dir flag (convert to local:// DSN)
	if cfg.DSN != "" && len(cfg.DSN) > 0 && cfg.DSN[0] == '/' {
		cfg.DSN = "local://" + cfg.DSN
	}

	srv, err := server.New(cfg)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	// Handle graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server in background
	if err := srv.StartBackground(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	// Print startup info
	fmt.Printf(`
╭─────────────────────────────────────────────────────────────╮
│                       LiteIO Server                        │
├─────────────────────────────────────────────────────────────┤
│  Endpoint:     http://%s                             │
│  Region:       %-45s│
│  Access Key:   %-45s│
│  Secret Key:   %-45s│
╰─────────────────────────────────────────────────────────────╯

AWS CLI example:
  export AWS_ACCESS_KEY_ID=%s
  export AWS_SECRET_ACCESS_KEY=%s
  aws --endpoint-url http://%s s3 ls

Press Ctrl+C to stop the server
`,
		padRight(srv.Addr(), 18),
		cfg.Region,
		cfg.AccessKeyID,
		maskSecret(cfg.SecretAccessKey),
		cfg.AccessKeyID,
		cfg.SecretAccessKey,
		srv.Addr(),
	)

	// Wait for shutdown signal
	<-ctx.Done()

	fmt.Println("\nShutting down...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	fmt.Println("Server stopped")
	return nil
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + string(make([]byte, n-len(s)))
}

func maskSecret(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + "****" + s[len(s)-2:]
}
