package clipboard

import (
	"os"
	"sync"

	"github.com/atotto/clipboard"
	osc52 "github.com/aymanbagabas/go-osc52/v2"
)

var (
	mu     sync.Mutex
	buffer string
)

// Write copies text to the clipboard.
//
// It always updates an in-process buffer so that paste within the same session
// works regardless of environment.
//
// It then tries the native system clipboard tool (pbcopy on macOS,
// xclip/wl-clipboard on Linux). If that succeeds, we're done.
//
// If the native tool is unavailable (e.g. a headless VPS), it falls back to
// an OSC 52 escape sequence written to stderr, which asks the terminal emulator
// on the user's local machine to write to its clipboard. This works over SSH
// with most modern emulators. The library handles tmux passthrough automatically.
func Write(text string) error {
	mu.Lock()
	buffer = text
	mu.Unlock()

	if err := clipboard.WriteAll(text); err == nil {
		return nil
	}

	_, err := osc52.New(text).WriteTo(os.Stderr)
	return err
}

// Read returns clipboard text. It first tries the native system clipboard
// (picks up text copied from outside perp), then falls back to the
// in-process buffer populated by the last Write call.
func Read() (string, error) {
	if text, err := clipboard.ReadAll(); err == nil {
		return text, nil
	}

	mu.Lock()
	defer mu.Unlock()
	return buffer, nil
}

// Clipboard implements the goeditor core.Clipboard interface so that the
// editor uses the same OSC 52-aware clipboard as the table view.
type Clipboard struct{}

func (c *Clipboard) Write(text string) error { return Write(text) }
func (c *Clipboard) Read() (string, error)   { return Read() }
