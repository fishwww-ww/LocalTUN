package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"localtun/internal/config"
	"localtun/internal/console"
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
	ui := console.ForStdout()
	cfg, err := config.Load(cfgFile)
	if err != nil {
		fmt.Printf("%s 配置文件未找到或格式错误，仅显示进程状态\n", ui.WarningMark())
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
			fmt.Printf("%s %s %s\n", ui.Label("状态:"), ui.Warning("未运行"), ui.Muted("(PID 文件损坏)"))
			return nil
		}
		fmt.Printf("%s %s\n", ui.Label("状态:"), ui.Warning("未运行"))
		if cfg != nil {
			printConfig(cfg)
		}
		return nil
	}

	if !processInfoRunning(info) {
		fmt.Printf("%s %s %s\n", ui.Label("状态:"), ui.Warning("未运行"), ui.Muted(fmt.Sprintf("(PID %d 已不存在或不是 localtun)", info.PID)))
		os.Remove(pidFile)
		return nil
	}

	fmt.Printf("%s %s (PID: %s)\n", ui.Label("状态:"), ui.Success("运行中"), ui.Accent(fmt.Sprint(info.PID)))
	if cfg != nil {
		printConfig(cfg)
	}

	logPath := filepath.Join(dataDir, "localtun.log")
	fmt.Printf("%s %s\n", ui.Label("日志文件:"), ui.Accent(logPath))

	return nil
}

func printConfig(cfg *config.Config) {
	ui := console.ForStdout()
	fmt.Printf("%s     %s@%s:%s\n", ui.Label("服务器:"), ui.Info(cfg.Server.User), ui.Accent(cfg.Server.Host), ui.Accent(fmt.Sprint(cfg.Server.Port)))
	fmt.Printf("%s       远程 %s → 本地 %s\n", ui.Label("隧道:"), ui.Accent(fmt.Sprintf(":%d", cfg.Tunnel.RemotePort)), ui.Accent(fmt.Sprintf(":%d", cfg.Tunnel.LocalPort)))
	fmt.Printf("%s  每 %s，最大失败 %s 次\n", ui.Label("Keepalive:"), ui.Accent(fmt.Sprintf("%ds", cfg.Keepalive.Interval)), ui.Accent(fmt.Sprint(cfg.Keepalive.MaxCount)))
}
