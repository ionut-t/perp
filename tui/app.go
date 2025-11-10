package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/coffee/styles"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"github.com/ionut-t/perp/internal/config"
	"github.com/ionut-t/perp/internal/keymap"
	"github.com/ionut-t/perp/internal/leader"
	"github.com/ionut-t/perp/internal/whichkey"
	"github.com/ionut-t/perp/pkg/db"
	"github.com/ionut-t/perp/pkg/export"
	"github.com/ionut-t/perp/pkg/history"
	"github.com/ionut-t/perp/pkg/llm"
	llmFactory "github.com/ionut-t/perp/pkg/llm/llm_factory"
	"github.com/ionut-t/perp/pkg/psql"
	"github.com/ionut-t/perp/pkg/server"
	"github.com/ionut-t/perp/pkg/utils"
	exportStore "github.com/ionut-t/perp/store/export"
	"github.com/ionut-t/perp/tui/command"
	"github.com/ionut-t/perp/tui/content"
	exportData "github.com/ionut-t/perp/tui/export_data"
	historyView "github.com/ionut-t/perp/tui/history"
	"github.com/ionut-t/perp/tui/menu"
	"github.com/ionut-t/perp/tui/prompt"
	"github.com/ionut-t/perp/tui/servers"
	"github.com/ionut-t/perp/ui/help"
)

type schemaFetchedMsg string

type schemaFailureMsg struct {
	err error
}

type llmResponseMsg llm.Response

type llmFailureMsg struct {
	err error
}

type executeQueryMsg content.ParsedQueryResult

type queryFailureMsg struct {
	err error
}

type llmSharedSchemaMsg struct {
	schema  string
	message string
	tables  []string
}

type notificationErrorMsg struct {
	err error
}

type view int

const (
	viewServers view = iota
	viewMain
	viewExportData
	viewHelp
	viewHistory
)

type focused int

const (
	focusedNone focused = iota
	focusedEditor
	focusedContent
	focusedCommand
	focusedHistory
)

const (
	editorMinHeight        = 10
	editorHalfScreenOffset = 4 // Offset for editor in split view (accounts for borders/padding)
)

type model struct {
	config          config.Config
	width, height   int
	view            view
	focused         focused
	serverSelection servers.Model
	server          server.Server
	db              db.Database
	error           error
	llm             llm.LLM
	llmError        error
	editor          editor.Model

	fullScreen bool

	loading bool
	spinner spinner.Model

	exportData            exportData.Model
	command               command.Model
	notification          string
	content               content.Model
	help                  help.Model
	llmSharedTablesSchema []string

	// styles
	llmKeywords  map[string]lipgloss.Style
	psqlCommands map[string]lipgloss.Style

	// commands
	expandedDisplay bool

	// history management
	historyLogs           []history.Entry
	currentHistoryIndex   int
	historyNavigating     bool
	originalEditorContent string
	history               historyView.Model

	// navigation components
	leaderMgr    *leader.Manager
	whichKeyMenu menu.Model
	menuRegistry *whichkey.Registry
	showingMenu  bool

	prompt         prompt.Model
	isPromptActive bool
}

func New(config config.Config) model {
	editor := editor.New(80, 10)

	llmKeywordsMap := make(map[string]lipgloss.Style, len(llm.LLMKeywords))
	for _, keyword := range llm.LLMKeywords {
		llmKeywordsMap[keyword] = styles.Accent.Bold(true)
	}

	psqlCommands := make(map[string]lipgloss.Style, len(psql.PSQL_COMMANDS))

	for cmd := range psql.PSQL_COMMANDS {
		psqlCommands[cmd] = styles.Primary.Bold(true)
	}

	editor.SetPlaceholder("Type your SQL query here...")

	editor.Focus()
	editor.DisableCommandMode(true)
	editor.WithTheme(styles.EditorTheme())
	editor.SetLanguage("postgres", styles.EditorLanguageTheme())

	historyLogs, err := history.Get(config.Storage())
	if err != nil {
		historyLogs = []history.Entry{}
	}

	llm, err := llmFactory.New(context.Background(), config, config.GetLLMInstructions())

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styles.Primary

	// Initialize navigation components
	menuRegistry := whichkey.NewRegistry()

	return model{
		config:          config,
		llm:             llm,
		editor:          editor,
		llmKeywords:     llmKeywordsMap,
		psqlCommands:    psqlCommands,
		command:         command.New(),
		serverSelection: servers.New(config.Storage()),
		historyLogs:     historyLogs,
		content:         content.New(0, 0),
		help:            help.New(),
		llmError:        err,
		spinner:         sp,
		leaderMgr:       leader.NewManager(500*time.Millisecond, config.GetLeaderKey()),
		whichKeyMenu:    menu.New(menuRegistry.GetRootMenu(), 0, 0),
		menuRegistry:    menuRegistry,
		showingMenu:     false,
		prompt:          prompt.New(),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("perp"),
		m.spinner.Tick,
		m.editor.CursorBlink(),
		m.checkForUpdates(),
	)
}

