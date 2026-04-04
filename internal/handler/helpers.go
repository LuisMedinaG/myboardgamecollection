package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func parseAidID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("aid_id"), 10, 64)
}

func allowedImageExtension(contentType string) (string, bool) {
	switch contentType {
	case "image/png":
		return ".png", true
	case "image/jpeg":
		return ".jpg", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

var driveFileIDRegex = regexp.MustCompile(`/d/([a-zA-Z0-9_-]+)`)

func driveEmbedURL(url string) string {
	if url == "" {
		return ""
	}
	matches := driveFileIDRegex.FindStringSubmatch(url)
	if len(matches) < 2 {
		return ""
	}
	return fmt.Sprintf("https://drive.google.com/file/d/%s/preview", matches[1])
}

func validateRulesURL(raw string) error {
	if raw == "" {
		return nil
	}

	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("invalid rules URL")
	}

	if !strings.EqualFold(u.Scheme, "https") {
		return errors.New("rules URL must use https")
	}

	host := strings.ToLower(u.Host)
	if host != "drive.google.com" && host != "docs.google.com" {
		return errors.New("rules URL must point to Google Drive")
	}

	return nil
}
