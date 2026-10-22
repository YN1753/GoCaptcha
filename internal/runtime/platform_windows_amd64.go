//go:build windows && amd64

package runtime

import _ "embed"

// Official ONNX Runtime 1.23.2 win-x64 CPU runtime.
//
//go:embed assets/windows_amd64/onnxruntime.dll
var onnxRuntimeLibrary []byte

func currentAsset() (asset, error) {
	return asset{
		Key:      "1.23.2-windows-amd64",
		Filename: "onnxruntime.dll",
		SHA256:   "dec964ab1ee36cc9b0ae247d13b376627992fc57dec0454354017ab8fd84f1ea",
		Data:     onnxRuntimeLibrary,
	}, nil
}
