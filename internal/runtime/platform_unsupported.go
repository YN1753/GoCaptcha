//go:build !darwin && !(linux && amd64)

package runtime

import "errors"

func currentAsset() (asset, error) {
	return asset{}, errors.New("当前平台暂未内置 ONNX Runtime，目前支持 macOS ARM64")
}
