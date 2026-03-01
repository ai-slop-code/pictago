// Package optimize resizes images (max 1920px) and re-encodes as JPEG for smaller size.
package optimize

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
	"net/http"

	_ "image/gif"
)

const (
	maxDimension = 1920
	jpegQuality  = 88
)

// Image optimizes the image bytes: resizes so the longest side is maxDimension
// and re-encodes as JPEG for smaller size. If decoding or encoding fails, returns nil, nil, false.
func Image(data []byte, contentType string) (out []byte, newContentType string, ok bool) {
	img, fmtName, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", false
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= maxDimension && h <= maxDimension && fmtName == "jpeg" {
		// already small enough and JPEG — optional: re-encode at lower quality
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
			return nil, "", false
		}
		return buf.Bytes(), "image/jpeg", true
	}
	// resize if needed
	if w > maxDimension || h > maxDimension {
		img = resize(img, w, h, maxDimension)
		bounds = img.Bounds()
		w, h = bounds.Dx(), bounds.Dy()
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, "", false
	}
	return buf.Bytes(), "image/jpeg", true
}

func resize(src image.Image, w, h, max int) image.Image {
	scale := 1.0
	if w > max || h > max {
		if w > h {
			scale = float64(max) / float64(w)
		} else {
			scale = float64(max) / float64(h)
		}
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

// OptimizeIfRequested reads the multipart file, and if contentType is an image and optimize is true,
// returns optimized bytes and new content type. Otherwise returns the original read and false for optimized.
func OptimizeIfRequested(r io.Reader, contentType string, optimize bool) (data []byte, outContentType string, optimized bool) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, "", false
	}
	outContentType = contentType
	if !optimize {
		return data, outContentType, false
	}
	// only optimize known image types
	switch contentType {
	case "image/jpeg", "image/png", "image/gif":
		break
	default:
		return data, outContentType, false
	}
	optimizedData, newCT, ok := Image(data, contentType)
	if !ok {
		return data, outContentType, false
	}
	return optimizedData, newCT, true
}

// SniffContentType returns the content type from the first bytes (e.g. from multipart).
func SniffContentType(data []byte) string {
	return http.DetectContentType(data)
}
