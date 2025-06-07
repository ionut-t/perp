package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/ionut-t/perp/ui/styles"
)

type ExportMsg struct {
	Rows     []int
	All      bool
	FileName string
}

type EditorChangedMsg struct {
	Editor string
}

type LLMUseDatabaseSchemaMsg struct {
	Enabled bool
}

type CancelMsg struct{}

type QuitMsg struct{}

type ErrorMsg struct {
	Err error
}

type Model struct {
	input *huh.Input
}

func New() Model {
	cmdInput := huh.NewInput().Prompt(": ")
	cmdInput.WithTheme(styles.ThemeCatppuccin())

	return Model{
		input: cmdInput,
	}
}

func (c Model) Reset() {
	empty := ""
	c.input.Value(&empty)
}

func (c Model) Init() tea.Cmd {
	return cursor.Blink
}

func (c Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.input.WithWidth(msg.Width)

	case tea.KeyMsg:
		return c.handleCmdRunner(msg)
	}

	cmdModel, cmd := c.input.Update(msg)
	c.input = cmdModel.(*huh.Input)

	return c, cmd
}

func (c Model) Focus() tea.Cmd {
	c.input.Focus()
	return c.Init()
}

func (c Model) View() string {
	return c.input.View()
}

func (c Model) handleCmdRunner(msg tea.KeyMsg) (Model, tea.Cmd) {
	c.input.Focus()

	switch msg.Type {
	case tea.KeyEsc:
		empty := ""
		c.input.Value(&empty)
		return c, dispatch(CancelMsg{})

	case tea.KeyEnter:
		cmdValue := c.input.GetValue().(string)
		cmdValue = strings.TrimSpace(cmdValue)

		if cmdValue == "" {
			return c, nil
		}

		if cmdValue == "q" {
			return c, dispatch(QuitMsg{})
		}

		if strings.HasPrefix(cmdValue, "export") {
			return c.handleExport()
		}

		if strings.HasPrefix(cmdValue, "set-editor") {
			return c.handleEditorSetCmd(cmdValue)
		}

		if strings.HasPrefix(cmdValue, "llm-schema") {
			return c.handleLLMDatabaseSchema(cmdValue)
		}
	}

	cmdModel, cmd := c.input.Update(msg)
	c.input = cmdModel.(*huh.Input)

	return c, cmd
}

func (c Model) handleExport() (Model, tea.Cmd) {
	value := c.input.GetValue().(string)

	rows, all, fileName, err := parseExportCommand(value)

	if err != nil {
		return c, dispatch(ErrorMsg{Err: err})
	}

	if len(rows) == 0 && !all {
		return c, dispatch(ErrorMsg{Err: fmt.Errorf("no rows specified")})
	}

	return c, dispatch(ExportMsg{
		Rows:     rows,
		All:      all,
		FileName: fileName,
	})
}

func (c Model) handleEditorSetCmd(cmdValue string) (Model, tea.Cmd) {
	editor := strings.TrimSpace(strings.TrimPrefix(cmdValue, "set-editor"))

	if editor == "" {
		return c, dispatch(ErrorMsg{Err: errors.New("no editor specified")})
	}

	empty := ""
	c.input.Value(&empty)

	return c, dispatch(EditorChangedMsg{Editor: editor})
}

func (c Model) handleLLMDatabaseSchema(cmdValue string) (Model, tea.Cmd) {
	value := strings.TrimSpace(strings.TrimPrefix(cmdValue, "llm-schema"))

	var enabled bool

	switch value {
	case "", "true":
		enabled = true
	case "false":
		enabled = false
	default:
		return c, dispatch(ErrorMsg{Err: errors.New("invalid value for enabling/disabling database schema usage in LLM")})
	}

	empty := ""
	c.input.Value(&empty)

	return c, dispatch(LLMUseDatabaseSchemaMsg{
		Enabled: enabled,
	})
}

func parseExportCommand(value string) ([]int, bool, string, error) {
	var rows []int
	var all bool
	var fileName string

	parts := strings.Fields(value)

	helper := "export row1,row2 filename"

	if len(parts) < 2 {
		return nil, false, "", fmt.Errorf("invalid export command format, expected: %s", helper)
	}

	if parts[1] == "all" {
		all = true
		fileName = strings.Join(parts[2:], " ")
	} else {
		all = false
		for part := range strings.SplitSeq(parts[1], ",") {
			var row int
			_, err := fmt.Sscanf(part, "%d", &row)
			if err != nil {
				return nil, false, "", fmt.Errorf("invalid row number: %s, expected format: %s", part, helper)
			}
			rows = append(rows, row)
		}
		fileName = strings.Join(parts[2:], " ")
		if fileName == "" {
			return nil, false, "", fmt.Errorf("file name cannot be empty, expected format: %s", helper)
		}
	}

	fileName = strings.TrimSuffix(fileName, ".json")
	return rows, all, fileName, nil
}

func dispatch(msg tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return msg
	}
}
