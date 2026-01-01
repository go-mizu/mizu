package meta

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"os"
	"strings"
)

// extractAudioMetadata extracts metadata from an audio file.
func extractAudioMetadata(ctx context.Context, filePath string) (*AudioMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	meta := &AudioMetadata{}

	// Try to detect format and extract metadata
	ext := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(ext, ".mp3"):
		extractMP3Metadata(file, meta)
	case strings.HasSuffix(ext, ".flac"):
		extractFLACMetadata(file, meta)
	case strings.HasSuffix(ext, ".wav"):
		extractWAVMetadata(file, meta)
	case strings.HasSuffix(ext, ".ogg"):
		extractOGGMetadata(file, meta)
	case strings.HasSuffix(ext, ".m4a") || strings.HasSuffix(ext, ".aac"):
		extractM4AMetadata(file, meta)
	}

	// Format duration string
	if meta.Duration > 0 {
		meta.DurationStr = FormatDuration(meta.Duration)
	}

	return meta, nil
}

// extractMP3Metadata extracts ID3 tags from MP3 files.
func extractMP3Metadata(r io.ReadSeeker, meta *AudioMetadata) {
	meta.Codec = "mp3"

	// Try ID3v2 first
	header := make([]byte, 10)
	if _, err := r.Read(header); err != nil {
		return
	}

	if bytes.HasPrefix(header, []byte("ID3")) {
		version := header[3]
		_ = version // ID3v2.x

		// Calculate tag size (syncsafe integer)
		size := int(header[6])<<21 | int(header[7])<<14 | int(header[8])<<7 | int(header[9])

		// Read ID3v2 frames
		tagData := make([]byte, size)
		if _, err := r.Read(tagData); err != nil {
			return
		}

		parseID3v2Frames(tagData, meta, header[3])
	}

	// Also try ID3v1 at end of file
	r.Seek(-128, io.SeekEnd)
	id3v1 := make([]byte, 128)
	if _, err := r.Read(id3v1); err == nil {
		if bytes.HasPrefix(id3v1, []byte("TAG")) {
			parseID3v1(id3v1, meta)
		}
	}

	// Estimate duration from file size and bitrate
	if meta.Bitrate > 0 {
		fileInfo, _ := r.Seek(0, io.SeekEnd)
		meta.Duration = float64(fileInfo*8) / float64(meta.Bitrate*1000)
	}
}

func parseID3v2Frames(data []byte, meta *AudioMetadata, version byte) {
	offset := 0
	for offset+10 < len(data) {
		// Check for padding
		if data[offset] == 0 {
			break
		}

		var frameID string
		var frameSize int

		if version >= 3 {
			// ID3v2.3/2.4 format
			frameID = string(data[offset : offset+4])
			if version == 4 {
				// Syncsafe integer in v2.4
				frameSize = int(data[offset+4])<<21 | int(data[offset+5])<<14 | int(data[offset+6])<<7 | int(data[offset+7])
			} else {
				frameSize = int(binary.BigEndian.Uint32(data[offset+4 : offset+8]))
			}
			offset += 10
		} else {
			// ID3v2.2 format (3-byte frame IDs)
			frameID = string(data[offset : offset+3])
			frameSize = int(data[offset+3])<<16 | int(data[offset+4])<<8 | int(data[offset+5])
			offset += 6
		}

		if frameSize <= 0 || offset+frameSize > len(data) {
			break
		}

		frameData := data[offset : offset+frameSize]
		offset += frameSize

		// Parse text frames
		if len(frameData) > 1 {
			encoding := frameData[0]
			text := decodeTextFrame(frameData[1:], encoding)

			switch frameID {
			case "TIT2", "TT2":
				meta.Title = text
			case "TPE1", "TP1":
				meta.Artist = text
			case "TALB", "TAL":
				meta.Album = text
			case "TPE2", "TP2":
				meta.AlbumArtist = text
			case "TCOM", "TCM":
				meta.Composer = text
			case "TCON", "TCO":
				meta.Genre = parseGenre(text)
			case "TYER", "TYE":
				meta.Year = parseYear(text)
			case "TRCK", "TRK":
				parseTrackNumber(text, meta)
			case "TPOS", "TPA":
				parseDiscNumber(text, meta)
			case "COMM", "COM":
				if len(text) > 4 {
					meta.Comment = text[4:]
				}
			case "TBPM", "TBP":
				meta.BPM = parseInt(text)
			case "TPUB", "TPB":
				meta.Publisher = text
			case "APIC", "PIC":
				meta.HasCoverArt = true
				meta.CoverArtSize = frameSize
			}
		}
	}
}

