package export_data

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"github.com/ionut-t/perp/internal/config"
	"github.com/ionut-t/perp/pkg/export"
	"github.com/ionut-t/perp/ui/styles"
)

var (
	viewPadding  = lipgloss.NewStyle().Padding(1, 1)
	activeBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Text.GetForeground())
	inactiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Overlay0.
				GetForeground())
	splitViewSeparator      = " "
	splitViewSeparatorWidth = lipgloss.Width(splitViewSeparator)
	minListWidth            = 50
)

type ClosedMsg struct{}

type clearMsg struct{}

var clearMessages = tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
	return clearMsg{}
})

type view int

const (
	viewSplit view = iota
	viewList
	viewRecord
)

type focusedView int

const (
	focusedViewList focusedView = iota
	focusedViewRecord
)

type Model struct {
	width, height  int
	view           view
	focusedView    focusedView
	error          error
	list           list.Model
	editor         editor.Model
	successMessage string
	recordsMap     map[string]export.Record
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func New(width, height int) Model {
	records, err := export.Load()

	delegate := list.NewDefaultDelegate()

	delegate.Styles = styles.ListItemStyles()

	editorModel := editor.New(80, 20)
	editorModel.SetCursorBlinkMode(true)

	if len(records) > 0 {
		editorModel.SetContent(records[0].Content)
	}

	recordsMap := make(map[string]export.Record, len(records))
	for _, record := range records {
		recordsMap[record.Name] = record
	}

	items := processRecords(recordsMap)

	list := list.New(items, delegate, 80, 20)

	list.Styles = styles.ListStyles()

	list.FilterInput.PromptStyle = styles.Accent
	list.FilterInput.Cursor.Style = styles.Accent

	list.InfiniteScrolling = true
	list.SetShowHelp(false)
	list.SetShowTitle(false)

	m := Model{
		error:      err,
		list:       list,
		editor:     editorModel,
		recordsMap: recordsMap,
	}

	m.handleWindowSize(tea.WindowSizeMsg{
		Width:  width,
		Height: height,
	})

	return m
}

func (m Model) Init() tea.Cmd {
	return m.editor.CursorBlink()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "q", "esc":
			if m.view == viewSplit {
				if m.focusedView == focusedViewList {
					return m, func() tea.Msg {
						return ClosedMsg{}
					}
				}
			}

		case "i":
			if m.focusedView == focusedViewList {
				m.focusedView = focusedViewRecord
				m.editor.Focus()
				m.editor.SetCursorPosition(0, 0)
				m.editor.SetInsertMode()
				return m, m.editor.CursorBlink()
			}

		case "shift+left":
			if m.view == viewSplit {
				if m.focusedView == focusedViewList {
					m.focusedView = focusedViewRecord
					m.editor.Focus()
					m.editor.SetCursorPosition(0, 0)
					m.editor.SetNormalMode()
				} else {
					m.focusedView = focusedViewList
					m.editor.Blur()
				}
			}

		case "shift+right":
			if m.view == viewSplit {
				if m.focusedView == focusedViewList {
					m.focusedView = focusedViewRecord
					m.editor.Focus()
					m.editor.SetCursorPosition(0, 0)
					m.editor.SetNormalMode()
				} else {
					m.focusedView = focusedViewList
					m.editor.Blur()
				}
			}

		case "e":
			if m.editor.IsInsertMode() || m.editor.IsCommandMode() {
				break
			}

			editor := config.GetEditor()
			name := m.list.SelectedItem().(item).Title()
			var path string

			if record, ok := m.recordsMap[name]; ok {
				path = record.Path
			} else {
				m.error = fmt.Errorf("record not found: %s", name)
				return m, nil
			}

			execCmd := tea.ExecProcess(exec.Command(editor, path), func(error) tea.Msg {
				return nil
			})
			return m, execCmd
		}

	case editor.SaveMsg:
		record := m.recordsMap[m.list.SelectedItem().(item).Title()]
		record.Content = string(msg)
		err := export.Update(record)
		if err != nil {
			m.error = fmt.Errorf("failed to save record: %w", err)
		} else {
			m.error = nil
			records, err := export.Load()
			m.error = err

			if err == nil {
				m.recordsMap = make(map[string]export.Record, len(records))
				for _, record := range records {
					m.recordsMap[record.Name] = record
				}

				records := processRecords(m.recordsMap)
				m.list.SetItems(records)
				m.list.Select(0)
			}
		}

	case editor.QuitMsg:
		return m, func() tea.Msg {
			return ClosedMsg{}
		}

	case editor.DeleteFileMsg:
		selected := m.list.SelectedItem()
		current := m.recordsMap[selected.(item).Title()]

		if err := export.Delete(current); err == nil {
			m.error = nil
			m.successMessage = "Record deleted successfully."
			records, err := export.Load()
			m.error = err

			if err == nil {
				m.recordsMap = make(map[string]export.Record, len(records))
				for _, record := range records {
					m.recordsMap[record.Name] = record
				}

				records := processRecords(m.recordsMap)
				m.list.SetItems(records)
			}

			if len(records) > 0 {
				selected = m.list.SelectedItem()
				current = m.recordsMap[selected.(item).Title()]
				m.editor.SetContent(current.Content)
			} else {
				m.editor.SetContent("")
			}

		} else {
			m.error = fmt.Errorf("failed to delete record: %w", err)
		}

		return m, clearMessages

	case editor.RenameMsg:
		selected := m.list.SelectedItem()
		current := m.recordsMap[selected.(item).Title()]

		oldRecordName := current.Name
		newName := msg.FileName

		if newName == oldRecordName {
			return m, nil
		}

		if err := current.Rename(newName); err == nil {
			m.successMessage = "Record renamed successfully."
			delete(m.recordsMap, oldRecordName)
			m.recordsMap[current.Name] = current
			m.list.SetItems(processRecords(m.recordsMap))

		} else {
			m.error = fmt.Errorf("failed to rename record: %w", err)
		}

		return m, clearMessages

	case clearMsg:
		m.successMessage = ""
		m.error = nil
	}

	var cmds []tea.Cmd

	if m.focusedView == focusedViewList {
		ls, cmd := m.list.Update(msg)
		m.list = ls
		selected := m.list.SelectedItem()
		current := m.recordsMap[selected.(item).Title()]
		m.editor.SetContent(current.Content)
		cmds = append(cmds, cmd)
	}

	em, cmd := m.editor.Update(msg)
	m.editor = em.(editor.Model)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.error != nil {
		return "Error loading export records: " + m.error.Error()
	}

	if len(m.recordsMap) == 0 {
		return "No export records found."
	}

	switch m.view {
	case viewList:
		// if m.delete.active {
		// 	return viewPadding.Render(m.list.View()) + "\n" + m.delete.View()
		// }

		// if m.renameInput.active {
		// 	return m.getViewInRenameMode(viewPadding.Render(m.list.View()))
		// }

		// if m.cmdInput.active {
		// 	return m.getViewInCmdMode()
		// }

		return viewPadding.Render(m.list.View()) + "\n" + m.statusBarView()

	case viewRecord:
		// if m.renameInput.active {
		// 	return m.getViewInRenameMode(m.editor.View())
		// }

		// if m.cmdInput.active {
		// 	return m.noteView.View() + "\n" + m.cmdInput.View()
		// }

		return m.editor.View()

	case viewSplit:
		return m.getSplitView()

	default:
		return ""
	}
}

