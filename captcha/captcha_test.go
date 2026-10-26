package captcha

import (
	"archive/zip"
	"io"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRecognizeYzm1(t *testing.T) {
	supported := runtime.GOOS == "darwin" ||
		(runtime.GOOS == "linux" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64")) ||
		(runtime.GOOS == "windows" && runtime.GOARCH == "amd64")
	if !supported {
		t.Skip("no embedded runtime for this platform")
	}
	t.Setenv("GOCAPTCHA_CACHE_DIR", t.TempDir())

	archive, err := zip.OpenReader(filepath.Join("..", "testdata", "corpus.zip"))
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	defer archive.Close()
	var reader io.ReadCloser
	for _, file := range archive.File {
		if file.Name == "yzm1/yzm1.png" {
			reader, err = file.Open()
			if err != nil {
				t.Fatalf("open yzm1: %v", err)
			}
			break
		}
	}
	if reader == nil {
		t.Fatal("yzm1 not found")
	}
	defer reader.Close()
	imageData, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read yzm1: %v", err)
	}

	recognizer, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer recognizer.Close()

	text, err := recognizer.Recognize(imageData)
	if err != nil {
		t.Fatalf("Recognize: %v", err)
	}
	if text != "3n3d" {
		t.Fatalf("Recognize = %q, want 3n3d", text)
	}
}

func TestRecognizeInvalidImage(t *testing.T) {
	t.Setenv("GOCAPTCHA_CACHE_DIR", t.TempDir())
	recognizer, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer recognizer.Close()
	if _, err := recognizer.Recognize([]byte("not an image")); err == nil {
		t.Fatal("expected invalid image to fail")
	}
}
