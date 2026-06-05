package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	rootCmd.AddCommand(stopCmd)
}

func runStop(cmd *cobra.Command, args []string) error {
	ui := console.ForStderr()
	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	pidFile := filepath.Join(dataDir, "localtun.pid")
	info, err := readPIDInfo(pidFile)
	if err != nil {
		if errors.Is(err, errInvalidPID) {
			os.Remove(pidFile)
			return fmt.Errorf("%s", ui.Warning("PID 文件格式错误，已清理"))
		}
		return fmt.Errorf("%s %s", ui.Warning("未找到运行中的隧道"), ui.Muted("(PID 文件不存在)"))
	}

	if !processInfoRunning(info) {
		os.Remove(pidFile)
		return fmt.Errorf("%s %s，已清理 PID 文件", ui.Warning("未找到 localtun 进程"), ui.Accent(fmt.Sprint(info.PID)))
	}

	proc, err := os.FindProcess(info.PID)
	if err != nil {
		os.Remove(pidFile)
		return fmt.Errorf("%s %s，已清理 PID 文件", ui.Warning("未找到进程"), ui.Accent(fmt.Sprint(info.PID)))
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		os.Remove(pidFile)
		return fmt.Errorf("%s %s: %w", ui.Error("发送停止信号失败"), ui.Muted("(进程可能已退出)"), err)
	}

	// Wait for process to exit
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			os.Remove(pidFile)
			out := console.ForStdout()
			fmt.Printf("%s 隧道已停止 (PID: %s)\n", out.SuccessMark(), out.Accent(fmt.Sprint(info.PID)))
			return nil
		}
	}

	return fmt.Errorf("进程 %s 仍在运行，未删除 PID 文件", ui.Accent(fmt.Sprint(info.PID)))
}
