package history

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/coffee/styles"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/pkg/history"
	"github.com/ionut-t/perp/pkg/llm"
	"github.com/ionut-t/perp/pkg/utils"
	"github.com/ionut-t/perp/ui/markdown"
)

type SelectedMsg struct {
	Query string
}

type ClosedMsg struct{}

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

	fmt.Fprint(w, fn(str))
}

func New(entries []history.Entry, width, height int) Model {
	delegate := itemDelegate{
		styles: styles.ListItemStyles(),
	}

	ls := list.New(processEntries(entries), delegate, 0, 0)
	ls.Title = "History"
	ls.Styles = styles.ListStyles()

	ls.FilterInput.PromptStyle = styles.Accent
	ls.FilterInput.Cursor.Style = styles.Accent
	ls.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "select query"),
			),
		}
	}

	vp := viewport.New(0, 0)

	m := Model{
		width:    width,
		height:   height,
		list:     ls,
		viewport: vp,
		markdown: markdown.New(),
	}

	m.SetSize(width, height)

	return m
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	w, h := m.getAvailableSizes()
	lsWidth := max(50, w/3)

	m.list.SetSize(lsWidth, h)

	vpW := max(1, w-lsWidth-styles.ViewPadding.GetHorizontalFrameSize()-2)
	m.viewport.Width = vpW
	m.viewport.Height = max(1, h)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keymap.Submit):
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

			return m, utils.Dispatch(ClosedMsg{})
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
					m.viewport.YOffset = 0
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
				styles.Primary.Render("No history available."),
				"\n",
				styles.Subtext0.Render("Press 'q' to go back."),
			),
		)
	}

	listBorder := styles.ActiveBorder
	vpBorder := styles.InactiveBorder

	if m.focused != focusedList {
		listBorder = styles.InactiveBorder
		vpBorder = styles.ActiveBorder
	}

	return styles.ViewPadding.Render(lipgloss.JoinHorizontal(
		lipgloss.Left,
		listBorder.Width(m.list.Width()).Render(m.list.View()),
		" ",
		vpBorder.Width(lipgloss.Width(m.viewport.View())).Render(m.viewport.View()),
	))
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

	availableHeight := m.height - v - styles.ActiveBorder.GetBorderBottomSize()
	availableWidth := m.width - h

	return availableWidth, availableHeight
}
