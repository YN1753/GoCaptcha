package runtime

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func testAsset(data []byte) asset {
	sum := sha256.Sum256(data)
	return asset{
		Key:      "test-1",
		Filename: "libtest.dylib",
		SHA256:   hex.EncodeToString(sum[:]),
		Data:     data,
	}
}

func TestExtractAndRepair(t *testing.T) {
	data := []byte("test runtime payload")
	a := testAsset(data)
	root := t.TempDir()

	path, err := extract(a, root)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read extracted: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("extracted data = %q, want %q", got, data)
	}

	again, err := extract(a, root)
	if err != nil {
		t.Fatalf("second extract: %v", err)
	}
	if again != path {
		t.Fatalf("second path = %q, want %q", again, path)
	}

	if err := os.WriteFile(path, []byte("corrupt"), 0o700); err != nil {
		t.Fatalf("corrupt cache: %v", err)
	}
	repaired, err := extract(a, root)
	if err != nil {
		t.Fatalf("repair extract: %v", err)
	}
	got, err = os.ReadFile(repaired)
	if err != nil {
		t.Fatalf("read repaired: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("repaired data = %q, want %q", got, data)
	}
}

func TestExtractConcurrent(t *testing.T) {
	data := []byte("concurrent runtime payload")
	a := testAsset(data)
	root := t.TempDir()

	const workers = 16
	var wait sync.WaitGroup
	wait.Add(workers)
	errs := make(chan error, workers)
	paths := make(chan string, workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wait.Done()
			path, err := extract(a, root)
			if err != nil {
				errs <- err
				return
			}
			paths <- path
		}()
	}
	wait.Wait()
	close(errs)
	close(paths)
	for err := range errs {
		t.Errorf("concurrent extract: %v", err)
	}
	var first string
	for path := range paths {
		if first == "" {
			first = path
		} else if path != first {
			t.Errorf("paths differ: %q and %q", first, path)
		}
	}
	if first == "" {
		t.Fatal("no path returned")
	}
	matches, err := fileMatches(first, a.SHA256)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !matches {
		t.Fatal("concurrent output does not match expected hash")
	}
}

func TestExtractRejectsBadAssetHash(t *testing.T) {
	data := []byte("payload")
	a := testAsset(data)
	a.SHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := extract(a, t.TempDir()); err == nil {
		t.Fatal("expected bad asset hash to fail")
	}
}

func TestExtractInvalidRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root-file")
	if err := os.WriteFile(root, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("write root file: %v", err)
	}
	a := testAsset([]byte("payload"))
	if _, err := extract(a, root); err == nil {
		t.Fatal("expected invalid cache root to fail")
	}
}

func TestExtractRejectsFilenameWithPath(t *testing.T) {
	a := testAsset([]byte("payload"))
	a.Filename = "nested/libtest.dylib"
	if _, err := extract(a, t.TempDir()); err == nil {
		t.Fatal("expected filename with path separator to fail")
	}
}

func TestCurrentAsset(t *testing.T) {
	a, err := currentAsset()
	if err != nil {
		t.Skipf("no embedded runtime for %s/%s: %v", runtime.GOOS, runtime.GOARCH, err)
	}
	sum := sha256.Sum256(a.Data)
	if hex.EncodeToString(sum[:]) != a.SHA256 {
		t.Fatalf("embedded runtime hash = %s, want %s", hex.EncodeToString(sum[:]), a.SHA256)
	}
	path, err := extract(a, t.TempDir())
	if err != nil {
		t.Fatalf("extract current runtime: %v", err)
	}
	matches, err := fileMatches(path, a.SHA256)
	if err != nil {
		t.Fatalf("verify current runtime: %v", err)
	}
	if !matches {
		t.Fatal("extracted current runtime does not match")
	}
}
