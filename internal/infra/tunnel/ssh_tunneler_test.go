package tunnel_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	domaintunnel "github.com/kdjun99/lazy-dbx/internal/domain/tunnel"
	infratunnel "github.com/kdjun99/lazy-dbx/internal/infra/tunnel"
)

func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func writePrivateKeyFile(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	pemBlock, err := ssh.MarshalPrivateKey(key, "")
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(pemBlock)
	path := filepath.Join(t.TempDir(), "id_rsa")
	require.NoError(t, os.WriteFile(path, pemBytes, 0o600))
	return path
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	require.NoError(t, err)
	return n
}

func copyRW(dst io.Writer, src io.Reader, done chan<- struct{}) {
	_, _ = io.Copy(dst, src)
	done <- struct{}{}
}

func handleSSHConn(conn net.Conn, config *ssh.ServerConfig, targetAddr string) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)
	for newChan := range chans {
		if newChan.ChannelType() != "direct-tcpip" {
			_ = newChan.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}
		ch, reqs2, err := newChan.Accept()
		if err != nil {
			return
		}
		go ssh.DiscardRequests(reqs2)
		go func() {
			defer ch.Close()
			target, err := net.Dial("tcp", targetAddr)
			if err != nil {
				return
			}
			defer target.Close()
			done := make(chan struct{}, 2)
			go copyRW(target, ch, done)
			go copyRW(ch, target, done)
			<-done
		}()
	}
}

func serveSSH(ln net.Listener, config *ssh.ServerConfig, targetAddr string) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go handleSSHConn(conn, config, targetAddr)
	}
}

// startFakeSSHServer starts a minimal in-process SSH server on a random port.
// Returns the SSH server address; cleans up via t.Cleanup.
func startFakeSSHServer(t *testing.T, authorizedKey ssh.PublicKey) string {
	t.Helper()

	hostKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	hostSigner, err := ssh.NewSignerFromKey(hostKey)
	require.NoError(t, err)

	srvConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if authorizedKey != nil && ssh.FingerprintSHA256(key) == ssh.FingerprintSHA256(authorizedKey) {
				return &ssh.Permissions{}, nil
			}
			return nil, assert.AnError
		},
	}
	srvConfig.AddHostKey(hostSigner)

	// Echo target server.
	target, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() {
		for {
			c, err := target.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				buf := make([]byte, 256)
				n, _ := c.Read(buf)
				_, _ = c.Write(buf[:n])
			}()
		}
	}()

	sshLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	go serveSSH(sshLn, srvConfig, target.Addr().String())

	t.Cleanup(func() {
		sshLn.Close()
		target.Close()
	})

	return sshLn.Addr().String()
}

func TestSSHTunneler_KeyFileAuth(t *testing.T) {
	clientKey := generateRSAKey(t)
	pub, err := ssh.NewPublicKey(&clientKey.PublicKey)
	require.NoError(t, err)

	sshAddr := startFakeSSHServer(t, pub)
	host, portStr, err := net.SplitHostPort(sshAddr)
	require.NoError(t, err)
	port := mustAtoi(t, portStr)

	keyPath := writePrivateKeyFile(t, clientKey)

	tunneler := infratunnel.NewSSHTunneler()
	cfg := domaintunnel.Config{
		SSHHost:  host,
		SSHPort:  port,
		SSHUser:  "testuser",
		KeyPath:  keyPath,
		UseAgent: false,
	}

	state, err := tunneler.Open(context.Background(), cfg, "127.0.0.1", 9999)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, domaintunnel.StatusOpen, state.Status)
	assert.Equal(t, "127.0.0.1", state.LocalHost)
	assert.Greater(t, state.LocalPort, 0)
	assert.Equal(t, "127.0.0.1", state.RemoteHost)
	assert.Equal(t, 9999, state.RemotePort)

	require.NoError(t, tunneler.Close())
}

