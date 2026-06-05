package cmd

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"localtun/internal/config"
)

var selectedServers []string

var profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

type selectedProfile struct {
	Name    string
	Profile config.ServerProfile
	Runtime *config.RuntimeConfig
}

func validateProfileName(name string) error {
	if !profileNamePattern.MatchString(name) {
		return fmt.Errorf("服务器名称 %q 非法，只能使用字母、数字、点、下划线和短横线，并且必须以字母或数字开头", name)
	}
	return nil
}

func sortedProfileNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Servers))
	for name := range cfg.Servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func selectProfiles(cfg *config.Config, names []string) ([]selectedProfile, error) {
	if len(names) == 0 {
		names = sortedProfileNames(cfg)
	}

	seen := map[string]bool{}
	selected := make([]selectedProfile, 0, len(names))
	var missing []string

	for _, name := range names {
		name = strings.TrimSpace(name)
		if seen[name] {
			continue
		}
		seen[name] = true

		profile, ok := cfg.Servers[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		selected = append(selected, selectedProfile{
			Name:    name,
			Profile: profile,
			Runtime: profile.ToRuntimeConfig(cfg.Keepalive),
		})
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("未知服务器: %s。可用服务器: %s", strings.Join(missing, ", "), strings.Join(sortedProfileNames(cfg), ", "))
	}
	return selected, nil
}

func profileRunDir(dataDir string) string {
	return filepath.Join(dataDir, "run")
}

func profileLogDir(dataDir string) string {
	return filepath.Join(dataDir, "logs")
}

func profilePIDFile(dataDir, name string) string {
	return filepath.Join(profileRunDir(dataDir), name+".pid")
}

func profileLogFile(dataDir, name string) string {
	return filepath.Join(profileLogDir(dataDir), name+".log")
}
