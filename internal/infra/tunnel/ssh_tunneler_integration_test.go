//go:build integration

package tunnel_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/crypto/ssh"

	domaintunnel "github.com/kdjun99/lazy-dbx/internal/domain/tunnel"
	infratunnel "github.com/kdjun99/lazy-dbx/internal/infra/tunnel"
)

// generateTestKey generates a 2048-bit RSA key pair.
// Returns the path to the PEM-encoded private key file and the authorized_keys-format public key string.
func generateTestKey(t *testing.T) (privateKeyPath string, publicKeyString string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Private key → PEM file
	pemBlock, err := ssh.MarshalPrivateKey(key, "")
	require.NoError(t, err)
	pemBytes := pem.EncodeToMemory(pemBlock)
	path := filepath.Join(t.TempDir(), "test_key")
	require.NoError(t, os.WriteFile(path, pemBytes, 0o600))

	// Public key → authorized_keys format
	pub, err := ssh.NewPublicKey(&key.PublicKey)
	require.NoError(t, err)
	pubStr := string(ssh.MarshalAuthorizedKey(pub))

	return path, pubStr
}

// startOpenSSH starts a linuxserver/openssh-server container and returns the SSH host, port, and private key path.
func startOpenSSH(t *testing.T) (sshHost string, sshPort int, keyPath string) {
	t.Helper()
	ctx := context.Background()

	keyPath, pubKeyStr := generateTestKey(t)

	req := testcontainers.ContainerRequest{
		Image:        "linuxserver/openssh-server:latest",
		ExposedPorts: []string{"2222/tcp"},
		Env: map[string]string{
			"PUID":            "1000",
			"PGID":            "1000",
			"USER_NAME":       "testuser",
			"PUBLIC_KEY":      pubKeyStr,
			"SUDO_ACCESS":     "false",
			"PASSWORD_ACCESS": "false",
		},
		WaitingFor: wait.ForListeningPort("2222/tcp").
			WithStartupTimeout(120 * time.Second).
			WithPollInterval(2 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = container.Terminate(ctx) //nolint:errcheck
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)

	mappedPort, err := container.MappedPort(ctx, "2222")
	require.NoError(t, err)

	portStr := mappedPort.Port()
	// mappedPort.Port() may include protocol suffix like "2222/tcp"; strip it.
	if idx := len(portStr); idx > 0 {
		if n, parseErr := strconv.Atoi(portStr); parseErr == nil {
			return host, n, keyPath
		}
		// Strip trailing "/tcp" or similar
		for i, c := range portStr {
			if c == '/' {
				portStr = portStr[:i]
				break
			}
		}
	}

	port, err := strconv.Atoi(portStr)
	require.NoError(t, err, "parsing mapped port %q", mappedPort.Port())

	return host, port, keyPath
}

// TestSSHTunneler_RealSSH_Integration opens an SSH tunnel to a real OpenSSH container
// and verifies:
//   - SSH authentication with an RSA key pair succeeds against a real sshd
//   - The returned State has the correct fields and StatusOpen
//   - The local listener port is actually accepting TCP connections
//   - Close transitions the state to StatusClosed
//
// Data-forwarding through the tunnel is covered by unit tests; here we focus on
// real SSH authentication and lifecycle correctness.
func TestSSHTunneler_RealSSH_Integration(t *testing.T) {
	sshHost, sshPort, keyPath := startOpenSSH(t)

	// Wait for the SSH daemon to be fully ready — the container TCP port may open
	// before sshd has completed initialization and is ready to serve connections.
	waitForSSHBanner(t, sshHost, sshPort)

	tunneler := infratunnel.NewSSHTunneler()
	cfg := domaintunnel.Config{
		SSHHost:  sshHost,
		SSHPort:  sshPort,
		SSHUser:  "testuser",
		KeyPath:  keyPath,
		UseAgent: false,
	}

	// Open tunnel forwarding to an arbitrary remote endpoint.
	// The important assertion is that SSH auth succeeds and the local listener opens.
	state, err := tunneler.Open(context.Background(), cfg, "127.0.0.1", 2222)
	require.NoError(t, err)
	require.NotNil(t, state)

	assert.Equal(t, domaintunnel.StatusOpen, state.Status)
	assert.Equal(t, "127.0.0.1", state.LocalHost)
	assert.Greater(t, state.LocalPort, 0)
	assert.Equal(t, "127.0.0.1", state.RemoteHost)
	assert.Equal(t, 2222, state.RemotePort)

	// The local listener must accept TCP connections — this proves the port-forward
	// goroutine is running and the listener is bound.
	localAddr := fmt.Sprintf("127.0.0.1:%d", state.LocalPort)
	conn, dialErr := net.DialTimeout("tcp", localAddr, 5*time.Second)
	require.NoError(t, dialErr, "local tunnel listener must accept connections at %s", localAddr)
	conn.Close() //nolint:errcheck

	require.NoError(t, tunneler.Close())
	assert.Equal(t, domaintunnel.StatusClosed, state.Status)
}

// waitForSSHBanner polls the SSH host:port until a valid SSH banner is received or the
// deadline is exceeded. The linuxserver/openssh-server image needs a few seconds after
// the port starts accepting TCP connections before sshd is fully initialized.
func waitForSSHBanner(t *testing.T, host string, port int) {
	t.Helper()
	addr := fmt.Sprintf("%s:%d", host, port)
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		buf := make([]byte, 4)
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, readErr := conn.Read(buf)
		conn.Close() //nolint:errcheck
		if readErr == nil && n == 4 && string(buf) == "SSH-" {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatal("SSH daemon did not become ready within 60s")
}

// TestTunnelManager_RealSSH_SharedTunnel_Integration verifies that GetOrOpen returns
// the same local port for the same tunnel name, and that releasing twice closes cleanly.
func TestTunnelManager_RealSSH_SharedTunnel_Integration(t *testing.T) {
	sshHost, sshPort, keyPath := startOpenSSH(t)
	waitForSSHBanner(t, sshHost, sshPort)

	mgr := infratunnel.NewTunnelManager()
	cfg := domaintunnel.Config{
		SSHHost:  sshHost,
		SSHPort:  sshPort,
		SSHUser:  "testuser",
		KeyPath:  keyPath,
		UseAgent: false,
	}

	ctx := context.Background()

	state1, err := mgr.GetOrOpen(ctx, "integration-tunnel", cfg, "127.0.0.1", 2222)
	require.NoError(t, err)
	require.NotNil(t, state1)
	assert.Equal(t, domaintunnel.StatusOpen, state1.Status)
	assert.Greater(t, state1.LocalPort, 0)

	state2, err := mgr.GetOrOpen(ctx, "integration-tunnel", cfg, "127.0.0.1", 2222)
	require.NoError(t, err)
	require.NotNil(t, state2)

	// Same tunnel entry reused: same local port.
	assert.Equal(t, state1.LocalPort, state2.LocalPort)

	// Release once — refcount 2→1, tunnel still open.
	require.NoError(t, mgr.Release("integration-tunnel"))
	assert.Equal(t, domaintunnel.StatusOpen, state1.Status)

	// Release again — refcount 1→0, tunnel closed.
	require.NoError(t, mgr.Release("integration-tunnel"))
	assert.Equal(t, domaintunnel.StatusClosed, state1.Status)
}
