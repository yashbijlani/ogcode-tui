package tui

import "github.com/charmbracelet/lipgloss"

var (
	colPrimary = lipgloss.Color("63")
	colAccent  = lipgloss.Color("212")
	colMuted   = lipgloss.Color("241")
	colFaint   = lipgloss.Color("238")
	colUser    = lipgloss.Color("45")
	colTool    = lipgloss.Color("214")
	colErr     = lipgloss.Color("203")
	colOK      = lipgloss.Color("78")

	styleTitle     = lipgloss.NewStyle().Bold(true).Foreground(colPrimary)
	styleUser      = lipgloss.NewStyle().Bold(true).Foreground(colUser)
	styleAssistant = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	styleLabel     = lipgloss.NewStyle().Bold(true).Foreground(colPrimary)
	styleMuted     = lipgloss.NewStyle().Foreground(colMuted)
	styleFaint     = lipgloss.NewStyle().Foreground(colFaint)
	styleTool      = lipgloss.NewStyle().Foreground(colTool)
	styleErr       = lipgloss.NewStyle().Foreground(colErr)
	styleOK        = lipgloss.NewStyle().Foreground(colOK)
	styleAccent    = lipgloss.NewStyle().Foreground(colAccent)
	styleBold      = lipgloss.NewStyle().Bold(true)
	styleSelected  = lipgloss.NewStyle().Bold(true).Foreground(colAccent)

	styleHelpKey  = lipgloss.NewStyle().Bold(true).Foreground(colPrimary)
	styleHelpDesc = lipgloss.NewStyle().Foreground(colMuted)

	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colPrimary).
			Padding(0, 1)

	styleModalBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colAccent).
			Padding(0, 1)
)
