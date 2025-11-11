package whichkey

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"
)

// Registry manages all available menus
type Registry struct {
	context      *MenuContext
	rootMenu     *Menu
	serverMenu   *Menu
	exportMenu   *Menu
	llmMenu      *Menu
	databaseMenu *Menu
	historyMenu  *Menu
	configMenu   *Menu
}

// NewRegistry creates a new menu registry with all menus
func NewRegistry() *Registry {
	r := &Registry{
		context: NewMenuContext(),
	}
	r.buildMenus()
	return r
}

// UpdateContext updates the menu context
func (r *Registry) UpdateContext(ctx *MenuContext) {
	// Update the existing context in place so dynamic menu closures see the changes
	*r.context = *ctx
}

func (r *Registry) buildServersMenu() *Menu {
	return NewMenu("Server Operations", []MenuItem{
		{
			Key:         "l",
			Label:       "List servers",
			Description: "View all configured servers",
			Action:      CommandAction{Cmd: ShowServersViewCmd},
		},
	})
}

func (r *Registry) buildExportsMenu() *Menu {
	return NewDynamicMenu("Export Operations", func() []MenuItem {
		// If in export workspace, show special navigation menu
		if r.context.InExportView {
			return []MenuItem{
				{
					Key:         "e",
					Label:       "Editor",
					Description: "Open in external editor",
					Action:      CommandAction{Cmd: ExternalEditorCmd},
				},
				{
					Key:         "c",
					Label:       "Close",
					Description: "Close export view",
					Action:      CommandAction{Cmd: CloseExportCmd},
				},
				{
					Key:         "?",
					Label:       "Help",
					Description: "Show help",
					Action:      CommandAction{Cmd: ToggleHelpCmd},
				},
				{
					Key:         "q",
					Label:       "Quit",
					Description: "Exit application",
					Action:      CommandAction{Cmd: QuitCmd},
				},
			}
		}

		// Normal export menu
		items := []MenuItem{
			{
				Key:         "l",
				Label:       "List exports",
				Description: "View exported files",
				Action:      CommandAction{Cmd: ListExportsCmd},
			},
		}

		if r.context.HasQueryResults {
			items = append(items,
				MenuItem{
					Key:         "j",
					Label:       "Export as JSON",
					Description: "Export all rows in JSON format",
					Action:      CommandAction{Cmd: ExportJSONCmd},
				},
				MenuItem{
					Key:         "c",
					Label:       "Export as CSV",
					Description: "Export all rows in CSV format",
					Action:      CommandAction{Cmd: ExportCSVCmd},
				},
			)
		}

		return items
	})
}

func (r *Registry) buildLLMMenu() *Menu {
	return NewDynamicMenu("LLM Operations", func() []MenuItem {
		items := []MenuItem{
			{
				Key:         "s",
				Label:       "Shared schema",
				Description: "View LLM shared schema",
				Action:      CommandAction{Cmd: ViewLLMSchemaCmd},
			},
			{
				Key:         "m",
				Label:       "Change model",
				Description: "Switch LLM model",
				Action:      CommandAction{Cmd: ChangeLLMModelCmd},
			},
		}

		if r.context.LLMSchemaShared {
			items = append(items, MenuItem{
				Key:         "d",
				Label:       "Disable DB schema",
				Description: "Exclude DB schema from prompts",
				Action:      CommandAction{Cmd: DisableDBSchemaCmd},
			})
		} else {
			items = append(items, MenuItem{
				Key:         "e",
				Label:       "Enable DB schema",
				Description: "Include DB schema in prompts",
				Action:      CommandAction{Cmd: EnableDBSchemaCmd},
			})
		}

		return items
	})
}

func (r *Registry) buildDatabaseMenu() *Menu {
	return NewMenu("Database Operations", []MenuItem{
		{
			Key:         "s",
			Label:       "View schema",
			Description: "Display database schema",
			Action: CommandAction{
				Cmd: ViewSchemaCmd,
				Validator: func(ctx *MenuContext) bool {
					return ctx.IsConnected
				},
			},
		},
		{
			Key:         "t",
			Label:       "List tables",
			Description: "Show all tables",
			Action: CommandAction{
				Cmd: ListTablesCmd,
				Validator: func(ctx *MenuContext) bool {
					return ctx.IsConnected
				},
			},
		},
		{
			Key:         "i",
			Label:       "View indexes",
			Description: "Show table indexes",
			Action: CommandAction{
				Cmd: ViewIndexesCmd,
				Validator: func(ctx *MenuContext) bool {
					return ctx.IsConnected
				},
			},
		},
		{
			Key:         "c",
			Label:       "View constraints",
			Description: "Show table constraints",
			Action: CommandAction{
				Cmd: ViewConstraintsCmd,
				Validator: func(ctx *MenuContext) bool {
					return ctx.IsConnected
				},
			},
		},
	})
}

