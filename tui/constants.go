package tui

import "time"

// view represents different view states in the application
type view int

const (
	viewServers view = iota
	viewMain
	viewExportData
	viewHelp
	viewHistory
	viewSnippets
)

// focused represents which component currently has focus
type focused int

const (
	focusedNone focused = iota
	focusedEditor
	focusedContent
	focusedCommand
	focusedHistory
	focusedSnippets
)

// Layout constants
const (
	editorMinHeight        = 10
	editorHalfScreenOffset = 4 // Offset for editor in split view (accounts for borders/padding)
)

// Timeout and duration constants
const (
	DatabaseQueryTimeout = 5 * time.Second
	LeaderKeyTimeout     = 500 * time.Millisecond
	NotificationDuration = 2 * time.Second
)

// Directory constants
const exportDataDirectory = "data"
