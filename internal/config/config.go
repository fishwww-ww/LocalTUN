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

type KeepaliveConfig struct {
	Interval int `yaml:"interval"`
	MaxCount int `yaml:"max_count"`
}

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Tunnel    TunnelConfig    `yaml:"tunnel"`
	Keepalive KeepaliveConfig `yaml:"keepalive"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 22,
			User: "root",
		},
		Tunnel: TunnelConfig{
			RemotePort: 1080,
			LocalPort:  7897,
		},
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
	if c.Server.Host == "" {
		return fmt.Errorf("server.host 不能为空")
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port 必须在 1-65535 之间")
	}
	if c.Server.User == "" {
		return fmt.Errorf("server.user 不能为空")
	}
	if c.Server.KeyPath == "" {
		return fmt.Errorf("server.key_path 不能为空")
	}
	if c.Tunnel.RemotePort <= 0 || c.Tunnel.RemotePort > 65535 {
		return fmt.Errorf("tunnel.remote_port 必须在 1-65535 之间")
	}
	if c.Tunnel.LocalPort <= 0 || c.Tunnel.LocalPort > 65535 {
		return fmt.Errorf("tunnel.local_port 必须在 1-65535 之间")
	}
	return nil
}

// ExpandKeyPath resolves ~ to home directory in the key path.
func (c *Config) ExpandKeyPath() (string, error) {
	p := c.Server.KeyPath
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
