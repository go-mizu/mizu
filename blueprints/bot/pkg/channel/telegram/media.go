package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const defaultMaxBytes = 5 * 1024 * 1024 // 5MB

// mediaDownloader handles downloading files from Telegram's servers.
type mediaDownloader struct {
	token    string
	client   *http.Client
	maxBytes int64  // max file size to download (default: 5MB)
	tempDir  string // directory for downloaded files
}

// resolveFileURL gets the download URL for a Telegram file_id.
// It calls the getFile API and returns the constructed download URL
// along with the file_path returned by the API.
func (m *mediaDownloader) resolveFileURL(ctx context.Context, fileID string) (string, string, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", m.token, fileID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", "", fmt.Errorf("build getFile request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("getFile request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read getFile response: %w", err)
	}

	var apiResp struct {
		OK     bool         `json:"ok"`
		Result TelegramFile `json:"result"`
	}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", "", fmt.Errorf("parse getFile response: %w", err)
	}

	if !apiResp.OK || apiResp.Result.FilePath == "" {
		return "", "", fmt.Errorf("getFile failed for file_id %s: %s", fileID, string(body))
	}

	downloadURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", m.token, apiResp.Result.FilePath)
	return downloadURL, apiResp.Result.FilePath, nil
}

// downloadFile downloads a file to the temp directory and returns the local path.
// It resolves the download URL via getFile, checks the content length against
// maxBytes, and writes the file to tempDir using the provided filename.
func (m *mediaDownloader) downloadFile(ctx context.Context, fileID, filename string) (string, error) {
	downloadURL, remotePath, err := m.resolveFileURL(ctx, fileID)
	if err != nil {
		return "", err
	}

	// Use the remote file extension if the caller-provided filename lacks one.
	if filepath.Ext(filename) == "" {
		if ext := filepath.Ext(remotePath); ext != "" {
			filename = filename + ext
		}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("build download request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download file HTTP %d", resp.StatusCode)
	}

	// Check Content-Length against maxBytes limit.
	maxBytes := m.maxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}
	if resp.ContentLength > 0 && resp.ContentLength > maxBytes {
		return "", fmt.Errorf("file too large: %d bytes exceeds limit of %d bytes", resp.ContentLength, maxBytes)
	}

	// Ensure the temp directory exists.
	if err := os.MkdirAll(m.tempDir, 0o755); err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	localPath := filepath.Join(m.tempDir, filename)
	f, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("create local file: %w", err)
	}
	defer f.Close()

	// Use LimitReader to enforce the size limit even when Content-Length is absent.
	reader := io.LimitReader(resp.Body, maxBytes+1)
	written, err := io.Copy(f, reader)
	if err != nil {
		os.Remove(localPath)
		return "", fmt.Errorf("write file: %w", err)
	}

	if written > maxBytes {
		os.Remove(localPath)
		return "", fmt.Errorf("file too large: downloaded %d bytes exceeds limit of %d bytes", written, maxBytes)
	}

	return localPath, nil
}

// extractMedia extracts media info from a TelegramMessage.
// It returns the media type (image, video, audio, document, sticker),
// the file_id to use for downloading, and a suggested filename.
// Media types are checked in priority order: photo, video, video_note,
// audio, voice, document, sticker.
func extractMedia(msg *TelegramMessage) (mediaType string, fileID string, filename string) {
	// Photo: array of sizes, pick the last (highest resolution).
	if len(msg.Photo) > 0 {
		best := msg.Photo[len(msg.Photo)-1]
		return "image", best.FileID, fmt.Sprintf("photo_%s.jpg", best.FileUniqueID)
	}

	// Video.
	if msg.Video != nil {
		name := "video_" + msg.Video.FileUniqueID
		if msg.Video.MimeType == "video/mp4" {
			name += ".mp4"
		}
		return "video", msg.Video.FileID, name
	}

	// Video note (round video).
	if msg.VideoNote != nil {
		return "video", msg.VideoNote.FileID, fmt.Sprintf("videonote_%s.mp4", msg.VideoNote.FileUniqueID)
	}

	// Audio.
	if msg.Audio != nil {
		name := msg.Audio.Title
		if name == "" {
			name = "audio_" + msg.Audio.FileUniqueID
		}
		if msg.Audio.MimeType == "audio/mpeg" {
			name += ".mp3"
		} else if msg.Audio.MimeType == "audio/ogg" {
			name += ".ogg"
		}
		return "audio", msg.Audio.FileID, name
	}

	// Voice message.
	if msg.Voice != nil {
		name := fmt.Sprintf("voice_%s", msg.Voice.FileUniqueID)
		if msg.Voice.MimeType == "audio/ogg" {
			name += ".ogg"
		} else {
			name += ".oga"
		}
		return "audio", msg.Voice.FileID, name
	}

	// Document (general file).
	if msg.Document != nil {
		name := msg.Document.FileName
		if name == "" {
			name = "document_" + msg.Document.FileUniqueID
		}
		return "document", msg.Document.FileID, name
	}

	// Sticker.
	if msg.Sticker != nil {
		ext := ".webp"
		if msg.Sticker.IsAnimated {
			ext = ".tgs"
		} else if msg.Sticker.IsVideo {
			ext = ".webm"
		}
		name := fmt.Sprintf("sticker_%s%s", msg.Sticker.FileUniqueID, ext)
		return "sticker", msg.Sticker.FileID, name
	}

	return "", "", ""
}

// extractLocation extracts location data from a message.
// It checks for a venue first (which includes name and address), then
// falls back to a plain location. Returns ok=false if no location is present.
func extractLocation(msg *TelegramMessage) (lat, lon float64, name, address string, ok bool) {
	// Venue includes both location and place metadata.
	if msg.Venue != nil {
		return msg.Venue.Location.Latitude,
			msg.Venue.Location.Longitude,
			msg.Venue.Title,
			msg.Venue.Address,
			true
	}

	// Plain location.
	if msg.Location != nil {
		return msg.Location.Latitude,
			msg.Location.Longitude,
			"", "",
			true
	}

	return 0, 0, "", "", false
}

// formatLocationText formats location data as readable text suitable for
// display or inclusion in message content.
func formatLocationText(lat, lon float64, name, address string) string {
	if name != "" {
		if address != "" {
			return fmt.Sprintf("Location: %s, %s (%.6f, %.6f)", name, address, lat, lon)
		}
		return fmt.Sprintf("Location: %s (%.6f, %.6f)", name, lat, lon)
	}
	return fmt.Sprintf("Location: %.6f, %.6f", lat, lon)
}
