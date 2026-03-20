# Wave 6: Editor — Highlighting & Completion

> Goal: comfortable query writing experience comparable to DataGrip.

## Features

| # | Feature | Description | Status |
|---|---------|-------------|--------|
| E2 | SQL syntax highlighting | `chroma` based, MySQL/PG dialect aware | Not Started |
| E3 | Autocomplete — table names | Suggest tables from current connection catalog | Not Started |
| E4 | Autocomplete — column names | Suggest columns from selected table | Not Started |
| E5 | Autocomplete — SQL keywords | SELECT, WHERE, JOIN, etc. | Not Started |
| E6 | Multi-tab | `Ctrl+N` new tab, `Ctrl+W` close tab | Not Started |
| E7 | Query history | Up/down arrow to browse previous queries | Not Started |
| R3 | Column sorting | Keybind to sort results ASC/DESC | Not Started |

## Implementation Notes

### Syntax Highlighting (E2)
- Use `alecthomas/chroma/v2` with SQL lexer
- Detect dialect from connection type (MySQL vs PostgreSQL)
- Render highlighted text in editor panel (may need custom tview widget)

### Autocomplete (E3~E5)
- Data source: Wave 4 catalog cache
- Trigger: Tab key or configurable trigger
- E3: match against table names in current database
- E4: context-aware — detect table alias in query, suggest columns
- E5: static list of SQL keywords, always available
- Popup dropdown in editor area

### Multi-tab (E6)
- Tab bar above editor panel
- Each tab holds independent query text + result set
- `Ctrl+N` creates new tab, `Ctrl+W` closes current
- Tab switching via `Ctrl+1~9` or tab bar navigation

### Query History (E7)
- Store executed queries in memory (per session) and on disk
- Up/Down in empty editor to cycle through history
- History file: `~/.config/lazy-dbx/history`

### Column Sorting (R3)
- Keybind on results table to toggle sort by focused column
- Client-side sort on loaded result set

## Done Criteria
- [ ] SQL keywords and identifiers are color-highlighted
- [ ] Autocomplete suggests tables, columns, and keywords
- [ ] Multiple query tabs work independently
- [ ] Query history persists across sessions
- [ ] Results can be sorted by column