func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) {
	if msg.Width < 2*minListWidth {
		if m.view == viewSplit {
			m.view = viewList
		} else if m.view == viewList {
			m.view = viewSplit
		}
	}

	m.width, m.height = msg.Width, msg.Height

	availableWidth, availableHeight, cmdViewHeight := m.getAvailableSizes()

	// m.help.SetSize(msg.Width, msg.Height)

	// m.delete.width = msg.Width

	if m.view == viewList {
		m.list.SetSize(availableWidth, availableHeight)
		// m.help.SetSize(msg.Width, msg.Height)
	}

	if m.view == viewRecord {
		m.editor.SetSize(msg.Width, msg.Height-cmdViewHeight)
	}

	if m.view == viewSplit {
		listWidth := min(availableWidth/2, minListWidth)

		// Set list dimensions
		m.list.SetHeight(availableHeight)
		m.list.SetWidth(listWidth)

		// Set note view dimensions
		m.editor.SetSize(availableWidth-listWidth, availableHeight)
	}
}

func (m *Model) getAvailableSizes() (int, int, int) {
	h, v := viewPadding.GetFrameSize()

	var cmdExecutorHeight int
	var deleteViewHeight int

	// if m.renameInput.active {
	// 	cmdExecutorHeight = lipgloss.Height(m.renameInput.View())
	// }

	// statusBarHeight := utils.Ternary(m.cmdInput.active || m.renameInput.active, 0, lipgloss.Height(m.statusBarView()))

	// if m.delete.active {
	// 	deleteViewHeight = lipgloss.Height(m.delete.View())
	// }

	statusBarHeight := lipgloss.Height(m.statusBarView())

	availableHeight := m.height - v - statusBarHeight - cmdExecutorHeight - deleteViewHeight - activeBorder.GetBorderBottomSize()
	availableWidth := m.width - h

	cmdViewHeight := cmdExecutorHeight - deleteViewHeight

	return availableWidth, availableHeight, cmdViewHeight
}

