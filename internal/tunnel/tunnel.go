package tunnel

import (
	"context"
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

type Tunnel struct {
	cfg      *config.Config
	logger   *log.Logger
	client   *ssh.Client
	listener net.Listener
	mu       sync.Mutex
}

func New(cfg *config.Config, logger *log.Logger) *Tunnel {
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
	listenAddr := fmt.Sprintf("0.0.0.0:%d", t.cfg.Tunnel.RemotePort)

	t.mu.Lock()
	client := t.client
	t.mu.Unlock()

	listener, err := client.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("远程端口 %d 监听失败: %w", t.cfg.Tunnel.RemotePort, err)
	}

	t.mu.Lock()
	t.listener = listener
	t.mu.Unlock()

	t.logger.Printf("隧道已建立: 远程 :%d → 本地 :%d", t.cfg.Tunnel.RemotePort, t.cfg.Tunnel.LocalPort)

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		remote, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return fmt.Errorf("接受连接失败: %w", err)
			}
		}
		go t.handleConnection(remote)
	}
}

func (t *Tunnel) handleConnection(remote net.Conn) {
	defer remote.Close()

	localAddr := fmt.Sprintf("127.0.0.1:%d", t.cfg.Tunnel.LocalPort)
	local, err := net.DialTimeout("tcp", localAddr, 5*time.Second)
	if err != nil {
		t.logger.Printf("连接本地端口 %d 失败: %v", t.cfg.Tunnel.LocalPort, err)
		return
	}
	defer local.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(local, remote)
	}()
	go func() {
		defer wg.Done()
		io.Copy(remote, local)
	}()

	wg.Wait()
}

func (t *Tunnel) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.listener != nil {
		t.listener.Close()
		t.listener = nil
	}
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
