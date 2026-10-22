//go:build linux && arm64

package runtime

import _ "embed"

// Official ONNX Runtime 1.23.2 linux-aarch64 CPU runtime.
//
//go:embed assets/linux_arm64/libonnxruntime.so.1.23.2
var onnxRuntimeLibrary []byte

func currentAsset() (asset, error) {
	return asset{
		Key:      "1.23.2-linux-arm64",
		Filename: "libonnxruntime.so.1.23.2",
		SHA256:   "648ffa64fbe027ae27139109410900cf776a030dec2dbbac51053318cc44c286",
		Data:     onnxRuntimeLibrary,
	}, nil
}
