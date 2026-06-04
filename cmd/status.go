package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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
	info, err := readPIDInfo(pidFile)
	if err != nil {
		if errors.Is(err, errInvalidPID) {
			fmt.Println("状态: 未运行 (PID 文件损坏)")
			return nil
		}
		fmt.Println("状态: 未运行")
		if cfg != nil {
			printConfig(cfg)
		}
		return nil
	}

	if !processInfoRunning(info) {
		fmt.Printf("状态: 未运行 (PID %d 已不存在或不是 localtun)\n", info.PID)
		os.Remove(pidFile)
		return nil
	}

	fmt.Printf("状态: 运行中 (PID: %d)\n", info.PID)
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
