package postprocess

import (
	"errors"
	"fmt"
	"strings"
)

// DecodeCTC decodes logits shaped [sequence, batch, classes] with batch size 1.
func DecodeCTC(logits []float32, sequence, batch, classes int, charset *Charset) (string, error) {
	if charset == nil {
		return "", errors.New("charset is nil")
	}
	if sequence < 0 {
		return "", fmt.Errorf("sequence length must not be negative: %d", sequence)
	}
	if batch != 1 {
		return "", fmt.Errorf("batch size must be 1, got %d", batch)
	}
	if classes <= 0 {
		return "", fmt.Errorf("class count must be positive: %d", classes)
	}
	if classes != charset.Len() {
		return "", fmt.Errorf("class count %d does not match charset size %d", classes, charset.Len())
	}
	if expected := sequence * batch * classes; len(logits) != expected {
		return "", fmt.Errorf("logits length %d does not match sequence*batch*classes %d", len(logits), expected)
	}

	var text strings.Builder
	previous := -1
	for t := 0; t < sequence; t++ {
		base := t * batch * classes
		best := 0
		bestScore := logits[base]
		for c := 1; c < classes; c++ {
			if score := logits[base+c]; score > bestScore {
				best = c
				bestScore = score
			}
		}

		if best == previous {
			continue
		}
		previous = best
		if best == 0 {
			continue
		}

		char, ok := charset.Char(best)
		if !ok {
			return "", fmt.Errorf("class index %d is outside charset", best)
		}
		text.WriteString(char)
	}

	return text.String(), nil
}
