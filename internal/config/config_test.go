package config

import "testing"

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

func validConfig() *Config {
	cfg := DefaultConfig()
	cfg.Server.Host = "example.com"
	cfg.Server.KeyPath = "~/.ssh/id_rsa"
	return cfg
}
