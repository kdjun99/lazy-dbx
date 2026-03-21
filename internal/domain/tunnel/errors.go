// Package tunnel defines domain interfaces and models for SSH tunnel management.
package tunnel

import "errors"

// ErrTunnelFailed is returned when a tunnel cannot be established.
var ErrTunnelFailed = errors.New("tunnel: failed to establish tunnel")

// ErrTunnelClosed is returned when an operation is attempted on a closed tunnel.
var ErrTunnelClosed = errors.New("tunnel: tunnel is closed")

// ErrKeyParse is returned when the private key file cannot be parsed.
var ErrKeyParse = errors.New("tunnel: failed to parse private key")
