package ui

import "github.com/charmbracelet/lipgloss"

var (
	colorGreen  = lipgloss.Color("10")
	colorYellow = lipgloss.Color("11")
	colorRed    = lipgloss.Color("9")
	colorCyan   = lipgloss.Color("14")
	colorGray   = lipgloss.Color("8")
	colorWhite  = lipgloss.Color("15")
	colorBlack  = lipgloss.Color("0")

	styleVBadge = lipgloss.NewStyle().Background(colorGreen).Foreground(colorBlack).Bold(true).Padding(0, 1)
	styleYBadge = lipgloss.NewStyle().Background(colorYellow).Foreground(colorBlack).Bold(true).Padding(0, 1)
	styleXBadge = lipgloss.NewStyle().Background(colorRed).Foreground(colorWhite).Bold(true).Padding(0, 1)

	styleBanner = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	styleHeader = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true).
			MarginBottom(1)

	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorCyan).
			Padding(1, 3).
			Width(58)

	styleLabel = lipgloss.NewStyle().
			Foreground(colorWhite).
			Bold(true)

	styleDim = lipgloss.NewStyle().
			Foreground(colorGray)

	styleSectionHeader = lipgloss.NewStyle().
				Foreground(colorCyan).
				Bold(true)

	styleSelected = lipgloss.NewStyle().
			Background(colorCyan).
			Foreground(colorBlack).
			Bold(true).
			Padding(0, 1)

	styleUnselected = lipgloss.NewStyle().
				Foreground(colorWhite).
				Padding(0, 1)

	styleKeyPressed = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	styleKeyUnpressed = lipgloss.NewStyle().
				Foreground(colorGray)

	styleTableHeader = lipgloss.NewStyle().
				Foreground(colorCyan).
				Bold(true)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	styleError = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	styleHeaderBar = lipgloss.NewStyle().
			Background(colorCyan).
			Foreground(colorBlack).
			Bold(true).
			Padding(0, 2)

	styleFooterBar = lipgloss.NewStyle().
			Foreground(colorGray).
			Padding(0, 1)
)

func statusStyle(s string) string {
	switch s {
	case "V":
		return styleVBadge.Render("V")
	case "Y":
		return styleYBadge.Render("Y")
	case "X":
		return styleXBadge.Render("X")
	default:
		return styleDim.Render(s)
	}
}

const banner = `
  ████████╗ █████╗      ██████╗██╗  ██╗██╗██████╗
  ╚══██╔══╝██╔══██╗    ██╔════╝██║  ██║██║██╔══██╗
     ██║   ███████║    ██║     ███████║██║██████╔╝
     ██║   ██╔══██║    ██║     ██╔══██║██║██╔═══╝
     ██║   ██║  ██║    ╚██████╗██║  ██║██║██║
     ╚═╝   ╚═╝  ╚═╝     ╚═════╝╚═╝  ╚═╝╚═╝╚═╝
  PC Health Inspection Tool`
