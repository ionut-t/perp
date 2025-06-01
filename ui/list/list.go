package list

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Constants for consistent height calculations
const (
	MainBorderHeight  = 2 // Top and bottom border of main container
	FilterBarHeight   = 1 // Height of filter/help bar
	ItemBorderHeight  = 2 // Top and bottom border of each item
	ItemSpacing       = 1 // Spacing between items
	MinViewportHeight = 3 // Minimum usable viewport height
)

// Item represents a list item with variable content
type Item struct {
	Title       string
	Subtitle    string
	Description string
	Selected    bool
	Style       string
}

// KeyMap defines the key bindings for the list
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Select   key.Binding
	Filter   key.Binding
	Clear    key.Binding
	Quit     key.Binding
	Enter    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
}

// DefaultKeyMap returns the default key mappings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "select"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Clear: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear filter"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdown", "page down"),
		),
	}
}

// Styles defines the visual styling for the list
type Styles struct {
	Title          lipgloss.Style
	Subtitle       lipgloss.Style
	Description    lipgloss.Style
	SelectedBorder lipgloss.Style
	Cursor         lipgloss.Style
	Filter         lipgloss.Style
	Help           lipgloss.Style
	Border         lipgloss.Style
}

// DefaultStyles returns the default styling
func DefaultStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")),
		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
		Description: lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")),
		SelectedBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("212")).
			Padding(0, 1),
		Cursor: lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true),
		Filter: lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")),
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")),
	}
}

// Model represents the list component state
type Model struct {
	items          []Item
	filteredItems  []Item
	cursor         int
	viewportStart  int
	width          int
	height         int
	filterMode     bool
	filter         textinput.Model
	keys           KeyMap
	styles         Styles
	itemHeights    []int
	viewportHeight int
	totalHeight    int
}

// calculateViewportHeight calculates stable viewport height
func (m *Model) calculateViewportHeight() {
	// Total reserved height = main borders + filter bar + spacing
	reservedHeight := MainBorderHeight + FilterBarHeight + 1 // +1 for spacing

	// Calculate available height for content
	availableHeight := m.height - reservedHeight

	// Ensure minimum viewport height
	if availableHeight < MinViewportHeight {
		availableHeight = MinViewportHeight
	}

	m.viewportHeight = availableHeight
}

// New creates a new list model
func New(items []Item, width, height int) Model {
	filter := textinput.New()
	filter.Placeholder = "Filter"
	filter.Width = width - 4
	filter.CharLimit = 50

	m := Model{
		items:         items,
		filteredItems: items,
		width:         width,
		height:        height,
		keys:          DefaultKeyMap(),
		styles:        DefaultStyles(),
		filter:        filter,
	}

	m.calculateViewportHeight()
	m.calculateItemHeights()
	return m
}

// SetItems updates the list items
func (m *Model) SetItems(items []Item) {
	m.items = items
	m.applyFilter()
	m.calculateItemHeights()
	m.cursor = 0
	m.viewportStart = 0
}

// GetSelectedItem returns the currently selected item
func (m Model) GetSelectedItem() (Item, bool) {
	if len(m.filteredItems) == 0 || m.cursor >= len(m.filteredItems) {
		return Item{}, false
	}
	return m.filteredItems[m.cursor], true
}

// GetSelectedItems returns all selected items
func (m Model) GetSelectedItems() []Item {
	var selected []Item
	for _, item := range m.items {
		if item.Selected {
			selected = append(selected, item)
		}
	}
	return selected
}

// wrapText wraps text without breaking words
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		// If adding this word would exceed the width, start a new line
		if len(currentLine) > 0 && len(currentLine)+1+len(word) > width {
			lines = append(lines, currentLine)
			currentLine = word
		} else if len(currentLine) == 0 {
			currentLine = word
		} else {
			currentLine += " " + word
		}
	}

	// Add the last line if it has content
	if len(currentLine) > 0 {
		lines = append(lines, currentLine)
	}

	return lines
}

// calculateContentWidth returns the available width for item content
func (m *Model) calculateContentWidth() int {
	// Main borders (4) + item padding (4)
	return m.width - 8
}

// calculateItemHeights calculates the height each item will take
func (m *Model) calculateItemHeights() {
	m.itemHeights = make([]int, len(m.filteredItems))
	contentWidth := m.calculateContentWidth()
	m.totalHeight = 0

	for i, item := range m.filteredItems {
		height := 0

		// Title (always present)
		if item.Title != "" {
			title := item.Title
			if item.Selected {
				title = "✓ " + title
			}
			titleLines := wrapText(title, contentWidth)
			height += len(titleLines)
		}

		// Subtitle
		if item.Subtitle != "" {
			subtitleLines := wrapText(item.Subtitle, contentWidth)
			height += len(subtitleLines)
		}

		// Description (can wrap and have multiple paragraphs)
		if item.Description != "" {
			paragraphs := strings.Split(item.Description, "\n")
			for _, paragraph := range paragraphs {
				if paragraph == "" {
					height += 1
					continue
				}
				descLines := wrapText(paragraph, contentWidth)
				height += len(descLines)
			}
		}

		// Minimum height of 1, plus border and spacing
		if height == 0 {
			height = 1
		}
		height += ItemBorderHeight + ItemSpacing

		m.itemHeights[i] = height
		m.totalHeight += height
	}
}

