package inference

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gocaptcha/assets"
	"gocaptcha/internal/postprocess"
	"gocaptcha/internal/preprocess"
	gocruntime "gocaptcha/internal/runtime"
)

type corpusManifest struct {
	Items []struct {
		File     string            `json:"file"`
		Label    string            `json:"label"`
		Baseline map[string]string `json:"baseline"`
	} `json:"items"`
}

func TestCorpusAgainstDDDDOCR(t *testing.T) {
	supported := runtime.GOOS == "darwin" ||
		(runtime.GOOS == "linux" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64")) ||
		(runtime.GOOS == "windows" && runtime.GOARCH == "amd64")
	if !supported {
		t.Skip("integration test not configured for this platform")
	}
	t.Setenv("GOCAPTCHA_CACHE_DIR", t.TempDir())
	libraryPath, err := gocruntime.EnsureCurrent()
	if err != nil {
		t.Fatalf("prepare embedded runtime: %v", err)
	}

	charset, err := postprocess.ParseCharsetJSON(assets.CharsetOld)
	if err != nil {
		t.Fatalf("parse charset: %v", err)
	}
	engine, err := New(libraryPath, assets.ModelDefault)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer engine.Close()

	archive, err := zip.OpenReader(filepath.Join("..", "..", "testdata", "corpus.zip"))
	if err != nil {
		t.Fatalf("open corpus: %v", err)
	}
	defer archive.Close()

	files := make(map[string]*zip.File, len(archive.File))
	for _, file := range archive.File {
		files[file.Name] = file
	}
	manifestData, err := readZipFile(files["manifest.json"])
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest corpusManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if len(manifest.Items) != 201 {
		t.Fatalf("manifest has %d items, want 201", len(manifest.Items))
	}

	pythonCorrect := 0
	for _, item := range manifest.Items {
		if item.Baseline["156_default"] == item.Label {
			pythonCorrect++
		}
	}

	matchedParity := 0
	goCorrect := 0
	pngMismatches := 0
	inferenceErrors := 0
	var mismatches []string
	for _, item := range manifest.Items {
		file, ok := files[item.File]
		if !ok {
			t.Fatalf("missing corpus file %s", item.File)
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatalf("open %s: %v", item.File, err)
		}
		img, err := preprocess.Decode(reader)
		reader.Close()
		if err != nil {
			t.Fatalf("decode %s: %v", item.File, err)
		}
		tensor, err := preprocess.Preprocess(img)
		if err != nil {
			t.Fatalf("preprocess %s: %v", item.File, err)
		}
		result, err := engine.Run(tensor.Data, tensor.Height, tensor.Width)
		if err != nil {
			inferenceErrors++
			mismatches = append(mismatches, fmt.Sprintf("%s: inference error: %v", item.File, err))
			continue
		}
		got, err := postprocess.DecodeCTC(result.Data, result.Sequence, result.Batch, result.Classes, charset)
		if err != nil {
			t.Fatalf("decode %s: %v", item.File, err)
		}
		want := item.Baseline["156_default"]
		if got == want {
			matchedParity++
		} else {
			if strings.HasSuffix(item.File, ".png") {
				pngMismatches++
			}
			if len(mismatches) < 10 {
				mismatches = append(mismatches, fmt.Sprintf("%s: baseline %q, got %q, label %q", item.File, want, got, item.Label))
			}
		}
		if got == item.Label {
			goCorrect++
		}
	}

	t.Logf("parity=%d/%d goAccuracy=%d/%d pythonAccuracy=%d/%d pngMismatches=%d",
		matchedParity, len(manifest.Items), goCorrect, len(manifest.Items), pythonCorrect, len(manifest.Items), pngMismatches)
	for _, mismatch := range mismatches {
		t.Logf("difference: %s", mismatch)
	}

	if inferenceErrors != 0 {
		t.Fatalf("%d inference errors\n%s", inferenceErrors, strings.Join(mismatches, "\n"))
	}
	if pngMismatches != 0 {
		t.Fatalf("%d PNG mismatches against the Python baseline\n%s", pngMismatches, strings.Join(mismatches, "\n"))
	}
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if goCorrect < pythonCorrect {
			t.Fatalf("Go accuracy %d/%d is below Python %d/%d\n%s", goCorrect, len(manifest.Items), pythonCorrect, len(manifest.Items), strings.Join(mismatches, "\n"))
		}
		const minParity = 195
		if matchedParity < minParity {
			t.Fatalf("parity %d/%d is below the minimum %d\n%s", matchedParity, len(manifest.Items), minParity, strings.Join(mismatches, "\n"))
		}
	} else {
		// ORT CPU kernels differ between arm64 and amd64; keep a regression floor.
		if goCorrect < pythonCorrect-5 {
			t.Fatalf("Go accuracy %d/%d is too far below Python %d/%d\n%s", goCorrect, len(manifest.Items), pythonCorrect, len(manifest.Items), strings.Join(mismatches, "\n"))
		}
		const minParity = 190
		if matchedParity < minParity {
			t.Fatalf("parity %d/%d is below the minimum %d\n%s", matchedParity, len(manifest.Items), minParity, strings.Join(mismatches, "\n"))
		}
	}
}

func readZipFile(file *zip.File) ([]byte, error) {
	if file == nil {
		return nil, fmt.Errorf("file not found")
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}
