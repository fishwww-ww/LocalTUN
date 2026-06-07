package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"localtun/internal/console"
	"localtun/internal/next"
	"localtun/internal/proxy"
	sessionstore "localtun/internal/session"
	"localtun/internal/sshutil"
	"localtun/internal/target"
)

type connectOptions struct {
	port        int
	identity    string
	localProxy  string
	remotePort  int
	shell       string
	detach      bool
	detachedRun bool
	sessionID   string
}

var connectOpts connectOptions

var connectCmd = &cobra.Command{
	Use:   "connect [user@]host[:port]",
	Short: "进入带外网能力的 SSH session",
	Args:  cobra.ExactArgs(1),
	RunE:  runConnect,
}

func init() {
	connectCmd.Flags().IntVarP(&connectOpts.port, "port", "p", 0, "SSH 端口，优先级高于 target 中的 :port")
	connectCmd.Flags().StringVarP(&connectOpts.identity, "identity", "i", "", "SSH 私钥路径，默认自动尝试 ~/.ssh/id_ed25519 和 ~/.ssh/id_rsa")
	connectCmd.Flags().StringVar(&connectOpts.localProxy, "local-proxy", "", "本地代理地址，格式为 127.0.0.1:7897 或 7897")
	connectCmd.Flags().IntVar(&connectOpts.remotePort, "remote-port", 0, "远端临时代理端口，0 表示自动选择")
	connectCmd.Flags().StringVar(&connectOpts.shell, "shell", "", "远端 shell 路径，默认读取远端 $SHELL")
	connectCmd.Flags().BoolVar(&connectOpts.detach, "detach", false, "后台保持 tunnel，不进入交互 shell")
	connectCmd.Flags().BoolVar(&connectOpts.detachedRun, "detached-run", false, "后台 watcher 内部参数")
	connectCmd.Flags().MarkHidden("detached-run")
	connectCmd.Flags().StringVar(&connectOpts.sessionID, "session-id", "", "后台 watcher session id")
	connectCmd.Flags().MarkHidden("session-id")
	rootCmd.AddCommand(connectCmd)
}

func runConnect(cmd *cobra.Command, args []string) error {
	tgt, err := target.Parse(args[0])
	if err != nil {
		return err
	}
	if connectOpts.port != 0 {
		tgt.Port = connectOpts.port
	}
	if connectOpts.detachedRun {
		return runDetachedWatcher(cmd.Context(), args[0], tgt, connectOpts)
	}
	if connectOpts.detach {
		return startDetached(args[0], tgt, connectOpts)
	}
	return runInteractive(cmd.Context(), args[0], tgt, connectOpts)
}

func runInteractive(parent context.Context, rawTarget string, tgt target.Target, opts connectOptions) error {
	ui := console.ForStderr()
	localProxy, err := proxy.Detector{}.Detect(opts.localProxy)
	if err != nil {
		return err
	}
	identity, err := sshutil.ResolveIdentity(opts.identity)
	if err != nil {
		return err
	}
	client, err := sshutil.DialOptions(sshutil.Options{
		User:     tgt.User,
		Host:     tgt.Host,
		Port:     tgt.Port,
		Identity: identity,
	})
	if err != nil {
		return err
	}
	defer client.Close()

	tun, remotePort, err := next.OpenReverseTunnel(client, localProxy, opts.remotePort)
	if err != nil {
		return err
	}
	defer tun.Close()

	ctx, cancel := signalContext(parent)
	defer cancel()

	tunnelErr := make(chan error, 1)
	go func() {
		tunnelErr <- tun.Serve(ctx)
	}()
	tunnelReported := make(chan struct{}, 1)
	go func() {
		err := <-tunnelErr
		if err != nil && !errors.Is(err, net.ErrClosed) {
			fmt.Fprintf(os.Stderr, "%s %s: %v\n", ui.Prefix("LocalTUN"), ui.Error("Tunnel disconnected"), err)
			fmt.Fprintf(os.Stderr, "%s %s\n", ui.Prefix("LocalTUN"), ui.Warning("Interactive SSH sessions cannot be recovered automatically; run `localtun connect` again."))
		}
		tunnelReported <- struct{}{}
	}()

	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", remotePort)
	fmt.Fprintf(os.Stderr, "%s %s %s %s %s\n", ui.Prefix("LocalTUN"), ui.Success("connected"), ui.Info(rawTarget), ui.Muted("via local proxy"), ui.Accent(localProxy))
	fmt.Fprintf(os.Stderr, "%s %s %s\n", ui.Prefix("LocalTUN"), ui.Label("remote session proxy:"), ui.Accent(proxyURL))

	shellPath := opts.shell
	if shellPath == "" {
		shellPath = detectRemoteShell(client)
	}
	if shellPath == "" {
		shellPath = "sh"
	}
	sessionErr := runRemoteShell(ctx, client, shellPath, proxyURL)
	cancel()
	tun.Close()

	select {
	case <-tunnelReported:
	default:
	}
	return sessionErr
}

