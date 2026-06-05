package cmd

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"localtun/internal/config"
	"localtun/internal/console"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "停止 SSH 反向隧道",
	RunE:  runStop,
}

func init() {
	stopCmd.Flags().StringArrayVarP(&selectedServers, "server", "s", nil, "只处理指定服务器，可重复传入")
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	ui := console.ForStderr()
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}
	profiles, err := selectProfiles(cfg, selectedServers)
	if err != nil {
		return err
	}
	requireRunning := len(selectedServers) > 0

	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	for _, profile := range profiles {
		if err := stopProfile(dataDir, profile.Name, ui, requireRunning); err != nil {
			return err
		}
	}
	return nil
}

func stopProfile(dataDir, name string, ui console.Styler, requireRunning bool) error {
	pidFile := profilePIDFile(dataDir, name)
	info, err := readPIDInfo(pidFile)
	if err != nil {
		if errors.Is(err, errInvalidPID) {
			os.Remove(pidFile)
			return fmt.Errorf("[%s] %s", name, ui.Warning("PID 文件格式错误，已清理"))
		}
		if !requireRunning {
			out := console.ForStdout()
			fmt.Printf("%s %s 未运行\n", out.WarningMark(), out.Info(name))
			return nil
		}
		return fmt.Errorf("[%s] %s %s", name, ui.Warning("未找到运行中的隧道"), ui.Muted("(PID 文件不存在)"))
	}

	if !processInfoRunning(info) {
		os.Remove(pidFile)
		if !requireRunning {
			out := console.ForStdout()
			fmt.Printf("%s %s 未运行，已清理 PID 文件\n", out.WarningMark(), out.Info(name))
			return nil
		}
		return fmt.Errorf("[%s] %s %s，已清理 PID 文件", name, ui.Warning("未找到 localtun 进程"), ui.Accent(fmt.Sprint(info.PID)))
	}

	proc, err := os.FindProcess(info.PID)
	if err != nil {
		os.Remove(pidFile)
		return fmt.Errorf("[%s] %s %s，已清理 PID 文件", name, ui.Warning("未找到进程"), ui.Accent(fmt.Sprint(info.PID)))
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		os.Remove(pidFile)
		return fmt.Errorf("[%s] %s %s: %w", name, ui.Error("发送停止信号失败"), ui.Muted("(进程可能已退出)"), err)
	}

	// Wait for process to exit
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			os.Remove(pidFile)
			out := console.ForStdout()
			fmt.Printf("%s %s 隧道已停止 (PID: %s)\n", out.SuccessMark(), out.Info(name), out.Accent(fmt.Sprint(info.PID)))
			return nil
		}
	}

	return fmt.Errorf("[%s] 进程 %s 仍在运行，未删除 PID 文件", name, ui.Accent(fmt.Sprint(info.PID)))
}
