//go:build linux && amd64

package runtime

import _ "embed"

// Official ONNX Runtime 1.23.2 linux-x64 CPU runtime.
//
//go:embed assets/linux_amd64/libonnxruntime.so.1.23.2
var onnxRuntimeLibrary []byte

func currentAsset() (asset, error) {
	return asset{
		Key:      "1.23.2-linux-amd64",
		Filename: "libonnxruntime.so.1.23.2",
		SHA256:   "13ab8084954fa4a47c777880180b90810d6020f021441395712b48a75b74c68b",
		Data:     onnxRuntimeLibrary,
	}, nil
}
