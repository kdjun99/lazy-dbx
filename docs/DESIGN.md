# lazy-dbx Design Document

> Terminal-based Database IDE — DataGrip experience in your terminal.

## Motivation

Managing 30+ database connections across multiple services (linkareer, cbt, linkbrary, etc.) with different roles (write/root) and environments (prod/staging/test) through DataGrip works, but:

- Heavy GUI app just for quick queries
- Can't integrate into terminal-centric workflow
- Existing CLI tools (pgcli, mycli) lack connection management at scale
- lazysql/harlequin lack production safety features and SSH tunneling

**lazy-dbx** aims to be a lightweight, terminal-native Database IDE with first-class support for multi-connection management, SSH tunneling, and production safety.

## Layout

```
┌─ Connections ──┬─ Query Editor ──────────────────┐
│ 📁 linkareer   │  SELECT u.name, u.email         │
│  📁 write      │  FROM users u                   │
│   🔵 main-db   │  WHERE u.created_at > '2026-01' │
│   🔵 chat-db   │  ORDER BY u.created_at DESC     │
│   🔵 xe-db     │  LIMIT 100;                     │
│  📁 root       │                                 │
│   🔵 main-db   ├─ Query Results (1,234 rows) ────┤
│ 📁 cbt         │ name     │ email     │ created  │
│ 📁 linkbrary   │ 김동준   │ kim@..    │ 2026-03  │
│                │ 이서연   │ lee@..    │ 2026-03  │
│─ Tables ───────│ 박지호   │ park@..   │ 2026-02  │
│ ▸ users        │                                 │
│ ▸ posts        │                                 │
│ ▸ comments     │                                 │
├────────────────┼──────────────────────────────────┤
│ ^q Quit  ^e Execute  ^s Save  ^c Connections     │
└────────────────┴──────────────────────────────────┘
```

### Panels

| Panel | Description |
|-------|-------------|
| **Connections** (top-left) | Grouped tree view of all DB connections. Navigate with j/k, Enter to connect |
| **Tables** (bottom-left) | Schema browser showing tables/columns/types for the active connection |
| **Query Editor** (top-right) | SQL editor with syntax highlighting, auto-completion, multi-tab support |
| **Query Results** (bottom-right) | Paginated table view with sorting, scrolling, row count |
| **Status Bar** (bottom) | Keybindings, connection info, execution time |

## Core Features

### 1. Connection Manager

- TOML-based configuration (`~/.config/lazy-dbx/connections.toml`)
- Hierarchical grouping: `group > subgroup > connection`
- Fuzzy search across all connections
- Support for MySQL and PostgreSQL
- Environment tagging: `production`, `staging`, `test`

### 2. SSH Tunneling

- Built-in SSH tunnel support via `golang.org/x/crypto/ssh`
- Reusable tunnel definitions in config
- Auto-connect through bastion hosts
- SSH key and ssh-agent support

### 3. Data Catalog

- Tree view of databases, tables, columns with types
- Lazy loading (fetch on expand)
- Quick table preview (select first N rows)

### 4. Query Editor

- SQL syntax highlighting (MySQL/PostgreSQL dialect aware)
- Auto-completion for table names, column names, SQL keywords
- Multi-tab support
- Query history (up/down navigation)

### 5. Query Script Storage

- Save frequently used queries to `~/.config/lazy-dbx/scripts/`
- Organize by connection group or custom tags
- Quick load via fuzzy search

### 6. Query Results

- Paginated table view with horizontal/vertical scrolling
- Column sorting (click/keybind)
- Export to CSV, JSON, TSV
- Row count and execution time display

### 7. Production Safety

| Feature | Description |
|---------|-------------|
| **Environment banner** | Red warning bar when connected to production |
| **DML confirmation** | `UPDATE`, `DELETE`, `DROP` require explicit confirmation |
| **Read-only mode** | Default mode; must pass `--write` or toggle in UI |
| **Query limit** | Auto-append `LIMIT` to SELECT queries without one |
| **Connection lock** | Optional pin/confirmation to connect to prod |

### 8. Audit Log

- Log all executed queries with timestamp, connection, user
- Stored at `~/.config/lazy-dbx/audit.log`
- Configurable retention

### 9. Password Management

| Method | Description |
|--------|-------------|
| **password_cmd** | Shell command to fetch password (AWS Secrets Manager, 1Password CLI, etc.) |
| **env var** | `$DB_PASSWORD_<NAME>` environment variable |
| **prompt** | Ask on connection |
| **plaintext** | Direct in config (not recommended, with warning) |

## Tech Stack

| Area | Choice | Reason |
|------|--------|--------|
| Language | Go 1.24 | Single binary, cross-platform, strong ecosystem |
| TUI Framework | `rivo/tview` | Mature, panel layouts, used by lazysql |
| DB Drivers | `go-sql-driver/mysql`, `lib/pq` | Standard Go database drivers |
| SSH | `golang.org/x/crypto/ssh` | Go standard SSH library |
| Config | `pelletier/go-toml/v2` | TOML parser |
| SQL Highlight | `alecthomas/chroma/v2` | Syntax highlighting |
| CLI Entry | `spf13/cobra` | Subcommand management |
| Fuzzy Search | `sahilm/fuzzy` | Fuzzy matching |

