package export

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// AsJson exports the provided data as a JSON file and opens it in the configured editor.
func AsJson(storage string, data any, fileName string) (string, error) {
	records, err := load(storage, ".json")

	if err != nil {
		return "", err
	}

	fileName = generateUniqueName(fileName, records)

	if err := os.MkdirAll(storage, 0755); err != nil {
		return "", err
	}

	path := filepath.Join(storage, fileName)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
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

	if err := os.MkdirAll(storage, 0755); err != nil {
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
func PrepareCSV(queryResults []map[string]any, rows []int, all bool) ([][]string, error) {
	if len(queryResults) == 0 {
		return nil, errors.New("no query results to export")
	}

	// Create header and determine column order from the first result.
	header := make([]string, 0, len(queryResults[0]))
	for k := range queryResults[0] {
		header = append(header, k)
	}
	slices.Sort(header)

	data := [][]string{header}

	if all {
		for _, result := range queryResults {
			data = append(data, toSlice(result, header))
		}
	} else {
		for _, rowIdx := range rows {
			idx := rowIdx - 1
			if idx >= 0 && idx < len(queryResults) {
				data = append(data, toSlice(queryResults[idx], header))
			}
		}
	}

	return data, nil
}

// toSlice converts a map to a slice based on the provided header.
func toSlice(m map[string]any, header []string) []string {
	record := make([]string, len(header))
	for i, key := range header {
		if val, ok := m[key]; ok {
			record[i] = fmt.Sprintf("%v", val)
		}
	}

	return record
}
