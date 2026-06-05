package console

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestPlainStylerDoesNotColor(t *testing.T) {
	styler := Plain()
	if got := styler.Success("ok"); got != "ok" {
		t.Fatalf("Success() = %q, want plain text", got)
	}
}

func TestNewWithColorWrapsText(t *testing.T) {
	styler := NewWithColor(true)
	got := styler.Error("bad")
	if !strings.Contains(got, "\x1b[31m") || !strings.HasSuffix(got, reset) {
		t.Fatalf("Error() = %q, want red ANSI wrapped text", got)
	}
}

func TestNewWithNonTTYDefaultsToPlain(t *testing.T) {
	styler := New(&bytes.Buffer{})
	if styler.Enabled() {
		t.Fatal("New() enabled color for non-TTY writer")
	}
	if got := styler.Info("info"); got != "info" {
		t.Fatalf("Info() = %q, want plain text", got)
	}
}

func TestNoColorDisablesColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	unsetEnv(t, "FORCE_COLOR")

	styler := New(os.Stdout)
	if styler.Enabled() {
		t.Fatal("New() enabled color when NO_COLOR is set")
	}
}

func TestForceColorEnablesColor(t *testing.T) {
	unsetEnv(t, "NO_COLOR")
	t.Setenv("FORCE_COLOR", "1")

	styler := New(&bytes.Buffer{})
	if !styler.Enabled() {
		t.Fatal("New() did not enable color when FORCE_COLOR is set")
	}
	if got := styler.Success("ok"); !strings.Contains(got, "\x1b[32m") {
		t.Fatalf("Success() = %q, want green ANSI text", got)
	}
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	old, hadOld := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("Unsetenv(%q) error = %v", key, err)
	}
	t.Cleanup(func() {
		if hadOld {
			_ = os.Setenv(key, old)
			return
		}
		_ = os.Unsetenv(key)
	})
}
