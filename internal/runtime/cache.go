package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type asset struct {
	Key      string
	Filename string
	SHA256   string
	Data     []byte
}

// EnsureCurrent extracts the platform runtime into the user cache and returns
// its absolute path. GOCAPTCHA_CACHE_DIR can override the cache root.
func EnsureCurrent() (string, error) {
	a, err := currentAsset()
	if err != nil {
		return "", err
	}
	root := os.Getenv("GOCAPTCHA_CACHE_DIR")
	if root == "" {
		root, err = os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("无法获取用户缓存目录: %w", err)
		}
	}
	return extract(a, root)
}

func extract(a asset, cacheRoot string) (string, error) {
	if cacheRoot == "" {
		return "", errors.New("运行时缓存目录为空")
	}
	if a.Key == "" || a.Filename == "" || len(a.Data) == 0 {
		return "", errors.New("内置运行时资源为空")
	}
	if filepath.Base(a.Filename) != a.Filename || strings.ContainsAny(a.Filename, `/\\`) {
		return "", errors.New("内置运行时文件名非法")
	}
	expected := strings.ToLower(a.SHA256)
	if len(expected) != 64 {
		return "", errors.New("内置运行时 SHA-256 长度非法")
	}
	if _, err := hex.DecodeString(expected); err != nil {
		return "", fmt.Errorf("内置运行时 SHA-256 非法: %w", err)
	}

	sum := sha256.Sum256(a.Data)
	if hex.EncodeToString(sum[:]) != expected {
		return "", fmt.Errorf("内置运行时数据校验失败: 期望 %s，实际 %s", expected, hex.EncodeToString(sum[:]))
	}

	directory := filepath.Join(cacheRoot, "gocaptcha", a.Key+"-"+expected[:12])
	target := filepath.Join(directory, a.Filename)

	matches, err := fileMatches(target, expected)
	if err != nil {
		return "", fmt.Errorf("无法检查运行时缓存 %q: %w", target, err)
	}
	if matches {
		return target, nil
	}

	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("无法创建运行时缓存目录 %q: %w", directory, err)
	}

	temp, err := os.CreateTemp(directory, "."+a.Filename+".tmp-")
	if err != nil {
		return "", fmt.Errorf("无法创建运行时缓存临时文件: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)

	if err := temp.Chmod(0o700); err != nil {
		temp.Close()
		return "", fmt.Errorf("无法设置运行时缓存权限: %w", err)
	}
	if _, err := temp.Write(a.Data); err != nil {
		temp.Close()
		return "", fmt.Errorf("无法写入运行时缓存: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return "", fmt.Errorf("无法同步运行时缓存: %w", err)
	}
	if err := temp.Close(); err != nil {
		return "", fmt.Errorf("无法关闭运行时缓存: %w", err)
	}

	matches, err = fileMatches(tempName, expected)
	if err != nil {
		return "", fmt.Errorf("无法校验运行时缓存临时文件: %w", err)
	}
	if !matches {
		return "", errors.New("释放后的运行时文件校验失败")
	}

	if err := os.Rename(tempName, target); err != nil {
		// Windows cannot rename over an existing file. If another process
		// completed the same content-addressed write, reuse it; otherwise
		// replace a corrupt target and retry.
		if matches, verifyErr := fileMatches(target, expected); verifyErr == nil && matches {
			return target, nil
		}
		if removeErr := os.Remove(target); removeErr == nil {
			if retryErr := os.Rename(tempName, target); retryErr == nil {
				matches, verifyErr := fileMatches(target, expected)
				if verifyErr != nil {
					return "", fmt.Errorf("无法校验运行时缓存 %q: %w", target, verifyErr)
				}
				if !matches {
					return "", errors.New("释放后的运行时文件校验失败")
				}
				return target, nil
			}
		}
		return "", fmt.Errorf("无法写入运行时缓存 %q: %w", target, err)
	}

	matches, err = fileMatches(target, expected)
	if err != nil {
		return "", fmt.Errorf("无法校验运行时缓存 %q: %w", target, err)
	}
	if !matches {
		return "", errors.New("释放后的运行时文件校验失败")
	}
	return target, nil
}

func fileMatches(path, expected string) (bool, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false, err
	}
	return hex.EncodeToString(hash.Sum(nil)) == expected, nil
}
