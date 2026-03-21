// Package tunnel defines domain interfaces and models for SSH tunnel management.
package tunnel

// Status represents the current state of a tunnel.
type Status string

const (
	// StatusOpen indicates the tunnel is active and forwarding connections.
	StatusOpen Status = "open"
	// StatusClosed indicates the tunnel is not active.
	StatusClosed Status = "closed"
)

// Config holds all parameters needed to establish an SSH tunnel.
type Config struct {
	// SSHHost is the hostname or IP of the SSH jump server.
	SSHHost string
	// SSHPort is the port on the SSH jump server (typically 22).
	SSHPort int
	// SSHUser is the username to authenticate with on the SSH server.
	SSHUser string
	// KeyPath is the path to the private key file. May contain a leading tilde.
	// If empty and UseAgent is false, authentication will fail.
	KeyPath string
	// UseAgent indicates that the SSH agent ($SSH_AUTH_SOCK) should be used.
	UseAgent bool
}

// State describes a currently-open tunnel.
type State struct {
	// LocalHost is the address of the local listener (always 127.0.0.1).
	LocalHost string
	// LocalPort is the ephemeral port chosen for the local listener.
	LocalPort int
	// RemoteHost is the host on the far side of the SSH server being forwarded.
	RemoteHost string
	// RemotePort is the port on the remote host being forwarded.
	RemotePort int
	// Status reflects whether the tunnel is currently open or closed.
	Status Status
}
