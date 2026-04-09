package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"localtun/internal/config"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看隧道运行状态",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		fmt.Println("配置文件未找到或格式错误，仅显示进程状态")
		cfg = nil
	}

	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	pidFile := filepath.Join(dataDir, "localtun.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Println("状态: 未运行")
		if cfg != nil {
			printConfig(cfg)
		}
		return nil
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		fmt.Println("状态: 未运行 (PID 文件损坏)")
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil || proc.Signal(syscall.Signal(0)) != nil {
		fmt.Printf("状态: 未运行 (PID %d 已不存在)\n", pid)
		os.Remove(pidFile)
		return nil
	}

	fmt.Printf("状态: 运行中 (PID: %d)\n", pid)
	if cfg != nil {
		printConfig(cfg)
	}

	logPath := filepath.Join(dataDir, "localtun.log")
	fmt.Printf("日志文件: %s\n", logPath)

	return nil
}

func printConfig(cfg *config.Config) {
	fmt.Printf("服务器:     %s@%s:%d\n", cfg.Server.User, cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("隧道:       远程 :%d → 本地 :%d\n", cfg.Tunnel.RemotePort, cfg.Tunnel.LocalPort)
	fmt.Printf("Keepalive:  每 %ds，最大失败 %d 次\n", cfg.Keepalive.Interval, cfg.Keepalive.MaxCount)
}
