package sshutil

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
)

type Options struct {
	User     string
	Host     string
	Port     int
	Identity string
}

func BuildClientConfig(opts Options) (*ssh.ClientConfig, error) {
	identity, err := ResolveIdentity(opts.Identity)
	if err != nil {
		return nil, err
	}

	keyData, err := os.ReadFile(identity)
	if err != nil {
		return nil, fmt.Errorf("读取密钥文件 %s 失败: %w", identity, err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("解析密钥失败: %w", err)
	}

	return &ssh.ClientConfig{
		User:            opts.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}, nil
}

func DialOptions(opts Options) (*ssh.Client, error) {
	if opts.Port == 0 {
		opts.Port = 22
	}
	sshCfg, err := BuildClientConfig(opts)
	if err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", opts.Host, opts.Port)
	client, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}
	return client, nil
}

func ResolveIdentity(path string) (string, error) {
	if path != "" {
		return expandHome(path)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取 home 目录失败: %w", err)
	}
	candidates := []string{
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_rsa"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return candidates[0], nil
}

func expandHome(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("获取 home 目录失败: %w", err)
		}
		return home, nil
	}
	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("获取 home 目录失败: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
