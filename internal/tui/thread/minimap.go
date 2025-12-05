package thread

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	minimapWidth = 3 // Ultra-thin vertical bar
)

// renderMinimap renders the vertical minimap showing thread structure
func (m Model) renderMinimap() string {
	if m.thread == nil || len(m.thread.Messages) == 0 {
		return ""
	}

	var lines []string

	// Title
	lines = append(lines, lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Bold(true).
		Render("│"))

	// One dot per message
	for i := range m.thread.Messages {
		var symbol string
		var style lipgloss.Style

		if i == m.selectedIndex {
			// Current message - highlighted
			symbol = "●"
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#61AFEF")).
				Bold(true)
		} else if m.expandedIndices[i] {
			// Expanded but not selected
			symbol = "○"
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#98C379"))
		} else {
			// Collapsed
			symbol = "·"
			style = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))
		}

		lines = append(lines, style.Render(symbol))
	}

	// Add vertical scroll indicator if needed
	var totalMessages = len(m.thread.Messages)
	if totalMessages > m.viewport.Height/4 {
		// Add scroll position indicator
		var scrollPercent = float64(m.selectedIndex) / float64(totalMessages-1)
		var scrollPos = int(scrollPercent * float64(len(lines)-2))

		if scrollPos >= 0 && scrollPos < len(lines) {
			lines[scrollPos+1] = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E5C07B")).
				Bold(true).
				Render("█")
		}
	}

	// Wrap in border
	var content = strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Width(minimapWidth).
		Height(m.viewport.Height).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderForeground(lipgloss.Color("240")).
		PaddingLeft(1).
		Render(content)
}

// getParticipantColor returns a consistent color for a participant
// Uses email hash to generate deterministic colors
func getParticipantColor(email string) lipgloss.Color {
	var colors = []string{
		"#E06C75", // Red
		"#98C379", // Green
		"#E5C07B", // Yellow
		"#61AFEF", // Blue
		"#C678DD", // Purple
		"#56B6C2", // Cyan
	}

	// Simple hash: sum of bytes modulo color count
	var hash = 0
	for _, c := range email {
		hash += int(c)
	}

	return lipgloss.Color(colors[hash%len(colors)])
}

// renderMinimapDetailed renders a more detailed minimap with participant info
// This is an alternative visualization - not used by default but available
func (m Model) renderMinimapDetailed() string {
	if m.thread == nil || len(m.thread.Messages) == 0 {
		return ""
	}

	var lines []string

	// Header with participant legend
	lines = append(lines, lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("┌%s┐", strings.Repeat("─", minimapWidth))))

	// Participant colors (first 3)
	var uniqueParticipants = m.thread.Participants
	if len(uniqueParticipants) > 3 {
		uniqueParticipants = uniqueParticipants[:3]
	}

	for _, participant := range uniqueParticipants {
		var color = getParticipantColor(participant)
		var initial = "?"
		if len(participant) > 0 {
			initial = string(participant[0])
		}

		lines = append(lines, lipgloss.NewStyle().
			Foreground(color).
			Bold(true).
			Render(fmt.Sprintf("│%s│", initial)))
	}

	if len(m.thread.Participants) > 3 {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(fmt.Sprintf("│+%d│", len(m.thread.Participants)-3)))
	}

	// Separator
	lines = append(lines, lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("├%s┤", strings.Repeat("─", minimapWidth))))

	// Messages
	for i, msg := range m.thread.Messages {
		var symbol string
		var color = getParticipantColor(msg.FromEmail)

		if i == m.selectedIndex {
			symbol = "●"
		} else if m.expandedIndices[i] {
			symbol = "○"
		} else {
			symbol = "·"
		}

		var style = lipgloss.NewStyle().Foreground(color)
		if i == m.selectedIndex {
			style = style.Bold(true)
		}

		lines = append(lines, style.Render(fmt.Sprintf("│%s│", symbol)))
	}

	// Footer
	lines = append(lines, lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("└%s┘", strings.Repeat("─", minimapWidth))))

	return lipgloss.NewStyle().
		Height(m.viewport.Height).
		AlignVertical(lipgloss.Top).
		Render(strings.Join(lines, "\n"))
}
