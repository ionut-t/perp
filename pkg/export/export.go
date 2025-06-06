package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
)

// AsJson exports the provided data as a JSON file and opens it in the configured editor.
func AsJson(storage string, data any, fileName string) (string, error) {
	records, err := load(storage)

	if err != nil {
		return "", err
	}

	fileName = generateUniqueName(fileName, records)

	storagePath := filepath.Join(storage, "exports")

	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return "", err
	}

	path := filepath.Join(storagePath, fileName+".json")

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return "", err
	}

	return fileName, nil
}

func load(storage string) ([]string, error) {
	path := filepath.Join(storage, "exports")

	var records []string

	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if filepath.Ext(file.Name()) == ".json" {
			records = append(records, file.Name()[:len(file.Name())-5]) // Remove .json extension
		}
	}
	return records, nil
}

func generateUniqueName(name string, names []string) string {
	uniqueName := name
	counter := 1

	for slices.Contains(names, uniqueName) {
		uniqueName = name + "-" + strconv.Itoa(counter)
		counter++
	}

	return uniqueName
}
