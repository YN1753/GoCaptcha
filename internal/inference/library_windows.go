//go:build windows

package inference

import (
	"fmt"

	"golang.org/x/sys/windows"
)

type library struct {
	handle windows.Handle
}

func openLibrary(path string) (*library, error) {
	handle, err := windows.LoadLibraryEx(
		path,
		0,
		windows.LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR|windows.LOAD_LIBRARY_SEARCH_SYSTEM32,
	)
	if err != nil {
		return nil, fmt.Errorf("load ONNX Runtime %q: %w", path, err)
	}
	return &library{handle: handle}, nil
}

func (l *library) symbol(name string) (uintptr, error) {
	return windows.GetProcAddress(l.handle, name)
}

func (l *library) close() error {
	return windows.FreeLibrary(l.handle)
}
