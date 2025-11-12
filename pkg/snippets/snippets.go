package snippets

import (
	"os"
	"path/filepath"
)

const snippetsDirectory = "snippets"

// GetGlobalSnippetsPath returns the global snippets directory path
func GetGlobalSnippetsPath(storageRoot string) string {
	return filepath.Join(storageRoot, snippetsDirectory)
}

// GetServerSnippetsPath returns the server-specific snippets directory path
func GetServerSnippetsPath(storageRoot, serverName string) string {
	if serverName == "" {
		return ""
	}
	return filepath.Join(storageRoot, serverName, snippetsDirectory)
}

// EnsureSnippetsDirectories creates the necessary snippet directories if they don't exist
func EnsureSnippetsDirectories(globalPath, serverPath string) error {
	if globalPath != "" {
		if err := os.MkdirAll(globalPath, 0o755); err != nil {
			return err
		}
	}

	if serverPath != "" {
		if err := os.MkdirAll(serverPath, 0o755); err != nil {
			return err
		}
	}

	return nil
}
