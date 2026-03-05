package validate

import "testing"

func TestIsImage(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"empty", nil, false},
		{"short", []byte{0xff, 0xd8}, false},
		{"jpeg", []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00\x00"), true},
		{"png", []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x00\x00"), true},
		{"gif87a", []byte("GIF87a\x01\x02\x03\x04\x05\x06"), true},
		{"gif89a", []byte("GIF89a\x01\x02\x03\x04\x05\x06"), true},
		{"webp", []byte("RIFF\x00\x00\x00\x00WEBP"), true},
		{"not image", []byte("not an image magic at all"), false},
		{"jpeg prefix but too short", []byte("\xff\xd8\xff"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsImage(tt.data); got != tt.want {
				t.Errorf("IsImage() = %v, want %v", got, tt.want)
			}
		})
	}
}
