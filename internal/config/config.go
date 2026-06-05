package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	User    string `yaml:"user"`
	KeyPath string `yaml:"key_path"`
}

type TunnelConfig struct {
	RemotePort int `yaml:"remote_port"`
	LocalPort  int `yaml:"local_port"`
}

type ServerProfile struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	User       string `yaml:"user"`
	KeyPath    string `yaml:"key_path"`
	RemotePort int    `yaml:"remote_port"`
	LocalPort  int    `yaml:"local_port"`
}

type KeepaliveConfig struct {
	Interval int `yaml:"interval"`
	MaxCount int `yaml:"max_count"`
}

type Config struct {
	Servers   map[string]ServerProfile `yaml:"servers"`
	Keepalive KeepaliveConfig          `yaml:"keepalive"`
}

func DefaultConfig() *Config {
	return &Config{
		Servers: map[string]ServerProfile{},
		Keepalive: KeepaliveConfig{
			Interval: 30,
			MaxCount: 3,
		},
	}
}

// DefaultConfigPath returns ~/.localtun/config.yaml.
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(home, ".localtun", "config.yaml")
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("配置文件不存在: %s\n请先运行 `localtun init` 生成配置", path)
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes the config to a YAML file, creating parent directories as needed.
func (c *Config) Save(path string) error {
	if path == "" {
		path = DefaultConfigPath()
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return nil
}

func (c *Config) Validate() error {
	if len(c.Servers) == 0 {
		return fmt.Errorf("servers 不能为空")
	}
	for name, profile := range c.Servers {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("servers 中存在空服务器名称")
		}
		if err := validateProfile(name, profile); err != nil {
			return err
		}
	}
	if c.Keepalive.Interval <= 0 {
		return fmt.Errorf("keepalive.interval 必须大于 0")
	}
	if c.Keepalive.MaxCount <= 0 {
		return fmt.Errorf("keepalive.max_count 必须大于 0")
	}
	return nil
}

func validateProfile(name string, profile ServerProfile) error {
	prefix := fmt.Sprintf("servers.%s", name)
	if profile.Host == "" {
		return fmt.Errorf("%s.host 不能为空", prefix)
	}
	if profile.Port <= 0 || profile.Port > 65535 {
		return fmt.Errorf("%s.port 必须在 1-65535 之间", prefix)
	}
	if profile.User == "" {
		return fmt.Errorf("%s.user 不能为空", prefix)
	}
	if profile.KeyPath == "" {
		return fmt.Errorf("%s.key_path 不能为空", prefix)
	}
	if profile.RemotePort <= 0 || profile.RemotePort > 65535 {
		return fmt.Errorf("%s.remote_port 必须在 1-65535 之间", prefix)
	}
	if profile.LocalPort <= 0 || profile.LocalPort > 65535 {
		return fmt.Errorf("%s.local_port 必须在 1-65535 之间", prefix)
	}
	return nil
}

func DefaultServerProfile() ServerProfile {
	return ServerProfile{
		Port:       22,
		User:       "root",
		RemotePort: 1080,
		LocalPort:  7897,
	}
}

func (p ServerProfile) ToRuntimeConfig(keepalive KeepaliveConfig) *RuntimeConfig {
	return &RuntimeConfig{
		Server: ServerConfig{
			Host:    p.Host,
			Port:    p.Port,
			User:    p.User,
			KeyPath: p.KeyPath,
		},
		Tunnel: TunnelConfig{
			RemotePort: p.RemotePort,
			LocalPort:  p.LocalPort,
		},
		Keepalive: keepalive,
	}
}

type RuntimeConfig struct {
	Server    ServerConfig
	Tunnel    TunnelConfig
	Keepalive KeepaliveConfig
}

// ExpandKeyPath resolves ~ to home directory in the key path.
func (c *RuntimeConfig) ExpandKeyPath() (string, error) {
	return expandHome(c.Server.KeyPath)
}

func (p ServerProfile) ExpandKeyPath() (string, error) {
	return expandHome(p.KeyPath)
}

func expandHome(p string) (string, error) {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("获取 home 目录失败: %w", err)
		}
		p = filepath.Join(home, p[2:])
	}
	return p, nil
}

// DataDir returns ~/.localtun, creating it if needed.
func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".localtun")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}
