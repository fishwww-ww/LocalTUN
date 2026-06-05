package cmd

import (
	"path/filepath"
	"testing"

	"localtun/internal/config"
)

func TestSelectProfilesDefaultsToAllSorted(t *testing.T) {
	cfg := testProfilesConfig()

	profiles, err := selectProfiles(cfg, nil)
	if err != nil {
		t.Fatalf("selectProfiles() error = %v", err)
	}
	if got, want := profileNames(profiles), []string{"east", "west"}; !sameStrings(got, want) {
		t.Fatalf("profiles = %v, want %v", got, want)
	}
}

func TestSelectProfilesFiltersRequestedNames(t *testing.T) {
	cfg := testProfilesConfig()

	profiles, err := selectProfiles(cfg, []string{"west", "west"})
	if err != nil {
		t.Fatalf("selectProfiles() error = %v", err)
	}
	if got, want := profileNames(profiles), []string{"west"}; !sameStrings(got, want) {
		t.Fatalf("profiles = %v, want %v", got, want)
	}
}

func TestSelectProfilesRejectsUnknownName(t *testing.T) {
	cfg := testProfilesConfig()

	if _, err := selectProfiles(cfg, []string{"missing"}); err == nil {
		t.Fatal("selectProfiles() succeeded, want error")
	}
}

func TestProfileRuntimePathsAreIsolatedByName(t *testing.T) {
	dataDir := t.TempDir()

	if got, want := profilePIDFile(dataDir, "west"), filepath.Join(dataDir, "run", "west.pid"); got != want {
		t.Fatalf("profilePIDFile() = %q, want %q", got, want)
	}
	if got, want := profileLogFile(dataDir, "east"), filepath.Join(dataDir, "logs", "east.log"); got != want {
		t.Fatalf("profileLogFile() = %q, want %q", got, want)
	}
}

func testProfilesConfig() *config.Config {
	cfg := config.DefaultConfig()
	west := config.DefaultServerProfile()
	west.Host = "west.example.com"
	west.KeyPath = "~/.ssh/id_rsa"
	west.Tunnels["proxy"] = config.DefaultTunnelConfig()
	east := config.DefaultServerProfile()
	east.Host = "east.example.com"
	east.KeyPath = "~/.ssh/id_rsa"
	east.Tunnels["proxy"] = config.DefaultTunnelConfig()
	cfg.Servers["west"] = west
	cfg.Servers["east"] = east
	return cfg
}

func profileNames(profiles []selectedProfile) []string {
	names := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		names = append(names, profile.Name)
	}
	return names
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
