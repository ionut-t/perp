package snippets

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ionut-t/perp/pkg/utils"
)

// SnippetScope defines where a snippet is stored
type SnippetScope string

const (
	ScopeGlobal SnippetScope = "global"
	ScopeServer SnippetScope = "server"
)

type Snippet struct {
	Name        string       // Filename (e.g., "user-query.sql")
	Path        string       // Full file path
	Content     string       // Full file content including metadata comments
	Query       string       // SQL query part (without metadata comments)
	Description string       // Parsed from @description comment
	Tags        []string     // Parsed from @tags comment
	CreatedAt   time.Time    // Parsed from @created comment
	UpdatedAt   time.Time    // File modification time or @updated comment
	Scope       SnippetScope // global or server-specific
}

type Store interface {
	Load() ([]Snippet, error)                              // Load retrieves all snippets from both global and server-specific directories
	Get(name string) (Snippet, error)                      // Get retrieves a specific snippet by name
	Create(name, content string, scope SnippetScope) error // Create creates a new snippet in the specified scope
	Update(snippet Snippet) error                          // Update writes the content of the provided snippet
	Delete(snippet Snippet) error                          // Delete removes the specified snippet
	Rename(snippet *Snippet, newName string) error         // Rename changes the name of the snippet file
	Editor() string                                        // Editor returns the configured editor
	GetCurrentSnippet() Snippet                            // GetCurrentSnippet returns the currently selected snippet
	SetCurrentSnippetName(name string)                     // SetCurrentSnippetName sets the name of the currently selected snippet
}

func New(globalStorage, serverStorage, editor string) Store {
	return &store{
		snippets:           []Snippet{},
		snippetsMap:        make(map[string]Snippet),
		globalStorage:      globalStorage,
		serverStorage:      serverStorage,
		editor:             editor,
		currentSnippetName: "",
	}
}

type store struct {
	snippets           []Snippet
	currentSnippetName string
	snippetsMap        map[string]Snippet
	globalStorage      string
	serverStorage      string
	editor             string
}

func (s *store) Load() ([]Snippet, error) {
	err := s.load()
	if err != nil {
		return nil, err
	}

	if len(s.snippets) > 0 && s.currentSnippetName == "" {
		s.currentSnippetName = s.snippets[0].Name
	}

	return s.snippets, nil
}

func (s *store) Get(name string) (Snippet, error) {
	snippet, exists := s.snippetsMap[name]
	if !exists {
		return Snippet{}, fmt.Errorf("snippet '%s' not found", name)
	}
	return snippet, nil
}

func (s *store) Create(name, content string, scope SnippetScope) error {
	// Ensure .sql extension
	if filepath.Ext(name) != ".sql" {
		name += ".sql"
	}

	// Determine storage directory based on scope
	var storage string
	if scope == ScopeGlobal {
		storage = s.globalStorage
	} else {
		storage = s.serverStorage
	}

	// Ensure directory exists
	if err := os.MkdirAll(storage, 0o755); err != nil {
		return err
	}

	// Read existing snippet names from filesystem
	existingNames, err := s.loadNamesFromDirectory(storage)
	if err != nil {
		return err
	}

	name = utils.GenerateUniqueName(existingNames, name, "")

	// Format content with metadata if not already present
	if !strings.Contains(content, "-- @name:") {
		content = formatSnippetWithMetadata(name, content)
	}

	path := filepath.Join(storage, name)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}

	return nil
}

func (s *store) Update(snippet Snippet) error {
	if err := os.WriteFile(snippet.Path, []byte(snippet.Content), 0o644); err != nil {
		return err
	}

	return nil
}

func (s *store) Delete(snippet Snippet) error {
	if err := os.Remove(snippet.Path); err != nil {
		return err
	}

	// Clean up from in-memory state
	delete(s.snippetsMap, snippet.Name)
	for i, snip := range s.snippets {
		if snip.Name == snippet.Name {
			s.snippets = append(s.snippets[:i], s.snippets[i+1:]...)
			break
		}
	}

	if s.currentSnippetName == snippet.Name && len(s.snippets) > 0 {
		s.currentSnippetName = s.snippets[0].Name
	}

	return nil
}

func (s *store) GetCurrentSnippet() Snippet {
	return s.snippetsMap[s.currentSnippetName]
}

func (s *store) SetCurrentSnippetName(name string) {
	s.currentSnippetName = name

	if _, exists := s.snippetsMap[name]; !exists {
		s.currentSnippetName = ""
	}
}

