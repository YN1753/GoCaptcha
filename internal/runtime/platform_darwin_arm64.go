//go:build darwin && arm64

package runtime

import _ "embed"

// Official ONNX Runtime 1.23.2 osx-arm64 CPU runtime.
//
//go:embed assets/darwin_arm64/libonnxruntime.1.23.2.dylib
var onnxRuntimeLibrary []byte

func currentAsset() (asset, error) {
	return asset{
		Key:      "1.23.2-darwin-arm64",
		Filename: "libonnxruntime.1.23.2.dylib",
		SHA256:   "d306d2bc768540766c7ed8a1e0ff05d2870c77a934ebeee4a7bafa1b732ef299",
		Data:     onnxRuntimeLibrary,
	}, nil
}
