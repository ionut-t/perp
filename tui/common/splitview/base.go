package splitview

import (
	"fmt"
	"os/exec"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	ProcessItems    func([]T) []list.Item          // Convert items to list items
	OnListSelection func(*Model[T, S], list.Item)  // Called when list selection changes
	RenderStatusBar func(*Model[T, S], int) string // Custom status bar rendering
	RenderHelp      func(*Model[T, S]) string      // Custom help rendering
	GetQuitCmd      func() tea.Msg                 // Command to dispatch when quitting

	Styles styles.Styles
	IsDark bool
}

// New creates a new split-view model
func New[T Item, S Store[T]](
	store S,
	config Config,
	processItems func([]T) []list.Item,
	renderStatusBar func(*Model[T, S], int) string,
	renderHelp func(*Model[T, S]) string,
	getQuitCmd func() tea.Msg,
	width, height int,
	s styles.Styles,
	isDark bool,
) *Model[T, S] {
	items, err := store.Load()

	delegate := list.NewDefaultDelegate()
	delegate.Styles = styles.ListItemStyles(s, isDark)

	editorModel := editor.New(80, 20)
	editorModel.WithTheme(styles.EditorTheme(s))
	editorModel.SetLanguage(config.EditorLanguage, styles.EditorLanguageTheme(isDark))

	if len(items) > 0 {
		editorModel.SetContent(items[0].GetContent())
	}

	listItems := processItems(items)

	l := list.New(listItems, delegate, 80, 20)
	l.Styles = styles.ListStyles(s, isDark)

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
		Styles:          s,
		IsDark:          isDark,
	}

	m.handleWindowSize(width, height)

	return m
}

// Init implements tea.Model
func (m Model[T, S]) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m Model[T, S]) Update(msg tea.Msg) (Model[T, S], tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleWindowSize(msg.Width, msg.Height)
		return m, nil

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
		m.help = hp
		return m, cmd
	}

	var cmds []tea.Cmd

	if m.focusedView == FocusedViewList {
		ls, cmd := m.list.Update(msg)
		m.list = ls

		// Update current item when list selection changes
		if selectedItem := m.list.SelectedItem(); selectedItem != nil {
			if m.OnListSelection != nil {
				m.OnListSelection(&m, selectedItem)
			}
		}

		current := m.store.GetCurrent()
		m.editor.SetContent(current.GetContent())
		cmds = append(cmds, cmd)
		_, cmd = m.editor.Update(nil)
		cmds = append(cmds, cmd)
	}

	em, cmd := m.editor.Update(msg)
	m.editor = em
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View implements tea.Model
func (m *Model[T, S]) View() string {
	if m.error != nil {
		return m.Styles.Error.Render(m.error.Error())
	}

	availableWidth, _ := m.getAvailableSizes()

	switch m.view {
	case ViewList:
		return styles.ViewPadding.Render(m.list.View()) + "\n" + m.statusBarView(availableWidth)

	case ViewDetail:
		return m.editor.View()

	case ViewSplit:
		return m.getSplitView()

	case ViewHelp:
		return m.help.View()

	case ViewPlaceholder:
		return styles.ViewPadding.Render(lipgloss.JoinVertical(
			lipgloss.Left,
			m.Styles.Primary.Render(m.config.PlaceholderTitle),
			"\n",
			m.Styles.Subtext0.Render(m.config.PlaceholderSubtitle),
		),
		)

	default:
		return ""
	}
}

