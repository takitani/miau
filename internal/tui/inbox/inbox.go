package inbox

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/imap"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B6B"))

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#FF6B6B")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	unreadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	readStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#73D216"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF6B6B")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1)

	folderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ECDC4"))

	folderSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#4ECDC4")).
				Foreground(lipgloss.Color("#000000"))
)

type state int

const (
	stateConnecting state = iota
	stateLoadingFolders
	stateLoadingEmails
	stateReady
	stateError
)

type Model struct {
	width         int
	height        int
	state         state
	err           error
	account       *config.Account
	client        *imap.Client
	mailboxes     []imap.Mailbox
	emails        []imap.Email
	selectedEmail int
	selectedBox   int
	currentBox    string
	showFolders   bool
}

// Messages
type connectedMsg struct {
	client *imap.Client
}

type foldersLoadedMsg struct {
	mailboxes []imap.Mailbox
}

type emailsLoadedMsg struct {
	emails []imap.Email
}

type errMsg struct {
	err error
}

func New(account *config.Account) Model {
	return Model{
		account:     account,
		state:       stateConnecting,
		currentBox:  "INBOX",
		showFolders: false,
	}
}

func (m Model) Init() tea.Cmd {
	return m.connect()
}

func (m Model) connect() tea.Cmd {
	return func() tea.Msg {
		var client, err = imap.Connect(m.account)
		if err != nil {
			return errMsg{err: err}
		}
		return connectedMsg{client: client}
	}
}

func (m Model) loadFolders() tea.Cmd {
	return func() tea.Msg {
		var mailboxes, err = m.client.ListMailboxes()
		if err != nil {
			return errMsg{err: err}
		}
		return foldersLoadedMsg{mailboxes: mailboxes}
	}
}

func (m Model) loadEmails() tea.Cmd {
	return func() tea.Msg {
		var _, err = m.client.SelectMailbox(m.currentBox)
		if err != nil {
			return errMsg{err: err}
		}

		var emails, err2 = m.client.FetchEmails(50)
		if err2 != nil {
			return errMsg{err: err2}
		}
		return emailsLoadedMsg{emails: emails}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.client != nil {
				m.client.Close()
			}
			return m, tea.Quit

		case "up", "k":
			if m.showFolders {
				if m.selectedBox > 0 {
					m.selectedBox--
				}
			} else {
				if m.selectedEmail > 0 {
					m.selectedEmail--
				}
			}

		case "down", "j":
			if m.showFolders {
				if m.selectedBox < len(m.mailboxes)-1 {
					m.selectedBox++
				}
			} else {
				if m.selectedEmail < len(m.emails)-1 {
					m.selectedEmail++
				}
			}

		case "tab":
			m.showFolders = !m.showFolders

		case "enter":
			if m.showFolders && len(m.mailboxes) > 0 {
				m.currentBox = m.mailboxes[m.selectedBox].Name
				m.showFolders = false
				m.state = stateLoadingEmails
				m.selectedEmail = 0
				return m, m.loadEmails()
			}

		case "r":
			// Refresh
			m.state = stateLoadingEmails
			return m, m.loadEmails()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case connectedMsg:
		m.client = msg.client
		m.state = stateLoadingFolders
		return m, m.loadFolders()

	case foldersLoadedMsg:
		m.mailboxes = msg.mailboxes
		m.state = stateLoadingEmails
		// Encontra índice do INBOX
		for i, mb := range m.mailboxes {
			if strings.EqualFold(mb.Name, "INBOX") {
				m.selectedBox = i
				break
			}
		}
		return m, m.loadEmails()

	case emailsLoadedMsg:
		m.emails = msg.emails
		m.state = stateReady

	case errMsg:
		m.err = msg.err
		m.state = stateError
	}

	return m, nil
}

