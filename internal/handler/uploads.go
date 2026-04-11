package handler

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type unsupportedPlayerAidTypeError string

func (e unsupportedPlayerAidTypeError) Error() string {
	return string(e)
}

var errUnsupportedPlayerAidType = unsupportedPlayerAidTypeError("unsupported file type; upload PNG, JPEG, GIF, or WebP")

func sanitizePlayerAidLabel(label, originalFilename string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		label = strings.TrimSuffix(originalFilename, filepath.Ext(originalFilename))
	}
	if len(label) > 200 {
		label = label[:200]
	}
	return label
}

func (h *Handler) savePlayerAidFile(file multipart.File) (string, error) {
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", err
	}

	contentType := http.DetectContentType(buffer[:n])
	ext, ok := allowedImageExtension(contentType)
	if !ok {
		return "", errUnsupportedPlayerAidType
	}

	filename, err := randomFilename(ext)
	if err != nil {
		return "", err
	}

	uploadDir := filepath.Join(h.DataDir, "uploads")
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return "", err
	}

	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		return "", err
	}
	defer dst.Close()

	reader := io.MultiReader(bytes.NewReader(buffer[:n]), file)
	if _, err := io.Copy(dst, reader); err != nil {
		return "", err
	}

	return filename, nil
}
