package app

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/kdjun99/lazy-dbx/internal/domain"
	domainconn "github.com/kdjun99/lazy-dbx/internal/domain/connection"
	domainlogger "github.com/kdjun99/lazy-dbx/internal/domain/logger"
	"github.com/kdjun99/lazy-dbx/internal/domain/query"
)

// Compile-time check: QueryService must satisfy query.Executor.
var _ query.Executor = (*QueryService)(nil)

// QueryService executes SQL statements against pooled database connections.
type QueryService struct {
	pool    domainconn.Pool
	logger  domainlogger.Logger
	maxRows int
}

// NewQueryService creates a QueryService with injected dependencies.
func NewQueryService(pool domainconn.Pool, logger domainlogger.Logger, maxRows int) *QueryService {
	return &QueryService{
		pool:    pool,
		logger:  logger,
		maxRows: maxRows,
	}
}

// Execute runs a SQL statement against the connection identified by path.
func (s *QueryService) Execute(ctx context.Context, path string, sqlStr string) domain.Result[query.Result] {
	s.logger.Info(ctx, "QueryService", "Execute", "starting",
		domainlogger.F("path", path),
		domainlogger.F("sql", sqlStr),
	)

	db, err := s.pool.Get(path)
	if err != nil {
		s.logger.Error(ctx, "QueryService", "Execute", "pool get error",
			domainlogger.F("path", path),
			domainlogger.F("error", err.Error()),
		)
		return domain.Result[query.Result]{Error: err}
	}

	stmtType := query.ClassifyStatement(sqlStr)
	cleaned := strings.TrimRight(sqlStr, "; \t\n")

	start := time.Now()

	var result query.Result
	result.Type = stmtType

	if stmtType == query.StatementQuery {
		r, execErr := s.execQuery(ctx, db, cleaned)
		if execErr != nil {
			s.logger.Error(ctx, "QueryService", "Execute", "query error",
				domainlogger.F("path", path),
				domainlogger.F("error", execErr.Error()),
			)
			return domain.Result[query.Result]{Error: execErr}
		}
		result = *r
		result.Type = stmtType
	} else {
		rowsAffected, execErr := s.execStatement(ctx, db, path, cleaned)
		if execErr != nil {
			return domain.Result[query.Result]{Error: execErr}
		}
		result.RowsAffected = rowsAffected
	}

	result.Duration = time.Since(start)

	s.logger.Info(ctx, "QueryService", "Execute", "completed",
		domainlogger.F("path", path),
		domainlogger.F("type", stmtType),
		domainlogger.F("duration_ms", result.Duration.Milliseconds()),
	)

	return domain.Result[query.Result]{Data: result}
}

func (s *QueryService) execStatement(ctx context.Context, db *sql.DB, path, sqlStr string) (int64, error) {
	sqlResult, err := db.ExecContext(ctx, sqlStr)
	if err != nil {
		s.logger.Error(ctx, "QueryService", "Execute", "exec error",
			domainlogger.F("path", path),
			domainlogger.F("error", err.Error()),
		)
		return 0, err
	}
	rowsAffected, err := sqlResult.RowsAffected()
	if err != nil {
		s.logger.Warn(ctx, "QueryService", "Execute", "rows affected unavailable",
			domainlogger.F("error", err.Error()),
		)
	}
	return rowsAffected, nil
}

func (s *QueryService) execQuery(ctx context.Context, db *sql.DB, sqlStr string) (*query.Result, error) {
	rows, err := db.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	colNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	columns := make([]query.ColumnInfo, len(colNames))
	for i, name := range colNames {
		columns[i] = query.ColumnInfo{
			Name:     name,
			TypeName: colTypes[i].DatabaseTypeName(),
		}
	}

	resultRows, truncated, err := scanRows(rows, len(colNames), s.maxRows)
	if err != nil {
		return nil, err
	}

	return &query.Result{
		Columns:   columns,
		Rows:      resultRows,
		TotalRows: len(resultRows),
		Truncated: truncated,
	}, nil
}

// scanRows reads all rows from a sql.Rows result, up to maxRows.
func scanRows(rows *sql.Rows, numCols, maxRows int) ([][]query.Value, bool, error) {
	var resultRows [][]query.Value
	truncated := false

	for rows.Next() {
		if len(resultRows) >= maxRows {
			truncated = true
			break
		}
		row, err := scanSingleRow(rows, numCols)
		if err != nil {
			return nil, false, err
		}
		resultRows = append(resultRows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return resultRows, truncated, nil
}

// scanSingleRow scans one row into a slice of query.Value.
func scanSingleRow(rows *sql.Rows, numCols int) ([]query.Value, error) {
	scanArgs := make([]any, numCols)
	nullStrings := make([]sql.NullString, numCols)
	for i := range scanArgs {
		scanArgs[i] = &nullStrings[i]
	}
	if err := rows.Scan(scanArgs...); err != nil {
		return nil, err
	}
	row := make([]query.Value, numCols)
	for i, ns := range nullStrings {
		row[i] = query.Value{String: ns.String, Valid: ns.Valid}
	}
	return row, nil
}
