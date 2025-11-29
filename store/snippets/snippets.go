package snippets

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ionut-t/perp/pkg/utils"
	"github.com/ionut-t/perp/store/common"
)

// SnippetScope defines where a snippet is stored
type SnippetScope string

const (
	ScopeGlobal SnippetScope = "global"
	ScopeServer SnippetScope = "server"
)

type Snippet struct {
	Name        string       // Filename (e.g., "user-query.sql")
	Content     string       // Full file content including metadata comments
	Query       string       // SQL query part (without metadata comments)
	Description string       // Parsed from @description comment
	Tags        []string     // Parsed from @tags comment
	CreatedAt   time.Time    // Parsed from @created comment
	UpdatedAt   time.Time    // File modification time or @updated comment
	Scope       SnippetScope // global or server-specific
}

// Implement FileItem interface (for compatibility with generic filestore)
func (s Snippet) GetName() string         { return s.Name }
func (s Snippet) GetContent() string      { return s.Content }
func (s Snippet) GetUpdatedAt() time.Time { return s.UpdatedAt }

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
	GetPath(snippet Snippet) string                        // GetPath returns the full file path for a snippet
}

func New(globalStorage, serverStorage, editor string) *store {
	// Create two FileStore instances, one for each scope
	globalFS := common.NewFileStore(
		globalStorage,
		editor,
		func(path string) (Snippet, error) {
			return loadSnippetFromFile(path, ScopeGlobal)
		},
		validateSnippetName,
		utils.GenerateUniqueName,
	)

	serverFS := common.NewFileStore(
		serverStorage,
		editor,
		func(path string) (Snippet, error) {
			return loadSnippetFromFile(path, ScopeServer)
		},
		validateSnippetName,
		utils.GenerateUniqueName,
	)

	return &store{
		globalFS:           globalFS,
		serverFS:           serverFS,
		currentSnippetName: "",
	}
}

type store struct {
	globalFS           *common.FileStore[Snippet]
	serverFS           *common.FileStore[Snippet]
	currentSnippetName string
	mu                 sync.RWMutex // Protects currentSnippetName
}

func (s *store) Load() ([]Snippet, error) {
	// Load from both stores
	globalSnippets, err := s.globalFS.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load global snippets: %w", err)
	}

	serverSnippets, err := s.serverFS.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load server snippets: %w", err)
	}

	// Combine and sort
	allSnippets := append(globalSnippets, serverSnippets...)
	slices.SortStableFunc(allSnippets, func(i, j Snippet) int {
		if i.UpdatedAt.After(j.UpdatedAt) {
			return -1
		}
		if i.UpdatedAt.Before(j.UpdatedAt) {
			return 1
		}
		return 0
	})

	// Set current snippet if not already set
	s.mu.Lock()
	if len(allSnippets) > 0 && s.currentSnippetName == "" {
		s.currentSnippetName = allSnippets[0].Name
	}
	s.mu.Unlock()

	return allSnippets, nil
}

func (s *store) Get(name string) (Snippet, error) {
	// Try global first, then server
	globalMap := s.globalFS.GetItemsMap()
	if snippet, exists := globalMap[name]; exists {
		return snippet, nil
	}

	serverMap := s.serverFS.GetItemsMap()
	if snippet, exists := serverMap[name]; exists {
		return snippet, nil
	}

	return Snippet{}, fmt.Errorf("snippet '%s' not found", name)
}

func (s *store) Create(name, content string, scope SnippetScope) error {
	// Ensure .sql extension
	if filepath.Ext(name) != ".sql" {
		name += ".sql"
	}

	// Format content with metadata if not already present
	if !strings.Contains(content, "-- @name:") {
		content = formatSnippetWithMetadata(name, content)
	}

	// Parse the metadata to create a proper snippet
	snippet, err := parseSnippetContent(name, content, scope)
	if err != nil {
		return fmt.Errorf("failed to parse snippet content: %w", err)
	}

	// Set as current if it's the first snippet
	s.mu.Lock()
	if s.currentSnippetName == "" {
		s.currentSnippetName = name
	}
	s.mu.Unlock()

	// Delegate to the appropriate FileStore
	if scope == ScopeGlobal {
		return s.globalFS.Update(snippet)
	}
	return s.serverFS.Update(snippet)
}