func TestSSHTunneler_InvalidKey_DescriptiveError(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "bad.key")
	require.NoError(t, os.WriteFile(keyPath, []byte("not a valid key"), 0o600))

	tunneler := infratunnel.NewSSHTunneler()
	cfg := domaintunnel.Config{
		SSHHost:  "127.0.0.1",
		SSHPort:  22,
		SSHUser:  "testuser",
		KeyPath:  keyPath,
		UseAgent: false,
	}

	_, err := tunneler.Open(context.Background(), cfg, "127.0.0.1", 9999)
	require.Error(t, err)
	assert.ErrorIs(t, err, domaintunnel.ErrKeyParse)
}

func TestSSHTunneler_ConnectionRefused_DescriptiveError(t *testing.T) {
	// Find a free port then do NOT listen on it so connection is refused.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	ln.Close()

	_, portStr, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	port := mustAtoi(t, portStr)

	clientKey := generateRSAKey(t)
	keyPath := writePrivateKeyFile(t, clientKey)

	tunneler := infratunnel.NewSSHTunneler()
	cfg := domaintunnel.Config{
		SSHHost:  "127.0.0.1",
		SSHPort:  port,
		SSHUser:  "testuser",
		KeyPath:  keyPath,
		UseAgent: false,
	}

	_, err = tunneler.Open(context.Background(), cfg, "127.0.0.1", 9999)
	require.Error(t, err)
	assert.ErrorIs(t, err, domaintunnel.ErrTunnelFailed)
}

func TestSSHTunneler_OpenCloseCycle(t *testing.T) {
	clientKey := generateRSAKey(t)
	pub, err := ssh.NewPublicKey(&clientKey.PublicKey)
	require.NoError(t, err)

	sshAddr := startFakeSSHServer(t, pub)
	host, portStr, _ := net.SplitHostPort(sshAddr)
	port := mustAtoi(t, portStr)
	keyPath := writePrivateKeyFile(t, clientKey)

	tunneler := infratunnel.NewSSHTunneler()
	cfg := domaintunnel.Config{
		SSHHost: host, SSHPort: port, SSHUser: "testuser", KeyPath: keyPath,
	}

	state, err := tunneler.Open(context.Background(), cfg, "127.0.0.1", 9999)
	require.NoError(t, err)
	assert.Equal(t, domaintunnel.StatusOpen, state.Status)

	require.NoError(t, tunneler.Close())

	// Second Close should not panic or return error.
	err = tunneler.Close()
	assert.NoError(t, err)
}

func TestSSHTunneler_ContextCancellation(t *testing.T) {
	clientKey := generateRSAKey(t)
	keyPath := writePrivateKeyFile(t, clientKey)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	tunneler := infratunnel.NewSSHTunneler()
	cfg := domaintunnel.Config{
		SSHHost:  "192.0.2.1", // TEST-NET, not routable
		SSHPort:  22,
		SSHUser:  "testuser",
		KeyPath:  keyPath,
		UseAgent: false,
	}

	_, err := tunneler.Open(ctx, cfg, "127.0.0.1", 9999)
	require.Error(t, err)
}