func (m *Model) getSplitView() string {
	horizontalFrameSize := viewPadding.GetHorizontalFrameSize()
	horizontalFrameBorderSize := activeBorder.GetHorizontalFrameSize()

	availableWidth := m.width - horizontalFrameSize

	listWidth := min(minListWidth, availableWidth/2) - horizontalFrameBorderSize*2 - splitViewSeparatorWidth
	noteWidth := availableWidth - listWidth - horizontalFrameBorderSize*2 - splitViewSeparatorWidth

	var joinedContent string

	if m.focusedView == focusedViewList {
		joinedContent = lipgloss.JoinHorizontal(
			lipgloss.Left,
			activeBorder.
				Width(listWidth).
				Render(m.list.View()),
			splitViewSeparator,
			inactiveBorder.
				Width(noteWidth).
				Height(m.list.Height()).
				Render(m.editor.View()),
		)
	} else {
		joinedContent = lipgloss.JoinHorizontal(
			lipgloss.Left,
			inactiveBorder.
				Width(listWidth).
				Render(m.list.View()),
			splitViewSeparator,
			activeBorder.
				Width(noteWidth).
				Height(m.list.Height()).
				Render(m.editor.View()),
		)
	}

	renderedView := viewPadding.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		joinedContent,
	))

	// if m.renameInput.active {
	// 	if m.error != nil {
	// 		return renderedView + "\n" + styles.Error.Margin(0, 2).Render(m.error.Error())
	// 	}

	// 	return renderedView + "\n" + m.renameInput.View()
	// }

	// if m.cmdInput.active {
	// 	if m.error != nil {
	// 		return renderedView + "\n" + styles.Error.Margin(0, 2).Render(m.error.Error())
	// 	}

	// 	return renderedView + "\n" + m.cmdInput.View()
	// }

	// if m.delete.active {
	// 	return lipgloss.JoinVertical(
	// 		lipgloss.Left,
	// 		renderedView,
	// 		m.delete.View(),
	// 	)
	// }

	return renderedView + "\n" + m.statusBarView()
}

func (m *Model) statusBarView() string {
	if m.error != nil {
		return styles.Error.Margin(0, 2).Render(m.error.Error())
	}

	if m.successMessage != "" {
		return styles.Success.Margin(0, 2).Render(m.successMessage)
	}

	// if m.list.FilterState() == list.Filtering {
	// 	m.help.Keys.ShortHelpBindings = []key.Binding{
	// 		keymap.Cancel,
	// 	}
	// } else {
	// 	m.help.Keys.ShortHelpBindings = []key.Binding{
	// 		keymap.Select,
	// 		keymap.QuickEditor,
	// 		keymap.Rename,
	// 		keymap.Search,
	// 		keymap.Delete,
	// 		keymap.New,
	// 		keymap.Quit,
	// 		keymap.Help,
	// 	}
	// }

	// if m.delete.active {
	// 	return ""
	// }

	// if m.help.FullView {
	// 	return m.help.View()
	// }

	// return lipgloss.NewStyle().Margin(0, 2).Render(m.help.View())
	return styles.Text.Render(fmt.Sprintf(
		"View: %d | Focused: %d | Records: %d",
		m.view,
		m.focusedView,
		m.list.Index(),
	))
}

func processRecords(records map[string]export.Record) []list.Item {
	items := make([]list.Item, 0, len(records))

	for _, record := range records {
		items = append(items, item{
			title: record.Name,
			desc:  fmt.Sprintf("Last modified: %s", record.UpdatedAt.Format("02/01/2006 15:04")),
		})
	}

	return items
}
