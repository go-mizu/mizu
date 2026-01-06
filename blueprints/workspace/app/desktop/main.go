package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/go-mizu/blueprints/workspace/app/web"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	// Configure logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Get data directory
	dataDir := getDataDir()
	slog.Info("Data directory", "path", dataDir)

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data dir: %v", err)
	}

	// Find available port for backend
	port, err := findAvailablePort()
	if err != nil {
		log.Fatalf("Failed to find available port: %v", err)
	}
	backendAddr := fmt.Sprintf("127.0.0.1:%d", port)
	backendURL := fmt.Sprintf("http://%s", backendAddr)
	slog.Info("Backend server", "addr", backendAddr)

	// Set DEV_MODE to seed sample data
	os.Setenv("DEV_MODE", "true")

	// Create and start backend server
	srv, err := web.New(web.Config{
		Addr:    backendAddr,
		DataDir: dataDir,
		Dev:     true, // Enable dev mode to seed data
	})
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start backend server in goroutine
	go func() {
		if err := srv.Run(); err != nil && err != http.ErrServerClosed {
			slog.Error("Backend server error", "error", err)
		}
	}()

	// Wait for server to be ready
	waitForServer(backendAddr)

	// Create desktop app
	app := NewDesktopApp(dataDir, backendAddr)

	// Create Wails app - use backend URL directly for proper HTML pages
	err = wails.Run(&options.App{
		Title:     "Workspace",
		Width:     1280,
		Height:    800,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Handler: NewProxyHandler(backendAddr),
		},
		StartHidden:      false,
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup: func(ctx context.Context) {
			app.startup(ctx)
			// Navigate to the app after startup
			slog.Info("App started, backend at", "url", backendURL)
		},
		OnDomReady: app.domReady,
		OnShutdown: app.shutdown,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			// Clean shutdown
			if err := srv.Close(); err != nil {
				slog.Error("Failed to close server", "error", err)
			}
			return false
		},
		Menu:     app.createMenu(),
		Logger:   logger.NewDefaultLogger(),
		LogLevel: logger.INFO,
		Bind: []interface{}{
			app,
		},
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: false,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
			},
			About: &mac.AboutInfo{
				Title:   "Workspace",
				Message: fmt.Sprintf("Version %s\nCommit %s\nBuilt %s", Version, Commit, BuildTime),
			},
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		log.Fatalf("Error running application: %v", err)
	}
}

// getDataDir returns the platform-appropriate data directory
func getDataDir() string {
	var baseDir string

	switch goruntime.GOOS {
	case "darwin":
		homeDir, _ := os.UserHomeDir()
		baseDir = filepath.Join(homeDir, "Library", "Application Support")
	case "windows":
		baseDir = os.Getenv("APPDATA")
		if baseDir == "" {
			homeDir, _ := os.UserHomeDir()
			baseDir = filepath.Join(homeDir, "AppData", "Roaming")
		}
	default: // linux and others
		baseDir = os.Getenv("XDG_DATA_HOME")
		if baseDir == "" {
			homeDir, _ := os.UserHomeDir()
			baseDir = filepath.Join(homeDir, ".local", "share")
		}
	}

	return filepath.Join(baseDir, "Workspace")
}

// findAvailablePort finds an available TCP port
func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// waitForServer waits for the backend server to be ready
func waitForServer(addr string) {
	client := &http.Client{Timeout: 1 * time.Second}
	url := fmt.Sprintf("http://%s/health", addr)

	for i := 0; i < 30; i++ {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				slog.Info("Backend server ready")
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	slog.Warn("Backend server may not be ready")
}
