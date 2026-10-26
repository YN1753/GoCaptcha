package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"

	"github.com/YN1753/GoCaptcha/captcha"
)

func TestServerAPI(t *testing.T) {
	supported := runtime.GOOS == "darwin" ||
		(runtime.GOOS == "linux" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64")) ||
		(runtime.GOOS == "windows" && runtime.GOARCH == "amd64")
	if !supported {
		t.Skip("no embedded runtime for this platform")
	}
	t.Setenv("GOCAPTCHA_CACHE_DIR", t.TempDir())
	imageData, err := os.ReadFile(extractYzm1(t))
	if err != nil {
		t.Fatalf("read yzm1: %v", err)
	}
	recognizer, err := captcha.New()
	if err != nil {
		t.Fatalf("captcha.New: %v", err)
	}
	defer recognizer.Close()
	handler := newServerMux(recognizer, 8<<20)

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", health.Code)
	}

	request := httptest.NewRequest(http.MethodPost, "/v1/ocr", bytes.NewReader(imageData))
	request.Header.Set("Content-Type", "image/png")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("ocr status = %d, body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Text != "3n3d" {
		t.Fatalf("text = %q, want 3n3d", payload.Text)
	}

	multipartBody := &bytes.Buffer{}
	writer := multipart.NewWriter(multipartBody)
	part, err := writer.CreateFormFile("image", "captcha.png")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(imageData); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	writer.Close()
	request = httptest.NewRequest(http.MethodPost, "/v1/ocr", multipartBody)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("multipart status = %d, body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/v1/ocr", bytes.NewBufferString("not an image"))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("bad image status = %d, want 400", response.Code)
	}
	if _, err := io.ReadAll(response.Body); err != nil {
		t.Fatal(err)
	}
}
