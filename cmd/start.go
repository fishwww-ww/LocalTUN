package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
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
	startCmd.Flags().StringArrayVarP(&selectedServers, "server", "s", nil, "只处理指定服务器，可重复传入")
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("创建数据目录失败: %w", err)
	}

	foreground, _ := cmd.Flags().GetBool("foreground")

	for _, profile := range profiles {
		pidFile := profilePIDFile(dataDir, profile.Name)
		if !foreground && isRunning(pidFile) {
			return fmt.Errorf("服务器 %s 的隧道已在运行中 (PID 文件: %s)，请先运行 `localtun stop --server %s`", profile.Name, pidFile, profile.Name)
		}
	}

	for _, profile := range profiles {
		if err := checkLocalProxyPorts(profile.Runtime); err != nil {
			return fmt.Errorf("[%s] %w", profile.Name, err)
		}
	}

	if daemonFlag && !foreground {
		return daemonizeProfiles(profiles)
	}

	if !daemonFlag || foreground {
		if err := os.MkdirAll(profileRunDir(dataDir), 0755); err != nil {
			return fmt.Errorf("创建 PID 目录失败: %w", err)
		}
	}

	if foreground {
		name, ok := foregroundServerName(cmd)
		if !ok {
			return fmt.Errorf("后台内部启动缺少 --server")
		}
		profiles, err = selectProfiles(cfg, []string{name})
		if err != nil {
			return err
		}
	}

	if !daemonFlag || foreground {
		for _, profile := range profiles {
			pidFile := profilePIDFile(dataDir, profile.Name)
			if err := writePIDInfo(pidFile, os.Getpid()); err != nil {
				return fmt.Errorf("写入 PID 文件失败: %w", err)
			}
		}
		defer func() {
			for _, profile := range profiles {
				removePIDInfo(profilePIDFile(dataDir, profile.Name), os.Getpid())
			}
		}()
	}

	logFile := os.Stdout
	if daemonFlag {
		if len(profiles) != 1 {
			return fmt.Errorf("后台内部启动一次只能处理一个服务器")
		}
		if err := os.MkdirAll(profileLogDir(dataDir), 0755); err != nil {
			return fmt.Errorf("创建日志目录失败: %w", err)
		}
		lf, err := os.OpenFile(
			profileLogFile(dataDir, profiles[0].Name),
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
		fmt.Println(ui.Label("隧道配置:"))
		for _, profile := range profiles {
			fmt.Printf("  %s %s@%s:%s\n", ui.Info(profile.Name), ui.Info(profile.Runtime.Server.User), ui.Accent(profile.Runtime.Server.Host), ui.Accent(fmt.Sprint(profile.Runtime.Server.Port)))
			for _, tunnelName := range sortedTunnelNames(profile.Runtime.Tunnels) {
				tunnelCfg := profile.Runtime.Tunnels[tunnelName]
				fmt.Printf("    %s 远程 %s:%s → 本地 %s\n", ui.Info(tunnelName), ui.Accent(tunnelCfg.RemoteBind), ui.Accent(fmt.Sprint(tunnelCfg.RemotePort)), ui.Accent(fmt.Sprintf(":%d", tunnelCfg.LocalPort)))
			}
		}
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

	return runProfiles(ctx, profiles, logger)
}

func checkLocalProxyPorts(cfg *config.RuntimeConfig) error {
	for _, tunnelName := range sortedTunnelNames(cfg.Tunnels) {
		tunnelCfg := cfg.Tunnels[tunnelName]
		localAddr := fmt.Sprintf("127.0.0.1:%d", tunnelCfg.LocalPort)
		conn, err := net.DialTimeout("tcp", localAddr, 2*time.Second)
		if err == nil {
			conn.Close()
			continue
		}

		return fmt.Errorf(
			"%s: %s (%s)\n\n"+
				"请先确认本地代理客户端已启动，并且 HTTP 或 mixed 代理端口是 %d。\n"+
				"常见检查:\n"+
				"  1. Clash/Mihomo/Surge/V2Ray 是否正在运行\n"+
				"  2. 配置里的 local_port 是否写对\n"+
				"  3. 本机是否允许连接 127.0.0.1:%d\n\n"+
				"原始错误: %v",
			console.ForStderr().Error("本地代理端口不可连接"),
			console.ForStderr().Accent(localAddr),
			tunnelName,
			tunnelCfg.LocalPort,
			tunnelCfg.LocalPort,
			err,
		)
	}
	return nil
}

func explainStartError(err error, cfg *config.RuntimeConfig) error {
	ui := console.ForStderr()
	if errors.Is(err, tunnel.ErrRemoteListenFailed) {
		return fmt.Errorf(
			"%w\n\n"+
				"%s 监听失败，隧道没有建立成功。\n"+
				"建议按顺序检查:\n"+
				"  1. 先运行 `localtun setup`，确保远端开启 AllowTcpForwarding 和 GatewayPorts\n"+
				"  2. 登录远端执行 `ss -lntp`，确认 remote_port 没有被占用\n"+
				"  3. 如果云厂商有安全组/防火墙，请放行对应 remote_port\n"+
				"  4. 如果刚改过 sshd_config，请重启 sshd 或重新连接后再试",
			err,
			ui.Error("远程端口"),
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

func runProfiles(ctx context.Context, profiles []selectedProfile, logger *log.Logger) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, len(profiles))
	var wg sync.WaitGroup

	for _, profile := range profiles {
		profile := profile
		wg.Add(1)
		go func() {
			defer wg.Done()
			profileLogger := logger
			if len(profiles) > 1 {
				profileLogger = log.New(logger.Writer(), fmt.Sprintf("[%s] ", profile.Name), logger.Flags())
			}
			t := tunnel.New(profile.Runtime, profileLogger)
			if err := t.Run(runCtx); err != nil {
				errCh <- fmt.Errorf("[%s] %w", profile.Name, explainStartError(err, profile.Runtime))
				cancel()
			}
		}()
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func foregroundServerName(cmd *cobra.Command) (string, bool) {
	values, err := cmd.Flags().GetStringArray("server")
	if err != nil || len(values) != 1 {
		return "", false
	}
	return values[0], true
}

func daemonizeProfiles(profiles []selectedProfile) error {
	for _, profile := range profiles {
		if err := daemonize(profile.Name); err != nil {
			return err
		}
	}
	return nil
}

func daemonize(serverName string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %w", err)
	}

	args := []string{exe, "start", "--daemon", "--foreground", "--server", serverName}
	if cfgFile != "" {
		args = append(args, "--config", cfgFile)
	}

	dataDir, _ := config.DataDir()
	if err := os.MkdirAll(profileRunDir(dataDir), 0755); err != nil {
		return fmt.Errorf("创建 PID 目录失败: %w", err)
	}
	if err := os.MkdirAll(profileLogDir(dataDir), 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}
	pidFile := profilePIDFile(dataDir, serverName)
	logPath := profileLogFile(dataDir, serverName)
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
	fmt.Printf("%s %s 隧道已在后台启动 (PID: %s)\n", ui.SuccessMark(), ui.Info(serverName), ui.Accent(fmt.Sprint(proc.Pid)))
	fmt.Printf("%s %s\n", ui.Label("日志文件:"), ui.Accent(logPath))
	fmt.Printf("%s %s\n", ui.Label("停止隧道:"), ui.Info(fmt.Sprintf("localtun stop --server %s", serverName)))
	return nil
}
