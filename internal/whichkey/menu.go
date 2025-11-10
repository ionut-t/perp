package whichkey

import (
	tea "github.com/charmbracelet/bubbletea"
)

// MenuItem represents a single menu item
type MenuItem struct {
	Key         string
	Label       string
	Description string
	Action      MenuAction
}

// MenuAction defines what happens when a menu item is selected
type MenuAction interface {
	Execute() tea.Msg
	IsSubmenu() bool
	CanExecute(ctx *MenuContext) bool
}

// CommandAction executes a command
type CommandAction struct {
	Cmd       func() tea.Msg
	Validator func(*MenuContext) bool // Optional validator function
}

func (a CommandAction) Execute() tea.Msg {
	if a.Cmd != nil {
		return a.Cmd()
	}
	return nil
}

func (a CommandAction) IsSubmenu() bool {
	return false
}

func (a CommandAction) CanExecute(ctx *MenuContext) bool {
	if a.Validator != nil {
		return a.Validator(ctx)
	}
	return true // Default to allowing execution if no validator
}

// SubmenuAction opens a submenu
type SubmenuAction struct {
	Menu *Menu
}

func (a SubmenuAction) Execute() tea.Msg {
	return ShowSubmenuMsg(a)
}

func (a SubmenuAction) IsSubmenu() bool {
	return true
}

func (a SubmenuAction) CanExecute(ctx *MenuContext) bool {
	return true // Submenus are always navigable
}

// Menu represents a which-key menu
type Menu struct {
	Title       string
	Items       []MenuItem
	Parent      *Menu
	contextFunc func() []MenuItem // Dynamic items based on context
}

// NewMenu creates a new menu with static items
func NewMenu(title string, items []MenuItem) *Menu {
	return &Menu{
		Title: title,
		Items: items,
	}
}

// NewDynamicMenu creates a menu with context-dependent items
func NewDynamicMenu(title string, contextFunc func() []MenuItem) *Menu {
	return &Menu{
		Title:       title,
		contextFunc: contextFunc,
	}
}

// GetItems returns menu items (dynamic or static)
func (m *Menu) GetItems() []MenuItem {
	if m.contextFunc != nil {
		return m.contextFunc()
	}
	return m.Items
}

// SetParent sets the parent menu for navigation
func (m *Menu) SetParent(parent *Menu) {
	m.Parent = parent
}

// Messages

// CloseMenuMsg signals that the menu should be closed
type CloseMenuMsg struct{}

// ShowSubmenuMsg signals that a submenu should be shown
type ShowSubmenuMsg struct {
	Menu *Menu
}

// ExecuteAndCloseMsg carries both an action message and signals menu closure
type ExecuteAndCloseMsg struct {
	ActionMsg tea.Msg
}
