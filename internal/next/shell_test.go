package next

import (
	"strings"
	"testing"
)

func TestShellCommandInjectsProxyEnv(t *testing.T) {
	cmd := ShellCommand("/bin/bash", "http://127.0.0.1:46327")
	for _, want := range []string{
		"HTTP_PROXY='http://127.0.0.1:46327'",
		"HTTPS_PROXY='http://127.0.0.1:46327'",
		"ALL_PROXY='http://127.0.0.1:46327'",
		"http_proxy='http://127.0.0.1:46327'",
		"https_proxy='http://127.0.0.1:46327'",
		"all_proxy='http://127.0.0.1:46327'",
		"'/bin/bash' -l",
	} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("command %q missing %q", cmd, want)
		}
	}
}

func TestShellCommandFallback(t *testing.T) {
	cmd := ShellCommand("", "http://127.0.0.1:1")
	if !strings.Contains(cmd, "'sh' -l") {
		t.Fatalf("command %q did not fallback to sh", cmd)
	}
}
