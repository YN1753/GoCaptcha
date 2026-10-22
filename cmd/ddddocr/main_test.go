package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunYzm1(t *testing.T) {
	supported := runtime.GOOS == "darwin" ||
		(runtime.GOOS == "linux" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64")) ||
		(runtime.GOOS == "windows" && runtime.GOARCH == "amd64")
	if !supported {
		t.Skip("CLI smoke test not configured for this platform")
	}
	t.Setenv("GOCAPTCHA_CACHE_DIR", t.TempDir())
	imagePath := extractYzm1(t)

	var stdout, stderr bytes.Buffer
	err := run([]string{imagePath}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("run: %v (stderr=%s)", err, stderr.String())
	}
	if got := stdout.String(); got != "3n3d\n" {
		t.Fatalf("stdout = %q, want %q", got, "3n3d\n")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--version"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("run --version: %v", err)
	}
	for _, want := range []string{"gocaptcha ", "ddddocr 1.5.6", "onnxruntime 1.23.2", "model common_old.onnx"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("version output missing %q: %q", want, stdout.String())
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunRequiresImage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(nil, strings.NewReader(""), &stdout, &stderr)
	if err == nil {
		t.Fatal("expected missing image error")
	}
	if !strings.Contains(stderr.String(), "用法:") {
		t.Fatalf("stderr = %q, want usage", stderr.String())
	}
}

func extractYzm1(t *testing.T) string {
	t.Helper()
	archive, err := zip.OpenReader(filepath.Join("..", "..", "testdata", "corpus.zip"))
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
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read yzm1: %v", err)
	}
	path := filepath.Join(t.TempDir(), "yzm1.png")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write yzm1: %v", err)
	}
	return path
}