func parseID3v1(data []byte, meta *AudioMetadata) {
	// ID3v1 has fixed field sizes
	if meta.Title == "" {
		meta.Title = strings.TrimRight(string(data[3:33]), "\x00 ")
	}
	if meta.Artist == "" {
		meta.Artist = strings.TrimRight(string(data[33:63]), "\x00 ")
	}
	if meta.Album == "" {
		meta.Album = strings.TrimRight(string(data[63:93]), "\x00 ")
	}
	if meta.Year == 0 {
		meta.Year = parseYear(string(data[93:97]))
	}
	if meta.Comment == "" {
		meta.Comment = strings.TrimRight(string(data[97:127]), "\x00 ")
	}

	// ID3v1.1 track number
	if data[125] == 0 && data[126] != 0 && meta.TrackNumber == 0 {
		meta.TrackNumber = int(data[126])
	}

	// Genre
	if data[127] < 192 && meta.Genre == "" {
		meta.Genre = id3v1Genres[int(data[127])]
	}
}

func decodeTextFrame(data []byte, encoding byte) string {
	switch encoding {
	case 0: // ISO-8859-1
		return strings.TrimRight(string(data), "\x00")
	case 1: // UTF-16 with BOM
		if len(data) >= 2 {
			var order binary.ByteOrder = binary.LittleEndian
			if data[0] == 0xFE && data[1] == 0xFF {
				order = binary.BigEndian
				data = data[2:]
			} else if data[0] == 0xFF && data[1] == 0xFE {
				data = data[2:]
			}
			return decodeUTF16(data, order)
		}
	case 2: // UTF-16BE without BOM
		return decodeUTF16(data, binary.BigEndian)
	case 3: // UTF-8
		return strings.TrimRight(string(data), "\x00")
	}
	return strings.TrimRight(string(data), "\x00")
}

func decodeUTF16(data []byte, order binary.ByteOrder) string {
	var result strings.Builder
	for i := 0; i+1 < len(data); i += 2 {
		ch := order.Uint16(data[i : i+2])
		if ch == 0 {
			break
		}
		result.WriteRune(rune(ch))
	}
	return result.String()
}

// extractFLACMetadata extracts metadata from FLAC files.
func extractFLACMetadata(r io.ReadSeeker, meta *AudioMetadata) {
	meta.Codec = "flac"

	// Read and verify FLAC signature
	sig := make([]byte, 4)
	if _, err := r.Read(sig); err != nil || string(sig) != "fLaC" {
		return
	}

	// Read metadata blocks
	for {
		blockHeader := make([]byte, 4)
		if _, err := r.Read(blockHeader); err != nil {
			break
		}

		isLast := blockHeader[0]&0x80 != 0
		blockType := blockHeader[0] & 0x7F
		blockSize := int(blockHeader[1])<<16 | int(blockHeader[2])<<8 | int(blockHeader[3])

		blockData := make([]byte, blockSize)
		if _, err := r.Read(blockData); err != nil {
			break
		}

		switch blockType {
		case 0: // STREAMINFO
			if len(blockData) >= 34 {
				// Minimum/maximum block sizes, frame sizes
				sampleRate := int(blockData[10])<<12 | int(blockData[11])<<4 | int(blockData[12])>>4
				channels := ((int(blockData[12]) >> 1) & 0x07) + 1
				bitsPerSample := ((int(blockData[12]) & 0x01) << 4) | (int(blockData[13]) >> 4) + 1
				totalSamples := int64(blockData[13]&0x0F)<<32 | int64(blockData[14])<<24 | int64(blockData[15])<<16 | int64(blockData[16])<<8 | int64(blockData[17])

				meta.SampleRate = sampleRate
				meta.Channels = channels
				meta.BitDepth = bitsPerSample
				if sampleRate > 0 {
					meta.Duration = float64(totalSamples) / float64(sampleRate)
				}
			}
		case 4: // VORBIS_COMMENT
			parseVorbisComment(blockData, meta)
		case 6: // PICTURE
			meta.HasCoverArt = true
			meta.CoverArtSize = blockSize
		}

		if isLast {
			break
		}
	}
}

