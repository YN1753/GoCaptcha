package postprocess

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func mustCharset(t *testing.T, entries []string) *Charset {
	t.Helper()
	charset, err := NewCharset(entries)
	if err != nil {
		t.Fatalf("NewCharset: %v", err)
	}
	return charset
}

func loadCharsetEntries(t *testing.T, name string) []string {
	t.Helper()
	path := filepath.Join("..", "..", "assets", "charset", name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var entries []string
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return entries
}

func oneHotLogits(indices []int, classes int) []float32 {
	logits := make([]float32, len(indices)*classes)
	for t, index := range indices {
		logits[t*classes+index] = 1
	}
	return logits
}

func TestNewCharsetValidation(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
	}{
		{name: "empty", entries: nil},
		{name: "missing blank", entries: []string{"a"}},
		{name: "empty entry after blank", entries: []string{"", "a", ""}},
		{name: "duplicate", entries: []string{"", "a", "a"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewCharset(test.entries); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestParseCharsetJSON(t *testing.T) {
	charset, err := ParseCharsetJSON([]byte(`["","a","b"]`))
	if err != nil {
		t.Fatalf("ParseCharsetJSON: %v", err)
	}
	if charset.Len() != 3 {
		t.Fatalf("Len = %d, want 3", charset.Len())
	}
	if char, ok := charset.Char(1); !ok || char != "a" {
		t.Fatalf("Char(1) = %q, %v", char, ok)
	}
	if _, ok := charset.Char(3); ok {
		t.Fatal("Char(3) should be out of range")
	}
	if _, err := ParseCharsetJSON([]byte(`["a"]`)); err == nil {
		t.Fatal("expected invalid blank marker to fail")
	}
	if _, err := ParseCharsetJSON([]byte(`not json`)); err == nil {
		t.Fatal("expected invalid JSON to fail")
	}
}

func TestDecodeCTC(t *testing.T) {
	entries := []string{"", "a", "b", " ", "\u3000", "''", "中"}
	charset := mustCharset(t, entries)
	classes := len(entries)

	tests := []struct {
		name    string
		indices []int
		want    string
	}{
		{name: "blank only", indices: []int{0, 0, 0}, want: ""},
		{name: "collapse repeats", indices: []int{1, 1, 1}, want: "a"},
		{name: "repeat separated by blank", indices: []int{1, 0, 1}, want: "aa"},
		{name: "special characters", indices: []int{3, 4, 5, 6}, want: " \u3000''中"},
		{name: "mixed sequence", indices: []int{0, 1, 1, 0, 2, 3, 0, 4, 5, 6}, want: "ab \u3000''中"},
		{name: "leading blank and repeat", indices: []int{0, 0, 2, 2, 0, 2}, want: "bb"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := DecodeCTC(oneHotLogits(test.indices, classes), len(test.indices), 1, classes, charset)
			if err != nil {
				t.Fatalf("DecodeCTC: %v", err)
			}
			if got != test.want {
				t.Fatalf("DecodeCTC = %q, want %q", got, test.want)
			}
		})
	}
}

func TestDecodeCTCTieUsesFirstIndex(t *testing.T) {
	charset := mustCharset(t, []string{"", "a", "b"})
	logits := []float32{0, 2, 2}
	got, err := DecodeCTC(logits, 1, 1, 3, charset)
	if err != nil {
		t.Fatalf("DecodeCTC: %v", err)
	}
	if got != "a" {
		t.Fatalf("DecodeCTC = %q, want %q", got, "a")
	}
}

func TestDecodeCTCValidation(t *testing.T) {
	charset := mustCharset(t, []string{"", "a"})
	tests := []struct {
		name     string
		logits   []float32
		sequence int
		batch    int
		classes  int
		charset  *Charset
	}{
		{name: "nil charset", logits: nil, sequence: 0, batch: 1, classes: 2, charset: nil},
		{name: "negative sequence", logits: nil, sequence: -1, batch: 1, classes: 2, charset: charset},
		{name: "batch not one", logits: make([]float32, 4), sequence: 1, batch: 2, classes: 2, charset: charset},
		{name: "zero classes", logits: nil, sequence: 0, batch: 1, classes: 0, charset: charset},
		{name: "class mismatch", logits: make([]float32, 3), sequence: 1, batch: 1, classes: 3, charset: charset},
		{name: "logits length mismatch", logits: make([]float32, 1), sequence: 1, batch: 1, classes: 2, charset: charset},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := DecodeCTC(test.logits, test.sequence, test.batch, test.classes, test.charset); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestDecodeCTCEmptySequence(t *testing.T) {
	charset := mustCharset(t, []string{"", "a"})
	got, err := DecodeCTC(nil, 0, 1, 2, charset)
	if err != nil {
		t.Fatalf("DecodeCTC: %v", err)
	}
	if got != "" {
		t.Fatalf("DecodeCTC = %q, want empty", got)
	}
}

func TestRealCharsetAssets(t *testing.T) {
	oldEntries := loadCharsetEntries(t, "old")
	betaEntries := loadCharsetEntries(t, "beta")

	oldCharset, err := NewCharset(oldEntries)
	if err != nil {
		t.Fatalf("old charset: %v", err)
	}
	betaCharset, err := NewCharset(betaEntries)
	if err != nil {
		t.Fatalf("beta charset: %v", err)
	}
	if oldCharset.Len() != 8210 || betaCharset.Len() != 8210 {
		t.Fatalf("charset sizes = %d, %d, want 8210, 8210", oldCharset.Len(), betaCharset.Len())
	}

	checks := []struct {
		charset *Charset
		index   int
		want    string
	}{
		{charset: oldCharset, index: 0, want: ""},
		{charset: oldCharset, index: 1, want: "掀"},
		{charset: oldCharset, index: 5162, want: " "},
		{charset: oldCharset, index: 7554, want: "\u3000"},
		{charset: betaCharset, index: 0, want: ""},
		{charset: betaCharset, index: 1, want: "笤"},
		{charset: betaCharset, index: 982, want: " "},
		{charset: betaCharset, index: 1173, want: "''"},
		{charset: betaCharset, index: 4088, want: "\u3000"},
	}
	for _, check := range checks {
		got, ok := check.charset.Char(check.index)
		if !ok || got != check.want {
			t.Fatalf("Char(%d) = %q, %v, want %q", check.index, got, ok, check.want)
		}
	}

	indices := []int{0, 0, 0, 0, 7721, 0, 0, 0, 0, 0, 4050, 0, 0, 7721, 0, 0, 0, 0, 0, 0, 5726, 0, 0, 0}
	got, err := DecodeCTC(oneHotLogits(indices, oldCharset.Len()), len(indices), 1, oldCharset.Len(), oldCharset)
	if err != nil {
		t.Fatalf("decode yzm1 indices: %v", err)
	}
	if got != "3n3d" {
		t.Fatalf("decoded yzm1 = %q, want 3n3d", got)
	}

	betaIndex := make(map[string]int, len(betaEntries))
	for i, char := range betaEntries {
		if _, exists := betaIndex[char]; !exists {
			betaIndex[char] = i
		}
	}
	betaIndices := []int{betaIndex["3"], 0, betaIndex["n"], 0, betaIndex["3"], 0, betaIndex["d"]}
	got, err = DecodeCTC(oneHotLogits(betaIndices, betaCharset.Len()), len(betaIndices), 1, betaCharset.Len(), betaCharset)
	if err != nil {
		t.Fatalf("decode beta sequence: %v", err)
	}
	if got != "3n3d" {
		t.Fatalf("decoded beta sequence = %q, want 3n3d", got)
	}
}
