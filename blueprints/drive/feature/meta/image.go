package meta

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"strings"

	_ "golang.org/x/image/webp"
)

// extractImageMetadata extracts metadata from an image file.
func extractImageMetadata(ctx context.Context, filePath string) (*ImageMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	meta := &ImageMetadata{}

	// Get basic image dimensions
	config, format, err := image.DecodeConfig(file)
	if err == nil {
		meta.Width = config.Width
		meta.Height = config.Height
		meta.ColorSpace = format
	}

	// Reset file position
	file.Seek(0, 0)

	// Try to extract EXIF data for JPEG
	if strings.HasSuffix(strings.ToLower(filePath), ".jpg") ||
		strings.HasSuffix(strings.ToLower(filePath), ".jpeg") {
		if exifMeta := extractExifData(file); exifMeta != nil {
			// Merge EXIF data
			meta.Make = exifMeta.Make
			meta.Model = exifMeta.Model
			meta.LensModel = exifMeta.LensModel
			meta.Software = exifMeta.Software
			meta.DateTimeOriginal = exifMeta.DateTimeOriginal
			meta.ExposureTime = exifMeta.ExposureTime
			meta.FNumber = exifMeta.FNumber
			meta.ISO = exifMeta.ISO
			meta.FocalLength = exifMeta.FocalLength
			meta.FocalLength35mm = exifMeta.FocalLength35mm
			meta.Flash = exifMeta.Flash
			meta.Orientation = exifMeta.Orientation
			meta.GPSLatitude = exifMeta.GPSLatitude
			meta.GPSLongitude = exifMeta.GPSLongitude
			meta.GPSAltitude = exifMeta.GPSAltitude
		}
	}

	// Check for GIF animation
	if strings.HasSuffix(strings.ToLower(filePath), ".gif") {
		file.Seek(0, 0)
		frameCount := countGifFrames(file)
		if frameCount > 1 {
			meta.IsAnimated = true
			meta.FrameCount = frameCount
		}
	}

	return meta, nil
}

// extractExifData extracts EXIF metadata from JPEG images.
// This is a simplified implementation that reads basic EXIF tags.
func extractExifData(r io.ReadSeeker) *ImageMetadata {
	meta := &ImageMetadata{}

	// Read JPEG header
	header := make([]byte, 2)
	if _, err := r.Read(header); err != nil {
		return nil
	}
	if header[0] != 0xFF || header[1] != 0xD8 {
		return nil // Not a JPEG
	}

	// Find APP1 marker (EXIF)
	for {
		marker := make([]byte, 2)
		if _, err := r.Read(marker); err != nil {
			return meta
		}
		if marker[0] != 0xFF {
			return meta
		}

		// Check for APP1 (EXIF)
		if marker[1] == 0xE1 {
			// Read segment length
			lenBytes := make([]byte, 2)
			if _, err := r.Read(lenBytes); err != nil {
				return meta
			}
			segLen := int(binary.BigEndian.Uint16(lenBytes)) - 2

			// Read EXIF data
			exifData := make([]byte, segLen)
			if _, err := r.Read(exifData); err != nil {
				return meta
			}

			// Check for "Exif\0\0" header
			if segLen > 6 && string(exifData[:4]) == "Exif" {
				parseExif(exifData[6:], meta)
			}
			return meta
		}

		// Skip to next marker
		if marker[1] >= 0xD0 && marker[1] <= 0xD9 {
			continue // Standalone markers
		}
		if marker[1] == 0xD8 || marker[1] == 0xD9 {
			continue // SOI/EOI
		}

		// Read and skip segment
		lenBytes := make([]byte, 2)
		if _, err := r.Read(lenBytes); err != nil {
			return meta
		}
		segLen := int(binary.BigEndian.Uint16(lenBytes)) - 2
		if segLen > 0 {
			r.Seek(int64(segLen), io.SeekCurrent)
		}
	}
}

// parseExif parses TIFF-format EXIF data.
func parseExif(data []byte, meta *ImageMetadata) {
	if len(data) < 8 {
		return
	}

	// Determine byte order
	var order binary.ByteOrder
	if string(data[:2]) == "II" {
		order = binary.LittleEndian
	} else if string(data[:2]) == "MM" {
		order = binary.BigEndian
	} else {
		return
	}

	// Check TIFF magic number
	if order.Uint16(data[2:4]) != 42 {
		return
	}

	// Get IFD0 offset
	ifdOffset := order.Uint32(data[4:8])
	if int(ifdOffset) >= len(data)-2 {
		return
	}

	// Parse IFD0
	parseIFD(data, int(ifdOffset), order, meta, 0)
}

