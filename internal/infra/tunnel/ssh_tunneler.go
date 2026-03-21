// Package tunnel provides SSH tunnel implementations for the domain tunnel interfaces.
package tunnel

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	domaintunnel "github.com/kdjun99/lazy-dbx/internal/domain/tunnel"
)

// SSHTunneler opens a single SSH port-forward tunnel.
type SSHTunneler struct {
	mu       sync.Mutex
	client   *ssh.Client
	listener net.Listener
	state    *domaintunnel.State
}

// NewSSHTunneler creates a new SSHTunneler.
func NewSSHTunneler() *SSHTunneler {
	return &SSHTunneler{}
}

// Open establishes the SSH connection and starts a local listener forwarding to remoteHost:remotePort.
func (t *SSHTunneler) Open(ctx context.Context, config domaintunnel.Config, remoteHost string, remotePort int) (*domaintunnel.State, error) {
	authMethods, err := buildAuthMethods(config)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User:            config.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // host key verification out of scope for Wave 1
	}

	addr := fmt.Sprintf("%s:%d", config.SSHHost, config.SSHPort)
	var d net.Dialer
	netConn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("%w: connecting to SSH host %s: %v", domaintunnel.ErrTunnelFailed, addr, err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(netConn, addr, sshConfig)
	if err != nil {
		netConn.Close() //nolint:errcheck
		return nil, fmt.Errorf("%w: SSH handshake with %s: %v", domaintunnel.ErrTunnelFailed, addr, err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		client.Close() //nolint:errcheck
		return nil, fmt.Errorf("%w: opening local listener: %v", domaintunnel.ErrTunnelFailed, err)
	}

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		client.Close() //nolint:errcheck
		ln.Close()     //nolint:errcheck
		return nil, fmt.Errorf("%w: unexpected listener address type", domaintunnel.ErrTunnelFailed)
	}
	localPort := tcpAddr.Port

	t.mu.Lock()
	t.client = client
	t.listener = ln
	t.state = &domaintunnel.State{
		LocalHost:  "127.0.0.1",
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		Status:     domaintunnel.StatusOpen,
	}
	t.mu.Unlock()

	go t.accept(ln, client, remoteHost, remotePort)

	return t.state, nil
}

// Close tears down the tunnel and releases all resources.
func (t *SSHTunneler) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.listener != nil {
		t.listener.Close() //nolint:errcheck
		t.listener = nil
	}
	if t.client != nil {
		err := t.client.Close()
		t.client = nil
		if t.state != nil {
			t.state.Status = domaintunnel.StatusClosed
		}
		return err
	}
	return nil
}

func (t *SSHTunneler) accept(ln net.Listener, client *ssh.Client, remoteHost string, remotePort int) {
	for {
		local, err := ln.Accept()
		if err != nil {
			return
		}
		go forward(local, client, remoteHost, remotePort)
	}
}

func forward(local net.Conn, client *ssh.Client, remoteHost string, remotePort int) {
	defer local.Close() //nolint:errcheck
	remote, err := client.Dial("tcp", fmt.Sprintf("%s:%d", remoteHost, remotePort))
	if err != nil {
		return
	}
	defer remote.Close() //nolint:errcheck

	done := make(chan struct{}, 2)
	go func() { io.Copy(local, remote); done <- struct{}{} }() //nolint:errcheck
	go func() { io.Copy(remote, local); done <- struct{}{} }() //nolint:errcheck
	<-done
}

func expandTilde(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

func buildAuthMethods(config domaintunnel.Config) ([]ssh.AuthMethod, error) {
	if config.UseAgent {
		sock := os.Getenv("SSH_AUTH_SOCK")
		if sock == "" {
			return nil, fmt.Errorf("%w: SSH_AUTH_SOCK not set but use_agent=true", domaintunnel.ErrTunnelFailed)
		}
		conn, err := net.Dial("unix", sock)
		if err != nil {
			return nil, fmt.Errorf("%w: connecting to SSH agent: %v", domaintunnel.ErrTunnelFailed, err)
		}
		ag := agent.NewClient(conn)
		return []ssh.AuthMethod{ssh.PublicKeysCallback(ag.Signers)}, nil
	}

	if config.KeyPath == "" {
		return nil, fmt.Errorf("%w: no key_path and use_agent=false", domaintunnel.ErrTunnelFailed)
	}

	keyData, err := os.ReadFile(expandTilde(config.KeyPath))
	if err != nil {
		return nil, fmt.Errorf("%w: reading key file %s: %v", domaintunnel.ErrTunnelFailed, config.KeyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("%w: parsing key file %s: %v", domaintunnel.ErrKeyParse, config.KeyPath, err)
	}

	return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
}
