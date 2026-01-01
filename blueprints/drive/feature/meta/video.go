package meta

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// extractVideoMetadata extracts metadata from video files.
func extractVideoMetadata(ctx context.Context, filePath string) (*VideoMetadata, error) {
	meta := &VideoMetadata{}

	// First try ffprobe if available
	if ffprobeMeta := extractWithFFprobe(ctx, filePath); ffprobeMeta != nil {
		return ffprobeMeta, nil
	}

	// Fall back to basic parsing
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	ext := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(ext, ".mp4") || strings.HasSuffix(ext, ".m4v") || strings.HasSuffix(ext, ".mov"):
		extractMP4Metadata(file, meta)
	case strings.HasSuffix(ext, ".webm") || strings.HasSuffix(ext, ".mkv"):
		extractWebMMetadata(file, meta)
	case strings.HasSuffix(ext, ".avi"):
		extractAVIMetadata(file, meta)
	}

	if meta.Duration > 0 {
		meta.DurationStr = FormatDuration(meta.Duration)
	}

	return meta, nil
}

// ffprobeResult represents the JSON output from ffprobe.
type ffprobeResult struct {
	Format struct {
		Duration   string            `json:"duration"`
		Size       string            `json:"size"`
		BitRate    string            `json:"bit_rate"`
		FormatName string            `json:"format_name"`
		Tags       map[string]string `json:"tags"`
	} `json:"format"`
	Streams []struct {
		CodecType     string  `json:"codec_type"`
		CodecName     string  `json:"codec_name"`
		Width         int     `json:"width"`
		Height        int     `json:"height"`
		AvgFrameRate  string  `json:"avg_frame_rate"`
		BitRate       string  `json:"bit_rate"`
		SampleRate    string  `json:"sample_rate"`
		Channels      int     `json:"channels"`
		ChannelLayout string  `json:"channel_layout"`
		PixFmt        string  `json:"pix_fmt"`
		ColorSpace    string  `json:"color_space"`
		BitsPerSample int     `json:"bits_per_sample"`
		Tags          map[string]string `json:"tags"`
	} `json:"streams"`
}

// extractWithFFprobe tries to use ffprobe for accurate metadata extraction.
func extractWithFFprobe(ctx context.Context, filePath string) *VideoMetadata {
	// Check if ffprobe is available
	_, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil
	}

	// Run ffprobe
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath)

	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var result ffprobeResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil
	}

	meta := &VideoMetadata{}

	// Parse format info
	if dur, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
		meta.Duration = dur
		meta.DurationStr = FormatDuration(dur)
	}
	meta.Container = strings.Split(result.Format.FormatName, ",")[0]

	// Parse format tags
	if result.Format.Tags != nil {
		meta.Title = result.Format.Tags["title"]
		meta.Description = result.Format.Tags["description"]
		meta.Creator = result.Format.Tags["artist"]
		meta.Date = result.Format.Tags["date"]
	}

	// Parse streams
	for _, stream := range result.Streams {
		switch stream.CodecType {
		case "video":
			meta.Width = stream.Width
			meta.Height = stream.Height
			meta.VideoCodec = stream.CodecName
			if bitrate, err := strconv.Atoi(stream.BitRate); err == nil {
				meta.VideoBitrate = bitrate / 1000
			}
			meta.ColorSpace = stream.ColorSpace
			meta.BitDepth = stream.BitsPerSample

			// Parse frame rate (e.g., "30000/1001" or "30/1")
			if parts := strings.Split(stream.AvgFrameRate, "/"); len(parts) == 2 {
				num, _ := strconv.ParseFloat(parts[0], 64)
				denom, _ := strconv.ParseFloat(parts[1], 64)
				if denom > 0 {
					meta.FrameRate = num / denom
				}
			}

			// Calculate aspect ratio
			if meta.Height > 0 {
				ratio := float64(meta.Width) / float64(meta.Height)
				meta.AspectRatio = calculateAspectRatio(meta.Width, meta.Height)
				_ = ratio
			}

			// Check for HDR
			if stream.PixFmt != "" {
				if strings.Contains(stream.PixFmt, "10le") || strings.Contains(stream.PixFmt, "10be") {
					meta.BitDepth = 10
				}
				if strings.Contains(stream.PixFmt, "12le") || strings.Contains(stream.PixFmt, "12be") {
					meta.BitDepth = 12
				}
			}

		case "audio":
			meta.HasAudio = true
			meta.AudioCodec = stream.CodecName
			if bitrate, err := strconv.Atoi(stream.BitRate); err == nil {
				meta.AudioBitrate = bitrate / 1000
			}
			if sr, err := strconv.Atoi(stream.SampleRate); err == nil {
				meta.AudioSampleRate = sr
			}
			meta.AudioChannels = stream.Channels

		case "subtitle":
			track := SubtitleTrack{
				Index:    len(meta.SubtitleTracks),
				Format:   stream.CodecName,
				Language: stream.Tags["language"],
				Title:    stream.Tags["title"],
			}
			meta.SubtitleTracks = append(meta.SubtitleTracks, track)
		}
	}

	return meta
}

