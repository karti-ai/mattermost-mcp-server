package file

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const MaxFileSize = 50 * 1024 * 1024

var AllowedMimeTypes = []string{
	"image/jpeg",
	"image/png",
	"image/gif",
	"image/webp",
	"application/pdf",
	"text/plain",
	"text/markdown",
	"application/msword",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation",
}

var DangerousExtensions = []string{
	".exe", ".sh", ".bat", ".cmd", ".com", ".scr", ".pif", ".vbs",
	".js", ".jse", ".wsf", ".wsh", ".ps1", ".ps2", ".msc",
	".dll", ".so", ".dylib", ".bin",
}

func ValidatePath(base, target string) (string, error) {
	cleanTarget := filepath.Clean(target)
	if filepath.IsAbs(cleanTarget) {
		return "", fmt.Errorf("absolute paths not allowed: %s", target)
	}

	fullPath := filepath.Join(base, cleanTarget)
	realBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for base: %w", err)
	}

	realTarget, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for target: %w", err)
	}

	if !filepath.HasPrefix(realTarget, realBase) {
		return "", fmt.Errorf("path traversal detected: %s", target)
	}

	return realTarget, nil
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func IsValidPath(path string) bool {
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return false
	}
	return true
}

func SanitizePath(path string) (string, error) {
	if !IsValidPath(path) {
		return "", fmt.Errorf("invalid path: path traversal detected: %s", path)
	}
	return filepath.Clean(path), nil
}

func ValidateFileSize(size int64) error {
	if size > MaxFileSize {
		return fmt.Errorf("file size %d bytes exceeds maximum allowed size of %d bytes (50MB)", size, MaxFileSize)
	}
	return nil
}

func IsAllowedMimeType(mimeType string) bool {
	baseMimeType := strings.Split(mimeType, ";")[0]
	baseMimeType = strings.TrimSpace(baseMimeType)
	for _, allowed := range AllowedMimeTypes {
		if strings.EqualFold(baseMimeType, allowed) {
			return true
		}
	}
	if strings.HasPrefix(baseMimeType, "application/vnd.openxmlformats-officedocument") {
		return true
	}
	return false
}

func DetectMimeType(data []byte) string {
	return http.DetectContentType(data)
}

func ValidateMimeType(data []byte) (string, error) {
	mimeType := DetectMimeType(data)
	if !IsAllowedMimeType(mimeType) {
		return "", fmt.Errorf("MIME type %q is not in the allowed whitelist", mimeType)
	}
	return mimeType, nil
}

func IsDangerousExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, dangerous := range DangerousExtensions {
		if ext == dangerous {
			return true
		}
	}
	return false
}

func ValidateFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if IsDangerousExtension(filename) {
		return fmt.Errorf("file type %q is not allowed for security reasons", filepath.Ext(filename))
	}
	if strings.Contains(filename, "\x00") {
		return fmt.Errorf("filename contains invalid characters")
	}
	return nil
}

func GetFileInfo(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func CheckDiskSpace(path string, requiredBytes int64) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	testFile := filepath.Join(dir, ".write_test_"+fmt.Sprintf("%d", os.Getpid()))
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("insufficient permissions or disk space at %s: %w", dir, err)
	}
	f.Close()
	os.Remove(testFile)
	return nil
}
