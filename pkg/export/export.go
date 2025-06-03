package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ionut-t/perp/internal/config"
)

type Record struct {
	Name      string
	Content   string
	UpdatedAt time.Time
	Extension string
	Path      string
}

// AsJson exports the provided data as a JSON file and opens it in the configured editor.
// It returns a tea.Cmd that can be executed to open the file, or an error if something goes wrong.
func AsJson(data any, fileName string) error {
	storage, err := config.GetStorage()

	if err != nil {
		return err
	}

	storagePath := filepath.Join(storage, "exports")

	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return err
	}

	path := filepath.Join(storagePath, fileName+".json")

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return err
	}

	return nil
}

// Load retrieves all export records from the configured storage directory.
func Load() ([]Record, error) {
	storage, err := config.GetStorage()
	if err != nil {
		return []Record{}, err
	}

	path := filepath.Join(storage, "exports")

	var records []Record

	err = filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if d.IsDir() {
			return nil
		}

		record, err := loadRecordFromFile(path)
		if err != nil {
			return err
		}

		records = append(records, record)
		return nil
	})

	slices.SortStableFunc(records, func(i, j Record) int {
		if i.UpdatedAt.After(j.UpdatedAt) {
			return -1
		}

		if i.UpdatedAt.Before(j.UpdatedAt) {
			return 1
		}

		return 0
	})

	return records, err
}

func Update(record Record) error {
	storage, err := config.GetStorage()
	if err != nil {
		return err
	}

	path := filepath.Join(storage, "exports", record.Name+record.Extension)

	if err := os.WriteFile(path, []byte(record.Content), 0644); err != nil {
		return err
	}

	return nil
}

func loadRecordFromFile(path string) (Record, error) {
	data, err := os.ReadFile(path)

	if err != nil {
		return Record{}, err
	}

	content := strings.TrimSuffix(string(data), "\n")

	extension := filepath.Ext(path)

	fileInfo, err := os.Stat(path)

	if err != nil {
		return Record{}, err
	}

	return Record{
		Name:      strings.TrimSuffix(filepath.Base(path), extension),
		Content:   content,
		Extension: extension,
		UpdatedAt: fileInfo.ModTime(),
		Path:      path,
	}, nil
}
