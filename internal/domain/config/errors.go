package config

import "errors"

// Sentinel errors for config operations.
var (
	ErrConfigNotFound     = errors.New("config file not found")
	ErrConfigParse        = errors.New("config parse error")
	ErrConnectionNotFound = errors.New("connection not found")
	ErrDanglingTunnelRef  = errors.New("dangling ssh_tunnel reference")
)
