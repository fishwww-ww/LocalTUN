package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"localtun/internal/config"
	"localtun/internal/console"
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

	if err := checkLocalProxyPort(cfg); err != nil {
		return err
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

	logStyle := console.Plain()
	if !daemonFlag {
		logStyle = console.ForStdout()
	}
	logger := log.New(logFile, logStyle.Prefix("tunnel"), log.LstdFlags)

	if !daemonFlag {
		ui := console.ForStdout()
		fmt.Printf("%s %s:%s → 本地 %s\n", ui.Label("隧道配置:"), ui.Accent(cfg.Server.Host), ui.Accent(fmt.Sprint(cfg.Tunnel.RemotePort)), ui.Accent(fmt.Sprintf(":%d", cfg.Tunnel.LocalPort)))
		fmt.Println(ui.Muted("按 Ctrl+C 停止隧道"))
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
	if err := t.Run(ctx); err != nil {
		return explainStartError(err, cfg)
	}
	return nil
}

func checkLocalProxyPort(cfg *config.Config) error {
	localAddr := fmt.Sprintf("127.0.0.1:%d", cfg.Tunnel.LocalPort)
	conn, err := net.DialTimeout("tcp", localAddr, 2*time.Second)
	if err == nil {
		conn.Close()
		return nil
	}

	return fmt.Errorf(
		"%s: %s\n\n"+
			"请先确认本地代理客户端已启动，并且 HTTP 或 mixed 代理端口是 %d。\n"+
			"常见检查:\n"+
			"  1. Clash/Mihomo/Surge/V2Ray 是否正在运行\n"+
			"  2. 配置里的 tunnel.local_port 是否写对\n"+
			"  3. 本机是否允许连接 127.0.0.1:%d\n\n"+
			"原始错误: %v",
		console.ForStderr().Error("本地代理端口不可连接"),
		console.ForStderr().Accent(localAddr),
		cfg.Tunnel.LocalPort,
		cfg.Tunnel.LocalPort,
		err,
	)
}

func explainStartError(err error, cfg *config.Config) error {
	ui := console.ForStderr()
	if errors.Is(err, tunnel.ErrRemoteListenFailed) {
		return fmt.Errorf(
			"%w\n\n"+
				"%s %s 监听失败，隧道没有建立成功。\n"+
				"建议按顺序检查:\n"+
				"  1. 先运行 `localtun setup`，确保远端开启 AllowTcpForwarding 和 GatewayPorts\n"+
				"  2. 登录远端执行 `ss -lntp | grep :%d`，确认端口没有被占用\n"+
				"  3. 如果云厂商有安全组/防火墙，请放行 TCP :%d\n"+
				"  4. 如果刚改过 sshd_config，请重启 sshd 或重新连接后再试",
			err,
			ui.Error("远程端口"),
			ui.Accent(fmt.Sprintf(":%d", cfg.Tunnel.RemotePort)),
			cfg.Tunnel.RemotePort,
			cfg.Tunnel.RemotePort,
		)
	}

	return fmt.Errorf(
		"%w\n\n"+
			"%s。建议检查:\n"+
			"  1. SSH 密钥是否可用: %s\n"+
			"  2. 是否能手动 SSH 登录: ssh -i %s -p %d %s@%s\n"+
			"  3. 远端网络和本地代理是否都已就绪",
		err,
		ui.Error("隧道启动失败"),
		ui.Accent(cfg.Server.KeyPath),
		ui.Accent(cfg.Server.KeyPath),
		cfg.Server.Port,
		cfg.Server.User,
		ui.Accent(cfg.Server.Host),
	)
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

	ui := console.ForStdout()
	fmt.Printf("%s 隧道已在后台启动 (PID: %s)\n", ui.SuccessMark(), ui.Accent(fmt.Sprint(proc.Pid)))
	fmt.Printf("%s %s\n", ui.Label("日志文件:"), ui.Accent(logPath))
	fmt.Printf("%s %s\n", ui.Label("停止隧道:"), ui.Info("localtun stop"))
	return nil
}
