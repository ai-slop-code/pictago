// Package optimize resizes images (max 1920px) and re-encodes as JPEG for smaller size.
package optimize

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"math"
	"net/http"

	_ "image/gif"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

const (
	maxDimension   = 1920
	thumbDimension = 200
	jpegQuality    = 88
	thumbQuality   = 80
)

var errDecode = errors.New("image decode failed")
var errEncode = errors.New("image encode failed")

// Image optimizes the image bytes: resizes so the longest side is maxDimension
// and re-encodes as JPEG. Returns (nil, "", err) on failure.
// If the image is already JPEG and within max dimension, returns it as-is without re-encoding.
func Image(data []byte) (out []byte, newContentType string, err error) {
	img, fmtName, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", errDecode
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if fmtName == "jpeg" && w <= maxDimension && h <= maxDimension {
		return data, "image/jpeg", nil
	}
	if w > maxDimension || h > maxDimension {
		img = resize(img, w, h, maxDimension)
		bounds = img.Bounds()
		w, h = bounds.Dx(), bounds.Dy()
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, "", errEncode
	}
	return buf.Bytes(), "image/jpeg", nil
}

func resize(src image.Image, w, h, max int) image.Image {
	scale := math.Min(float64(max)/float64(w), float64(max)/float64(h))
	if scale >= 1 {
		return src
	}
	newW := int(float64(w) * scale)
	newH := int(float64(h) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			sx := int(float64(x) / scale)
			sy := int(float64(y) / scale)
			if sx >= w {
				sx = w - 1
			}
			if sy >= h {
				sy = h - 1
			}
			dst.Set(x, y, src.At(sx, sy))
		}
	}
	return dst
}

// Thumbnail generates a small JPEG thumbnail (longest side maxPx). Returns nil, err on failure.
func Thumbnail(data []byte, maxPx int) ([]byte, error) {
	if maxPx < 1 {
		maxPx = thumbDimension
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errDecode
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w > maxPx || h > maxPx {
		img = resize(img, w, h, maxPx)
		bounds = img.Bounds()
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: thumbQuality}); err != nil {
		return nil, errEncode
	}
	return buf.Bytes(), nil
}

// SniffContentType returns the content type from the first bytes (e.g. from multipart).
func SniffContentType(data []byte) string {
	return http.DetectContentType(data)
}
