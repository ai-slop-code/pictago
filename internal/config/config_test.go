package config

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	t.Setenv("TEST_GETENV_KEY", "set-value")
	t.Setenv("TEST_GETENV_EMPTY", "")

	if got := GetEnv("TEST_GETENV_KEY", "default"); got != "set-value" {
		t.Errorf("GetEnv(set) = %q, want set-value", got)
	}
	if got := GetEnv("TEST_GETENV_EMPTY", "default"); got != "default" {
		t.Errorf("GetEnv(empty) = %q, want default", got)
	}
	if got := GetEnv("TEST_GETENV_UNSET", "default"); got != "default" {
		t.Errorf("GetEnv(unset) = %q, want default", got)
	}
}

func TestGetEnvInt(t *testing.T) {
	t.Setenv("TEST_GETENVINT_VALID", "42")
	t.Setenv("TEST_GETENVINT_ZERO", "0")
	t.Setenv("TEST_GETENVINT_BAD", "not-a-number")

	if got := GetEnvInt("TEST_GETENVINT_VALID", 10); got != 42 {
		t.Errorf("GetEnvInt(42) = %d, want 42", got)
	}
	if got := GetEnvInt("TEST_GETENVINT_ZERO", 10); got != 10 {
		t.Errorf("GetEnvInt(0) returns default = %d, want 10", got)
	}
	if got := GetEnvInt("TEST_GETENVINT_BAD", 10); got != 10 {
		t.Errorf("GetEnvInt(bad) returns default = %d, want 10", got)
	}
	if got := GetEnvInt("TEST_GETENVINT_UNSET", 7); got != 7 {
		t.Errorf("GetEnvInt(unset) = %d, want 7", got)
	}
	// Ensure we don't pollute env for other tests
	_ = os.Unsetenv("TEST_GETENVINT_VALID")
}
