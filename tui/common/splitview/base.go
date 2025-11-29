package splitview

import (
	"fmt"
	"os/exec"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/coffee/styles"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/ui/help"
)

var (
	splitViewSeparator      = " "
	splitViewSeparatorWidth = lipgloss.Width(splitViewSeparator)
	minListWidth            = 50
)

// ChangeFocused is the key binding for switching focus between list and editor
var ChangeFocused = key.NewBinding(
	key.WithKeys("tab"),
	key.WithHelp("tab", "change focus between editor and list"),
)

// View represents the different view states
type View int

const (
	ViewSplit View = iota
	ViewList
	ViewDetail
	ViewHelp
	ViewPlaceholder
)

// FocusedView represents which pane has focus in split view
type FocusedView int

const (
	FocusedViewList FocusedView = iota
	FocusedViewDetail
)

// Item represents an item that can be displayed in the list
type Item interface {
	GetName() string
	GetContent() string
	SetContent(string)
}

// Store represents a generic file-based storage interface
type Store[T Item] interface {
	Load() ([]T, error)
	Update(T) error
	Delete(T) error
	Rename(*T, string) error
	Editor() string
	GetCurrent() T
	SetCurrentName(string)
	GetPath(T) string
}

// Config holds configuration for customizing split-view behavior
type Config struct {
	EditorLanguage      string // Language for syntax highlighting
	PlaceholderTitle    string // Title shown when no items exist
	PlaceholderSubtitle string // Subtitle shown when no items exist
	SuccessDeleteMsg    string // Message shown on successful delete
	SuccessRenameMsg    string // Message shown on successful rename
}

// Model is the generic split-view model
type Model[T Item, S Store[T]] struct {
	store  S
	config Config

	width, height int

	view           View
	focusedView    FocusedView
	error          error
	list           list.Model
	editor         editor.Model
	successMessage string
	help           help.Model

	// Callbacks for custom behavior
	ProcessItems    func([]T) []list.Item         // Convert items to list items
	OnListSelection func(*Model[T, S], list.Item) // Called when list selection changes
	RenderStatusBar func(*Model[T, S]) string     // Custom status bar rendering
	RenderHelp      func(*Model[T, S]) string     // Custom help rendering
	GetQuitCmd      func() tea.Msg                // Command to dispatch when quitting
}

// New creates a new split-view model
func New[T Item, S Store[T]](
	store S,
	config Config,
	processItems func([]T) []list.Item,
	renderStatusBar func(*Model[T, S]) string,
	renderHelp func(*Model[T, S]) string,
	getQuitCmd func() tea.Msg,
	width, height int,
) *Model[T, S] {
	items, err := store.Load()

	delegate := list.NewDefaultDelegate()
	delegate.Styles = styles.ListItemStyles()

	editorModel := editor.New(80, 20)
	editorModel.WithTheme(styles.EditorTheme())
	editorModel.SetLanguage(config.EditorLanguage, styles.EditorLanguageTheme())

	if len(items) > 0 {
		editorModel.SetContent(items[0].GetContent())
	}

	listItems := processItems(items)

	l := list.New(listItems, delegate, 80, 20)
	l.Styles = styles.ListStyles()
	l.FilterInput.PromptStyle = styles.Accent
	l.FilterInput.Cursor.Style = styles.Accent
	l.InfiniteScrolling = true
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.DisableQuitKeybindings()

	view := ViewSplit
	if len(listItems) == 0 {
		view = ViewPlaceholder
	}

	m := &Model[T, S]{
		store:           store,
		config:          config,
		error:           err,
		list:            l,
		editor:          editorModel,
		help:            help.New(),
		view:            view,
		ProcessItems:    processItems,
		RenderStatusBar: renderStatusBar,
		RenderHelp:      renderHelp,
		GetQuitCmd:      getQuitCmd,
	}

	m.handleWindowSize(tea.WindowSizeMsg{
		Width:  width,
		Height: height,
	})

	return m
}

