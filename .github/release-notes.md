# GoCaptcha release

这是 GoCaptcha 的 alpha 发布，包含全部目标平台的构建产物，但验收程度不同。

## 支持平台

- macOS 13.4+ ARM64：已验证
- macOS 13.4+ x86_64：Rosetta 验证；物理 Intel 未测试
- Linux x86_64，glibc 2.27+：已验证
- Linux ARM64：已打包，未验证
- Windows x64：已打包，CI 完整测试通过；依赖系统 VC++ 运行库，干净环境验收未完成
- Windows ARM64：已打包，未验证

Beta 模型不在本次发布中。

## 各平台使用

### macOS ARM64 / x64

```bash
tar -xzf gocaptcha-darwin-arm64.tar.gz   # Intel 用 darwin-amd64
xattr -d com.apple.quarantine ./ddddocr
./ddddocr image.png
```

macOS 未签名、未公证；首次运行前执行 `xattr` 命令。

### Linux x64 / ARM64

```bash
tar -xzf gocaptcha-linux-amd64.tar.gz   # ARM64 用 linux-arm64
chmod +x ./ddddocr
./ddddocr image.png
```

要求 glibc 2.27+，不支持 Alpine/musl。

### Windows x64 / ARM64

```powershell
Expand-Archive .\gocaptcha-windows-amd64.zip -DestinationPath .\gocaptcha
cd .\gocaptcha
.\ddddocr.exe image.png
```

Windows 当前需要系统安装 Microsoft Visual C++ 2015-2022 Redistributable：

- x64：https://aka.ms/vs/17/release/vc_redist.x64.exe
- ARM64：https://aka.ms/vs/17/release/vc_redist.arm64.exe

缺少 `MSVCP140.dll` / `VCRUNTIME140.dll` 时安装对应架构运行库。Windows x64 有 CI 完整测试记录，但 runner 自带 VC 运行库，干净环境验收仍未完成；Windows ARM64 未真实运行。

### HTTP API

```bash
./ddddocr serve --addr 127.0.0.1:8080
curl --data-binary @captcha.png -H 'Content-Type: image/png' http://127.0.0.1:8080/v1/ocr
```

返回 `{"text":"..."}`。默认只监听 loopback，无认证；跨域浏览器调用需要 `--cors-origin`。

### 通用

```bash
./ddddocr --version
cat image.png | ./ddddocr -   # macOS/Linux
```

首次运行自动释放 ONNX Runtime 到用户缓存；设置 `GOCAPTCHA_CACHE_DIR` 可覆盖缓存根目录。程序完全本地运行，不访问网络。

## 已知限制

- macOS 未签名、未公证；需要手动移除 quarantine。
- Go JPEG 解码与 Pillow/libjpeg 存在像素差异，少量 JPEG 结果可能与 Python 基准不同；各平台真实标签准确率见 `testdata/README.md`。
- WebP、BMP、TIFF 尚未支持。
- Windows 需要系统 VC++ 2015-2022 运行库，当前未内置。
- Windows x64 已有 CI 完整测试记录，但 runner 自带 VC 运行库，干净环境仍未验收。
- Windows ARM64 和 Linux ARM64 未完成真实运行验收；本 Release 不应被理解为已完成全部平台验收。

模型和第三方许可证见发布包中的 `LICENSES/` 与 `THIRD_PARTY_NOTICES.md`。