func (s *store) Update(snippet Snippet) error {
	// Delegate to the appropriate FileStore based on scope
	if snippet.Scope == ScopeGlobal {
		return s.globalFS.Update(snippet)
	}
	return s.serverFS.Update(snippet)
}

func (s *store) Delete(snippet Snippet) error {
	var err error
	if snippet.Scope == ScopeGlobal {
		err = s.globalFS.Delete(snippet)
	} else {
		err = s.serverFS.Delete(snippet)
	}

	if err != nil {
		return err
	}

	// Update current snippet name if the deleted one was current
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentSnippetName == snippet.Name {
		// Try to set to the first available snippet (global then server)
		globalMap := s.globalFS.GetItemsMap()
		serverMap := s.serverFS.GetItemsMap()

		if len(globalMap) > 0 {
			for name := range globalMap {
				s.currentSnippetName = name
				break
			}
		} else if len(serverMap) > 0 {
			for name := range serverMap {
				s.currentSnippetName = name
				break
			}
		} else {
			s.currentSnippetName = ""
		}
	}

	// Clean up empty storage directory for the specific scope
	if snippet.Scope == ScopeGlobal && len(s.globalFS.GetItemsMap()) == 0 {
		_ = s.globalFS.CleanEmptyStorageDir()
	} else if snippet.Scope == ScopeServer && len(s.serverFS.GetItemsMap()) == 0 {
		_ = s.serverFS.CleanEmptyStorageDir()
	}

	return nil
}

func (s *store) GetCurrentSnippet() Snippet {
	s.mu.RLock()
	currentName := s.currentSnippetName
	s.mu.RUnlock()

	// Try to find in global first, then server
	globalMap := s.globalFS.GetItemsMap()
	if snippet, exists := globalMap[currentName]; exists {
		return snippet
	}

	serverMap := s.serverFS.GetItemsMap()
	if snippet, exists := serverMap[currentName]; exists {
		return snippet
	}

	return Snippet{} // Return zero value if not found
}

func (s *store) SetCurrentSnippetName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Only set if the snippet exists in either store
	globalMap := s.globalFS.GetItemsMap()
	serverMap := s.serverFS.GetItemsMap()

	if _, exists := globalMap[name]; exists {
		s.currentSnippetName = name
	} else if _, exists := serverMap[name]; exists {
		s.currentSnippetName = name
	} else {
		s.currentSnippetName = "" // Clear if name doesn't exist
	}
}

func (s *store) GetPath(snippet Snippet) string {
	if snippet.Scope == ScopeGlobal {
		return s.globalFS.GetPath(snippet)
	}
	return s.serverFS.GetPath(snippet)
}

func (s *store) Rename(snippet *Snippet, newName string) error {
	oldName := snippet.Name

	var err error
	if snippet.Scope == ScopeGlobal {
		err = s.globalFS.Rename(snippet, newName)
	} else {
		err = s.serverFS.Rename(snippet, newName)
	}

	if err != nil {
		return err
	}

	// Update current snippet name if the renamed one was current
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.currentSnippetName == oldName {
		s.currentSnippetName = snippet.Name
	}

	return nil
}

func (s *store) Editor() string {
	// Both stores have the same editor, use global
	return s.globalFS.Editor()
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

func parseSnippetContent(name, content string, scope SnippetScope) (Snippet, error) {
	snippet := Snippet{
		Name:      name,
		Content:   content,
		UpdatedAt: time.Now(),
		Scope:     scope,
	}

	parseMetadata(&snippet)

	return snippet, nil
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

func validateSnippetName(oldName, newName string) (string, error) {
	ext := filepath.Ext(newName)
	if ext == "" {
		ext = filepath.Ext(oldName)
		newName += ext
	}
	if ext != ".sql" {
		return "", fmt.Errorf("snippet name must have a .sql extension, got %q", newName)
	}
	return newName, nil
}
