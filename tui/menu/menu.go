package menu

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/coffee/styles"
	"github.com/ionut-t/perp/internal/whichkey"
	"github.com/ionut-t/perp/pkg/utils"
)

type Model struct {
	currentMenu *whichkey.Menu
	context     *whichkey.MenuContext
	width       int
	height      int
	styles      menuStyles
	visible     bool
}

// menuStyles defines the visual styling for menus
type menuStyles struct {
	Border      lipgloss.Style
	Title       lipgloss.Style
	Item        lipgloss.Style
	Key         lipgloss.Style
	Description lipgloss.Style
	Footer      lipgloss.Style
	Dimmed      lipgloss.Style
}

func defaultMenuStyles() menuStyles {
	return menuStyles{
		Border: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Primary.GetForeground()).
			Padding(1, 2),
		Title: styles.Primary.
			Bold(true).
			MarginBottom(1),
		Item:        styles.Text.Padding(0, 1),
		Key:         styles.Accent.Bold(true),
		Description: styles.Subtext0,
		Footer:      styles.Overlay1.MarginTop(1),
		Dimmed:      styles.Overlay0,
	}
}

// New creates a new which-key menu model
func New(menu *whichkey.Menu, width, height int) Model {
	return Model{
		currentMenu: menu,
		context:     whichkey.NewMenuContext(),
		width:       width,
		height:      height,
		styles:      defaultMenuStyles(),
		visible:     true,
	}
}

// SetContext updates the menu context for validation
func (m *Model) SetContext(ctx *whichkey.MenuContext) {
	m.context = ctx
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case whichkey.ShowSubmenuMsg:
		m.currentMenu = msg.Menu
	}
	return m, nil
}

// handleKeyPress processes keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (Model, tea.Cmd) {
	items := m.currentMenu.GetItems()

	switch msg.String() {
	case "esc":
		m.visible = false
		return m, utils.Dispatch(whichkey.CloseMenuMsg{})

	case "backspace":
		if m.currentMenu.Parent != nil {
			m.currentMenu = m.currentMenu.Parent
		} else {
			// No parent, close the menu
			m.visible = false
			return m, utils.Dispatch(whichkey.CloseMenuMsg{})
		}

		return m, nil

	default:
		// Check if key matches any menu item
		for _, item := range items {
			if msg.String() == item.Key {
				// Validate before execution
				if !item.Action.CanExecute(m.context) {
					return m, nil // Silently ignore invalid actions
				}

				if item.Action.IsSubmenu() {
					return m, utils.Dispatch(item.Action.Execute())
				} else {
					// Execute action and close menu
					m.visible = false
					actionMsg := item.Action.Execute()
					return m, utils.Dispatch(whichkey.ExecuteAndCloseMsg{ActionMsg: actionMsg})
				}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if !m.visible {
		return ""
	}

	menuItems := m.renderItems()

	// Assemble the menu
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.Title.Render(m.currentMenu.Title),
		lipgloss.JoinVertical(lipgloss.Left, menuItems...),
		m.renderFooter(),
	)

	// Add border
	bordered := m.styles.Border.Render(content)

	// Center the menu
	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		bordered,
	)
}

func (m Model) renderItems() []string {
	items := m.currentMenu.GetItems()

	// Build menu content
	var menuItems []string
	maxLabelWidth := 0

	// First pass: find max label width for alignment
	for _, item := range items {
		labelWidth := lipgloss.Width(item.Label)
		if labelWidth > maxLabelWidth {
			maxLabelWidth = labelWidth
		}
	}

	// Second pass: render items with aligned descriptions
	for _, item := range items {
		var itemStr string

		keyStr := m.styles.Key.Render("[" + item.Key + "]")
		paddedLabel := item.Label + lipgloss.NewStyle().
			Width(maxLabelWidth-lipgloss.Width(item.Label)).
			Render("")

		if item.Description != "" {
			descStr := m.styles.Description.Render(item.Description)
			itemStr = " " + keyStr + " " + paddedLabel + "  " + descStr + " "
		} else {
			itemStr = " " + keyStr + " " + paddedLabel + " "
		}
		itemStr = m.styles.Item.Render(itemStr)

		menuItems = append(menuItems, itemStr)
	}
	return menuItems
}

func (m Model) renderFooter() string {
	footerText := "Press [esc] to close"

	if m.currentMenu.Parent != nil {
		footerText = "Press [esc] to close, [backspace] to go back"
	}

	return m.styles.Footer.Render(footerText)
}

// SetMenu changes the current menu
func (m *Model) SetMenu(menu *whichkey.Menu) {
	m.currentMenu = menu
}

// GetCurrentMenu returns the current menu
func (m Model) GetCurrentMenu() *whichkey.Menu {
	return m.currentMenu
}

// Show makes the menu visible
func (m *Model) Show() {
	m.visible = true
}

// IsVisible returns whether the menu is visible
func (m Model) IsVisible() bool {
	return m.visible
}
