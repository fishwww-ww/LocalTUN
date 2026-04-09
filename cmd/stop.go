package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"localtun/internal/config"
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
	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	pidFile := filepath.Join(dataDir, "localtun.pid")
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("未找到运行中的隧道 (PID 文件不存在)")
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		os.Remove(pidFile)
		return fmt.Errorf("PID 文件格式错误，已清理")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		os.Remove(pidFile)
		return fmt.Errorf("未找到进程 %d，已清理 PID 文件", pid)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		os.Remove(pidFile)
		return fmt.Errorf("发送停止信号失败 (进程可能已退出): %w", err)
	}

	// Wait for process to exit
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			break
		}
	}

	os.Remove(pidFile)
	fmt.Printf("隧道已停止 (PID: %d)\n", pid)
	return nil
}