// extractMP4Metadata extracts metadata from MP4/MOV files using basic parsing.
func extractMP4Metadata(r io.ReadSeeker, meta *VideoMetadata) {
	meta.Container = "mp4"

	// Read atoms
	for {
		header := make([]byte, 8)
		if _, err := r.Read(header); err != nil {
			break
		}

		size := int64(binary.BigEndian.Uint32(header[:4]))
		atomType := string(header[4:8])

		if size == 1 {
			// 64-bit size
			size64 := make([]byte, 8)
			r.Read(size64)
			size = int64(binary.BigEndian.Uint64(size64))
			size -= 8
		}

		if size < 8 {
			break
		}

		switch atomType {
		case "moov", "trak", "mdia", "minf", "stbl", "udta", "meta":
			// Container atoms - continue parsing inside
			if atomType == "meta" {
				// Skip 4-byte version/flags
				r.Seek(4, io.SeekCurrent)
			}
			continue

		case "mvhd":
			// Movie header
			data := make([]byte, size-8)
			r.Read(data)
			if len(data) >= 20 {
				version := data[0]
				var duration, timescale int64
				if version == 0 {
					timescale = int64(binary.BigEndian.Uint32(data[12:16]))
					duration = int64(binary.BigEndian.Uint32(data[16:20]))
				} else if len(data) >= 32 {
					timescale = int64(binary.BigEndian.Uint32(data[20:24]))
					duration = int64(binary.BigEndian.Uint64(data[24:32]))
				}
				if timescale > 0 {
					meta.Duration = float64(duration) / float64(timescale)
				}
			}

		case "tkhd":
			// Track header - extract dimensions
			data := make([]byte, size-8)
			r.Read(data)
			if len(data) >= 84 {
				// Width and height are in 16.16 fixed point at offset 76 and 80
				meta.Width = int(binary.BigEndian.Uint32(data[76:80]) >> 16)
				meta.Height = int(binary.BigEndian.Uint32(data[80:84]) >> 16)
			}

		case "hdlr":
			// Handler reference - detect audio/video
			data := make([]byte, size-8)
			r.Read(data)
			if len(data) >= 12 {
				handlerType := string(data[8:12])
				if handlerType == "soun" {
					meta.HasAudio = true
				}
			}

		case "stsd":
			// Sample description - codec info
			data := make([]byte, size-8)
			r.Read(data)
			if len(data) >= 16 {
				codec := string(data[12:16])
				codec = strings.TrimRight(codec, "\x00")
				switch codec {
				case "avc1", "avc2", "avc3", "avc4":
					meta.VideoCodec = "h264"
				case "hvc1", "hev1":
					meta.VideoCodec = "h265"
				case "vp08":
					meta.VideoCodec = "vp8"
				case "vp09":
					meta.VideoCodec = "vp9"
				case "av01":
					meta.VideoCodec = "av1"
				case "mp4a":
					meta.AudioCodec = "aac"
				case "ac-3":
					meta.AudioCodec = "ac3"
				case "ec-3":
					meta.AudioCodec = "eac3"
				default:
					if meta.VideoCodec == "" {
						meta.VideoCodec = codec
					}
				}
			}

		default:
			// Skip unknown atoms
			r.Seek(size-8, io.SeekCurrent)
		}
	}

	if meta.Width > 0 && meta.Height > 0 {
		meta.AspectRatio = calculateAspectRatio(meta.Width, meta.Height)
	}
}

