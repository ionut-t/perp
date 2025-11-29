package common

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

// FileItem represents an item that can be stored in a file
type FileItem interface {
	GetName() string
	GetContent() string
	GetUpdatedAt() time.Time
}

// LoadFileFunc is a function that loads a single file into an item
type LoadFileFunc[T FileItem] func(path string) (T, error)

// ValidateNameFunc is a function that validates and potentially modifies a filename
type ValidateNameFunc func(oldName, newName string) (string, error)

// FileStore provides common file-based CRUD operations
type FileStore[T FileItem] struct {
	storage        string
	editor         string
	LoadFromFile   LoadFileFunc[T]
	ValidateName   ValidateNameFunc
	GenerateUnique func(existingNames []string, name, oldName string) string

	// Mutable fields (protected by mu)
	mu       sync.RWMutex
	items    []T
	itemsMap map[string]T
}

// NewFileStore creates a new generic file store
func NewFileStore[T FileItem](
	storage, editor string,
	loadFromFile LoadFileFunc[T],
	validateName ValidateNameFunc,
	generateUnique func([]string, string, string) string,
) *FileStore[T] {
	return &FileStore[T]{
		items:          []T{},
		itemsMap:       make(map[string]T),
		storage:        storage,
		editor:         editor,
		LoadFromFile:   loadFromFile,
		ValidateName:   validateName,
		GenerateUnique: generateUnique,
	}
}

// Load retrieves all items from the storage directory
func (s *FileStore[T]) Load() ([]T, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.load()
	if err != nil {
		return nil, err
	}

	return s.items, nil
}

// Update writes the content of the provided item to storage
func (s *FileStore[T]) Update(item T) error {
	path := filepath.Join(s.storage, item.GetName())

	if err := os.WriteFile(path, []byte(item.GetContent()), 0o644); err != nil {
		return err
	}

	return nil
}

// Delete removes the specified item from storage
func (s *FileStore[T]) Delete(item T) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.storage, item.GetName())

	// Remove the specific file
	if err := os.Remove(path); err != nil {
		return err
	}

	// Clean up in-memory state
	delete(s.itemsMap, item.GetName())

	// Rebuild items slice from map and re-sort to maintain consistency
	s.items = s.getSortedItemsNoLock()

	return nil
}

// CleanEmptyStorageDir checks if the storage directory is empty and removes it.
// This is an optional cleanup operation that should be called explicitly by
// higher-level components when appropriate (e.g., after deleting the last item).
func (s *FileStore[T]) CleanEmptyStorageDir() error {
	entries, err := os.ReadDir(s.storage)
	if err != nil {
		return fmt.Errorf("failed to read storage directory %q: %w", s.storage, err)
	}

	// Only remove the directory if it's completely empty
	if len(entries) == 0 {
		return os.Remove(s.storage)
	}

	return nil
}

// GetItemsMap returns the map of items by name for efficient lookup
func (s *FileStore[T]) GetItemsMap() map[string]T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.itemsMap
}

// Rename changes the name of the item file to a new unique name
func (s *FileStore[T]) Rename(item *T, newName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate and potentially modify the new name
	if s.ValidateName != nil {
		validatedName, err := s.ValidateName((*item).GetName(), newName)
		if err != nil {
			return err
		}
		newName = validatedName
	}

	// Generate unique name if needed
	if s.GenerateUnique != nil {
		existingNames := make([]string, 0, len(s.items))
		for _, it := range s.items {
			if it.GetName() != (*item).GetName() {
				existingNames = append(existingNames, it.GetName())
			}
		}
		newName = s.GenerateUnique(existingNames, newName, (*item).GetName())
	}

	oldPath := filepath.Join(s.storage, (*item).GetName())
	newPath := filepath.Join(s.storage, newName)

	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}

	oldName := (*item).GetName()

	// Reload the item from the new path to get updated info
	newItem, err := s.LoadFromFile(newPath)
	if err != nil {
		// Rollback the rename if we can't load the new item
		rErr := os.Rename(newPath, oldPath)
		return errors.Join(err, rErr)
	}

	// Update in-memory structures
	delete(s.itemsMap, oldName)
	s.itemsMap[newName] = newItem

	// Rebuild the items slice from the map to ensure proper sorting
	// This is necessary because the renamed item's UpdatedAt may have changed
	s.items = make([]T, 0, len(s.itemsMap))
	for _, it := range s.itemsMap {
		s.items = append(s.items, it)
	}

	// Re-sort by modification time (most recent first)
	slices.SortStableFunc(s.items, func(i, j T) int {
		if i.GetUpdatedAt().After(j.GetUpdatedAt()) {
			return -1
		}
		if i.GetUpdatedAt().Before(j.GetUpdatedAt()) {
			return 1
		}
		return 0
	})

	// Update the item pointer to reflect the new state
	*item = newItem

	return nil
}

// Editor returns the configured editor
func (s *FileStore[T]) Editor() string {
	return s.editor
}

// load reads all items from the storage directory
func (s *FileStore[T]) load() error {
	var items []T

	err := filepath.WalkDir(s.storage, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if d.IsDir() {
			return nil
		}

		item, err := s.LoadFromFile(path)
		if err != nil {
			return err
		}

		items = append(items, item)
		s.itemsMap[item.GetName()] = item
		return nil
	})

	// Sort by modification time (most recent first)
	slices.SortStableFunc(items, func(i, j T) int {
		if i.GetUpdatedAt().After(j.GetUpdatedAt()) {
			return -1
		}

		if i.GetUpdatedAt().Before(j.GetUpdatedAt()) {
			return 1
		}

		return 0
	})

	if err == nil {
		s.items = items
	}

	return err
}

// GetAllNames returns all item names (useful for uniqueness checking)
func (s *FileStore[T]) GetAllNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, 0, len(s.items))
	for _, item := range s.items {
		names = append(names, item.GetName())
	}
	return names
}

// GetPath returns the full file path for an item
func (s *FileStore[T]) GetPath(item T) string {
	return filepath.Join(s.storage, item.GetName())
}

// getSortedItemsNoLock returns a sorted slice of items from the internal map.
func (s *FileStore[T]) getSortedItemsNoLock() []T {
	items := make([]T, 0, len(s.itemsMap))
	for _, item := range s.itemsMap {
		items = append(items, item)
	}

	slices.SortStableFunc(items, func(i, j T) int {
		if i.GetUpdatedAt().After(j.GetUpdatedAt()) {
			return -1
		}
		if i.GetUpdatedAt().Before(j.GetUpdatedAt()) {
			return 1
		}
		return 0
	})

	return items
}
