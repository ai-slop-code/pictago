// Package validate provides image type validation (magic-byte checks for JPEG, PNG, GIF, WebP).
package validate

import "bytes"

// Allowed image magic bytes (JPEG, PNG, GIF, WebP)
var (
	jpegPrefix = []byte{0xFF, 0xD8, 0xFF}
	pngPrefix  = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	gifPrefix  = []byte("GIF87a")
	gifPrefix2 = []byte("GIF89a")
	webpPrefix = []byte("RIFF") // WebP: RIFF....WEBP
	webpFmt    = []byte("WEBP")
)

// IsImage returns true if data has a valid image magic header (JPEG, PNG, GIF, WebP).
func IsImage(data []byte) bool {
	if len(data) < 12 {
		return false
	}
	if bytes.HasPrefix(data, jpegPrefix) {
		return true
	}
	if bytes.HasPrefix(data, pngPrefix) {
		return true
	}
	if bytes.HasPrefix(data, gifPrefix) || bytes.HasPrefix(data, gifPrefix2) {
		return true
	}
	if len(data) >= 12 && bytes.Equal(data[0:4], webpPrefix) && bytes.Equal(data[8:12], webpFmt) {
		return true
	}
	return false
}
