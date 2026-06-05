package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateRejectsInvalidKeepalive(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "zero interval",
			mutate: func(cfg *Config) {
				cfg.Keepalive.Interval = 0
			},
		},
		{
			name: "zero max count",
			mutate: func(cfg *Config) {
				cfg.Keepalive.MaxCount = 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() succeeded, want error")
			}
		})
	}
}

func TestValidateAcceptsDefaultShapeWithRequiredFields(t *testing.T) {
	cfg := validConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestLoadAcceptsMultiServerConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, path, `
servers:
  west:
    host: 1.2.3.4
    port: 22
    user: root
    key_path: ~/.ssh/id_rsa
    remote_port: 1080
    local_port: 7897
  east:
    host: example.com
    port: 2222
    user: ubuntu
    key_path: ~/.ssh/id_ed25519
    remote_port: 1080
    local_port: 7897
keepalive:
  interval: 30
  max_count: 3
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.Servers) != 2 {
		t.Fatalf("servers len = %d, want 2", len(cfg.Servers))
	}
}

func TestLoadRejectsLegacySingleServerConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, path, `
server:
  host: 1.2.3.4
  port: 22
  user: root
  key_path: ~/.ssh/id_rsa
tunnel:
  remote_port: 1080
  local_port: 7897
keepalive:
  interval: 30
  max_count: 3
`)

	if _, err := Load(path); err == nil {
		t.Fatal("Load() succeeded, want error")
	}
}

func TestValidateRejectsEmptyServers(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() succeeded, want error")
	}
}

func TestValidateRejectsInvalidProfile(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ServerProfile)
	}{
		{
			name: "missing host",
			mutate: func(profile *ServerProfile) {
				profile.Host = ""
			},
		},
		{
			name: "invalid ssh port",
			mutate: func(profile *ServerProfile) {
				profile.Port = 0
			},
		},
		{
			name: "missing key",
			mutate: func(profile *ServerProfile) {
				profile.KeyPath = ""
			},
		},
		{
			name: "invalid remote port",
			mutate: func(profile *ServerProfile) {
				profile.RemotePort = 70000
			},
		},
		{
			name: "invalid local port",
			mutate: func(profile *ServerProfile) {
				profile.LocalPort = -1
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := validProfile()
			tt.mutate(&profile)
			cfg := DefaultConfig()
			cfg.Servers["west"] = profile
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() succeeded, want error")
			}
		})
	}
}

func validConfig() *Config {
	cfg := DefaultConfig()
	cfg.Servers["west"] = validProfile()
	return cfg
}

func validProfile() ServerProfile {
	profile := DefaultServerProfile()
	profile.Host = "example.com"
	profile.KeyPath = "~/.ssh/id_rsa"
	return profile
}

func writeConfig(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
