package contenttype

import (
	"net/url"
	"strings"
)

func IsPlainText(contentType string) bool {
	if strings.HasPrefix(contentType, "text/plain") {
		return true
	}
	if strings.HasPrefix(contentType, "application/octet-stream") {
		return true
	}
	return false
}

func IsPDF(contentType string) bool {
	return strings.HasPrefix(contentType, "application/pdf")
}

// IsURL returns true if content contains only URL
func IsURL(content string) bool {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "http://") && !strings.HasPrefix(content, "https://") {
		return false
	}

	u, err := url.Parse(content)
	if err != nil {
		return false
	}

	// check if host is empty because url.Parse("http://") returns nil error
	if u.Hostname() == "" {
		return false
	}

	return true
}