func TestSSHTunneler_NoAgentSock(t *testing.T) {
	t.Setenv("SSH_AUTH_SOCK", "")
	os.Unsetenv("SSH_AUTH_SOCK") //nolint:errcheck

	tunneler := infratunnel.NewSSHTunneler()
	cfg := domaintunnel.Config{
		SSHHost:  "127.0.0.1",
		SSHPort:  22,
		SSHUser:  "testuser",
		UseAgent: true,
	}

	_, err := tunneler.Open(context.Background(), cfg, "127.0.0.1", 9999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SSH_AUTH_SOCK not set")
}

func TestSSHTunneler_NoKeyPath(t *testing.T) {
	tunneler := infratunnel.NewSSHTunneler()
	cfg := domaintunnel.Config{
		SSHHost:  "127.0.0.1",
		SSHPort:  22,
		SSHUser:  "testuser",
		KeyPath:  "",
		UseAgent: false,
	}

	_, err := tunneler.Open(context.Background(), cfg, "127.0.0.1", 9999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no key_path")
}

func TestSSHTunneler_KeyFileNotFound(t *testing.T) {
	tunneler := infratunnel.NewSSHTunneler()
	cfg := domaintunnel.Config{
		SSHHost:  "127.0.0.1",
		SSHPort:  22,
		SSHUser:  "testuser",
		KeyPath:  "/nonexistent/key",
		UseAgent: false,
	}

	_, err := tunneler.Open(context.Background(), cfg, "127.0.0.1", 9999)
	require.Error(t, err)
	assert.ErrorIs(t, err, domaintunnel.ErrTunnelFailed)
}

// --- TunnelManager tests ---

func TestTunnelManager_SharedTunnel_Refcount(t *testing.T) {
	clientKey := generateRSAKey(t)
	pub, err := ssh.NewPublicKey(&clientKey.PublicKey)
	require.NoError(t, err)

	sshAddr := startFakeSSHServer(t, pub)
	host, portStr, _ := net.SplitHostPort(sshAddr)
	port := mustAtoi(t, portStr)
	keyPath := writePrivateKeyFile(t, clientKey)

	mgr := infratunnel.NewTunnelManager()
	cfg := domaintunnel.Config{
		SSHHost: host, SSHPort: port, SSHUser: "testuser", KeyPath: keyPath,
	}

	state1, err := mgr.GetOrOpen(context.Background(), "tunnel-a", cfg, "127.0.0.1", 9999)
	require.NoError(t, err)

	state2, err := mgr.GetOrOpen(context.Background(), "tunnel-a", cfg, "127.0.0.1", 9999)
	require.NoError(t, err)

	// Same tunnel reused: same local port.
	assert.Equal(t, state1.LocalPort, state2.LocalPort)

	// Release once — refcount goes 2→1, tunnel still open.
	require.NoError(t, mgr.Release("tunnel-a"))

	// Release again — refcount goes 1→0, tunnel closed.
	require.NoError(t, mgr.Release("tunnel-a"))
}

func TestTunnelManager_ReleaseNonexistent(t *testing.T) {
	mgr := infratunnel.NewTunnelManager()
	err := mgr.Release("nonexistent")
	assert.NoError(t, err)
}

func TestTunnelManager_ConcurrentAccess(t *testing.T) {
	clientKey := generateRSAKey(t)
	pub, err := ssh.NewPublicKey(&clientKey.PublicKey)
	require.NoError(t, err)

	sshAddr := startFakeSSHServer(t, pub)
	host, portStr, _ := net.SplitHostPort(sshAddr)
	port := mustAtoi(t, portStr)
	keyPath := writePrivateKeyFile(t, clientKey)

	mgr := infratunnel.NewTunnelManager()
	cfg := domaintunnel.Config{
		SSHHost: host, SSHPort: port, SSHUser: "testuser", KeyPath: keyPath,
	}

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, _ = mgr.GetOrOpen(context.Background(), "concurrent-tunnel", cfg, "127.0.0.1", 9999)
			_ = mgr.Release("concurrent-tunnel")
		}()
	}
	wg.Wait()
}

func TestTunnelManager_CloseAll(t *testing.T) {
	clientKey := generateRSAKey(t)
	pub, err := ssh.NewPublicKey(&clientKey.PublicKey)
	require.NoError(t, err)

	sshAddr := startFakeSSHServer(t, pub)
	host, portStr, _ := net.SplitHostPort(sshAddr)
	port := mustAtoi(t, portStr)
	keyPath := writePrivateKeyFile(t, clientKey)

	mgr := infratunnel.NewTunnelManager()
	cfg := domaintunnel.Config{
		SSHHost: host, SSHPort: port, SSHUser: "testuser", KeyPath: keyPath,
	}

	_, err = mgr.GetOrOpen(context.Background(), "tunnel-b", cfg, "127.0.0.1", 9999)
	require.NoError(t, err)

	require.NoError(t, mgr.CloseAll())

	// CloseAll is idempotent.
	require.NoError(t, mgr.CloseAll())
}
