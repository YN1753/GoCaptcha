// Package captcha exposes the in-process ddddocr-compatible recognizer.
package captcha

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/YN1753/GoCaptcha/assets"
	"github.com/YN1753/GoCaptcha/internal/inference"
	"github.com/YN1753/GoCaptcha/internal/postprocess"
	"github.com/YN1753/GoCaptcha/internal/preprocess"
	gocruntime "github.com/YN1753/GoCaptcha/internal/runtime"
)

// Recognizer is safe for concurrent Recognize calls. Close must not run
// concurrently with Recognize.
type Recognizer struct {
	engine    *inference.Engine
	charset   *postprocess.Charset
	closeOnce sync.Once
	closeErr  error
}

// New creates a recognizer and loads the embedded default model.
func New() (*Recognizer, error) {
	charset, err := postprocess.ParseCharsetJSON(assets.CharsetOld)
	if err != nil {
		return nil, fmt.Errorf("load charset: %w", err)
	}
	libraryPath, err := gocruntime.EnsureCurrent()
	if err != nil {
		return nil, fmt.Errorf("prepare runtime: %w", err)
	}
	engine, err := inference.New(libraryPath, assets.ModelDefault)
	if err != nil {
		return nil, fmt.Errorf("create engine: %w", err)
	}
	return &Recognizer{engine: engine, charset: charset}, nil
}

// Recognize decodes an image and returns the recognized text. The first call
// may extract the embedded ONNX Runtime into the user cache.
func (r *Recognizer) Recognize(imageData []byte) (string, error) {
	if r == nil || r.engine == nil {
		return "", errors.New("captcha: recognizer is closed")
	}
	if len(imageData) == 0 {
		return "", errors.New("captcha: empty image")
	}
	img, err := preprocess.Decode(bytes.NewReader(imageData))
	if err != nil {
		return "", fmt.Errorf("captcha: %w", err)
	}
	tensor, err := preprocess.Preprocess(img)
	if err != nil {
		return "", fmt.Errorf("captcha: %w", err)
	}
	result, err := r.engine.Run(tensor.Data, tensor.Height, tensor.Width)
	if err != nil {
		return "", fmt.Errorf("captcha: inference: %w", err)
	}
	text, err := postprocess.DecodeCTC(result.Data, result.Sequence, result.Batch, result.Classes, r.charset)
	if err != nil {
		return "", fmt.Errorf("captcha: decode text: %w", err)
	}
	return text, nil
}

// Close releases the ONNX Runtime session and library.
func (r *Recognizer) Close() error {
	if r == nil {
		return nil
	}
	r.closeOnce.Do(func() {
		if r.engine != nil {
			r.closeErr = r.engine.Close()
			r.engine = nil
		}
	})
	return r.closeErr
}
