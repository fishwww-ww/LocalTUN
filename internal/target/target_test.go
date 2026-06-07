package target

import "testing"

func TestParseTarget(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		user string
		host string
		port int
	}{
		{name: "user host", raw: "root@gpu01", user: "root", host: "gpu01", port: 22},
		{name: "user host port", raw: "ubuntu@gpu01:2222", user: "ubuntu", host: "gpu01", port: 2222},
		{name: "ipv6", raw: "root@[2001:db8::1]:2200", user: "root", host: "2001:db8::1", port: 2200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.raw)
			if err != nil {
				t.Fatal(err)
			}
			if got.User != tt.user || got.Host != tt.host || got.Port != tt.port {
				t.Fatalf("Parse(%q) = %#v", tt.raw, got)
			}
		})
	}
}

func TestParseTargetDefaultUser(t *testing.T) {
	got, err := Parse("gpu01")
	if err != nil {
		t.Fatal(err)
	}
	if got.User == "" || got.Host != "gpu01" || got.Port != 22 {
		t.Fatalf("Parse default user = %#v", got)
	}
}

func TestParseTargetErrors(t *testing.T) {
	for _, raw := range []string{"", "@gpu01", "root@", "root@gpu01:bad", "root@gpu01:70000", "ssh://root@gpu01"} {
		if _, err := Parse(raw); err == nil {
			t.Fatalf("Parse(%q) expected error", raw)
		}
	}
}
