package preprocess

import (
	"fmt"
	"image"
	"math"
)

const pillowPrecisionBits = 22

type filterCoeff struct {
	start   int
	weights []int32
}

func resizeToGray(img image.Image, outWidth, outHeight int) ([]uint8, error) {
	if outWidth < 1 || outHeight < 1 {
		return nil, fmt.Errorf("resize: invalid output size %dx%d", outWidth, outHeight)
	}

	switch src := img.(type) {
	case *image.Paletted:
		return resizePalettedGray(src, outWidth, outHeight)
	case *image.Gray:
		return resizeGray(src, outWidth, outHeight)
	case *image.Gray16:
		return resizeGray16(src, outWidth, outHeight)
	default:
		return resizeNRGBAGray(img, outWidth, outHeight)
	}
}

func resizeNRGBAGray(img image.Image, outWidth, outHeight int) ([]uint8, error) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("resize: invalid input size %dx%d", width, height)
	}

	rgba := make([]uint8, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := nrgbaFromColor(img.At(bounds.Min.X+x, bounds.Min.Y+y))
			index := (y*width + x) * 4
			rgba[index] = c.R
			rgba[index+1] = c.G
			rgba[index+2] = c.B
			rgba[index+3] = c.A
		}
	}

	premultiplied := premultiplyRGBA(rgba)
	resized := resampleChannels(premultiplied, width, height, 4, outWidth, outHeight)

	gray := make([]uint8, outWidth*outHeight)
	for i := range gray {
		index := i * 4
		alpha := resized[index+3]
		r := unpremultiply(resized[index], alpha)
		g := unpremultiply(resized[index+1], alpha)
		b := unpremultiply(resized[index+2], alpha)
		gray[i] = pillowGray(r, g, b)
	}
	return gray, nil
}

func resizeGray(src *image.Gray, outWidth, outHeight int) ([]uint8, error) {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("resize: invalid gray image size %dx%d", width, height)
	}

	pixels := make([]uint8, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixels[y*width+x] = src.GrayAt(bounds.Min.X+x, bounds.Min.Y+y).Y
		}
	}
	return resampleChannels(pixels, width, height, 1, outWidth, outHeight), nil
}

func resizeGray16(src *image.Gray16, outWidth, outHeight int) ([]uint8, error) {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("resize: invalid gray16 image size %dx%d", width, height)
	}

	// Pillow's 16-bit resampling path is not reproduced here; use the high byte.
	pixels := make([]uint8, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixels[y*width+x] = uint8(src.Gray16At(bounds.Min.X+x, bounds.Min.Y+y).Y >> 8)
		}
	}
	return resampleChannels(pixels, width, height, 1, outWidth, outHeight), nil
}

func resizePalettedGray(src *image.Paletted, outWidth, outHeight int) ([]uint8, error) {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("resize: invalid paletted image size %dx%d", width, height)
	}
	palette := src.Palette
	if len(palette) == 0 {
		return nil, fmt.Errorf("resize: paletted image has no palette")
	}

	indices := make([]uint8, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			indices[y*width+x] = src.ColorIndexAt(bounds.Min.X+x, bounds.Min.Y+y)
		}
	}

	indices = nearestResizeIndices(indices, width, height, outWidth, outHeight)
	gray := make([]uint8, outWidth*outHeight)
	for i, index := range indices {
		if int(index) >= len(palette) {
			continue
		}
		gray[i] = nrgbaGray(nrgbaFromColor(palette[index]))
	}
	return gray, nil
}

func nearestResizeIndices(src []uint8, width, height, outWidth, outHeight int) []uint8 {
	out := make([]uint8, outWidth*outHeight)
	for y := 0; y < outHeight; y++ {
		sy := int((float64(y) + 0.5) * float64(height) / float64(outHeight))
		if sy >= height {
			sy = height - 1
		}
		for x := 0; x < outWidth; x++ {
			sx := int((float64(x) + 0.5) * float64(width) / float64(outWidth))
			if sx >= width {
				sx = width - 1
			}
			out[y*outWidth+x] = src[sy*width+sx]
		}
	}
	return out
}

