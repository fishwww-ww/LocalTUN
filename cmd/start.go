package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"localtun/internal/config"
	"localtun/internal/tunnel"
)

var daemonFlag bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 SSH 反向隧道",
	Long:  `建立 SSH 反向隧道，将远程服务器端口流量转发到本地代理端口。`,
	RunE:  runStart,
}

func init() {
	startCmd.Flags().BoolVarP(&daemonFlag, "daemon", "d", false, "后台运行")
	startCmd.Flags().Bool("foreground", false, "前台运行 (内部使用)")
	startCmd.Flags().MarkHidden("foreground")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}

	dataDir, err := config.DataDir()
	if err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
	}

	pidFile := filepath.Join(dataDir, "localtun.pid")
	foreground, _ := cmd.Flags().GetBool("foreground")

	if !foreground && isRunning(pidFile) {
		return fmt.Errorf("隧道已在运行中 (PID 文件: %s)，请先运行 `localtun stop`", pidFile)
	}

	if daemonFlag && !foreground {
		return daemonize(pidFile)
	}

	if !daemonFlag {
		if err := writePIDInfo(pidFile, os.Getpid()); err != nil {
			return fmt.Errorf("写入 PID 文件失败: %w", err)
		}
		defer removePIDInfo(pidFile, os.Getpid())
	}

	logFile := os.Stdout
	if daemonFlag {
		lf, err := os.OpenFile(
			filepath.Join(dataDir, "localtun.log"),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644,
		)
		if err != nil {
			return fmt.Errorf("打开日志文件失败: %w", err)
		}
		defer lf.Close()
		logFile = lf
	}

	logger := log.New(logFile, "[tunnel] ", log.LstdFlags)

	if !daemonFlag {
		fmt.Printf("隧道配置: %s:%d → 本地 :%d\n", cfg.Server.Host, cfg.Tunnel.RemotePort, cfg.Tunnel.LocalPort)
		fmt.Println("按 Ctrl+C 停止隧道")
		fmt.Println()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		<-sigCh
		logger.Println("收到停止信号，正在关闭隧道...")
		cancel()
	}()

	t := tunnel.New(cfg, logger)
	return t.Run(ctx)
}

func daemonize(pidFile string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	args := []string{exe, "start", "--daemon", "--foreground"}
	if cfgFile != "" {
		args = append(args, "--config", cfgFile)
	}

	dataDir, _ := config.DataDir()
	logPath := filepath.Join(dataDir, "localtun.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	attr := &os.ProcAttr{
		Dir:   ".",
		Env:   os.Environ(),
		Files: []*os.File{os.Stdin, logFile, logFile},
		Sys:   daemonSysProcAttr(),
	}

	proc, err := os.StartProcess(exe, args, attr)
	if err != nil {
		logFile.Close()
		return fmt.Errorf("启动后台进程失败: %w", err)
	}
	logFile.Close()

	if err := writePIDInfo(pidFile, proc.Pid); err != nil {
		return fmt.Errorf("写入 PID 文件失败: %w", err)
	}

	proc.Release()

	fmt.Printf("隧道已在后台启动 (PID: %d)\n", proc.Pid)
	fmt.Printf("日志文件: %s\n", logPath)
	fmt.Printf("停止隧道: localtun stop\n")
	return nil
}
