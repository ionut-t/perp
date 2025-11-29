package export

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ionut-t/perp/pkg/utils"
	"github.com/ionut-t/perp/store/common"
)

type Record struct {
	Name      string
	Content   string
	UpdatedAt time.Time
}

// Implement FileItem interface
func (r Record) GetName() string         { return r.Name }
func (r Record) GetContent() string      { return r.Content }
func (r Record) GetUpdatedAt() time.Time { return r.UpdatedAt }

type Store interface {
	Load() ([]Record, error)                     // Load retrieves all export records from the configured storage directory.
	Update(record Record) error                  // Update writes the content of the provided record to the storage directory.
	Delete(record Record) error                  // Delete removes the specified record from the storage directory.
	Rename(record *Record, newName string) error // Rename changes the name of the record file to a new unique name.
	Editor() string                              // Editor returns the configured editor for opening records.
	GetCurrentRecord() Record                    // GetCurrentRecord returns the currently selected record.
	SetCurrentRecordName(name string)            // SetCurrentRecordName sets the name of the currently selected record.
	GetPath(record Record) string                // GetPath returns the full file path for a record.
}

func New(storage, editor string) *store {
	return &store{
		FileStore: common.NewFileStore(
			storage,
			editor,
			loadRecordFromFile,
			validateRecordName,
			utils.GenerateUniqueName,
		),
	}
}

type store struct {
	*common.FileStore[Record]
	currentRecordName string
}

// Load retrieves all records and sets the current record if needed
func (s *store) Load() ([]Record, error) {
	records, err := s.FileStore.Load()
	if err != nil {
		return nil, err
	}

	// Set the first record as current if we don't have one yet
	if len(records) > 0 && s.currentRecordName == "" {
		s.currentRecordName = records[0].Name
	}

	return records, nil
}

// Delete removes the record and optionally cleans up empty directories
func (s *store) Delete(record Record) error {
	if err := s.FileStore.Delete(record); err != nil {
		return err
	}

	// Update current record if we just deleted it
	itemsMap := s.GetItemsMap()
	if s.currentRecordName == record.Name {
		// Try to set to the first available record
		for name := range itemsMap {
			s.currentRecordName = name
			break
		}
		// If no records left, clear current
		if len(itemsMap) == 0 {
			s.currentRecordName = ""
		}
	}

	// Clean up empty storage directory
	if len(itemsMap) == 0 {
		// Ignore cleanup errors - file deletion succeeded
		_ = s.CleanEmptyStorageDir()
	}

	return nil
}

// Rename changes the record name and updates current if needed
func (s *store) Rename(record *Record, newName string) error {
	oldName := record.Name
	if err := s.FileStore.Rename(record, newName); err != nil {
		return err
	}

	// Update current record name if we renamed the current record
	if s.currentRecordName == oldName {
		s.currentRecordName = record.Name
	}

	return nil
}

// GetCurrentRecord returns the currently selected record
func (s *store) GetCurrentRecord() Record {
	itemsMap := s.GetItemsMap()
	return itemsMap[s.currentRecordName]
}

// SetCurrentRecordName sets the name of the currently selected record
func (s *store) SetCurrentRecordName(name string) {
	itemsMap := s.GetItemsMap()
	if _, exists := itemsMap[name]; exists {
		s.currentRecordName = name
	} else {
		s.currentRecordName = ""
	}
}

// GetPath returns the full file path for a record
func (s *store) GetPath(record Record) string {
	return s.FileStore.GetPath(record)
}

// loadRecordFromFile loads a single record from a file
func loadRecordFromFile(path string) (Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}

	content := strings.TrimSuffix(string(data), "\n")

	fileInfo, err := os.Stat(path)
	if err != nil {
		return Record{}, err
	}

	return Record{
		Name:      filepath.Base(path),
		Content:   content,
		UpdatedAt: fileInfo.ModTime(),
	}, nil
}

// validateRecordName validates and ensures proper extension for record names
func validateRecordName(oldName, newName string) (string, error) {
	ext := filepath.Ext(newName)

	if ext == "" {
		ext = filepath.Ext(oldName)
		newName += ext
	}

	if ext != filepath.Ext(oldName) {
		return "", errors.New("cannot change file extension when renaming record")
	}

	return newName, nil
}
