package ui

import (
	"charm.land/lipgloss/v2"
	"github.com/SayYoungMan/tfui/pkg/terraform"
)

var (
	colorBlue       = lipgloss.Color("111")
	colorGreen      = lipgloss.Color("114")
	colorCoral      = lipgloss.Color("167")
	colorAmber      = lipgloss.Color("178")
	colorCreamWhite = lipgloss.Color("230")
	colorCharcoal   = lipgloss.Color("234")
	colorLightGrey  = lipgloss.Color("240")
	colorDimGrey    = lipgloss.Color("245")
	colorSoftGrey   = lipgloss.Color("248")
)

var (
	cursorStyle         = lipgloss.NewStyle().Background(colorCreamWhite).Foreground(colorCharcoal)
	selectedStyle       = lipgloss.NewStyle().Background(colorLightGrey)
	borderStyle         = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorDimGrey).Padding(0, 1)
	focusedBorderStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorCreamWhite).Padding(0, 1)
	buttonStyle         = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorDimGrey).Padding(0, 2)
	focusedButtonStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorCreamWhite).Padding(0, 2)
	dimStyle            = lipgloss.NewStyle().Foreground(colorDimGrey)
	shutdownBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorCoral).
				Padding(6, 6)
	errorStyle          = lipgloss.NewStyle().Foreground(colorCoral)
	warningStyle        = lipgloss.NewStyle().Foreground(colorAmber)
	infoBarStyle        = lipgloss.NewStyle().Foreground(colorCreamWhite)
	helpKeyStyle        = lipgloss.NewStyle().Foreground(colorCreamWhite)
	helpDescStyle       = lipgloss.NewStyle().Foreground(colorSoftGrey)
	resourceBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorDimGrey).Padding(0, 1)

	moduleStyle            = lipgloss.NewStyle().Foreground(colorSoftGrey)
	treePrefixDefaultStyle = lipgloss.NewStyle().Foreground(colorDimGrey)
	treePrefixCurrentStyle = lipgloss.NewStyle().Foreground(colorCreamWhite)
)

var actionStyles = map[terraform.Action]lipgloss.Style{
	terraform.ActionCreate:    lipgloss.NewStyle().Foreground(colorGreen),
	terraform.ActionDelete:    lipgloss.NewStyle().Foreground(colorCoral),
	terraform.ActionUpdate:    lipgloss.NewStyle().Foreground(colorAmber),
	terraform.ActionReplace:   lipgloss.NewStyle().Foreground(colorAmber),
	terraform.ActionMove:      lipgloss.NewStyle().Foreground(colorBlue),
	terraform.ActionImport:    lipgloss.NewStyle().Foreground(colorBlue),
	terraform.ActionUncertain: dimStyle,
}
