package log

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithWriter(t *testing.T) {
	tests := []struct {
		name   string
		writer *os.File
		expect *os.File
	}{
		{"Should set writer when not nil", os.Stdout, os.Stdout},
		{"Should not set writer when nil", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			opt := WithWriter(tt.writer)
			opt.apply(cfg)
			assert.Equal(t, tt.expect, cfg.writer)
		})
	}
}

func TestWithLevel(t *testing.T) {
	tests := []struct {
		name   string
		level  string
		expect string
	}{
		{"Should set level when not empty", "debug", "debug"},
		{"Should not set level when empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			opt := WithLevel(tt.level)
			opt.apply(cfg)
			assert.Equal(t, tt.expect, cfg.level)
		})
	}
}

func TestWithSource(t *testing.T) {
	t.Run("Should set addSource to true", func(t *testing.T) {
		cfg := &Config{}
		opt := WithSource()
		opt.apply(cfg)
		assert.True(t, cfg.addSource)
	})
}

func TestOptionFunc_apply(t *testing.T) {
	t.Run("Should apply OptionFunc", func(t *testing.T) {
		cfg := &Config{}
		opt := OptionFunc(func(c *Config) { c.level = "info" })
		opt.apply(cfg)
		assert.Equal(t, "info", cfg.level)
	})
}
