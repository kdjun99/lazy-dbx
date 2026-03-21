# Wave 3: Core Loop — Query & Results

> Goal: type a SQL query, execute it, see results. The minimum usable tool.

## Features

| # | Feature | Description | Status |
|---|---------|-------------|--------|
| T4 | Query Editor panel | Top-right SQL input area | **Done** |
| T5 | Query Results panel | Bottom-right result table | **Done** |
| E1 | SQL input & execution | Basic text input + `Ctrl+E` execute | **Done** |
| R1 | Paginated table view | Configurable page size from settings | **Done** |
| R2 | Horizontal/vertical scroll | Navigate wide result sets | **Done** |
| R4 | Row count display | Show total rows returned | **Done** |
| R5 | Execution time display | Show query duration | **Done** |

## Implementation Notes

### Query Editor (T4, E1)
- Basic tview TextArea for SQL input
- `Ctrl+E` sends text to active DB connection
- Support multi-line queries
- Trim trailing semicolons for driver compatibility if needed

### Results Table (T5, R1, R2, R4, R5)
- tview Table component
- Column headers from result set metadata
- Paginate: load `result_page_size` rows (default 100 from settings)
- Footer: row count + execution time
- Arrow keys / hjkl for scrolling

### Layout Update
- Expand TUI frame from Wave 2:
  - Left: connections panel (existing)
  - Top-right: query editor (new)
  - Bottom-right: results table (new)
  - Bottom: status bar (existing)

## Done Criteria
- [x] Full 4-panel layout renders correctly
- [x] User can type SQL in editor and execute with Ctrl+E
- [x] Results display in paginated table with scroll
- [x] Row count and execution time shown
