package inbox

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/opik/miau/internal/tui/thread"
)

// openThreadView opens the thread view for the current email
func (m Model) openThreadView() (Model, tea.Cmd) {
	if len(m.emails) == 0 {
		return m, nil
	}

	var email = m.emails[m.selectedEmail]

	// Create thread view
	var threadView = thread.New(m.app, email.ID, m.returnFromThread)

	// Store current state
	m.previousState = m.state
	m.state = stateViewingThread
	m.threadView = threadView

	// Initialize thread view
	return m, threadView.Init()
}

// returnFromThread is called when user exits thread view
func (m Model) returnFromThread() tea.Cmd {
	return func() tea.Msg {
		return returnToInboxMsg{}
	}
}

// returnToInboxMsg signals to return from thread view
type returnToInboxMsg struct{}

// updateThreadView delegates updates to the thread view
func (m Model) updateThreadView(msg tea.Msg) (Model, tea.Cmd) {
	// Check for return message
	if _, ok := msg.(returnToInboxMsg); ok {
		m.state = m.previousState
		m.threadView = nil
		return m, nil
	}

	// Delegate to thread view
	if m.threadView != nil {
		var threadModel, ok = m.threadView.(thread.Model)
		if ok {
			var updated, cmd = threadModel.Update(msg)
			m.threadView = updated
			return m, cmd
		}
	}

	return m, nil
}

// viewThreadView renders the thread view
func (m Model) viewThreadView() string {
	if m.threadView != nil {
		var threadModel, ok = m.threadView.(thread.Model)
		if ok {
			return threadModel.View()
		}
	}
	return "Thread view not available"
}