// applyFilter filters items based on the current filter text
func (m *Model) applyFilter() {
	filterText := strings.ToLower(m.filter.Value())
	if filterText == "" {
		m.filteredItems = m.items
	} else {
		m.filteredItems = nil
		for _, item := range m.items {
			if strings.Contains(strings.ToLower(item.Title), filterText) ||
				strings.Contains(strings.ToLower(item.Subtitle), filterText) ||
				strings.Contains(strings.ToLower(item.Description), filterText) {
				m.filteredItems = append(m.filteredItems, item)
			}
		}
	}
	m.calculateItemHeights()

	// Reset cursor and viewport when filter changes
	m.cursor = 0
	m.viewportStart = 0
}

// adjustView adjusts the viewport to keep the cursor visible
func (m *Model) adjustView() {
	if len(m.filteredItems) == 0 || len(m.itemHeights) == 0 {
		m.viewportStart = 0
		return
	}

	// Ensure cursor is within bounds
	if m.cursor >= len(m.filteredItems) {
		m.cursor = len(m.filteredItems) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	// Calculate the position of the cursor item
	currentItemTop := 0
	for i := 0; i < m.cursor && i < len(m.itemHeights); i++ {
		currentItemTop += m.itemHeights[i]
	}

	currentItemBottom := currentItemTop
	if m.cursor < len(m.itemHeights) {
		currentItemBottom += m.itemHeights[m.cursor]
	}

	// Calculate current viewport bounds
	viewportTop := 0
	for i := 0; i < m.viewportStart && i < len(m.itemHeights); i++ {
		viewportTop += m.itemHeights[i]
	}
	viewportBottom := viewportTop + m.viewportHeight

	// Adjust viewport if cursor is out of bounds
	if currentItemTop < viewportTop {
		// Cursor is above viewport - scroll up
		m.viewportStart = m.cursor
	} else if currentItemBottom > viewportBottom {
		// Cursor is below viewport - scroll down to fit the item
		targetBottom := currentItemBottom
		targetTop := targetBottom - m.viewportHeight

		// Find which item index corresponds to targetTop
		runningHeight := 0
		newStart := 0
		for i := 0; i < len(m.itemHeights); i++ {
			if runningHeight >= targetTop {
				newStart = i
				break
			}
			runningHeight += m.itemHeights[i]
		}
		m.viewportStart = newStart
	}

	// Ensure viewportStart doesn't go beyond valid bounds
	if m.viewportStart >= len(m.filteredItems) {
		m.viewportStart = len(m.filteredItems) - 1
	}
	if m.viewportStart < 0 {
		m.viewportStart = 0
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filterMode {
			switch {
			case key.Matches(msg, m.keys.Clear):
				m.filterMode = false
				m.filter.Blur()
				m.filter.SetValue("")
				m.applyFilter()
				m.cursor = 0
				m.viewportStart = 0
				return m, nil
			case key.Matches(msg, m.keys.Enter):
				m.filterMode = false
				m.filter.Blur()
				m.applyFilter()
				m.cursor = 0
				m.viewportStart = 0
				return m, nil
			default:
				m.filter, cmd = m.filter.Update(msg)
				cmds = append(cmds, cmd)
				m.applyFilter()
				m.cursor = 0
				m.viewportStart = 0
			}
		} else {
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keys.Up):
				if m.cursor > 0 {
					m.cursor--
					m.adjustView()
				}
			case key.Matches(msg, m.keys.Down):
				if m.cursor < len(m.filteredItems)-1 {
					m.cursor++
					m.adjustView()
				}
			case key.Matches(msg, m.keys.PageUp):
				m.cursor -= 5
				if m.cursor < 0 {
					m.cursor = 0
				}
				m.adjustView()
			case key.Matches(msg, m.keys.PageDown):
				m.cursor += 5
				if m.cursor >= len(m.filteredItems) {
					m.cursor = len(m.filteredItems) - 1
				}
				if m.cursor < 0 {
					m.cursor = 0
				}
				m.adjustView()
			case key.Matches(msg, m.keys.Select):
				if m.cursor < len(m.filteredItems) {
					// Toggle selection in original items
					selectedTitle := m.filteredItems[m.cursor].Title
					for i := range m.items {
						if m.items[i].Title == selectedTitle {
							m.items[i].Selected = !m.items[i].Selected
							break
						}
					}
					// Update filtered items
					m.filteredItems[m.cursor].Selected = !m.filteredItems[m.cursor].Selected
				}
			case key.Matches(msg, m.keys.Filter):
				m.filterMode = true
				m.filter.Focus()
				return m, textinput.Blink
			case key.Matches(msg, m.keys.Clear):
				if m.filter.Value() != "" {
					m.filter.SetValue("")
					m.applyFilter()
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.calculateViewportHeight() // Recalculate viewport height first
		m.calculateItemHeights()    // Then recalculate item heights
		m.adjustView()              // Finally adjust the view
	}

	return m, tea.Batch(cmds...)
}

// renderItem renders a single item
func (m Model) renderItem(item Item, index int, isCursor bool) string {
	var parts []string
	contentWidth := m.calculateContentWidth()

	// Title
	if item.Title != "" {
		title := item.Title
		if item.Selected {
			title = "✓ " + title
		}
		titleLines := wrapText(title, contentWidth)
		for _, line := range titleLines {
			parts = append(parts, m.styles.Title.Render(line))
		}
	}

	// Subtitle
	if item.Subtitle != "" {
		subtitleLines := wrapText(item.Subtitle, contentWidth)
		for _, line := range subtitleLines {
			parts = append(parts, m.styles.Subtitle.Render(line))
		}
	}

	// Description (with wrapping and paragraph support)
	if item.Description != "" {
		paragraphs := strings.Split(item.Description, "\n")
		for _, paragraph := range paragraphs {
			if paragraph == "" {
				parts = append(parts, "")
				continue
			}

			descLines := wrapText(paragraph, contentWidth)
			for _, line := range descLines {
				parts = append(parts, m.styles.Description.Render(line))
			}
		}
	}

	content := strings.Join(parts, "\n")

	// Calculate the width for the item (full available width minus main borders)
	itemWidth := m.width - 4

	// Create border style (visible for selected, hidden for others)
	var borderStyle lipgloss.Style
	if isCursor {
		borderStyle = m.styles.SelectedBorder.Copy().Width(itemWidth)
	} else {
		// Hidden border - same structure but transparent
		borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("0")). // Transparent/hidden
			Padding(0, 1).
			Width(itemWidth)
	}

	return borderStyle.Render(content)
}

// View renders the list
func (m Model) View() string {
	if len(m.filteredItems) == 0 {
		noItems := "No items to display"
		if m.filter.Value() != "" {
			noItems = "No items match filter: " + m.filter.Value()
		}

		// Create filter bar
		var filterBar string
		if m.filterMode {
			filterBar = m.styles.Filter.Render("Filter: ") + m.filter.View()
		} else if m.filter.Value() != "" {
			filterBar = m.styles.Filter.Render("Filter: " + m.filter.Value() + " (press / to edit, esc to clear)")
		} else {
			filterBar = m.styles.Help.Render("Press / to filter, space to select, q to quit")
		}

		content := lipgloss.NewStyle().
			Width(m.width - 4).
			Height(m.viewportHeight).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Render(noItems)

		fullContent := content + "\n" + filterBar
		return m.styles.Border.Width(m.width).Render(fullContent)
	}

	// Build the viewport content line by line
	var viewportLines []string
	currentHeight := 0
	itemIndex := m.viewportStart

	// Track where we are in the current item
	currentItemLines := []string{}
	if itemIndex < len(m.filteredItems) {
		itemContent := m.renderItem(m.filteredItems[itemIndex], itemIndex, itemIndex == m.cursor)
		currentItemLines = strings.Split(itemContent, "\n")
	}
	currentLineInItem := 0

	// Fill the viewport line by line
	for currentHeight < m.viewportHeight && itemIndex < len(m.filteredItems) {
		// If we've consumed all lines of current item, move to next
		if currentLineInItem >= len(currentItemLines) {
			itemIndex++
			if itemIndex < len(m.filteredItems) {
				itemContent := m.renderItem(m.filteredItems[itemIndex], itemIndex, itemIndex == m.cursor)
				currentItemLines = strings.Split(itemContent, "\n")
				currentLineInItem = 0
			} else {
				break
			}
		}

		// Add the current line
		if currentLineInItem < len(currentItemLines) {
			viewportLines = append(viewportLines, currentItemLines[currentLineInItem])
			currentLineInItem++
			currentHeight++
		}
	}

	// Pad the viewport to maintain consistent height
	for currentHeight < m.viewportHeight {
		viewportLines = append(viewportLines, "")
		currentHeight++
	}

	content := strings.Join(viewportLines, "\n")

	// Add filter bar
	var filterBar string
	if m.filterMode {
		filterBar = m.styles.Filter.Render("Filter: ") + m.filter.View()
	} else if m.filter.Value() != "" {
		filterBar = m.styles.Filter.Render("Filter: " + m.filter.Value() + " (press / to edit, esc to clear)")
	} else {
		filterBar = m.styles.Help.Render("Press / to filter, space to select, q to quit")
	}

	// Combine content
	listContent := content + "\n" + filterBar

	return m.styles.Border.Width(m.width).Render(listContent)
}

// SetSize updates the component size
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.calculateViewportHeight() // Always recalculate viewport height first
	m.calculateItemHeights()    // Then item heights
	m.adjustView()              // Finally adjust the view
}

// SetKeyMap sets custom key bindings
func (m *Model) SetKeyMap(keys KeyMap) {
	m.keys = keys
}

// SetStyles sets custom styling
func (m *Model) SetStyles(styles Styles) {
	m.styles = styles
}
