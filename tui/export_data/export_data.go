package export_data

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/pkg/server"
	"github.com/ionut-t/perp/store/export"
	"github.com/ionut-t/perp/ui/help"
	"github.com/ionut-t/perp/ui/styles"
)

var (
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
	viewPlaceholder
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
	editorModel.WithTheme(styles.EditorTheme())
	editorModel.SetLanguage("json", styles.HighlighterTheme())

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

	view := viewSplit
	if len(items) == 0 {
		view = viewPlaceholder
	}

	m := Model{
		store:  store,
		error:  err,
		list:   list,
		editor: editorModel,
		help:   help.New(),
		server: server,
		view:   view,
	}

	m.handleWindowSize(tea.WindowSizeMsg{
		Width:  width,
		Height: height,
	})

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, keymap.Quit) || key.Matches(msg, keymap.Cancel):
			switch m.view {
			case viewSplit:
				if m.focusedView == focusedViewList {
					return m, func() tea.Msg {
						return ClosedMsg{}
					}
				}

			case viewHelp:
				m.view = viewSplit
				return m, nil

			case viewPlaceholder:
				return m, func() tea.Msg {
					return ClosedMsg{}
				}
			}

		case key.Matches(msg, keymap.Insert):
			if m.view == viewHelp || m.view == viewPlaceholder {
				break
			}

			if m.focusedView == focusedViewList {
				m.focusedView = focusedViewRecord
				m.editor.Focus()
				_ = m.editor.SetCursorPosition(0, 0)
				m.editor.SetInsertMode()
				return m, nil
			}

		case key.Matches(msg, changeFocused):
			if m.view == viewHelp || m.view == viewPlaceholder {
				break
			}

			if m.view == viewSplit && !m.editor.IsInsertMode() {
				if m.focusedView == focusedViewList {
					m.focusedView = focusedViewRecord
					m.editor.Focus()
					_ = m.editor.SetCursorPosition(0, 0)
					m.editor.SetNormalMode()
				} else {
					m.focusedView = focusedViewList
					m.editor.Blur()
				}
			}

		case key.Matches(msg, keymap.Editor):
			if m.editor.IsInsertMode() ||
				m.editor.IsCommandMode() ||
				m.view == viewHelp ||
				m.view == viewPlaceholder {
				break
			}

			path := m.store.GetCurrentRecord().Path

			execCmd := tea.ExecProcess(exec.Command(m.store.Editor(), path), func(error) tea.Msg {
				return nil
			})
			return m, execCmd

		case key.Matches(msg, keymap.Help):
			if m.editor.IsInsertMode() ||
				m.editor.IsCommandMode() ||
				m.view == viewPlaceholder {
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
		record.Content = string(msg.Content)
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
				if selectedItem, ok := m.list.SelectedItem().(item); ok {
					m.store.SetCurrentRecordName(selectedItem.Title())
				}
			}

			if len(records) > 0 {
				current = m.store.GetCurrentRecord()
				m.editor.SetContent(current.Content)
			} else {
				m.editor.SetContent("")
				m.view = viewPlaceholder
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

		lang := "json"
		if filepath.Ext(current.Path) == ".json" {
			lang = "json"
		} else if filepath.Ext(current.Path) == ".csv" {
			lang = "csv"
		}

		m.editor.SetLanguage(lang, styles.HighlighterTheme())
		cmds = append(cmds, cmd)
		_, cmd = m.editor.Update(nil)
		cmds = append(cmds, cmd)
	}

	em, cmd := m.editor.Update(msg)
	m.editor = em.(editor.Model)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.error != nil {
		return styles.Error.Render(m.error.Error())
	}

	switch m.view {
	case viewList:
		return styles.ViewPadding.Render(m.list.View()) + "\n" + m.statusBarView()

	case viewRecord:
		return m.editor.View()

	case viewSplit:
		return m.getSplitView()

	case viewHelp:
		return m.help.View()

	case viewPlaceholder:
		return styles.ViewPadding.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				styles.Primary.Render("No data exported."),
				"\n",
				styles.Subtext0.Render("Press 'q' to go back."),
			),
		)

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

		m.list.SetHeight(availableHeight)
		m.list.SetWidth(listWidth)

		m.editor.SetSize(availableWidth-listWidth, availableHeight)
	}
}

func (m *Model) getAvailableSizes() (int, int) {
	h, v := styles.ViewPadding.GetFrameSize()

	statusBarHeight := lipgloss.Height(m.statusBarView())

	availableHeight := m.height - v - statusBarHeight - styles.ActiveBorder.GetBorderBottomSize()
	availableWidth := m.width - h

	return availableWidth, availableHeight
}

func (m *Model) getSplitView() string {
	horizontalFrameSize := styles.ViewPadding.GetHorizontalFrameSize()
	horizontalFrameBorderSize := styles.ActiveBorder.GetHorizontalFrameSize()

	availableWidth := m.width - horizontalFrameSize

	listWidth := min(minListWidth, availableWidth/2) - horizontalFrameBorderSize*2 - splitViewSeparatorWidth
	noteWidth := availableWidth - listWidth - horizontalFrameBorderSize*2 - splitViewSeparatorWidth

	var joinedContent string

	if m.focusedView == focusedViewList {
		joinedContent = lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.ActiveBorder.
				Width(listWidth).
				Render(m.list.View()),
			splitViewSeparator,
			styles.InactiveBorder.
				Width(noteWidth).
				Height(m.list.Height()).
				Render(m.editor.View()),
		)
	} else {
		joinedContent = lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.InactiveBorder.
				Width(listWidth).
				Render(m.list.View()),
			splitViewSeparator,
			styles.ActiveBorder.
				Width(noteWidth).
				Height(m.list.Height()).
				Render(m.editor.View()),
		)
	}

	renderedView := styles.ViewPadding.Render(lipgloss.JoinVertical(
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
		Render(m.renderStatusBar())
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

func (m *Model) renderStatusBar() string {
	bg := styles.Surface0.GetBackground()

	separator := styles.Surface0.Render(" | ")

	serverName := styles.Primary.Background(bg).Render(m.server.Name)

	database := styles.Accent.Background(bg).Render(m.server.Database)

	left := serverName + separator + database

	leftInfo := styles.Surface0.Padding(0, 1).Render(left)

	helpText := styles.Info.Background(bg).PaddingRight(1).Render("? Help")

	displayedInfoWidth := m.width -
		lipgloss.Width(leftInfo) -
		lipgloss.Width(helpText) -
		lipgloss.Width(separator)

	spaces := styles.Surface0.Render(strings.Repeat(" ", max(0, displayedInfoWidth)))

	return styles.Surface0.Width(m.width).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Right,
			leftInfo,
			spaces,
			helpText,
		),
	)
}
