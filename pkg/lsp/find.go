package lsp

import "os/exec"

// binaryNames contains the known names for the postgres-language-server binary.
var binaryNames = []string{"postgres-language-server", "pglsp"}

// FindBinary searches PATH for a known postgres-language-server binary.
// Returns the full path and true if found.
func FindBinary() (string, bool) {
	for _, name := range binaryNames {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, true
		}
	}
	return "", false
}
