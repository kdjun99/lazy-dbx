# Wave 4: Schema Catalog — Browse & Preview

> Goal: browse databases, tables, columns with types in the connected DB.

## Features

| # | Feature | Description | Status |
|---|---------|-------------|--------|
| T3 | Tables panel | Bottom-left schema browser area | Not Started |
| D1 | DB/table/column tree | Hierarchical schema browser | Not Started |
| D2 | Lazy loading | Fetch on node expand | Not Started |
| D3 | Column type display | Show data type for each column | Not Started |
| D4 | Table preview | Enter on table → SELECT first N rows | Not Started |

## Implementation Notes

### Schema Discovery (D1, D2, D3)
- MySQL: `information_schema.COLUMNS`, `information_schema.TABLES`
- PostgreSQL: `information_schema.columns`, `pg_catalog.pg_tables`
- Abstract behind a common catalog interface
- Lazy load: only fetch table list on DB expand, columns on table expand
- Cache catalog data per connection session

### Table Preview (D4)
- Enter on table node → `SELECT * FROM <table> LIMIT <auto_limit>`
- Reuse Wave 3 results panel to display preview
- Show preview indicator in status bar

### Layout Update
- Split left panel: top = connections (Wave 2), bottom = tables (new)

## Done Criteria
- [ ] Tables panel shows schema tree for connected DB
- [ ] Expanding a table shows columns with types
- [ ] Lazy loading works (no upfront full schema fetch)
- [ ] Enter on table shows preview in results panel
