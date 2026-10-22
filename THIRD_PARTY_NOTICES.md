# Third-party notices

GoCaptcha 本身使用 MIT License。发布包还包含或链接以下第三方组件。

## ddddocr 1.5.6

- 来源：https://github.com/sml2h3/ddddocr
- 许可证：MIT
- 使用内容：`common_old.onnx`、`CHARSET_OLD`、预处理和 CTC 解码行为
- 许可证全文：`LICENSES/ddddocr-MIT.txt`

模型和字符集随 ddddocr 仓库以 MIT 分发。原始仓库没有单独说明训练数据、字体和验证码样本的授权链，因此不能据此声称训练数据权利已经独立核实。商用再分发前应进行法律审查。

## ONNX Runtime 1.23.2

- 来源：https://github.com/microsoft/onnxruntime
- 许可证：MIT
- 许可证全文：`LICENSES/onnxruntime-MIT.txt`
- 完整第三方声明：发布包 `LICENSES/onnxruntime-<platform>-ThirdPartyNotices.txt`

发布包中的 `libonnxruntime.dylib` / `libonnxruntime.so` 来自官方 ONNX Runtime 1.23.2 平台包，经 SHA-256 校验后嵌入。

## purego 0.11.2

- 来源：https://github.com/ebitengine/purego
- 许可证：Apache-2.0
- 许可证全文：`LICENSES/purego-Apache-2.0.txt`

## Go 标准库

- 许可证：BSD-3-Clause
- 许可证全文：`LICENSES/go-BSD-3-Clause.txt`

## 测试语料

`testdata/corpus.zip` 中的样本分别标记为 WTFPL、MIT 或 ddddocr 仓库 MIT。测试语料不包含在最终可执行文件中。具体来源见 `testdata/README.md`。

## Windows

Windows x64/ARM64 当前是**未完成干净环境验收的实验构建**。发布包包含官方 ONNX Runtime Windows DLL，但不包含 Microsoft Visual C++ 运行库；运行时需要目标系统已有 VC++ 2015-2022 Redistributable。后续正式支持必须补 app-local 运行库或静态 CRT，并附带微软许可条款。
