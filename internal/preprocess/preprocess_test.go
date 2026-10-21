package preprocess

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPillowGray(t *testing.T) {
	tests := []struct {
		r, g, b uint8
		want    uint8
	}{
		{r: 255, want: 76},
		{g: 255, want: 150},
		{b: 255, want: 29},
		{r: 255, g: 255, b: 255, want: 255},
		{r: 1, g: 2, b: 3, want: 2},
		{r: 10, g: 20, b: 30, want: 18},
		{r: 123, g: 45, b: 67, want: 71},
		{r: 128, g: 128, want: 113},
	}
	for _, test := range tests {
		if got := pillowGray(test.r, test.g, test.b); got != test.want {
			t.Errorf("pillowGray(%d, %d, %d) = %d, want %d", test.r, test.g, test.b, got, test.want)
		}
	}
}

func TestPreprocessYzm1Golden(t *testing.T) {
	archive, err := zip.OpenReader(filepath.Join("..", "..", "testdata", "corpus.zip"))
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	defer archive.Close()

	var imageData []byte
	for _, file := range archive.File {
		if file.Name != "yzm1/yzm1.png" {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatalf("open yzm1: %v", err)
		}
		imageData, err = io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatalf("read yzm1: %v", err)
		}
		break
	}
	if imageData == nil {
		t.Fatal("yzm1/yzm1.png not found in corpus")
	}

	img, err := Decode(bytes.NewReader(imageData))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	tensor, err := Preprocess(img)
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	if tensor.Height != 64 || tensor.Width != 189 {
		t.Fatalf("tensor size = %dx%d, want 64x189", tensor.Height, tensor.Width)
	}
	if len(tensor.Data) != 64*189 {
		t.Fatalf("tensor length = %d, want %d", len(tensor.Data), 64*189)
	}

	goldenBytes, err := os.ReadFile(filepath.Join("testdata", "yzm1_golden.f32"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if len(goldenBytes)%4 != 0 {
		t.Fatalf("golden length %d is not float32 aligned", len(goldenBytes))
	}
	golden := make([]float32, len(goldenBytes)/4)
	for i := range golden {
		golden[i] = math.Float32frombits(binary.LittleEndian.Uint32(goldenBytes[i*4:]))
	}
	if len(golden) != len(tensor.Data) {
		t.Fatalf("golden length = %d, tensor length = %d", len(golden), len(tensor.Data))
	}

	var maxDiff float32
	var maxIndex int
	for i := range golden {
		diff := float32(math.Abs(float64(tensor.Data[i] - golden[i])))
		if diff > maxDiff {
			maxDiff = diff
			maxIndex = i
		}
	}
	if maxDiff > 0.002 {
		t.Fatalf("max tensor difference %.8f at %d, want <= 0.002", maxDiff, maxIndex)
	}
	t.Logf("yzm1 max tensor difference: %.8f", maxDiff)
}

func TestPreprocessRGBA(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 128))
	for y := 0; y < 128; y++ {
		img.SetNRGBA(0, y, color.NRGBA{R: 255, A: 255})
		img.SetNRGBA(1, y, color.NRGBA{G: 255, A: 0})
	}

	tensor, err := Preprocess(img)
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	if tensor.Width != 1 {
		t.Fatalf("width = %d, want 1", tensor.Width)
	}
	want := normalizedGray(76)
	if diff := math.Abs(float64(tensor.Data[0] - want)); diff > 1e-6 {
		t.Fatalf("tensor[0] = %.8f, want %.8f", tensor.Data[0], want)
	}
}

func TestPreprocessPaletted(t *testing.T) {
	palette := color.Palette{
		color.RGBA{R: 255, A: 255},
		color.RGBA{G: 255, A: 255},
	}
	img := image.NewPaletted(image.Rect(0, 0, 2, 128), palette)
	for y := 0; y < 128; y++ {
		img.SetColorIndex(0, y, 0)
		img.SetColorIndex(1, y, 1)
	}

	tensor, err := Preprocess(img)
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	if tensor.Width != 1 {
		t.Fatalf("width = %d, want 1", tensor.Width)
	}
	want := normalizedGray(150)
	if diff := math.Abs(float64(tensor.Data[0] - want)); diff > 1e-6 {
		t.Fatalf("tensor[0] = %.8f, want %.8f", tensor.Data[0], want)
	}
}

func TestPreprocessRGB(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 128))
	for y := 0; y < 128; y++ {
		img.SetNRGBA(0, y, color.NRGBA{R: 255, A: 255})
		img.SetNRGBA(1, y, color.NRGBA{G: 255, A: 255})
	}

	tensor, err := Preprocess(img)
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	want := normalizedGray(113)
	if diff := math.Abs(float64(tensor.Data[0] - want)); diff > 1e-6 {
		t.Fatalf("tensor[0] = %.8f, want %.8f", tensor.Data[0], want)
	}
}

func TestPreprocessTargetWidthAndErrors(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 665, 225))
	tensor, err := Preprocess(img)
	if err != nil {
		t.Fatalf("Preprocess: %v", err)
	}
	if tensor.Width != 189 || tensor.Height != 64 {
		t.Fatalf("tensor size = %dx%d, want 189x64", tensor.Width, tensor.Height)
	}

	narrow := image.NewGray(image.Rect(0, 0, 1, 1000))
	if _, err := Preprocess(narrow); err == nil {
		t.Fatal("expected narrow image to fail")
	}
	if _, err := Preprocess(nil); err == nil {
		t.Fatal("expected nil image to fail")
	}
}

func TestCorpusPreprocess(t *testing.T) {
	archive, err := zip.OpenReader(filepath.Join("..", "..", "testdata", "corpus.zip"))
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	defer archive.Close()

	count := 0
	for _, file := range archive.File {
		if !strings.HasSuffix(file.Name, ".jpg") && !strings.HasSuffix(file.Name, ".png") {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatalf("open %s: %v", file.Name, err)
		}
		img, decodeErr := Decode(reader)
		reader.Close()
		if decodeErr != nil {
			t.Fatalf("decode %s: %v", file.Name, decodeErr)
		}
		tensor, err := Preprocess(img)
		if err != nil {
			t.Fatalf("preprocess %s: %v", file.Name, err)
		}
		if tensor.Height != 64 || tensor.Width < 1 || len(tensor.Data) != 64*tensor.Width {
			t.Fatalf("%s: invalid tensor %dx%d len=%d", file.Name, tensor.Height, tensor.Width, len(tensor.Data))
		}
		for _, value := range tensor.Data {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) || value < -1 || value > 1 {
				t.Fatalf("%s: invalid tensor value %v", file.Name, value)
			}
		}
		count++
	}
	if count != 201 {
		t.Fatalf("processed %d images, want 201", count)
	}
}

func normalizedGray(value uint8) float32 {
	normalized := float32(value) / 255
	return (normalized - 0.5) / 0.5
}
