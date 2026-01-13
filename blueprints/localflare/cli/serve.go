package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/go-mizu/blueprints/localflare/app/web"
)

// NewServe creates the serve command
func NewServe() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start all Localflare services",
		Long: `Start the Localflare services including:
  - Dashboard UI on :8787
  - DNS server on :5353
  - HTTP proxy on :8080
  - HTTPS proxy on :8443

Examples:
  localflare serve                    # Start with defaults
  localflare serve --addr :9000       # Dashboard on port 9000
  localflare serve --dev              # Enable development mode`,
		RunE: runServe,
	}

	cmd.Flags().StringP("addr", "a", ":8787", "Dashboard address")
	cmd.Flags().Int("dns-port", 5353, "DNS server port")
	cmd.Flags().Int("http-port", 8080, "HTTP proxy port")
	cmd.Flags().Int("https-port", 8443, "HTTPS proxy port")

	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	addr, _ := cmd.Flags().GetString("addr")
	dnsPort, _ := cmd.Flags().GetInt("dns-port")
	httpPort, _ := cmd.Flags().GetInt("http-port")
	httpsPort, _ := cmd.Flags().GetInt("https-port")
	dev, _ := cmd.Root().PersistentFlags().GetBool("dev")

	Blank()
	Header("", "Localflare Server")
	Blank()

	Summary(
		"Dashboard", addr,
		"DNS", fmt.Sprintf(":%d", dnsPort),
		"HTTP Proxy", fmt.Sprintf(":%d", httpPort),
		"HTTPS Proxy", fmt.Sprintf(":%d", httpsPort),
		"Data", dataDir,
		"Mode", modeString(dev),
		"Version", Version,
	)
	Blank()

	// Create server
	srv, err := web.New(web.Config{
		Addr:      addr,
		DNSPort:   dnsPort,
		HTTPPort:  httpPort,
		HTTPSPort: httpsPort,
		DataDir:   dataDir,
		Dev:       dev,
	})
	if err != nil {
		Error(fmt.Sprintf("Failed to create server: %v", err))
		return err
	}
	defer srv.Close()

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		Step("", fmt.Sprintf("Dashboard: http://localhost%s", addr))
		Step("", fmt.Sprintf("DNS Server: localhost:%d", dnsPort))
		Step("", fmt.Sprintf("HTTP Proxy: localhost:%d", httpPort))
		Step("", fmt.Sprintf("HTTPS Proxy: localhost:%d", httpsPort))
		Blank()
		errCh <- srv.Run()
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		Error(fmt.Sprintf("Server error: %v", err))
		return err
	case <-quit:
		Blank()
		Step("", "Shutting down...")
		Success("Server stopped")
	}

	return nil
}
