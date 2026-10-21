package assets

import _ "embed"

//go:embed model/common_old.onnx
var ModelDefault []byte

//go:embed charset/old.json
var CharsetOld []byte

//go:embed charset/beta.json
var CharsetBeta []byte
