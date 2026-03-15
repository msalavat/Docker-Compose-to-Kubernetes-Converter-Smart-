package wizard

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			MarginBottom(1)

	serviceHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("14")).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				MarginTop(1)

	questionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("208"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))
)