## Configuration

### connections.toml

```toml
[groups.linkareer.write]
  [groups.linkareer.write.main-db]
  type = "mysql"
  host = "db.linkareer.com"
  port = 3306
  user = "linkareer_db"
  password_cmd = "aws secretsmanager get-secret-value --secret-id prod/main-db | jq -r .SecretString"
  database = "linkareer_db"
  ssh_tunnel = "bastion-prod"
  env = "production"

  [groups.linkareer.write.chat-db]
  type = "mysql"
  host = "chat-db.linkareer.com"
  port = 3306
  user = "linkareer_chat"
  database = "linkareer_chat"
  ssh_tunnel = "bastion-prod"
  env = "production"

[groups.linkbrary.root]
  [groups.linkbrary.root.main]
  type = "postgresql"
  host = "linkbrary-cluster-xxx.ap-northeast-2.rds.amazonaws.com"
  port = 5432
  user = "linkbrary"
  database = "linkbrary"
  ssh_tunnel = "bastion-linkbrary"
  env = "production"

[ssh_tunnels]
  [ssh_tunnels.bastion-prod]
  host = "bastion.linkareer.com"
  user = "ec2-user"
  key = "~/.ssh/id_ed25519"

  [ssh_tunnels.bastion-linkbrary]
  host = "bastion.linkbrary.com"
  user = "ec2-user"
  key = "~/.ssh/id_ed25519"
```

### settings.toml

```toml
[editor]
tab_size = 2
auto_limit = 500        # auto-append LIMIT to SELECT
show_line_numbers = true

[safety]
read_only_default = true
confirm_dml = true
prod_warning_banner = true

[audit]
enabled = true
path = "~/.config/lazy-dbx/audit.log"
retention_days = 90

[ui]
theme = "dark"           # dark, light
show_column_types = true
result_page_size = 100
```

## Project Structure

```
lazy-dbx/
├── cmd/                    # CLI entrypoint (cobra)
│   └── root.go
├── internal/
│   ├── config/             # TOML config loading
│   ├── connection/         # DB connection pool management
│   ├── tunnel/             # SSH tunnel management
│   ├── tui/                # TUI layout and event handling
│   ├── catalog/            # Schema/table/column discovery
│   ├── editor/             # Query editor component
│   ├── result/             # Query result table component
│   ├── safety/             # Production safety checks
│   ├── audit/              # Query audit logging
│   └── script/             # Saved query management
├── docs/
│   └── DESIGN.md           # This document
├── go.mod
├── go.sum
├── main.go
└── README.md
```

## Development Roadmap

### Phase 1: MVP — Connect & Query
- [ ] TOML config loading (connections, ssh tunnels)
- [ ] MySQL and PostgreSQL connection
- [ ] Basic TUI layout (3 panels: connections, editor, results)
- [ ] Query execution and result display
- [ ] Keyboard navigation between panels

### Phase 2: Catalog & Tunneling
- [ ] SSH tunnel auto-setup through bastion
- [ ] Data catalog tree (databases, tables, columns with types)
- [ ] Table preview (double-click/enter on table)
- [ ] Connection status indicator

### Phase 3: Editor & Scripts
- [ ] SQL syntax highlighting
- [ ] Auto-completion (tables, columns, keywords)
- [ ] Multi-tab query editor
- [ ] Save/load query scripts
- [ ] Query history

### Phase 4: Safety & Operations
- [ ] Production environment warning banner
- [ ] DML confirmation prompt
- [ ] Read-only mode toggle
- [ ] Auto LIMIT append
- [ ] Audit logging
- [ ] Password command execution
- [ ] Export results (CSV, JSON)

## Keybindings

| Key | Action |
|-----|--------|
| `Ctrl+Q` | Quit |
| `Ctrl+E` | Execute query |
| `Ctrl+S` | Save query to script |
| `Ctrl+O` | Open saved script |
| `Ctrl+C` | Connection selector |
| `Tab` | Switch panel focus |
| `Ctrl+N` | New query tab |
| `Ctrl+W` | Close query tab |
| `Ctrl+R` | Toggle read-only mode |
| `j/k` | Navigate tree/results |
| `Enter` | Connect / Expand node |
| `Esc` | Cancel / Back |

## Competitors & Differentiation

| Feature | DataGrip | harlequin | lazysql | **lazy-dbx** |
|---------|----------|-----------|---------|-------------|
| TUI | ❌ | ✅ | ✅ | ✅ |
| Multi-DB (MySQL+PG) | ✅ | ✅ | ✅ | ✅ |
| Connection groups | ✅ | ✅ | ✅ | ✅ |
| SSH tunneling | ✅ | ❌ | ❌ | ✅ |
| Prod safety | ❌ | ❌ | ❌ | ✅ |
| Query scripts | ✅ | ✅ | ❌ | ✅ |
| Audit log | ❌ | ❌ | ❌ | ✅ |
| Secret manager | ❌ | ❌ | ❌ | ✅ |
| Single binary | ❌ | ❌ | ✅ | ✅ |
| Free | ❌ | ✅ | ✅ | ✅ |
