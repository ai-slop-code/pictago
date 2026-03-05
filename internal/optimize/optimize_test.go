package optimize

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"testing"
)

// minJPEG returns a minimal valid JPEG (1x1 pixel).
func minJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// minPNG returns a minimal valid PNG (1x1 pixel).
func minPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestThumbnail(t *testing.T) {
	jpg := minJPEG(t)
	out, err := Thumbnail(jpg, 200)
	if err != nil {
		t.Fatalf("Thumbnail: %v", err)
	}
	if len(out) == 0 {
		t.Error("Thumbnail returned empty")
	}
	// Thumbnail always produces JPEG
	if len(jpg) > 0 && !bytes.HasPrefix(out, []byte("\xff\xd8\xff")) {
		t.Error("output is not JPEG")
	}
}

func TestThumbnail_invalid(t *testing.T) {
	_, err := Thumbnail([]byte("not an image"), 200)
	if err == nil {
		t.Error("expected error for invalid image")
	}
}

func TestImage_smallJPEGPassthrough(t *testing.T) {
	jpg := minJPEG(t)
	out, ct, err := Image(jpg)
	if err != nil {
		t.Fatalf("Image: %v", err)
	}
	if ct != "image/jpeg" {
		t.Errorf("content type = %q", ct)
	}
	if !bytes.Equal(out, jpg) {
		t.Error("small JPEG should be returned as-is")
	}
}

func TestImage_resizesLarge(t *testing.T) {
	// Create a 3000x2000 RGBA and encode as JPEG (over maxDimension 1920)
	img := image.NewRGBA(image.Rect(0, 0, 3000, 2000))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	out, ct, err := Image(data)
	if err != nil {
		t.Fatalf("Image: %v", err)
	}
	if ct != "image/jpeg" {
		t.Errorf("content type = %q", ct)
	}
	decoded, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode output: %v", err)
	}
	b := decoded.Bounds()
	if b.Dx() > 1920 || b.Dy() > 1920 {
		t.Errorf("resized image too large: %dx%d", b.Dx(), b.Dy())
	}
}

func TestImage_invalid(t *testing.T) {
	_, _, err := Image([]byte("not an image"))
	if err == nil {
		t.Error("expected error for invalid image")
	}
}

func TestSniffContentType(t *testing.T) {
	if got := SniffContentType(minJPEG(t)); got != "image/jpeg" {
		t.Errorf("SniffContentType(jpeg) = %q", got)
	}
}
