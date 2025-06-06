package export

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Record struct {
	Name      string
	Content   string
	UpdatedAt time.Time
	Extension string
	Path      string
}

type Store interface {
	Load() ([]Record, error)                     // Load retrieves all export records from the configured storage directory.
	Update(record Record) error                  // Update writes the content of the provided record to the storage directory.
	Delete(record Record) error                  // Delete removes the specified record from the storage directory.
	Rename(record *Record, newName string) error // Rename changes the name of the record file to a new unique name.
	Editor() string                              // Editor returns the configured editor for opening records.
	GetCurrentRecord() Record                    // GetCurrentRecord returns the currently selected record.
	SetCurrentRecordName(name string)            // SetCurrentRecordName sets the name of the currently selected record.
}

func New(storage, editor string) Store {
	return &store{
		records:           []Record{},
		recordsMap:        make(map[string]Record),
		storage:           storage,
		editor:            editor,
		currentRecordName: "",
	}
}

type store struct {
	records           []Record
	currentRecordName string
	recordsMap        map[string]Record
	storage           string
	editor            string
}

func (s *store) Load() ([]Record, error) {
	err := s.load()

	if err != nil {
		return nil, err
	}

	if len(s.records) > 0 && s.currentRecordName == "" {
		s.currentRecordName = s.records[0].Name
	}

	return s.records, nil
}

func (s *store) Update(record Record) error {
	path := filepath.Join(s.storage, "exports", record.Name+record.Extension)

	if err := os.WriteFile(path, []byte(record.Content), 0644); err != nil {
		return err
	}

	return nil
}

func (s *store) Delete(record Record) error {
	path := filepath.Join(s.storage, "exports", record.Name+record.Extension)

	if err := os.Remove(path); err != nil {
		return err
	}

	return nil
}

func (s *store) GetCurrentRecord() Record {
	return s.recordsMap[s.currentRecordName]
}

func (s *store) SetCurrentRecordName(name string) {
	s.currentRecordName = name

	if _, exists := s.recordsMap[name]; !exists {
		s.currentRecordName = ""
	}
}
func (s *store) Rename(record *Record, newName string) error {
	uniqueName := s.generateUniqueName(newName)

	oldPath := record.Path
	newPath := filepath.Join(s.storage, "exports", uniqueName+record.Extension)

	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}

	oldName := record.Name

	delete(s.recordsMap, oldName)

	record.Name = uniqueName
	record.Path = newPath

	s.recordsMap[uniqueName] = *record

	for i := range s.records {
		if s.records[i].Name == oldName {
			s.records[i] = *record
			break
		}
	}

	if s.currentRecordName == oldName {
		s.currentRecordName = uniqueName
	}

	return nil
}

func (s *store) Editor() string {
	return s.editor
}

func (s store) generateUniqueName(name string) string {
	originalName := name
	counter := 1

	for _, record := range s.records {
		if strings.EqualFold(record.Name, name) {
			name = originalName + "-" + strconv.Itoa(counter)
			counter++
			continue
		}
	}

	return name
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

func (s *store) load() error {
	path := filepath.Join(s.storage, "exports")

	var records []Record

	err := filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
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
		s.recordsMap[record.Name] = record
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

	if err == nil {
		s.records = records
	}

	return err
}
