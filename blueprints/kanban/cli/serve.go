package cli

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/kanban/app/web"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web server",
	Long: `Start the Kanban web server.

The server provides:
  - Web UI for project management
  - REST API for programmatic access
  - Real-time updates via SSE

Examples:
  kanban serve                    # Start on default port 8080
  kanban serve --port 3000        # Start on port 3000
  kanban serve --host 0.0.0.0     # Listen on all interfaces`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringP("host", "H", "127.0.0.1", "Host to bind to")
	serveCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	serveCmd.Flags().Bool("dev", false, "Enable development mode")
}

func runServe(cmd *cobra.Command, args []string) error {
	host, _ := cmd.Flags().GetString("host")
	port, _ := cmd.Flags().GetInt("port")
	dev, _ := cmd.Flags().GetBool("dev")
	dbPath, _ := cmd.Root().PersistentFlags().GetString("db")

	dataDir := filepath.Dir(dbPath)
	if dataDir == "." {
		dataDir = "."
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	// Create server
	srv, err := web.New(web.Config{
		Addr:    addr,
		DataDir: dataDir,
		Dev:     dev,
	})
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	defer srv.Close()

	// Start server in goroutine
	go func() {
		log.Printf("🚀 Kanban server starting on http://%s", addr)
		log.Printf("📁 Database: %s", dbPath)
		if err := srv.Run(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server stopped")
	return nil
}
