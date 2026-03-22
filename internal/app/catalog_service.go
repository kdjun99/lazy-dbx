package app

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/kdjun99/lazy-dbx/internal/domain"
	"github.com/kdjun99/lazy-dbx/internal/domain/catalog"
	domainconn "github.com/kdjun99/lazy-dbx/internal/domain/connection"
	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
)

// Compile-time check: CatalogService must satisfy catalog.Provider.
var _ catalog.Provider = (*CatalogService)(nil)

// catalogCache holds cached schema data for a single connection.
type catalogCache struct {
	databases []catalog.Database
	tables    map[string][]catalog.Table  // key: database name
	columns   map[string][]catalog.Column // key: "database.table"
}

// CatalogService lists databases, tables, and columns from pooled connections.
type CatalogService struct {
	pool           domainconn.Pool
	logger         domainlogger.Logger
	mu             sync.RWMutex
	cache          map[string]*catalogCache
	driverTypeFunc func(path string) string
}

// NewCatalogService creates a CatalogService with injected dependencies.
// driverTypeFunc returns "mysql" or "postgresql" for a given connection path.
func NewCatalogService(pool domainconn.Pool, log domainlogger.Logger, driverTypeFunc func(path string) string) *CatalogService {
	return &CatalogService{
		pool:           pool,
		logger:         log,
		cache:          make(map[string]*catalogCache),
		driverTypeFunc: driverTypeFunc,
	}
}

// ClearCache removes cached schema data for the given connection path.
func (s *CatalogService) ClearCache(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cache, path)
}

// ListDatabases lists all non-system databases for the connection at path.
func (s *CatalogService) ListDatabases(ctx context.Context, path string) domain.Result[[]catalog.Database] {
	s.mu.RLock()
	if c, ok := s.cache[path]; ok && c.databases != nil {
		s.mu.RUnlock()
		return domain.Result[[]catalog.Database]{Data: c.databases}
	}
	s.mu.RUnlock()

	db, err := s.getDB(path)
	if err != nil {
		return domain.Result[[]catalog.Database]{Error: err}
	}

	driver := s.driverTypeFunc(path)
	query, systemSchemas := listDatabasesQuery(driver)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		s.logError(ctx, "ListDatabases", path, err)
		return domain.Result[[]catalog.Database]{Error: fmt.Errorf("list databases: %w", err)}
	}
	defer rows.Close() //nolint:errcheck

	var databases []catalog.Database
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			s.logError(ctx, "ListDatabases", path, scanErr)
			return domain.Result[[]catalog.Database]{Error: fmt.Errorf("scan database row: %w", scanErr)}
		}
		if isSystemSchema(name, systemSchemas) {
			continue
		}
		databases = append(databases, catalog.Database{Name: name})
	}
	if err := rows.Err(); err != nil {
		return domain.Result[[]catalog.Database]{Error: fmt.Errorf("rows error: %w", err)}
	}

	s.ensureCache(path).databases = databases
	s.logInfo(ctx, "ListDatabases", path, fmt.Sprintf("found %d databases", len(databases)))
	return domain.Result[[]catalog.Database]{Data: databases}
}

// ListTables lists all non-system tables/views for the given database.
func (s *CatalogService) ListTables(ctx context.Context, path string, database string) domain.Result[[]catalog.Table] {
	s.mu.RLock()
	if c, ok := s.cache[path]; ok {
		if tables, ok := c.tables[database]; ok {
			s.mu.RUnlock()
			return domain.Result[[]catalog.Table]{Data: tables}
		}
	}
	s.mu.RUnlock()

	db, err := s.getDB(path)
	if err != nil {
		return domain.Result[[]catalog.Table]{Error: err}
	}

	driver := s.driverTypeFunc(path)
	query := listTablesQuery(driver)

	rows, err := db.QueryContext(ctx, query, database)
	if err != nil {
		s.logError(ctx, "ListTables", path, err)
		return domain.Result[[]catalog.Table]{Error: fmt.Errorf("list tables: %w", err)}
	}
	defer rows.Close() //nolint:errcheck

	var tables []catalog.Table
	for rows.Next() {
		var name, schema, tableType string
		if scanErr := rows.Scan(&name, &schema, &tableType); scanErr != nil {
			s.logError(ctx, "ListTables", path, scanErr)
			return domain.Result[[]catalog.Table]{Error: fmt.Errorf("scan table row: %w", scanErr)}
		}
		tables = append(tables, catalog.Table{Name: name, Schema: schema, Type: tableType})
	}
	if err := rows.Err(); err != nil {
		return domain.Result[[]catalog.Table]{Error: fmt.Errorf("rows error: %w", err)}
	}

	c := s.ensureCache(path)
	if c.tables == nil {
		c.tables = make(map[string][]catalog.Table)
	}
	c.tables[database] = tables
	s.logInfo(ctx, "ListTables", path, fmt.Sprintf("found %d tables in %s", len(tables), database))
	return domain.Result[[]catalog.Table]{Data: tables}
}

