# GoCaptcha release

这是 GoCaptcha 的 alpha 发布，当前包含已经完成真实验收的 macOS ARM64、macOS x64 和 Linux x64。

## 支持平台

- macOS 13.4+ ARM64
- macOS 13.4+ x86_64（Rosetta 验证；物理 Intel 尚未测试）
- Linux x86_64，glibc 2.27+

Windows、Linux ARM64 和 Beta 模型不在本次发布中。

## 使用

```bash
tar -xzf gocaptcha-darwin-arm64.tar.gz
xattr -d com.apple.quarantine ./ddddocr
./ddddocr image.png
```

Linux：

```bash
tar -xzf gocaptcha-linux-amd64.tar.gz
chmod +x ./ddddocr
./ddddocr image.png
```

首次运行会将内嵌的 ONNX Runtime 释放到用户缓存目录，之后校验 SHA-256 并复用。程序不访问网络，不调用远程 OCR API。

## 已知限制

- macOS 未签名、未公证；需要手动移除 quarantine。
- Go JPEG 解码与 Pillow/libjpeg 存在像素差异，少量 JPEG 结果可能与 Python 基准不同；各平台真实标签准确率见 `testdata/README.md`。
- WebP、BMP、TIFF 尚未支持。
- Windows 和 Linux ARM64 尚未交付；本 Release 不应被理解为已完成全部平台验收。

模型和第三方许可证见发布包中的 `LICENSES/` 与 `THIRD_PARTY_NOTICES.md`。
