package providers

import (
	"crypto/sha1" //nolint:gosec // non-cryptographic id generation
	"encoding/hex"
	"strings"
)

func hashURL(u string) string {
	sum := sha1.Sum([]byte(u))
	return hex.EncodeToString(sum[:])
}

func responseSnippet(body []byte) string {
	const maxLen = 512
	s := strings.TrimSpace(string(body))
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	if s == "" {
		return "<empty>"
	}
	return s
}
