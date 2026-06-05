package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"localtun/internal/config"
	"localtun/internal/sshutil"
)

var ErrRemoteListenFailed = errors.New("remote listen failed")

type Tunnel struct {
	cfg       *config.RuntimeConfig
	logger    *log.Logger
	client    *ssh.Client
	listeners []net.Listener
	mu        sync.Mutex
}

func New(cfg *config.RuntimeConfig, logger *log.Logger) *Tunnel {
	return &Tunnel{cfg: cfg, logger: logger}
}

func (t *Tunnel) connect() error {
	addr := fmt.Sprintf("%s:%d", t.cfg.Server.Host, t.cfg.Server.Port)
	t.logger.Printf("正在连接 %s@%s ...", t.cfg.Server.User, addr)

	client, err := sshutil.Dial(t.cfg)
	if err != nil {
		return err
	}

	t.mu.Lock()
	t.client = client
	t.mu.Unlock()

	t.logger.Printf("SSH 连接成功")
	return nil
}

func (t *Tunnel) setupKeepalive(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(t.cfg.Keepalive.Interval) * time.Second)
	defer ticker.Stop()

	failures := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.mu.Lock()
			client := t.client
			t.mu.Unlock()

			if client == nil {
				return
			}

			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil {
				failures++
				t.logger.Printf("Keepalive 失败 (%d/%d)", failures, t.cfg.Keepalive.MaxCount)
				if failures >= t.cfg.Keepalive.MaxCount {
					t.logger.Printf("Keepalive 超过最大失败次数，断开连接")
					client.Close()
					return
				}
			} else {
				failures = 0
			}
		}
	}
}

func (t *Tunnel) startForwarding(ctx context.Context) error {
	t.mu.Lock()
	client := t.client
	t.mu.Unlock()

	errCh := make(chan error, len(t.cfg.Tunnels))
	var wg sync.WaitGroup

	for name, tunnelCfg := range t.cfg.Tunnels {
		name := name
		tunnelCfg := tunnelCfg
		listenAddr := fmt.Sprintf("%s:%d", tunnelCfg.RemoteBind, tunnelCfg.RemotePort)

		listener, err := client.Listen("tcp", listenAddr)
		if err != nil {
			t.Close()
			return fmt.Errorf("%w: 隧道 %s 远程 %s 监听失败: %v", ErrRemoteListenFailed, name, listenAddr, err)
		}

		t.mu.Lock()
		t.listeners = append(t.listeners, listener)
		t.mu.Unlock()

		t.logger.Printf("隧道已建立 [%s]: 远程 %s → 本地 :%d", name, listenAddr, tunnelCfg.LocalPort)

		wg.Add(1)
		go func() {
			defer wg.Done()
			t.acceptLoop(ctx, name, tunnelCfg, listener, errCh)
		}()
	}

	go func() {
		<-ctx.Done()
		t.Close()
	}()

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Tunnel) acceptLoop(ctx context.Context, name string, tunnelCfg config.TunnelConfig, listener net.Listener, errCh chan<- error) {
	for {
		remote, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				t.Close()
				errCh <- fmt.Errorf("隧道 %s 接受连接失败: %w", name, err)
				return
			}
		}
		go t.handleConnection(name, tunnelCfg, remote)
	}
}

func (t *Tunnel) handleConnection(name string, tunnelCfg config.TunnelConfig, remote net.Conn) {
	defer remote.Close()

	localAddr := fmt.Sprintf("127.0.0.1:%d", tunnelCfg.LocalPort)
	local, err := net.DialTimeout("tcp", localAddr, 5*time.Second)
	if err != nil {
		t.logger.Printf("隧道 %s 连接本地端口 %d 失败: %v", name, tunnelCfg.LocalPort, err)
		return
	}
	defer local.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if _, err := io.Copy(local, remote); err != nil {
			t.logger.Printf("远程到本地转发结束: %v", err)
		}
		closeWrite(local)
	}()
	go func() {
		defer wg.Done()
		if _, err := io.Copy(remote, local); err != nil {
			t.logger.Printf("本地到远程转发结束: %v", err)
		}
		closeWrite(remote)
	}()

	wg.Wait()
}

func closeWrite(conn net.Conn) {
	if c, ok := conn.(interface{ CloseWrite() error }); ok {
		_ = c.CloseWrite()
		return
	}
	_ = conn.Close()
}

func (t *Tunnel) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, listener := range t.listeners {
		listener.Close()
	}
	t.listeners = nil
	if t.client != nil {
		t.client.Close()
		t.client = nil
	}
}

// Run starts the tunnel with auto-reconnect. Blocks until ctx is cancelled.
func (t *Tunnel) Run(ctx context.Context) error {
	backoff := time.Second
	maxBackoff := 60 * time.Second

	for {
		select {
		case <-ctx.Done():
			t.Close()
			return nil
		default:
		}

		if err := t.connect(); err != nil {
			t.logger.Printf("连接失败: %v", err)
			t.logger.Printf("将在 %v 后重试...", backoff)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(backoff):
			}
			backoff = min(backoff*2, maxBackoff)
			continue
		}

		backoff = time.Second

		kaCtx, kaCancel := context.WithCancel(ctx)
		go t.setupKeepalive(kaCtx)

		err := t.startForwarding(ctx)
		kaCancel()
		t.Close()

		if ctx.Err() != nil {
			return nil
		}

		if err != nil {
			t.logger.Printf("隧道断开: %v", err)
			if errors.Is(err, ErrRemoteListenFailed) {
				return err
			}
		} else {
			t.logger.Printf("隧道断开")
		}
		t.logger.Printf("将在 %v 后重连...", backoff)

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, maxBackoff)
	}
}
