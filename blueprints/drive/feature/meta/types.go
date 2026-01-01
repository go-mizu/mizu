// Package meta provides metadata extraction for files.
package meta

import (
	"time"
)

// FileMetadata contains extracted metadata for a file.
type FileMetadata struct {
	FileID      string    `json:"file_id"`
	ExtractedAt time.Time `json:"extracted_at"`
	MimeType    string    `json:"mime_type"`
	Size        int64     `json:"size"`

	// Type-specific metadata
	Image    *ImageMetadata    `json:"image,omitempty"`
	Audio    *AudioMetadata    `json:"audio,omitempty"`
	Video    *VideoMetadata    `json:"video,omitempty"`
	Document *DocumentMetadata `json:"document,omitempty"`
}

// ImageMetadata contains image-specific metadata.
type ImageMetadata struct {
	// Dimensions
	Width  int `json:"width"`
	Height int `json:"height"`

	// Format info
	ColorSpace string `json:"color_space,omitempty"`
	BitDepth   int    `json:"bit_depth,omitempty"`
	HasAlpha   bool   `json:"has_alpha,omitempty"`
	IsAnimated bool   `json:"is_animated,omitempty"`
	FrameCount int    `json:"frame_count,omitempty"`

	// EXIF Camera Info
	Make      string `json:"make,omitempty"`
	Model     string `json:"model,omitempty"`
	LensModel string `json:"lens_model,omitempty"`
	Software  string `json:"software,omitempty"`

	// EXIF Capture Settings
	DateTimeOriginal string  `json:"date_time_original,omitempty"`
	ExposureTime     string  `json:"exposure_time,omitempty"`
	FNumber          float64 `json:"f_number,omitempty"`
	ISO              int     `json:"iso,omitempty"`
	FocalLength      float64 `json:"focal_length,omitempty"`
	FocalLength35mm  int     `json:"focal_length_35mm,omitempty"`
	Flash            string  `json:"flash,omitempty"`
	MeteringMode     string  `json:"metering_mode,omitempty"`
	ExposureProgram  string  `json:"exposure_program,omitempty"`
	WhiteBalance     string  `json:"white_balance,omitempty"`

	// GPS Location
	GPSLatitude  float64 `json:"gps_latitude,omitempty"`
	GPSLongitude float64 `json:"gps_longitude,omitempty"`
	GPSAltitude  float64 `json:"gps_altitude,omitempty"`

	// IPTC/XMP
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Copyright   string   `json:"copyright,omitempty"`
	Artist      string   `json:"artist,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Rating      int      `json:"rating,omitempty"`

	// Orientation
	Orientation int `json:"orientation,omitempty"`
}

// AudioMetadata contains audio-specific metadata.
type AudioMetadata struct {
	// Technical
	Duration      float64 `json:"duration"`
	DurationStr   string  `json:"duration_str"`
	Bitrate       int     `json:"bitrate,omitempty"`
	SampleRate    int     `json:"sample_rate,omitempty"`
	Channels      int     `json:"channels,omitempty"`
	ChannelLayout string  `json:"channel_layout,omitempty"`
	Codec         string  `json:"codec,omitempty"`
	BitDepth      int     `json:"bit_depth,omitempty"`

	// ID3/Tags
	Title       string `json:"title,omitempty"`
	Artist      string `json:"artist,omitempty"`
	Album       string `json:"album,omitempty"`
	AlbumArtist string `json:"album_artist,omitempty"`
	Composer    string `json:"composer,omitempty"`
	Genre       string `json:"genre,omitempty"`
	Year        int    `json:"year,omitempty"`
	TrackNumber int    `json:"track_number,omitempty"`
	TrackTotal  int    `json:"track_total,omitempty"`
	DiscNumber  int    `json:"disc_number,omitempty"`
	DiscTotal   int    `json:"disc_total,omitempty"`
	Comment     string `json:"comment,omitempty"`

	// Album Art
	HasCoverArt  bool   `json:"has_cover_art"`
	CoverArtType string `json:"cover_art_type,omitempty"`
	CoverArtSize int    `json:"cover_art_size,omitempty"`

	// Additional
	BPM       int    `json:"bpm,omitempty"`
	Key       string `json:"key,omitempty"`
	Publisher string `json:"publisher,omitempty"`
}

// VideoMetadata contains video-specific metadata.
type VideoMetadata struct {
	// Duration & Format
	Duration    float64 `json:"duration"`
	DurationStr string  `json:"duration_str"`
	Container   string  `json:"container,omitempty"`

	// Video Track
	Width        int     `json:"width"`
	Height       int     `json:"height"`
	AspectRatio  string  `json:"aspect_ratio,omitempty"`
	VideoCodec   string  `json:"video_codec,omitempty"`
	VideoBitrate int     `json:"video_bitrate,omitempty"`
	FrameRate    float64 `json:"frame_rate,omitempty"`

	// Audio Track
	HasAudio        bool   `json:"has_audio"`
	AudioCodec      string `json:"audio_codec,omitempty"`
	AudioBitrate    int    `json:"audio_bitrate,omitempty"`
	AudioChannels   int    `json:"audio_channels,omitempty"`
	AudioSampleRate int    `json:"audio_sample_rate,omitempty"`

	// Subtitles
	SubtitleTracks []SubtitleTrack `json:"subtitle_tracks,omitempty"`

	// HDR/Color
	ColorSpace string `json:"color_space,omitempty"`
	HDRFormat  string `json:"hdr_format,omitempty"`
	BitDepth   int    `json:"bit_depth,omitempty"`

	// Tags
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Creator     string `json:"creator,omitempty"`
	Date        string `json:"date,omitempty"`
}

// SubtitleTrack represents a subtitle track in a video.
type SubtitleTrack struct {
	Index    int    `json:"index"`
	Language string `json:"language"`
	Title    string `json:"title,omitempty"`
	Format   string `json:"format,omitempty"`
}

// DocumentMetadata contains document-specific metadata.
type DocumentMetadata struct {
	// PDF specific
	PageCount  int     `json:"page_count,omitempty"`
	PageWidth  float64 `json:"page_width,omitempty"`
	PageHeight float64 `json:"page_height,omitempty"`
	PDFVersion string  `json:"pdf_version,omitempty"`

	// Office documents
	WordCount      int `json:"word_count,omitempty"`
	CharacterCount int `json:"character_count,omitempty"`
	ParagraphCount int `json:"paragraph_count,omitempty"`
	SlideCount     int `json:"slide_count,omitempty"`
	SheetCount     int `json:"sheet_count,omitempty"`

	// Common metadata
	Title      string `json:"title,omitempty"`
	Author     string `json:"author,omitempty"`
	Subject    string `json:"subject,omitempty"`
	Keywords   string `json:"keywords,omitempty"`
	Creator    string `json:"creator,omitempty"`
	Producer   string `json:"producer,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	ModifiedAt string `json:"modified_at,omitempty"`

	// Security
	IsEncrypted bool `json:"is_encrypted,omitempty"`
	HasPassword bool `json:"has_password,omitempty"`
}

// FormatDuration formats duration in seconds to human readable string.
func FormatDuration(seconds float64) string {
	total := int(seconds)
	hours := total / 3600
	minutes := (total % 3600) / 60
	secs := total % 60

	if hours > 0 {
		return formatDuration(hours, minutes, secs)
	}
	return formatMinSec(minutes, secs)
}

func formatDuration(h, m, s int) string {
	return formatNum(h) + ":" + formatNum(m) + ":" + formatNum(s)
}

func formatMinSec(m, s int) string {
	return formatNum(m) + ":" + formatNum(s)
}

func formatNum(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
