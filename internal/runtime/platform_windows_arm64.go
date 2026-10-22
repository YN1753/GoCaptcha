//go:build windows && arm64

package runtime

import _ "embed"

// Official ONNX Runtime 1.23.2 win-arm64 CPU runtime.
//
//go:embed assets/windows_arm64/onnxruntime.dll
var onnxRuntimeLibrary []byte

func currentAsset() (asset, error) {
	return asset{
		Key:      "1.23.2-windows-arm64",
		Filename: "onnxruntime.dll",
		SHA256:   "99a11bb077f723a81d100fd98099f332e4178bbb7025ac10fd40699a0b82c2f0",
		Data:     onnxRuntimeLibrary,
	}, nil
}