// extractWebMMetadata extracts metadata from WebM/MKV files.
func extractWebMMetadata(r io.ReadSeeker, meta *VideoMetadata) {
	meta.Container = "webm"

	// Read EBML header
	header := make([]byte, 4)
	if _, err := r.Read(header); err != nil {
		return
	}

	// Check for EBML ID (0x1A45DFA3)
	if header[0] != 0x1A || header[1] != 0x45 || header[2] != 0xDF || header[3] != 0xA3 {
		return
	}

	// Skip EBML header size and content
	size := readVINT(r)
	r.Seek(int64(size), io.SeekCurrent)

	// Look for Segment element (0x18538067)
	for {
		id := readVINT(r)
		size := readVINT(r)

		if id == 0 || size == 0 {
			break
		}

		switch id {
		case 0x18538067: // Segment
			// Continue parsing inside segment
			continue
		case 0x1549A966: // SegmentInfo
			parseSegmentInfo(r, int64(size), meta)
		case 0x1654AE6B: // Tracks
			parseTracks(r, int64(size), meta)
		default:
			r.Seek(int64(size), io.SeekCurrent)
		}
	}
}

func readVINT(r io.Reader) uint64 {
	first := make([]byte, 1)
	if _, err := r.Read(first); err != nil {
		return 0
	}

	b := first[0]
	if b == 0 {
		return 0
	}

	// Determine length from leading 1s
	length := 0
	mask := byte(0x80)
	for i := 0; i < 8; i++ {
		if b&mask != 0 {
			length = i + 1
			break
		}
		mask >>= 1
	}

	if length == 0 {
		return 0
	}

	// Read remaining bytes
	result := uint64(b & (0xFF >> length))
	for i := 1; i < length; i++ {
		next := make([]byte, 1)
		r.Read(next)
		result = (result << 8) | uint64(next[0])
	}

	return result
}

func parseSegmentInfo(r io.ReadSeeker, size int64, meta *VideoMetadata) {
	end, _ := r.Seek(0, io.SeekCurrent)
	end += size

	for {
		pos, _ := r.Seek(0, io.SeekCurrent)
		if pos >= end {
			break
		}

		id := readVINT(r)
		elemSize := readVINT(r)

		if id == 0 {
			break
		}

		switch id {
		case 0x4489: // Duration (float)
			data := make([]byte, elemSize)
			r.Read(data)
			if len(data) == 8 {
				bits := binary.BigEndian.Uint64(data)
				meta.Duration = float64FromBits(bits) / 1000 // Convert from ms
			} else if len(data) == 4 {
				bits := binary.BigEndian.Uint32(data)
				meta.Duration = float64(float32FromBits(bits)) / 1000
			}
		case 0x7BA9: // Title
			data := make([]byte, elemSize)
			r.Read(data)
			meta.Title = string(data)
		case 0x4461: // DateUTC
			data := make([]byte, elemSize)
			r.Read(data)
			// Could parse date here
		default:
			r.Seek(int64(elemSize), io.SeekCurrent)
		}
	}
}

func parseTracks(r io.ReadSeeker, size int64, meta *VideoMetadata) {
	end, _ := r.Seek(0, io.SeekCurrent)
	end += size

	for {
		pos, _ := r.Seek(0, io.SeekCurrent)
		if pos >= end {
			break
		}

		id := readVINT(r)
		elemSize := readVINT(r)

		if id == 0 {
			break
		}

		if id == 0xAE { // TrackEntry
			parseTrackEntry(r, int64(elemSize), meta)
		} else {
			r.Seek(int64(elemSize), io.SeekCurrent)
		}
	}
}