// Init implements tea.Model
func (m *Model[T, S]) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m *Model[T, S]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, keymap.Quit), key.Matches(msg, keymap.Cancel):
			if m.view == ViewHelp {
				m.view = ViewSplit
				return m, nil
			}

		case key.Matches(msg, keymap.Insert):
			if m.view == ViewHelp || m.view == ViewPlaceholder {
				break
			}

			if m.focusedView == FocusedViewList {
				m.focusedView = FocusedViewDetail
				m.editor.Focus()
				_ = m.editor.SetCursorPosition(0, 0)
				m.editor.SetInsertMode()
				return m, nil
			}

		case key.Matches(msg, ChangeFocused):
			if m.view == ViewHelp || m.view == ViewPlaceholder {
				break
			}

			if m.view == ViewSplit && !m.editor.IsInsertMode() {
				if m.focusedView == FocusedViewList {
					m.focusedView = FocusedViewDetail
					m.editor.Focus()
					_ = m.editor.SetCursorPosition(0, 0)
					m.editor.SetNormalMode()
				} else {
					m.focusedView = FocusedViewList
					m.editor.Blur()
				}
			}
		}

	case EditorClosedMsg:
		return m.handleEditorClose()

	case editor.SaveMsg:
		return m.handleSave(msg)

	case editor.QuitMsg:
		if m.GetQuitCmd != nil {
			return m, func() tea.Msg { return m.GetQuitCmd() }
		}

	case editor.DeleteFileMsg:
		return m.handleDelete()

	case editor.RenameMsg:
		return m.handleRename(msg)

	case ClearMsg:
		m.successMessage = ""
		m.error = nil
	}

	if m.view == ViewHelp {
		hp, cmd := m.help.Update(msg)
		m.help = hp.(help.Model)
		return m, cmd
	}

	var cmds []tea.Cmd

	if m.focusedView == FocusedViewList {
		ls, cmd := m.list.Update(msg)
		m.list = ls

		// Update current item when list selection changes
		if selectedItem := m.list.SelectedItem(); selectedItem != nil {
			if m.OnListSelection != nil {
				m.OnListSelection(m, selectedItem)
			}
		}

		current := m.store.GetCurrent()
		m.editor.SetContent(current.GetContent())
		cmds = append(cmds, cmd)
		_, cmd = m.editor.Update(nil)
		cmds = append(cmds, cmd)
	}

	em, cmd := m.editor.Update(msg)
	m.editor = em.(editor.Model)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *Model[T, S]) View() string {
	if m.error != nil {
		return styles.Error.Render(m.error.Error())
	}

	switch m.view {
	case ViewList:
		return styles.ViewPadding.Render(m.list.View()) + "\n" + m.statusBarView()

	case ViewDetail:
		return m.editor.View()

	case ViewSplit:
		return m.getSplitView()

	case ViewHelp:
		return m.help.View()

	case ViewPlaceholder:
		return styles.ViewPadding.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				styles.Primary.Render(m.config.PlaceholderTitle),
				"\n",
				styles.Subtext0.Render(m.config.PlaceholderSubtitle),
			),
		)

	default:
		return ""
	}
}

// handleWindowSize updates dimensions and recalculates layouts
func (m *Model[T, S]) handleWindowSize(msg tea.WindowSizeMsg) {
	m.help.SetSize(msg.Width, msg.Height)
	if m.RenderHelp != nil {
		m.help.SetContent(m.RenderHelp(m))
	}

	if msg.Width < 2*minListWidth {
		switch m.view {
		case ViewSplit:
			m.view = ViewList
		case ViewList:
			m.view = ViewSplit
		}
	}

	m.width, m.height = msg.Width, msg.Height

	availableWidth, availableHeight := m.getAvailableSizes()

	if m.view == ViewList {
		m.list.SetSize(availableWidth, availableHeight)
	}

	if m.view == ViewDetail {
		m.editor.SetSize(msg.Width, msg.Height)
	}

	if m.view == ViewSplit {
		listWidth := min(availableWidth/2, minListWidth)

		m.list.SetHeight(availableHeight)
		m.list.SetWidth(listWidth)

		m.editor.SetSize(availableWidth-listWidth, availableHeight)
	}
}

// getAvailableSizes calculates available space for content
func (m *Model[T, S]) getAvailableSizes() (int, int) {
	h, v := styles.ViewPadding.GetFrameSize()

	statusBarHeight := lipgloss.Height(m.statusBarView())

	availableHeight := m.height - v - statusBarHeight - styles.ActiveBorder.GetBorderBottomSize()
	availableWidth := m.width - h

	return availableWidth, availableHeight
}

