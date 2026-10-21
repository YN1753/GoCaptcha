//go:build darwin && amd64

package runtime

import _ "embed"

// Official ONNX Runtime 1.23.2 osx-x86_64 CPU runtime.
//
//go:embed assets/darwin_amd64/libonnxruntime.1.23.2.dylib
var onnxRuntimeLibrary []byte

func currentAsset() (asset, error) {
	return asset{
		Key:      "1.23.2-darwin-amd64",
		Filename: "libonnxruntime.1.23.2.dylib",
		SHA256:   "8c9c78de65ea3786f987c0d980e9c1b13a3a5fbc6b3e2965ba05b450e6e4c054",
		Data:     onnxRuntimeLibrary,
	}, nil
}
