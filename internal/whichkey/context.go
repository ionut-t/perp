package whichkey

// MenuContext represents the current application state for context-aware menus
type MenuContext struct {
	// Connection state
	IsConnected bool
	ServerName  string

	// View state
	InServersView   bool
	InExportView    bool
	InMainView      bool
	InHistoryView   bool
	InSnippetsView  bool
	FocusedOnTable  bool
	FocusedOnEditor bool
	IsFullScreen    bool
	IsHelpVisible   bool

	// Data state
	HasQueryResults bool
	HasHistory      bool
	ResultCount     int
	HistoryCount    int

	// Feature availability
	LLMEnabled      bool
	LLMSchemaShared bool
}

// NewMenuContext creates a new menu context with default values
func NewMenuContext() *MenuContext {
	return &MenuContext{
		IsConnected:     false,
		InMainView:      true,
		InServersView:   false,
		InExportView:    false,
		InHistoryView:   false,
		InSnippetsView:  false,
		FocusedOnEditor: true,
		FocusedOnTable:  false,
		IsFullScreen:    false,
		HasQueryResults: false,
		HasHistory:      false,
		ResultCount:     0,
		HistoryCount:    0,
		LLMEnabled:      false,
	}
}
