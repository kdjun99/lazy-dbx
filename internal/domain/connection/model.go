// Package connection defines domain interfaces and models for database connection management.
package connection

// Info describes a database connection's identity and current status.
type Info struct {
	Path     string
	Group    string
	Subgroup string
	Name     string
	Type     string
	Host     string
	Env      string
	Status   string
}

// PingResult holds the result of a connection ping.
type PingResult struct {
	Path          string
	LatencyMs     int64
	ServerVersion string
}

// PoolConfig holds database connection pool settings.
type PoolConfig struct {
	MaxOpen     int
	MaxIdle     int
	MaxLifetime string // duration string e.g. "5m"
}
