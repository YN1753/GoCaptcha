package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	stdruntime "runtime"

	"gocaptcha/assets"
	"gocaptcha/internal/inference"
	"gocaptcha/internal/postprocess"
	"gocaptcha/internal/preprocess"
	gocruntime "gocaptcha/internal/runtime"
)

var version = "0.1.0-dev"

const (
	ddddocrVersion     = "1.5.6"
	onnxRuntimeVersion = "1.23.2"
	modelSHA256        = "b8f2ad9cbc1f2e3922a6cb9459e30824e7e2467f3fb4fd61420640e34ea0bf68"
)

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("ddddocr", flag.ContinueOnError)
	flags.SetOutput(stderr)
	showVersion := flags.Bool("version", false, "显示版本信息")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "用法: ddddocr [--version] <图片路径>")
		fmt.Fprintln(stderr, "  <图片路径> 为 - 时从标准输入读取")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if *showVersion {
		printVersion(stdout)
		return nil
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return errors.New("必须指定一张图片")
	}
	imagePath := flags.Arg(0)

	libraryPath, err := gocruntime.EnsureCurrent()
	if err != nil {
		return fmt.Errorf("无法准备 ONNX Runtime: %w", err)
	}

	charset, err := postprocess.ParseCharsetJSON(assets.CharsetOld)
	if err != nil {
		return fmt.Errorf("无法加载字符集: %w", err)
	}
	engine, err := inference.New(libraryPath, assets.ModelDefault)
	if err != nil {
		return fmt.Errorf("无法初始化识别引擎: %w", err)
	}
	defer func() {
		if closeErr := engine.Close(); closeErr != nil {
			fmt.Fprintf(stderr, "警告: 释放识别引擎失败: %v\n", closeErr)
		}
	}()

	var reader io.Reader
	if imagePath == "-" {
		reader = stdin
	} else {
		file, err := os.Open(imagePath)
		if err != nil {
			return fmt.Errorf("无法打开图片 %q: %w", imagePath, err)
		}
		defer file.Close()
		reader = file
	}

	img, err := preprocess.Decode(reader)
	if err != nil {
		return fmt.Errorf("无法解析图片: %w", err)
	}
	tensor, err := preprocess.Preprocess(img)
	if err != nil {
		return fmt.Errorf("图片预处理失败: %w", err)
	}
	result, err := engine.Run(tensor.Data, tensor.Height, tensor.Width)
	if err != nil {
		return fmt.Errorf("模型推理失败: %w", err)
	}
	text, err := postprocess.DecodeCTC(result.Data, result.Sequence, result.Batch, result.Classes, charset)
	if err != nil {
		return fmt.Errorf("文字解码失败: %w", err)
	}

	fmt.Fprintln(stdout, text)
	return nil
}

func printVersion(out io.Writer) {
	fmt.Fprintf(out, "gocaptcha %s\n", version)
	fmt.Fprintf(out, "ddddocr %s\n", ddddocrVersion)
	fmt.Fprintf(out, "onnxruntime %s\n", onnxRuntimeVersion)
	fmt.Fprintf(out, "platform %s/%s\n", stdruntime.GOOS, stdruntime.GOARCH)
	fmt.Fprintf(out, "model common_old.onnx %s\n", modelSHA256)
}
