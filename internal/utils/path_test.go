package utils

import "testing"

func TestEnsureTrailingSlash(t *testing.T) {
	t.Run("Should be able to ensure trailing slash", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"folder", "folder/"},
			{"folder/", "folder/"},
			{"", "/"},
			{"/", "/"},
			{"foo/bar", "foo/bar/"},
			{"foo/bar/", "foo/bar/"},
		}

		for _, tt := range tests {
			got := EnsureTrailingSlash(tt.input)
			if got != tt.expected {
				t.Errorf("EnsureTrailingSlash(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		}
	})
}
