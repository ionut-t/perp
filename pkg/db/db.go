package db

import (
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Database defines the contract for database operations
type Database interface {
	// Execute a SQL query and return the result
	Query(ctx context.Context, query string, args ...any) (QueryResult, error)
	// Generate a human-readable schema of the database
	GenerateSchema() (string, error)
	// Generate a human-readable schema for specific tables
	GenerateSchemaForTables(tables []string) (string, error)
	// Close the database connection
	Close()
}

// QueryResult defines the contract for query results
type QueryResult interface {
	Type() QueryType
	Query() string
	Rows() pgx.Rows
	ExecutionTime() time.Duration
}

// QueryType represents the type of SQL query
type QueryType int

const (
	QuerySelect QueryType = iota
	QueryInsert
	QueryUpdate
	QueryDelete
	QueryCreate
	QueryDrop
	QueryAlter
	QueryUnknown
)

// ColumnInfo represents database column metadata
type ColumnInfo struct {
	TableName     string
	ColumnName    string
	DataType      string
	IsNullable    bool
	ColumnDefault string
}

// New creates a new database pool based on the provided DSN
func New(dbDSN string) (Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbDSN)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	return &database{pool: pool}, nil
}

// database encapsulates the pgx database connection pool
type database struct {
	pool *pgxpool.Pool
}

var _ Database = (*database)(nil)

// queryResult implements QueryResult interface
type queryResult struct {
	queryType QueryType
	query     string
	rows      pgx.Rows
	startTime time.Time
	endTime   time.Time
}

func (r queryResult) Type() QueryType {
	return r.queryType
}

func (r queryResult) Query() string {
	return r.query
}

func (r queryResult) Rows() pgx.Rows {
	return r.rows
}

func (r queryResult) ExecutionTime() time.Duration {
	return r.endTime.Sub(r.startTime)
}

// Close closes the underlying database connection
func (d *database) Close() {
	if d == nil || d.pool == nil {
		return
	}

	d.pool.Close()
}

func (d *database) Query(ctx context.Context, query string, args ...any) (QueryResult, error) {
	startTime := time.Now()
	rows, err := d.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	result := queryResult{
		rows:      rows,
		query:     query,
		startTime: startTime,
		endTime:   time.Now(),
	}

	q := stripSQLComments(query)
	q = strings.ToLower(strings.TrimSpace(q))

	switch {
	case strings.HasPrefix(q, "select"):
		result.queryType = QuerySelect
	case strings.HasPrefix(q, "insert"):
		if strings.Contains(q, "returning") {
			result.queryType = QuerySelect
		} else {
			result.queryType = QueryInsert
		}
	case strings.HasPrefix(q, "update"):
		if strings.Contains(q, "returning") {
			result.queryType = QuerySelect
		} else {
			result.queryType = QueryUpdate
		}
	case strings.HasPrefix(q, "delete"):
		result.queryType = QueryDelete

	case strings.HasPrefix(q, "create table"):
		result.queryType = QueryCreate

	case strings.HasPrefix(q, "drop table"):
		result.queryType = QueryDrop

	case strings.HasPrefix(q, "alter table"):
		result.queryType = QueryAlter
	default:
		result.queryType = QueryUnknown
	}

	return result, nil
}

// GenerateSchema fetches schema from DB and formats it as a human-readable string
func (d *database) GenerateSchema() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := d.pool.Query(ctx, `
		SELECT
			table_name,
			column_name,
			data_type,
			is_nullable,
			column_default
		FROM
			information_schema.columns
		WHERE
			table_schema = 'public' 
		ORDER BY
			table_name, ordinal_position;
	`)
	if err != nil {
		return "", fmt.Errorf("failed to query information_schema: %w", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var isNullableStr string
		var columnDefault *string
		if err := rows.Scan(&col.TableName, &col.ColumnName, &col.DataType, &isNullableStr, &columnDefault); err != nil {
			return "", fmt.Errorf("failed to scan column info: %w", err)
		}
		col.IsNullable = (isNullableStr == "YES")
		if columnDefault != nil {
			col.ColumnDefault = *columnDefault
		} else {
			col.ColumnDefault = ""
		}
		columns = append(columns, col)
	}

	if rows.Err() != nil {
		return "", fmt.Errorf("error after reading rows: %w", rows.Err())
	}

	tables := make(map[string][]ColumnInfo)
	for _, col := range columns {
		tables[col.TableName] = append(tables[col.TableName], col)
	}

	var b strings.Builder
	tmpl := template.Must(template.New("schema").Parse(`
Database Schema:
{{range $tableName, $cols := .}}
Table: {{$tableName}}
{{range $col := $cols}}- {{$col.ColumnName}} ({{$col.DataType}}): {{$col.ColumnDefault}} {{if $col.IsNullable}}[nullable]{{else}}[not nullable]{{end}}
{{end}}
{{end}}
`))

	if err := tmpl.Execute(&b, tables); err != nil {
		return "", fmt.Errorf("failed to execute schema template: %w", err)
	}

	return b.String(), nil
}
func (d *database) GenerateSchemaForTables(tables []string) (string, error) {
	if len(tables) == 0 {
		return "", fmt.Errorf("no tables specified")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	placeholders := make([]string, len(tables))
	args := make([]any, len(tables))
	for i, table := range tables {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = table
	}

	query := fmt.Sprintf(`
		SELECT
			table_name,
			column_name,
			data_type,
			is_nullable,
			column_default
		FROM
			information_schema.columns
		WHERE
			table_schema = 'public' 
			AND table_name IN (%s)
		ORDER BY
			table_name, ordinal_position;
	`, strings.Join(placeholders, ","))

	rows, err := d.pool.Query(ctx, query, args...)
	if err != nil {
		return "", fmt.Errorf("failed to query information_schema: %w", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var isNullableStr string
		var columnDefault *string
		if err := rows.Scan(&col.TableName, &col.ColumnName, &col.DataType, &isNullableStr, &columnDefault); err != nil {
			return "", fmt.Errorf("failed to scan column info: %w", err)
		}
		col.IsNullable = (isNullableStr == "YES")
		if columnDefault != nil {
			col.ColumnDefault = *columnDefault
		} else {
			col.ColumnDefault = ""
		}
		columns = append(columns, col)
	}

	if rows.Err() != nil {
		return "", fmt.Errorf("error after reading rows: %w", rows.Err())
	}

	// Group columns by table name
	tableColumns := make(map[string][]ColumnInfo)
	for _, col := range columns {
		tableColumns[col.TableName] = append(tableColumns[col.TableName], col)
	}

	var b strings.Builder
	tmpl := template.Must(template.New("schema").Parse(`
{{range $tableName, $cols := .}}
Table: {{$tableName}}
{{range $col := $cols}}- {{$col.ColumnName}} ({{$col.DataType}}): {{$col.ColumnDefault}} {{if $col.IsNullable}}[nullable]{{else}}[not nullable]{{end}}
{{end}}
{{end}}
`))

	if err := tmpl.Execute(&b, tableColumns); err != nil {
		return "", fmt.Errorf("failed to execute schema template: %w", err)
	}

	return strings.TrimSpace(b.String()), nil
}
