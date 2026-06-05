package cmd

import (
	"errors"
	"fmt"
	"os"

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
	statusCmd.Flags().StringArrayVarP(&selectedServers, "server", "s", nil, "只处理指定服务器，可重复传入")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}
	profiles, err := selectProfiles(cfg, selectedServers)
	if err != nil {
		return err
	}

	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	for _, profile := range profiles {
		printProfileStatus(dataDir, profile)
	}
	return nil
}

func printProfileStatus(dataDir string, profile selectedProfile) {
	ui := console.ForStdout()
	pidFile := profilePIDFile(dataDir, profile.Name)
	info, err := readPIDInfo(pidFile)
	fmt.Printf("%s %s\n", ui.Label("服务器:"), ui.Info(profile.Name))
	if err != nil {
		if errors.Is(err, errInvalidPID) {
			fmt.Printf("  %s %s %s\n", ui.Label("状态:"), ui.Warning("未运行"), ui.Muted("(PID 文件损坏)"))
		} else {
			fmt.Printf("  %s %s\n", ui.Label("状态:"), ui.Warning("未运行"))
		}
		printConfig(profile)
		fmt.Println()
		return
	}

	if !processInfoRunning(info) {
		fmt.Printf("  %s %s %s\n", ui.Label("状态:"), ui.Warning("未运行"), ui.Muted(fmt.Sprintf("(PID %d 已不存在或不是 localtun)", info.PID)))
		os.Remove(pidFile)
		printConfig(profile)
		fmt.Println()
		return
	}

	fmt.Printf("  %s %s (PID: %s)\n", ui.Label("状态:"), ui.Success("运行中"), ui.Accent(fmt.Sprint(info.PID)))
	printConfig(profile)

	logPath := profileLogFile(dataDir, profile.Name)
	fmt.Printf("  %s %s\n", ui.Label("日志文件:"), ui.Accent(logPath))
	fmt.Println()
}

func printConfig(profile selectedProfile) {
	ui := console.ForStdout()
	cfg := profile.Runtime
	fmt.Printf("  %s     %s@%s:%s\n", ui.Label("SSH:"), ui.Info(cfg.Server.User), ui.Accent(cfg.Server.Host), ui.Accent(fmt.Sprint(cfg.Server.Port)))
	fmt.Printf("  %s   远程 %s → 本地 %s\n", ui.Label("隧道:"), ui.Accent(fmt.Sprintf(":%d", cfg.Tunnel.RemotePort)), ui.Accent(fmt.Sprintf(":%d", cfg.Tunnel.LocalPort)))
	fmt.Printf("  %s  每 %s，最大失败 %s 次\n", ui.Label("Keepalive:"), ui.Accent(fmt.Sprintf("%ds", cfg.Keepalive.Interval)), ui.Accent(fmt.Sprint(cfg.Keepalive.MaxCount)))
}
