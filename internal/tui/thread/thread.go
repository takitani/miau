package thread

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opik/miau/internal/ports"
)

// State represents the thread view state
type State int

const (
	stateLoading State = iota
	stateReady
	stateError
)

// Model represents the thread view
type Model struct {
	app ports.App

	// Thread data
	thread          *ports.Thread
	emailID         int64
	expandedIndices map[int]bool // which messages are expanded

	// UI state
	state           State
	selectedIndex   int  // current message cursor
	showMinimap     bool // toggle minimap visibility
	width           int
	height          int
	viewport        viewport.Model
	errorMsg        string

	// Navigation history (for going back to inbox)
	returnToInbox func() tea.Cmd
}

// New creates a new thread view model
func New(app ports.App, emailID int64, returnToInbox func() tea.Cmd) Model {
	var vp = viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240"))

	return Model{
		app:             app,
		emailID:         emailID,
		state:           stateLoading,
		showMinimap:     true,
		expandedIndices: make(map[int]bool),
		viewport:        vp,
		returnToInbox:   returnToInbox,
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return m.loadThread()
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Reserve space for header (3 lines) and help (2 lines)
		var contentHeight = msg.Height - 5
		var contentWidth = msg.Width

		// Reserve space for minimap if visible
		if m.showMinimap {
			contentWidth -= minimapWidth + 2 // +2 for border
		}

		m.viewport.Width = contentWidth
		m.viewport.Height = contentHeight
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case threadLoadedMsg:
		m.thread = msg.thread
		m.state = stateReady

		// Expand first (newest) message by default
		if len(m.thread.Messages) > 0 {
			m.expandedIndices[0] = true
		}

		return m, m.renderContent()

	case threadErrorMsg:
		m.state = stateError
		m.errorMsg = msg.error
		return m, nil
	}

	// Handle viewport updates
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View implements tea.Model
func (m Model) View() string {
	switch m.state {
	case stateLoading:
		return m.viewLoading()
	case stateError:
		return m.viewError()
	case stateReady:
		return m.viewReady()
	}
	return ""
}

func (m Model) viewLoading() string {
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render("⏳ Carregando thread...")
}

func (m Model) viewError() string {
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Foreground(lipgloss.Color("#FF6B6B")).
		Render(fmt.Sprintf("❌ Erro: %s\n\nPressione Esc para voltar", m.errorMsg))
}

func (m Model) viewReady() string {
	if m.thread == nil {
		return ""
	}

	var sections []string

	// Header
	sections = append(sections, m.renderHeader())

	// Main content area
	var content string
	if m.showMinimap {
		content = lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.viewport.View(),
			m.renderMinimap(),
		)
	} else {
		content = m.viewport.View()
	}
	sections = append(sections, content)

	// Help footer
	sections = append(sections, m.renderHelp())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderHeader() string {
	if m.thread == nil {
		return ""
	}

	var unreadBadge string
	if !m.thread.IsRead {
		var unreadCount = 0
		for _, msg := range m.thread.Messages {
			if !msg.IsRead {
				unreadCount++
			}
		}
		if unreadCount > 0 {
			unreadBadge = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF6B6B")).
				Bold(true).
				Render(fmt.Sprintf(" [●%d]", unreadCount))
		}
	}

	var title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#61AFEF")).
		Render(m.thread.Subject)

	var meta = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(fmt.Sprintf("(%d mensagens)%s", m.thread.MessageCount, unreadBadge))

	var participants string
	if len(m.thread.Participants) > 0 {
		participants = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Render(fmt.Sprintf("👥 %s", strings.Join(m.thread.Participants, ", ")))
	}

	return lipgloss.NewStyle().
		Width(m.width).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			lipgloss.JoinHorizontal(lipgloss.Left, title, " ", meta),
			participants,
		))
}

func (m Model) renderHelp() string {
	var keys []string

	if m.showMinimap {
		keys = []string{
			"↑↓:navegar",
			"Enter:expandir",
			"m:esconder minimap",
			"r:marcar lida",
			"Esc:voltar",
		}
	} else {
		keys = []string{
			"↑↓:navegar",
			"Enter:expandir",
			"m:mostrar minimap",
			"r:marcar lida",
			"Esc:voltar",
		}
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(m.width).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(strings.Join(keys, " • "))
}

// Messages

type threadLoadedMsg struct {
	thread *ports.Thread
}

type threadErrorMsg struct {
	error string
}

// Commands

func (m Model) loadThread() tea.Cmd {
	return func() tea.Msg {
		var ctx = context.Background()
		var thread, err = m.app.Thread().GetThread(ctx, m.emailID)
		if err != nil {
			return threadErrorMsg{error: err.Error()}
		}
		return threadLoadedMsg{thread: thread}
	}
}

func (m Model) renderContent() tea.Cmd {
	return func() tea.Msg {
		if m.thread == nil {
			return nil
		}

		var content strings.Builder

		// Render each message (newest first)
		for i, msg := range m.thread.Messages {
			var isExpanded = m.expandedIndices[i]
			var isSelected = i == m.selectedIndex

			if isExpanded {
				content.WriteString(m.renderExpandedMessage(msg, isSelected))
			} else {
				content.WriteString(m.renderCollapsedMessage(msg, isSelected))
			}

			// Separator between messages
			if i < len(m.thread.Messages)-1 {
				content.WriteString("\n")
			}
		}

		m.viewport.SetContent(content.String())
		return nil
	}
}

func (m Model) renderCollapsedMessage(msg ports.EmailContent, isSelected bool) string {
	var style = collapsedMessageStyle
	if isSelected {
		style = collapsedMessageStyleSelected
	}

	var icon = "▸"
	var timeAgo = formatTimeAgo(msg.Date)
	var snippet = msg.Snippet
	if snippet == "" && msg.BodyText != "" {
		snippet = truncate(msg.BodyText, 60)
	}

	var header = fmt.Sprintf("%s %s → %s  %s",
		icon,
		msg.FromName,
		extractRecipients(msg.ToAddresses),
		timeAgo,
	)

	var preview = lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true).
		Render(fmt.Sprintf("   \"%s\"", snippet))

	return style.Render(header + "\n" + preview)
}

