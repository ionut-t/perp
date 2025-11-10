package keymap

import "github.com/charmbracelet/bubbles/key"

var Quit = key.NewBinding(
	key.WithKeys("q"),
	key.WithHelp("q", "quit"),
)

var ForceQuit = key.NewBinding(
	key.WithKeys("ctrl+c"),
	key.WithHelp("ctrl+c", "force quit"),
)

var Editor = key.NewBinding(
	key.WithKeys("ctrl+e"),
	key.WithHelp("ctrl+e", "open in external editor"),
)

var Insert = key.NewBinding(
	key.WithKeys("i"),
	key.WithHelp("i", "enter insert mode"),
)

var Cancel = key.NewBinding(
	key.WithKeys("esc"),
	key.WithHelp("esc", "cancel current operation"),
)

var Submit = key.NewBinding(
	key.WithKeys("enter"),
	key.WithHelp("enter", "submit"),
)
