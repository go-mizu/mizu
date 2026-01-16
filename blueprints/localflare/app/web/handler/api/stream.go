package api

import (
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/go-mizu/blueprints/localflare/store"
	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"
)

// Stream handles Cloudflare Stream requests.
type Stream struct {
	store   store.Store
	dataDir string
}

// NewStream creates a new Stream handler.
func NewStream(st store.Store, dataDir string) *Stream {
	return &Stream{store: st, dataDir: filepath.Join(dataDir, "stream")}
}

// StreamVideoResponse represents a video response.
type StreamVideoResponse struct {
	UID       string            `json:"uid"`
	Name      string            `json:"name"`
	Created   string            `json:"created"`
	Duration  float64           `json:"duration"`
	Size      int64             `json:"size"`
	Status    map[string]string `json:"status"`
	Thumbnail string            `json:"thumbnail,omitempty"`
	Playback  map[string]string `json:"playback"`
}

// LiveInputResponse represents a live streaming input response.
type LiveInputResponse struct {
	UID     string            `json:"uid"`
	Name    string            `json:"name"`
	Created string            `json:"created"`
	Status  string            `json:"status"`
	RTMPS   map[string]string `json:"rtmps"`
}

// ListVideos lists all videos.
func (h *Stream) ListVideos(c *mizu.Ctx) error {
	videos, err := h.store.Stream().ListVideos(c.Request().Context(), 100, 0)
	if err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	var result []StreamVideoResponse
	for _, v := range videos {
		result = append(result, h.videoToResponse(v))
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"videos": result,
		},
	})
}

// GetVideo retrieves a video by ID.
func (h *Stream) GetVideo(c *mizu.Ctx) error {
	uid := c.Param("id")

	video, err := h.store.Stream().GetVideo(c.Request().Context(), uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(404, map[string]any{
				"success": false,
				"errors":  []map[string]any{{"code": 1001, "message": "Video not found"}},
			})
		}
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  h.videoToResponse(video),
	})
}

// Upload handles video upload.
func (h *Stream) Upload(c *mizu.Ctx) error {
	// Ensure data directory exists
	if err := os.MkdirAll(h.dataDir, 0755); err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	// Parse multipart form
	file, header, err := c.Request().FormFile("file")
	if err != nil {
		// If no file, create a pending upload entry
		var input struct {
			Name string `json:"name"`
		}
		if bindErr := c.BindJSON(&input, 1<<20); bindErr != nil {
			input.Name = "Uploaded Video"
		}

		uid := ulid.Make().String()[:12]
		now := time.Now()

		video := &store.StreamVideo{
			ID:        "vid_" + ulid.Make().String(),
			UID:       "vid-" + uid,
			Name:      input.Name,
			Status:    "pendingupload",
			CreatedAt: now,
		}

		if err := h.store.Stream().CreateVideo(c.Request().Context(), video); err != nil {
			return c.JSON(500, map[string]any{
				"success": false,
				"errors":  []map[string]any{{"message": err.Error()}},
			})
		}

		return c.JSON(201, map[string]any{
			"success": true,
			"result":  h.videoToResponse(video),
		})
	}
	defer file.Close()

	uid := ulid.Make().String()[:12]
	storageKey := uid + filepath.Ext(header.Filename)
	filePath := filepath.Join(h.dataDir, storageKey)

	// Save file to disk
	dst, err := os.Create(filePath)
	if err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}
	defer dst.Close()

	size, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(filePath)
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	now := time.Now()
	video := &store.StreamVideo{
		ID:          "vid_" + ulid.Make().String(),
		UID:         "vid-" + uid,
		Name:        header.Filename,
		Size:        size,
		Status:      "ready",
		StorageKey:  storageKey,
		PlaybackHLS: "/api/stream/videos/vid-" + uid + "/manifest.m3u8",
		CreatedAt:   now,
		ReadyAt:     &now,
	}

	if err := h.store.Stream().CreateVideo(c.Request().Context(), video); err != nil {
		os.Remove(filePath)
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  h.videoToResponse(video),
	})
}

// DeleteVideo deletes a video.
func (h *Stream) DeleteVideo(c *mizu.Ctx) error {
	uid := c.Param("id")

	if err := h.store.Stream().DeleteVideo(c.Request().Context(), uid); err != nil {
		if err == sql.ErrNoRows {
			return c.JSON(404, map[string]any{
				"success": false,
				"errors":  []map[string]any{{"code": 1001, "message": "Video not found"}},
			})
		}
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"uid": uid},
	})
}

// ListLiveInputs lists live streaming inputs.
func (h *Stream) ListLiveInputs(c *mizu.Ctx) error {
	inputs, err := h.store.Stream().ListLiveInputs(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	var result []LiveInputResponse
	for _, i := range inputs {
		result = append(result, h.liveInputToResponse(i))
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"live_inputs": result,
		},
	})
}

// CreateLiveInput creates a new live input.
func (h *Stream) CreateLiveInput(c *mizu.Ctx) error {
	var input struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": "Invalid input"}},
		})
	}

	if input.Name == "" {
		input.Name = "Live Input"
	}

	uid := ulid.Make().String()[:12]
	streamKey := ulid.Make().String()[:16]

	liveInput := &store.StreamLiveInput{
		ID:        "live_" + ulid.Make().String(),
		UID:       "live-" + uid,
		Name:      input.Name,
		RTMPSUrl:  "rtmps://live.localflare.local:443/live",
		RTMPSKey:  streamKey,
		SRTUrl:    "srt://live.localflare.local:10000?streamid=" + uid,
		WebRTCUrl: "https://live.localflare.local/whip/" + uid,
		Status:    "disconnected",
		CreatedAt: time.Now(),
	}

	if err := h.store.Stream().CreateLiveInput(c.Request().Context(), liveInput); err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  h.liveInputToResponse(liveInput),
	})
}

// DeleteLiveInput deletes a live input.
func (h *Stream) DeleteLiveInput(c *mizu.Ctx) error {
	uid := c.Param("id")

	if err := h.store.Stream().DeleteLiveInput(c.Request().Context(), uid); err != nil {
		return c.JSON(500, map[string]any{
			"success": false,
			"errors":  []map[string]any{{"message": err.Error()}},
		})
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"uid": uid},
	})
}

func (h *Stream) videoToResponse(v *store.StreamVideo) StreamVideoResponse {
	playback := map[string]string{}
	if v.PlaybackHLS != "" {
		playback["hls"] = v.PlaybackHLS
	}
	if v.PlaybackDASH != "" {
		playback["dash"] = v.PlaybackDASH
	}

	return StreamVideoResponse{
		UID:       v.UID,
		Name:      v.Name,
		Created:   v.CreatedAt.Format(time.RFC3339),
		Duration:  v.Duration,
		Size:      v.Size,
		Status:    map[string]string{"state": v.Status},
		Thumbnail: v.ThumbnailURL,
		Playback:  playback,
	}
}

func (h *Stream) liveInputToResponse(i *store.StreamLiveInput) LiveInputResponse {
	return LiveInputResponse{
		UID:     i.UID,
		Name:    i.Name,
		Created: i.CreatedAt.Format(time.RFC3339),
		Status:  i.Status,
		RTMPS: map[string]string{
			"url":       i.RTMPSUrl,
			"streamKey": i.RTMPSKey,
		},
	}
}