// getSplitView renders the split view layout
func (m *Model[T, S]) getSplitView() string {
	horizontalFrameSize := styles.ViewPadding.GetHorizontalFrameSize()
	horizontalFrameBorderSize := styles.ActiveBorder.GetHorizontalFrameSize()

	availableWidth := m.width - horizontalFrameSize

	listWidth := min(minListWidth, availableWidth/2) - horizontalFrameBorderSize*2 - splitViewSeparatorWidth
	detailWidth := availableWidth - listWidth - horizontalFrameBorderSize*2 - splitViewSeparatorWidth

	var joinedContent string

	if m.focusedView == FocusedViewList {
		joinedContent = lipgloss.JoinHorizontal(
			lipgloss.Left,
			styles.ActiveBorder.
				Width(listWidth).
				Render(m.list.View()),
			splitViewSeparator,
			styles.InactiveBorder.
				Width(detailWidth).
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
				Width(detailWidth).
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

// statusBarView renders the status bar
func (m *Model[T, S]) statusBarView() string {
	if m.error != nil {
		return styles.Error.Margin(0, 2).Render(m.error.Error())
	}

	if m.successMessage != "" {
		return styles.Success.Margin(0, 2).Render(m.successMessage)
	}

	if m.RenderStatusBar != nil {
		return lipgloss.NewStyle().Margin(0, 1).Render(m.RenderStatusBar(m))
	}

	return ""
}

// OpenInExternalEditor opens the current item in an external editor
func (m *Model[T, S]) OpenInExternalEditor() (tea.Model, tea.Cmd) {
	current := m.store.GetCurrent()
	path := m.store.GetPath(current)

	execCmd := tea.ExecProcess(exec.Command(m.store.Editor(), path), func(error) tea.Msg {
		return EditorClosedMsg{}
	})
	return m, execCmd
}

// handleEditorClose handles the return from external editor
func (m *Model[T, S]) handleEditorClose() (tea.Model, tea.Cmd) {
	items, err := m.store.Load()
	if err != nil {
		m.error = err
		return m, nil
	}

	m.list.SetItems(m.ProcessItems(items))

	current := m.store.GetCurrent()
	m.editor.SetContent(current.GetContent())

	m.list.ResetFilter()

	textEditor, cmd := m.editor.Update(nil)
	m.editor = textEditor.(editor.Model)

	return m, cmd
}

// handleSave handles editor save messages
func (m *Model[T, S]) handleSave(msg editor.SaveMsg) (tea.Model, tea.Cmd) {
	current := m.store.GetCurrent()
	current.SetContent(string(msg.Content))
	err := m.store.Update(current)
	if err != nil {
		m.error = fmt.Errorf("failed to save: %w", err)
	} else {
		m.error = nil
		items, err := m.store.Load()
		m.error = err

		if err == nil {
			m.list.SetItems(m.ProcessItems(items))
		}
	}

	return m, nil
}

// handleDelete handles file deletion
func (m *Model[T, S]) handleDelete() (tea.Model, tea.Cmd) {
	current := m.store.GetCurrent()

	var cmd tea.Cmd

	if err := m.store.Delete(current); err == nil {
		m.error = nil
		m.successMessage = m.config.SuccessDeleteMsg
		items, err := m.store.Load()
		m.error = err

		if err == nil {
			listItems := m.ProcessItems(items)
			m.list.SetItems(listItems)
			if selectedItem := m.list.SelectedItem(); selectedItem != nil {
				if m.OnListSelection != nil {
					m.OnListSelection(m, selectedItem)
				}
			}
		}

		if len(items) > 0 {
			current = m.store.GetCurrent()
			m.editor.SetContent(current.GetContent())
		} else {
			m.editor.SetContent("")
			m.view = ViewPlaceholder
		}

		var textEditor tea.Model
		textEditor, cmd = m.editor.Update(nil)
		m.editor = textEditor.(editor.Model)
		m.editor.SetLanguage(m.config.EditorLanguage, styles.EditorLanguageTheme())

	} else {
		m.error = fmt.Errorf("failed to delete: %w", err)
	}

	return m, tea.Batch(
		cmd,
		ClearMessages(),
	)
}

// handleRename handles file rename
func (m *Model[T, S]) handleRename(msg editor.RenameMsg) (tea.Model, tea.Cmd) {
	current := m.store.GetCurrent()

	oldName := current.GetName()
	newName := msg.FileName

	if newName == oldName {
		return m, nil
	}

	if err := m.store.Rename(&current, newName); err == nil {
		m.successMessage = m.config.SuccessRenameMsg

		// Reload to get updated item for display
		items, _ := m.store.Load()
		updatedListItems := m.ProcessItems(items)

		return m, tea.Batch(
			m.list.SetItem(m.list.Index(), updatedListItems[m.list.Index()]),
			ClearMessages(),
		)

	} else {
		m.error = fmt.Errorf("failed to rename: %w", err)
	}

	return m, ClearMessages()
}

// CanTriggerLeaderKey returns whether the leader key can be triggered
func (m *Model[T, S]) CanTriggerLeaderKey() bool {
	return m.editor.IsNormalMode() && m.list.FilterState() != list.Filtering
}

// HandleHelpToggle toggles the help view
func (m *Model[T, S]) HandleHelpToggle() {
	if m.editor.IsInsertMode() ||
		m.editor.IsCommandMode() ||
		m.view == ViewPlaceholder {
		return
	}

	switch m.view {
	case ViewSplit:
		m.view = ViewHelp
		m.editor.Blur()
	case ViewHelp:
		m.view = ViewSplit
	}
}

// GetList returns the list model (useful for external access)
func (m *Model[T, S]) GetList() *list.Model {
	return &m.list
}

// GetEditor returns the editor model (useful for external access)
func (m *Model[T, S]) GetEditor() *editor.Model {
	return &m.editor
}

// GetStore returns the store (useful for external access)
func (m *Model[T, S]) GetStore() S {
	return m.store
}

// GetWidth returns the current width
func (m *Model[T, S]) GetWidth() int {
	return m.width
}

// SetLanguage sets the editor language
func (m *Model[T, S]) SetLanguage(language string) {
	m.editor.SetLanguage(language, styles.EditorLanguageTheme())
}
