package export

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/ionut-t/perp/pkg/db"
)

// AsJson exports the provided data as a JSON file and opens it in the configured editor.
func AsJson(storage string, data any, fileName string) (string, error) {
	records, err := load(storage, ".json")
	if err != nil {
		return "", err
	}

	fileName = generateUniqueName(fileName, records)

	if err := os.MkdirAll(storage, 0o755); err != nil {
		return "", err
	}

	path := filepath.Join(storage, fileName)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, jsonData, 0o644); err != nil {
		return "", err
	}

	return fileName, nil
}

// AsCsv exports the provided data as a CSV file.
func AsCsv(storage string, data [][]string, fileName string) (string, error) {
	records, err := load(storage, ".csv")
	if err != nil {
		return "", err
	}

	fileName = generateUniqueName(fileName, records)

	if err := os.MkdirAll(storage, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(storage, fileName)

	file, err := os.Create(path)
	if err != nil {
		return "", err
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing file: %v\n", err)
		}
	}()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, record := range data {
		if err := writer.Write(record); err != nil {
			return "", err
		}
	}

	return fileName, nil
}

func load(path string, ext string) ([]string, error) {
	var records []string

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return records, nil
	}

	files, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return records, nil
		}

		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if filepath.Ext(file.Name()) != ext {
			continue
		}

		name := file.Name()
		name = name[:len(name)-len(ext)]
		records = append(records, name)
	}
	return records, nil
}

func generateUniqueName(name string, names []string) string {
	ext := filepath.Ext(name)
	name = strings.TrimSuffix(name, ext)

	uniqueName := name
	counter := 1

	for slices.Contains(names, uniqueName) {
		uniqueName = name + "-" + strconv.Itoa(counter)
		counter++
	}

	return uniqueName + ext
}

// PrepareJSON processes query results and selected rows for export.
func PrepareJSON(queryResults []map[string]any, rows []int, all bool) (any, error) {
	if queryResults != nil {
		var data any
		if len(rows) > 1 {
			data = make([]map[string]any, 0)

			for _, rowIdx := range rows {
				idx := rowIdx - 1
				if idx >= 0 && idx < len(queryResults) {
					data = append(data.([]map[string]any), queryResults[idx])
				}
			}
		} else if len(rows) == 1 {
			idx := rows[0] - 1
			if idx >= 0 && idx < len(queryResults) {
				data = queryResults[idx]
			}
		}

		if all {
			data = make([]map[string]any, 0)
			data = append(data.([]map[string]any), queryResults...)
		}

		return data, nil
	}

	return nil, errors.New("no query results to export")
}

// PrepareCSV processes query results and selected rows for CSV export.
// The columns argument carries the column order returned by the query and is
// used as the CSV header. When it is empty (or does not match the results), the
// column names are derived from the first result and sorted alphabetically,
// since a result row is a map and therefore carries no order of its own.
func PrepareCSV(
	queryResults []map[string]any,
	columns []string,
	columnTypes map[string]uint32,
	rows []int,
	all bool,
) ([][]string, error) {
	if len(queryResults) == 0 {
		return nil, errors.New("no query results to export")
	}

	header := buildHeader(queryResults[0], columns)

	data := [][]string{header}

	if all {
		for _, result := range queryResults {
			data = append(data, toSlice(result, header, columnTypes))
		}
	} else {
		for _, rowIdx := range rows {
			idx := rowIdx - 1
			if idx >= 0 && idx < len(queryResults) {
				data = append(data, toSlice(queryResults[idx], header, columnTypes))
			}
		}
	}

	return data, nil
}

// buildHeader returns the CSV header, preferring the column order reported by
// the query and falling back to the sorted keys of a result row.
func buildHeader(result map[string]any, columns []string) []string {
	if len(columns) > 0 {
		header := make([]string, 0, len(columns))
		for _, column := range columns {
			if _, ok := result[column]; ok {
				header = append(header, column)
			}
		}

		if len(header) == len(result) {
			return header
		}
	}

	header := make([]string, 0, len(result))
	for k := range result {
		header = append(header, k)
	}
	slices.Sort(header)

	return header
}

// toSlice converts a map to a slice based on the provided header.
func toSlice(m map[string]any, header []string, columnTypes map[string]uint32) []string {
	record := make([]string, len(header))
	for i, key := range header {
		if val, ok := m[key]; ok {
			record[i] = formatValue(val, columnTypes[key])
		}
	}

	return record
}

// formatValue renders a value for a CSV field.
//
// NULL is written as an empty field (the convention used by COPY ... TO CSV)
// rather than Go's "<nil>". Other values go through db.FormatValue, so bytea,
// json, numeric and array columns are written the way PostgreSQL writes them
// instead of dumping their Go representation.
//
// Floats and timestamps are the exception: db.FormatValue renders them for
// display in the results table, which pads floats to six decimals and prints
// timestamps in Go's own layout. A CSV file is read back by other tools, so
// they are written in a round-trippable form here.
//
// A zero OID (the psql command path, whose values are already formatted by
// db.ExtractPsqlResults) matches no type and leaves the value as it is.
func formatValue(val any, oid uint32) string {
	if isNil(val) {
		return ""
	}

	switch v := val.(type) {
	case time.Time:
		return v.Format(time.RFC3339Nano)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}

	return fmt.Sprintf("%v", db.FormatValue(val, oid))
}

// isNil reports whether val is nil, including a typed nil held in an interface.
func isNil(val any) bool {
	if val == nil {
		return true
	}

	switch v := reflect.ValueOf(val); v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