// handleWindowSize updates dimensions and recalculates layouts
func (m *Model[T, S]) handleWindowSize(width, height int) {
	m.help.SetSize(width, height)
	if m.RenderHelp != nil {
		m.help.SetContent(m.RenderHelp(m))
	}

	m.width, m.height = width, height

	if width < 2*minListWidth {
		switch m.view {
		case ViewSplit:
			m.view = ViewList
		case ViewList:
			m.view = ViewSplit
		}
	}

	availableWidth, availableHeight := m.getAvailableSizes()

	if m.view == ViewList {
		m.list.SetSize(availableWidth, availableHeight)
	}

	if m.view == ViewDetail {
		m.editor.SetSize(width, height)
	}

	if m.view == ViewSplit {
		// Calculate split
		horizontalFrameBorderSize := m.Styles.ActiveBorder.GetHorizontalFrameSize()

		listWidth := min(minListWidth, availableWidth/2)
		detailWidth := availableWidth - listWidth - splitViewSeparatorWidth

		// Content dimensions inside borders
		listContentWidth := listWidth - horizontalFrameBorderSize
		detailContentWidth := detailWidth - horizontalFrameBorderSize

		borderV := m.Styles.ActiveBorder.GetVerticalFrameSize()
		paneContentHeight := availableHeight - borderV

		m.list.SetSize(listContentWidth, paneContentHeight)
		m.editor.SetSize(detailContentWidth, paneContentHeight)
	}
}

// getAvailableSizes calculates available space for content
func (m *Model[T, S]) getAvailableSizes() (int, int) {
	h, _ := styles.ViewPadding.GetFrameSize()
	statusBarHeight := 1

	availableHeight := m.height - statusBarHeight
	availableWidth := m.width - h

	return availableWidth, availableHeight
}

// getSplitView renders the split view layout
func (m *Model[T, S]) getSplitView() string {
	availableWidth, availableHeight := m.getAvailableSizes()

	listWidth := min(minListWidth, availableWidth/2)
	detailWidth := availableWidth - listWidth - splitViewSeparatorWidth

	listBorder := m.Styles.InactiveBorder
	detailBorder := m.Styles.InactiveBorder

	if m.focusedView == FocusedViewList {
		listBorder = m.Styles.ActiveBorder
	} else {
		detailBorder = m.Styles.ActiveBorder
	}

	joinedContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		listBorder.
			Width(listWidth).
			Height(availableHeight).
			Render(m.list.View()),
		splitViewSeparator,
		detailBorder.
			Width(detailWidth).
			Height(availableHeight).
			Render(m.editor.View()),
	)

	padding := lipgloss.NewStyle().Padding(0, 1)

	return padding.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		joinedContent,
		m.statusBarView(availableWidth),
	))
}

// statusBarView renders the status bar
func (m *Model[T, S]) statusBarView(width int) string {
	if m.error != nil {
		return m.Styles.Error.Margin(0, 2).Render(m.error.Error())
	}

	if m.successMessage != "" {
		return m.Styles.Success.Margin(0, 2).Render(m.successMessage)
	}

	if m.RenderStatusBar != nil {
		return m.RenderStatusBar(m, width)
	}

	return ""
}

// OpenInExternalEditor opens the current item in an external editor
func (m Model[T, S]) OpenInExternalEditor() (Model[T, S], tea.Cmd) {
	current := m.store.GetCurrent()
	path := m.store.GetPath(current)

	execCmd := tea.ExecProcess(exec.Command(m.store.Editor(), path), func(error) tea.Msg {
		return EditorClosedMsg{}
	})
	return m, execCmd
}

// handleEditorClose handles the return from external editor
func (m Model[T, S]) handleEditorClose() (Model[T, S], tea.Cmd) {
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
	m.editor = textEditor

	return m, cmd
}

// handleSave handles editor save messages
func (m Model[T, S]) handleSave(msg editor.SaveMsg) (Model[T, S], tea.Cmd) {
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
func (m Model[T, S]) handleDelete() (Model[T, S], tea.Cmd) {
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
					m.OnListSelection(&m, selectedItem)
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

		var textEditor editor.Model
		textEditor, cmd = m.editor.Update(nil)
		m.editor = textEditor
		m.editor.SetLanguage(m.config.EditorLanguage, styles.EditorLanguageTheme(m.IsDark))

	} else {
		m.error = fmt.Errorf("failed to delete: %w", err)
	}

	return m, tea.Batch(
		cmd,
		ClearMessages(),
	)
}

// handleRename handles file rename
func (m Model[T, S]) handleRename(msg editor.RenameMsg) (Model[T, S], tea.Cmd) {
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
	m.editor.SetLanguage(language, styles.EditorLanguageTheme(m.IsDark))
}