// parseIFD parses an Image File Directory.
func parseIFD(data []byte, offset int, order binary.ByteOrder, meta *ImageMetadata, depth int) {
	if depth > 2 || offset+2 > len(data) {
		return
	}

	entryCount := int(order.Uint16(data[offset : offset+2]))
	offset += 2

	for i := 0; i < entryCount && offset+12 <= len(data); i++ {
		tag := order.Uint16(data[offset : offset+2])
		tagType := order.Uint16(data[offset+2 : offset+4])
		count := order.Uint32(data[offset+4 : offset+8])
		valueOffset := data[offset+8 : offset+12]

		parseExifTag(tag, tagType, count, valueOffset, data, order, meta)
		offset += 12
	}

	// Check for EXIF SubIFD
	if offset+4 <= len(data) {
		nextIFD := order.Uint32(data[offset : offset+4])
		if nextIFD > 0 && int(nextIFD) < len(data) && depth == 0 {
			parseIFD(data, int(nextIFD), order, meta, depth+1)
		}
	}
}

// parseExifTag parses a single EXIF tag.
func parseExifTag(tag, tagType uint16, count uint32, valueOffset []byte, data []byte, order binary.ByteOrder, meta *ImageMetadata) {
	// Get value based on type
	getValue := func() uint32 {
		if tagType == 3 { // SHORT
			return uint32(order.Uint16(valueOffset))
		}
		return order.Uint32(valueOffset)
	}

	getStringValue := func() string {
		offset := order.Uint32(valueOffset)
		if int(offset)+int(count) > len(data) || count == 0 {
			return ""
		}
		s := string(data[offset : offset+count-1])
		return strings.TrimSpace(s)
	}

	getRational := func() (num, denom uint32) {
		offset := order.Uint32(valueOffset)
		if int(offset)+8 > len(data) {
			return 0, 1
		}
		num = order.Uint32(data[offset : offset+4])
		denom = order.Uint32(data[offset+4 : offset+8])
		return
	}

	switch tag {
	case 0x010F: // Make
		meta.Make = getStringValue()
	case 0x0110: // Model
		meta.Model = getStringValue()
	case 0x0112: // Orientation
		meta.Orientation = int(getValue())
	case 0x0131: // Software
		meta.Software = getStringValue()
	case 0x9003: // DateTimeOriginal
		meta.DateTimeOriginal = getStringValue()
	case 0x829A: // ExposureTime
		num, denom := getRational()
		if denom > 0 {
			if num < denom {
				meta.ExposureTime = "1/" + itoa(int(denom/num))
			} else {
				meta.ExposureTime = ftoa(float64(num) / float64(denom))
			}
		}
	case 0x829D: // FNumber
		num, denom := getRational()
		if denom > 0 {
			meta.FNumber = float64(num) / float64(denom)
		}
	case 0x8827: // ISO
		meta.ISO = int(getValue())
	case 0x920A: // FocalLength
		num, denom := getRational()
		if denom > 0 {
			meta.FocalLength = float64(num) / float64(denom)
		}
	case 0xA405: // FocalLengthIn35mmFilm
		meta.FocalLength35mm = int(getValue())
	case 0x9209: // Flash
		flashVal := getValue()
		if flashVal&1 == 1 {
			meta.Flash = "On"
		} else {
			meta.Flash = "Off"
		}
	case 0x8769: // ExifIFDPointer
		offset := getValue()
		if int(offset) < len(data) {
			parseIFD(data, int(offset), order, meta, 1)
		}
	case 0x8825: // GPSInfoIFDPointer
		offset := getValue()
		if int(offset) < len(data) {
			parseGPSIFD(data, int(offset), order, meta)
		}
	}
}

