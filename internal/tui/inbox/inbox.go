package inbox

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
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

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ECDC4"))

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

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B"))
)

type state int

const (
	stateConnecting state = iota
	stateLoadingFolders
	stateLoadingEmails
	stateReady
	stateError
	stateNeedsAppPassword
)

type Model struct {
	width           int
	height          int
	state           state
	err             error
	account         *config.Account
	cfg             *config.Config
	client          *imap.Client
	mailboxes       []imap.Mailbox
	emails          []imap.Email
	selectedEmail   int
	selectedBox     int
	currentBox      string
	showFolders     bool
	passwordInput   textinput.Model
	retrying        bool
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

type configSavedMsg struct{}

func New(account *config.Account) Model {
	var input = textinput.New()
	input.Placeholder = "xxxx xxxx xxxx xxxx"
	input.CharLimit = 20
	input.Width = 25
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '•'

	return Model{
		account:       account,
		state:         stateConnecting,
		currentBox:    "INBOX",
		showFolders:   false,
		passwordInput: input,
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

func (m Model) saveConfig() tea.Cmd {
	return func() tea.Msg {
		// Recarrega config completa
		var cfg, err = config.Load()
		if err != nil {
			return errMsg{err: err}
		}

		// Atualiza a senha da conta
		for i := range cfg.Accounts {
			if cfg.Accounts[i].Email == m.account.Email {
				cfg.Accounts[i].Password = m.account.Password
				break
			}
		}

		// Salva
		if err := config.Save(cfg); err != nil {
			return errMsg{err: err}
		}

		return configSavedMsg{}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Estado de pedir App Password
		if m.state == stateNeedsAppPassword {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "enter":
				var password = strings.TrimSpace(m.passwordInput.Value())
				if password == "" {
					return m, nil
				}
				// Atualiza senha e tenta reconectar
				m.account.Password = strings.ReplaceAll(password, " ", "")
				m.state = stateConnecting
				m.retrying = true
				m.err = nil
				return m, tea.Batch(m.saveConfig(), m.connect())
			case "esc", "q":
				return m, tea.Quit
			}

			var cmd tea.Cmd
			m.passwordInput, cmd = m.passwordInput.Update(msg)
			return m, cmd
		}

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
		m.retrying = false
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

	case configSavedMsg:
		// Config salva, continue esperando a conexão
		return m, nil

	case errMsg:
		m.err = msg.err
		// Verifica se é erro de App Password
		if isAppPasswordError(msg.err) {
			m.state = stateNeedsAppPassword
			m.passwordInput.Focus()
			return m, textinput.Blink
		}
		m.state = stateError
	}

	return m, nil
}

func isAppPasswordError(err error) bool {
	if err == nil {
		return false
	}
	var errStr = err.Error()
	return strings.Contains(errStr, "Application-specific password required") ||
		strings.Contains(errStr, "Invalid credentials") ||
		strings.Contains(errStr, "AUTHENTICATIONFAILED") ||
		strings.Contains(errStr, "Username and Password not accepted")
}

func (m Model) View() string {
	switch m.state {
	case stateConnecting:
		var msg = "Conectando ao servidor IMAP..."
		if m.retrying {
			msg = "Reconectando com nova senha..."
		}
		return m.viewLoading(msg)
	case stateLoadingFolders:
		return m.viewLoading("Carregando pastas...")
	case stateLoadingEmails:
		return m.viewLoading(fmt.Sprintf("Carregando emails de %s...", m.currentBox))
	case stateNeedsAppPassword:
		return m.viewAppPasswordPrompt()
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

func (m Model) viewAppPasswordPrompt() string {
	var header = titleStyle.Render("miau 🐱 - App Password Necessária")

	var explanation = infoStyle.Render(`
O Google requer uma "App Password" para apps de email.

Como criar:
1. Acesse: myaccount.google.com/apppasswords
2. Selecione "Mail" e "Outro (miau)"
3. Clique em "Gerar"
4. Copie a senha de 16 caracteres abaixo
`)

	var prompt = "\nApp Password:\n"
	var input = inputStyle.Render(m.passwordInput.View())

	var hint = "\n\n" + subtitleStyle.Render("Enter: conectar • Esc: sair")

	var content = fmt.Sprintf("%s%s%s%s%s", header, explanation, prompt, input, hint)
	var box = boxStyle.Padding(1, 2).Render(content)

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
