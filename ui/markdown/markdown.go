package markdown

import (
	_ "embed"

	"github.com/charmbracelet/glamour"
)

//go:embed glamour-themes/catppuccin-mocha.json
var catppuccinMochaTheme []byte

//go:embed glamour-themes/catppuccin-latte.json
var catppuccinLatteTheme []byte

func getThemeBytes(isDark bool) []byte {
	if isDark {
		return catppuccinMochaTheme
	}

	return catppuccinLatteTheme
}

type Model struct {
	renderer *glamour.TermRenderer
	error    error
}

func New(isDark bool) Model {
	renderer, err := createGlamourRenderer(isDark)

	return Model{
		renderer: renderer,
		error:    err,
	}
}

// Render renders markdown
func (m Model) Render(markdown string) (string, error) {
	if m.error != nil {
		return "", m.error
	}

	return m.renderer.Render(markdown)
}

func createGlamourRenderer(isDark bool) (*glamour.TermRenderer, error) {
	themeBytes := getThemeBytes(isDark)

	return glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(themeBytes),
	)
}