func resampleChannels(src []uint8, width, height, channels, outWidth, outHeight int) []uint8 {
	if width == outWidth && height == outHeight {
		return append([]uint8(nil), src...)
	}

	needHorizontal := width != outWidth
	needVertical := height != outHeight
	horizontalFirst := !((height-outHeight) > 0 && (height-outHeight) > (width-outWidth)*2)

	if horizontalFirst {
		if needHorizontal {
			src = horizontalPass(src, width, height, channels, outWidth)
			width = outWidth
		}
		if needVertical {
			src = verticalPass(src, width, height, channels, outHeight)
			height = outHeight
		}
	} else {
		if needVertical {
			src = verticalPass(src, width, height, channels, outHeight)
			height = outHeight
		}
		if needHorizontal {
			src = horizontalPass(src, width, height, channels, outWidth)
			width = outWidth
		}
	}
	return src
}

func horizontalPass(src []uint8, width, height, channels, outWidth int) []uint8 {
	coeffs := precomputeCoeffs(width, outWidth)
	out := make([]uint8, outWidth*height*channels)
	for y := 0; y < height; y++ {
		for x := 0; x < outWidth; x++ {
			coeff := coeffs[x]
			for channel := 0; channel < channels; channel++ {
				sum := int64(1 << (pillowPrecisionBits - 1))
				for i, weight := range coeff.weights {
					source := (y*width + coeff.start + i) * channels
					sum += int64(weight) * int64(src[source+channel])
				}
				out[(y*outWidth+x)*channels+channel] = clip8(sum)
			}
		}
	}
	return out
}

func verticalPass(src []uint8, width, height, channels, outHeight int) []uint8 {
	coeffs := precomputeCoeffs(height, outHeight)
	out := make([]uint8, width*outHeight*channels)
	for y := 0; y < outHeight; y++ {
		coeff := coeffs[y]
		for x := 0; x < width; x++ {
			for channel := 0; channel < channels; channel++ {
				sum := int64(1 << (pillowPrecisionBits - 1))
				for i, weight := range coeff.weights {
					source := ((coeff.start+i)*width + x) * channels
					sum += int64(weight) * int64(src[source+channel])
				}
				out[(y*width+x)*channels+channel] = clip8(sum)
			}
		}
	}
	return out
}

func precomputeCoeffs(inSize, outSize int) []filterCoeff {
	scale := float64(inSize) / float64(outSize)
	filterScale := math.Max(scale, 1)
	support := 3 * filterScale
	factor := float64(int64(1) << pillowPrecisionBits)

	coeffs := make([]filterCoeff, outSize)
	for out := 0; out < outSize; out++ {
		center := (float64(out) + 0.5) * scale
		start := int(center - support + 0.5)
		if start < 0 {
			start = 0
		}
		end := int(center + support + 0.5)
		if end > inSize {
			end = inSize
		}
		if end <= start {
			end = start + 1
		}

		weights := make([]float64, end-start)
		var total float64
		for i := range weights {
			x := (float64(start+i) - center + 0.5) / filterScale
			weight := lanczosKernel(x)
			weights[i] = weight
			total += weight
		}
		if total != 0 {
			for i := range weights {
				weights[i] /= total
			}
		}

		quantized := make([]int32, len(weights))
		for i, weight := range weights {
			if weight < 0 {
				quantized[i] = int32(-0.5 + weight*factor)
			} else {
				quantized[i] = int32(0.5 + weight*factor)
			}
		}
		coeffs[out] = filterCoeff{start: start, weights: quantized}
	}
	return coeffs
}

func lanczosKernel(x float64) float64 {
	if x < -3 || x >= 3 {
		return 0
	}
	return sinc(x) * sinc(x/3)
}

func sinc(x float64) float64 {
	if x == 0 {
		return 1
	}
	x *= math.Pi
	return math.Sin(x) / x
}

func clip8(sum int64) uint8 {
	value := int(sum >> pillowPrecisionBits)
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return uint8(value)
}

func premultiplyRGBA(src []uint8) []uint8 {
	out := make([]uint8, len(src))
	for i := 0; i < len(src); i += 4 {
		alpha := src[i+3]
		out[i] = mulDiv255(src[i], alpha)
		out[i+1] = mulDiv255(src[i+1], alpha)
		out[i+2] = mulDiv255(src[i+2], alpha)
		out[i+3] = alpha
	}
	return out
}

func mulDiv255(a, b uint8) uint8 {
	value := int(a)*int(b) + 128
	return uint8(((value >> 8) + value) >> 8)
}

func unpremultiply(value, alpha uint8) uint8 {
	if alpha == 0 || alpha == 255 {
		return value
	}
	result := 255 * int(value) / int(alpha)
	if result > 255 {
		return 255
	}
	return uint8(result)
}
