package db

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type RowResult struct {
	Value any
	Type  uint32
}

func ExtractResults(rows pgx.Rows) ([]map[string]RowResult, []string, error) {
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}

	var results []map[string]RowResult
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get row values: %w", err)
		}

		rowMap := make(map[string]RowResult, len(columns))
		for i, val := range values {
			rowMap[columns[i]] = RowResult{
				Value: val,
				Type:  fieldDescriptions[i].DataTypeOID,
			}
		}
		results = append(results, rowMap)
	}

	if rows.Err() != nil {
		return nil, nil, fmt.Errorf("error after iterating rows: %w", rows.Err())
	}

	return results, columns, nil
}

// ExtractPsqlResults processes the pgx.Rows and returns a slice of maps and column names.
func ExtractPsqlResults(rows pgx.Rows) ([]map[string]any, []string, error) {
	results, columns, err := ExtractResults(rows)
	if err != nil {
		return nil, nil, err
	}

	formattedResults := make([]map[string]any, len(results))
	for i, row := range results {
		formattedRow := make(map[string]any, len(row))
		for k, v := range row {
			formattedRow[k] = FormatValue(v.Value, v.Type)
		}
		formattedResults[i] = formattedRow
	}

	return formattedResults, columns, nil
}

// FormatValue converts a database value to a user-friendly string representation based on its PostgreSQL OID.
func FormatValue(val any, oid uint32) any {
	if val == nil {
		return nil
	}

	switch oid {
	case pgtype.UUIDOID:
		if v, ok := val.([16]byte); ok {
			return fmt.Sprintf("%x-%x-%x-%x-%x", v[0:4], v[4:6], v[6:8], v[8:10], v[10:16])
		}

	case pgtype.JSONOID, pgtype.JSONBOID:
		switch v := val.(type) {
		case []byte:
			var out bytes.Buffer
			if err := json.Compact(&out, v); err == nil {
				return out.String()
			}
			return string(v)
		case map[string]any:
			jsonBytes, err := json.Marshal(v)
			if err == nil {
				return string(jsonBytes)
			}
		}

	case pgtype.ByteaOID:
		if v, ok := val.([]byte); ok {
			return "\\x" + hex.EncodeToString(v)
		}

	case pgtype.NumericOID:
		if v, ok := val.(pgtype.Numeric); ok && v.Valid {
			str, err := v.Value()
			if err == nil {
				return str
			}
		}
	}

	switch reflect.TypeOf(val).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", val)
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", val)
	case reflect.Slice:
		return formatSlice(val)
	}

	return val
}

// formatSlice handles the formatting of slice types into a string representation.
func formatSlice(val any) string {
	slice := reflect.ValueOf(val)
	var elements []string
	for i := 0; i < slice.Len(); i++ {
		elements = append(elements, fmt.Sprintf("%v", slice.Index(i).Interface()))
	}
	return fmt.Sprintf("{%s}", strings.Join(elements, ","))
}

// stripSQLComments removes SQL comments from a query string, correctly handling
// various string literal and comment formats, including PostgreSQL-specific syntax.
func stripSQLComments(q string) string {
	var sb strings.Builder
	sb.Grow(len(q))

	for i := 0; i < len(q); {
		c := q[i]

		// Single-quoted strings
		if c == '\'' {
			sb.WriteByte(c)
			i++
			for i < len(q) {
				sc := q[i]
				sb.WriteByte(sc)
				i++
				if sc == '\'' {
					if i < len(q) && q[i] == '\'' {
						sb.WriteByte(q[i])
						i++
					} else {
						break
					}
				}
			}
			continue
		}

		// Double-quoted identifiers
		if c == '"' {
			sb.WriteByte(c)
			i++
			for i < len(q) {
				dc := q[i]
				sb.WriteByte(dc)
				i++
				if dc == '"' {
					break
				}
			}
			continue
		}

		// Dollar-quoted strings (PostgreSQL)
		if c == '$' && i+1 < len(q) && (q[i+1] == '$' || (q[i+1] >= 'a' && q[i+1] <= 'z')) {
			tagEnd := strings.Index(q[i+1:], "$")
			if tagEnd != -1 {
				tag := q[i : i+tagEnd+2]
				closingTag := tag
				closingTagPos := strings.Index(q[i+len(tag):], closingTag)
				if closingTagPos != -1 {
					fullBlock := q[i : i+len(tag)+closingTagPos+len(closingTag)]
					sb.WriteString(fullBlock)
					i += len(fullBlock)
					continue
				}
			}
		}

		// Single-line comments
		if c == '-' && i+1 < len(q) && q[i+1] == '-' {
			for i < len(q) && q[i] != '\n' && q[i] != '\r' {
				i++
			}
			// Preserve the newline character
			if i < len(q) {
				sb.WriteByte(q[i])
				i++
			}
			continue
		}

		// Multi-line comments
		if c == '/' && i+1 < len(q) && q[i+1] == '*' {
			i += 2
			for i < len(q)-1 {
				if q[i] == '*' && q[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			continue
		}

		sb.WriteByte(c)
		i++
	}

	return sb.String()
}