func parseTrackEntry(r io.ReadSeeker, size int64, meta *VideoMetadata) {
	end, _ := r.Seek(0, io.SeekCurrent)
	end += size

	var trackType uint64

	for {
		pos, _ := r.Seek(0, io.SeekCurrent)
		if pos >= end {
			break
		}

		id := readVINT(r)
		elemSize := readVINT(r)

		if id == 0 {
			break
		}

		switch id {
		case 0x83: // TrackType
			data := make([]byte, elemSize)
			r.Read(data)
			if len(data) > 0 {
				trackType = uint64(data[0])
			}
		case 0x86: // CodecID
			data := make([]byte, elemSize)
			r.Read(data)
			codec := string(data)
			if trackType == 1 { // Video
				switch {
				case strings.HasPrefix(codec, "V_VP8"):
					meta.VideoCodec = "vp8"
				case strings.HasPrefix(codec, "V_VP9"):
					meta.VideoCodec = "vp9"
				case strings.HasPrefix(codec, "V_AV1"):
					meta.VideoCodec = "av1"
				case strings.HasPrefix(codec, "V_MPEG4/ISO/AVC"):
					meta.VideoCodec = "h264"
				case strings.HasPrefix(codec, "V_MPEGH/ISO/HEVC"):
					meta.VideoCodec = "h265"
				}
			} else if trackType == 2 { // Audio
				meta.HasAudio = true
				switch {
				case strings.HasPrefix(codec, "A_VORBIS"):
					meta.AudioCodec = "vorbis"
				case strings.HasPrefix(codec, "A_OPUS"):
					meta.AudioCodec = "opus"
				case strings.HasPrefix(codec, "A_AAC"):
					meta.AudioCodec = "aac"
				}
			}
		case 0xE0: // Video settings
			parseVideoSettings(r, int64(elemSize), meta)
		case 0xE1: // Audio settings
			parseAudioSettings(r, int64(elemSize), meta)
		default:
			r.Seek(int64(elemSize), io.SeekCurrent)
		}
	}
}

func parseVideoSettings(r io.ReadSeeker, size int64, meta *VideoMetadata) {
	end, _ := r.Seek(0, io.SeekCurrent)
	end += size

	for {
		pos, _ := r.Seek(0, io.SeekCurrent)
		if pos >= end {
			break
		}

		id := readVINT(r)
		elemSize := readVINT(r)

		if id == 0 {
			break
		}

		switch id {
		case 0xB0: // PixelWidth
			data := make([]byte, elemSize)
			r.Read(data)
			meta.Width = readUInt(data)
		case 0xBA: // PixelHeight
			data := make([]byte, elemSize)
			r.Read(data)
			meta.Height = readUInt(data)
		default:
			r.Seek(int64(elemSize), io.SeekCurrent)
		}
	}

	if meta.Width > 0 && meta.Height > 0 {
		meta.AspectRatio = calculateAspectRatio(meta.Width, meta.Height)
	}
}

func parseAudioSettings(r io.ReadSeeker, size int64, meta *VideoMetadata) {
	end, _ := r.Seek(0, io.SeekCurrent)
	end += size

	for {
		pos, _ := r.Seek(0, io.SeekCurrent)
		if pos >= end {
			break
		}

		id := readVINT(r)
		elemSize := readVINT(r)

		if id == 0 {
			break
		}

		switch id {
		case 0xB5: // SamplingFrequency
			data := make([]byte, elemSize)
			r.Read(data)
			if len(data) == 8 {
				bits := binary.BigEndian.Uint64(data)
				meta.AudioSampleRate = int(float64FromBits(bits))
			} else if len(data) == 4 {
				bits := binary.BigEndian.Uint32(data)
				meta.AudioSampleRate = int(float32FromBits(bits))
			}
		case 0x9F: // Channels
			data := make([]byte, elemSize)
			r.Read(data)
			meta.AudioChannels = readUInt(data)
		default:
			r.Seek(int64(elemSize), io.SeekCurrent)
		}
	}
}