// ListColumns lists all columns for the given table in the given database.
func (s *CatalogService) ListColumns(ctx context.Context, path string, database string, table string) domain.Result[[]catalog.Column] {
	cacheKey := database + "." + table
	s.mu.RLock()
	if c, ok := s.cache[path]; ok {
		if cols, ok := c.columns[cacheKey]; ok {
			s.mu.RUnlock()
			return domain.Result[[]catalog.Column]{Data: cols}
		}
	}
	s.mu.RUnlock()

	db, err := s.getDB(path)
	if err != nil {
		return domain.Result[[]catalog.Column]{Error: err}
	}

	driver := s.driverTypeFunc(path)
	query := listColumnsQuery(driver)

	rows, err := db.QueryContext(ctx, query, database, table)
	if err != nil {
		s.logError(ctx, "ListColumns", path, err)
		return domain.Result[[]catalog.Column]{Error: fmt.Errorf("list columns: %w", err)}
	}
	defer rows.Close() //nolint:errcheck

	var columns []catalog.Column
	for rows.Next() {
		var col catalog.Column
		var ordinal int
		var nullable, keyType, defaultVal sql.NullString
		if scanErr := rows.Scan(&col.Name, &ordinal, &col.DataType, &nullable, &keyType, &defaultVal); scanErr != nil {
			s.logError(ctx, "ListColumns", path, scanErr)
			return domain.Result[[]catalog.Column]{Error: fmt.Errorf("scan column row: %w", scanErr)}
		}
		col.OrdinalPos = ordinal
		col.Nullable = nullable.Valid && strings.EqualFold(nullable.String, "YES")
		col.KeyType = keyType.String
		col.Default = defaultVal.String
		columns = append(columns, col)
	}
	if err := rows.Err(); err != nil {
		return domain.Result[[]catalog.Column]{Error: fmt.Errorf("rows error: %w", err)}
	}

	c := s.ensureCache(path)
	if c.columns == nil {
		c.columns = make(map[string][]catalog.Column)
	}
	c.columns[cacheKey] = columns
	s.logInfo(ctx, "ListColumns", path, fmt.Sprintf("found %d columns in %s.%s", len(columns), database, table))
	return domain.Result[[]catalog.Column]{Data: columns}
}

// --- helpers ---

func (s *CatalogService) getDB(path string) (*sql.DB, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("no connection pool available")
	}
	db, err := s.pool.Get(path)
	if err != nil {
		return nil, fmt.Errorf("get connection %q: %w", path, err)
	}
	return db, nil
}

func (s *CatalogService) ensureCache(path string) *catalogCache {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.cache[path]; !ok {
		s.cache[path] = &catalogCache{}
	}
	return s.cache[path]
}

func (s *CatalogService) logInfo(ctx context.Context, action, path, detail string) {
	if s.logger == nil {
		return
	}
	s.logger.Info(ctx, "CatalogService", action, detail, domainlogger.F("path", path))
}

func (s *CatalogService) logError(ctx context.Context, action, path string, err error) {
	if s.logger == nil {
		return
	}
	s.logger.Error(ctx, "CatalogService", action, "error",
		domainlogger.F("path", path),
		domainlogger.F("error", err.Error()),
	)
}

func isSystemSchema(name string, systemSchemas []string) bool {
	for _, s := range systemSchemas {
		if strings.EqualFold(name, s) {
			return true
		}
	}
	return false
}

// listDatabasesQuery returns the SQL to list databases and the system schemas to filter.
func listDatabasesQuery(driver string) (string, []string) {
	switch driver {
	case "postgresql":
		return `SELECT schema_name FROM information_schema.schemata ORDER BY schema_name`,
			[]string{"information_schema", "pg_catalog", "pg_toast"}
	default: // mysql
		return `SELECT schema_name FROM information_schema.SCHEMATA ORDER BY schema_name`,
			[]string{"information_schema", "mysql", "performance_schema", "sys"}
	}
}

// listTablesQuery returns the SQL to list tables for a given database (placeholder $1 or ?).
func listTablesQuery(driver string) string {
	switch driver {
	case "postgresql":
		return `SELECT table_name, table_schema, table_type
				FROM information_schema.tables
				WHERE table_schema = $1
				ORDER BY table_name`
	default: // mysql
		return `SELECT table_name, table_schema, table_type
				FROM information_schema.TABLES
				WHERE table_schema = ?
				ORDER BY table_name`
	}
}

// listColumnsQuery returns the SQL to list columns for a given database+table.
func listColumnsQuery(driver string) string {
	switch driver {
	case "postgresql":
		return `SELECT column_name, ordinal_position, data_type,
				       is_nullable, '' as column_key, column_default
				FROM information_schema.columns
				WHERE table_schema = $1 AND table_name = $2
				ORDER BY ordinal_position`
	default: // mysql
		return `SELECT column_name, ordinal_position, column_type,
				       is_nullable, column_key, column_default
				FROM information_schema.COLUMNS
				WHERE table_schema = ? AND table_name = ?
				ORDER BY ordinal_position`
	}
}
