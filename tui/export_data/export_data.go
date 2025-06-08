package export_data

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/goeditor/adapter-bubbletea/editor"
	"github.com/ionut-t/perp/pkg/server"
	"github.com/ionut-t/perp/store/export"
	"github.com/ionut-t/perp/ui/help"
	statusbar "github.com/ionut-t/perp/ui/status-bar"
	"github.com/ionut-t/perp/ui/styles"
)

var (
	viewPadding  = lipgloss.NewStyle().Padding(1, 1)
	activeBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Primary.GetForeground())
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

func clearMessages() tea.Cmd {
	return tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		return clearMsg{}
	})
}

type view int

const (
	viewSplit view = iota
	viewList
	viewRecord
	viewHelp
)

type focusedView int

const (
	focusedViewList focusedView = iota
	focusedViewRecord
)

type Model struct {
	store export.Store

	width, height int

	view           view
	focusedView    focusedView
	error          error
	list           list.Model
	editor         editor.Model
	successMessage string
	help           help.Model
	server         server.Server
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func New(store export.Store, server server.Server, width, height int) Model {
	records, err := store.Load()

	delegate := list.NewDefaultDelegate()

	delegate.Styles = styles.ListItemStyles()

	editorModel := editor.New(80, 20)
	editorModel.SetCursorBlinkMode(true)
	editorModel.WithTheme(styles.EditorTheme())

	if len(records) > 0 {
		editorModel.SetContent(records[0].Content)
	}

	items := processRecords(records)

	list := list.New(items, delegate, 80, 20)

	list.Styles = styles.ListStyles()

	list.FilterInput.PromptStyle = styles.Accent
	list.FilterInput.Cursor.Style = styles.Accent

	list.InfiniteScrolling = true
	list.SetShowHelp(false)
	list.SetShowTitle(false)

	m := Model{
		store:  store,
		error:  err,
		list:   list,
		editor: editorModel,
		help:   help.New(),
		server: server,
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
			if m.view == viewHelp {
				m.view = viewSplit
				return m, nil
			}

			if m.view == viewSplit {
				if m.focusedView == focusedViewList {
					return m, func() tea.Msg {
						return ClosedMsg{}
					}
				}
			}

		case "i":
			if m.view == viewHelp {
				break
			}

			if m.focusedView == focusedViewList {
				m.focusedView = focusedViewRecord
				m.editor.Focus()
				m.editor.SetCursorPosition(0, 0)
				m.editor.SetInsertMode()
				return m, m.editor.CursorBlink()
			}

		case "tab":
			if m.view == viewHelp {
				break
			}

			if m.view == viewSplit && !m.editor.IsInsertMode() {
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
			if m.editor.IsInsertMode() || m.editor.IsCommandMode() || m.view == viewHelp {
				break
			}

			path := m.store.GetCurrentRecord().Path

			execCmd := tea.ExecProcess(exec.Command(m.store.Editor(), path), func(error) tea.Msg {
				return nil
			})
			return m, execCmd

		case "?":
			if m.editor.IsInsertMode() {
				break
			}

			switch m.view {
			case viewSplit:
				m.view = viewHelp
				m.editor.Blur()
			case viewHelp:
				m.view = viewSplit
			}
		}

	case editor.SaveMsg:
		record := m.store.GetCurrentRecord()
		record.Content = string(msg)
		err := m.store.Update(record)
		if err != nil {
			m.error = fmt.Errorf("failed to save record: %w", err)
		} else {
			m.error = nil
			records, err := m.store.Load()
			m.error = err

			if err == nil {
				records := processRecords(records)
				m.list.SetItems(records)
			}
		}

	case editor.QuitMsg:
		return m, func() tea.Msg {
			return ClosedMsg{}
		}

	case editor.DeleteFileMsg:
		current := m.store.GetCurrentRecord()

		if err := m.store.Delete(current); err == nil {
			m.error = nil
			m.successMessage = "Record deleted successfully."
			records, err := m.store.Load()
			m.error = err

			if err == nil {
				records := processRecords(records)
				m.list.SetItems(records)
			}

			if len(records) > 0 {
				current = m.store.GetCurrentRecord()
				m.editor.SetContent(current.Content)
			} else {
				m.editor.SetContent("")
			}

		} else {
			m.error = fmt.Errorf("failed to delete record: %w", err)
		}

		return m, clearMessages()

	case editor.RenameMsg:
		current := m.store.GetCurrentRecord()

		oldRecordName := current.Name
		newName := msg.FileName

		if newName == oldRecordName {
			return m, nil
		}

		if err := m.store.Rename(&current, newName); err == nil {
			m.successMessage = "Record renamed successfully."

			return m, tea.Batch(
				m.list.SetItem(m.list.Index(), item{
					title: current.Name,
					desc:  fmt.Sprintf("Last modified: %s", current.UpdatedAt.Format("02/01/2006 15:04")),
				}),
				clearMessages(),
			)

		} else {
			m.error = fmt.Errorf("failed to rename record: %w", err)
		}

		return m, clearMessages()

	case clearMsg:
		m.successMessage = ""
		m.error = nil
	}

	if m.view == viewHelp {
		hp, cmd := m.help.Update(msg)
		m.help = hp.(help.Model)

		return m, cmd
	}

	var cmds []tea.Cmd

	if m.focusedView == focusedViewList {
		ls, cmd := m.list.Update(msg)
		m.list = ls
		if selectedItem, ok := m.list.SelectedItem().(item); ok {
			m.store.SetCurrentRecordName(selectedItem.Title())
		}
		current := m.store.GetCurrentRecord()
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

	switch m.view {
	case viewList:
		return viewPadding.Render(m.list.View()) + "\n" + m.statusBarView()

	case viewRecord:
		return m.editor.View()

	case viewSplit:
		return m.getSplitView()

	case viewHelp:
		return m.help.View()

	default:
		return ""
	}
}

func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) {
	m.help.SetSize(msg.Width, msg.Height)
	m.help.SetContent(m.renderHelp())

	if msg.Width < 2*minListWidth {
		switch m.view {
		case viewSplit:
			m.view = viewList
		case viewList:
			m.view = viewSplit
		}
	}

	m.width, m.height = msg.Width, msg.Height

	availableWidth, availableHeight := m.getAvailableSizes()

	if m.view == viewList {
		m.list.SetSize(availableWidth, availableHeight)
	}

	if m.view == viewRecord {
		m.editor.SetSize(msg.Width, msg.Height)
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

func (m *Model) getAvailableSizes() (int, int) {
	h, v := viewPadding.GetFrameSize()

	statusBarHeight := lipgloss.Height(m.statusBarView())

	availableHeight := m.height - v - statusBarHeight - activeBorder.GetBorderBottomSize()
	availableWidth := m.width - h

	return availableWidth, availableHeight
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

	return renderedView + "\n" + m.statusBarView()
}

func (m *Model) statusBarView() string {
	if m.error != nil {
		return styles.Error.Margin(0, 2).Render(m.error.Error())
	}

	if m.successMessage != "" {
		return styles.Success.Margin(0, 2).Render(m.successMessage)
	}

	return lipgloss.NewStyle().Margin(0, 1).
		Render(statusbar.StatusBarView(m.server, m.width-2))
}

func processRecords(records []export.Record) []list.Item {
	items := make([]list.Item, 0, len(records))

	for _, record := range records {
		items = append(items, item{
			title: record.Name,
			desc:  fmt.Sprintf("Last modified: %s", record.UpdatedAt.Format("02/01/2006 15:04")),
		})
	}

	return items
}
