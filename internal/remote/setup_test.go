package remote

import "testing"

func TestFirstNonEmptyLine(t *testing.T) {
	got := firstNonEmptyLine("\n\nHTTP/2 200\r\nserver: test\n")
	if got != "HTTP/2 200" {
		t.Fatalf("firstNonEmptyLine() = %q, want %q", got, "HTTP/2 200")
	}
}

func TestFirstNonEmptyLineReturnsFallbackForBlankOutput(t *testing.T) {
	got := firstNonEmptyLine(" \n\t\n")
	if got != "无输出" {
		t.Fatalf("firstNonEmptyLine() = %q, want %q", got, "无输出")
	}
}

func TestTrimDiagnosticOutputLimitsLines(t *testing.T) {
	got := trimDiagnosticOutput("one\ntwo\nthree\nfour")
	want := "one | two | three"
	if got != want {
		t.Fatalf("trimDiagnosticOutput() = %q, want %q", got, want)
	}
}
