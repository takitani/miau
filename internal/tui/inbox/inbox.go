package inbox

import (
	"database/sql"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/imap"
	"github.com/opik/miau/internal/storage"
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

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)
)

type state int

const (
	stateInitDB state = iota
	stateConnecting
	stateLoadingFolders
	stateSyncing
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
	dbAccount       *storage.Account
	dbFolder        *storage.Folder
	client          *imap.Client
	mailboxes       []imap.Mailbox
	emails          []storage.EmailSummary
	selectedEmail   int
	selectedBox     int
	currentBox      string
	showFolders     bool
	passwordInput   textinput.Model
	retrying        bool
	syncStatus      string
	totalEmails     int
	syncedEmails    int
	// AI panel
	showAI          bool
	aiInput         textinput.Model
	aiResponse      string
	aiLastQuestion  string
	aiLoading       bool
	aiScrollOffset  int
}

// Messages
type dbInitMsg struct{}

type connectedMsg struct {
	client *imap.Client
}

type foldersLoadedMsg struct {
	mailboxes []imap.Mailbox
}

type syncProgressMsg struct {
	status string
	synced int
	total  int
}

type syncDoneMsg struct{}

type emailsLoadedMsg struct {
	emails []storage.EmailSummary
}

type errMsg struct {
	err error
}

type configSavedMsg struct{}

type aiResponseMsg struct {
	response string
	err      error
}

func New(account *config.Account) Model {
	var input = textinput.New()
	input.Placeholder = "xxxx xxxx xxxx xxxx"
	input.CharLimit = 20
	input.Width = 25
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '•'

	var aiInput = textinput.New()
	aiInput.Placeholder = "Pergunte algo sobre seus emails..."
	aiInput.CharLimit = 500
	aiInput.Width = 60

	return Model{
		account:       account,
		state:         stateInitDB,
		currentBox:    "INBOX",
		showFolders:   false,
		passwordInput: input,
		aiInput:       aiInput,
	}
}

func (m Model) Init() tea.Cmd {
	return m.initDB()
}