func (m *model) updateSize() {
	width, height := m.getAvailableSizes()

	commandLineHeight := lipgloss.Height(m.command.View())

	if m.fullScreen {
		fullScreenHeight := height - commandLineHeight

		if m.editor.IsFocused() {
			m.editor.SetSize(width, fullScreenHeight)
			return
		}

		m.content.SetSize(width, fullScreenHeight)
		return
	}

	editorHeight := max(height/2-editorHalfScreenOffset, editorMinHeight)
	m.editor.SetSize(width, editorHeight)

	contentHeight := height - editorHeight - commandLineHeight

	m.content.SetSize(width, contentHeight)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		width, height := m.getAvailableSizes()
		m.updateSize()

		m.help.SetSize(msg.Width, msg.Height)
		m.help.SetContent(m.renderHelp())

		if m.view == viewHistory {
			m.history.SetSize(width, height)
		}

		m.prompt.SetSize(width, height)

		// Update which-key menu size
		var cmd tea.Cmd
		m.whichKeyMenu, cmd = m.whichKeyMenu.Update(msg)
		if cmd != nil {
			return m, cmd
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// Priority 1: Which-key menu is showing - let it handle all keys
		if m.showingMenu {
			return m.handleWhichKeyPress(msg)
		}

		// Priority 2: Leader key handling
		if m.canTriggerLeaderKey() {
			if m.leaderMgr.IsActive() {
				return m.handleLeaderSequence(msg)
			}

			if m.leaderMgr.IsLeaderKey(msg.String()) {
				return m.handleLeaderKeyPress(msg)
			}
		}

		if m.historyNavigating && m.editor.IsFocused() && m.focused == focusedEditor {
			// Check if it's a character input (not a special key)
			if len(msg.String()) == 1 || msg.Type == tea.KeySpace {
				// User is typing, exit history navigation
				m.resetHistory()
			}
		}

		if msg.Type == tea.KeyCtrlC {
			m.closeDbConnection()
			return m, tea.Quit
		}

		if m.focused == focusedCommand ||
			m.view == viewServers ||
			m.view == viewExportData ||
			m.view == viewHistory {
			break
		}

		switch {
		case key.Matches(msg, keymap.Quit):
			if m.error != nil {
				m.serverSelection = servers.New(m.config.Storage())
				_, cmd := m.serverSelection.Update(nil)

				m.view = viewServers
				m.error = nil
				return m, cmd
			}

			if m.focused == focusedContent && m.content.IsViewChangeRequired() ||
				m.editor.IsNormalMode() && m.fullScreen {
				m.fullScreen = false

				m.updateSize()
				contentModel, cmd := m.content.Update(content.ResizeMsg{})
				m.content = contentModel.(content.Model)

				return m, cmd
			}

			if m.view == viewHelp {
				m.view = viewMain
				m.focused = focusedEditor
				m.editor.Focus()
				break
			}

		case key.Matches(msg, changeFocused):
			if m.view == viewMain && !m.editor.IsInsertMode() {
				switch m.focused {
				case focusedEditor:
					m.focused = focusedContent
					m.editor.Blur()
				case focusedContent:
					m.focused = focusedEditor
					m.editor.Focus()
				}

				if m.fullScreen {
					m.updateSize()
				}

				_, cmd := m.content.Update(nil)

				return m, tea.Batch(
					cmd,
					m.editor.CursorBlink(),
					utils.Dispatch(content.ResizeMsg{}),
				)
			}

		case key.Matches(msg, enterCommand):
			if m.view == viewMain && m.editor.IsNormalMode() {
				m.focused = focusedCommand
				m.editor.Blur()

				ed, cmd := m.editor.Update(nil)
				m.editor = ed.(editor.Model)

				return m, tea.Batch(
					m.command.Focus(),
					cmd,
				)
			}

		case key.Matches(msg, keymap.Insert):
			if m.view == viewMain && m.focused == focusedContent {
				m.focused = focusedEditor
				m.editor.Focus()
				m.editor.SetInsertMode()

				_, cmd := m.editor.Update(nil)

				return m, tea.Batch(
					cmd,
					m.editor.CursorBlink(),
				)
			}

		case key.Matches(msg, keymap.Submit):
			if m.editor.IsNormalMode() {
				content := m.editor.GetCurrentContent()

				if content == "" {
					break
				}

				if !m.loading {
					m.loading = true
					m.resetHistory()
					m.addToHistory()
					m.fullScreen = false
					m.updateSize()

					return m, m.sendQueryCmd()
				}
			}

		case key.Matches(msg, executeQuery):
			if !m.loading {
				m.loading = true
				m.resetHistory()
				m.addToHistory()
				m.fullScreen = false
				m.updateSize()

				return m, m.sendQueryCmd()
			}

		case key.Matches(msg, keymap.Cancel):
			if m.view == viewMain && m.focused == focusedEditor {
				m.resetHistory()

				if m.editor.IsNormalMode() {
					if m.editor.IsFocused() {
						m.focused = focusedContent
						m.editor.Blur()
					}
				}
			}

		case key.Matches(msg, previousHistory):
			if m.editor.IsFocused() && len(m.historyLogs) > 0 {
				m.previousHistory()
			}

		case key.Matches(msg, nextHistory):
			if m.editor.IsFocused() && m.historyNavigating {
				m.nextHistory()
			}

		case key.Matches(msg, viewHistoryEntries):
			if entries, err := history.Get(m.config.Storage()); err != nil {
				m.content.SetError(err)
			} else {
				m.view = viewHistory
				m.focused = focusedHistory
				m.editor.Blur()
				m.historyLogs = entries

				m.history = historyView.New(entries, m.width, m.height)
			}

		case key.Matches(msg, openRelease):
			return m.openReleaseNotes()

		case key.Matches(msg, dismissUpdate):
			return m.dismissUpdate()
		}

	case servers.SelectedServerMsg:
		m.closeDbConnection()
		m.view = viewMain
		m.focused = focusedEditor
		m.loading = true
		m.server = msg.Server
		m.db, m.error = db.New(m.server.String())
		if m.error == nil {
			m.content.SetConnectionInfo(m.server)

			if m.server.ShareDatabaseSchemaLLM {
				m.editor.SetPlaceholder("Type your SQL query or /ask your question here...")
			} else {
				m.editor.SetPlaceholder("Type your SQL query")
			}

			return m, m.generateSchema()
		}
		m.loading = false

	case utils.ClearMsg:
		m.notification = ""

	case updateAvailableMsg:
		m.content.SetLatestReleaseInfo(msg.release)

	case schemaFetchedMsg:
		schema := string(msg)
		m.loading = false

		m.content.SetSchema(schema)

	case schemaFailureMsg:
		m.loading = false
		m.content.SetError(msg.err)

	case executeQueryMsg:
		m.loading = false
		m.editor.SetContent("")

		err := m.content.SetQueryResults(content.ParsedQueryResult(msg))
		if err != nil {
			return m, nil
		}

		m.focused = focusedContent
		m.editor.Blur()
		m.editor.SetNormalMode()

		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)

		message := fmt.Sprintf("Query executed successfully. Affected rows: %d", msg.AffectedRows)

		if m.server.TimingEnabled {
			message += fmt.Sprintf(". Execution time: %s", utils.Duration(msg.ExecutionTime))
		}

		var schemaCmd tea.Cmd
		if msg.Type == db.QueryCreate ||
			msg.Type == db.QueryDrop ||
			msg.Type == db.QueryAlter {
			schemaCmd = m.generateSchema()
		}

		return m, tea.Batch(
			cmd,
			m.successNotification(message),
			schemaCmd,
		)

	case queryFailureMsg:
		m.loading = false
		m.content.SetError(msg.err)

	case psqlCommandMsg:
		m.loading = true
		return m, m.runPsqlCommand(msg.command)

	case psqlResultMsg:
		m.loading = false
		m.editor.SetContent("")

		var timingCmd tea.Cmd
		if m.server.TimingEnabled {
			timingCmd = m.successNotification(fmt.Sprintf("Execution time: %s", utils.Duration(msg.result.ExecutionTime)))
		}

		m.content.SetPsqlResult(msg.result)

		m.focused = focusedContent
		m.editor.Blur()
		m.editor.SetNormalMode()

		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)

		return m, tea.Batch(
			cmd,
			timingCmd,
		)

	case psqlErrorMsg:
		m.loading = false
		m.content.SetError(msg.err)

	case toggleExpandedMsg:
		m.loading = false
		m.expandedDisplay = !m.expandedDisplay
		m.content.SetExpandedDisplay(m.expandedDisplay)
		status := "OFF"
		if m.expandedDisplay {
			status = "ON"
		}

		m.editor.SetContent("")

		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)

		return m, tea.Batch(
			cmd,
			m.successNotification(fmt.Sprintf("Expanded display is %s", status)),
		)

	case toggleTimingMsg:
		m.loading = false
		enabled := !m.server.TimingEnabled
		if err := m.server.ToggleTiming(m.config.Storage()); err != nil {
			m.server.TimingEnabled = enabled
		}
		status := "OFF"
		if m.server.TimingEnabled {
			status = "ON"
		}

		m.editor.SetContent("")

		m.editor.SetContent("")

		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)

		return m, tea.Batch(
			cmd,
			m.successNotification(fmt.Sprintf("Timing is %s", status)),
		)

	case showPsqlHelpMsg:
		m.content.ShowPsqlHelp()
		m.editor.SetContent("")
		return m, nil

	case llmResponseMsg:
		m.loading = false
		query := strings.TrimSpace(m.editor.GetCurrentContent())
		m.content.SetLLMLogs(llm.Response(msg), query)

		if msg.Command == llm.Optimise || msg.Command == llm.Fix {
			content := llm.ExtractQuery(string(msg.Response))
			m.editor.SetContent(content)
		} else {
			m.editor.SetContent("")
			m.editor.Blur()
			m.focused = focusedContent
		}

		m.editor.SetNormalMode()

		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)
		return m, cmd

	case llmFailureMsg:
		m.loading = false
		query := m.editor.GetCurrentContent()

		for _, keyword := range llm.LLMKeywords {
			query = strings.TrimPrefix(query, keyword)
		}

		query = strings.TrimSpace(query)
		m.content.SetLLMLogsError(msg.err, query)

	case llmSharedSchemaMsg:
		m.loading = false
		m.content.SetLLMSharedSchema(msg.schema)
		m.llmSharedTablesSchema = msg.tables
		m.content.SetLLMSharedTables(m.llmSharedTablesSchema)

		m.editor.SetContent("")
		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)

		return m, tea.Batch(
			cmd,
			m.successNotification(msg.message),
		)

	case notificationErrorMsg:
		m.loading = false
		return m, m.errorNotification(msg.err)

	case content.LLMResponseSelectedMsg:
		m.editor.SetContent(msg.Response)
		m.editor.Focus()
		_ = m.editor.SetCursorPositionEnd()
		m.view = viewMain
		m.focused = focusedEditor
		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)

		return m, tea.Batch(
			cmd,
			m.editor.CursorBlink(),
		)

	case command.QuitMsg, psqlQuitMsg:
		m.closeDbConnection()
		return m, tea.Quit

	case command.CancelMsg:
		m.focused = focusedEditor
		m.editor.Focus()

	case command.ExportMsg:
		return m.handleDataExport(msg)

	case command.EditorChangedMsg:
		err := m.config.SetEditor(msg.Editor)
		if err != nil {
			return m, m.errorNotification(err)
		}

		m.focused = focusedEditor
		m.editor.Focus()
		return m, m.successNotification(
			fmt.Sprintf("Editor changed to %s", msg.Editor),
		)

	case command.LLMUseDatabaseSchemaMsg:
		if m.llmError != nil {
			return m, m.errorNotification(fmt.Errorf("LLM is not configured: %w", m.llmError))
		}

		done := func() {
			m.content.SetConnectionInfo(m.server)
			m.focused = focusedEditor
			m.editor.Focus()
		}

		if m.server.ShareDatabaseSchemaLLM == msg.Enabled {
			done()
			return m, m.successNotification("No change in LLM database schema usage")
		}

		if err := m.server.EnableDatabaseSchemaLLM(msg.Enabled, m.config.Storage()); err != nil {
			return m, m.errorNotification(err)
		}

		if msg.Enabled {
			done()
			return m, m.successNotification("LLM will now use the database schema")
		}

		done()
		m.llm.ResetInstructions()
		m.llmSharedTablesSchema = []string{}
		m.content.SetLLMSharedSchema("")
		m.content.SetLLMSharedTables(m.llmSharedTablesSchema)
		return m, m.successNotification("LLM will no longer use the database schema")

	case command.LLMModelChangedMsg:
		if m.llmError != nil {
			return m, m.errorNotification(fmt.Errorf("LLM is not configured: %w", m.llmError))
		}

		existingModel, _ := m.config.GetLLMModel()
		if existingModel == msg.Model {
			return m, m.successNotification("LLM model is already set to " + msg.Model)
		}

		if err := m.llm.SetModel(msg.Model); err != nil {
			return m, m.errorNotification(fmt.Errorf("invalid LLM model: %w", err))
		}

		if err := m.config.SetLLMModel(msg.Model); err != nil {
			return m, m.errorNotification(err)
		}

		m.focused = focusedEditor
		m.editor.Focus()
		return m, m.successNotification("LLM model changed to " + msg.Model)

	case command.LeaderKeyChangedMsg:
		existingLeader := m.config.GetLeaderKey()
		if existingLeader == msg.Key {
			return m, nil
		}

		if err := m.config.SetLeaderKey(msg.Key); err != nil {
			return m, m.errorNotification(err)
		}

		m.leaderMgr.SetLeaderKey(msg.Key)
		return m, m.successNotification("Leader key changed")

	case command.ErrorMsg:
		return m, m.errorNotification(msg.Err)

	case historyView.SelectedMsg:
		m.editor.SetContent(msg.Query)
		m.editor.Focus()
		_ = m.editor.SetCursorPositionEnd()
		m.view = viewMain
		m.focused = focusedEditor
		ed, cmd := m.editor.Update(nil)
		m.editor = ed.(editor.Model)
		return m, tea.Batch(
			cmd,
			m.editor.CursorBlink(),
		)

	// Leader key and which-key messages
	case leader.TimeoutMsg:
		// Leader key timeout - show menu
		if m.leaderMgr.IsActive() {
			// Update context before showing menu
			m.updateMenuContext()

			// Always show the root menu initially after leader timeout.
			// The root menu is context-aware and will return appropriate items.
			menu := m.menuRegistry.GetRootMenu()
			m.whichKeyMenu.SetMenu(menu)
			m.whichKeyMenu.Show()
			m.showingMenu = true
		}
		return m, nil

	case whichkey.CloseMenuMsg:
		m.showingMenu = false
		m.leaderMgr.Reset()
		return m, nil

	case whichkey.ExecuteAndCloseMsg:
		m.showingMenu = false
		m.leaderMgr.Reset()
		// Re-process the action message through Update
		return m.Update(msg.ActionMsg)

	case whichkey.ShowSubmenuMsg:
		var cmd tea.Cmd
		m.whichKeyMenu, cmd = m.whichKeyMenu.Update(msg)
		return m, cmd

	// Which-key menu action handlers
	case whichkey.ShowServersViewMsg:
		m.serverSelection = servers.New(m.config.Storage())
		m.serverSelection.SetSize(m.width, m.height)
		m.view = viewServers
		m.error = nil
		return m, nil

	case whichkey.ListExportsMsg:
		m.view = viewExportData
		storage := filepath.Join(m.config.Storage(), m.server.Name)
		exportStore := exportStore.New(storage, m.config.Editor())
		m.exportData = exportData.New(exportStore, m.server, m.width, m.height)

	case whichkey.ViewSchemaMsg:
		if m.editor.IsNormalMode() || !m.editor.IsFocused() {
			m.focused = focusedContent
			m.editor.Blur()
			m.content.ShowDBSchema()
			c, cmd := m.content.Update(nil)
			m.content = c.(content.Model)
			return m, cmd
		}
		return m, nil

	case whichkey.ViewLLMSchemaMsg:
		if m.editor.IsNormalMode() || !m.editor.IsFocused() {
			m.focused = focusedContent
			m.editor.Blur()
			m.content.ShowLLMSharedSchema()
			c, cmd := m.content.Update(nil)
			m.content = c.(content.Model)
			return m, cmd
		}
		return m, nil

	case whichkey.ViewLLMLogsMsg:
		if m.focused == focusedContent || m.view == viewMain {
			m.content.ShowLLMLogs()
		}
		return m, nil

	case whichkey.ListHistoryMsg:
		if entries, err := history.Get(m.config.Storage()); err != nil {
			m.content.SetError(err)
		} else {
			m.view = viewHistory
			m.focused = focusedHistory
			m.editor.Blur()
			m.historyLogs = entries

			m.history = historyView.New(entries, m.width, m.height)
		}

	case whichkey.ToggleFullscreenMsg:
		if m.editor.IsNormalMode() || m.focused == focusedContent {
			m.fullScreen = !m.fullScreen
			m.updateSize()
			contentModel, cmd := m.content.Update(content.ResizeMsg{})
			m.content = contentModel.(content.Model)
			return m, cmd
		}
		return m, nil

	case whichkey.ToggleHelpMsg:
		m.handleHelpToggle()

	case whichkey.ExportJSONMsg:
		m.isPromptActive = true
		m.prompt.SetAction((prompt.ExportAllAsJSONAction))

	case whichkey.ExportCSVMsg:
		m.isPromptActive = true
		m.prompt.SetAction((prompt.ExportAllAsCSVAction))

	case whichkey.CloseExportMsg:
		m.view = viewMain
		m.focused = focusedEditor
		m.editor.Focus()
		return m, nil

	case whichkey.CloseHistoryMsg:
		m.view = viewMain
		m.focused = focusedEditor
		m.editor.Focus()
		return m, nil

	// Database schema actions
	case whichkey.ListTablesMsg:
		return m, m.executePsqlCommand("\\dt")

	case whichkey.ViewIndexesMsg:
		return m, m.executePsqlCommand("\\di")

	case whichkey.ViewConstraintsMsg:
		return m, m.executeQuery("SELECT * FROM information_schema.table_constraints;")

	// History actions
	case whichkey.ClearHistoryMsg:
		// TODO: Create a state machine for handling cleaning history only for current session
		// without updating the history store until the application exits.
		m.historyLogs = []history.Entry{}
		return m, m.successNotification("History cleared for this session")

	case whichkey.EnableDBSchemaMsg:
		return m, utils.Dispatch(command.LLMUseDatabaseSchemaMsg{
			Enabled: true,
		})

	case whichkey.DisableDBSchemaMsg:
		return m, utils.Dispatch(command.LLMUseDatabaseSchemaMsg{
			Enabled: false,
		})

	// Config actions
	case whichkey.ChangeLLMModelMsg:
		m.isPromptActive = true
		m.prompt.SetAction((prompt.LLMModelAction))
		model, _ := m.config.GetLLMModel()
		m.prompt.SetInitialValue(model)

	case whichkey.SetEditorMsg:
		m.isPromptActive = true
		m.prompt.SetAction((prompt.EditorAction))
		m.prompt.SetInitialValue(m.config.Editor())

	case whichkey.ChangeLeaderMsg:
		m.isPromptActive = true
		m.prompt.SetAction(prompt.ChangeLeaderKeyAction)

	// Application control
	case whichkey.QuitMsg:
		m.closeDbConnection()
		return m, tea.Quit

	case prompt.CancelMsg:
		m.isPromptActive = false
	}

	if m.isPromptActive {
		promptModel, cmd := m.prompt.Update(msg)
		m.prompt = promptModel.(prompt.Model)
		return m, cmd
	}

	var cmds []tea.Cmd

	if m.view == viewMain && m.focused == focusedEditor {
		m.editor.SetHighlightedWords(m.setHighlightedKeywords())
		lang := "postgres"
		if strings.HasPrefix(m.editor.GetCurrentContent(), "/") {
			lang = "markdown"
		}

		m.editor.SetLanguage(lang, styles.EditorLanguageTheme())

		editorModel, cmd := m.editor.Update(msg)
		m.editor = editorModel.(editor.Model)
		cmds = append(cmds, cmd)
	}

	if m.view == viewServers {
		s, cmd := m.serverSelection.Update(msg)
		m.serverSelection = s.(servers.Model)
		cmds = append(cmds, cmd)
	}

	if m.focused == focusedContent {
		contentModel, cmd := m.content.Update(msg)
		m.content = contentModel.(content.Model)
		cmds = append(cmds, cmd)
	}

	if m.view == viewExportData {
		exportDataModel, cmd := m.exportData.Update(msg)
		m.exportData = exportDataModel.(exportData.Model)
		cmds = append(cmds, cmd)
	}

	if m.focused == focusedCommand {
		cmdModel, cmd := m.command.Update(msg)
		m.command = cmdModel.(command.Model)
		cmds = append(cmds, cmd)
	}

	if m.view == viewHelp {
		helpModel, cmd := m.help.Update(msg)
		m.help = helpModel.(help.Model)
		cmds = append(cmds, cmd)
	}

	if m.view == viewHistory {
		historyModel, cmd := m.history.Update(msg)
		m.history = historyModel.(historyView.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) handleHelpToggle() {
	switch m.view {
	case viewMain:
		m.view = viewHelp
		m.editor.Blur()
	case viewHelp:
		m.view = viewMain
		m.focused = focusedEditor
		m.editor.Focus()
	case viewExportData:
		m.exportData.HandleHelpToggle()
	}
}

func (m model) View() string {
	width, height := m.getAvailableSizes()

	if m.error != nil {
		return m.renderDBError(width, height)
	}

	if m.isPromptActive {
		return m.prompt.View()
	}

	if m.showingMenu {
		return m.whichKeyMenu.View()
	}

	switch m.view {
	case viewServers:
		return m.renderServers()

	case viewMain:
		return m.renderMain(width, height)

	case viewExportData:
		return m.exportData.View()

	case viewHelp:
		return m.help.View()

	case viewHistory:
		return m.history.View()

	default:
		return ""
	}
}

func (m model) generateSchema() tea.Cmd {
	return func() tea.Msg {
		schema, err := m.db.GenerateSchema()
		if err != nil {
			return schemaFailureMsg{err: err}
		}

		return schemaFetchedMsg(schema)
	}
}

func (m model) executeQuery(query string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := m.db.Query(ctx, query)
		if err != nil {
			return queryFailureMsg{err: err}
		}

		var queryResult content.ParsedQueryResult

		rows, columns, err := db.ExtractResults(result.Rows())
		if err != nil {
			return queryFailureMsg{err: err}
		}

		queryResult.Type = result.Type()
		queryResult.Query = result.Query()
		result.Rows().Close()
		queryResult.AffectedRows = result.Rows().CommandTag().RowsAffected()
		queryResult.Columns = columns
		queryResult.Rows = rows
		queryResult.ExecutionTime = result.ExecutionTime()

		return executeQueryMsg(queryResult)
	}
}

