package styles

import (
	"image/color"

	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/x/exp/charmtone"
)

func FangColorScheme(c lipgloss.LightDarkFunc) fang.ColorScheme {
	return fang.ColorScheme{
		Base:           Primary.GetForeground(),
		Title:          Accent.GetForeground(),
		Codeblock:      Surface0.GetBackground(),
		Program:        Primary.GetForeground(),
		Command:        Primary.GetForeground(),
		DimmedArgument: Overlay1.GetForeground(),
		Comment:        Subtext1.GetForeground(),
		Flag:           Success.GetForeground(),
		Argument:       Text.GetForeground(),
		Description:    Text.GetForeground(), // flag and command descriptions
		FlagDefault:    Text.GetForeground(), // flag default values in descriptions
		QuotedString:   Accent.GetForeground(),
		ErrorHeader: [2]color.Color{
			charmtone.Butter,
			charmtone.Cherry,
		},
	}
}
