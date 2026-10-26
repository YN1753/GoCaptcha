package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/YN1753/GoCaptcha/captcha"
)

func runServer(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("ddddocr serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	addr := flags.String("addr", "127.0.0.1:8080", "监听地址")
	maxBytes := flags.Int64("max-bytes", 8<<20, "单张图片最大字节数")
	corsOrigin := flags.String("cors-origin", "", "允许的 CORS Origin；留空不启用")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "用法: ddddocr serve [--addr 127.0.0.1:8080] [--max-bytes 8388608] [--cors-origin ORIGIN]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		flags.Usage()
		return errors.New("serve 不接受位置参数")
	}
	if *maxBytes <= 0 {
		return errors.New("--max-bytes 必须大于 0")
	}

	recognizer, err := captcha.New()
	if err != nil {
		return fmt.Errorf("初始化识别器失败: %w", err)
	}
	defer recognizer.Close()

	var handler http.Handler = newServerMux(recognizer, *maxBytes)
	if *corsOrigin != "" {
		handler = withCORS(handler, *corsOrigin)
	}
	server := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	fmt.Fprintf(stderr, "GoCaptcha API 监听 %s\n", *addr)
	fmt.Fprintf(stderr, "POST /v1/ocr  (image/png 原始字节，或 multipart 字段 image)\n")
	fmt.Fprintf(stderr, "GET  /healthz\n")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func newServerMux(recognizer *captcha.Recognizer, maxBytes int64) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/v1/ocr", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		imageData, status, err := readImageRequest(w, r, maxBytes)
		if err != nil {
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		text, err := recognizer.Recognize(imageData)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"text": text})
	})
	return mux
}

func withCORS(next http.Handler, origin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func readImageRequest(w http.ResponseWriter, r *http.Request, maxBytes int64) ([]byte, int, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(maxBytes); err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("解析 multipart 失败: %w", err)
		}
		if r.MultipartForm != nil {
			defer r.MultipartForm.RemoveAll()
		}
		file, _, err := r.FormFile("image")
		if err != nil {
			return nil, http.StatusBadRequest, errors.New("缺少 multipart 字段 image")
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			return nil, http.StatusBadRequest, fmt.Errorf("读取图片失败: %w", err)
		}
		if len(data) == 0 {
			return nil, http.StatusBadRequest, errors.New("图片为空")
		}
		return data, http.StatusOK, nil
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return nil, http.StatusRequestEntityTooLarge, fmt.Errorf("图片超过 %d 字节限制", maxBytes)
		}
		return nil, http.StatusBadRequest, fmt.Errorf("读取图片失败: %w", err)
	}
	if len(data) == 0 {
		return nil, http.StatusBadRequest, errors.New("图片为空")
	}
	return data, http.StatusOK, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