func (m model) ask(prompt string, cmd llm.Command) tea.Cmd {
	return func() tea.Msg {
		if m.llmError != nil {
			return llmFailureMsg{err: fmt.Errorf("LLM is not configured: %w", m.llmError)}
		}

		response, err := m.llm.Ask(prompt, cmd)
		if err != nil {
			return llmFailureMsg{err: err}
		}

		return llmResponseMsg(*response)
	}
}

func (m model) setHighlightedKeywords() map[string]lipgloss.Style {
	if strings.HasPrefix(m.editor.GetCurrentContent(), "/") {
		return m.llmKeywords
	}

	if strings.HasPrefix(m.editor.GetCurrentContent(), "\\") {
		return m.psqlCommands
	}

	return nil
}

func (m model) sendQueryCmd() tea.Cmd {
	prompt := m.editor.GetCurrentContent()

	if prompt == "" {
		return nil
	}

	prompt = strings.TrimSpace(prompt)

	if llm.IsAskCommand(prompt) {
		m.focused = focusedContent
		return m.ask(prompt, llm.Ask)
	}

	if llm.IsExplainCommand(prompt) {
		m.focused = focusedContent
		return m.ask(prompt, llm.Explain)
	}

	if llm.IsOptimiseCommand(prompt) {
		m.focused = focusedContent
		return m.ask(prompt, llm.Optimise)
	}

	if llm.IsFixCommand(prompt) {
		m.focused = focusedContent
		error := m.content.GetError()
		if error != nil {
			prompt += "\nError: " + error.Error()
		}
		return m.ask(prompt, llm.Fix)
	}

	if strings.HasPrefix(prompt, "/add") {
		schema, err := m.addTablesSchemaToLLM()
		if err != nil {
			return utils.Dispatch(notificationErrorMsg{err: err})
		}

		return func() tea.Msg {
			var message string
			if len(m.llmSharedTablesSchema) == 1 {
				message = "Table added to LLM schema"
			} else {
				message = "Tables added to LLM schema"
			}

			return llmSharedSchemaMsg{schema: schema, message: message, tables: m.llmSharedTablesSchema}
		}
	}

	if strings.HasPrefix(prompt, "/remove") {
		schema, err := m.removeTablesSchemaToLLM()
		if err != nil {
			return utils.Dispatch(notificationErrorMsg{err: err})
		}

		return func() tea.Msg {
			var message string
			switch len(m.llmSharedTablesSchema) {
			case 0:
				message = "All tables removed from LLM instructions"
			case 1:
				message = "Table removed from LLM schema"
			default:
				message = "Tables removed from LLM schema"
			}

			return llmSharedSchemaMsg{schema: schema, message: message, tables: m.llmSharedTablesSchema}
		}
	}

	if strings.HasPrefix(prompt, "\\") {
		return m.executePsqlCommand(prompt)
	}

	return m.executeQuery(prompt)
}

