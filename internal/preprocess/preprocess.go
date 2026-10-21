package preprocess

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
)

// Tensor is the single-image NCHW tensor expected by the ddddocr model.
type Tensor struct {
	Data   []float32
	Height int
	Width  int
}

func Decode(r io.Reader) (image.Image, error) {
	img, format, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	_ = format
	return img, nil
}

func Preprocess(img image.Image) (*Tensor, error) {
	if img == nil {
		return nil, fmt.Errorf("preprocess: nil image")
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("preprocess: invalid image size %dx%d", width, height)
	}

	targetWidth := int(float64(width) * 64 / float64(height))
	if targetWidth < 1 {
		return nil, fmt.Errorf("preprocess: image aspect ratio is too narrow: %dx%d", width, height)
	}

	gray, err := resizeToGray(img, targetWidth, 64)
	if err != nil {
		return nil, err
	}

	pixels := make([]float32, len(gray))
	for i, value := range gray {
		normalized := float32(value) / 255
		pixels[i] = (normalized - 0.5) / 0.5
	}

	return &Tensor{
		Data:   pixels,
		Height: 64,
		Width:  targetWidth,
	}, nil
}

func nrgbaFromColor(c color.Color) color.NRGBA {
	return color.NRGBAModel.Convert(c).(color.NRGBA)
}
