package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"localtun/internal/config"
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
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func promptInt(reader *bufio.Reader, label string, defaultVal int) int {
	s := prompt(reader, label, strconv.Itoa(defaultVal))
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

func runInit(cmd *cobra.Command, args []string) error {
	cfgPath := cfgFile
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath()
	}

	if _, err := os.Stat(cfgPath); err == nil {
		if !confirm(fmt.Sprintf("配置文件 %s 已存在，是否覆盖?", cfgPath)) {
			fmt.Println("已取消")
			return nil
		}
		fmt.Println()
	}

	reader := bufio.NewReader(os.Stdin)
	defaults := config.DefaultConfig()

	fmt.Println("=== LocalTUN 配置初始化 ===")
	fmt.Println()

	host := prompt(reader, "服务器 IP", "")
	for host == "" {
		fmt.Println("  服务器 IP 不能为空")
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

	if err := cfg.Save(cfgPath); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("配置已保存到 %s\n", cfgPath)
	fmt.Println()
	fmt.Println("后续步骤:")
	fmt.Println("  1. localtun setup    — 配置远程服务器")
	fmt.Println("  2. localtun start    — 启动隧道")
	fmt.Println("  3. localtun test     — 测试连通性")

	return nil
}
