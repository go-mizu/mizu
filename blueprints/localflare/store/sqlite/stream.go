package sqlite

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	"github.com/go-mizu/blueprints/localflare/store"
)

// StreamStoreImpl implements store.StreamStore.
type StreamStoreImpl struct {
	db      *sql.DB
	dataDir string
}

// CreateVideo creates a new video entry.
func (s *StreamStoreImpl) CreateVideo(ctx context.Context, video *store.StreamVideo) error {
	// Ensure data directory exists
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return err
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO stream_videos (id, uid, name, size, duration, width, height, status, thumbnail_url, playback_hls, playback_dash, storage_key, created_at, ready_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		video.ID, video.UID, video.Name, video.Size, video.Duration, video.Width, video.Height,
		video.Status, video.ThumbnailURL, video.PlaybackHLS, video.PlaybackDASH, video.StorageKey,
		video.CreatedAt, video.ReadyAt)
	return err
}

// GetVideo retrieves a video by UID.
func (s *StreamStoreImpl) GetVideo(ctx context.Context, uid string) (*store.StreamVideo, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, uid, name, size, duration, width, height, status, thumbnail_url, playback_hls, playback_dash, storage_key, created_at, ready_at
		FROM stream_videos WHERE uid = ?`, uid)
	return s.scanVideo(row)
}

// ListVideos lists all videos with pagination.
func (s *StreamStoreImpl) ListVideos(ctx context.Context, limit, offset int) ([]*store.StreamVideo, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, uid, name, size, duration, width, height, status, thumbnail_url, playback_hls, playback_dash, storage_key, created_at, ready_at
		FROM stream_videos ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []*store.StreamVideo
	for rows.Next() {
		video, err := s.scanVideo(rows)
		if err != nil {
			return nil, err
		}
		videos = append(videos, video)
	}
	return videos, rows.Err()
}

// UpdateVideo updates a video.
func (s *StreamStoreImpl) UpdateVideo(ctx context.Context, video *store.StreamVideo) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE stream_videos SET name = ?, size = ?, duration = ?, width = ?, height = ?, status = ?,
		thumbnail_url = ?, playback_hls = ?, playback_dash = ?, storage_key = ?, ready_at = ? WHERE uid = ?`,
		video.Name, video.Size, video.Duration, video.Width, video.Height, video.Status,
		video.ThumbnailURL, video.PlaybackHLS, video.PlaybackDASH, video.StorageKey, video.ReadyAt, video.UID)
	return err
}

// DeleteVideo deletes a video.
func (s *StreamStoreImpl) DeleteVideo(ctx context.Context, uid string) error {
	// First get the video to find its storage key
	video, err := s.GetVideo(ctx, uid)
	if err == nil && video.StorageKey != "" {
		// Delete the file from disk
		filePath := filepath.Join(s.dataDir, video.StorageKey)
		os.Remove(filePath)
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM stream_videos WHERE uid = ?`, uid)
	return err
}

// CreateLiveInput creates a new live input.
func (s *StreamStoreImpl) CreateLiveInput(ctx context.Context, input *store.StreamLiveInput) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO stream_live_inputs (id, uid, name, rtmps_url, rtmps_key, srt_url, webrtc_url, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.ID, input.UID, input.Name, input.RTMPSUrl, input.RTMPSKey, input.SRTUrl, input.WebRTCUrl,
		input.Status, input.CreatedAt)
	return err
}

// GetLiveInput retrieves a live input by UID.
func (s *StreamStoreImpl) GetLiveInput(ctx context.Context, uid string) (*store.StreamLiveInput, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, uid, name, rtmps_url, rtmps_key, srt_url, webrtc_url, status, created_at
		FROM stream_live_inputs WHERE uid = ?`, uid)
	return s.scanLiveInput(row)
}

// ListLiveInputs lists all live inputs.
func (s *StreamStoreImpl) ListLiveInputs(ctx context.Context) ([]*store.StreamLiveInput, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, uid, name, rtmps_url, rtmps_key, srt_url, webrtc_url, status, created_at
		FROM stream_live_inputs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inputs []*store.StreamLiveInput
	for rows.Next() {
		input, err := s.scanLiveInput(rows)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)
	}
	return inputs, rows.Err()
}

// UpdateLiveInput updates a live input.
func (s *StreamStoreImpl) UpdateLiveInput(ctx context.Context, input *store.StreamLiveInput) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE stream_live_inputs SET name = ?, rtmps_url = ?, rtmps_key = ?, srt_url = ?, webrtc_url = ?, status = ? WHERE uid = ?`,
		input.Name, input.RTMPSUrl, input.RTMPSKey, input.SRTUrl, input.WebRTCUrl, input.Status, input.UID)
	return err
}

// DeleteLiveInput deletes a live input.
func (s *StreamStoreImpl) DeleteLiveInput(ctx context.Context, uid string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM stream_live_inputs WHERE uid = ?`, uid)
	return err
}

func (s *StreamStoreImpl) scanVideo(row scanner) (*store.StreamVideo, error) {
	var video store.StreamVideo
	var thumbnailURL, playbackHLS, playbackDASH, storageKey sql.NullString
	var readyAt sql.NullTime
	if err := row.Scan(&video.ID, &video.UID, &video.Name, &video.Size, &video.Duration,
		&video.Width, &video.Height, &video.Status, &thumbnailURL, &playbackHLS, &playbackDASH,
		&storageKey, &video.CreatedAt, &readyAt); err != nil {
		return nil, err
	}
	video.ThumbnailURL = thumbnailURL.String
	video.PlaybackHLS = playbackHLS.String
	video.PlaybackDASH = playbackDASH.String
	video.StorageKey = storageKey.String
	if readyAt.Valid {
		video.ReadyAt = &readyAt.Time
	}
	return &video, nil
}

func (s *StreamStoreImpl) scanLiveInput(row scanner) (*store.StreamLiveInput, error) {
	var input store.StreamLiveInput
	var rtmpsURL, rtmpsKey, srtURL, webrtcURL sql.NullString
	if err := row.Scan(&input.ID, &input.UID, &input.Name, &rtmpsURL, &rtmpsKey, &srtURL,
		&webrtcURL, &input.Status, &input.CreatedAt); err != nil {
		return nil, err
	}
	input.RTMPSUrl = rtmpsURL.String
	input.RTMPSKey = rtmpsKey.String
	input.SRTUrl = srtURL.String
	input.WebRTCUrl = webrtcURL.String
	return &input, nil
}
