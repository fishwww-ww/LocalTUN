package cmd

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"localtun/internal/config"
	"localtun/internal/console"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "交互式生成配置文件",
	Long:  `引导填写服务器信息，生成 ~/.localtun/config.yaml 配置文件。`,
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func prompt(reader *bufio.Reader, label, defaultVal string) string {
	ui := console.ForStdout()
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", ui.Label(label), ui.Muted(defaultVal))
	} else {
		fmt.Printf("%s: ", ui.Label(label))
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func promptInt(reader *bufio.Reader, label string, defaultVal int) int {
	for {
		s := prompt(reader, label, strconv.Itoa(defaultVal))
		v, err := strconv.Atoi(s)
		if err == nil && v > 0 && v <= 65535 {
			return v
		}
		fmt.Printf("  %s %s\n", console.ForStdout().WarningMark(), console.ForStdout().Warning("请输入 1-65535 之间的端口号"))
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	cfgPath := cfgFile
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath()
	}

	if _, err := os.Stat(cfgPath); err == nil {
		ui := console.ForStdout()
		if !confirm(fmt.Sprintf("配置文件 %s 已存在，是否覆盖?", ui.Accent(cfgPath))) {
			fmt.Println(ui.Muted("已取消"))
			return nil
		}
		fmt.Println()
	}

	reader := bufio.NewReader(os.Stdin)
	defaults := config.DefaultConfig()

	ui := console.ForStdout()
	fmt.Println(ui.Label("=== LocalTUN 配置初始化 ==="))
	fmt.Println()

	host := prompt(reader, "服务器 IP", "")
	for host == "" {
		fmt.Printf("  %s %s\n", ui.WarningMark(), ui.Warning("服务器 IP 不能为空"))
		host = prompt(reader, "服务器 IP", "")
	}

	user := prompt(reader, "SSH 用户名", defaults.Server.User)
	sshPort := promptInt(reader, "SSH 端口", defaults.Server.Port)
	keyPath := prompt(reader, "SSH 密钥路径", "~/.ssh/id_rsa")
	remotePort := promptInt(reader, "远程代理端口", defaults.Tunnel.RemotePort)
	localPort := promptInt(reader, "本地代理端口", defaults.Tunnel.LocalPort)

	cfg := config.DefaultConfig()
	cfg.Server.Host = host
	cfg.Server.User = user
	cfg.Server.Port = sshPort
	cfg.Server.KeyPath = keyPath
	cfg.Tunnel.RemotePort = remotePort
	cfg.Tunnel.LocalPort = localPort

	fmt.Println()
	printInitSummary(cfg, cfgPath)

	if err := cfg.Save(cfgPath); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("%s 配置已保存到 %s\n", ui.SuccessMark(), ui.Accent(cfgPath))
	fmt.Println()
	fmt.Println(ui.Label("后续步骤:"))
	fmt.Printf("  1. %s    — 配置远程服务器\n", ui.Info("localtun setup"))
	fmt.Printf("  2. %s    — 启动隧道\n", ui.Info("localtun start"))
	fmt.Printf("  3. %s     — 测试连通性\n", ui.Info("localtun test"))

	return nil
}

func printInitSummary(cfg *config.Config, cfgPath string) {
	ui := console.ForStdout()
	fmt.Println(ui.Label("配置摘要:"))
	fmt.Printf("  %s   %s\n", ui.Label("配置文件:"), ui.Accent(cfgPath))
	fmt.Printf("  %s     %s@%s:%s\n", ui.Label("服务器:"), ui.Info(cfg.Server.User), ui.Accent(cfg.Server.Host), ui.Accent(strconv.Itoa(cfg.Server.Port)))
	fmt.Printf("  %s       远程 %s → 本地 %s\n", ui.Label("隧道:"), ui.Accent(fmt.Sprintf(":%d", cfg.Tunnel.RemotePort)), ui.Accent(fmt.Sprintf(":%d", cfg.Tunnel.LocalPort)))
	fmt.Printf("  %s   %s\n", ui.Label("SSH 密钥:"), ui.Accent(cfg.Server.KeyPath))
	fmt.Println()
	fmt.Println(ui.Label("预检结果:"))

	keyPath, err := cfg.ExpandKeyPath()
	if err != nil {
		fmt.Printf("  %s %s: %v\n", ui.WarningMark(), ui.Warning("SSH 密钥路径无法展开"), err)
	} else if _, err := os.Stat(keyPath); err != nil {
		fmt.Printf("  %s %s: %s\n", ui.WarningMark(), ui.Warning("SSH 密钥未找到"), ui.Accent(keyPath))
		fmt.Printf("    %s 请确认路径正确，或稍后手动编辑配置文件。\n", ui.Warning("提示:"))
	} else {
		fmt.Printf("  %s SSH 密钥存在: %s\n", ui.SuccessMark(), ui.Accent(keyPath))
	}

	if cfg.Server.Port == cfg.Tunnel.RemotePort {
		fmt.Printf("  %s %s (%s)\n", ui.WarningMark(), ui.Warning("远程代理端口与 SSH 端口相同"), ui.Accent(fmt.Sprintf(":%d", cfg.Tunnel.RemotePort)))
		fmt.Printf("    %s 建议为远程代理端口换一个未占用端口，例如 %s。\n", ui.Warning("提示:"), ui.Accent("1080"))
	} else {
		fmt.Printf("  %s 远程代理端口看起来可用: %s\n", ui.SuccessMark(), ui.Accent(fmt.Sprintf(":%d", cfg.Tunnel.RemotePort)))
	}

	localAddr := fmt.Sprintf("127.0.0.1:%d", cfg.Tunnel.LocalPort)
	conn, err := net.DialTimeout("tcp", localAddr, 800*time.Millisecond)
	if err != nil {
		fmt.Printf("  %s %s: %s\n", ui.WarningMark(), ui.Warning("本地代理端口暂时无法连接"), ui.Accent(localAddr))
		fmt.Printf("    %s 启动隧道前请先启动 Clash、Mihomo、Surge、V2Ray 等本地代理。\n", ui.Warning("提示:"))
		return
	}
	conn.Close()
	fmt.Printf("  %s 本地代理端口可连接: %s\n", ui.SuccessMark(), ui.Accent(localAddr))
}
