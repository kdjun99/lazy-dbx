# Wave 1: Foundation — Config + DB Connection

> All other waves depend on this. Goal: connect to a real DB through SSH tunnel from CLI.

## Features

| # | Feature | Description | Status |
|---|---------|-------------|--------|
| C1 | TOML config loading | Parse `~/.config/lazy-dbx/connections.toml` | Done |
| C2 | Settings loading | Parse `~/.config/lazy-dbx/settings.toml` | Done |
| C3 | Hierarchical connection structure | `group > subgroup > connection` tree | Done |
| C15 | Environment tagging | `production`, `staging`, `test` labels | Done |
| C4 | MySQL driver connection | `go-sql-driver/mysql` based | Done |
| C5 | PostgreSQL driver connection | `lib/pq` based | Done |
| C6 | Connection pool management | Create / reuse / close lifecycle | Done |
| C11 | password_cmd | Shell command to fetch password (AWS SM, 1Password, etc.) | Done |
| C12 | Env var password | `$DB_PASSWORD_<NAME>` support | Done |
| C13 | Prompt password | Interactive input on connect | Done |
| C14 | Plaintext password (with warning) | Direct in config + warning | Done |
| C7 | SSH tunneling | Tunnel via bastion (`golang.org/x/crypto/ssh`) | Done |
| C8 | SSH key auth | `~/.ssh/id_ed25519` key file support | Done |
| C9 | ssh-agent support | System ssh-agent integration | Done |
| C10 | Reusable tunnel definitions | Shared `ssh_tunnels` section in config | Done |

## Implementation Notes

### Config (C1~C3, C15)
- Use `pelletier/go-toml/v2` for parsing
- Config path: `~/.config/lazy-dbx/connections.toml`
- Settings path: `~/.config/lazy-dbx/settings.toml`
- Connection struct should hold environment tag for later safety checks

### Password Management (C11~C14)
- Priority resolution order: `password_cmd` > `env` > `prompt` > `plaintext`
- `password_cmd`: execute shell command, capture stdout, trim whitespace
- Plaintext: log warning on startup

### DB Drivers (C4~C6)
- Abstract behind a common interface for MySQL and PostgreSQL
- Connection pool with configurable max connections
- Health check / ping on connect

### SSH Tunneling (C7~C10)
- Create local port forward through bastion host
- Support key file and ssh-agent auth methods
- Tunnel must be established before DB connection attempt
- Clean shutdown: close tunnel when connection is released

## Done Criteria
- [x] `lazy-dbx` can parse a sample `connections.toml` with MySQL + PostgreSQL entries
- [x] `lazy-dbx` can connect to MySQL and PostgreSQL through SSH tunnel
- [x] All 4 password methods work
- [x] Connection pool properly manages lifecycle
