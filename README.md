# GoCaptcha

[![CI](https://github.com/YN1753/GoCaptcha/actions/workflows/ci.yml/badge.svg)](https://github.com/YN1753/GoCaptcha/actions/workflows/ci.yml)

GoCaptcha 是 [ddddocr](https://github.com/sml2h3/ddddocr) 默认验证码识别能力的 Go 移植版。模型、字符集和 ONNX Runtime 运行时都嵌入可执行文件，首次运行自动释放原生库到用户缓存目录。

- 不使用 Python，不调用远程 OCR API；
- 不依赖用户安装 Go、OpenCV 或 ONNX Runtime；
- 模型通过 ONNX Runtime C API 从内存加载；
- 识别完全在本地完成，默认不需要网络。

## 项目状态

这是一个 **alpha 版本**。已打包所有目标平台，但 Windows 和 Linux ARM64 **尚未完成独立干净环境验收**，不能视为正式支持。

| 平台 | 状态 | 说明 |
|---|---|---|
| macOS ARM64 | 已验证 | 197/201 与 Python 基准一致，147/201 真实标签准确率 |
| macOS x64 | 已验证（Rosetta） | 195/201 一致，145/201 准确率；物理 Intel 未测 |
| Linux x64 | 已验证 | Ubuntu 24.04 原生验证；196/201 一致，146/201 准确率；glibc 2.27+ |
| Linux ARM64 | 已打包，未验证 | 交叉构建，尚未在 ARM64 Linux 上真实运行 |
| Windows x64 | 已打包，CI 测试通过 | 依赖系统 VC++ 运行库；尚未完成独立干净 Windows/VC 环境验收 |
| Windows ARM64 | 已打包，未验证 | 仅交叉构建，尚未真实运行 |

验收测试使用 201 张带标签的验证码，数据集来源和已知差异见 [testdata/README.md](testdata/README.md)。

Windows 构建当前**没有内置 VC++ 运行库**，运行前需要目标机器已安装 Microsoft Visual C++ 2015-2022 Redistributable。Windows x64 已有 GitHub Actions 完整测试记录，但测试 runner 自带 VC 运行库，因此这还不等于干净机器验收，也不是最终的“只下载 Release 即可运行”形态；后续需要用 app-local 运行库或静态 CRT 补齐。

## 下载与使用

从 [Releases](https://github.com/YN1753/GoCaptcha/releases) 下载对应平台文件。每个平台一个压缩包，包内只有一个可执行文件 `ddddocr`（Windows 为 `ddddocr.exe`）以及 README、许可证和第三方声明。

| 平台 | 下载文件 | 状态 |
|---|---|---|
| macOS Apple Silicon | `gocaptcha-darwin-arm64.tar.gz` | 已验证 |
| macOS Intel / Rosetta | `gocaptcha-darwin-amd64.tar.gz` | Rosetta 已验证；物理 Intel 未测 |
| Linux x86_64 | `gocaptcha-linux-amd64.tar.gz` | 已验证 |
| Linux ARM64 | `gocaptcha-linux-arm64.tar.gz` | 已打包，未验证 |
| Windows x64 | `gocaptcha-windows-amd64.zip` | CI 测试通过；干净环境未验收 |
| Windows ARM64 | `gocaptcha-windows-arm64.zip` | 已打包，未验证 |

### macOS（ARM64 / x64）

```bash
tar -xzf gocaptcha-darwin-arm64.tar.gz   # Intel 用 darwin-amd64
cd ./
xattr -d com.apple.quarantine ./ddddocr  # 首次运行前执行一次
./ddddocr /path/to/captcha.png
./ddddocr --version
```

macOS 包没有 Developer ID 签名和公证。如果系统仍拦截，可在“系统设置 → 隐私与安全性”中允许，或继续使用 `xattr` 命令。企业 MDM 环境可能不允许加载未签名程序。

### Linux（x64 / ARM64）

```bash
tar -xzf gocaptcha-linux-amd64.tar.gz   # ARM64 用 linux-arm64
chmod +x ./ddddocr
./ddddocr /path/to/captcha.png
./ddddocr --version
```

要求 glibc 2.27+，Alpine/musl 不支持。首次运行会把内嵌的 ONNX Runtime 释放到 `$XDG_CACHE_HOME/gocaptcha/` 或 `~/.cache/gocaptcha/`。

### Windows（x64 / ARM64）

PowerShell：

```powershell
Expand-Archive .\gocaptcha-windows-amd64.zip -DestinationPath .\gocaptcha
cd .\gocaptcha
.\ddddocr.exe .\captcha.png
.\ddddocr.exe --version
```

Windows ARM64 使用 `gocaptcha-windows-arm64.zip` 和 `ddddocr.exe`；当前未在 ARM64 Windows 上真实运行。

**Windows 当前需要系统已安装 Microsoft Visual C++ 2015-2022 Redistributable**，发布包没有内置 VC 运行库：

- x64：https://aka.ms/vs/17/release/vc_redist.x64.exe
- ARM64：https://aka.ms/vs/17/release/vc_redist.arm64.exe

如果启动时报缺少 `MSVCP140.dll`、`VCRUNTIME140.dll` 等，安装对应架构的 VC++ Redistributable 后重试。x64 已有 CI 测试，但测试 runner 自带 VC 运行库，不能替代干净 Windows 环境验收。未签名程序可能触发 SmartScreen，选择“更多信息 → 仍要运行”。Windows 建议直接传图片路径；PowerShell 管道传二进制文件不可靠。

### 通用用法

```bash
./ddddocr image.png          # 文件路径
cat image.png | ./ddddocr -  # 标准输入（macOS/Linux）
./ddddocr --version          # 版本和模型信息
```

识别结果输出到标准输出，每张图一行。首次运行会释放 ONNX Runtime，后续运行校验 SHA-256 并复用缓存。设置 `GOCAPTCHA_CACHE_DIR` 可以覆盖缓存根目录。程序完全本地运行，不上传图片、不调用远程 OCR API。

下载后可校验压缩包：

```bash
# macOS / Linux
shasum -a 256 -c SHA256SUMS
```

```powershell
# Windows：手动比对 SHA256SUMS 与 Get-FileHash 的结果
Get-FileHash .\gocaptcha-windows-amd64.zip -Algorithm SHA256
```

## 作为 Go 库使用

模块路径已迁移为 `github.com/YN1753/GoCaptcha`。应用后端可以直接在进程内调用，不需要启动 CLI 子进程：

```bash
go get github.com/YN1753/GoCaptcha@v0.1.0-alpha.7
```

```go
package main

import (
	"fmt"
	"os"

	"github.com/YN1753/GoCaptcha/captcha"
)

func main() {
	recognizer, err := captcha.New()
	if err != nil {
		panic(err)
	}
	defer recognizer.Close()

	imageData, err := os.ReadFile("captcha.png")
	if err != nil {
		panic(err)
	}
	text, err := recognizer.Recognize(imageData)
	if err != nil {
		panic(err)
	}
	fmt.Println(text)
}
```

`New` 只加载一次模型；同一个 `Recognizer` 可以复用。`Recognize` 不修改共享识别状态，可以并发调用；`Close` 不要与 `Recognize` 并发执行。首次调用会把内嵌的 ONNX Runtime 释放到用户缓存目录，设置 `GOCAPTCHA_CACHE_DIR` 可以改变缓存根目录。

## 目标站点验证码的适配

“通用”是指模型不绑定某个网站，不是对任意验证码都保证准确率。自带的 201 张样本上，默认模型真实标签准确率是 147/201；目标站点的字体、噪声和交互方式不同，准确率需要单独测。

建议不要一开始就阻塞在样本收集上：

1. 先用通用模型跑打卡流程；
2. 每次获取验证码时把图片字节保存到本地，例如 `data/captcha/日期/任务-时间.png`；
3. 同时记录识别文本、提交结果和失败原因；
4. 提交成功说明识别正确；提交失败时把图片和识别结果留下来人工纠正；
5. 积累 20–50 张后跑一遍批量评估，再决定是否切换模型或训练自定义模型。

也可以用开发期的“取一张验证码”页面：后端获取图片并展示，人工输入正确文本后保存为 `图片 + 标签`。不要保存账号密码、Cookie 或完整请求头到样本目录；这些样本只留在本地，并注意目标站点的使用条款和访问频率限制。

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
