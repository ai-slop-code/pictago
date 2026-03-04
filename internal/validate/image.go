// Package validate provides image type validation (magic-byte checks for JPEG, PNG, GIF, WebP).
package validate

import "bytes"

// Image magic-byte constants (JPEG, PNG, GIF, WebP).
const (
	// JPEG: FF D8 FF
	jpegMagic = "\xff\xd8\xff"
	// PNG: 89 50 4E 47 0D 0A 1A 0A
	pngMagic = "\x89PNG\r\n\x1a\n"
	// GIF87a / GIF89a
	gifMagic1 = "GIF87a"
	gifMagic2 = "GIF89a"
	// WebP: RIFF....WEBP
	webpMagic = "RIFF"
	webpFmt   = "WEBP"
)

var (
	jpegPrefix = []byte(jpegMagic)
	pngPrefix  = []byte(pngMagic)
	gifPrefix  = []byte(gifMagic1)
	gifPrefix2 = []byte(gifMagic2)
	webpPrefix = []byte(webpMagic)
	webpFmtB   = []byte(webpFmt)
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
	if len(data) >= 12 && bytes.Equal(data[0:4], webpPrefix) && bytes.Equal(data[8:12], webpFmtB) {
		return true
	}
	return false
}
