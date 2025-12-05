package thread

import "github.com/charmbracelet/lipgloss"

var (
	// Collapsed message styles
	collapsedMessageStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginBottom(1)

	collapsedMessageStyleSelected = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#61AFEF")).
		Padding(0, 1).
		MarginBottom(1).
		Background(lipgloss.Color("236"))

	// Expanded message styles
	expandedMessageStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		MarginBottom(1)

	expandedMessageStyleSelected = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#61AFEF")).
		Padding(1).
		MarginBottom(1).
		Background(lipgloss.Color("236"))
)
