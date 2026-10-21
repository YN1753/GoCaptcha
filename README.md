# GoCaptcha

[![CI](https://github.com/YN1753/GoCaptcha/actions/workflows/ci.yml/badge.svg)](https://github.com/YN1753/GoCaptcha/actions/workflows/ci.yml)

GoCaptcha 是 [ddddocr](https://github.com/sml2h3/ddddocr) 默认验证码识别能力的 Go 移植版。模型、字符集和 ONNX Runtime 运行时都嵌入可执行文件，首次运行自动释放原生库到用户缓存目录。

- 不使用 Python，不调用远程 OCR API；
- 不依赖用户安装 Go、OpenCV 或 ONNX Runtime；
- 模型通过 ONNX Runtime C API 从内存加载；
- 识别完全在本地完成，默认不需要网络。

## 项目状态

这是一个 **alpha 版本**，Windows 尚未交付。

| 平台 | 状态 | 说明 |
|---|---|---|
| macOS ARM64 | 已验证并发布 | 197/201 与 Python 基准一致，147/201 真实标签准确率 |
| macOS x64 | 已验证并发布 | Apple Silicon + Rosetta 验证；195/201 一致，145/201 准确率；物理 Intel 未测 |
| Linux x64 | 已验证并发布 | Ubuntu 24.04 原生验证；196/201 一致，146/201 准确率；glibc 2.27+ |
| Linux ARM64 | 未交付 | 尚未完成运行时资产和真实验收 |
| Windows x64 | 未交付 | 运行时加载器、VC++ 运行库方案尚未完成 |
| Windows ARM64 | 未交付 | 暂不声明支持 |

验收测试使用 201 张带标签的验证码，数据集来源和已知差异见 [testdata/README.md](testdata/README.md)。

## 快速使用

从 [Releases](https://github.com/YN1753/GoCaptcha/releases) 下载对应平台压缩包，解压后直接运行：

```bash
# macOS 首次运行前移除 quarantine
xattr -d com.apple.quarantine ./ddddocr

./ddddocr image.png
cat image.png | ./ddddocr -
./ddddocr --version
```

Linux 如果解压后没有执行权限：

```bash
chmod +x ./ddddocr
```

首次运行会把内嵌的 ONNX Runtime 释放到：

- macOS：`~/Library/Caches/gocaptcha/`
- Linux：`$XDG_CACHE_HOME/gocaptcha/` 或 `~/.cache/gocaptcha/`

后续运行会校验缓存文件 SHA-256 并复用。设置了 `GOCAPTCHA_CACHE_DIR` 时，使用该目录作为缓存根目录。

## 支持范围

当前支持 PNG、JPEG、GIF（Go 标准库解码器）。WebP、BMP、TIFF 尚未接入。

识别逻辑复刻 ddddocr 1.5.6：

- 默认模型：`common_old.onnx`，动态量化；
- 字符集：`CHARSET_OLD`，8210 项，索引 0 为 CTC blank；
- 预处理：等比缩放到高度 64，Pillow 兼容 Lanczos，灰度，`[-1,1]` 归一化；
- 解码：逐时间步 argmax，CTC 去重并跳过索引 0。

Beta 模型 `common.onnx` 尚未打包。当前版本不包含目标检测、滑块匹配、自定义模型、GPU 和颜色过滤。

## 自行构建

要求 Go 1.25+。运行时资产已经保存在仓库中，因此构建阶段不需要 Python、OpenCV 或 ORT：

```bash
CGO_ENABLED=0 go build -trimpath -o ddddocr ./cmd/ddddocr
```

交叉构建示例：

```bash
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -o ddddocr-darwin-arm64 ./cmd/ddddocr
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -o ddddocr-darwin-amd64 ./cmd/ddddocr
CGO_ENABLED=0 GOOS=linux  GOARCH=amd64 go build -trimpath -o ddddocr-linux-amd64  ./cmd/ddddocr
```

## 测试

在 macOS 或 Linux x64 上：

```bash
go test ./...
go vet ./...
```

测试会自动使用嵌入的运行时，不需要设置 `ORT_LIBRARY`。测试内容包括：

- Pillow 预处理黄金张量逐像素对照；
- 真实模型 201 张验证码对照；
- CTC、字符集和特殊空格测试；
- 缓存释放、损坏修复、并发启动和错误路径。

## 项目结构

```text
cmd/ddddocr/              CLI
internal/preprocess/     图片解码、Lanczos、灰度和张量生成
internal/postprocess/    字符集与 CTC 解码
internal/inference/      purego + ONNX Runtime C API
internal/runtime/        运行时资产、缓存释放和平台适配
assets/                  模型与字符集
testdata/                可复现的验证码语料与基线
LICENSES/                第三方许可证
```

## 已知限制

- Go 标准库 JPEG 解码与 Pillow/libjpeg 存在像素差异，因此少数 JPEG 的结果与 Python 基准不同；真实标签准确率已在各平台测试中记录。
- macOS 二进制没有 Developer ID 签名和公证，需要手动移除 quarantine；企业 MDM 仍可能拦截。
- macOS 最低版本为 13.4+。
- Linux 最低 glibc 2.27；Alpine/musl 未支持。
- 16 位灰度当前取高 8 位；P 模式 PNG 使用最近邻，与 Pillow 的行为一致。
- Windows、Linux ARM64、Beta 模型尚不在发布范围。
- 这是“单文件 + 首次释放原生运行时”，不是严格静态零动态依赖；操作系统基础库仍由系统提供。

## 许可与来源

本项目使用 MIT License。模型和字符集来自 ddddocr 1.5.6；ONNX Runtime 1.23.2 为 MIT，附带完整第三方声明。purego 为 Apache-2.0。详见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) 和 [LICENSES/](LICENSES/)。

ddddocr 训练数据的具体授权链没有在原始仓库中说明；商用再分发前应自行完成法律审查。本项目不是 ddddocr 官方项目。