func (m model) handleDataExport(msg command.ExportMsg) (tea.Model, tea.Cmd) {
	if filepath.Ext(msg.Filename) != ".json" && filepath.Ext(msg.Filename) != ".csv" {
		return m, m.errorNotification(
			fmt.Errorf("invalid file extension: %s. Supported extensions are .json and .csv", msg.Filename),
		)
	}

	if filepath.Ext(msg.Filename) == ".csv" {
		return m.handleExportAsCsv(msg)
	}

	return m.handleExportAsJson(msg)
}

func (m model) handleExportAsJson(msg command.ExportMsg) (tea.Model, tea.Cmd) {
	queryResults := m.content.GetQueryResults()

	data, err := export.PrepareJSON(queryResults, msg.Rows, msg.All)
	if err != nil {
		m.focused = focusedEditor
		m.editor.Focus()
		return m, m.errorNotification(err)
	}

	storage := filepath.Join(m.config.Storage(), m.server.Name)
	fileName, err := export.AsJson(storage, data, msg.Filename)
	if err != nil {
		return m, m.errorNotification(err)
	}

	m.focused = focusedEditor
	m.editor.Focus()
	m.command.Reset()

	return m, m.successNotification(
		fmt.Sprintf("Data exported successfully as JSON to %s", fileName),
	)
}

func (m model) handleExportAsCsv(msg command.ExportMsg) (tea.Model, tea.Cmd) {
	queryResults := m.content.GetQueryResults()

	data, err := export.PrepareCSV(queryResults, msg.Rows, msg.All)
	if err != nil {
		m.focused = focusedEditor
		m.editor.Focus()
		return m, m.errorNotification(err)
	}

	storage := filepath.Join(m.config.Storage(), m.server.Name)
	fileName, err := export.AsCsv(storage, data, msg.Filename)
	if err != nil {
		return m, m.errorNotification(err)
	}

	m.focused = focusedEditor
	m.editor.Focus()
	m.command.Reset()

	return m, m.successNotification(
		fmt.Sprintf("Data exported successfully as CSV to %s", fileName),
	)
}

func (m *model) successNotification(msg string) tea.Cmd {
	m.notification = styles.Success.Render(msg)

	return utils.ClearAfter(time.Second * 2)
}

func (m *model) errorNotification(err error) tea.Cmd {
	m.notification = styles.Error.Render(err.Error())

	return utils.ClearAfter(time.Second * 2)
}

func (m model) closeDbConnection() {
	if m.db != nil {
		m.db.Close()
	}
}
