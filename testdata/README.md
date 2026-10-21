# 测试语料

`corpus.zip` 是 Phase 1 的起步语料，用于验证 Go 预处理、字符集映射、CTC 解码以及与 Python 原版的结果一致性。它不是模型训练集，也不会被打包进最终可执行文件。

## 内容

| 数据集 | 数量 | 样式 | 许可证 | 来源 |
|---|---:|---|---|---|
| project-sloth/captcha-images | 100 | 6 位数字，含前导零 | WTFPL | https://huggingface.co/datasets/project-sloth/captcha-images |
| muddokon/outlined-text-captcha | 100 | 5 位描边字母数字 | MIT（发布者声明） | https://huggingface.co/datasets/muddokon/outlined-text-captcha |
| sml2h3/ddddocr `samples/yzm1.png` | 1 | 4 位混合字母数字 | MIT | https://github.com/sml2h3/ddddocr |

共 201 张图片。`manifest.json` 记录每张图片的来源、许可证、标签、SHA-256，以及 ddddocr 1.5.6/1.6.1、默认/Beta 四种基准输出。

压缩包中的 `LICENSE-project-sloth.txt`、`NOTICE-muddokon.txt`、`NOTICE-ddddocr.txt` 是对应数据来源的许可与出处说明。`muddokon` 数据集卡片说明图片采集自使用 SimpleCaptcha 的站点；若需要更严格的再分发审查，应把它改为测试时下载，而不是随仓库分发。

压缩包 SHA-256：

```text
1244b837a3e73b1e5247999a76d1c64ee8676875d4cc12ded46644021cc54f9b
```

## 基准结果

测试时间：2026-10-21。准确率是整串完全匹配，CER 是字符错误率。

| 配置 | 6 位数字 | 5 位描边 | 总计 201 | CER |
|---|---:|---:|---:|---:|
| **ddddocr 1.5.6 默认** | 75/100 | 71/100 | **147/201（73.1%）** | 7.61% |
| **ddddocr 1.5.6 Beta** | 80/100 | 80/100 | **161/201（80.1%）** | **5.25%** |
| ddddocr 1.6.1 默认 | 69/100 | 63/100 | 133/201（66.2%） | 11.05% |
| ddddocr 1.6.1 Beta | 75/100 | 69/100 | 145/201（72.1%） | 7.43% |

结论：本项目锁定 ddddocr 1.5.6 的预处理。1.6.1 删除 `[-1,1]` 归一化后，在这批语料上默认和 Beta 都明显下降，不能作为行为基准。

1.5.6 配合 ORT 1.23.2 时，默认/Beta 仍分别为 147/201 和 161/201，因此可以继续使用计划中的 ORT 1.23.2。

Go 全链路（Phase 2/3，默认模型）对照结果：

macOS ARM64：

- 与 Python 基准逐字一致：**197/201**；
- Go 对真实标签准确率：**147/201**，与 Python 默认模型相同；
- PNG 样本逐字一致；
- 4 张 JPEG 存在差异，原因是 Go 标准库的 JPEG 解码与 Pillow/libjpeg 的像素值不同。将其中一张图先转成 PNG 再送入 Go，结果恢复为 Python 的正确结果，因此差异在解码器，不在预处理或 ORT。

macOS x64（Rosetta）：

- 预处理黄金张量仍为逐像素一致；
- 与 Python 基准逐字一致：**195/201**；
- Go 对真实标签准确率：**145/201**，低于 ARM64/Python 的 147/201；
- 差异来自 x86_64 ORT 量化内核/运行环境，预处理已排除；物理 Intel Mac 尚未测试。

Linux x64（Ubuntu 24.04，glibc 2.39，原生 x86_64）：

- 与 Python 基准逐字一致：**196/201**；
- Go 对真实标签准确率：**146/201**，Python 为 147/201；
- PNG 样本逐字一致；
- CLI 首次运行自动释放 22.3MB `.so`，首次识别约 0.18s；
- 缓存损坏自动修复、并发启动、`env -i` 最小环境均通过；
- `.so` 要求的最高 glibc 符号版本为 `GLIBC_2.27`。

## 使用

```bash
unzip -l testdata/corpus.zip
unzip -q testdata/corpus.zip -d /tmp/gocaptcha-corpus
```

目录结构：

```text
project/*.jpg       100 张，文件名前缀是标签
outlined/*.jpg      100 张，文件名去掉扩展名是标签
yzm1/yzm1.png       标签固定为 3n3d
manifest.json       逐图标签、来源和基准输出
```

## 局限

- 语料只覆盖数字和部分小写字母，没有中文文字验证码；不能据此宣称中文准确率。
- 图片来自公开数据集或上游示例，不代表最终目标站点的验证码分布。
- 准确率数字只用于选择版本和建立 Go/Python 一致性基线，不是最终产品的识别率承诺。
- P 模式 PNG、RGBA 透明背景、超大图等边界情况不在这批语料中，Phase 1 需要额外构造测试。