// parseGPSIFD parses GPS data from EXIF.
func parseGPSIFD(data []byte, offset int, order binary.ByteOrder, meta *ImageMetadata) {
	if offset+2 > len(data) {
		return
	}

	entryCount := int(order.Uint16(data[offset : offset+2]))
	offset += 2

	var latRef, lonRef string
	var lat, lon [3]float64

	for i := 0; i < entryCount && offset+12 <= len(data); i++ {
		tag := order.Uint16(data[offset : offset+2])
		tagType := order.Uint16(data[offset+2 : offset+4])
		count := order.Uint32(data[offset+4 : offset+8])
		valueOffset := data[offset+8 : offset+12]

		switch tag {
		case 1: // GPSLatitudeRef
			if tagType == 2 && count >= 1 {
				latRef = string(valueOffset[0])
			}
		case 2: // GPSLatitude
			lat = parseGPSCoord(data, order.Uint32(valueOffset), order)
		case 3: // GPSLongitudeRef
			if tagType == 2 && count >= 1 {
				lonRef = string(valueOffset[0])
			}
		case 4: // GPSLongitude
			lon = parseGPSCoord(data, order.Uint32(valueOffset), order)
		case 6: // GPSAltitude
			valOffset := order.Uint32(valueOffset)
			if int(valOffset)+8 <= len(data) {
				num := order.Uint32(data[valOffset : valOffset+4])
				denom := order.Uint32(data[valOffset+4 : valOffset+8])
				if denom > 0 {
					meta.GPSAltitude = float64(num) / float64(denom)
				}
			}
		}
		offset += 12
	}

	// Calculate decimal coordinates
	meta.GPSLatitude = lat[0] + lat[1]/60 + lat[2]/3600
	if latRef == "S" {
		meta.GPSLatitude = -meta.GPSLatitude
	}

	meta.GPSLongitude = lon[0] + lon[1]/60 + lon[2]/3600
	if lonRef == "W" {
		meta.GPSLongitude = -meta.GPSLongitude
	}
}

func parseGPSCoord(data []byte, offset uint32, order binary.ByteOrder) [3]float64 {
	var result [3]float64
	for i := 0; i < 3; i++ {
		pos := int(offset) + i*8
		if pos+8 > len(data) {
			break
		}
		num := order.Uint32(data[pos : pos+4])
		denom := order.Uint32(data[pos+4 : pos+8])
		if denom > 0 {
			result[i] = float64(num) / float64(denom)
		}
	}
	return result
}

// countGifFrames counts frames in a GIF file.
func countGifFrames(r io.ReadSeeker) int {
	// Read GIF header
	header := make([]byte, 13)
	if _, err := r.Read(header); err != nil {
		return 1
	}

	// Check signature
	if !bytes.HasPrefix(header, []byte("GIF")) {
		return 1
	}

	// Skip global color table if present
	flags := header[10]
	if flags&0x80 != 0 {
		colorTableSize := 3 * (1 << ((flags & 0x07) + 1))
		r.Seek(int64(colorTableSize), io.SeekCurrent)
	}

	frameCount := 0
	for {
		var intro [1]byte
		if _, err := r.Read(intro[:]); err != nil {
			break
		}

		switch intro[0] {
		case 0x2C: // Image descriptor
			frameCount++
			// Skip image descriptor
			r.Seek(8, io.SeekCurrent)
			// Check for local color table
			var localFlags [1]byte
			r.Read(localFlags[:])
			if localFlags[0]&0x80 != 0 {
				localTableSize := 3 * (1 << ((localFlags[0] & 0x07) + 1))
				r.Seek(int64(localTableSize), io.SeekCurrent)
			}
			// Skip LZW minimum code size
			r.Seek(1, io.SeekCurrent)
			// Skip sub-blocks
			for {
				var blockSize [1]byte
				if _, err := r.Read(blockSize[:]); err != nil || blockSize[0] == 0 {
					break
				}
				r.Seek(int64(blockSize[0]), io.SeekCurrent)
			}
		case 0x21: // Extension
			var extType [1]byte
			r.Read(extType[:])
			// Skip sub-blocks
			for {
				var blockSize [1]byte
				if _, err := r.Read(blockSize[:]); err != nil || blockSize[0] == 0 {
					break
				}
				r.Seek(int64(blockSize[0]), io.SeekCurrent)
			}
		case 0x3B: // Trailer
			return frameCount
		default:
			return frameCount
		}
	}

	return frameCount
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b) - 1
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		b[i] = byte('0' + n%10)
		n /= 10
		i--
	}
	if neg {
		b[i] = '-'
		i--
	}
	return string(b[i+1:])
}

func ftoa(f float64) string {
	if f < 0 {
		return "-" + ftoa(-f)
	}
	whole := int(f)
	frac := int((f - float64(whole)) * 10)
	return itoa(whole) + "." + itoa(frac)
}
