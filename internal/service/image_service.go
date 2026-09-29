package service

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImageService persists images pasted or dropped into the editor. Images are
// stored under an "assets" sub-directory next to the document and named after
// the SHA-1 of their binary content (de-duplicated).
type ImageService struct {
	cfg *ConfigService
}

// NewImageService creates a new ImageService with the given config service.
func NewImageService(cfg *ConfigService) *ImageService {
	return &ImageService{cfg: cfg}
}

// SaveImage decodes a data URL ("data:image/png;base64,....") and writes the
// binary to the configured image directory (config.imageDir, defaulting to
// <userConfigDir>/XinText/images), named after the SHA-1 of its content
// (de-duplicated). It returns the absolute path with forward slashes so the
// markdown reference works across editors and pandoc exports:
// e.g. "D:/XinText/images/abc.png".
func (s *ImageService) SaveImage(dataURL string) (string, error) {
	mime, data, err := parseDataURL(dataURL)
	if err != nil {
		return "", err
	}
	return writeImage(data, mimeExt(mime), s.imageDir())
}

// imageDir resolves the directory new images are stored in: the user-configured
// imageDir, or <userConfigDir>/XinText/images when unset.
func (s *ImageService) imageDir() string {
	if s.cfg != nil {
		if cfg, err := s.cfg.GetConfig(); err == nil && cfg.ImageDir != "" {
			return cfg.ImageDir
		}
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir, _ = os.UserHomeDir()
	}
	return filepath.Join(dir, "XinText", "images")
}

// writeImage stores data as <dir>/<sha1><ext> and returns the absolute path
// with forward slashes for direct embedding in markdown.
func writeImage(data []byte, ext, dir string) (string, error) {
	sum := sha1.Sum(data)
	name := hex.EncodeToString(sum[:]) + ext
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	full := filepath.Join(dir, name)
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return "", err
	}
	return filepath.ToSlash(full), nil
}

// ReadFileBase64 reads a local image file and returns it as a data URL so the
// WebView can display images referenced by relative paths in markdown (the
// webview origin cannot fetch files from disk directly). Missing files return
// an error which the frontend ignores, leaving the original src untouched.
func (s *ImageService) ReadFileBase64(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	mime := "image/png"
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	case ".svg":
		mime = "image/svg+xml"
	case ".bmp":
		mime = "image/bmp"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

// parseDataURL splits a data URL into its MIME type and raw binary payload.
func parseDataURL(dataURL string) (string, []byte, error) {
	const prefix = "data:"
	if !strings.HasPrefix(dataURL, prefix) {
		return "", nil, fmt.Errorf("not a data URL")
	}
	comma := strings.IndexByte(dataURL, ',')
	if comma < 0 {
		return "", nil, fmt.Errorf("malformed data URL")
	}
	header := dataURL[len(prefix):comma]
	payload := dataURL[comma+1:]

	mime := "image/png"
	for _, part := range strings.Split(header, ";") {
		if strings.HasPrefix(part, "image/") {
			mime = part
			break
		}
	}

	var decoded []byte
	var err error
	if strings.Contains(header, "base64") {
		decoded, err = base64.StdEncoding.DecodeString(payload)
	} else {
		decoded = []byte(payload)
	}
	if err != nil {
		return "", nil, fmt.Errorf("decode image data: %w", err)
	}
	return mime, decoded, nil
}

func mimeExt(mime string) string {
	switch mime {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "image/bmp":
		return ".bmp"
	default:
		return ".png"
	}
}
