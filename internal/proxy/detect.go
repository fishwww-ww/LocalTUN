package proxy

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

var DefaultPorts = []int{7890, 7897, 1080, 20170}

type Detector struct {
	Timeout time.Duration
	Dial    func(address string, timeout time.Duration) (net.Conn, error)
}

func (d Detector) Detect(override string) (string, error) {
	timeout := d.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	dial := d.Dial
	if dial == nil {
		dial = func(address string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("tcp", address, timeout)
		}
	}

	if strings.TrimSpace(override) != "" {
		addr, err := NormalizeAddress(override)
		if err != nil {
			return "", err
		}
		if err := probe(dial, addr, timeout); err != nil {
			return "", fmt.Errorf("本地代理不可连接 %s: %w", addr, err)
		}
		return addr, nil
	}

	var lastErr error
	for _, port := range DefaultPorts {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		if err := probe(dial, addr, timeout); err == nil {
			return addr, nil
		} else {
			lastErr = err
		}
	}
	return "", fmt.Errorf("未发现可用本地代理端口，已扫描 %v: %w", DefaultPorts, lastErr)
}

func NormalizeAddress(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("本地代理地址不能为空")
	}
	if _, err := strconv.Atoi(raw); err == nil {
		raw = "127.0.0.1:" + raw
	}
	host, portRaw, err := net.SplitHostPort(raw)
	if err != nil {
		return "", fmt.Errorf("本地代理地址格式应为 host:port 或 port: %s", raw)
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil || port <= 0 || port > 65535 {
		return "", fmt.Errorf("本地代理端口必须在 1-65535 之间")
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}

func probe(dial func(address string, timeout time.Duration) (net.Conn, error), addr string, timeout time.Duration) error {
	conn, err := dial(addr, timeout)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