func parseVorbisComment(data []byte, meta *AudioMetadata) {
	if len(data) < 4 {
		return
	}

	// Skip vendor string
	vendorLen := int(binary.LittleEndian.Uint32(data[:4]))
	offset := 4 + vendorLen
	if offset+4 > len(data) {
		return
	}

	// Read comments
	commentCount := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
	offset += 4

	for i := 0; i < commentCount && offset+4 <= len(data); i++ {
		commentLen := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		offset += 4
		if offset+commentLen > len(data) {
			break
		}

		comment := string(data[offset : offset+commentLen])
		offset += commentLen

		parts := strings.SplitN(comment, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToUpper(parts[0])
		value := parts[1]

		switch key {
		case "TITLE":
			meta.Title = value
		case "ARTIST":
			meta.Artist = value
		case "ALBUM":
			meta.Album = value
		case "ALBUMARTIST":
			meta.AlbumArtist = value
		case "COMPOSER":
			meta.Composer = value
		case "GENRE":
			meta.Genre = value
		case "DATE":
			meta.Year = parseYear(value)
		case "TRACKNUMBER":
			parseTrackNumber(value, meta)
		case "DISCNUMBER":
			parseDiscNumber(value, meta)
		case "COMMENT":
			meta.Comment = value
		}
	}
}

// extractWAVMetadata extracts metadata from WAV files.
func extractWAVMetadata(r io.ReadSeeker, meta *AudioMetadata) {
	meta.Codec = "pcm"

	// Read RIFF header
	header := make([]byte, 12)
	if _, err := r.Read(header); err != nil {
		return
	}

	if string(header[:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return
	}

	// Read chunks
	for {
		chunkHeader := make([]byte, 8)
		if _, err := r.Read(chunkHeader); err != nil {
			break
		}

		chunkID := string(chunkHeader[:4])
		chunkSize := int(binary.LittleEndian.Uint32(chunkHeader[4:8]))

		if chunkID == "fmt " {
			fmtData := make([]byte, chunkSize)
			r.Read(fmtData)

			if len(fmtData) >= 16 {
				// audioFormat := binary.LittleEndian.Uint16(fmtData[0:2])
				meta.Channels = int(binary.LittleEndian.Uint16(fmtData[2:4]))
				meta.SampleRate = int(binary.LittleEndian.Uint32(fmtData[4:8]))
				// byteRate := binary.LittleEndian.Uint32(fmtData[8:12])
				// blockAlign := binary.LittleEndian.Uint16(fmtData[12:14])
				meta.BitDepth = int(binary.LittleEndian.Uint16(fmtData[14:16]))
				meta.Bitrate = meta.SampleRate * meta.Channels * meta.BitDepth / 1000
			}
		} else if chunkID == "data" {
			// Calculate duration from data size
			if meta.SampleRate > 0 && meta.Channels > 0 && meta.BitDepth > 0 {
				bytesPerSample := meta.BitDepth / 8
				totalSamples := chunkSize / (meta.Channels * bytesPerSample)
				meta.Duration = float64(totalSamples) / float64(meta.SampleRate)
			}
			break
		} else {
			// Skip chunk
			r.Seek(int64(chunkSize), io.SeekCurrent)
		}

		// Pad to even byte boundary
		if chunkSize%2 == 1 {
			r.Seek(1, io.SeekCurrent)
		}
	}
}

// extractOGGMetadata extracts metadata from OGG Vorbis files.
func extractOGGMetadata(r io.ReadSeeker, meta *AudioMetadata) {
	meta.Codec = "vorbis"

	// Read OGG page header
	header := make([]byte, 27)
	if _, err := r.Read(header); err != nil {
		return
	}

	if string(header[:4]) != "OggS" {
		return
	}

	// Read segment table
	segments := int(header[26])
	segTable := make([]byte, segments)
	r.Read(segTable)

	// Calculate page size
	pageSize := 0
	for _, seg := range segTable {
		pageSize += int(seg)
	}

	// Read first page (identification header)
	pageData := make([]byte, pageSize)
	r.Read(pageData)

	if len(pageData) > 7 && string(pageData[1:7]) == "vorbis" {
		// Parse Vorbis identification header
		if len(pageData) >= 30 {
			meta.Channels = int(pageData[11])
			meta.SampleRate = int(binary.LittleEndian.Uint32(pageData[12:16]))
			// Bitrate info is in bytes 16-28
		}
	}

	// Read second page for comments
	header2 := make([]byte, 27)
	r.Read(header2)
	if string(header2[:4]) == "OggS" {
		segments2 := int(header2[26])
		segTable2 := make([]byte, segments2)
		r.Read(segTable2)

		pageSize2 := 0
		for _, seg := range segTable2 {
			pageSize2 += int(seg)
		}

		pageData2 := make([]byte, pageSize2)
		r.Read(pageData2)

		// Skip vorbis header
		if len(pageData2) > 7 && string(pageData2[1:7]) == "vorbis" {
			parseVorbisComment(pageData2[7:], meta)
		}
	}
}

// extractM4AMetadata extracts metadata from M4A/AAC files.
func extractM4AMetadata(r io.ReadSeeker, meta *AudioMetadata) {
	meta.Codec = "aac"

	// M4A uses the ISO Base Media File Format (similar to MP4)
	// Read atoms/boxes
	for {
		header := make([]byte, 8)
		if _, err := r.Read(header); err != nil {
			break
		}

		size := int64(binary.BigEndian.Uint32(header[:4]))
		atomType := string(header[4:8])

		if size < 8 {
			break
		}

		switch atomType {
		case "moov":
			// Container atom, recurse
			continue
		case "trak", "mdia", "minf", "stbl":
			// Container atoms, continue
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
				} else {
					timescale = int64(binary.BigEndian.Uint32(data[20:24]))
					duration = int64(binary.BigEndian.Uint64(data[24:32]))
				}
				if timescale > 0 {
					meta.Duration = float64(duration) / float64(timescale)
				}
			}
		case "udta", "meta", "ilst":
			// Metadata container
			if atomType == "meta" {
				// Skip 4-byte version
				r.Seek(4, io.SeekCurrent)
			}
			continue
		default:
			// Skip atom
			r.Seek(size-8, io.SeekCurrent)
		}
	}
}