func (m Model) renderExpandedMessage(msg ports.EmailContent, isSelected bool) string {
	var style = expandedMessageStyle
	if isSelected {
		style = expandedMessageStyleSelected
	}

	var sections []string

	// Header
	var icon = "▾"
	var timeAgo = formatTimeAgo(msg.Date)
	var header = fmt.Sprintf("%s %s → %s  %s",
		icon,
		lipgloss.NewStyle().Bold(true).Render(msg.FromName),
		extractRecipients(msg.ToAddresses),
		timeAgo,
	)
	sections = append(sections, header)

	// Separator
	sections = append(sections, strings.Repeat("─", 60))

	// Body (prefer text, fallback to HTML converted)
	var body = msg.BodyText
	if body == "" && msg.BodyHTML != "" {
		body = "[HTML content - view in web browser]"
	}

	// Wrap and indent body
	var bodyLines = strings.Split(body, "\n")
	for _, line := range bodyLines {
		if len(line) > 70 {
			// Simple word wrap
			sections = append(sections, wrapText(line, 70)...)
		} else {
			sections = append(sections, line)
		}
	}

	// Attachments indicator
	if msg.HasAttachments {
		sections = append(sections, "")
		sections = append(sections, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5C07B")).
			Render("📎 Anexos disponíveis"))
	}

	return style.Render(strings.Join(sections, "\n"))
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		// Return to inbox
		if m.returnToInbox != nil {
			return m, m.returnToInbox()
		}
		return m, tea.Quit

	case "j", "down":
		// Navigate down
		if m.selectedIndex < len(m.thread.Messages)-1 {
			m.selectedIndex++
			return m, m.renderContent()
		}

	case "k", "up":
		// Navigate up
		if m.selectedIndex > 0 {
			m.selectedIndex--
			return m, m.renderContent()
		}

	case "enter", " ":
		// Toggle expand/collapse
		m.expandedIndices[m.selectedIndex] = !m.expandedIndices[m.selectedIndex]
		return m, m.renderContent()

	case "m":
		// Toggle minimap
		m.showMinimap = !m.showMinimap

		// Recalculate viewport width
		var contentWidth = m.width
		if m.showMinimap {
			contentWidth -= minimapWidth + 2
		}
		m.viewport.Width = contentWidth

		return m, m.renderContent()

	case "r":
		// Mark thread as read
		return m, m.markThreadAsRead()

	case "u":
		// Mark thread as unread
		return m, m.markThreadAsUnread()

	case "t":
		// Collapse all messages
		m.expandedIndices = make(map[int]bool)
		// Keep first (newest) expanded
		m.expandedIndices[0] = true
		return m, m.renderContent()
	}

	return m, nil
}

func (m Model) markThreadAsRead() tea.Cmd {
	return func() tea.Msg {
		if m.thread == nil {
			return nil
		}

		var ctx = context.Background()
		if err := m.app.Thread().MarkThreadAsRead(ctx, m.thread.ThreadID); err != nil {
			return threadErrorMsg{error: err.Error()}
		}

		// Reload thread to reflect changes
		return m.loadThread()()
	}
}

func (m Model) markThreadAsUnread() tea.Cmd {
	return func() tea.Msg {
		if m.thread == nil {
			return nil
		}

		var ctx = context.Background()
		if err := m.app.Thread().MarkThreadAsUnread(ctx, m.thread.ThreadID); err != nil {
			return threadErrorMsg{error: err.Error()}
		}

		// Reload thread
		return m.loadThread()()
	}
}

// Helper functions

func formatTimeAgo(t time.Time) string {
	var duration = time.Since(t)

	if duration < time.Minute {
		return "agora"
	} else if duration < time.Hour {
		var mins = int(duration.Minutes())
		return fmt.Sprintf("%dm", mins)
	} else if duration < 24*time.Hour {
		var hours = int(duration.Hours())
		return fmt.Sprintf("%dh", hours)
	} else if duration < 7*24*time.Hour {
		var days = int(duration.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	} else {
		return t.Format("02/01")
	}
}

func extractRecipients(addresses string) string {
	if addresses == "" {
		return "todos"
	}

	// Simple extraction - just show first recipient
	var parts = strings.Split(addresses, ",")
	if len(parts) > 1 {
		return fmt.Sprintf("%s +%d", strings.TrimSpace(parts[0]), len(parts)-1)
	}
	return strings.TrimSpace(parts[0])
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func wrapText(text string, width int) []string {
	var words = strings.Fields(text)
	var lines []string
	var currentLine string

	for _, word := range words {
		if len(currentLine)+len(word)+1 > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		} else {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}
