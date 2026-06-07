package next

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type ReverseTunnel struct {
	client     *ssh.Client
	localProxy string
	listener   net.Listener
	mu         sync.Mutex
}

func OpenReverseTunnel(client *ssh.Client, localProxy string, preferredRemotePort int) (*ReverseTunnel, int, error) {
	if preferredRemotePort < 0 || preferredRemotePort > 65535 {
		return nil, 0, fmt.Errorf("远端端口必须在 0-65535 之间")
	}
	listenAddr := fmt.Sprintf("127.0.0.1:%d", preferredRemotePort)
	listener, err := client.Listen("tcp", listenAddr)
	if err != nil {
		return nil, 0, fmt.Errorf("远端监听 %s 失败，请确认 SSH 服务允许 AllowTcpForwarding: %w", listenAddr, err)
	}
	port, err := listenerPort(listener.Addr())
	if err != nil {
		_ = listener.Close()
		return nil, 0, err
	}
	return &ReverseTunnel{
		client:     client,
		localProxy: localProxy,
		listener:   listener,
	}, port, nil
}

func (t *ReverseTunnel) Serve(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		<-ctx.Done()
		t.Close()
	}()
	go func() {
		errCh <- t.acceptLoop(ctx)
	}()

	err := <-errCh
	if ctx.Err() != nil && errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (t *ReverseTunnel) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.listener != nil {
		_ = t.listener.Close()
		t.listener = nil
	}
}

func (t *ReverseTunnel) acceptLoop(ctx context.Context) error {
	for {
		t.mu.Lock()
		listener := t.listener
		t.mu.Unlock()
		if listener == nil {
			return net.ErrClosed
		}
		remote, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return net.ErrClosed
			}
			return fmt.Errorf("隧道断开: %w", err)
		}
		go forwardConn(remote, t.localProxy)
	}
}

func forwardConn(remote net.Conn, localProxy string) {
	defer remote.Close()
	local, err := net.DialTimeout("tcp", localProxy, 5*time.Second)
	if err != nil {
		return
	}
	defer local.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(local, remote)
		closeWrite(local)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(remote, local)
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

func listenerPort(addr net.Addr) (int, error) {
	_, rawPort, err := net.SplitHostPort(addr.String())
	if err != nil {
		return 0, fmt.Errorf("无法识别远端监听地址 %s: %w", addr, err)
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil || port <= 0 || port > 65535 {
		return 0, fmt.Errorf("远端监听端口无效: %s", rawPort)
	}
	return port, nil
}
