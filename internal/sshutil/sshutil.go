package sshutil

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
	"localtun/internal/config"
)

func BuildClientConfig(cfg *config.Config) (*ssh.ClientConfig, error) {
	keyPath, err := cfg.ExpandKeyPath()
	if err != nil {
		return nil, err
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("读取密钥文件 %s 失败: %w", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("解析密钥失败: %w", err)
	}

	return &ssh.ClientConfig{
		User:            cfg.Server.User,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}, nil
}

func Dial(cfg *config.Config) (*ssh.Client, error) {
	sshCfg, err := BuildClientConfig(cfg)
	if err != nil {
		return nil, err
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	client, err := ssh.Dial("tcp", addr, sshCfg)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}
	return client, nil
}
