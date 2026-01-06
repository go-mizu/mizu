package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DesktopApp provides desktop-specific functionality to the frontend
type DesktopApp struct {
	ctx         context.Context
	dataDir     string
	backendAddr string
}

// NewDesktopApp creates a new desktop app instance
func NewDesktopApp(dataDir, backendAddr string) *DesktopApp {
	return &DesktopApp{
		dataDir:     dataDir,
		backendAddr: backendAddr,
	}
}

// startup is called when the app starts
func (a *DesktopApp) startup(ctx context.Context) {
	a.ctx = ctx
	slog.Info("Desktop app started")
}

// domReady is called when the DOM is ready
func (a *DesktopApp) domReady(ctx context.Context) {
	slog.Info("DOM ready")
}

// shutdown is called when the app is closing
func (a *DesktopApp) shutdown(ctx context.Context) {
	slog.Info("Desktop app shutting down")
}

// GetDataDir returns the data directory path
func (a *DesktopApp) GetDataDir() string {
	return a.dataDir
}

// GetBackendURL returns the backend server URL
func (a *DesktopApp) GetBackendURL() string {
	return fmt.Sprintf("http://%s", a.backendAddr)
}

// GetVersion returns the app version info
func (a *DesktopApp) GetVersion() map[string]string {
	return map[string]string{
		"version":   Version,
		"commit":    Commit,
		"buildTime": BuildTime,
	}
}

// ShowNotification displays a system notification
func (a *DesktopApp) ShowNotification(title, message string) {
	runtime.EventsEmit(a.ctx, "notification", map[string]string{
		"title":   title,
		"message": message,
	})
}

// OpenExternal opens a URL in the default browser
func (a *DesktopApp) OpenExternal(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

// ShowOpenDialog shows a file open dialog
func (a *DesktopApp) ShowOpenDialog(title string, filters []string) ([]string, error) {
	var dialogFilters []runtime.FileFilter
	for i := 0; i < len(filters); i += 2 {
		if i+1 < len(filters) {
			dialogFilters = append(dialogFilters, runtime.FileFilter{
				DisplayName: filters[i],
				Pattern:     filters[i+1],
			})
		}
	}

	selection, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   title,
		Filters: dialogFilters,
	})
	if err != nil {
		return nil, err
	}
	return selection, nil
}

// ShowSaveDialog shows a file save dialog
func (a *DesktopApp) ShowSaveDialog(title, defaultFilename string) (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultFilename,
	})
}

// ShowDirectoryDialog shows a directory selection dialog
func (a *DesktopApp) ShowDirectoryDialog(title string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
}

// ReadFile reads a file from the filesystem
func (a *DesktopApp) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data to a file
func (a *DesktopApp) WriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// FileExists checks if a file exists
func (a *DesktopApp) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetHomeDir returns the user's home directory
func (a *DesktopApp) GetHomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

// Minimize minimizes the window
func (a *DesktopApp) Minimize() {
	runtime.WindowMinimise(a.ctx)
}

// Maximize toggles window maximization
func (a *DesktopApp) Maximize() {
	runtime.WindowToggleMaximise(a.ctx)
}

// Close closes the application
func (a *DesktopApp) Close() {
	runtime.Quit(a.ctx)
}

// SetTitle sets the window title
func (a *DesktopApp) SetTitle(title string) {
	runtime.WindowSetTitle(a.ctx, title)
}

// ToggleFullscreen toggles fullscreen mode
func (a *DesktopApp) ToggleFullscreen() {
	runtime.WindowFullscreen(a.ctx)
}