func (r *Registry) buildHistoryMenu() *Menu {
	return NewDynamicMenu("History Operations", func() []MenuItem {
		if r.context.InHistoryView {
			return []MenuItem{
				{
					Key:         "c",
					Label:       "Close",
					Description: "Close history view",
					Action:      CommandAction{Cmd: CloseHistoryCmd},
				},
				{
					Key:         "q",
					Label:       "Quit",
					Description: "Exit application",
					Action:      CommandAction{Cmd: QuitCmd},
				},
			}
		}

		return []MenuItem{
			{
				Key:         "l",
				Label:       "List history",
				Description: "View query history",
				Action:      CommandAction{Cmd: ListHistoryCmd},
			},
			// {
			// 	Key:         "c",
			// 	Label:       "Clear history",
			// 	Description: "Delete all history",
			// 	Action: CommandAction{
			// 		Cmd: ClearHistoryCmd,
			// 		Validator: func(ctx *MenuContext) bool {
			// 			return ctx.HasHistory
			// 		},
			// 	},
			// },
		}
	})
}

func (r *Registry) buildConfigMenu() *Menu {
	return NewMenu("Configuration", []MenuItem{
		{
			Key:         "e",
			Label:       "External editor",
			Description: "Set external editor",
			Action:      CommandAction{Cmd: SetEditorCmd},
		},
		{
			Key:         "l",
			Label:       "Leader key",
			Description: "Change leader key",
			Action:      CommandAction{Cmd: ChangeLeaderCmd},
		},
	})
}

func (r *Registry) buildRootMenu() *Menu {
	return NewDynamicMenu("Perp Commands", func() []MenuItem {
		fullScreenLabel := "Enter full-screen"
		helpLabel := "Show help"

		if r.context.IsFullScreen {
			fullScreenLabel = "Exit full-screen"
		}

		if r.context.IsHelpVisible {
			helpLabel = "Hide help"
		}

		// In servers view - only show quit
		if r.context.InServersView {
			return []MenuItem{
				{
					Key:         "q",
					Label:       "Quit",
					Description: "Exit application",
					Action:      CommandAction{Cmd: QuitCmd},
				},
			}
		}

		// In export view - return export menu items (which are context-aware)
		if r.context.InExportView {
			return r.exportMenu.GetItems()
		}

		// In history view - return history menu items (which are context-aware)
		if r.context.InHistoryView {
			return r.historyMenu.GetItems()
		}

		items := []MenuItem{
			{
				Key:         "d",
				Label:       "Database",
				Description: "Database schema operations",
				Action:      SubmenuAction{Menu: r.databaseMenu},
			},
			{
				Key:         "e",
				Label:       "Export",
				Description: "Export query results",
				Action:      SubmenuAction{Menu: r.exportMenu},
			},
			{
				Key:         "s",
				Label:       "Servers",
				Description: "Manage database connections",
				Action:      SubmenuAction{Menu: r.serverMenu},
			},
			{
				Key:         "h",
				Label:       "History",
				Description: "Query history management",
				Action:      SubmenuAction{Menu: r.historyMenu},
			},
			{
				Key:         "l",
				Label:       "LLM",
				Description: "AI-powered SQL assistance",
				Action:      SubmenuAction{Menu: r.llmMenu},
			},
			{
				Key:         "c",
				Label:       "Config",
				Description: "Application settings",
				Action:      SubmenuAction{Menu: r.configMenu},
			},
			{
				Key:         "f",
				Label:       fullScreenLabel,
				Description: "Toggle full-screen mode",
				Action:      CommandAction{Cmd: ToggleFullscreenCmd},
			},
			{
				Key:         "?",
				Label:       helpLabel,
				Description: "Toggle help",
				Action:      CommandAction{Cmd: ToggleHelpCmd},
			},
			{
				Key:         "q",
				Label:       "Quit",
				Description: "Exit application",
				Action:      CommandAction{Cmd: QuitCmd},
			},
		}

		if !r.context.LLMEnabled {
			items = slices.DeleteFunc(items, func(item MenuItem) bool {
				return item.Label == "llm"
			})
		}

		return items
	})
}