func (m Model) initDB() tea.Cmd {
	return func() tea.Msg {
		var cfg, _ = config.Load()
		var dbPath = cfg.Storage.Database
		if err := storage.Init(dbPath); err != nil {
			return errMsg{err: err}
		}
		return dbInitMsg{}
	}
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

func (m Model) syncEmails() tea.Cmd {
	return func() tea.Msg {
		// Seleciona a mailbox
		var _, err = m.client.SelectMailbox(m.currentBox)
		if err != nil {
			return errMsg{err: fmt.Errorf("erro ao selecionar pasta: %w", err)}
		}

		// Carrega config para obter dias de sync
		var cfg, _ = config.Load()
		var syncDays = 30 // default
		if cfg != nil && cfg.Sync.InitialDays > 0 {
			syncDays = cfg.Sync.InitialDays
		}
		// 0 = todos os emails
		if cfg != nil && cfg.Sync.InitialDays == 0 {
			syncDays = 0
		}

		// Busca emails do servidor (por data)
		var emails, err2 = m.client.FetchEmailsSince(syncDays)
		if err2 != nil {
			return errMsg{err: err2}
		}

		// Salva no banco
		for _, email := range emails {
			var dbEmail = &storage.Email{
				AccountID:   m.dbAccount.ID,
				FolderID:    m.dbFolder.ID,
				UID:         email.UID,
				MessageID:   sql.NullString{String: email.MessageID, Valid: email.MessageID != ""},
				Subject:     email.Subject,
				FromName:    email.From,
				FromEmail:   email.FromEmail,
				ToAddresses: email.To,
				Date:        storage.SQLiteTime{Time: email.Date},
				IsRead:      email.Seen,
				IsStarred:   email.Flagged,
				Size:        email.Size,
			}
			storage.UpsertEmail(dbEmail)
		}

		// Atualiza stats da pasta
		var total, unread, _ = storage.CountEmails(m.dbAccount.ID, m.dbFolder.ID)
		storage.UpdateFolderStats(m.dbFolder.ID, total, unread)

		return syncDoneMsg{}
	}
}

func (m Model) loadEmailsFromDB() tea.Cmd {
	return func() tea.Msg {
		var emails, err = storage.GetEmails(m.dbAccount.ID, m.dbFolder.ID, 100, 0)
		if err != nil {
			return errMsg{err: err}
		}
		return emailsLoadedMsg{emails: emails}
	}
}

func (m Model) runAI(prompt string) tea.Cmd {
	return func() tea.Msg {
		var fullPrompt = fmt.Sprintf(`[Context: Email database at ~/.config/miau/data/miau.db | Account: %s | Folder: %s]
[Schema: emails(id, subject, from_name, from_email, date, is_read, is_starred, is_deleted, body_text)]
[Use sqlite3 to query. Be concise.]

%s`, m.account.Email, m.currentBox, prompt)

		// Debug: salva o comando em arquivo de log
		var logFile, _ = exec.Command("sh", "-c", "echo '['+$(date)+'] Running claude -p ...' >> /tmp/miau-ai.log").Output()
		_ = logFile

		var cmd = exec.Command("claude", "-p", "--permission-mode", "bypassPermissions", fullPrompt)

		// Log do prompt
		exec.Command("sh", "-c", fmt.Sprintf("echo 'Prompt: %s' >> /tmp/miau-ai.log", prompt)).Run()

		var output, err = cmd.CombinedOutput()

		// Log do resultado
		exec.Command("sh", "-c", fmt.Sprintf("echo 'Output (%d bytes): %s' >> /tmp/miau-ai.log", len(output), string(output[:min(100, len(output))]))).Run()

		if err != nil {
			return aiResponseMsg{err: fmt.Errorf("%s: %w", string(output), err)}
		}
		return aiResponseMsg{response: string(output)}
	}
}

func (m Model) saveConfig() tea.Cmd {
	return func() tea.Msg {
		var cfg, err = config.Load()
		if err != nil {
			return errMsg{err: err}
		}

		for i := range cfg.Accounts {
			if cfg.Accounts[i].Email == m.account.Email {
				cfg.Accounts[i].Password = m.account.Password
				break
			}
		}

		if err := config.Save(cfg); err != nil {
			return errMsg{err: err}
		}

		return configSavedMsg{}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == stateNeedsAppPassword {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "enter":
				var password = strings.TrimSpace(m.passwordInput.Value())
				if password == "" {
					return m, nil
				}
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

		// AI panel mode
		if m.showAI {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.showAI = false
				m.aiInput.Blur()
				return m, nil
			case "enter":
				var prompt = strings.TrimSpace(m.aiInput.Value())
				if prompt == "" || m.aiLoading {
					return m, nil
				}
				m.aiLastQuestion = prompt
				m.aiInput.SetValue("")
				m.aiLoading = true
				m.aiResponse = ""
				return m, m.runAI(prompt)
			case "up":
				if m.aiScrollOffset > 0 {
					m.aiScrollOffset--
				}
				return m, nil
			case "down":
				m.aiScrollOffset++
				return m, nil
			}
			var cmd tea.Cmd
			m.aiInput, cmd = m.aiInput.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "q":
			if m.client != nil {
				m.client.Close()
			}
			storage.Close()
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
				m.state = stateSyncing
				m.selectedEmail = 0
				// Cria/obtém a pasta no DB
				var folder, _ = storage.GetOrCreateFolder(m.dbAccount.ID, m.currentBox)
				m.dbFolder = folder
				return m, m.syncEmails()
			}

		case "r":
			m.state = stateSyncing
			return m, m.syncEmails()

		case "a":
			m.showAI = true
			m.aiInput.Focus()
			m.aiScrollOffset = 0
			return m, textinput.Blink
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case dbInitMsg:
		// Cria/obtém a conta no DB
		var account, err = storage.GetOrCreateAccount(m.account.Email, m.account.Name)
		if err != nil {
			m.err = err
			m.state = stateError
			return m, nil
		}
		m.dbAccount = account

		// Cria/obtém a pasta INBOX
		var folder, err2 = storage.GetOrCreateFolder(account.ID, "INBOX")
		if err2 != nil {
			m.err = err2
			m.state = stateError
			return m, nil
		}
		m.dbFolder = folder

		// Carrega emails do cache primeiro, conecta em paralelo
		m.state = stateLoadingEmails
		return m, tea.Batch(m.loadEmailsFromDB(), m.connect())

	case connectedMsg:
		m.client = msg.client
		m.retrying = false
		// Se já temos emails do cache, faz sync em background sem bloquear UI
		if m.state == stateReady {
			return m, m.loadFolders()
		}
		m.state = stateLoadingFolders
		return m, m.loadFolders()

	case foldersLoadedMsg:
		m.mailboxes = msg.mailboxes

		// Salva pastas no DB e encontra INBOX
		for i, mb := range m.mailboxes {
			var folder, _ = storage.GetOrCreateFolder(m.dbAccount.ID, mb.Name)
			storage.UpdateFolderStats(folder.ID, int(mb.Messages), int(mb.Unseen))

			if strings.EqualFold(mb.Name, "INBOX") {
				m.selectedBox = i
				m.currentBox = mb.Name
				m.dbFolder = folder
			}
		}

		// Sync em background - não muda state se já estamos ready
		if m.state != stateReady {
			m.state = stateSyncing
		}
		return m, m.syncEmails()

	case syncProgressMsg:
		m.syncStatus = msg.status
		m.syncedEmails = msg.synced
		m.totalEmails = msg.total
		return m, nil

	case syncDoneMsg:
		// Recarrega emails do DB após sync
		if m.state != stateReady {
			m.state = stateLoadingEmails
		}
		return m, m.loadEmailsFromDB()

	case emailsLoadedMsg:
		m.emails = msg.emails
		// Sempre vai para ready quando temos emails do cache
		m.state = stateReady

	case configSavedMsg:
		return m, nil

	case aiResponseMsg:
		m.aiLoading = false
		if msg.err != nil {
			m.aiResponse = errorStyle.Render("Erro: " + msg.err.Error())
		} else {
			m.aiResponse = msg.response
		}
		m.aiScrollOffset = 0
		// Recarrega emails (AI pode ter feito alterações)
		return m, m.loadEmailsFromDB()

	case errMsg:
		m.err = msg.err
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
	case stateInitDB:
		return m.viewLoading("Inicializando banco de dados...")
	case stateConnecting:
		var msg = "Conectando ao servidor IMAP..."
		if m.retrying {
			msg = "Reconectando com nova senha..."
		}
		return m.viewLoading(msg)
	case stateLoadingFolders:
		return m.viewLoading("Carregando pastas...")
	case stateSyncing:
		var msg = fmt.Sprintf("Sincronizando %s...", m.currentBox)
		if m.syncStatus != "" {
			msg = m.syncStatus
		}
		return m.viewLoading(msg)
	case stateLoadingEmails:
		return m.viewLoading("Carregando emails do banco local...")
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

	var link = "\x1b]8;;https://myaccount.google.com/apppasswords\x1b\\myaccount.google.com/apppasswords\x1b]8;;\x1b\\"

	var explanation = infoStyle.Render(fmt.Sprintf(`
O Google requer uma "App Password" para apps de email.

Como criar:
1. Acesse: %s
2. Selecione "Mail" e "Outro (miau)"
3. Clique em "Gerar"
4. Copie a senha de 16 caracteres abaixo
`, link))

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
	// Header com stats
	var stats = ""
	if len(m.emails) > 0 {
		stats = fmt.Sprintf(" (%d emails)", len(m.emails))
	}
	var header = headerStyle.Render(fmt.Sprintf(" miau 🐱  %s  [%s]%s ",
		m.account.Email,
		m.currentBox,
		stats,
	))

	// Folders panel (se ativo)
	var foldersPanel string
	if m.showFolders {
		foldersPanel = m.renderFolders()
	}

	// Email list
	var emailList = m.renderEmailList()

	// Footer
	var footer string
	if m.showAI {
		footer = subtitleStyle.Render(" Enter:enviar  ↑↓:scroll  Esc:fechar ")
	} else {
		footer = subtitleStyle.Render(" ↑↓:navegar  Tab:pastas  r:sync  a:AI  q:sair ")
	}

	// Layout
	var content string
	if m.showFolders {
		content = lipgloss.JoinHorizontal(lipgloss.Top, foldersPanel, emailList)
	} else {
		content = emailList
	}

	// AI Panel
	if m.showAI {
		var aiPanel = m.renderAIPanel()
		content = lipgloss.JoinVertical(lipgloss.Left, content, aiPanel)
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
		return boxStyle.Render(subtitleStyle.Render("Nenhum email encontrado.\nPressione 'r' para sincronizar."))
	}

	var lines []string
	var listHeight = m.height - 4
	if listHeight < 5 {
		listHeight = 10
	}

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
		} else if email.IsRead {
			lines = append(lines, readStyle.Render(line))
		} else {
			lines = append(lines, unreadStyle.Render(line))
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) formatEmailLine(email storage.EmailSummary, width int) string {
	var indicator = "●"
	if email.IsRead {
		indicator = " "
	}
	if email.IsStarred {
		indicator = "★"
	}

	var from = email.FromName
	if from == "" {
		from = email.FromEmail
	}
	from = truncate(from, 20)

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

func (m Model) renderAIPanel() string {
	var aiBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4ECDC4")).
		Padding(0, 1)

	var width = m.width - 4
	if width < 40 {
		width = 60
	}

	// Input
	var inputLabel = infoStyle.Render("🤖 AI: ")
	var input = m.aiInput.View()

	// Last question (se houver resposta)
	var lastQ string
	if m.aiLastQuestion != "" && (m.aiResponse != "" || m.aiLoading) {
		lastQ = subtitleStyle.Render("> " + m.aiLastQuestion)
	}

	// Response area
	var response string
	if m.aiLoading {
		response = statusStyle.Render("Pensando...")
	} else if m.aiResponse != "" {
		var lines = strings.Split(m.aiResponse, "\n")
		var maxLines = 10
		var start = m.aiScrollOffset
		if start >= len(lines) {
			start = len(lines) - 1
		}
		if start < 0 {
			start = 0
		}
		var end = start + maxLines
		if end > len(lines) {
			end = len(lines)
		}
		var visibleLines = lines[start:end]
		response = strings.Join(visibleLines, "\n")
		if len(lines) > maxLines {
			response += subtitleStyle.Render(fmt.Sprintf("\n[%d-%d de %d linhas]", start+1, end, len(lines)))
		}
	}

	var content = inputLabel + input
	if lastQ != "" {
		content += "\n" + lastQ
	}
	if response != "" {
		content += "\n\n" + response
	}

	return aiBoxStyle.Width(width).Render(content)
}