func (s *store) Rename(snippet *Snippet, newName string) error {
	if filepath.Ext(newName) != ".sql" {
		newName += ".sql"
	}

	// Determine storage directory based on scope
	var storage string
	if snippet.Scope == ScopeGlobal {
		storage = s.globalStorage
	} else {
		storage = s.serverStorage
	}

	// Read existing snippet names from filesystem
	existingNames, err := s.loadNamesFromDirectory(storage)
	if err != nil {
		return err
	}

	uniqueName := utils.GenerateUniqueName(existingNames, newName, snippet.Name)

	oldPath := snippet.Path
	newPath := filepath.Join(storage, uniqueName)

	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}

	oldName := snippet.Name

	delete(s.snippetsMap, oldName)

	snippet.Name = uniqueName
	snippet.Path = newPath

	s.snippetsMap[uniqueName] = *snippet

	for i := range s.snippets {
		if s.snippets[i].Name == oldName {
			s.snippets[i] = *snippet
			break
		}
	}

	if s.currentSnippetName == oldName {
		s.currentSnippetName = uniqueName
	}

	return nil
}

func (s *store) Editor() string {
	return s.editor
}

func (s *store) loadNamesFromDirectory(directory string) ([]string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sql" {
			names = append(names, entry.Name())
		}
	}

	return names, nil
}

func (s *store) load() error {
	var snippets []Snippet

	// Load global snippets
	if s.globalStorage != "" {
		if err := s.loadFromDirectory(s.globalStorage, ScopeGlobal, &snippets); err != nil {
			return err
		}
	}

	// Load server-specific snippets
	if s.serverStorage != "" {
		if err := s.loadFromDirectory(s.serverStorage, ScopeServer, &snippets); err != nil {
			return err
		}
	}

	// Sort by modification time (most recent first)
	slices.SortStableFunc(snippets, func(i, j Snippet) int {
		if i.UpdatedAt.After(j.UpdatedAt) {
			return -1
		}

		if i.UpdatedAt.Before(j.UpdatedAt) {
			return 1
		}

		return 0
	})

	s.snippets = snippets

	return nil
}

func (s *store) loadFromDirectory(directory string, scope SnippetScope, snippets *[]Snippet) error {
	err := filepath.WalkDir(directory, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".sql" {
			return nil
		}

		snippet, err := loadSnippetFromFile(path, scope)
		if err != nil {
			return err
		}

		*snippets = append(*snippets, snippet)
		s.snippetsMap[snippet.Name] = snippet
		return nil
	})

	return err
}

func loadSnippetFromFile(path string, scope SnippetScope) (Snippet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Snippet{}, err
	}

	content := string(data)
	fileInfo, err := os.Stat(path)
	if err != nil {
		return Snippet{}, err
	}

	snippet := Snippet{
		Name:      filepath.Base(path),
		Path:      path,
		Content:   content,
		UpdatedAt: fileInfo.ModTime(),
		Scope:     scope,
	}

	parseMetadata(&snippet)

	return snippet, nil
}

func parseMetadata(snippet *Snippet) {
	lines := strings.Split(snippet.Content, "\n")
	var queryLines []string
	inMetadata := true

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inMetadata && strings.HasPrefix(trimmed, "-- @") {
			// Parse metadata comment
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(strings.TrimPrefix(parts[0], "-- @"))
			value := strings.TrimSpace(parts[1])

			switch key {
			case "name":
				// We already have the filename, but could use this as display name if different
			case "description":
				snippet.Description = value
			case "tags":
				snippet.Tags = strings.Split(value, ",")
				for i := range snippet.Tags {
					snippet.Tags[i] = strings.TrimSpace(snippet.Tags[i])
				}
			case "created":
				if t, err := time.Parse(time.RFC3339, value); err == nil {
					snippet.CreatedAt = t
				}
			case "updated":
				if t, err := time.Parse(time.RFC3339, value); err == nil {
					snippet.UpdatedAt = t
				}
			}
		} else if strings.HasPrefix(trimmed, "--") && inMetadata {
			// Regular comment, still in header
			continue
		} else if trimmed == "" && inMetadata {
			// Empty line, still in header
			continue
		} else {
			// Non-comment, non-empty line - end of metadata
			inMetadata = false
			queryLines = append(queryLines, line)
		}
	}

	snippet.Query = strings.TrimSpace(strings.Join(queryLines, "\n"))
}

func formatSnippetWithMetadata(name, query string) string {
	now := time.Now().Format(time.RFC3339)
	displayName := strings.TrimSuffix(name, ".sql")

	return fmt.Sprintf(`-- @name: %s
-- @description:
-- @tags:
-- @created: %s
-- @updated: %s

%s
`, displayName, now, now, strings.TrimSpace(query))
}
