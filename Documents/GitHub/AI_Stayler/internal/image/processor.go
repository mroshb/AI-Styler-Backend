package image

import (
	"context"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// ImageProcessorImpl implements the ImageProcessor interface
type ImageProcessorImpl struct{}

// NewImageProcessor creates a new image processor
func NewImageProcessor() *ImageProcessorImpl {
	return &ImageProcessorImpl{}
}

// ProcessImage processes an image and returns processed data with dimensions
func (p *ImageProcessorImpl) ProcessImage(ctx context.Context, data []byte, fileName string) ([]byte, int, int, error) {
	// Decode image to get dimensions
	img, _, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to decode image: %w", err)
	}

	// Get dimensions
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// For now, return original data (in production, you might resize, compress, etc.)
	return data, width, height, nil
}

// GenerateThumbnail generates a thumbnail of the specified dimensions
func (p *ImageProcessorImpl) GenerateThumbnail(ctx context.Context, data []byte, fileName string, width, height int) ([]byte, error) {
	// Decode image
	img, _, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Get original dimensions
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	// Calculate scaling factor
	scaleX := float64(width) / float64(origWidth)
	scaleY := float64(height) / float64(origHeight)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Calculate new dimensions
	newWidth := int(float64(origWidth) * scale)
	newHeight := int(float64(origHeight) * scale)

	// Create new image
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Simple nearest neighbor scaling (in production, use better algorithms)
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := int(float64(x) / scale)
			srcY := int(float64(y) / scale)
			if srcX < origWidth && srcY < origHeight {
				newImg.Set(x, y, img.At(srcX, srcY))
			}
		}
	}

	// Encode as JPEG
	var buf strings.Builder
	err = jpeg.Encode(&buf, newImg, &jpeg.Options{Quality: 80})
	if err != nil {
		return nil, fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return []byte(buf.String()), nil
}

// ResizeImage resizes an image to the specified dimensions
func (p *ImageProcessorImpl) ResizeImage(ctx context.Context, data []byte, fileName string, width, height int) ([]byte, error) {
	// Decode image
	img, _, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Create new image
	newImg := image.NewRGBA(image.Rect(0, 0, width, height))

	// Get original dimensions
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	// Calculate scaling factors
	scaleX := float64(width) / float64(origWidth)
	scaleY := float64(height) / float64(origHeight)

	// Simple nearest neighbor scaling
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := int(float64(x) / scaleX)
			srcY := int(float64(y) / scaleY)
			if srcX < origWidth && srcY < origHeight {
				newImg.Set(x, y, img.At(srcX, srcY))
			}
		}
	}

	// Determine output format based on original
	var buf strings.Builder
	if strings.HasSuffix(strings.ToLower(fileName), ".png") {
		err = png.Encode(&buf, newImg)
	} else if strings.HasSuffix(strings.ToLower(fileName), ".gif") {
		err = gif.Encode(&buf, newImg, nil)
	} else {
		// Default to JPEG
		err = jpeg.Encode(&buf, newImg, &jpeg.Options{Quality: 90})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode resized image: %w", err)
	}

	return []byte(buf.String()), nil
}

// ValidateImage validates an image
func (p *ImageProcessorImpl) ValidateImage(ctx context.Context, data []byte, fileName string, mimeType string) error {
	// Check file size
	if len(data) == 0 {
		return fmt.Errorf("empty file")
	}

	// Check MIME type
	if !p.isValidMimeType(mimeType) {
		return fmt.Errorf("unsupported MIME type: %s", mimeType)
	}

	// Try to decode the image
	_, _, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return fmt.Errorf("invalid image format: %w", err)
	}

	return nil
}

// GetImageDimensions gets image dimensions
func (p *ImageProcessorImpl) GetImageDimensions(ctx context.Context, data []byte) (int, int, error) {
	img, _, err := image.Decode(strings.NewReader(string(data)))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy(), nil
}

// isValidMimeType checks if the MIME type is supported
func (p *ImageProcessorImpl) isValidMimeType(mimeType string) bool {
	for _, validType := range SupportedImageTypes {
		if mimeType == validType {
			return true
		}
	}
	return false
}
