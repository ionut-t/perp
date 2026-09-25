package common

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type testItem struct {
	name      string
	content   string
	updatedAt time.Time
}

func (i testItem) GetName() string         { return i.name }
func (i testItem) GetContent() string      { return i.content }
func (i testItem) GetUpdatedAt() time.Time { return i.updatedAt }

func newTestStore(storage string) *FileStore[testItem] {
	return NewFileStore(
		storage,
		"vim",
		func(path string) (testItem, error) {
			data, err := os.ReadFile(path)
			if err != nil {
				return testItem{}, err
			}
			info, err := os.Stat(path)
			if err != nil {
				return testItem{}, err
			}
			return testItem{
				name:      filepath.Base(path),
				content:   string(data),
				updatedAt: info.ModTime(),
			}, nil
		},
		nil,
		nil,
	)
}

func TestFileStore_Update_CreatesMissingStorageDir(t *testing.T) {
	t.Parallel()

	storage := filepath.Join(t.TempDir(), "items")
	s := newTestStore(storage)

	item := testItem{name: "foo.txt", content: "hello", updatedAt: time.Now()}
	if err := s.Update(item); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(storage, "foo.txt"))
	if err != nil {
		t.Fatalf("expected file to be written: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("expected content 'hello', got %q", string(data))
	}
}

func TestFileStore_Update_CreatesNestedMissingStorageDir(t *testing.T) {
	t.Parallel()

	storage := filepath.Join(t.TempDir(), "server", "nested", "items")
	s := newTestStore(storage)

	if err := s.Update(testItem{name: "foo.txt", content: "hello", updatedAt: time.Now()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(storage, "foo.txt")); err != nil {
		t.Fatalf("expected file to be written: %v", err)
	}
}

func TestFileStore_Update_ExistingDirOverwritesFile(t *testing.T) {
	t.Parallel()

	storage := t.TempDir()
	s := newTestStore(storage)

	if err := s.Update(testItem{name: "foo.txt", content: "first", updatedAt: time.Now()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Update(testItem{name: "foo.txt", content: "second", updatedAt: time.Now()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(storage, "foo.txt"))
	if err != nil {
		t.Fatalf("expected file to be written: %v", err)
	}
	if string(data) != "second" {
		t.Errorf("expected content 'second', got %q", string(data))
	}
}

func TestFileStore_Update_EmptyStorageReturnsError(t *testing.T) {
	t.Parallel()

	s := newTestStore("")

	err := s.Update(testItem{name: "foo.txt", content: "hello", updatedAt: time.Now()})
	if err == nil {
		t.Fatal("expected an error when the storage directory is not configured")
	}

	// The item must not leak into the working directory.
	if _, statErr := os.Stat("foo.txt"); statErr == nil {
		t.Fatal("expected no file to be written to the working directory")
	}
}

func TestFileStore_Update_RecreatesStorageDirAfterCleanup(t *testing.T) {
	t.Parallel()

	storage := filepath.Join(t.TempDir(), "items")
	s := newTestStore(storage)

	item := testItem{name: "foo.txt", content: "hello", updatedAt: time.Now()}
	if err := s.Update(item); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := s.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Delete(item); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.CleanEmptyStorageDir(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(storage); !os.IsNotExist(err) {
		t.Fatalf("expected storage dir to be removed, got %v", err)
	}

	// Writing again must recreate the directory instead of failing.
	if err := s.Update(item); err != nil {
		t.Fatalf("unexpected error after cleanup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(storage, "foo.txt")); err != nil {
		t.Fatalf("expected file to be written: %v", err)
	}
}
