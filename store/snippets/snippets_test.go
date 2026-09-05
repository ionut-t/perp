package snippets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestStore returns a store whose snippet directories do not exist yet,
// which is the state right after a fresh install or after the last snippet of
// a scope has been deleted.
func newTestStore(t *testing.T) (*store, string, string) {
	t.Helper()

	root := t.TempDir()
	globalPath := filepath.Join(root, "snippets")
	serverPath := filepath.Join(root, "my-server", "snippets")

	return New(globalPath, serverPath, "vim"), globalPath, serverPath
}

func TestStore_Create_ServerScope_MissingDirectory(t *testing.T) {
	t.Parallel()

	s, _, serverPath := newTestStore(t)

	if err := s.Create("users", "SELECT * FROM users;", ScopeServer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(serverPath, "users.sql"))
	if err != nil {
		t.Fatalf("expected snippet file to be written: %v", err)
	}
	if !strings.Contains(string(data), "SELECT * FROM users;") {
		t.Errorf("expected query in file, got %q", string(data))
	}
}

func TestStore_Create_GlobalScope_MissingDirectory(t *testing.T) {
	t.Parallel()

	s, globalPath, _ := newTestStore(t)

	if err := s.Create("users", "SELECT * FROM users;", ScopeGlobal); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(globalPath, "users.sql")); err != nil {
		t.Fatalf("expected snippet file to be written: %v", err)
	}
}

func TestStore_Create_LoadsAfterCreation(t *testing.T) {
	t.Parallel()

	s, _, _ := newTestStore(t)

	if err := s.Create("global-one", "SELECT 1;", ScopeGlobal); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Create("server-one", "SELECT 2;", ScopeServer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snippets, err := s.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snippets) != 2 {
		t.Fatalf("expected 2 snippets, got %d", len(snippets))
	}

	byName := make(map[string]Snippet, len(snippets))
	for _, snippet := range snippets {
		byName[snippet.Name] = snippet
	}

	global, ok := byName["global-one.sql"]
	if !ok {
		t.Fatal("expected global-one.sql to be loaded")
	}
	if global.Scope != ScopeGlobal {
		t.Errorf("expected global scope, got %q", global.Scope)
	}
	if global.Query != "SELECT 1;" {
		t.Errorf("expected query 'SELECT 1;', got %q", global.Query)
	}

	server, ok := byName["server-one.sql"]
	if !ok {
		t.Fatal("expected server-one.sql to be loaded")
	}
	if server.Scope != ScopeServer {
		t.Errorf("expected server scope, got %q", server.Scope)
	}
	if server.Query != "SELECT 2;" {
		t.Errorf("expected query 'SELECT 2;', got %q", server.Query)
	}
}

func TestStore_Create_AddsSQLExtensionAndMetadata(t *testing.T) {
	t.Parallel()

	s, _, serverPath := newTestStore(t)

	if err := s.Create("users", "SELECT * FROM users;", ScopeServer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(serverPath, "users.sql"))
	if err != nil {
		t.Fatalf("expected snippet file to be written: %v", err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "-- @name: users\n") {
		t.Errorf("expected metadata header with the snippet name, got %q", content)
	}
	for _, key := range []string{"-- @description:", "-- @tags:", "-- @created:", "-- @updated:"} {
		if !strings.Contains(content, key) {
			t.Errorf("expected %q in metadata header, got %q", key, content)
		}
	}
}

func TestStore_Create_KeepsExistingMetadata(t *testing.T) {
	t.Parallel()

	s, _, serverPath := newTestStore(t)

	content := "-- @name: custom\n-- @description: my description\n\nSELECT 1;\n"
	if err := s.Create("users.sql", content, ScopeServer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(serverPath, "users.sql"))
	if err != nil {
		t.Fatalf("expected snippet file to be written: %v", err)
	}
	if string(data) != content {
		t.Errorf("expected content to be written as-is, got %q", string(data))
	}
}

func TestStore_Create_SetsCurrentSnippet(t *testing.T) {
	t.Parallel()

	s, _, _ := newTestStore(t)

	if err := s.Create("first", "SELECT 1;", ScopeServer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Create("second", "SELECT 2;", ScopeServer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := s.Load(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := s.GetCurrentSnippet().Name; got != "first.sql" {
		t.Errorf("expected current snippet to be 'first.sql', got %q", got)
	}
}

func TestStore_Create_AfterDeletingLastSnippet(t *testing.T) {
	t.Parallel()

	s, _, serverPath := newTestStore(t)

	if err := s.Create("users", "SELECT 1;", ScopeServer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	snippets, err := s.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snippets) != 1 {
		t.Fatalf("expected 1 snippet, got %d", len(snippets))
	}

	// Deleting the last snippet of a scope removes its directory.
	if err := s.Delete(snippets[0]); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(serverPath); !os.IsNotExist(err) {
		t.Fatalf("expected server snippets dir to be removed, got %v", err)
	}

	if err := s.Create("users", "SELECT 2;", ScopeServer); err != nil {
		t.Fatalf("unexpected error after deleting the last snippet: %v", err)
	}
	if _, err := os.Stat(filepath.Join(serverPath, "users.sql")); err != nil {
		t.Fatalf("expected snippet file to be written: %v", err)
	}
}

func TestStore_Create_WithoutServerStorageReturnsError(t *testing.T) {
	t.Parallel()

	// No server is connected, so the server-scoped storage path is empty.
	s := New(filepath.Join(t.TempDir(), "snippets"), "", "vim")

	if err := s.Create("users", "SELECT 1;", ScopeServer); err == nil {
		t.Fatal("expected an error when the server snippets path is not configured")
	}

	if got := s.GetCurrentSnippet().Name; got != "" {
		t.Errorf("expected no current snippet after a failed create, got %q", got)
	}
	if _, err := os.Stat("users.sql"); err == nil {
		t.Fatal("expected no snippet file to be written to the working directory")
	}
}
