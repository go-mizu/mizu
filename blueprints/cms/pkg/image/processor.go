// Package image provides image processing capabilities.
package image

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	"github.com/go-mizu/blueprints/cms/config"

	// Import for side effects - register image formats
	_ "golang.org/x/image/webp"
)

// Processor handles image processing operations.
type Processor struct{}

// NewProcessor creates a new image processor.
func NewProcessor() *Processor {
	return &Processor{}
}

// ProcessedImage represents a processed image variant.
type ProcessedImage struct {
	Reader   io.ReadCloser
	Width    int
	Height   int
	Filename string
	MimeType string
	Filesize int64
}

// CanProcess returns true if the mime type can be processed.
func (p *Processor) CanProcess(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

// GetDimensions returns the dimensions of an image.
func (p *Processor) GetDimensions(reader io.Reader) (width, height int, err error) {
	cfg, _, err := image.DecodeConfig(reader)
	if err != nil {
		return 0, 0, fmt.Errorf("decode config: %w", err)
	}
	return cfg.Width, cfg.Height, nil
}

// Resize resizes an image maintaining aspect ratio.
func (p *Processor) Resize(src io.Reader, maxWidth, maxHeight int) (io.ReadCloser, int, int, error) {
	img, format, err := image.Decode(src)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("decode: %w", err)
	}

	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	// Calculate new dimensions maintaining aspect ratio
	newWidth, newHeight := calculateDimensions(origWidth, origHeight, maxWidth, maxHeight)

	// Resize using simple nearest-neighbor (for production, use a proper resize library)
	resized := resizeImage(img, newWidth, newHeight)

	// Encode
	var buf bytes.Buffer
	if err := encodeImage(&buf, resized, format); err != nil {
		return nil, 0, 0, fmt.Errorf("encode: %w", err)
	}

	return io.NopCloser(bytes.NewReader(buf.Bytes())), newWidth, newHeight, nil
}

// ResizeWithFocalPoint resizes with focal point consideration.
func (p *Processor) ResizeWithFocalPoint(src io.Reader, maxWidth, maxHeight int, focalX, focalY float64) (io.ReadCloser, int, int, error) {
	img, format, err := image.Decode(src)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("decode: %w", err)
	}

	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	// Calculate crop area centered on focal point
	cropX, cropY, cropW, cropH := calculateFocalCrop(origWidth, origHeight, maxWidth, maxHeight, focalX, focalY)

	// Crop first
	cropped := cropImage(img, cropX, cropY, cropW, cropH)

	// Then resize to target
	resized := resizeImage(cropped, maxWidth, maxHeight)

	var buf bytes.Buffer
	if err := encodeImage(&buf, resized, format); err != nil {
		return nil, 0, 0, fmt.Errorf("encode: %w", err)
	}

	return io.NopCloser(bytes.NewReader(buf.Bytes())), maxWidth, maxHeight, nil
}

