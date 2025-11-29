package export_data

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/coffee/styles"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/internal/whichkey"
	"github.com/ionut-t/perp/pkg/server"
	"github.com/ionut-t/perp/pkg/utils"
	"github.com/ionut-t/perp/store/export"
	"github.com/ionut-t/perp/tui/common/splitview"
)

// Wrapper to implement splitview.Item interface for Record
type recordItem struct {
	*export.Record
}

func (r recordItem) GetName() string     { return r.Name }
func (r recordItem) GetContent() string  { return r.Content }
func (r recordItem) SetContent(c string) { r.Content = c }

// Adapter store to work with splitview.Store interface
type storeAdapter struct {
	export.Store
}

func (s *storeAdapter) Load() ([]recordItem, error) {
	records, err := s.Store.Load()
	if err != nil {
		return nil, err
	}
	items := make([]recordItem, len(records))
	for i := range records {
		items[i] = recordItem{&records[i]}
	}
	return items, nil
}

func (s *storeAdapter) Update(item recordItem) error {
	return s.Store.Update(*item.Record)
}

func (s *storeAdapter) Delete(item recordItem) error {
	return s.Store.Delete(*item.Record)
}

func (s *storeAdapter) Rename(item *recordItem, newName string) error {
	return s.Store.Rename(item.Record, newName)
}

func (s *storeAdapter) GetCurrent() recordItem {
	rec := s.GetCurrentRecord()
	return recordItem{&rec}
}

func (s *storeAdapter) SetCurrentName(name string) {
	s.SetCurrentRecordName(name)
}

func (s *storeAdapter) GetPath(item recordItem) string {
	return s.Store.GetPath(*item.Record)
}

type Model struct {
	*splitview.Model[recordItem, *storeAdapter]
	server server.Server
}

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

func New(store export.Store, server server.Server, width, height int) Model {
	adapter := &storeAdapter{Store: store}

	config := splitview.Config{
		EditorLanguage:      "json",
		PlaceholderTitle:    "No data exported.",
		PlaceholderSubtitle: "Press '<leader>c' to go back.",
		SuccessDeleteMsg:    "Record deleted successfully.",
		SuccessRenameMsg:    "Record renamed successfully.",
	}

	baseModel := splitview.New(
		adapter,
		config,
		processRecords,
		func(m *splitview.Model[recordItem, *storeAdapter]) string {
			return renderStatusBar(m, server)
		},
		func(m *splitview.Model[recordItem, *storeAdapter]) string {
			return renderHelp(m)
		},
		func() tea.Msg {
			return whichkey.CloseExportCmd()
		},
		width,
		height,
	)

	// Set the custom list selection handler
	baseModel.OnListSelection = func(m *splitview.Model[recordItem, *storeAdapter], listItem list.Item) {
		if i, ok := listItem.(item); ok {
			m.GetStore().SetCurrentName(i.Title())
		}

		// Update editor with current content
		current := m.GetStore().GetCurrent()
		m.GetEditor().SetContent(current.GetContent())

		// Set language based on file extension
		path := m.GetStore().GetPath(current)
		lang := getLanguageForEditor(path)
		m.SetLanguage(lang)
	}

	m := Model{
		Model:  baseModel,
		server: server,
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return m.Model.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case whichkey.ExternalEditorMsg:
		updatedBase, cmd := m.OpenInExternalEditor()
		m.Model = updatedBase.(*splitview.Model[recordItem, *storeAdapter])
		return m, cmd

	case tea.KeyMsg:
		if m.GetEditor().IsNormalMode() && key.Matches(msg, keymap.Editor) {
			return m, utils.Dispatch(whichkey.ExternalEditorCmd())
		}
	}

	// Delegate to base model
	updatedModel, cmd := m.Model.Update(msg)
	m.Model = updatedModel.(*splitview.Model[recordItem, *storeAdapter])
	return m, cmd
}

func (m Model) View() string {
	return m.Model.View()
}

func processRecords(records []recordItem) []list.Item {
	items := make([]list.Item, 0, len(records))

	for _, record := range records {
		items = append(items, item{
			title: record.Name,
			desc:  fmt.Sprintf("Last modified: %s", record.UpdatedAt.Format("02/01/2006 15:04")),
		})
	}

	return items
}

func renderStatusBar(m *splitview.Model[recordItem, *storeAdapter], server server.Server) string {
	bg := styles.Surface0.GetBackground()

	separator := styles.Surface0.Render(" | ")

	serverName := styles.Primary.Background(bg).Render(server.Name)

	database := styles.Accent.Background(bg).Render(server.Database)

	left := serverName + separator + database

	leftInfo := styles.Surface0.Padding(0, 1).Render(left)

	helpText := styles.Info.Background(bg).PaddingRight(1).Render("<leader>? Help")

	displayedInfoWidth := m.GetWidth() -
		lipgloss.Width(leftInfo) -
		lipgloss.Width(helpText) -
		lipgloss.Width(separator)

	spaces := styles.Surface0.Render(strings.Repeat(" ", max(0, displayedInfoWidth)))

	return styles.Surface0.Width(m.GetWidth()).Render(
		lipgloss.JoinHorizontal(
			lipgloss.Right,
			leftInfo,
			spaces,
			helpText,
		),
	)
}

func renderHelp(m *splitview.Model[recordItem, *storeAdapter]) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		renderUsefulHelp(m),
		splitview.RenderCommonListHelp(m.GetWidth(), *m.GetList()),
		splitview.RenderCommonEditorHelp(m.GetWidth()),
	)
}

func renderUsefulHelp(m *splitview.Model[recordItem, *storeAdapter]) string {
	bindings := []key.Binding{
		key.NewBinding(
			key.WithKeys("<leader>c"),
			key.WithHelp("leader>c", "go back to the main view"),
		),
		keymap.ForceQuit,
		splitview.ChangeFocused,
		keymap.Editor,
	}

	return splitview.RenderCommonUsefulHelp(m.GetWidth(), bindings)
}

func getLanguageForEditor(path string) string {
	lang := "json"
	if filepath.Ext(path) == ".json" {
		lang = "json"
	} else if filepath.Ext(path) == ".csv" {
		lang = "csv"
	}
	return lang
}

func (m Model) CanTriggerLeaderKey() bool {
	return m.Model.CanTriggerLeaderKey()
}

func (m *Model) HandleHelpToggle() {
	m.Model.HandleHelpToggle()
}
