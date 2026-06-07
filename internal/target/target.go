package target

import (
	"fmt"
	"os/user"
	"strconv"
	"strings"
)

const DefaultPort = 22

type Target struct {
	User string
	Host string
	Port int
}

func Parse(raw string) (Target, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Target{}, fmt.Errorf("SSH target 不能为空")
	}
	if strings.Contains(raw, "://") {
		return Target{}, fmt.Errorf("SSH target 不应包含协议前缀: %s", raw)
	}

	t := Target{Port: DefaultPort}
	hostPart := raw
	if at := strings.LastIndex(hostPart, "@"); at >= 0 {
		if at == 0 {
			return Target{}, fmt.Errorf("SSH target 缺少用户名")
		}
		t.User = hostPart[:at]
		hostPart = hostPart[at+1:]
	}
	if t.User == "" {
		t.User = defaultUser()
	}

	host, port, err := splitHostPort(hostPart)
	if err != nil {
		return Target{}, err
	}
	t.Host = host
	if port != 0 {
		t.Port = port
	}
	if t.Host == "" {
		return Target{}, fmt.Errorf("SSH target 缺少 host")
	}
	if strings.ContainsAny(t.User, " \t\r\n") {
		return Target{}, fmt.Errorf("SSH 用户名不能包含空白字符")
	}
	return t, nil
}

func splitHostPort(raw string) (string, int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, fmt.Errorf("SSH target 缺少 host")
	}
	if strings.HasPrefix(raw, "[") {
		end := strings.Index(raw, "]")
		if end < 0 {
			return "", 0, fmt.Errorf("IPv6 host 缺少右括号")
		}
		host := raw[1:end]
		rest := raw[end+1:]
		if rest == "" {
			return host, 0, nil
		}
		if !strings.HasPrefix(rest, ":") {
			return "", 0, fmt.Errorf("IPv6 host 后只能跟 :port")
		}
		port, err := parsePort(rest[1:])
		return host, port, err
	}

	if strings.Count(raw, ":") == 1 {
		parts := strings.SplitN(raw, ":", 2)
		port, err := parsePort(parts[1])
		return parts[0], port, err
	}
	if strings.Count(raw, ":") > 1 {
		return raw, 0, nil
	}
	return raw, 0, nil
}

func parsePort(raw string) (int, error) {
	if raw == "" {
		return 0, fmt.Errorf("SSH 端口不能为空")
	}
	port, err := strconv.Atoi(raw)
	if err != nil || port <= 0 || port > 65535 {
		return 0, fmt.Errorf("SSH 端口必须在 1-65535 之间")
	}
	return port, nil
}

func defaultUser() string {
	u, err := user.Current()
	if err == nil && strings.TrimSpace(u.Username) != "" {
		name := u.Username
		if slash := strings.LastIndexAny(name, `\/`); slash >= 0 {
			name = name[slash+1:]
		}
		return name
	}
	return "root"
}
