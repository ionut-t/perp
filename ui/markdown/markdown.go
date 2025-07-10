package markdown

import (
	_ "embed"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

//go:embed glamour-themes/catppuccin-mocha.json
var catppuccinMochaTheme []byte

//go:embed glamour-themes/catppuccin-latte.json
var catppuccinLatteTheme []byte

func getThemeBytes() []byte {
	if lipgloss.HasDarkBackground() {
		return catppuccinMochaTheme
	}

	return catppuccinLatteTheme
}

type Model struct {
	renderer *glamour.TermRenderer
	error    error
}

func New() Model {
	renderer, err := createGlamourRenderer()

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

func createGlamourRenderer() (*glamour.TermRenderer, error) {
	themeBytes := getThemeBytes()

	return glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(themeBytes),
	)
}
