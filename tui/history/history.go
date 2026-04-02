package history

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ionut-t/coffee/styles"
	"github.com/ionut-t/perp/internal/debug"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/internal/whichkey"
	"github.com/ionut-t/perp/pkg/history"
	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/pkg/utils"
	"github.com/ionut-t/perp/ui/markdown"
)

var (
	splitViewSeparator      = " "
	splitViewSeparatorWidth = lipgloss.Width(splitViewSeparator)
	minListWidth            = 50
)

type SelectedMsg struct {
	Query string
}

type focused int

const (
	focusedList focused = iota
	focusedViewport
)

type Model struct {
	width, height int
	list          list.Model
	err           error
	viewport      viewport.Model
	focused       focused
	markdown      markdown.Model
	styles        styles.Styles
}

type item struct {
	title, query string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return i.query }

type itemDelegate struct {
	styles list.DefaultItemStyles
}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 1 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d) %s", index+1, i.title)

	fn := d.styles.NormalTitle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return d.styles.SelectedTitle.Render(s...)
		}
	}

	_, _ = fmt.Fprint(w, fn(str))
}

func New(entries []history.Entry, width, height int) Model {
	ls := list.New(processEntries(entries), list.NewDefaultDelegate(), 0, 0)
	ls.Title = "History"

	ls.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "select query"),
			),
		}
	}

	vp := viewport.New()

	m := Model{
		width:    width,
		height:   height,
		list:     ls,
		viewport: vp,
	}

	m.SetSize(width, height)

	return m
}

func (m *Model) SetStyles(s styles.Styles, isDark bool) {
	m.styles = s
	m.list.Styles = styles.ListStyles(s, isDark)
	delegate := itemDelegate{
		styles: styles.ListItemStyles(s, isDark),
	}
	m.list.SetDelegate(delegate)
	m.markdown = markdown.New(isDark)
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	w, h := m.getAvailableSizes()

	horizontalFrameBorderSize := m.styles.ActiveBorder.GetHorizontalFrameSize()

	listWidth := max(minListWidth, w/3)
	detailWidth := w - listWidth - splitViewSeparatorWidth

	// Content dimensions inside borders
	listContentWidth := listWidth - horizontalFrameBorderSize
	detailContentWidth := detailWidth - horizontalFrameBorderSize

	borderV := m.styles.ActiveBorder.GetVerticalFrameSize()
	paneContentHeight := h - borderV

	m.list.SetSize(listContentWidth, paneContentHeight)

	m.viewport.SetWidth(detailContentWidth)
	m.viewport.SetHeight(paneContentHeight)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	debug.Printf("History Update: received message of type %T", msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keymap.Submit):
			if m.list.FilterState() == list.Filtering {
				break
			}

			selected := m.list.SelectedItem()
			if selected != nil {
				if item, ok := selected.(item); ok {
					return m, utils.Dispatch(SelectedMsg{
						Query: item.query,
					})
				}
			}

		case key.Matches(msg, keymap.Quit) || key.Matches(msg, keymap.Cancel):
			if m.list.FilterState() == list.Filtering {
				break
			}

			return m, utils.Dispatch(whichkey.CloseHistoryCmd())
		}

		switch msg.String() {
		case "tab":
			if m.focused == focusedList {
				m.focused = focusedViewport
			} else {
				m.focused = focusedList
			}
		}
	}

	switch m.focused {
	case focusedList:
		ls, cmd := m.list.Update(msg)
		m.list = ls
		selected := m.list.SelectedItem()
		cmds = append(cmds, cmd)
		if selected != nil {
			if item, ok := selected.(item); ok {
				query := fmt.Sprintf("```sql\n%s\n```", item.query)

				for _, key := range llm.LLMKeywords {
					if strings.HasPrefix(strings.ToLower(item.query), key) {
						query = strings.TrimPrefix(item.query, key)
						query = fmt.Sprintf("`%s` %s", key, strings.TrimSpace(query))
					}
				}

				if out, err := m.markdown.Render(query); err != nil {
					m.err = err
				} else {
					m.viewport.SetContent(out)
					m.viewport.SetYOffset(0)
				}
			}
		}
	case focusedViewport:
		vp, cmd := m.viewport.Update(msg)
		m.viewport = vp
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if len(m.list.Items()) == 0 {
		return styles.ViewPadding.Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.styles.Primary.Render("No history available."),
				"\n",
				m.styles.Subtext0.Render("Press 'q' to go back."),
			),
		)
	}

	return m.getSplitView()
}

func (m *Model) getSplitView() string {
	availableWidth, availableHeight := m.getAvailableSizes()

	listWidth := max(minListWidth, availableWidth/3)
	detailWidth := availableWidth - listWidth - splitViewSeparatorWidth

	borderV := m.styles.ActiveBorder.GetVerticalFrameSize()
	paneContentHeight := availableHeight - borderV

	listBorder := m.styles.InactiveBorder
	detailBorder := m.styles.InactiveBorder

	if m.focused == focusedList {
		listBorder = m.styles.ActiveBorder
	} else {
		detailBorder = m.styles.ActiveBorder
	}

	joinedContent := lipgloss.JoinHorizontal(
		lipgloss.Left,
		listBorder.
			Width(listWidth).
			Height(paneContentHeight).
			Render(m.list.View()),
		splitViewSeparator,
		detailBorder.
			Width(detailWidth).
			Height(paneContentHeight).
			Render(m.viewport.View()),
	)

	return lipgloss.NewStyle().Padding(0, 1).Render(joinedContent)
}

func processEntries(entries []history.Entry) []list.Item {
	items := make([]list.Item, len(entries))
	for i, entry := range entries {
		items[i] = item{
			title: entry.Time.Format("02/01/2006 15:04:05"),
			query: entry.Query,
		}
	}
	return items
}

func (m *Model) getAvailableSizes() (int, int) {
	h, v := styles.ViewPadding.GetFrameSize()

	availableWidth := m.width - h

	return availableWidth, m.height - v
}

func (m *Model) CanTriggerLeaderKey() bool {
	return m.list.FilterState() != list.Filtering
}
