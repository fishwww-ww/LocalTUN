package console

import (
	"fmt"
	"io"
	"os"
)

const (
	reset  = "\x1b[0m"
	bold   = "\x1b[1m"
	red    = "\x1b[31m"
	green  = "\x1b[32m"
	yellow = "\x1b[33m"
	blue   = "\x1b[34m"
	cyan   = "\x1b[36m"
	gray   = "\x1b[90m"
)

type Styler struct {
	enabled bool
}

func ForStdout() Styler {
	return New(os.Stdout)
}

func ForStderr() Styler {
	return New(os.Stderr)
}

func Plain() Styler {
	return Styler{}
}

func New(w io.Writer) Styler {
	return Styler{enabled: shouldColor(w)}
}

func NewWithColor(enabled bool) Styler {
	return Styler{enabled: enabled}
}

func (s Styler) Enabled() bool {
	return s.enabled
}

func (s Styler) Success(text string) string {
	return s.wrap(green, text)
}

func (s Styler) Warning(text string) string {
	return s.wrap(yellow, text)
}

func (s Styler) Error(text string) string {
	return s.wrap(red, text)
}

func (s Styler) Info(text string) string {
	return s.wrap(cyan, text)
}

func (s Styler) Accent(text string) string {
	return s.wrap(blue, text)
}

func (s Styler) Muted(text string) string {
	return s.wrap(gray, text)
}

func (s Styler) Label(text string) string {
	return s.wrap(bold, text)
}

func (s Styler) SuccessMark() string {
	return s.Success("✓")
}

func (s Styler) ErrorMark() string {
	return s.Error("✗")
}

func (s Styler) WarningMark() string {
	return s.Warning("!")
}

func (s Styler) Prefix(name string) string {
	return s.Info(fmt.Sprintf("[%s]", name)) + " "
}

func (s Styler) wrap(code, text string) string {
	if !s.enabled {
		return text
	}
	return code + text + reset
}

func shouldColor(w io.Writer) bool {
	if force, ok := os.LookupEnv("FORCE_COLOR"); ok && force != "" && force != "0" {
		return true
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