func readUInt(data []byte) int {
	result := 0
	for _, b := range data {
		result = (result << 8) | int(b)
	}
	return result
}

func float64FromBits(bits uint64) float64 {
	return math.Float64frombits(bits)
}

func float32FromBits(bits uint32) float32 {
	return math.Float32frombits(bits)
}

// extractAVIMetadata extracts metadata from AVI files.
func extractAVIMetadata(r io.ReadSeeker, meta *VideoMetadata) {
	meta.Container = "avi"

	// Read RIFF header
	header := make([]byte, 12)
	if _, err := r.Read(header); err != nil {
		return
	}

	if string(header[:4]) != "RIFF" || string(header[8:12]) != "AVI " {
		return
	}

	// Read chunks
	for {
		chunkHeader := make([]byte, 8)
		if _, err := r.Read(chunkHeader); err != nil {
			break
		}

		chunkID := string(chunkHeader[:4])
		chunkSize := int64(binary.LittleEndian.Uint32(chunkHeader[4:8]))

		switch chunkID {
		case "LIST":
			listType := make([]byte, 4)
			r.Read(listType)
			if string(listType) == "hdrl" || string(listType) == "movi" {
				continue
			}
			r.Seek(chunkSize-4, io.SeekCurrent)

		case "avih":
			// Main AVI header
			data := make([]byte, chunkSize)
			r.Read(data)
			if len(data) >= 32 {
				microSecPerFrame := binary.LittleEndian.Uint32(data[0:4])
				if microSecPerFrame > 0 {
					meta.FrameRate = 1000000.0 / float64(microSecPerFrame)
				}
				totalFrames := binary.LittleEndian.Uint32(data[16:20])
				meta.Width = int(binary.LittleEndian.Uint32(data[32:36]))
				meta.Height = int(binary.LittleEndian.Uint32(data[36:40]))
				if meta.FrameRate > 0 {
					meta.Duration = float64(totalFrames) / meta.FrameRate
				}
			}

		case "strh":
			// Stream header
			data := make([]byte, chunkSize)
			r.Read(data)
			if len(data) >= 8 {
				streamType := string(bytes.TrimRight(data[0:4], "\x00"))
				if streamType == "vids" {
					meta.VideoCodec = string(bytes.TrimRight(data[4:8], "\x00"))
				} else if streamType == "auds" {
					meta.HasAudio = true
				}
			}

		default:
			r.Seek(chunkSize, io.SeekCurrent)
		}

		// Pad to even boundary
		if chunkSize%2 == 1 {
			r.Seek(1, io.SeekCurrent)
		}
	}

	if meta.Width > 0 && meta.Height > 0 {
		meta.AspectRatio = calculateAspectRatio(meta.Width, meta.Height)
	}
}

// calculateAspectRatio returns common aspect ratio string.
func calculateAspectRatio(width, height int) string {
	if height == 0 {
		return ""
	}
	ratio := float64(width) / float64(height)

	// Check common ratios
	switch {
	case abs(ratio-16.0/9.0) < 0.01:
		return "16:9"
	case abs(ratio-4.0/3.0) < 0.01:
		return "4:3"
	case abs(ratio-21.0/9.0) < 0.01:
		return "21:9"
	case abs(ratio-2.35) < 0.05:
		return "2.35:1"
	case abs(ratio-2.39) < 0.05:
		return "2.39:1"
	case abs(ratio-1.85) < 0.02:
		return "1.85:1"
	case abs(ratio-1.0) < 0.01:
		return "1:1"
	case abs(ratio-9.0/16.0) < 0.01:
		return "9:16"
	default:
		// Find GCD for custom ratio
		gcd := gcd(width, height)
		return strconv.Itoa(width/gcd) + ":" + strconv.Itoa(height/gcd)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
