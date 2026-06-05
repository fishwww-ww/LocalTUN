package remote

import (
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"localtun/internal/config"
	"localtun/internal/sshutil"
)

type Setup struct {
	cfg    *config.Config
	client *ssh.Client
	logger *log.Logger
}

type DiagnosticResult struct {
	Name   string
	OK     bool
	Detail string
	Hint   string
}

func NewSetup(cfg *config.Config, logger *log.Logger) *Setup {
	return &Setup{cfg: cfg, logger: logger}
}

func (s *Setup) Connect() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	s.logger.Printf("正在连接 %s@%s ...", s.cfg.Server.User, addr)

	client, err := sshutil.Dial(s.cfg)
	if err != nil {
		return err
	}
	s.client = client
	s.logger.Printf("SSH 连接成功")
	return nil
}

func (s *Setup) Close() {
	if s.client != nil {
		s.client.Close()
	}
}

func (s *Setup) runCommand(cmd string) (string, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建 SSH session 失败: %w", err)
	}
	defer session.Close()

	out, err := session.CombinedOutput(cmd)
	return string(out), err
}

func (s *Setup) ConfigureSSHD() error {
	s.logger.Println("=== 配置 sshd_config ===")

	timestamp := time.Now().Format("20060102_150405")
	backupCmd := fmt.Sprintf("cp /etc/ssh/sshd_config /etc/ssh/sshd_config.bak.%s", timestamp)
	if _, err := s.runCommand(backupCmd); err != nil {
		return fmt.Errorf("备份 sshd_config 失败: %w", err)
	}
	s.logger.Printf("已备份 sshd_config → sshd_config.bak.%s", timestamp)

	settings := []struct {
		key string
		val string
	}{
		{"AllowTcpForwarding", "yes"},
		{"GatewayPorts", "yes"},
		{"PermitTunnel", "yes"},
	}

	for _, setting := range settings {
		cmd := fmt.Sprintf(
			`grep -q '^[[:space:]]*#*[[:space:]]*%s[[:space:]]' /etc/ssh/sshd_config && `+
				`sed -i 's/^[[:space:]]*#*[[:space:]]*%s[[:space:]].*/%s %s/' /etc/ssh/sshd_config || `+
				`echo '%s %s' >> /etc/ssh/sshd_config`,
			setting.key, setting.key, setting.key, setting.val, setting.key, setting.val,
		)
		if _, err := s.runCommand(cmd); err != nil {
			return fmt.Errorf("设置 %s 失败: %w", setting.key, err)
		}
		s.logger.Printf("  %s %s ✓", setting.key, setting.val)
	}

	return nil
}

func (s *Setup) RestartSSHD() error {
	s.logger.Println("正在重启 sshd ...")
	if _, err := s.runCommand("systemctl restart sshd"); err != nil {
		// Fallback: some systems use ssh instead of sshd
		if _, err2 := s.runCommand("systemctl restart ssh"); err2 != nil {
			return fmt.Errorf("重启 sshd 失败 (尝试了 sshd 和 ssh): %w", err)
		}
	}
	s.logger.Println("sshd 重启成功 ✓")
	return nil
}

func (s *Setup) ConfigureBashRC() error {
	s.logger.Println("=== 配置 .bashrc ===")

	remotePort := s.cfg.Tunnel.RemotePort

	out, _ := s.runCommand("cat ~/.bashrc")
	if strings.Contains(out, "# === LocalTUN Proxy Config ===") {
		s.logger.Println("检测到已有 LocalTUN 配置，将先移除旧配置")
		removeCmd := `sed -i '/# === LocalTUN Proxy Config ===/,/# === End LocalTUN ===/d' ~/.bashrc`
		if _, err := s.runCommand(removeCmd); err != nil {
			return fmt.Errorf("移除旧配置失败: %w", err)
		}
	}

	timestamp := time.Now().Format("20060102_150405")
	backupCmd := fmt.Sprintf("cp ~/.bashrc ~/.bashrc.bak.%s", timestamp)
	if _, err := s.runCommand(backupCmd); err != nil {
		s.logger.Printf("备份 .bashrc 失败 (非致命): %v", err)
	} else {
		s.logger.Printf("已备份 .bashrc → .bashrc.bak.%s", timestamp)
	}

	proxyBlock := fmt.Sprintf(`
# === LocalTUN Proxy Config ===
case "$-" in
    *i*) ;;
    *) return 0 2>/dev/null || exit 0 ;;
esac

PROXY_SERVER="127.0.0.1"
PROXY_PORT="%d"
PROXY_URL="http://$PROXY_SERVER:$PROXY_PORT"

proxy_on() {
    export http_proxy="$PROXY_URL"
    export https_proxy="$PROXY_URL"
    export HTTP_PROXY="$PROXY_URL"
    export HTTPS_PROXY="$PROXY_URL"
    echo "✓ 代理已启用"
}

proxy_off() {
    unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY
    echo "✗ 代理已关闭"
}

proxy_test() {
    echo "测试代理..."
    curl --proxy "$PROXY_URL" -I -s --max-time 5 https://www.google.com
}
# === End LocalTUN ===
`, remotePort)

	encodedBlock := base64.StdEncoding.EncodeToString([]byte(proxyBlock))
	appendCmd := fmt.Sprintf(`printf '%%s' '%s' | base64 -d >> ~/.bashrc`, encodedBlock)
	if _, err := s.runCommand(appendCmd); err != nil {
		return fmt.Errorf("写入 .bashrc 失败: %w", err)
	}

	s.logger.Println(".bashrc 配置写入成功 ✓")
	s.logger.Println("  已添加: proxy_on / proxy_off / proxy_test 函数")
	s.logger.Println("  提示: 重新登录后运行 proxy_on 可启用代理")
	return nil
}