func startDetached(rawTarget string, tgt target.Target, opts connectOptions) error {
	ui := console.ForStdout()
	localProxy, err := proxy.Detector{}.Detect(opts.localProxy)
	if err != nil {
		return err
	}
	identity, err := sshutil.ResolveIdentity(opts.identity)
	if err != nil {
		return err
	}
	store, err := sessionstore.DefaultStore()
	if err != nil {
		return err
	}
	id := opts.sessionID
	if id == "" {
		id = newSessionID(tgt.Host)
	}
	if err := os.MkdirAll(store.Dir(), 0755); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	logPath := filepath.Join(store.Dir(), id+".log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer logFile.Close()

	childArgs := []string{
		"connect", rawTarget,
		"--detached-run",
		"--session-id", id,
		"--identity", identity,
		"--local-proxy", localProxy,
	}
	if tgt.Port != target.DefaultPort {
		childArgs = append(childArgs, "--port", strconv.Itoa(tgt.Port))
	}
	if opts.remotePort != 0 {
		childArgs = append(childArgs, "--remote-port", strconv.Itoa(opts.remotePort))
	}
	child := exec.Command(exe, childArgs...)
	child.Stdout = logFile
	child.Stderr = logFile
	child.SysProcAttr = daemonSysProcAttr()
	if err := child.Start(); err != nil {
		return err
	}

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		meta, err := store.Load(id)
		if err == nil && meta.PID == child.Process.Pid {
			fmt.Printf("%s %s %s\n", ui.SuccessMark(), ui.Label("LocalTUN detached session:"), ui.Info(meta.ID))
			fmt.Printf("%s %s %s\n", ui.Label("Remote proxy:"), ui.Accent(meta.ProxyURL), ui.Muted(fmt.Sprintf("(local %s)", meta.LocalProxy)))
			fmt.Println(ui.Muted("Use these variables in the remote shell that should use this tunnel:"))
			printProxyExports(meta.ProxyURL)
			fmt.Printf("%s %s\n", ui.Label("Stop it with:"), ui.Info(fmt.Sprintf("localtun disconnect %s", meta.ID)))
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	_ = child.Process.Kill()
	return fmt.Errorf("后台 tunnel 未能在 8 秒内启动，请查看日志: %s", logPath)
}

func runDetachedWatcher(parent context.Context, rawTarget string, tgt target.Target, opts connectOptions) error {
	ui := console.ForStderr()
	if opts.sessionID == "" {
		return fmt.Errorf("后台 watcher 缺少 session id")
	}
	localProxy, err := proxy.Detector{}.Detect(opts.localProxy)
	if err != nil {
		return err
	}
	identity, err := sshutil.ResolveIdentity(opts.identity)
	if err != nil {
		return err
	}
	store, err := sessionstore.DefaultStore()
	if err != nil {
		return err
	}

	backoff := time.Second
	for {
		select {
		case <-parent.Done():
			return nil
		default:
		}

		client, err := sshutil.DialOptions(sshutil.Options{
			User:     tgt.User,
			Host:     tgt.Host,
			Port:     tgt.Port,
			Identity: identity,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s %s: %v\n", ui.Prefix("LocalTUN"), ui.Warning("SSH connect failed"), err)
			sleepBackoff(parent, &backoff)
			continue
		}

		tun, remotePort, err := next.OpenReverseTunnel(client, localProxy, opts.remotePort)
		if err != nil {
			_ = client.Close()
			return err
		}
		proxyURL := fmt.Sprintf("http://127.0.0.1:%d", remotePort)
		if err := store.Save(sessionstore.Metadata{
			ID:         opts.sessionID,
			Target:     rawTarget,
			User:       tgt.User,
			Host:       tgt.Host,
			SSHPort:    tgt.Port,
			Identity:   identity,
			LocalProxy: localProxy,
			RemotePort: remotePort,
			PID:        os.Getpid(),
			CreatedAt:  time.Now().UTC(),
			ProxyURL:   proxyURL,
		}); err != nil {
			tun.Close()
			_ = client.Close()
			return err
		}
		fmt.Fprintf(os.Stderr, "%s %s %s\n", ui.Prefix("LocalTUN"), ui.Success("Tunnel connected:"), ui.Accent(proxyURL))

		err = tun.Serve(parent)
		tun.Close()
		_ = client.Close()
		if parent.Err() != nil {
			return nil
		}
		fmt.Fprintf(os.Stderr, "%s %s: %v\n", ui.Prefix("LocalTUN"), ui.Warning("Tunnel disconnected, retrying"), err)
		sleepBackoff(parent, &backoff)
	}
}

func runRemoteShell(ctx context.Context, client *ssh.Client, shellPath, proxyURL string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	if terminal.IsTerminal(int(os.Stdin.Fd())) {
		width, height, err := terminal.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			width, height = 120, 40
		}
		modes := ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}
		if err := session.RequestPty("xterm-256color", height, width, modes); err != nil {
			return fmt.Errorf("请求远端 PTY 失败: %w", err)
		}
		oldState, err := terminal.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			defer terminal.Restore(int(os.Stdin.Fd()), oldState)
		}
	}

	done := make(chan error, 1)
	go func() {
		done <- session.Run(next.ShellCommand(shellPath, proxyURL))
	}()
	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGTERM)
		return ctx.Err()
	case err := <-done:
		if err == nil || errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
}

func detectRemoteShell(client *ssh.Client) string {
	session, err := client.NewSession()
	if err != nil {
		return ""
	}
	defer session.Close()
	out, err := session.Output(`printf %s "$SHELL"`)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func signalContext(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-ch:
			cancel()
		case <-ctx.Done():
		}
		signal.Stop(ch)
	}()
	return ctx, cancel
}

func sleepBackoff(ctx context.Context, backoff *time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(*backoff):
		if *backoff < 30*time.Second {
			*backoff *= 2
		}
	}
}

func printProxyExports(proxyURL string) {
	for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		fmt.Printf("export %s=%s\n", key, proxyURL)
	}
}

func newSessionID(host string) string {
	var buf [4]byte
	_, _ = rand.Read(buf[:])
	cleanHost := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "@", "-").Replace(host)
	if cleanHost == "" {
		cleanHost = "session"
	}
	return fmt.Sprintf("%s-%d-%s", cleanHost, time.Now().Unix(), hex.EncodeToString(buf[:]))
}
