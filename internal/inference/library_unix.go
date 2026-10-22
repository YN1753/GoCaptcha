//go:build darwin || linux

package inference

import (
	"fmt"

	"github.com/ebitengine/purego"
)

type library struct {
	handle uintptr
}

func openLibrary(path string) (*library, error) {
	handle, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_LOCAL)
	if err != nil {
		return nil, fmt.Errorf("load ONNX Runtime %q: %w", path, err)
	}
	return &library{handle: handle}, nil
}

func (l *library) symbol(name string) (uintptr, error) {
	return purego.Dlsym(l.handle, name)
}

func (l *library) close() error {
	return purego.Dlclose(l.handle)
}