// Helper functions
func parseGenre(s string) string {
	// Handle ID3v1 genre numbers like "(17)" or "(17)Rock"
	if len(s) > 2 && s[0] == '(' {
		end := strings.Index(s, ")")
		if end > 1 {
			num := parseInt(s[1:end])
			if num >= 0 && num < len(id3v1Genres) {
				return id3v1Genres[num]
			}
		}
	}
	return s
}

func parseYear(s string) int {
	if len(s) >= 4 {
		return parseInt(s[:4])
	}
	return parseInt(s)
}

func parseTrackNumber(s string, meta *AudioMetadata) {
	parts := strings.Split(s, "/")
	meta.TrackNumber = parseInt(parts[0])
	if len(parts) > 1 {
		meta.TrackTotal = parseInt(parts[1])
	}
}

func parseDiscNumber(s string, meta *AudioMetadata) {
	parts := strings.Split(s, "/")
	meta.DiscNumber = parseInt(parts[0])
	if len(parts) > 1 {
		meta.DiscTotal = parseInt(parts[1])
	}
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	result := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		} else {
			break
		}
	}
	return result
}

// ID3v1 genre list
var id3v1Genres = []string{
	"Blues", "Classic Rock", "Country", "Dance", "Disco", "Funk", "Grunge", "Hip-Hop",
	"Jazz", "Metal", "New Age", "Oldies", "Other", "Pop", "R&B", "Rap",
	"Reggae", "Rock", "Techno", "Industrial", "Alternative", "Ska", "Death Metal", "Pranks",
	"Soundtrack", "Euro-Techno", "Ambient", "Trip-Hop", "Vocal", "Jazz+Funk", "Fusion", "Trance",
	"Classical", "Instrumental", "Acid", "House", "Game", "Sound Clip", "Gospel", "Noise",
	"Alternative Rock", "Bass", "Soul", "Punk", "Space", "Meditative", "Instrumental Pop", "Instrumental Rock",
	"Ethnic", "Gothic", "Darkwave", "Techno-Industrial", "Electronic", "Pop-Folk", "Eurodance", "Dream",
	"Southern Rock", "Comedy", "Cult", "Gangsta", "Top 40", "Christian Rap", "Pop/Funk", "Jungle",
	"Native American", "Cabaret", "New Wave", "Psychedelic", "Rave", "Showtunes", "Trailer", "Lo-Fi",
	"Tribal", "Acid Punk", "Acid Jazz", "Polka", "Retro", "Musical", "Rock & Roll", "Hard Rock",
}
