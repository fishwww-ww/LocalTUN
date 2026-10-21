package next

import (
	"strings"
	"testing"
)

func TestShellCommandInjectsProxyEnv(t *testing.T) {
	cmd := ShellCommand("/bin/bash", "http://127.0.0.1:46327", "C.UTF-8")
	for _, want := range []string{
		"HTTP_PROXY='http://127.0.0.1:46327'",
		"HTTPS_PROXY='http://127.0.0.1:46327'",
		"ALL_PROXY='http://127.0.0.1:46327'",
		"http_proxy='http://127.0.0.1:46327'",
		"https_proxy='http://127.0.0.1:46327'",
		"all_proxy='http://127.0.0.1:46327'",
		"LANG='C.UTF-8'",
		"LC_CTYPE='C.UTF-8'",
		"LC_ALL='C.UTF-8'",
		"'/bin/bash' -l",
	} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("command %q missing %q", cmd, want)
		}
	}
}

func TestShellCommandFallback(t *testing.T) {
	cmd := ShellCommand("", "http://127.0.0.1:1", "")
	if !strings.Contains(cmd, "'sh' -l") {
		t.Fatalf("command %q did not fallback to sh", cmd)
	}
}
