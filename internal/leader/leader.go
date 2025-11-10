package leader

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TimeoutMsg is sent when leader key timeout expires
type TimeoutMsg struct {
	Time time.Time
}

// State represents the current leader key state
type State int

const (
	StateNone    State = iota
	StateWaiting       // Leader pressed, waiting for next key
)

// Manager handles leader key state transitions
type Manager struct {
	currentState State
	timeout      time.Duration
	leaderKey    string
	timeoutStart time.Time
}

// NewManager creates a new leader key manager
func NewManager(timeout time.Duration, leaderKey string) *Manager {
	if leaderKey == "" {
		leaderKey = " "
	}

	return &Manager{
		currentState: StateNone,
		timeout:      timeout,
		leaderKey:    leaderKey,
	}
}

// HandleKey processes a key press in leader mode
// Returns the new state and an optional command
func (m *Manager) HandleKey(key string) (State, tea.Cmd) {
	// Check if this is the leader key
	if m.currentState == StateNone && key == m.leaderKey {
		m.currentState = StateWaiting
		m.timeoutStart = time.Now()
		return StateWaiting, m.startTimeout()
	}

	// If we're waiting for the next key, the manager's role is done.
	// The TUI layer will interpret the subsequent key via the whichkey menu.
	// Manager only transitions from StateNone to StateWaiting.
	if m.currentState == StateWaiting {
		return StateWaiting, nil
	}

	return m.currentState, nil
}

// Reset clears the leader key state
func (m *Manager) Reset() {
	m.currentState = StateNone
	m.timeoutStart = time.Time{}
}

// IsActive returns true if leader key mode is active
func (m *Manager) IsActive() bool {
	return m.currentState != StateNone
}

// GetCurrentState returns the current state
func (m *Manager) GetCurrentState() State {
	return m.currentState
}

// SetLeaderKey changes the leader key
func (m *Manager) SetLeaderKey(key string) {
	if key == "" {
		key = " "
	}

	m.leaderKey = key
}

// GetLeaderKey returns the current leader key
func (m *Manager) GetLeaderKey() string {
	return m.leaderKey
}

// IsLeaderKey checks if the given key is the leader key
func (m *Manager) IsLeaderKey(key string) bool {
	return key == m.leaderKey
}

// startTimeout returns a command that fires after timeout
func (m *Manager) startTimeout() tea.Cmd {
	return tea.Tick(m.timeout, func(t time.Time) tea.Msg {
		return TimeoutMsg{Time: t}
	})
}
