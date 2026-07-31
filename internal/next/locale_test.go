package next

import "testing"

func TestSelectUTF8Locale(t *testing.T) {
	tests := []struct {
		name      string
		available string
		want      string
	}{
		{"prefers neutral locale", "POSIX\nzh_CN.UTF-8\nC.UTF-8\n", "C.UTF-8"},
		{"accepts glibc spelling", "C\nC.utf8\nPOSIX\n", "C.utf8"},
		{"falls back to any UTF-8 locale", "C\nfr_FR.UTF-8\n", "fr_FR.UTF-8"},
		{"returns empty without UTF-8", "C\nPOSIX\n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SelectUTF8Locale(tt.available); got != tt.want {
				t.Fatalf("SelectUTF8Locale(%q) = %q, want %q", tt.available, got, tt.want)
			}
		})
	}
}