func (m Model) View() string {
	switch m.state {
	case stateConnecting:
		return m.viewLoading("Conectando ao servidor IMAP...")
	case stateLoadingFolders:
		return m.viewLoading("Carregando pastas...")
	case stateLoadingEmails:
		return m.viewLoading(fmt.Sprintf("Carregando emails de %s...", m.currentBox))
	case stateError:
		return m.viewError()
	case stateReady:
		return m.viewInbox()
	}
	return ""
}

func (m Model) viewLoading(msg string) string {
	var content = fmt.Sprintf("%s\n\n%s",
		titleStyle.Render("miau 🐱"),
		subtitleStyle.Render(msg),
	)

	var box = boxStyle.Render(content)

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}

func (m Model) viewError() string {
	var content = fmt.Sprintf("%s\n\n%s\n\n%s",
		titleStyle.Render("miau 🐱"),
		errorStyle.Render("Erro: "+m.err.Error()),
		subtitleStyle.Render("Pressione 'q' para sair"),
	)

	var box = boxStyle.Render(content)

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}

func (m Model) viewInbox() string {
	// Header
	var header = headerStyle.Render(fmt.Sprintf(" miau 🐱  %s  [%s] ",
		m.account.Email,
		m.currentBox,
	))

	// Folders panel (se ativo)
	var foldersPanel string
	if m.showFolders {
		foldersPanel = m.renderFolders()
	}

	// Email list
	var emailList = m.renderEmailList()

	// Footer
	var footer = subtitleStyle.Render(" ↑↓:navegar  Tab:pastas  r:refresh  q:sair ")

	// Layout
	var content string
	if m.showFolders {
		content = lipgloss.JoinHorizontal(lipgloss.Top, foldersPanel, emailList)
	} else {
		content = emailList
	}

	var view = lipgloss.JoinVertical(lipgloss.Left,
		header,
		content,
		footer,
	)

	return view
}

func (m Model) renderFolders() string {
	var lines []string
	lines = append(lines, folderStyle.Render("  Pastas  "))
	lines = append(lines, "")

	for i, mb := range m.mailboxes {
		var line string
		var name = truncate(mb.Name, 20)

		if mb.Unseen > 0 {
			line = fmt.Sprintf(" %s (%d)", name, mb.Unseen)
		} else {
			line = fmt.Sprintf(" %s", name)
		}

		if i == m.selectedBox {
			lines = append(lines, folderSelectedStyle.Render(line))
		} else {
			lines = append(lines, folderStyle.Render(line))
		}
	}

	var content = strings.Join(lines, "\n")
	return boxStyle.Width(25).Render(content)
}

func (m Model) renderEmailList() string {
	if len(m.emails) == 0 {
		return boxStyle.Render(subtitleStyle.Render("Nenhum email encontrado"))
	}

	var lines []string
	var listHeight = m.height - 4 // header + footer
	if listHeight < 5 {
		listHeight = 10
	}

	// Calcula janela de visualização
	var start = 0
	var end = len(m.emails)
	if len(m.emails) > listHeight {
		start = m.selectedEmail - listHeight/2
		if start < 0 {
			start = 0
		}
		end = start + listHeight
		if end > len(m.emails) {
			end = len(m.emails)
			start = end - listHeight
		}
	}

	var emailWidth = m.width - 4
	if m.showFolders {
		emailWidth -= 27
	}
	if emailWidth < 40 {
		emailWidth = 60
	}

	for i := start; i < end; i++ {
		var email = m.emails[i]
		var line = m.formatEmailLine(email, emailWidth)

		if i == m.selectedEmail {
			lines = append(lines, selectedStyle.Render(line))
		} else if email.Seen {
			lines = append(lines, readStyle.Render(line))
		} else {
			lines = append(lines, unreadStyle.Render(line))
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) formatEmailLine(email imap.Email, width int) string {
	var indicator = "●"
	if email.Seen {
		indicator = " "
	}

	var from = truncate(email.From, 20)
	var subject = truncate(email.Subject, width-35)
	var date = email.Date.Format("02/01 15:04")

	return fmt.Sprintf(" %s %-20s │ %-*s │ %s ",
		indicator,
		from,
		width-35,
		subject,
		date,
	)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