// buildMenus constructs the menu hierarchy
func (r *Registry) buildMenus() {
	r.serverMenu = r.buildServersMenu()
	r.exportMenu = r.buildExportsMenu()
	r.llmMenu = r.buildLLMMenu()
	r.databaseMenu = r.buildDatabaseMenu()
	r.historyMenu = r.buildHistoryMenu()
	r.configMenu = r.buildConfigMenu()
	r.rootMenu = r.buildRootMenu()

	// Set parent references for navigation
	r.serverMenu.SetParent(r.rootMenu)
	r.exportMenu.SetParent(r.rootMenu)
	r.llmMenu.SetParent(r.rootMenu)
	r.databaseMenu.SetParent(r.rootMenu)
	r.historyMenu.SetParent(r.rootMenu)
	r.configMenu.SetParent(r.rootMenu)
}

// GetRootMenu returns the root menu
func (r *Registry) GetRootMenu() *Menu {
	return r.rootMenu
}

// GetMenu returns a specific menu by type
func (r *Registry) GetMenu(menuType string) *Menu {
	switch menuType {
	case "server":
		return r.serverMenu
	case "export":
		return r.exportMenu
	case "llm":
		return r.llmMenu
	case "database":
		return r.databaseMenu
	case "history":
		return r.historyMenu
	case "config":
		return r.configMenu
	default:
		return r.rootMenu
	}
}

// Action messages - these will be handled by the main TUI model

// Server actions
type ShowServersViewMsg struct{}

func ShowServersViewCmd() tea.Msg { return ShowServersViewMsg{} }

// Export actions
type (
	ListExportsMsg    struct{}
	ExportJSONMsg     struct{}
	ExportCSVMsg      struct{}
	BackToMainMsg     struct{}
	CloseExportMsg    struct{}
	ExternalEditorMsg struct{}
)

func ListExportsCmd() tea.Msg    { return ListExportsMsg{} }
func ExportJSONCmd() tea.Msg     { return ExportJSONMsg{} }
func ExportCSVCmd() tea.Msg      { return ExportCSVMsg{} }
func CloseExportCmd() tea.Msg    { return CloseExportMsg{} }
func ExternalEditorCmd() tea.Msg { return ExternalEditorMsg{} }

// LLM actions
type (
	ViewLLMSchemaMsg   struct{}
	ChangeLLMModelMsg  struct{}
	EnableDBSchemaMsg  struct{}
	DisableDBSchemaMsg struct{}
)

func ViewLLMSchemaCmd() tea.Msg   { return ViewLLMSchemaMsg{} }
func ChangeLLMModelCmd() tea.Msg  { return ChangeLLMModelMsg{} }
func EnableDBSchemaCmd() tea.Msg  { return EnableDBSchemaMsg{} }
func DisableDBSchemaCmd() tea.Msg { return DisableDBSchemaMsg{} }

// Database actions
type (
	ViewSchemaMsg      struct{}
	ListTablesMsg      struct{}
	ViewIndexesMsg     struct{}
	ViewConstraintsMsg struct{}
)

func ViewSchemaCmd() tea.Msg      { return ViewSchemaMsg{} }
func ListTablesCmd() tea.Msg      { return ListTablesMsg{} }
func ViewIndexesCmd() tea.Msg     { return ViewIndexesMsg{} }
func ViewConstraintsCmd() tea.Msg { return ViewConstraintsMsg{} }

// History actions
type (
	ListHistoryMsg  struct{}
	ClearHistoryMsg struct{}
	CloseHistoryMsg struct{}
)

func ListHistoryCmd() tea.Msg  { return ListHistoryMsg{} }
func ClearHistoryCmd() tea.Msg { return ClearHistoryMsg{} }
func CloseHistoryCmd() tea.Msg { return CloseHistoryMsg{} }

// Config actions
type (
	SetEditorMsg    struct{}
	ChangeLeaderMsg struct{}
)

func SetEditorCmd() tea.Msg    { return SetEditorMsg{} }
func ChangeLeaderCmd() tea.Msg { return ChangeLeaderMsg{} }

// Window actions
type (
	ToggleFullscreenMsg struct{}
	ToggleHelpMsg       struct{}
	QuitMsg             struct{}
)

func ToggleFullscreenCmd() tea.Msg { return ToggleFullscreenMsg{} }
func ToggleHelpCmd() tea.Msg       { return ToggleHelpMsg{} }
func QuitCmd() tea.Msg             { return QuitMsg{} }
