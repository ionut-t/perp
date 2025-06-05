package db

import (
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/jackc/pgx/pgtype"
	"github.com/jackc/pgx/v5"
)

// Database encapsulates the pgx database connection.
type Database struct {
	conn *pgx.Conn
}

type QueryType int

const (
	QuerySelect QueryType = iota
	QueryInsert
	QueryUpdate
	QueryDelete
	QueryUnknown
)

type QueryResult interface {
	Type() QueryType
	Query() string
	Rows() pgx.Rows
}

type queryResult struct {
	queryType QueryType
	query     string
	rows      pgx.Rows
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

// Connect establishes a connection to the PostgreSQL database using the provided DSN.
func Connect(dbDSN string) (*Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, dbDSN)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	return &Database{conn: conn}, nil
}

// Close closes the underlying database connection.
func Close(d *Database) error {
	if d != nil && d.conn != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return d.conn.Close(ctx)
	}

	return nil
}

func (d *Database) ExecuteQuery(query string) (QueryResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := d.conn.Query(ctx, query)

	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	result := queryResult{
		rows:  rows,
		query: query,
	}

	q := strings.ToLower(query)
	switch {
	case strings.HasPrefix(q, "select"):
		result.queryType = QuerySelect
	case strings.HasPrefix(q, "insert"):
		if strings.Contains(q, "returning") {
			// If the query has a RETURNING clause, we treat it as a SELECT query
			result.queryType = QuerySelect
		} else {
			result.queryType = QueryInsert
		}
	case strings.HasPrefix(q, "update"):
		if strings.Contains(q, "returning") {
			// If the query has a RETURNING clause, we treat it as a SELECT query
			result.queryType = QuerySelect
		} else {
			result.queryType = QueryUpdate
		}
	case strings.HasPrefix(q, "delete"):
		result.queryType = QueryDelete
	default:
		result.queryType = QueryUnknown
	}

	return result, nil
}

// FetchQueryResults reads all rows from a pgx.Rows object and returns their data
// as a slice of maps (column name to value) and the column headers.
func (d *Database) FetchQueryResults(rows pgx.Rows) ([]map[string]any, []string, error) {
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}

	var results []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		scanTargets := make([]any, len(columns))

		for i, fd := range fieldDescriptions {
			switch fd.DataTypeOID {
			case pgtype.UUIDOID:
				var uuid pgtype.UUID
				values[i] = &uuid
				scanTargets[i] = &uuid
			default:
				var generic any
				values[i] = &generic
				scanTargets[i] = &generic
			}
		}

		if err := rows.Scan(scanTargets...); err != nil {
			return nil, nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowMap := make(map[string]any)
		for i, col := range columns {
			switch v := values[i].(type) {
			case *pgtype.UUID:
				if v.Status == pgtype.Present {
					rowMap[col] = fmt.Sprintf("%x-%x-%x-%x-%x", v.Bytes[0:4], v.Bytes[4:6], v.Bytes[6:8], v.Bytes[8:10], v.Bytes[10:16])
				} else {
					rowMap[col] = nil
				}

			default:
				rowMap[col] = *(values[i].(*any))
			}
		}
		results = append(results, rowMap)
	}

	if rows.Err() != nil {
		return nil, nil, fmt.Errorf("error after iterating rows: %w", rows.Err())
	}

	return results, columns, nil
}

type ColumnInfo struct {
	TableName     string
	ColumnName    string
	DataType      string
	IsNullable    bool
	ColumnDefault string
}

// GenerateSchema fetches schema from DB and formats it as a human-readable string.
func (d *Database) GenerateSchema() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := d.conn.Query(ctx, `
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