// RunTest executes proxy test commands on the remote server.
func (s *Setup) RunTest() (string, error) {
	remotePort := s.cfg.Tunnel.RemotePort

	tests := []struct {
		name string
		url  string
	}{
		{"国内 (baidu.com)", "https://www.baidu.com"},
		{"国外 (google.com)", "https://www.google.com"},
	}

	var results []string
	for _, t := range tests {
		cmd := fmt.Sprintf("curl --proxy http://127.0.0.1:%d -I -s --max-time 5 %s", remotePort, t.url)
		out, err := s.runCommand(cmd)
		if err != nil {
			results = append(results, fmt.Sprintf("  ✗ %s: 请求失败", t.name))
		} else {
			firstLine := strings.SplitN(out, "\n", 2)[0]
			results = append(results, fmt.Sprintf("  ✓ %s: %s", t.name, strings.TrimSpace(firstLine)))
		}
	}

	return strings.Join(results, "\n"), nil
}

func (s *Setup) RunDiagnostics() []DiagnosticResult {
	results := []DiagnosticResult{
		{
			Name:   "SSH 连接",
			OK:     s.client != nil,
			Detail: fmt.Sprintf("%s@%s:%d", s.cfg.Server.User, s.cfg.Server.Host, s.cfg.Server.Port),
		},
	}

	if s.client == nil {
		results[0].Hint = "SSH 未连接，无法继续远端诊断。"
		return results
	}

	if out, err := s.runCommand("command -v curl"); err != nil || strings.TrimSpace(out) == "" {
		results = append(results, DiagnosticResult{
			Name:   "远端 curl",
			OK:     false,
			Detail: strings.TrimSpace(out),
			Hint:   "请在远端安装 curl，或手动使用其它工具测试代理端口。",
		})
		return results
	}

	results = append(results, DiagnosticResult{
		Name:   "远端 curl",
		OK:     true,
		Detail: "curl 可用",
	})

	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", s.cfg.Tunnel.RemotePort)
	proxyProbeCmd := fmt.Sprintf("curl --proxy %s -I -sS --max-time 5 https://www.baidu.com", proxyURL)
	out, err := s.runCommand(proxyProbeCmd)
	if err != nil {
		results = append(results, DiagnosticResult{
			Name:   "远端代理端口",
			OK:     false,
			Detail: trimDiagnosticOutput(out),
			Hint:   fmt.Sprintf("远端 127.0.0.1:%d 暂时无法作为代理使用。请确认 `localtun start` 正在运行，并检查本地代理端口是否可连接。", s.cfg.Tunnel.RemotePort),
		})
		return results
	}

	results = append(results, DiagnosticResult{
		Name:   "远端代理端口",
		OK:     true,
		Detail: firstNonEmptyLine(out),
	})

	tests := []struct {
		name string
		url  string
	}{
		{"国内站点 baidu.com", "https://www.baidu.com"},
		{"国外站点 google.com", "https://www.google.com"},
	}

	for _, test := range tests {
		cmd := fmt.Sprintf("curl --proxy %s -I -sS --max-time 8 %s", proxyURL, test.url)
		out, err := s.runCommand(cmd)
		if err != nil {
			results = append(results, DiagnosticResult{
				Name:   test.name,
				OK:     false,
				Detail: trimDiagnosticOutput(out),
				Hint:   "代理链路可达但目标请求失败，请检查本地代理规则、节点可用性或远端 DNS/网络限制。",
			})
			continue
		}
		results = append(results, DiagnosticResult{
			Name:   test.name,
			OK:     true,
			Detail: firstNonEmptyLine(out),
		})
	}

	return results
}

func firstNonEmptyLine(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "无输出"
}

func trimDiagnosticOutput(out string) string {
	out = strings.TrimSpace(out)
	if out == "" {
		return "无输出"
	}
	lines := strings.Split(out, "\n")
	if len(lines) > 3 {
		lines = lines[:3]
	}
	return strings.Join(lines, " | ")
}
