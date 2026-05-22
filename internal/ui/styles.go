package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
)

type theme struct {
	blue, green, coral, amber color.Color
	cursorBg, cursorFg        color.Color
	selectedBg, selectedFg    color.Color
	border, focus, dim, soft  color.Color
	chroma                    string
}

var darkTheme = theme{
	blue:       lipgloss.Color("111"),
	green:      lipgloss.Color("114"),
	coral:      lipgloss.Color("167"),
	amber:      lipgloss.Color("178"),
	cursorBg:   lipgloss.Color("230"),
	cursorFg:   lipgloss.Color("234"),
	selectedBg: lipgloss.Color("240"),
	border:     lipgloss.Color("245"),
	focus:      lipgloss.Color("230"),
	dim:        lipgloss.Color("245"),
	soft:       lipgloss.Color("248"),
	chroma:     "catppuccin-mocha",
}

var lightTheme = theme{
	blue:       lipgloss.Color("25"),
	green:      lipgloss.Color("28"),
	coral:      lipgloss.Color("160"),
	amber:      lipgloss.Color("136"),
	cursorBg:   lipgloss.Color("236"),
	cursorFg:   lipgloss.Color("255"),
	selectedBg: lipgloss.Color("254"),
	selectedFg: lipgloss.Color("235"),
	border:     lipgloss.Color("244"),
	focus:      lipgloss.Color("236"),
	dim:        lipgloss.Color("244"),
	soft:       lipgloss.Color("242"),
	chroma:     "github",
}

type styles struct {
	cursor, selected                             lipgloss.Style
	border, focusedBorder                        lipgloss.Style
	button, focusedButton                        lipgloss.Style
	dim, shutdownBorder                          lipgloss.Style
	error, warning, success                      lipgloss.Style
	infoBar, helpKey, helpInfo                   lipgloss.Style
	module, treePrefixDefault, treePrefixCurrent lipgloss.Style
	actions                                      map[terraform.Action]lipgloss.Style
	chromaTheme                                  string
}

func newStyles(isDark bool) styles {
	if isDark {
		return darkTheme.styles()
	}
	return lightTheme.styles()
}

func (t *theme) styles() styles {
	selected := lipgloss.NewStyle().Background(t.selectedBg)
	if t.selectedFg != nil {
		selected = selected.Foreground(t.selectedFg)
	}

	return styles{
		cursor:        lipgloss.NewStyle().Background(t.cursorBg).Foreground(t.cursorFg),
		selected:      selected,
		border:        lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderBottomForeground(t.border).Padding(0, 1),
		focusedBorder: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderBottomForeground(t.focus).Padding(0, 1),
		button:        lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderBottomForeground(t.border).Padding(0, 2),
		focusedButton: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderBottomForeground(t.focus).Padding(0, 2),
		dim:           lipgloss.NewStyle().Foreground(t.dim),

		shutdownBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.coral).
			Padding(6, 6),

		error:   lipgloss.NewStyle().Foreground(t.coral),
		warning: lipgloss.NewStyle().Foreground(t.amber),
		success: lipgloss.NewStyle().Foreground(t.green),

		infoBar:           lipgloss.NewStyle().Foreground(t.focus),
		helpKey:           lipgloss.NewStyle().Foreground(t.focus),
		helpInfo:          lipgloss.NewStyle().Foreground(t.soft),
		module:            lipgloss.NewStyle().Foreground(t.soft),
		treePrefixDefault: lipgloss.NewStyle().Foreground(t.dim),
		treePrefixCurrent: lipgloss.NewStyle().Foreground(t.focus),

		chromaTheme: t.chroma,
		actions:     t.actionStyles(),
	}
}

func (t *theme) actionStyles() map[terraform.Action]lipgloss.Style {
	return map[terraform.Action]lipgloss.Style{
		terraform.ActionCreate:    lipgloss.NewStyle().Foreground(t.green),
		terraform.ActionDelete:    lipgloss.NewStyle().Foreground(t.coral),
		terraform.ActionUpdate:    lipgloss.NewStyle().Foreground(t.amber),
		terraform.ActionReplace:   lipgloss.NewStyle().Foreground(t.amber),
		terraform.ActionMove:      lipgloss.NewStyle().Foreground(t.blue),
		terraform.ActionImport:    lipgloss.NewStyle().Foreground(t.blue),
		terraform.ActionUncertain: lipgloss.NewStyle().Foreground(t.dim),
	}
}
