package export

import (
	"encoding/json"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ionut-t/perp/internal/config"
)

type ClosedMsg struct{}

// AsJson exports the provided data as a JSON file and opens it in the configured editor.
// It returns a tea.Cmd that can be executed to open the file, or an error if something goes wrong.
func AsJson(data any) (tea.Cmd, error) {
	editor := config.GetEditor()

	storage, err := config.GetStorage()

	if err != nil {
		return nil, err
	}

	uniqFile, err := os.CreateTemp(storage, "perp-export-*.json")
	if err != nil {
		return nil, err
	}
	defer uniqFile.Close()
	path := uniqFile.Name()

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return nil, err
	}

	execCmd := tea.ExecProcess(exec.Command(editor, path), func(error) tea.Msg {
		return ClosedMsg{}
	})

	return execCmd, nil
}
