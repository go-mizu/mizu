package api

import (
	"time"

	"github.com/go-mizu/mizu"
	"github.com/oklog/ulid/v2"
)

// Stream handles Cloudflare Stream requests.
type Stream struct{}

// NewStream creates a new Stream handler.
func NewStream() *Stream {
	return &Stream{}
}

// StreamVideo represents a video.
type StreamVideo struct {
	UID       string            `json:"uid"`
	Name      string            `json:"name"`
	Created   string            `json:"created"`
	Duration  int               `json:"duration"`
	Size      int64             `json:"size"`
	Status    map[string]string `json:"status"`
	Thumbnail string            `json:"thumbnail,omitempty"`
	Playback  map[string]string `json:"playback"`
}

// LiveInput represents a live streaming input.
type LiveInput struct {
	UID     string            `json:"uid"`
	Name    string            `json:"name"`
	Created string            `json:"created"`
	Status  string            `json:"status"`
	RTMPS   map[string]string `json:"rtmps"`
}

// ListVideos lists all videos.
func (h *Stream) ListVideos(c *mizu.Ctx) error {
	now := time.Now()
	videos := []StreamVideo{
		{
			UID:       "vid-" + ulid.Make().String()[:8],
			Name:      "Product Demo",
			Created:   now.Add(-1 * time.Hour).Format(time.RFC3339),
			Duration:  245,
			Size:      45 * 1024 * 1024,
			Status:    map[string]string{"state": "ready"},
			Thumbnail: "https://example.com/thumb1.jpg",
			Playback:  map[string]string{"hls": "https://customer-xxx.cloudflarestream.com/vid-1/manifest/video.m3u8"},
		},
		{
			UID:       "vid-" + ulid.Make().String()[:8],
			Name:      "Getting Started Tutorial",
			Created:   now.Add(-24 * time.Hour).Format(time.RFC3339),
			Duration:  1234,
			Size:      234 * 1024 * 1024,
			Status:    map[string]string{"state": "ready"},
			Thumbnail: "https://example.com/thumb2.jpg",
			Playback:  map[string]string{"hls": "https://customer-xxx.cloudflarestream.com/vid-2/manifest/video.m3u8"},
		},
		{
			UID:       "vid-" + ulid.Make().String()[:8],
			Name:      "Conference Recording",
			Created:   now.Add(-48 * time.Hour).Format(time.RFC3339),
			Duration:  5678,
			Size:      1288490188, // ~1.2 GB
			Status:    map[string]string{"state": "ready"},
			Thumbnail: "https://example.com/thumb3.jpg",
			Playback:  map[string]string{"hls": "https://customer-xxx.cloudflarestream.com/vid-3/manifest/video.m3u8"},
		},
		{
			UID:      "vid-" + ulid.Make().String()[:8],
			Name:     "Marketing Video",
			Created:  now.Add(-2 * time.Hour).Format(time.RFC3339),
			Duration: 0,
			Size:     0,
			Status:   map[string]string{"state": "pendingupload"},
			Playback: map[string]string{},
		},
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"videos": videos,
		},
	})
}

// GetVideo retrieves a video by ID.
func (h *Stream) GetVideo(c *mizu.Ctx) error {
	id := c.Param("id")
	now := time.Now()

	video := StreamVideo{
		UID:       id,
		Name:      "Sample Video",
		Created:   now.Add(-1 * time.Hour).Format(time.RFC3339),
		Duration:  300,
		Size:      50 * 1024 * 1024,
		Status:    map[string]string{"state": "ready"},
		Thumbnail: "https://example.com/thumb.jpg",
		Playback:  map[string]string{"hls": "https://customer-xxx.cloudflarestream.com/" + id + "/manifest/video.m3u8"},
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result":  video,
	})
}

// Upload handles video upload.
func (h *Stream) Upload(c *mizu.Ctx) error {
	video := StreamVideo{
		UID:      "vid-" + ulid.Make().String()[:8],
		Name:     "Uploaded Video",
		Created:  time.Now().Format(time.RFC3339),
		Duration: 0,
		Size:     0,
		Status:   map[string]string{"state": "pendingupload"},
		Playback: map[string]string{},
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  video,
	})
}

// DeleteVideo deletes a video.
func (h *Stream) DeleteVideo(c *mizu.Ctx) error {
	id := c.Param("id")
	return c.JSON(200, map[string]any{
		"success": true,
		"result":  map[string]string{"uid": id},
	})
}

// ListLiveInputs lists live streaming inputs.
func (h *Stream) ListLiveInputs(c *mizu.Ctx) error {
	now := time.Now()
	liveInputs := []LiveInput{
		{
			UID:     "live-" + ulid.Make().String()[:8],
			Name:    "Main Studio",
			Created: now.Add(-168 * time.Hour).Format(time.RFC3339),
			Status:  "connected",
			RTMPS: map[string]string{
				"url":       "rtmps://live.cloudflare.com:443/live",
				"streamKey": "xxx-yyy-zzz",
			},
		},
		{
			UID:     "live-" + ulid.Make().String()[:8],
			Name:    "Backup Stream",
			Created: now.Add(-72 * time.Hour).Format(time.RFC3339),
			Status:  "disconnected",
			RTMPS: map[string]string{
				"url":       "rtmps://live.cloudflare.com:443/live",
				"streamKey": "aaa-bbb-ccc",
			},
		},
	}

	return c.JSON(200, map[string]any{
		"success": true,
		"result": map[string]any{
			"live_inputs": liveInputs,
		},
	})
}

// CreateLiveInput creates a new live input.
func (h *Stream) CreateLiveInput(c *mizu.Ctx) error {
	var input struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&input, 1<<20); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid input"})
	}

	liveInput := LiveInput{
		UID:     "live-" + ulid.Make().String()[:8],
		Name:    input.Name,
		Created: time.Now().Format(time.RFC3339),
		Status:  "disconnected",
		RTMPS: map[string]string{
			"url":       "rtmps://live.cloudflare.com:443/live",
			"streamKey": ulid.Make().String()[:12],
		},
	}

	return c.JSON(201, map[string]any{
		"success": true,
		"result":  liveInput,
	})
}
