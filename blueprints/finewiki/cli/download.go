package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

// Downloader handles file downloads with progress display
type Downloader struct {
	useCurl bool
}

// NewDownloader creates a downloader, detecting curl availability
func NewDownloader() *Downloader {
	_, err := exec.LookPath("curl")
	return &Downloader{useCurl: err == nil}
}

// UseCurl returns true if curl is available
func (d *Downloader) UseCurl() bool {
	return d.useCurl
}

// Download downloads a file with progress display
func (d *Downloader) Download(ctx context.Context, url, dst string, totalSize int64) error {
	if d.useCurl {
		return d.downloadWithCurl(ctx, url, dst)
	}
	return d.downloadWithGo(ctx, url, dst, totalSize)
}

func (d *Downloader) downloadWithCurl(ctx context.Context, url, dst string) error {
	tmp := dst + ".partial"

	// Build curl command with progress bar
	args := []string{
		"-#",           // Progress bar
		"-L",           // Follow redirects
		"-o", tmp,      // Output file
		"--fail",       // Fail on HTTP errors
		"--retry", "3", // Retry on failure
	}

	// Pass through HF_TOKEN if set
	if token := os.Getenv("HF_TOKEN"); token != "" {
		args = append(args, "-H", "Authorization: Bearer "+token)
	}

	args = append(args, url)

	cmd := exec.CommandContext(ctx, "curl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("curl failed: %w", err)
	}

	return os.Rename(tmp, dst)
}

func (d *Downloader) downloadWithGo(ctx context.Context, url, dst string, totalSize int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if token := os.Getenv("HF_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 0} // No timeout for large files
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	// Use Content-Length if available and totalSize is 0
	if totalSize == 0 && resp.ContentLength > 0 {
		totalSize = resp.ContentLength
	}

	tmp := dst + ".partial"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	// Run the download with bubbles progress bar
	m := newDownloadModel(filepath.Base(dst), resp.Body, f, totalSize)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return err
	}

	dm := finalModel.(downloadModel)
	if dm.err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return dm.err
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	return os.Rename(tmp, dst)
}

// downloadModel is the bubbletea model for download progress
type downloadModel struct {
	filename   string
	reader     io.Reader
	writer     io.Writer
	totalSize  int64
	downloaded int64
	progress   progress.Model
	speed      float64 // bytes per second
	startTime  time.Time
	lastUpdate time.Time
	lastBytes  int64
	done       bool
	err        error
	mu         sync.Mutex
}

type tickMsg time.Time
type progressMsg struct {
	downloaded int64
	speed      float64
}
type doneMsg struct {
	err error
}

func newDownloadModel(filename string, reader io.Reader, writer io.Writer, totalSize int64) downloadModel {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)
	return downloadModel{
		filename:   filename,
		reader:     reader,
		writer:     writer,
		totalSize:  totalSize,
		progress:   p,
		startTime:  time.Now(),
		lastUpdate: time.Now(),
	}
}

func (m downloadModel) Init() tea.Cmd {
	return tea.Batch(
		m.startDownload(),
		tickCmd(),
	)
}

func (m downloadModel) startDownload() tea.Cmd {
	return func() tea.Msg {
		buf := make([]byte, 32*1024)
		var downloaded int64
		lastReport := time.Now()
		lastBytes := int64(0)

		for {
			n, err := m.reader.Read(buf)
			if n > 0 {
				_, werr := m.writer.Write(buf[:n])
				if werr != nil {
					return doneMsg{err: werr}
				}
				downloaded += int64(n)

				// Calculate speed every 100ms
				now := time.Now()
				if now.Sub(lastReport) >= 100*time.Millisecond {
					elapsed := now.Sub(lastReport).Seconds()
					speed := float64(downloaded-lastBytes) / elapsed
					lastReport = now
					lastBytes = downloaded

					m.mu.Lock()
					m.downloaded = downloaded
					m.speed = speed
					m.mu.Unlock()
				}
			}

			if err != nil {
				if err == io.EOF {
					m.mu.Lock()
					m.downloaded = downloaded
					m.mu.Unlock()
					return doneMsg{err: nil}
				}
				return doneMsg{err: err}
			}
		}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m downloadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case doneMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit

	case tickMsg:
		m.mu.Lock()
		downloaded := m.downloaded
		speed := m.speed
		m.mu.Unlock()

		m.downloaded = downloaded
		m.speed = speed

		if m.totalSize > 0 {
			percent := float64(downloaded) / float64(m.totalSize)
			cmd := m.progress.SetPercent(percent)
			return m, tea.Batch(cmd, tickCmd())
		}
		return m, tickCmd()

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.err = fmt.Errorf("download cancelled")
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m downloadModel) View() string {
	if m.done {
		if m.err != nil {
			return fmt.Sprintf("  %s: error - %v\n", m.filename, m.err)
		}
		return fmt.Sprintf("  %s: done (%s)\n", m.filename, formatBytes(m.downloaded))
	}

	var progressView string
	if m.totalSize > 0 {
		percent := float64(m.downloaded) / float64(m.totalSize) * 100
		eta := formatETA(m.totalSize-m.downloaded, m.speed)
		progressView = fmt.Sprintf("  %s %s %.0f%% | %s/%s | %s/s | ETA %s\n",
			m.filename,
			m.progress.View(),
			percent,
			formatBytes(m.downloaded),
			formatBytes(m.totalSize),
			formatBytes(int64(m.speed)),
			eta,
		)
	} else {
		progressView = fmt.Sprintf("  %s: %s | %s/s\n",
			m.filename,
			formatBytes(m.downloaded),
			formatBytes(int64(m.speed)),
		)
	}

	return progressView
}

func formatETA(remaining int64, speed float64) string {
	if speed <= 0 {
		return "..."
	}
	seconds := float64(remaining) / speed
	if seconds < 60 {
		return fmt.Sprintf("%.0fs", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%.0fm%.0fs", seconds/60, float64(int(seconds)%60))
	}
	return fmt.Sprintf("%.0fh%.0fm", seconds/3600, float64(int(seconds)%3600)/60)
}
