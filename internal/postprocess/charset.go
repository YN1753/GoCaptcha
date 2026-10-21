package postprocess

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Charset maps model class indices to decoded text. Index 0 is the CTC blank.
type Charset struct {
	entries []string
}

func NewCharset(entries []string) (*Charset, error) {
	if len(entries) == 0 {
		return nil, errors.New("charset is empty")
	}
	if entries[0] != "" {
		return nil, errors.New("charset index 0 must be empty")
	}

	copied := append([]string(nil), entries...)
	seen := make(map[string]struct{}, len(copied))
	seen[copied[0]] = struct{}{}
	for i := 1; i < len(copied); i++ {
		if copied[i] == "" {
			return nil, fmt.Errorf("charset entry %d is empty", i)
		}
		if _, ok := seen[copied[i]]; ok {
			return nil, fmt.Errorf("charset entry %d duplicates %q", i, copied[i])
		}
		seen[copied[i]] = struct{}{}
	}

	return &Charset{entries: copied}, nil
}

func ParseCharsetJSON(data []byte) (*Charset, error) {
	var entries []string
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse charset JSON: %w", err)
	}
	return NewCharset(entries)
}

func (c *Charset) Len() int {
	if c == nil {
		return 0
	}
	return len(c.entries)
}

func (c *Charset) Char(index int) (string, bool) {
	if c == nil || index < 0 || index >= len(c.entries) {
		return "", false
	}
	return c.entries[index], true
}