// Crop crops an image to specified dimensions.
func (p *Processor) Crop(src io.Reader, x, y, width, height int) (io.ReadCloser, error) {
	img, format, err := image.Decode(src)
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	cropped := cropImage(img, x, y, width, height)

	var buf bytes.Buffer
	if err := encodeImage(&buf, cropped, format); err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

// GenerateSizes generates all configured image sizes.
func (p *Processor) GenerateSizes(src io.Reader, sizes []config.ImageSize, focalX, focalY float64) (map[string]*ProcessedImage, error) {
	// Read source into buffer for multiple operations
	var srcBuf bytes.Buffer
	if _, err := io.Copy(&srcBuf, src); err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}

	// Decode once to get format
	img, format, err := image.Decode(bytes.NewReader(srcBuf.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	result := make(map[string]*ProcessedImage)

	for _, size := range sizes {
		var resized image.Image
		var newWidth, newHeight int

		if size.Crop {
			// Crop to exact dimensions using focal point
			cropX, cropY, cropW, cropH := calculateFocalCrop(origWidth, origHeight, size.Width, size.Height, focalX, focalY)
			cropped := cropImage(img, cropX, cropY, cropW, cropH)
			resized = resizeImage(cropped, size.Width, size.Height)
			newWidth, newHeight = size.Width, size.Height
		} else {
			// Resize maintaining aspect ratio
			newWidth, newHeight = calculateDimensions(origWidth, origHeight, size.Width, size.Height)
			resized = resizeImage(img, newWidth, newHeight)
		}

		var buf bytes.Buffer
		if err := encodeImage(&buf, resized, format); err != nil {
			return nil, fmt.Errorf("encode size %s: %w", size.Name, err)
		}

		ext := "." + format
		if format == "jpeg" {
			ext = ".jpg"
		}

		result[size.Name] = &ProcessedImage{
			Reader:   io.NopCloser(bytes.NewReader(buf.Bytes())),
			Width:    newWidth,
			Height:   newHeight,
			Filename: size.Name + ext,
			MimeType: "image/" + format,
			Filesize: int64(buf.Len()),
		}
	}

	return result, nil
}

func calculateDimensions(origWidth, origHeight, maxWidth, maxHeight int) (int, int) {
	if maxWidth == 0 && maxHeight == 0 {
		return origWidth, origHeight
	}

	ratio := float64(origWidth) / float64(origHeight)

	if maxWidth == 0 {
		return int(float64(maxHeight) * ratio), maxHeight
	}
	if maxHeight == 0 {
		return maxWidth, int(float64(maxWidth) / ratio)
	}

	// Fit within bounds
	newWidth := maxWidth
	newHeight := int(float64(maxWidth) / ratio)

	if newHeight > maxHeight {
		newHeight = maxHeight
		newWidth = int(float64(maxHeight) * ratio)
	}

	return newWidth, newHeight
}

func calculateFocalCrop(origWidth, origHeight, targetWidth, targetHeight int, focalX, focalY float64) (x, y, w, h int) {
	targetRatio := float64(targetWidth) / float64(targetHeight)
	origRatio := float64(origWidth) / float64(origHeight)

	if origRatio > targetRatio {
		// Original is wider - crop width
		h = origHeight
		w = int(float64(h) * targetRatio)
		y = 0
		x = int(focalX*float64(origWidth)) - w/2
		if x < 0 {
			x = 0
		}
		if x+w > origWidth {
			x = origWidth - w
		}
	} else {
		// Original is taller - crop height
		w = origWidth
		h = int(float64(w) / targetRatio)
		x = 0
		y = int(focalY*float64(origHeight)) - h/2
		if y < 0 {
			y = 0
		}
		if y+h > origHeight {
			y = origHeight - h
		}
	}

	return x, y, w, h
}

func cropImage(img image.Image, x, y, width, height int) image.Image {
	bounds := img.Bounds()

	// Clamp values
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+width > bounds.Dx() {
		width = bounds.Dx() - x
	}
	if y+height > bounds.Dy() {
		height = bounds.Dy() - y
	}

	cropped := image.NewRGBA(image.Rect(0, 0, width, height))
	for cy := 0; cy < height; cy++ {
		for cx := 0; cx < width; cx++ {
			cropped.Set(cx, cy, img.At(bounds.Min.X+x+cx, bounds.Min.Y+y+cy))
		}
	}

	return cropped
}

func resizeImage(img image.Image, width, height int) image.Image {
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	resized := image.NewRGBA(image.Rect(0, 0, width, height))

	xRatio := float64(origWidth) / float64(width)
	yRatio := float64(origHeight) / float64(height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := int(float64(x) * xRatio)
			srcY := int(float64(y) * yRatio)
			resized.Set(x, y, img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
		}
	}

	return resized
}

func encodeImage(w io.Writer, img image.Image, format string) error {
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return jpeg.Encode(w, img, &jpeg.Options{Quality: 85})
	case "png":
		return png.Encode(w, img)
	case "gif":
		return gif.Encode(w, img, nil)
	default:
		// Default to JPEG
		return jpeg.Encode(w, img, &jpeg.Options{Quality: 85})
	}
}

// GenerateFilename generates a filename for a size variant.
func GenerateFilename(originalFilename, sizeName string) string {
	ext := filepath.Ext(originalFilename)
	base := strings.TrimSuffix(originalFilename, ext)
	return base + "-" + sizeName + ext
}
