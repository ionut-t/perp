package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ionut-t/coffee/styles"
	editor "github.com/ionut-t/goeditor/adapter-bubbletea"
	"github.com/ionut-t/perp/internal/config"
	"github.com/ionut-t/perp/internal/leader"
	"github.com/ionut-t/perp/internal/whichkey"
	"github.com/ionut-t/perp/pkg/db"
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
		leaderMgr:       leader.NewManager(LeaderKeyTimeout, config.GetLeaderKey()),
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

		// Don't handle keys if in special views or command mode
		if m.focused == focusedCommand ||
			m.view == viewServers ||
			m.view == viewExportData ||
			m.view == viewHistory ||
			m.isPromptActive ||
			!m.editor.IsNormalMode() && m.focused == focusedEditor {
			break
		}

		// Try to handle special key bindings
		// If handled, return early; otherwise, break to let editor handle it
		updatedModel, cmd, handled := m.tryHandleKeyPress(msg)

		if handled {
			return updatedModel, cmd
		}
		m = updatedModel.(model)

	case servers.SelectedServerMsg:
		return m.handleServerConnection(msg)

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
		return m.handleQueryResult(msg)

	case queryFailureMsg:
		m.loading = false
		m.content.SetError(msg.err)

	case psqlCommandMsg:
		m.loading = true
		return m, m.runPsqlCommand(msg.command)

	case psqlResultMsg:
		return m.handlePsqlResult(msg)

	case psqlErrorMsg:
		m.loading = false
		m.content.SetError(msg.err)

	case toggleExpandedMsg:
		return m.toggleExpandedDisplay()

	case toggleTimingMsg:
		return m.toggleQueryTiming()

	case showPsqlHelpMsg:
		m.loading = false
		m.content.ShowPsqlHelp()
		m.focused = focusedContent
		return m, m.resetEditor()

	case llmResponseMsg:
		m.handleLLMResponse(msg)

	case llmFailureMsg:
		m.loading = false
		m.content.SetError(msg.err)

	case llmSharedSchemaMsg:
		return m.updateSharedSchema(msg)

	case notificationErrorMsg:
		m.loading = false
		return m, m.errorNotification(msg.err)

	case content.LLMResponseSelectedMsg:
		return m.applyLLMResponse(msg)

	case command.QuitMsg, psqlQuitMsg:
		m.closeDbConnection()
		return m, tea.Quit

	case command.CancelMsg:
		m.focusEditor()

	case command.ExportMsg:
		return m.exportQueryData(msg)

	case command.EditorChangedMsg:
		err := m.config.SetEditor(msg.Editor)
		if err != nil {
			return m, m.errorNotification(err)
		}

		m.focusEditor()
		return m, m.successNotification(
			fmt.Sprintf("Editor changed to %s", msg.Editor),
		)

	case command.LLMUseDatabaseSchemaMsg:
		return m.toggleDBSchemaSharing(msg)

	case command.LLMModelChangedMsg:
		return m.updateLLMModel(msg)

	case command.LeaderKeyChangedMsg:
		return m.updateLeaderKey(msg)

	case command.ErrorMsg:
		return m, m.errorNotification(msg.Err)

	case historyView.SelectedMsg:
		return m.applyHistoryQuery(msg)

	// Leader key and which-key messages
	case leader.TimeoutMsg:
		// Leader key timeout - show menu
		return m.showLeaderMenu()

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
		m.focusEditor()
		return m, nil

	case whichkey.CloseHistoryMsg:
		m.view = viewMain
		m.focusEditor()
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

		textEditor, cmd := m.editor.Update(msg)
		m.editor = textEditor.(editor.Model)
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

func (m model) showLeaderMenu() (tea.Model, tea.Cmd) {
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
}

func (m model) updateLeaderKey(msg command.LeaderKeyChangedMsg) (tea.Model, tea.Cmd) {
	existingLeader := m.config.GetLeaderKey()
	if existingLeader == msg.Key {
		return m, nil
	}

	if err := m.config.SetLeaderKey(msg.Key); err != nil {
		return m, m.errorNotification(err)
	}

	m.leaderMgr.SetLeaderKey(msg.Key)
	return m, m.successNotification("Leader key changed")
}

func (m model) applyHistoryQuery(msg historyView.SelectedMsg) (tea.Model, tea.Cmd) {
	return m, m.applyQueryToEditor(msg.Query)
}
