package contenttype

import (
	"net/url"
	"strings"
)

const (
	PlainText   = "text/plain"
	OctetStream = "application/octet-stream"
	PDF         = "application/pdf"
	EPUB        = "application/epub+zip"
)

func IsURLs(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		if !IsURL(line) {
			return false
		}
	}
	return true
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
