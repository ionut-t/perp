package styles

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

func ThemeCatppuccin() *huh.Theme {
	t := huh.ThemeBase()

	var (
		base     = Base.GetForeground()
		text     = Text.GetForeground()
		subtext1 = Subtext1.GetForeground()
		subtext0 = Subtext0.GetForeground()
		overlay1 = Overlay1.GetForeground()
		overlay0 = Overlay0.GetForeground()
		green    = Success.GetForeground()
		red      = Error.GetForeground()
		accent   = Accent.GetForeground()
		primary  = Primary.GetForeground()
		cursor   = Primary.GetForeground()
		surface0 = Surface0.GetBackground()
	)

	t.Focused.Base = t.Focused.Base.BorderForeground(subtext1)
	t.Focused.Title = t.Focused.Title.Foreground(primary)
	t.Focused.NoteTitle = t.Focused.NoteTitle.Foreground(primary)
	t.Focused.Directory = t.Focused.Directory.Foreground(primary)
	t.Focused.Description = t.Focused.Description.Foreground(subtext0)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(red)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(red)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(accent)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(accent)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(accent)
	t.Focused.Option = t.Focused.Option.Foreground(text)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(accent)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(green)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(green)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(text)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(text)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(base).Background(primary)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(text).Background(surface0)

	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(cursor)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(overlay0)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(accent)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Blurred.Base.BorderStyle(lipgloss.HiddenBorder())

	t.Help.Ellipsis = t.Help.Ellipsis.Foreground(subtext0)
	t.Help.ShortKey = t.Help.ShortKey.Foreground(subtext0)
	t.Help.ShortDesc = t.Help.ShortDesc.Foreground(overlay1)
	t.Help.ShortSeparator = t.Help.ShortSeparator.Foreground(subtext0)
	t.Help.FullKey = t.Help.FullKey.Foreground(subtext0)
	t.Help.FullDesc = t.Help.FullDesc.Foreground(overlay1)
	t.Help.FullSeparator = t.Help.FullSeparator.Foreground(subtext0)

	return t
}
