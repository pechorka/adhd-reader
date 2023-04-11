package contenttype

import "strings"

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
