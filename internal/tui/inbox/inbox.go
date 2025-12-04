package inbox

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opik/miau/internal/auth"
	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/gmail"
	"github.com/opik/miau/internal/imap"
	"github.com/opik/miau/internal/smtp"
	"github.com/opik/miau/internal/storage"
	"golang.org/x/net/html"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/htmlindex"
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
	showAI           bool
	aiInput          textinput.Model
	aiResponse       string
	aiLastQuestion   string
	aiLoading        bool
	aiScrollOffset   int
	aiEmailContext   *storage.EmailSummary // Email selecionado para contexto (quando usa Shift+A)
	aiEmailBody      string                // Corpo do email para contexto
	// Spinner
	spinner         spinner.Model
	// Email viewer
	showViewer      bool
	viewerViewport  viewport.Model
	viewerEmail     *storage.EmailSummary
	viewerLoading   bool
	// Compose
	showCompose           bool
	composeTo             textinput.Model
	composeSubject        textinput.Model
	composeBody           viewport.Model
	composeBodyText       string
	composeFocus          int // 0=To, 1=Subject, 2=Body, 3=Classification
	composeSending        bool
	composeReplyTo        *storage.EmailSummary
	composeClassification int // índice em smtp.Classifications
	// Debug
	debugMode       bool
	debugLogs       []string
	debugScroll     int
	// Bounce monitoring
	sentEmails      []SentEmailTracker
	alerts          []Alert
	showAlert       bool
	// Drafts
	showDrafts       bool
	drafts           []storage.Draft
	selectedDraft    int
	editingDraftID   *int64           // Se estamos editando um draft existente
	scheduledDraft   *storage.Draft   // Draft atualmente agendado (para overlay de undo)
	showUndoOverlay  bool
	// Batch operation filter mode
	filterActive      bool                   // Modo de filtro ativo (preview de batch op)
	filterDescription string                 // "Arquivar 15 emails de zaqueu@..."
	pendingBatchOp    *storage.PendingBatchOp // Operação pendente
	originalEmails    []storage.EmailSummary  // Emails originais antes do filtro
	// Fuzzy search
	searchMode    bool                    // Modo de busca ativo
	searchInput   textinput.Model         // Input de busca
	searchResults []storage.EmailSummary  // Resultados da busca
	searchQuery   string                  // Query atual (para highlight)
}

// SentEmailTracker rastreia emails enviados para detectar bounces
type SentEmailTracker struct {
	MessageID    string
	To           string
	Subject      string
	SentAt       time.Time
	MonitorUntil time.Time
}

// Alert representa um alerta para o usuário
type Alert struct {
	Type      string // "bounce", "error", "warning"
	Title     string
	Message   string
	Timestamp time.Time
	EmailTo   string
	Dismissed bool
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

type syncDoneMsg struct {
	synced   int
	total    int
	purged   int
	archived int // emails movidos para arquivo permanente
}

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

type htmlOpenedMsg struct {
	err error
}

type emailContentMsg struct {
	content string
	err     error
}

type aiEmailContextMsg struct {
	email   *storage.EmailSummary
	content string
	err     error
}

type emailSentMsg struct {
	err     error
	host    string
	port    int
	to      string
	msgID   string
	backend string // "smtp" ou "gmail_api"
}

type markReadMsg struct {
	emailID int64
	uid     uint32
}

type debugLogMsg struct {
	msg string
}

type bounceCheckTickMsg struct{}

type bounceFoundMsg struct {
	originalTo      string
	originalSubject string
	bounceReason    string
	bounceFrom      string
	bounceSubject   string
}

// Draft messages
type draftCreatedMsg struct {
	draft *storage.Draft
	err   error
}

type draftScheduledMsg struct {
	draft   *storage.Draft
	sendAt  time.Time
	err     error
}

type draftSentMsg struct {
	draftID int64
	to      string
	backend string
	err     error
}

type draftSendTickMsg struct{}

type draftsLoadedMsg struct {
	drafts    []storage.Draft
	err       error
	accountID int64
}

// Archive/Delete messages
type emailArchivedMsg struct {
	emailID int64
	err     error
}

type emailDeletedMsg struct {
	emailID int64
	err     error
}

// Batch operation filter messages
type batchFilterAppliedMsg struct {
	op     *storage.PendingBatchOp
	emails []storage.EmailSummary
	err    error
}

type batchOpExecutedMsg struct {
	count int
	err   error
}

type checkPendingBatchOpsMsg struct {
	op  *storage.PendingBatchOp
	err error
}

// Search messages
type searchResultsMsg struct {
	results []storage.EmailSummary
	query   string
	err     error
}

type searchDebounceMsg struct {
	query string
}

func New(account *config.Account, debug bool) Model {
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

	// Compose inputs
	var composeTo = textinput.New()
	composeTo.Placeholder = "destinatario@email.com"
	composeTo.CharLimit = 200
	composeTo.Width = 50

	var composeSubject = textinput.New()
	composeSubject.Placeholder = "Assunto"
	composeSubject.CharLimit = 200
	composeSubject.Width = 50

	// Search input
	var searchInput = textinput.New()
	searchInput.Placeholder = "Buscar emails..."
	searchInput.CharLimit = 100
	searchInput.Width = 40

	var s = spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))

	var debugLogs []string
	if debug {
		debugLogs = []string{
			fmt.Sprintf("[%s] 🐛 Debug mode ativado", time.Now().Format("15:04:05")),
			fmt.Sprintf("[%s] 📧 Conta: %s", time.Now().Format("15:04:05"), account.Email),
		}
	}

	return Model{
		account:        account,
		state:          stateInitDB,
		currentBox:     "INBOX",
		showFolders:    false,
		passwordInput:  input,
		aiInput:        aiInput,
		spinner:        s,
		composeTo:      composeTo,
		composeSubject: composeSubject,
		searchInput:    searchInput,
		debugMode:      debug,
		debugLogs:      debugLogs,
	}
}

// log adiciona uma mensagem ao painel de debug
func (m *Model) log(format string, args ...interface{}) {
	if !m.debugMode {
		return
	}
	var timestamp = time.Now().Format("15:04:05")
	var msg = fmt.Sprintf(format, args...)
	m.debugLogs = append(m.debugLogs, fmt.Sprintf("[%s] %s", timestamp, msg))
	// Mantém só as últimas 100 linhas
	if len(m.debugLogs) > 100 {
		m.debugLogs = m.debugLogs[len(m.debugLogs)-100:]
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.initDB(), m.spinner.Tick)
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
		if m.client == nil {
			return errMsg{err: fmt.Errorf("cliente IMAP não conectado")}
		}
		var mailboxes, err = m.client.ListMailboxes()
		if err != nil {
			return errMsg{err: err}
		}
		return foldersLoadedMsg{mailboxes: mailboxes}
	}
}

func (m Model) syncEmails() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{err: fmt.Errorf("cliente IMAP não conectado")}
		}
		// Seleciona a mailbox
		var selectData, err = m.client.SelectMailbox(m.currentBox)
		if err != nil {
			return errMsg{err: fmt.Errorf("erro ao selecionar pasta: %w", err)}
		}

		// Busca último UID que temos no banco
		var latestUID, _ = storage.GetLatestUID(m.dbAccount.ID, m.dbFolder.ID)

		var emails []imap.Email
		var err2 error

		// Log para debug
		if m.debugMode {
			logBounceCheck(fmt.Sprintf("Sync: latestUID=%d, folder=%s", latestUID, m.currentBox))
		}

		if latestUID > 0 {
			// Quick sync: só busca emails novos (UID > último)
			emails, err2 = m.client.FetchNewEmails(latestUID, 100)
		} else {
			// Sync inicial: usa dias configurados
			var cfg, _ = config.Load()
			var syncDays = 30
			if cfg != nil && cfg.Sync.InitialDays > 0 {
				syncDays = cfg.Sync.InitialDays
			}
			if cfg != nil && cfg.Sync.InitialDays == 0 {
				syncDays = 0
			}
			emails, err2 = m.client.FetchEmailsSince(syncDays)
		}

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

		// Detecta emails deletados no servidor
		var serverUIDs, errUIDs = m.client.GetAllUIDs()
		if errUIDs != nil {
			// Log error but continue
			serverUIDs = nil
		}
		var purged, _ = storage.PurgeDeletedFromServer(m.dbAccount.ID, m.dbFolder.ID, serverUIDs)

		// Move emails deletados há mais de 30 dias para arquivo permanente
		var archived, _ = storage.PurgeToArchive(m.dbAccount.ID, 30)

		// Atualiza stats da pasta
		var total, unread, _ = storage.CountEmails(m.dbAccount.ID, m.dbFolder.ID)
		storage.UpdateFolderStats(m.dbFolder.ID, total, unread)

		// Retorna total da caixa para mostrar na UI
		var totalInBox uint32
		if selectData != nil {
			totalInBox = selectData.NumMessages
		}

		return syncDoneMsg{synced: len(emails), total: int(totalInBox), purged: purged, archived: archived}
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

// === SEARCH COMMANDS ===

func (m Model) searchDebounce() tea.Cmd {
	var query = m.searchQuery
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
		return searchDebounceMsg{query: query}
	})
}

func (m Model) performSearch(query string) tea.Cmd {
	var accountID = m.dbAccount.ID
	return func() tea.Msg {
		var results, err = storage.FuzzySearchEmails(accountID, query, 100)
		return searchResultsMsg{results: results, query: query, err: err}
	}
}

func (m Model) loadEmailForAI() tea.Cmd {
	return func() tea.Msg {
		if len(m.emails) == 0 || m.selectedEmail >= len(m.emails) {
			return aiEmailContextMsg{err: fmt.Errorf("nenhum email selecionado")}
		}

		var email = m.emails[m.selectedEmail]

		if m.client == nil {
			return aiEmailContextMsg{email: &email, content: "", err: nil}
		}

		// Seleciona a mailbox antes de buscar
		if _, err := m.client.SelectMailbox(m.currentBox); err != nil {
			return aiEmailContextMsg{email: &email, content: "", err: nil}
		}

		var rawData, err = m.client.FetchEmailRaw(email.UID)
		if err != nil {
			return aiEmailContextMsg{email: &email, content: "", err: nil}
		}

		// Tenta extrair texto plain primeiro, depois HTML convertido
		var textContent = extractText(rawData)
		if textContent == "" {
			var htmlContent = extractHTML(rawData)
			if htmlContent != "" {
				textContent = htmlToText(htmlContent)
			}
		}

		return aiEmailContextMsg{email: &email, content: textContent, err: nil}
	}
}

func (m Model) runAI(prompt string) tea.Cmd {
	// Copia contexto para a goroutine
	var emailContext = m.aiEmailContext
	var emailBody = m.aiEmailBody

	return func() tea.Msg {
		var fullPrompt string

		if emailContext != nil {
			// Prompt com contexto do email selecionado
			var emailInfo = fmt.Sprintf(`[Email selecionado]
De: %s <%s>
Assunto: %s
Data: %s
---
%s
---`, emailContext.FromName, emailContext.FromEmail, emailContext.Subject, emailContext.Date.Time.Format("02/01/2006 15:04"), emailBody)

			fullPrompt = fmt.Sprintf(`[Context: Email database at ~/.config/miau/data/miau.db | Account: %s | Folder: %s]
[Schema: emails(id, subject, from_name, from_email, date, is_read, is_starred, is_deleted, body_text)]
[Use sqlite3 to query. Be concise.]

%s

Pergunta do usuário sobre este email:
%s`, m.account.Email, m.currentBox, emailInfo, prompt)
		} else {
			// Prompt geral (sem contexto de email específico)
			fullPrompt = fmt.Sprintf(`[Context: Email database at ~/.config/miau/data/miau.db | Account: %s | Folder: %s]
[Schema: emails(id, subject, from_name, from_email, date, is_read, is_starred, is_deleted, body_text)]
[Use sqlite3 to query. Be concise.]

%s`, m.account.Email, m.currentBox, prompt)
		}

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

func (m Model) markAsRead(emailID int64, uid uint32) tea.Cmd {
	return func() tea.Msg {
		// Marca no banco local
		storage.MarkAsRead(emailID, true)

		// Marca no servidor IMAP
		if m.client != nil {
			m.client.MarkAsRead(uid)
		}

		return markReadMsg{emailID: emailID, uid: uid}
	}
}

func (m Model) sendEmail() tea.Cmd {
	return func() tea.Msg {
		var to = strings.TrimSpace(m.composeTo.Value())
		var subject = strings.TrimSpace(m.composeSubject.Value())
		var body = m.composeBodyText

		// Carrega config para verificar formato e assinatura
		var cfg, _ = config.Load()
		var useHTML = cfg == nil || cfg.Compose.Format != "plain"

		// Monta o corpo do email
		var emailBody string
		if useHTML {
			emailBody = "<html><body>"
			emailBody += "<div style=\"font-family: Arial, sans-serif; font-size: 14px;\">"
			emailBody += strings.ReplaceAll(body, "\n", "<br>")
			emailBody += "</div>"

			// Adiciona assinatura HTML se configurada
			if m.account.Signature != nil && m.account.Signature.Enabled && m.account.Signature.HTML != "" {
				emailBody += "<br><br>"
				emailBody += "<div style=\"border-top: 1px solid #ccc; padding-top: 10px; margin-top: 10px;\">"
				emailBody += m.account.Signature.HTML
				emailBody += "</div>"
			}

			emailBody += "</body></html>"
		} else {
			// Plain text
			emailBody = body

			// Adiciona assinatura de texto se configurada
			if m.account.Signature != nil && m.account.Signature.Enabled && m.account.Signature.Text != "" {
				emailBody += "\n\n--\n"
				emailBody += m.account.Signature.Text
			}
		}

		// Headers de threading
		var inReplyTo, references string
		if m.composeReplyTo != nil && m.composeReplyTo.MessageID.Valid {
			inReplyTo = m.composeReplyTo.MessageID.String
			references = inReplyTo
		}

		// Verifica se deve usar Gmail API
		if m.account.SendMethod == config.SendMethodGmailAPI && m.account.OAuth2 != nil {
			return m.sendViaGmailAPI(to, subject, emailBody, useHTML, inReplyTo, references)
		}

		// Fallback para SMTP
		var client = smtp.NewClient(m.account)
		var email = &smtp.Email{
			To:             []string{to},
			Subject:        subject,
			Body:           emailBody,
			Classification: smtp.Classifications[m.composeClassification],
			IsHTML:         useHTML,
			InReplyTo:      inReplyTo,
			References:     references,
		}

		var result, err = client.Send(email)
		if err != nil {
			return emailSentMsg{err: err, to: to, backend: "smtp"}
		}

		return emailSentMsg{
			host:    result.Host,
			port:    result.Port,
			to:      to,
			msgID:   result.MessageID,
			backend: "smtp",
		}
	}
}

func (m Model) sendViaGmailAPI(to, subject, body string, isHTML bool, inReplyTo, references string) tea.Msg {
	var tokenPath = auth.GetTokenPath(config.GetConfigPath(), m.account.Name)
	var oauthCfg = auth.GetOAuth2Config(m.account.OAuth2.ClientID, m.account.OAuth2.ClientSecret)

	var token, err = auth.GetValidToken(oauthCfg, tokenPath)
	if err != nil {
		return emailSentMsg{err: fmt.Errorf("erro ao obter token OAuth2: %w", err), to: to, backend: "gmail_api"}
	}

	var client = gmail.NewClient(token, oauthCfg, m.account.Email)

	// Monta request
	var req = &gmail.SendRequest{
		To:         []string{to},
		Subject:    subject,
		Body:       body,
		IsHTML:     isHTML,
		InReplyTo:  inReplyTo,
		References: references,
	}

	// Adiciona classification se houver (índice 0 = sem classificação)
	if m.composeClassification > 0 {
		// Por enquanto usa o nome da classificação como label ID
		// TODO: mapear para label IDs reais do Gmail
		req.ClassificationID = smtp.Classifications[m.composeClassification]
	}

	var resp, err2 = client.SendMessage(req)
	if err2 != nil {
		return emailSentMsg{err: err2, to: to, backend: "gmail_api"}
	}

	return emailSentMsg{
		to:      to,
		msgID:   resp.ID,
		backend: "gmail_api",
	}
}

// checkForBounces verifica se há mensagens de bounce para emails enviados recentemente
func (m Model) checkForBounces() tea.Cmd {
	// Copia dados necessários para a goroutine
	var trackers = m.sentEmails
	var accountID int64
	if m.dbAccount != nil {
		accountID = m.dbAccount.ID
	}
	var client = m.client
	var debugMode = m.debugMode

	return func() tea.Msg {
		// Remove trackers expirados
		var now = time.Now()
		var activeTrackers []SentEmailTracker
		for _, tracker := range trackers {
			if now.Before(tracker.MonitorUntil) {
				activeTrackers = append(activeTrackers, tracker)
			}
		}

		if len(activeTrackers) == 0 || accountID == 0 {
			return nil
		}

		// Sincroniza pastas que podem ter bounces
		if client != nil {
			var bounceFolders = []string{"INBOX", "CATEGORY_UPDATES", "[Gmail]/All Mail"}
			for _, folderName := range bounceFolders {
				var folder, _ = storage.GetOrCreateFolder(accountID, folderName)
				if folder == nil {
					continue
				}
				var selectData, err = client.SelectMailbox(folderName)
				if err != nil {
					continue
				}
				if selectData.NumMessages > 0 {
					var emails, _ = client.FetchEmailsSeqNum(selectData, 15)
					for _, email := range emails {
						var dbEmail = &storage.Email{
							AccountID: accountID,
							FolderID:  folder.ID,
							UID:       email.UID,
							MessageID: sql.NullString{String: email.MessageID, Valid: email.MessageID != ""},
							Subject:   email.Subject,
							FromName:  email.From,
							FromEmail: email.FromEmail,
							Date:      storage.SQLiteTime{Time: email.Date},
							IsRead:    email.Seen,
							Size:      email.Size,
						}
						storage.UpsertEmail(dbEmail)
					}
				}
			}
		}

		// Busca TODOS os emails recentes de TODAS as pastas do banco
		var allEmails []storage.EmailSummary
		var folders, _ = storage.GetFolders(accountID)
		for _, folder := range folders {
			var emails, _ = storage.GetEmails(accountID, folder.ID, 30, 0)
			allEmails = append(allEmails, emails...)
		}

		// Log para debug
		if debugMode {
			// Escreve log em arquivo para não perder
			logBounceCheck(fmt.Sprintf("Verificando %d emails para bounce", len(allEmails)))
		}

		// Procura por bounces em TODOS os emails
		for _, email := range allEmails {
			// Log cada email verificado para debug
			if debugMode {
				var isBounce = isBounceEmail(email.FromEmail, email.FromName, email.Subject)
				logBounceCheck(fmt.Sprintf("  Email: from='%s' name='%s' subj='%s' date='%s' => bounce=%v",
					email.FromEmail, email.FromName, email.Subject, email.Date.Time.Format("15:04:05"), isBounce))
			}

			// Detecta se é bounce pelo remetente/subject
			if !isBounceEmail(email.FromEmail, email.FromName, email.Subject) {
				continue
			}

			// Encontrou um bounce! Verifica se corresponde a algum tracker
			for _, tracker := range activeTrackers {
				// Bounce deve ser DEPOIS do envio
				if email.Date.Time.Before(tracker.SentAt) {
					if debugMode {
						logBounceCheck(fmt.Sprintf("    Bounce muito antigo: bounce=%s sent=%s",
							email.Date.Time.Format("15:04:05"), tracker.SentAt.Format("15:04:05")))
					}
					continue
				}

				// Verifica se o bounce menciona o destinatário do email enviado
				var bounceContent = strings.ToLower(email.Subject + " " + email.Snippet)
				var recipientLower = strings.ToLower(tracker.To)
				if !strings.Contains(bounceContent, recipientLower) {
					if debugMode {
						logBounceCheck(fmt.Sprintf("    Bounce não menciona destinatário '%s'", tracker.To))
					}
					continue
				}

				// Match! Bounce corresponde ao email enviado
				var reason = extractBounceReason(email.Snippet, email.Subject)
				logBounceCheck(fmt.Sprintf("🚨 BOUNCE ENCONTRADO! to=%s reason=%s", tracker.To, reason))

				return bounceFoundMsg{
					originalTo:      tracker.To,
					originalSubject: tracker.Subject,
					bounceReason:    reason,
					bounceFrom:      email.FromEmail,
					bounceSubject:   email.Subject,
				}
			}
		}

		return nil
	}
}

// logBounceCheck escreve log de bounce em arquivo para debug
func logBounceCheck(msg string) {
	var f, _ = os.OpenFile("/tmp/miau-bounce.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if f != nil {
		f.WriteString(fmt.Sprintf("[%s] %s\n", time.Now().Format("15:04:05"), msg))
		f.Close()
	}
}

// isBounceEmail detecta se um email é uma mensagem de bounce/NDR
func isBounceEmail(fromEmail, fromName, subject string) bool {
	var from = strings.ToLower(fromEmail + " " + fromName)
	var subj = strings.ToLower(subject)

	// Remetentes típicos de bounce
	var bounceSenders = []string{
		"mailer-daemon",
		"postmaster",
		"mail delivery subsystem",
		"mail delivery",
		"mailerdaemon",
		"noreply",
		"no-reply",
		"mail-daemon",
		"delivery",
		"daemon",
		"bounce",
		"mailmaster",
	}

	for _, sender := range bounceSenders {
		if strings.Contains(from, sender) {
			return true
		}
	}

	// Subjects típicos de bounce
	var bounceSubjects = []string{
		"delivery status notification",
		"delivery status",
		"delivery failed",
		"delivery failure",
		"undeliverable",
		"undelivered",
		"returned mail",
		"mail delivery failed",
		"failure notice",
		"não foi possível entregar",
		"falha na entrega",
		"mensagem devolvida",
		"não entregue",
		"rejected",
		"mail returned",
		"returned to sender",
		"could not be delivered",
		"notification (failure)",
		"(failure)",
	}

	for _, bs := range bounceSubjects {
		if strings.Contains(subj, bs) {
			return true
		}
	}

	return false
}

// extractBounceReason extrai a razão do bounce do conteúdo
func extractBounceReason(snippet, subject string) string {
	var content = strings.ToLower(snippet + " " + subject)

	// Razões comuns
	var reasons = map[string]string{
		"classificação":             "Requer classificação de email",
		"classification":            "Requires email classification",
		"spam":                      "Marcado como spam",
		"rejected":                  "Rejeitado pelo servidor",
		"user unknown":              "Usuário desconhecido",
		"mailbox full":              "Caixa de correio cheia",
		"quota exceeded":            "Cota excedida",
		"does not exist":            "Endereço não existe",
		"address rejected":          "Endereço rejeitado",
		"policy":                    "Violação de política",
		"blocked":                   "Bloqueado",
		"blacklist":                 "Na lista negra",
		"administrador":             "Bloqueado pelo administrador",
		"administrator":             "Blocked by administrator",
		"enterprise administrator":  "Bloqueado pela política corporativa",
	}

	for key, reason := range reasons {
		if strings.Contains(content, key) {
			return reason
		}
	}

	// Se não encontrar razão específica, retorna genérico
	if len(snippet) > 100 {
		return snippet[:100] + "..."
	}
	return snippet
}

// scheduleBounceCheck agenda verificação de bounce
func scheduleBounceCheck() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return bounceCheckTickMsg{}
	})
}

// scheduleDraftSend agenda verificação de drafts prontos para envio
func scheduleDraftSend() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return draftSendTickMsg{}
	})
}

// loadDrafts carrega drafts pendentes do banco
func (m Model) loadDrafts() tea.Cmd {
	var accountID int64
	if m.dbAccount != nil {
		accountID = m.dbAccount.ID
	}
	return func() tea.Msg {
		if accountID == 0 {
			return draftsLoadedMsg{err: fmt.Errorf("conta não inicializada")}
		}
		var drafts, err = storage.GetPendingDrafts(accountID)
		return draftsLoadedMsg{drafts: drafts, err: err, accountID: accountID}
	}
}

// archiveEmail arquiva um email (local + servidor)
func (m Model) archiveEmail(emailID int64, uid uint32, messageID string) tea.Cmd {
	return func() tea.Msg {
		// 1. Marca como arquivado no banco local
		if err := storage.MarkAsArchived(emailID, true); err != nil {
			return emailArchivedMsg{emailID: emailID, err: err}
		}

		// 2. Arquiva no servidor
		if m.account.AuthType == config.AuthTypeOAuth2 && m.account.SendMethod == config.SendMethodGmailAPI {
			// Usa Gmail API
			var tokenPath = auth.GetTokenPath(config.GetConfigPath(), m.account.Name)
			var oauthCfg = auth.GetOAuth2Config(m.account.OAuth2.ClientID, m.account.OAuth2.ClientSecret)
			var token, err = auth.GetValidToken(oauthCfg, tokenPath)
			if err == nil {
				var gmailClient = gmail.NewClient(token, oauthCfg, m.account.Email)
				// Busca ID da mensagem no Gmail pelo Message-ID RFC822
				if gmailMsgID, err := gmailClient.GetMessageIDByRFC822MsgID(messageID); err == nil {
					gmailClient.ArchiveMessage(gmailMsgID)
				}
			}
		} else if m.client != nil {
			// Usa IMAP
			m.client.ArchiveEmail(uid)
		}

		return emailArchivedMsg{emailID: emailID, err: nil}
	}
}

// deleteEmail deleta um email (move para lixeira - local + servidor)
func (m Model) deleteEmail(emailID int64, uid uint32, messageID string) tea.Cmd {
	return func() tea.Msg {
		// 1. Marca como deletado no banco local
		if err := storage.DeleteEmail(emailID); err != nil {
			return emailDeletedMsg{emailID: emailID, err: err}
		}

		// 2. Move para lixeira no servidor
		if m.account.AuthType == config.AuthTypeOAuth2 && m.account.SendMethod == config.SendMethodGmailAPI {
			// Usa Gmail API
			var tokenPath = auth.GetTokenPath(config.GetConfigPath(), m.account.Name)
			var oauthCfg = auth.GetOAuth2Config(m.account.OAuth2.ClientID, m.account.OAuth2.ClientSecret)
			var token, err = auth.GetValidToken(oauthCfg, tokenPath)
			if err == nil {
				var gmailClient = gmail.NewClient(token, oauthCfg, m.account.Email)
				// Busca ID da mensagem no Gmail pelo Message-ID RFC822
				if gmailMsgID, err := gmailClient.GetMessageIDByRFC822MsgID(messageID); err == nil {
					gmailClient.TrashMessage(gmailMsgID)
				}
			}
		} else if m.client != nil {
			// Usa IMAP
			var trashFolder = m.client.GetTrashFolder()
			m.client.TrashEmail(uid, trashFolder)
		}

		return emailDeletedMsg{emailID: emailID, err: nil}
	}
}

// applyBatchFilter aplica filtro para preview de batch operation
func (m Model) applyBatchFilter(op *storage.PendingBatchOp) tea.Cmd {
	return func() tea.Msg {
		// Parse email IDs do JSON
		var emailIDs []int64
		if err := json.Unmarshal([]byte(op.EmailIDs), &emailIDs); err != nil {
			return batchFilterAppliedMsg{err: err}
		}

		// Busca os emails
		var emails, err = storage.GetEmailsByIDs(emailIDs)
		if err != nil {
			return batchFilterAppliedMsg{err: err}
		}

		return batchFilterAppliedMsg{op: op, emails: emails, err: nil}
	}
}

// executeBatchOp executa a operação em lote confirmada
func (m Model) executeBatchOp() tea.Cmd {
	return func() tea.Msg {
		if m.pendingBatchOp == nil {
			return batchOpExecutedMsg{err: fmt.Errorf("nenhuma operação pendente")}
		}

		var count, err = storage.ExecuteBatchOp(m.pendingBatchOp.ID)
		return batchOpExecutedMsg{count: count, err: err}
	}
}

// cancelBatchOp cancela a operação em lote
func (m Model) cancelBatchOp() tea.Cmd {
	return func() tea.Msg {
		if m.pendingBatchOp != nil {
			storage.CancelBatchOp(m.pendingBatchOp.ID)
		}
		return nil
	}
}

// checkPendingBatchOps verifica se há operações em lote pendentes (após AI response)
func (m Model) checkPendingBatchOps() tea.Cmd {
	var accountID int64
	if m.dbAccount != nil {
		accountID = m.dbAccount.ID
	}
	return func() tea.Msg {
		if accountID == 0 {
			return checkPendingBatchOpsMsg{err: nil}
		}

		var ops, err = storage.GetPendingBatchOps(accountID)
		if err != nil {
			return checkPendingBatchOpsMsg{err: err}
		}

		// Retorna a operação mais recente pendente
		if len(ops) > 0 {
			return checkPendingBatchOpsMsg{op: &ops[0], err: nil}
		}

		return checkPendingBatchOpsMsg{err: nil}
	}
}

// createScheduledDraft cria um draft e agenda para envio
func (m Model) createScheduledDraft() tea.Cmd {
	return func() tea.Msg {
		var cfg, err = config.Load()
		if err != nil {
			return draftCreatedMsg{err: err}
		}

		var to = m.composeTo.Value()
		var subject = m.composeSubject.Value()
		var bodyText = m.composeBodyText

		// Determina formato e prepara corpo
		var isHTML = cfg.Compose.Format != "plain"
		var bodyHTML string

		if isHTML {
			bodyHTML = "<html><body>" + strings.ReplaceAll(bodyText, "\n", "<br>") + "</body></html>"
			// Adiciona assinatura HTML se configurada
			if m.account.Signature != nil && m.account.Signature.Enabled && m.account.Signature.HTML != "" {
				bodyHTML = strings.Replace(bodyHTML, "</body></html>",
					"<br><br>"+m.account.Signature.HTML+"</body></html>", 1)
			}
		} else {
			// Adiciona assinatura texto se configurada
			if m.account.Signature != nil && m.account.Signature.Enabled && m.account.Signature.Text != "" {
				bodyText = bodyText + "\n\n--\n" + m.account.Signature.Text
			}
		}

		// Threading headers
		var inReplyTo, references string
		var replyToEmailID sql.NullInt64
		if m.composeReplyTo != nil && m.composeReplyTo.MessageID.Valid {
			inReplyTo = m.composeReplyTo.MessageID.String
			references = m.composeReplyTo.MessageID.String
			replyToEmailID = sql.NullInt64{Int64: m.composeReplyTo.ID, Valid: true}
		}

		// Classificação
		var classification string
		if m.composeClassification > 0 && m.composeClassification < len(smtp.Classifications) {
			classification = smtp.Classifications[m.composeClassification]
		}

		// Cria draft
		var draft = &storage.Draft{
			AccountID:        m.dbAccount.ID,
			ToAddresses:      to,
			Subject:          subject,
			BodyHTML:         sql.NullString{String: bodyHTML, Valid: bodyHTML != ""},
			BodyText:         sql.NullString{String: bodyText, Valid: bodyText != ""},
			Classification:   sql.NullString{String: classification, Valid: classification != ""},
			InReplyTo:        sql.NullString{String: inReplyTo, Valid: inReplyTo != ""},
			ReferenceIDs:     sql.NullString{String: references, Valid: references != ""},
			ReplyToEmailID:   replyToEmailID,
			Status:           storage.DraftStatusScheduled,
			GenerationSource: "manual",
		}

		// Calcula tempo de envio
		var delay = time.Duration(cfg.Compose.SendDelaySeconds) * time.Second
		var sendAt = time.Now().Add(delay)
		draft.ScheduledSendAt = sql.NullTime{Time: sendAt, Valid: true}

		var draftID, err2 = storage.CreateDraft(draft)
		if err2 != nil {
			return draftCreatedMsg{err: err2}
		}

		draft.ID = draftID
		return draftScheduledMsg{draft: draft, sendAt: sendAt, err: nil}
	}
}

// sendDraft envia um draft específico
func (m Model) sendDraft(draftID int64) tea.Cmd {
	return func() tea.Msg {
		var draft, err = storage.GetDraftByID(draftID)
		if err != nil {
			return draftSentMsg{draftID: draftID, err: err}
		}

		// Marca como enviando
		storage.MarkDraftSending(draftID)

		// Determina backend e envia
		var backend = "smtp"
		if m.account.SendMethod == config.SendMethodGmailAPI {
			// Gmail API
			var tokenPath = auth.GetTokenPath(config.GetConfigPath(), m.account.Name)
			var oauthCfg = auth.GetOAuth2Config(m.account.OAuth2.ClientID, m.account.OAuth2.ClientSecret)
			var token, err = auth.GetValidToken(oauthCfg, tokenPath)
			if err != nil {
				storage.MarkDraftFailed(draftID, err.Error())
				return draftSentMsg{draftID: draftID, err: err}
			}

			var client = gmail.NewClient(token, oauthCfg, m.account.Email)
			var req = &gmail.SendRequest{
				To:         []string{draft.ToAddresses},
				Subject:    draft.Subject,
				Body:       draft.BodyText.String,
				InReplyTo:  draft.InReplyTo.String,
				References: draft.ReferenceIDs.String,
				IsHTML:     draft.BodyHTML.Valid && draft.BodyHTML.String != "",
			}
			if draft.BodyHTML.Valid && draft.BodyHTML.String != "" {
				req.Body = draft.BodyHTML.String
			}

			var _, err2 = client.SendMessage(req)
			if err2 != nil {
				storage.MarkDraftFailed(draftID, err2.Error())
				return draftSentMsg{draftID: draftID, err: err2}
			}
			backend = "gmail_api"
		} else {
			// SMTP
			var smtpClient = smtp.NewClient(m.account)
			var email = smtp.Email{
				To:             []string{draft.ToAddresses},
				Subject:        draft.Subject,
				Body:           draft.BodyText.String,
				InReplyTo:      draft.InReplyTo.String,
				References:     draft.ReferenceIDs.String,
				Classification: draft.Classification.String,
				IsHTML:         draft.BodyHTML.Valid && draft.BodyHTML.String != "",
			}
			if draft.BodyHTML.Valid && draft.BodyHTML.String != "" {
				email.Body = draft.BodyHTML.String
			}

			var _, err = smtpClient.Send(&email)
			if err != nil {
				storage.MarkDraftFailed(draftID, err.Error())
				return draftSentMsg{draftID: draftID, err: err}
			}
		}

		// Registra email enviado permanentemente
		storage.RecordSentEmail(
			draft.AccountID,
			"", // messageID (preenchido pelo servidor)
			draft.ToAddresses,
			draft.CcAddresses.String,
			draft.BccAddresses.String,
			draft.Subject,
			draft.BodyHTML.String,
			draft.BodyText.String,
			draft.InReplyTo.String,
			draft.ReferenceIDs.String,
			backend,
			draft.ReplyToEmailID,
			sql.NullInt64{Int64: draftID, Valid: true},
		)

		// Arquiva o draft no histórico permanente (nunca deletamos)
		storage.ArchiveDraftPermanently(draftID, "sent")

		// Se era reply, marca email original como respondido
		if draft.ReplyToEmailID.Valid {
			storage.MarkAsReplied(draft.ReplyToEmailID.Int64)
		}

		return draftSentMsg{draftID: draftID, to: draft.ToAddresses, backend: backend, err: nil}
	}
}

func (m Model) openEmailHTML() tea.Cmd {
	return func() tea.Msg {
		if len(m.emails) == 0 || m.selectedEmail >= len(m.emails) {
			return htmlOpenedMsg{err: fmt.Errorf("nenhum email selecionado")}
		}

		var email = m.emails[m.selectedEmail]

		// Tenta buscar do servidor IMAP
		if m.client == nil {
			return htmlOpenedMsg{err: fmt.Errorf("não conectado ao servidor")}
		}

		// Seleciona a mailbox antes de buscar
		if _, err := m.client.SelectMailbox(m.currentBox); err != nil {
			return htmlOpenedMsg{err: fmt.Errorf("erro ao selecionar pasta: %w", err)}
		}

		var rawData, err = m.client.FetchEmailRaw(email.UID)
		if err != nil {
			return htmlOpenedMsg{err: err}
		}

		// Parseia o email para extrair HTML
		var htmlContent = extractHTML(rawData)
		if htmlContent == "" {
			return htmlOpenedMsg{err: fmt.Errorf("email não contém HTML")}
		}

		// Salva em arquivo temporário
		var tmpDir = os.TempDir()
		var tmpFile = filepath.Join(tmpDir, fmt.Sprintf("miau-email-%d.html", email.ID))
		if err := os.WriteFile(tmpFile, []byte(htmlContent), 0600); err != nil {
			return htmlOpenedMsg{err: err}
		}

		// Abre no navegador padrão
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "linux":
			cmd = exec.Command("xdg-open", tmpFile)
		case "darwin":
			cmd = exec.Command("open", tmpFile)
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", tmpFile)
		default:
			return htmlOpenedMsg{err: fmt.Errorf("sistema operacional não suportado")}
		}

		if err := cmd.Start(); err != nil {
			return htmlOpenedMsg{err: err}
		}

		return htmlOpenedMsg{}
	}
}

func (m Model) loadEmailContent() tea.Cmd {
	return func() tea.Msg {
		if len(m.emails) == 0 || m.selectedEmail >= len(m.emails) {
			return emailContentMsg{err: fmt.Errorf("nenhum email selecionado")}
		}

		var email = m.emails[m.selectedEmail]

		if m.client == nil {
			return emailContentMsg{err: fmt.Errorf("não conectado ao servidor")}
		}

		// Seleciona a mailbox antes de buscar
		if _, err := m.client.SelectMailbox(m.currentBox); err != nil {
			return emailContentMsg{err: fmt.Errorf("erro ao selecionar pasta: %w", err)}
		}

		var rawData, err = m.client.FetchEmailRaw(email.UID)
		if err != nil {
			return emailContentMsg{err: err}
		}

		// Tenta extrair texto plain primeiro, depois HTML convertido
		var textContent = extractText(rawData)
		if textContent == "" {
			var htmlContent = extractHTML(rawData)
			if htmlContent != "" {
				textContent = htmlToText(htmlContent)
			}
		}

		if textContent == "" {
			return emailContentMsg{err: fmt.Errorf("email sem conteúdo de texto")}
		}

		return emailContentMsg{content: textContent}
	}
}

// extractText extrai conteúdo text/plain de um email MIME
func extractText(rawData []byte) string {
	var msg, err = mail.ReadMessage(bytes.NewReader(rawData))
	if err != nil {
		return ""
	}

	var contentType = msg.Header.Get("Content-Type")
	var mediaType, params, _ = mime.ParseMediaType(contentType)

	// Se for texto direto
	if strings.HasPrefix(mediaType, "text/plain") {
		var body, _ = io.ReadAll(msg.Body)
		return decodeBody(body, msg.Header.Get("Content-Transfer-Encoding"))
	}

	// Se for multipart, procura a parte text/plain
	if strings.HasPrefix(mediaType, "multipart/") {
		var boundary = params["boundary"]
		if boundary != "" {
			return findTextPart(msg.Body, boundary)
		}
	}

	return ""
}

func findTextPart(r io.Reader, boundary string) string {
	var mr = multipart.NewReader(r, boundary)
	for {
		var part, err = mr.NextPart()
		if err != nil {
			break
		}

		var contentType = part.Header.Get("Content-Type")
		var mediaType, params, _ = mime.ParseMediaType(contentType)

		if strings.HasPrefix(mediaType, "text/plain") {
			var body, _ = io.ReadAll(part)
			return decodeBody(body, part.Header.Get("Content-Transfer-Encoding"))
		}

		// Multipart aninhado
		if strings.HasPrefix(mediaType, "multipart/") {
			var boundary = params["boundary"]
			if boundary != "" {
				if text := findTextPart(part, boundary); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

// extractHTML extrai conteúdo HTML de um email MIME
func extractHTML(rawData []byte) string {
	var htmlContent, cidMap = extractHTMLWithCID(rawData)

	// Substitui referências cid: por data URIs
	if len(cidMap) > 0 {
		htmlContent = replaceCIDReferences(htmlContent, cidMap)
	}

	return htmlContent
}

// extractHTMLWithCID extrai HTML e mapa de imagens CID
func extractHTMLWithCID(rawData []byte) (string, map[string]string) {
	var cidMap = make(map[string]string)

	var msg, err = mail.ReadMessage(bytes.NewReader(rawData))
	if err != nil {
		return "", cidMap
	}

	var contentType = msg.Header.Get("Content-Type")
	var mediaType, params, _ = mime.ParseMediaType(contentType)

	// Se for HTML direto
	if strings.HasPrefix(mediaType, "text/html") {
		var body, _ = io.ReadAll(msg.Body)
		var charset = params["charset"]
		return decodeBodyWithCharset(body, msg.Header.Get("Content-Transfer-Encoding"), charset), cidMap
	}

	// Se for multipart, procura a parte HTML e imagens
	if strings.HasPrefix(mediaType, "multipart/") {
		var boundary = params["boundary"]
		if boundary != "" {
			return findHTMLAndImages(msg.Body, boundary, cidMap)
		}
	}

	return "", cidMap
}

// findHTMLAndImages procura HTML e extrai imagens embutidas
func findHTMLAndImages(r io.Reader, boundary string, cidMap map[string]string) (string, map[string]string) {
	var htmlContent string
	var mr = multipart.NewReader(r, boundary)

	// Primeira passagem: coleta todas as partes
	type mimePart struct {
		contentType string
		contentID   string
		encoding    string
		body        []byte
	}
	var parts []mimePart

	for {
		var part, err = mr.NextPart()
		if err != nil {
			break
		}

		var body, _ = io.ReadAll(part)
		parts = append(parts, mimePart{
			contentType: part.Header.Get("Content-Type"),
			contentID:   part.Header.Get("Content-Id"),
			encoding:    part.Header.Get("Content-Transfer-Encoding"),
			body:        body,
		})
	}

	// Processa as partes
	for _, part := range parts {
		var mediaType, params, _ = mime.ParseMediaType(part.contentType)

		// HTML
		if strings.HasPrefix(mediaType, "text/html") && htmlContent == "" {
			var charset = params["charset"]
			htmlContent = decodeBodyWithCharset(part.body, part.encoding, charset)
		}

		// Imagens com Content-ID
		var contentID = part.contentID
		if contentID != "" && strings.HasPrefix(mediaType, "image/") {
			// Remove < > do Content-ID
			contentID = strings.Trim(contentID, "<>")

			// Decodifica o body da imagem
			var imageData = decodeImageBody(part.body, part.encoding)

			// Cria data URI
			var dataURI = fmt.Sprintf("data:%s;base64,%s", mediaType, base64.StdEncoding.EncodeToString(imageData))
			cidMap[contentID] = dataURI
		}

		// Multipart aninhado
		if strings.HasPrefix(mediaType, "multipart/") {
			var nestedBoundary = params["boundary"]
			if nestedBoundary != "" {
				var nestedHTML, nestedCID = findHTMLAndImages(bytes.NewReader(part.body), nestedBoundary, cidMap)
				if nestedHTML != "" && htmlContent == "" {
					htmlContent = nestedHTML
				}
				for k, v := range nestedCID {
					cidMap[k] = v
				}
			}
		}
	}

	return htmlContent, cidMap
}

// decodeImageBody decodifica o corpo de uma imagem
func decodeImageBody(body []byte, encoding string) []byte {
	switch strings.ToLower(encoding) {
	case "base64":
		var decoded, err = base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			// Tenta remover espaços/newlines
			var cleaned = strings.ReplaceAll(string(body), "\n", "")
			cleaned = strings.ReplaceAll(cleaned, "\r", "")
			cleaned = strings.ReplaceAll(cleaned, " ", "")
			decoded, _ = base64.StdEncoding.DecodeString(cleaned)
		}
		return decoded
	case "quoted-printable":
		var decoded, _ = io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		return decoded
	default:
		return body
	}
}

// replaceCIDReferences substitui cid:xxx por data URIs
func replaceCIDReferences(html string, cidMap map[string]string) string {
	// Padrão: src="cid:xxx" ou src='cid:xxx'
	var cidRegex = regexp.MustCompile(`(src=["'])cid:([^"']+)(["'])`)

	return cidRegex.ReplaceAllStringFunc(html, func(match string) string {
		var submatches = cidRegex.FindStringSubmatch(match)
		if len(submatches) >= 4 {
			var cid = submatches[2]
			if dataURI, ok := cidMap[cid]; ok {
				return submatches[1] + dataURI + submatches[3]
			}
		}
		return match
	})
}

func findHTMLPart(r io.Reader, boundary string) string {
	var mr = multipart.NewReader(r, boundary)
	for {
		var part, err = mr.NextPart()
		if err != nil {
			break
		}

		var contentType = part.Header.Get("Content-Type")
		var mediaType, params, _ = mime.ParseMediaType(contentType)

		if strings.HasPrefix(mediaType, "text/html") {
			var body, _ = io.ReadAll(part)
			return decodeBody(body, part.Header.Get("Content-Transfer-Encoding"))
		}

		// Multipart aninhado
		if strings.HasPrefix(mediaType, "multipart/") {
			var boundary = params["boundary"]
			if boundary != "" {
				if html := findHTMLPart(part, boundary); html != "" {
					return html
				}
			}
		}
	}
	return ""
}

func decodeBody(body []byte, encoding string) string {
	return decodeBodyWithCharset(body, encoding, "")
}

func decodeBodyWithCharset(body []byte, encoding string, charset string) string {
	var decoded []byte

	switch strings.ToLower(encoding) {
	case "quoted-printable":
		var d, err = io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		if err != nil {
			decoded = body
		} else {
			decoded = d
		}
	case "base64":
		var d, err = base64.StdEncoding.DecodeString(string(body))
		if err != nil {
			// Tenta limpar
			var cleaned = strings.ReplaceAll(string(body), "\n", "")
			cleaned = strings.ReplaceAll(cleaned, "\r", "")
			d, _ = base64.StdEncoding.DecodeString(cleaned)
		}
		decoded = d
	default:
		decoded = body
	}

	// Converte charset se necessário
	if charset != "" && !strings.EqualFold(charset, "utf-8") && !strings.EqualFold(charset, "us-ascii") {
		var converted = convertCharset(decoded, charset)
		if converted != "" {
			return converted
		}
	}

	return string(decoded)
}

// convertCharset converte de um charset para UTF-8
func convertCharset(data []byte, charset string) string {
	// Tenta usar htmlindex primeiro
	var enc, err = htmlindex.Get(charset)
	if err == nil {
		var decoder = enc.NewDecoder()
		var result, err2 = decoder.Bytes(data)
		if err2 == nil {
			return string(result)
		}
	}

	// Fallback para charsets comuns
	charset = strings.ToLower(charset)
	switch {
	case strings.Contains(charset, "iso-8859-1"), strings.Contains(charset, "latin1"):
		var decoder = charmap.ISO8859_1.NewDecoder()
		var result, _ = decoder.Bytes(data)
		return string(result)
	case strings.Contains(charset, "iso-8859-15"), strings.Contains(charset, "latin9"):
		var decoder = charmap.ISO8859_15.NewDecoder()
		var result, _ = decoder.Bytes(data)
		return string(result)
	case strings.Contains(charset, "windows-1252"):
		var decoder = charmap.Windows1252.NewDecoder()
		var result, _ = decoder.Bytes(data)
		return string(result)
	}

	return ""
}

// htmlToText converte HTML para texto legível
func htmlToText(htmlContent string) string {
	var doc, err = html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return htmlContent
	}

	var buf bytes.Buffer
	var extractTextFromNode func(*html.Node)
	extractTextFromNode = func(n *html.Node) {
		// Ignora scripts, styles e comments
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "head", "noscript":
				return
			case "br":
				buf.WriteString("\n")
				return
			case "p", "div", "tr", "li", "h1", "h2", "h3", "h4", "h5", "h6":
				buf.WriteString("\n")
			case "td", "th":
				buf.WriteString("\t")
			}
		}

		if n.Type == html.TextNode {
			var text = strings.TrimSpace(n.Data)
			if text != "" {
				buf.WriteString(text)
				buf.WriteString(" ")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractTextFromNode(c)
		}

		// Adiciona quebra de linha após elementos de bloco
		if n.Type == html.ElementNode {
			switch n.Data {
			case "p", "div", "tr", "li", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote":
				buf.WriteString("\n")
			}
		}
	}

	extractTextFromNode(doc)

	// Limpa múltiplas linhas em branco
	var result = buf.String()
	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(result)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Primeiro verifica se há alerta overlay aberto
		if m.showAlert && len(m.alerts) > 0 {
			switch msg.String() {
			case "enter", "esc", "x", " ":
				m.showAlert = false
				m.alerts = []Alert{} // Limpa todos os alertas
				m.log("🧹 Alerta fechado")
				return m, nil
			case "ctrl+c", "q":
				return m, tea.Quit
			}
			return m, nil // Bloqueia outras teclas enquanto alerta está aberto
		}

		// Verifica overlay de Undo Send
		if m.showUndoOverlay && m.scheduledDraft != nil {
			switch msg.String() {
			case "enter":
				// Cancela o envio - volta para draft
				storage.CancelDraft(m.scheduledDraft.ID)
				m.log("🚫 Envio cancelado, draft salvo")
				m.aiResponse = infoStyle.Render("📝 Envio cancelado. Draft salvo.")
				m.showUndoOverlay = false
				m.scheduledDraft = nil
				return m, m.loadDrafts()
			case "esc":
				// Fecha overlay mas continua o envio
				m.showUndoOverlay = false
				return m, nil
			case "ctrl+c", "q":
				return m, tea.Quit
			}
			return m, nil // Bloqueia outras teclas enquanto overlay está aberto
		}

		// Verifica drafts panel
		if m.showDrafts {
			switch msg.String() {
			case "esc", "d":
				m.showDrafts = false
				return m, nil
			case "up", "k":
				if m.selectedDraft > 0 {
					m.selectedDraft--
				}
				return m, nil
			case "down", "j":
				if m.selectedDraft < len(m.drafts)-1 {
					m.selectedDraft++
				}
				return m, nil
			case "e":
				// Editar draft
				if len(m.drafts) > 0 {
					var draft = m.drafts[m.selectedDraft]
					m.showDrafts = false
					m.showCompose = true
					m.composeTo.SetValue(draft.ToAddresses)
					m.composeSubject.SetValue(draft.Subject)
					m.composeBodyText = draft.BodyText.String
					m.composeFocus = 2 // Foca no body
					m.editingDraftID = &draft.ID
					return m, textinput.Blink
				}
			case "s":
				// Enviar draft (agenda com delay)
				if len(m.drafts) > 0 {
					var draft = m.drafts[m.selectedDraft]
					if draft.Status == storage.DraftStatusDraft {
						var cfg, _ = config.Load()
						var delay = time.Duration(cfg.Compose.SendDelaySeconds) * time.Second
						storage.ScheduleDraft(draft.ID, time.Now().Add(delay))
						m.log("📤 Draft #%d agendado para envio", draft.ID)
						// Recarrega draft para obter dados atualizados
						var updatedDraft, _ = storage.GetDraftByID(draft.ID)
						if updatedDraft != nil {
							m.scheduledDraft = updatedDraft
							m.showUndoOverlay = true
						}
						return m, tea.Batch(m.loadDrafts(), scheduleDraftSend())
					}
				}
			case "x":
				// Cancelar/deletar draft (move para histórico permanente)
				if len(m.drafts) > 0 {
					var draft = m.drafts[m.selectedDraft]
					storage.ArchiveDraftPermanently(draft.ID, "deleted")
					m.log("🗑️ Draft #%d arquivado no histórico", draft.ID)
					if m.selectedDraft >= len(m.drafts)-1 && m.selectedDraft > 0 {
						m.selectedDraft--
					}
					return m, m.loadDrafts()
				}
			case "ctrl+c", "q":
				return m, tea.Quit
			}
			return m, nil
		}

		// Verifica batch filter mode (preview de operação em lote)
		if m.filterActive {
			switch msg.String() {
			case "y", "Y":
				// Confirma e executa a operação
				if m.pendingBatchOp != nil {
					m.log("✅ Operação confirmada: %s", m.pendingBatchOp.Description)
					return m, m.executeBatchOp()
				}
			case "n", "N", "esc":
				// Cancela a operação e restaura lista original
				if m.pendingBatchOp != nil {
					storage.CancelBatchOp(m.pendingBatchOp.ID)
					m.log("❌ Operação cancelada")
				}
				m.filterActive = false
				m.filterDescription = ""
				m.pendingBatchOp = nil
				m.emails = m.originalEmails
				m.originalEmails = nil
				m.selectedEmail = 0
				return m, nil
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.selectedEmail > 0 {
					m.selectedEmail--
				}
				return m, nil
			case "down", "j":
				if m.selectedEmail < len(m.emails)-1 {
					m.selectedEmail++
				}
				return m, nil
			}
			return m, nil // Bloqueia outras teclas no modo filtro
		}

		// Search mode
		if m.searchMode {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				// Sai do modo de busca e restaura lista original
				m.searchMode = false
				m.searchInput.Blur()
				m.searchInput.SetValue("")
				m.searchQuery = ""
				m.searchResults = nil
				m.emails = m.originalEmails
				m.originalEmails = nil
				m.selectedEmail = 0
				m.log("🔍 Busca cancelada")
				return m, nil
			case "enter":
				// Seleciona email atual e sai da busca mantendo resultados
				if len(m.emails) > 0 {
					m.searchMode = false
					m.searchInput.Blur()
					// Mantém os resultados da busca como lista atual
					m.originalEmails = nil
					m.log("🔍 Busca finalizada: %d resultados", len(m.emails))
				}
				return m, nil
			case "up", "k":
				if m.selectedEmail > 0 {
					m.selectedEmail--
				}
				return m, nil
			case "down", "j":
				if m.selectedEmail < len(m.emails)-1 {
					m.selectedEmail++
				}
				return m, nil
			}
			// Atualiza input e dispara busca com debounce
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			var newQuery = m.searchInput.Value()
			if newQuery != m.searchQuery {
				m.searchQuery = newQuery
				return m, tea.Batch(cmd, m.searchDebounce())
			}
			return m, cmd
		}

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
				m.editingDraftID = nil
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
			case "e":
				// Editar draft criado pelo AI (só quando input está vazio)
				if m.aiInput.Value() == "" && m.editingDraftID != nil && !m.aiLoading {
					var draft, err = storage.GetDraftByID(*m.editingDraftID)
					if err == nil && draft != nil {
						m.showAI = false
						m.showCompose = true
						m.composeTo.SetValue(draft.ToAddresses)
						m.composeSubject.SetValue(draft.Subject)
						m.composeBodyText = draft.BodyText.String
						m.composeFocus = 2
						return m, textinput.Blink
					}
				}
			case "d":
				// Ir para drafts panel (só quando input está vazio)
				if m.aiInput.Value() == "" && !m.aiLoading {
					m.showAI = false
					m.showDrafts = true
					m.selectedDraft = 0
					m.editingDraftID = nil
					return m, m.loadDrafts()
				}
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

		// Email viewer mode
		if m.showViewer {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc", "q":
				m.showViewer = false
				return m, nil
			case "h":
				// Abre no navegador e marca como lido
				m.showViewer = false
				var cmds []tea.Cmd
				cmds = append(cmds, m.openEmailHTML())
				if m.viewerEmail != nil && !m.viewerEmail.IsRead {
					cmds = append(cmds, m.markAsRead(m.viewerEmail.ID, m.viewerEmail.UID))
				}
				return m, tea.Batch(cmds...)
			}
			// Passa eventos de scroll para o viewport
			var cmd tea.Cmd
			m.viewerViewport, cmd = m.viewerViewport.Update(msg)
			return m, cmd
		}

		// Compose mode
		if m.showCompose {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.showCompose = false
				m.composeTo.Blur()
				m.composeSubject.Blur()
				return m, nil
			case "tab":
				// Cicla entre campos: To(0), Subject(1), Body(2), Classification(3)
				m.composeFocus = (m.composeFocus + 1) % 4
				m.composeTo.Blur()
				m.composeSubject.Blur()
				switch m.composeFocus {
				case 0:
					m.composeTo.Focus()
				case 1:
					m.composeSubject.Focus()
				}
				return m, nil
			case "left", "h":
				// Muda classificação para anterior
				if m.composeFocus == 3 {
					m.composeClassification--
					if m.composeClassification < 0 {
						m.composeClassification = len(smtp.Classifications) - 1
					}
					return m, nil
				}
			case "right", "l":
				// Muda classificação para próxima
				if m.composeFocus == 3 {
					m.composeClassification = (m.composeClassification + 1) % len(smtp.Classifications)
					return m, nil
				}
			case "ctrl+s":
				// Cria draft e agenda envio com delay
				if m.composeSending {
					return m, nil
				}
				var to = strings.TrimSpace(m.composeTo.Value())
				var subject = strings.TrimSpace(m.composeSubject.Value())
				if to == "" || subject == "" {
					return m, nil
				}
				m.composeSending = true
				return m, m.createScheduledDraft()
			}
			// Atualiza input focado
			var cmd tea.Cmd
			switch m.composeFocus {
			case 0:
				m.composeTo, cmd = m.composeTo.Update(msg)
			case 1:
				m.composeSubject, cmd = m.composeSubject.Update(msg)
			case 2:
				// Body - por enquanto usa texto simples
				if msg.String() == "enter" {
					m.composeBodyText += "\n"
				} else if msg.String() == "backspace" && len(m.composeBodyText) > 0 {
					m.composeBodyText = m.composeBodyText[:len(m.composeBodyText)-1]
				} else if len(msg.String()) == 1 {
					m.composeBodyText += msg.String()
				}
			}
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

		case "enter", "v":
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
			// Abre viewer do email
			if !m.showFolders && len(m.emails) > 0 {
				m.viewerEmail = &m.emails[m.selectedEmail]
				m.viewerLoading = true
				m.showViewer = true
				return m, m.loadEmailContent()
			}

		case "r":
			m.state = stateSyncing
			return m, m.syncEmails()

		case "a":
			// AI geral (sem contexto de email)
			m.showAI = true
			m.aiInput.Focus()
			m.aiScrollOffset = 0
			m.aiEmailContext = nil
			m.aiEmailBody = ""
			return m, textinput.Blink

		case "A":
			// AI com contexto do email selecionado
			if !m.showFolders && len(m.emails) > 0 {
				m.aiLoading = true
				m.aiResponse = statusStyle.Render("Carregando email para contexto...")
				return m, m.loadEmailForAI()
			}

		case "c":
			// Novo email
			m.showCompose = true
			m.composeTo.SetValue("")
			m.composeSubject.SetValue("")
			m.composeBodyText = ""
			m.composeFocus = 0
			m.composeTo.Focus()
			m.composeReplyTo = nil
			m.editingDraftID = nil
			return m, textinput.Blink

		case "d":
			// Abre drafts panel
			m.showDrafts = true
			m.selectedDraft = 0
			return m, m.loadDrafts()

		case "R":
			// Reply
			if !m.showFolders && len(m.emails) > 0 {
				var email = m.emails[m.selectedEmail]
				m.showCompose = true
				m.composeTo.SetValue(email.FromEmail)
				m.composeSubject.SetValue("Re: " + email.Subject)
				m.composeBodyText = ""
				m.composeFocus = 2 // Foca no body
				m.composeReplyTo = &email
				return m, nil
			}

		case "h":
			// Abre email em HTML no navegador e marca como lido
			if !m.showFolders && len(m.emails) > 0 {
				var email = m.emails[m.selectedEmail]
				var cmds []tea.Cmd
				cmds = append(cmds, m.openEmailHTML())
				if !email.IsRead {
					cmds = append(cmds, m.markAsRead(email.ID, email.UID))
				}
				return m, tea.Batch(cmds...)
			}

		case "e":
			// Archive email (Gmail style)
			if !m.showFolders && len(m.emails) > 0 {
				var email = m.emails[m.selectedEmail]
				var messageID = ""
				if email.MessageID.Valid {
					messageID = email.MessageID.String
				}
				m.log("📦 Arquivando email: %s", email.Subject)
				return m, m.archiveEmail(email.ID, email.UID, messageID)
			}

		case "x", "#":
			// Delete email (move to trash)
			if !m.showFolders && len(m.emails) > 0 {
				var email = m.emails[m.selectedEmail]
				var messageID = ""
				if email.MessageID.Valid {
					messageID = email.MessageID.String
				}
				m.log("🗑️ Deletando email: %s", email.Subject)
				return m, m.deleteEmail(email.ID, email.UID, messageID)
			}

		case "X":
			// Limpa alertas
			if len(m.alerts) > 0 {
				m.alerts = []Alert{}
				m.showAlert = false
				m.log("🧹 Alertas limpos")
			}

		case "/":
			// Ativa modo de busca
			if m.state == stateReady && !m.showFolders && !m.showViewer && !m.showCompose && !m.showDrafts && !m.showAI {
				m.searchMode = true
				m.searchInput.Focus()
				m.searchInput.SetValue("")
				m.searchQuery = ""
				m.originalEmails = m.emails
				m.selectedEmail = 0
				m.log("🔍 Modo de busca ativado")
				return m, textinput.Blink
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case dbInitMsg:
		m.log("📦 DB inicializado")
		// Cria/obtém a conta no DB
		var account, err = storage.GetOrCreateAccount(m.account.Email, m.account.Name)
		if err != nil {
			m.err = err
			m.state = stateError
			return m, nil
		}
		m.dbAccount = account
		m.log("👤 Conta: %s (ID: %d)", account.Email, account.ID)

		// Cria/obtém a pasta INBOX
		var folder, err2 = storage.GetOrCreateFolder(account.ID, "INBOX")
		if err2 != nil {
			m.err = err2
			m.state = stateError
			return m, nil
		}
		m.dbFolder = folder
		m.log("📁 Pasta INBOX (ID: %d)", folder.ID)

		// Carrega emails do cache primeiro, conecta em paralelo
		m.log("🔄 Carregando cache + conectando...")
		m.state = stateLoadingEmails
		return m, tea.Batch(m.loadEmailsFromDB(), m.connect())

	case connectedMsg:
		m.log("✅ IMAP conectado")
		m.client = msg.client
		m.retrying = false
		// Se já temos emails do cache, faz sync em background sem bloquear UI
		if m.state == stateReady {
			return m, m.loadFolders()
		}
		m.state = stateLoadingFolders
		return m, m.loadFolders()

	case foldersLoadedMsg:
		m.log("📂 %d pastas carregadas", len(msg.mailboxes))
		m.mailboxes = msg.mailboxes

		// Salva pastas no DB e encontra INBOX
		if m.dbAccount == nil {
			m.err = fmt.Errorf("conta não inicializada")
			m.state = stateError
			return m, nil
		}
		for i, mb := range m.mailboxes {
			var folder, err = storage.GetOrCreateFolder(m.dbAccount.ID, mb.Name)
			if err != nil || folder == nil {
				continue
			}
			storage.UpdateFolderStats(folder.ID, int(mb.Messages), int(mb.Unseen))

			if strings.EqualFold(mb.Name, "INBOX") {
				m.selectedBox = i
				m.currentBox = mb.Name
				m.dbFolder = folder
			}
		}

		// Sync em background - não muda state se já estamos ready
		var latestUID2, _ = storage.GetLatestUID(m.dbAccount.ID, m.dbFolder.ID)
		m.log("🔄 Iniciando sync... (lastUID=%d)", latestUID2)
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
		if msg.archived > 0 {
			m.log("✅ Sync: %d novos, %d removidos, %d arquivados permanentemente", msg.synced, msg.purged, msg.archived)
		} else {
			m.log("✅ Sync: %d novos, %d removidos (total servidor: %d)", msg.synced, msg.purged, msg.total)
		}
		// Recarrega emails do DB após sync
		if m.state != stateReady {
			m.state = stateLoadingEmails
		}
		return m, m.loadEmailsFromDB()

	case emailsLoadedMsg:
		m.log("📧 %d emails carregados do cache", len(msg.emails))
		// Não sobrescreve se filtro está ativo (evita race condition)
		if m.filterActive {
			m.log("⚠️ Filtro ativo, mantendo emails filtrados")
			// Atualiza originalEmails para quando sair do filtro
			m.originalEmails = msg.emails
		} else {
			m.emails = msg.emails
		}
		// Sempre vai para ready quando temos emails do cache
		m.state = stateReady

	case configSavedMsg:
		return m, nil

	case aiResponseMsg:
		m.aiLoading = false
		if msg.err != nil {
			m.aiResponse = errorStyle.Render("Erro: " + msg.err.Error())
		} else {
			// Se tinha contexto de email, cria draft automaticamente
			if m.aiEmailContext != nil && msg.response != "" {
				// Cria draft com a resposta do AI
				var draft = &storage.Draft{
					AccountID:        m.dbAccount.ID,
					ToAddresses:      m.aiEmailContext.FromEmail,
					Subject:          "Re: " + m.aiEmailContext.Subject,
					BodyText:         sql.NullString{String: msg.response, Valid: true},
					GenerationSource: "ai",
					AIPrompt:         sql.NullString{String: m.aiLastQuestion, Valid: true},
					Status:           storage.DraftStatusDraft,
				}
				if m.aiEmailContext.MessageID.Valid {
					draft.InReplyTo = sql.NullString{String: m.aiEmailContext.MessageID.String, Valid: true}
					draft.ReferenceIDs = sql.NullString{String: m.aiEmailContext.MessageID.String, Valid: true}
				}
				draft.ReplyToEmailID = sql.NullInt64{Int64: m.aiEmailContext.ID, Valid: true}

				var draftID, err = storage.CreateDraft(draft)
				if err != nil {
					m.aiResponse = errorStyle.Render("Erro ao criar draft: " + err.Error())
				} else {
					draft.ID = draftID
					m.log("📝 Draft AI criado #%d para %s", draftID, draft.ToAddresses)
					m.aiResponse = successStyle.Render(fmt.Sprintf(
						"📝 Draft criado!\n\nPara: %s\nAssunto: %s\n\n--- Resposta ---\n%s\n\n[d] Ver drafts  [e] Editar agora  [Esc] Fechar",
						draft.ToAddresses,
						draft.Subject,
						truncate(msg.response, 200)))
					// Guarda o draft ID para possível edição
					m.editingDraftID = &draftID
				}
				m.aiEmailContext = nil
				m.aiEmailBody = ""
				return m, m.loadDrafts()
			}
			m.aiResponse = msg.response
		}
		m.aiScrollOffset = 0
		// Recarrega emails (AI pode ter feito alterações) e verifica batch ops pendentes
		return m, tea.Batch(m.loadEmailsFromDB(), m.checkPendingBatchOps())

	case aiEmailContextMsg:
		m.aiLoading = false
		if msg.err != nil {
			m.aiResponse = errorStyle.Render("Erro: " + msg.err.Error())
			return m, nil
		}
		// Configura contexto e abre AI
		m.aiEmailContext = msg.email
		m.aiEmailBody = msg.content
		m.showAI = true
		m.aiInput.Focus()
		m.aiScrollOffset = 0
		// Mostra preview do contexto
		var preview = fmt.Sprintf("📧 Contexto: %s\nDe: %s\n\nDigite sua pergunta sobre este email...",
			truncate(msg.email.Subject, 50),
			msg.email.FromEmail)
		m.aiResponse = infoStyle.Render(preview)
		return m, textinput.Blink

	case htmlOpenedMsg:
		if msg.err != nil {
			// Mostra erro temporário no AI panel
			m.showAI = true
			m.aiResponse = errorStyle.Render("Erro ao abrir HTML: " + msg.err.Error())
		}
		return m, nil

	case emailSentMsg:
		m.composeSending = false
		if msg.err != nil {
			m.showAI = true
			m.aiResponse = errorStyle.Render(fmt.Sprintf("❌ Erro ao enviar para %s:\n%s", msg.to, msg.err.Error()))
		} else {
			// Marca como respondido se era um reply
			if m.composeReplyTo != nil {
				storage.MarkAsReplied(m.composeReplyTo.ID)
				// Atualiza na lista local
				for i := range m.emails {
					if m.emails[i].ID == m.composeReplyTo.ID {
						m.emails[i].IsReplied = true
						break
					}
				}
			}

			// Adiciona tracker para monitorar bounce
			var tracker = SentEmailTracker{
				MessageID:    msg.msgID,
				To:           msg.to,
				Subject:      m.composeSubject.Value(),
				SentAt:       time.Now(),
				MonitorUntil: time.Now().Add(5 * time.Minute), // Monitora por 5 minutos
			}
			m.sentEmails = append(m.sentEmails, tracker)
			m.log("📧 Monitorando bounce para %s por 5 min", msg.to)

			m.showCompose = false
			m.showAI = true
			// Mensagem detalhada para o usuário saber exatamente o que aconteceu
			var details string
			if msg.backend == "gmail_api" {
				details = fmt.Sprintf(`✅ Email enviado via Gmail API

📤 Para: %s
🆔 Message ID: %s

⏱️  Monitorando bounces por 5 minutos...
Se houver rejeição, você será alertado.`, msg.to, msg.msgID)
			} else {
				details = fmt.Sprintf(`✅ Email aceito pelo servidor SMTP

📤 Para: %s
🖥️  Servidor: %s:%d

⏱️  Monitorando bounces por 5 minutos...
Se houver rejeição, você será alertado.`, msg.to, msg.host, msg.port)
			}
			m.aiResponse = infoStyle.Render(details)

			// Inicia monitoramento de bounce
			return m, tea.Batch(scheduleBounceCheck(), m.syncEmails())
		}
		return m, nil

	case emailContentMsg:
		m.viewerLoading = false
		if msg.err != nil {
			m.showViewer = false
			m.showAI = true
			m.aiResponse = errorStyle.Render("Erro ao carregar email: " + msg.err.Error())
			return m, nil
		}
		// Configura viewport com o conteúdo
		m.viewerViewport = viewport.New(m.width-4, m.height-8)
		m.viewerViewport.SetContent(msg.content)

		// Marca como lido
		if m.viewerEmail != nil && !m.viewerEmail.IsRead {
			return m, m.markAsRead(m.viewerEmail.ID, m.viewerEmail.UID)
		}
		return m, nil

	case markReadMsg:
		// Atualiza na lista local
		for i := range m.emails {
			if m.emails[i].ID == msg.emailID {
				m.emails[i].IsRead = true
				break
			}
		}
		if m.viewerEmail != nil && m.viewerEmail.ID == msg.emailID {
			m.viewerEmail.IsRead = true
		}
		return m, nil

	case emailArchivedMsg:
		if msg.err != nil {
			m.log("❌ Erro ao arquivar: %v", msg.err)
			return m, nil
		}
		m.log("✓ Email arquivado")
		// Remove da lista local
		for i := range m.emails {
			if m.emails[i].ID == msg.emailID {
				m.emails = append(m.emails[:i], m.emails[i+1:]...)
				break
			}
		}
		// Ajusta seleção se necessário
		if m.selectedEmail >= len(m.emails) && m.selectedEmail > 0 {
			m.selectedEmail--
		}
		return m, nil

	case emailDeletedMsg:
		if msg.err != nil {
			m.log("❌ Erro ao deletar: %v", msg.err)
			return m, nil
		}
		m.log("✓ Email movido para lixeira")
		// Remove da lista local
		for i := range m.emails {
			if m.emails[i].ID == msg.emailID {
				m.emails = append(m.emails[:i], m.emails[i+1:]...)
				break
			}
		}
		// Ajusta seleção se necessário
		if m.selectedEmail >= len(m.emails) && m.selectedEmail > 0 {
			m.selectedEmail--
		}
		return m, nil

	case batchFilterAppliedMsg:
		if msg.err != nil {
			m.log("❌ Erro ao aplicar filtro: %v", msg.err)
			return m, nil
		}
		// Salva emails originais e aplica filtro
		m.originalEmails = m.emails
		m.emails = msg.emails
		m.filterActive = true
		m.filterDescription = msg.op.Description
		m.pendingBatchOp = msg.op
		m.selectedEmail = 0
		m.log("🔍 Filtro aplicado: %d emails. [y] confirmar, [n] cancelar", len(msg.emails))
		return m, nil

	case batchOpExecutedMsg:
		// Limpa filtro e restaura view
		m.filterActive = false
		m.filterDescription = ""
		m.pendingBatchOp = nil
		if msg.err != nil {
			m.log("❌ Erro na operação: %v", msg.err)
			m.emails = m.originalEmails
		} else {
			m.log("✅ Operação concluída: %d emails processados", msg.count)
			// Recarrega do banco
			return m, m.loadEmailsFromDB()
		}
		m.originalEmails = nil
		return m, nil

	case checkPendingBatchOpsMsg:
		// Verifica se há operações pendentes após resposta do AI
		m.log("🔎 Verificando batch ops pendentes...")
		if msg.err != nil {
			m.log("❌ Erro ao verificar batch ops: %v", msg.err)
			return m, nil
		}
		if msg.op != nil {
			// Há operação pendente, aplica filtro
			m.log("📋 Operação pendente detectada: %s (ID=%d, %d emails)", msg.op.Description, msg.op.ID, msg.op.EmailCount)
			m.showAI = false // Fecha AI panel para mostrar preview
			return m, m.applyBatchFilter(msg.op)
		}
		m.log("✓ Nenhuma operação pendente encontrada")
		return m, nil

	case bounceCheckTickMsg:
		// Verifica se ainda há emails para monitorar
		var now = time.Now()
		var hasActive = false
		var activeCount = 0
		for _, tracker := range m.sentEmails {
			if now.Before(tracker.MonitorUntil) {
				hasActive = true
				activeCount++
			}
		}

		m.log("⏱️ Bounce tick: %d trackers ativos", activeCount)

		if hasActive {
			// Sincroniza inbox e verifica bounces
			return m, tea.Batch(m.syncEmails(), m.checkForBounces(), scheduleBounceCheck())
		}
		m.log("⏱️ Monitoramento encerrado")
		return m, nil

	// === DRAFT HANDLERS ===

	case draftSendTickMsg:
		// Verifica se há drafts prontos para envio
		var readyDrafts, err = storage.GetScheduledDraftsReady()
		if err != nil {
			return m, scheduleDraftSend()
		}

		// Envia o primeiro draft pronto
		if len(readyDrafts) > 0 {
			var draft = readyDrafts[0]
			m.log("📤 Enviando draft #%d para %s", draft.ID, draft.ToAddresses)
			return m, tea.Batch(m.sendDraft(draft.ID), scheduleDraftSend())
		}

		// Verifica se ainda há drafts agendados (não prontos ainda)
		if m.dbAccount != nil {
			var pending, _ = storage.CountPendingDrafts(m.dbAccount.ID)
			if pending > 0 {
				return m, scheduleDraftSend()
			}
		}
		return m, nil

	case draftScheduledMsg:
		if msg.err != nil {
			m.aiResponse = errorStyle.Render("Erro ao agendar: " + msg.err.Error())
			return m, nil
		}

		m.showCompose = false
		m.composeSending = false
		m.scheduledDraft = msg.draft
		m.showUndoOverlay = true
		m.composeTo.SetValue("")
		m.composeSubject.SetValue("")
		m.composeBodyText = ""
		m.editingDraftID = nil

		// Inicia scheduler de envio se não estiver rodando
		return m, tea.Batch(m.loadDrafts(), scheduleDraftSend())

	case draftSentMsg:
		if msg.err != nil {
			m.log("❌ Erro ao enviar draft: %v", msg.err)
			m.aiResponse = errorStyle.Render("Erro no envio: " + msg.err.Error())
		} else {
			m.log("✅ Draft enviado via %s para %s", msg.backend, msg.to)
			var backendMsg = "SMTP"
			if msg.backend == "gmail_api" {
				backendMsg = "Gmail API"
			}
			m.aiResponse = successStyle.Render(fmt.Sprintf("📨 Email enviado via %s!\nPara: %s", backendMsg, msg.to))
		}
		// Remove do overlay se era o draft sendo exibido
		if m.scheduledDraft != nil && m.scheduledDraft.ID == msg.draftID {
			m.scheduledDraft = nil
			m.showUndoOverlay = false
		}
		return m, tea.Batch(m.loadDrafts(), m.syncEmails())

	case draftsLoadedMsg:
		if msg.err != nil {
			m.log("❌ Erro ao carregar drafts: %v", msg.err)
		} else {
			m.drafts = msg.drafts
			m.log("📝 Drafts carregados: %d (account_id=%d)", len(msg.drafts), msg.accountID)
		}
		return m, nil

	case bounceFoundMsg:
		m.log("🚨 BOUNCE detectado para %s!", msg.originalTo)

		// Cria alerta
		var alert = Alert{
			Type:      "bounce",
			Title:     "📬 Email Rejeitado!",
			Message:   fmt.Sprintf("Para: %s\nAssunto: %s\nRazão: %s", msg.originalTo, msg.originalSubject, msg.bounceReason),
			Timestamp: time.Now(),
			EmailTo:   msg.originalTo,
		}
		m.alerts = append(m.alerts, alert)
		m.showAlert = true

		// Remove o tracker desse email
		var newTrackers []SentEmailTracker
		for _, tracker := range m.sentEmails {
			if tracker.To != msg.originalTo {
				newTrackers = append(newTrackers, tracker)
			}
		}
		m.sentEmails = newTrackers

		// Mostra no AI panel também
		m.showAI = true
		m.aiResponse = errorStyle.Render(fmt.Sprintf(`🚨 EMAIL REJEITADO!

📤 Para: %s
📋 Assunto: %s

❌ Razão: %s

📧 Bounce de: %s
📋 Subject: %s

Verifique as configurações ou contate o administrador.`,
			msg.originalTo, msg.originalSubject, msg.bounceReason, msg.bounceFrom, msg.bounceSubject))

		return m, nil

	// === SEARCH HANDLERS ===

	case searchDebounceMsg:
		// Se a query mudou enquanto esperava, ignora
		if msg.query != m.searchQuery {
			return m, nil
		}
		// Dispara busca real
		if m.searchMode && m.dbAccount != nil {
			return m, m.performSearch(msg.query)
		}
		return m, nil

	case searchResultsMsg:
		if msg.err != nil {
			m.log("❌ Erro na busca: %v", msg.err)
			return m, nil
		}
		// Atualiza resultados se ainda em modo busca e query ainda é a mesma
		if m.searchMode && msg.query == m.searchQuery {
			if len(msg.results) > 0 {
				m.emails = msg.results
				m.selectedEmail = 0
			} else if msg.query == "" {
				// Se query vazia, restaura lista original
				m.emails = m.originalEmails
			} else {
				// Mostra lista vazia para indicar "sem resultados"
				m.emails = nil
			}
			m.log("🔍 Busca '%s': %d resultados", msg.query, len(msg.results))
		}
		return m, nil

	case errMsg:
		m.log("❌ Erro: %v", msg.err)
		m.err = msg.err
		if isAppPasswordError(msg.err) {
			m.state = stateNeedsAppPassword
			m.passwordInput.Focus()
			return m, textinput.Blink
		}
		m.state = stateError

	case debugLogMsg:
		m.log("%s", msg.msg)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
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
	var baseView string

	switch m.state {
	case stateInitDB:
		baseView = m.viewLoading("Inicializando banco de dados...")
	case stateConnecting:
		var msg = "Conectando ao servidor IMAP..."
		if m.retrying {
			msg = "Reconectando com nova senha..."
		}
		baseView = m.viewLoading(msg)
	case stateLoadingFolders:
		baseView = m.viewLoading("Carregando pastas...")
	case stateSyncing:
		var msg = fmt.Sprintf("Sincronizando %s...", m.currentBox)
		if m.syncStatus != "" {
			msg = m.syncStatus
		}
		// Se já temos emails no cache, mostra a quantidade
		if len(m.emails) > 0 {
			msg = fmt.Sprintf("Sincronizando %s (%d emails em cache)...", m.currentBox, len(m.emails))
		}
		baseView = m.viewLoading(msg)
	case stateLoadingEmails:
		baseView = m.viewLoading("Carregando emails do banco local...")
	case stateNeedsAppPassword:
		baseView = m.viewAppPasswordPrompt()
	case stateError:
		baseView = m.viewError()
	case stateReady:
		if m.showCompose {
			baseView = m.viewCompose()
		} else if m.showViewer {
			baseView = m.viewEmailViewer()
		} else {
			baseView = m.viewInbox()
		}
	}

	// Overlay de alerta se tiver bounce
	if m.showAlert && len(m.alerts) > 0 {
		return m.viewAlertOverlay(baseView)
	}

	// Overlay de Undo Send
	if m.showUndoOverlay && m.scheduledDraft != nil {
		return m.viewUndoSendOverlay(baseView)
	}

	// Panel de drafts
	if m.showDrafts {
		return m.viewDraftsPanel(baseView)
	}

	return baseView
}

func (m Model) viewLoading(msg string) string {
	var spinnerView = m.spinner.View()
	var content = fmt.Sprintf("%s\n\n%s %s",
		titleStyle.Render("miau 🐱"),
		spinnerView,
		subtitleStyle.Render(msg),
	)

	// Mostra progresso se disponível
	if m.syncedEmails > 0 || m.totalEmails > 0 {
		content += fmt.Sprintf("\n\n%s",
			infoStyle.Render(fmt.Sprintf("📧 %d emails sincronizados", m.syncedEmails)))
	}

	var box = boxStyle.Render(content)

	// Debug panel no loading
	if m.debugMode && m.width > 0 {
		var debugPanel = m.viewDebugPanel()
		var centered = lipgloss.Place(m.width-45, m.height, lipgloss.Center, lipgloss.Center, box)
		return lipgloss.JoinHorizontal(lipgloss.Top, centered, debugPanel)
	}

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

	// Indicador de monitoramento de bounce
	var monitorIndicator = ""
	if len(m.sentEmails) > 0 {
		monitorIndicator = infoStyle.Render(" ⏱️ ")
	}

	// Indicador de alertas
	var alertIndicator = ""
	var activeAlerts = 0
	for _, alert := range m.alerts {
		if !alert.Dismissed {
			activeAlerts++
		}
	}

	// Indicador de drafts pendentes
	var draftIndicator = ""
	if len(m.drafts) > 0 {
		draftIndicator = infoStyle.Render(fmt.Sprintf(" 📝%d ", len(m.drafts)))
	}
	if activeAlerts > 0 {
		alertIndicator = errorStyle.Render(fmt.Sprintf(" 🚨%d ", activeAlerts))
	}

	var header = headerStyle.Render(fmt.Sprintf(" miau 🐱  %s  [%s]%s ",
		m.account.Email,
		m.currentBox,
		stats,
	)) + draftIndicator + monitorIndicator + alertIndicator

	// Search banner (quando em modo de busca)
	var searchBanner = ""
	if m.searchMode {
		var searchBoxStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#FFD93D")).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Padding(0, 1)
		var resultInfo = ""
		if m.searchQuery != "" {
			if len(m.emails) > 0 {
				resultInfo = fmt.Sprintf("  (%d resultados)", len(m.emails))
			} else {
				resultInfo = "  (sem resultados)"
			}
		}
		searchBanner = searchBoxStyle.Render(fmt.Sprintf("🔍 Buscar: %s%s", m.searchInput.View(), resultInfo))
	}

	// Filter banner (quando em modo de preview de operação em lote)
	var filterBanner = ""
	if m.filterActive && m.pendingBatchOp != nil {
		var bannerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#4ECDC4")).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Padding(0, 1).
			Width(m.width)
		filterBanner = bannerStyle.Render(fmt.Sprintf("⚡ %s  |  y:confirmar  n:cancelar  ↑↓:navegar", m.filterDescription))
	}

	// Folders panel (se ativo)
	var foldersPanel string
	if m.showFolders {
		foldersPanel = m.renderFolders()
	}

	// Email list
	var emailList = m.renderEmailList()

	// Footer
	var footer string
	if m.searchMode {
		footer = subtitleStyle.Render(" ↑↓:navegar  Enter:selecionar  Esc:cancelar  /:buscar ")
	} else if m.filterActive {
		footer = subtitleStyle.Render(" y:CONFIRMAR operação  n/Esc:CANCELAR e voltar  ↑↓:navegar preview ")
	} else if m.showAI {
		var contextHint = ""
		if m.aiEmailContext != nil {
			contextHint = " [com email]"
		}
		footer = subtitleStyle.Render(fmt.Sprintf(" Enter:enviar  ↑↓:scroll  Esc:fechar%s ", contextHint))
	} else if activeAlerts > 0 {
		footer = subtitleStyle.Render(" ↑↓:navegar  Enter:ver  x:limpar alertas  c:novo  R:reply  a:AI  q:sair ")
	} else {
		footer = subtitleStyle.Render(" ↑↓:navegar  Enter:ver  c:novo  R:reply  d:drafts  Tab:pastas  /:buscar  a:AI  q:sair ")
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

	// Debug panel (lado direito)
	if m.debugMode {
		var debugPanel = m.viewDebugPanel()
		content = lipgloss.JoinHorizontal(lipgloss.Top, content, debugPanel)
	}

	var view string
	if searchBanner != "" {
		view = lipgloss.JoinVertical(lipgloss.Left,
			header,
			searchBanner,
			content,
			footer,
		)
	} else if filterBanner != "" {
		view = lipgloss.JoinVertical(lipgloss.Left,
			header,
			filterBanner,
			content,
			footer,
		)
	} else {
		view = lipgloss.JoinVertical(lipgloss.Left,
			header,
			content,
			footer,
		)
	}

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
		if m.searchMode && m.searchQuery != "" {
			return boxStyle.Render(subtitleStyle.Render(fmt.Sprintf("Nenhum email encontrado para '%s'\nTente outros termos ou pressione Esc para cancelar.", m.searchQuery)))
		}
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
	if m.debugMode {
		emailWidth -= 44 // largura do debug panel
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

	var content = strings.Join(lines, "\n")

	// Limita largura quando debug ativo
	if m.debugMode {
		return lipgloss.NewStyle().MaxWidth(emailWidth + 4).Render(content)
	}
	return content
}

func (m Model) formatEmailLine(email storage.EmailSummary, width int) string {
	var indicator = "●"
	if email.IsRead {
		indicator = " "
	}
	if email.IsStarred {
		indicator = "★"
	}
	if email.IsReplied {
		indicator = "↩"
	}

	// Em modo de busca, mostra indicador de match
	if m.searchMode && m.searchQuery != "" {
		indicator = "➤"
	}

	var from = email.FromName
	if from == "" {
		from = email.FromEmail
	}
	from = truncate(from, 18)

	// Calcula espaço disponível para subject
	// formato: " X from(18) │ subject │ dd/mm hh:mm "
	// fixo: 1 + 1 + 18 + 3 + 3 + 11 + 1 = 38
	var subjectWidth = width - 38
	if subjectWidth < 10 {
		subjectWidth = 10
	}
	var subject = truncate(email.Subject, subjectWidth)
	var date = email.Date.Format("02/01 15:04")

	// Pad subject para alinhar
	for len(subject) < subjectWidth {
		subject += " "
	}

	return fmt.Sprintf(" %s %-18s │ %s │ %s ", indicator, from, subject, date)
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
	// Desconta largura do debug panel se ativo
	if m.debugMode {
		width -= 44 // 40 (debug width) + 4 (border/padding)
	}
	if width < 40 {
		width = 40
	}

	// Input com indicador de contexto
	var inputLabel string
	if m.aiEmailContext != nil {
		inputLabel = infoStyle.Render("🤖 AI [📧]: ")
	} else {
		inputLabel = infoStyle.Render("🤖 AI: ")
	}
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

func (m Model) viewEmailViewer() string {
	var viewerBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF6B6B")).
		Padding(0, 1)

	// Header com info do email
	var header string
	if m.viewerEmail != nil {
		header = titleStyle.Render("miau 🐱") + " - " + subtitleStyle.Render(m.viewerEmail.Subject) + "\n"
		header += infoStyle.Render(fmt.Sprintf("De: %s <%s>", m.viewerEmail.FromName, m.viewerEmail.FromEmail)) + "\n"
		header += subtitleStyle.Render(m.viewerEmail.Date.Time.Format("02/01/2006 15:04"))
	}

	// Conteúdo
	var content string
	if m.viewerLoading {
		content = statusStyle.Render("Carregando email...")
	} else {
		content = m.viewerViewport.View()
	}

	// Footer
	var footer = subtitleStyle.Render(" ↑↓/PgUp/PgDn:scroll  h:abrir no navegador  q/Esc:voltar ")

	// Scroll info
	var scrollInfo = subtitleStyle.Render(fmt.Sprintf(" %d%% ", int(m.viewerViewport.ScrollPercent()*100)))

	var body = viewerBoxStyle.Width(m.width - 4).Height(m.height - 8).Render(content)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		body,
		lipgloss.JoinHorizontal(lipgloss.Left, footer, scrollInfo),
	)
}

func (m Model) viewDebugPanel() string {
	var debugBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF6B6B")).
		Padding(0, 1)

	var width = 40
	var height = m.height - 4
	if height < 10 {
		height = 10
	}

	// Header
	var header = errorStyle.Render("🐛 Debug")

	// Log lines
	var maxLines = height - 3
	var logs = m.debugLogs
	var start = len(logs) - maxLines - m.debugScroll
	if start < 0 {
		start = 0
	}
	var end = start + maxLines
	if end > len(logs) {
		end = len(logs)
	}

	var logContent string
	if len(logs) == 0 {
		logContent = subtitleStyle.Render("Aguardando eventos...")
	} else {
		var visibleLogs = logs[start:end]
		logContent = strings.Join(visibleLogs, "\n")
	}

	var content = lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		logContent,
	)

	return debugBoxStyle.Width(width).Height(height).Render(content)
}

func (m Model) viewCompose() string {
	var composeBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4ECDC4")).
		Padding(1, 2)

	// Header
	var title = "Novo Email"
	if m.composeReplyTo != nil {
		title = "Responder"
	}
	var header = titleStyle.Render("miau 🐱") + " - " + infoStyle.Render(title)

	// Indicadores de formato e assinatura
	var indicators string
	var cfg, _ = config.Load()
	var useHTML = cfg == nil || cfg.Compose.Format != "plain"
	if useHTML {
		indicators += infoStyle.Render(" [HTML] ")
	} else {
		indicators += subtitleStyle.Render(" [Plain] ")
	}
	if m.account.Signature != nil && m.account.Signature.Enabled {
		indicators += successStyle.Render(" [Assinatura ✓] ")
	} else {
		indicators += subtitleStyle.Render(" [Sem assinatura] ")
	}
	header += "  " + indicators

	// Campos
	var toLabel = "Para: "
	var subjectLabel = "Assunto: "
	var bodyLabel = "Mensagem:"
	var classLabel = "Classificação: "

	// Destaca campo focado
	if m.composeFocus == 0 {
		toLabel = folderSelectedStyle.Render("→ Para: ")
	}
	if m.composeFocus == 1 {
		subjectLabel = folderSelectedStyle.Render("→ Assunto: ")
	}
	if m.composeFocus == 2 {
		bodyLabel = folderSelectedStyle.Render("→ Mensagem:")
	}
	if m.composeFocus == 3 {
		classLabel = folderSelectedStyle.Render("→ Classificação: ")
	}

	// Renderiza classificações
	var classOptions string
	for i, c := range smtp.Classifications {
		if i == m.composeClassification {
			classOptions += selectedStyle.Render(" " + c + " ")
		} else {
			classOptions += subtitleStyle.Render(" " + c + " ")
		}
		if i < len(smtp.Classifications)-1 {
			classOptions += "│"
		}
	}

	var fields = lipgloss.JoinVertical(lipgloss.Left,
		toLabel+m.composeTo.View(),
		"",
		subjectLabel+m.composeSubject.View(),
		"",
		classLabel+classOptions,
		"",
		bodyLabel,
	)

	// Área do corpo
	var bodyLines = strings.Split(m.composeBodyText, "\n")
	var bodyDisplay string
	if len(bodyLines) > 10 {
		bodyDisplay = strings.Join(bodyLines[len(bodyLines)-10:], "\n")
	} else {
		bodyDisplay = m.composeBodyText
	}
	if m.composeFocus == 2 {
		bodyDisplay += "█" // cursor
	}

	// Mostra preview da assinatura se habilitada
	var sigPreview string
	if m.account.Signature != nil && m.account.Signature.Enabled {
		if useHTML && m.account.Signature.HTML != "" {
			// Preview simplificado da assinatura HTML
			sigPreview = subtitleStyle.Render("\n--\n[Assinatura será adicionada automaticamente]")
		} else if !useHTML && m.account.Signature.Text != "" {
			sigPreview = subtitleStyle.Render("\n--\n" + truncate(m.account.Signature.Text, 50))
		}
	}

	var bodyBox = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#666")).
		Padding(0, 1).
		Width(m.width - 12).
		Height(10).
		Render(bodyDisplay + sigPreview)

	// Footer
	var footer string
	if m.composeSending {
		footer = statusStyle.Render(" Enviando... ")
	} else {
		footer = subtitleStyle.Render(" Tab:próximo campo  ←→:classificação  Ctrl+S:enviar  Esc:cancelar ")
	}

	var content = lipgloss.JoinVertical(lipgloss.Left,
		fields,
		bodyBox,
	)

	var box = composeBoxStyle.Width(m.width - 6).Render(content)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		box,
		"",
		footer,
	)
}

// viewAlertOverlay renderiza um overlay de alerta sobre a tela base
func (m Model) viewAlertOverlay(baseView string) string {
	if len(m.alerts) == 0 {
		return baseView
	}

	var alert = m.alerts[len(m.alerts)-1] // Pega o último alerta

	// Estilo do overlay (modal centralizado)
	var overlayStyle = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#FF0000")).
		Background(lipgloss.Color("#1a0000")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(1, 2).
		Width(60)

	// Título do alerta
	var title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF6B6B")).
		Render(alert.Title)

	// Mensagem
	var message = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Render(alert.Message)

	// Timestamp
	var timestamp = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Italic(true).
		Render(alert.Timestamp.Format("15:04:05"))

	// Footer do modal
	var footer = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4ECDC4")).
		Render("\n\n[Enter/Esc/x para fechar]")

	var modalContent = lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		message,
		"",
		timestamp,
		footer,
	)

	var modal = overlayStyle.Render(modalContent)

	// Centraliza o modal na tela
	var centeredModal = lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
	)

	// Sobrepõe o modal sobre o conteúdo base (escurecido)
	// Como não podemos fazer transparência real, apenas mostramos o modal centralizado
	return centeredModal
}

// viewUndoSendOverlay renderiza overlay de "Undo Send"
func (m Model) viewUndoSendOverlay(baseView string) string {
	if m.scheduledDraft == nil {
		return baseView
	}

	var remaining = time.Until(m.scheduledDraft.ScheduledSendAt.Time)
	if remaining < 0 {
		remaining = 0
	}

	var overlayStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4ECDC4")).
		Background(lipgloss.Color("#1a1a1a")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(1, 2).
		Width(50)

	var titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4ECDC4"))

	var content = lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(fmt.Sprintf("📤 Enviando em %d segundos...", int(remaining.Seconds()))),
		"",
		fmt.Sprintf("Para: %s", m.scheduledDraft.ToAddresses),
		fmt.Sprintf("Assunto: %s", truncate(m.scheduledDraft.Subject, 35)),
		"",
		subtitleStyle.Render("[Enter] Cancelar envio  [Esc] Fechar"),
	)

	var modal = overlayStyle.Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

// viewDraftsPanel renderiza painel de drafts
func (m Model) viewDraftsPanel(baseView string) string {
	var panelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4ECDC4")).
		Background(lipgloss.Color("#1a1a1a")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(1, 2).
		Width(60)

	var headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4ECDC4"))

	var header = headerStyle.Render("📝 Drafts Pendentes")

	var lines []string
	if len(m.drafts) == 0 {
		lines = append(lines, statusStyle.Render("Nenhum draft pendente"))
	} else {
		for i, draft := range m.drafts {
			var status string
			switch draft.Status {
			case storage.DraftStatusDraft:
				status = "⏳"
			case storage.DraftStatusScheduled:
				var remaining = time.Until(draft.ScheduledSendAt.Time)
				if remaining > 0 {
					status = fmt.Sprintf("🕐%ds", int(remaining.Seconds()))
				} else {
					status = "🚀"
				}
			case storage.DraftStatusSending:
				status = "📤"
			default:
				status = "  "
			}

			var line = fmt.Sprintf(" %s │ %s │ %s",
				status,
				truncate(draft.ToAddresses, 20),
				truncate(draft.Subject, 25))

			if i == m.selectedDraft {
				lines = append(lines, selectedStyle.Render(line))
			} else {
				lines = append(lines, line)
			}
		}
	}

	var footer = subtitleStyle.Render(" ↑↓:navegar  e:editar  s:enviar  x:deletar  Esc:voltar ")

	var content = lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		strings.Join(lines, "\n"),
		"",
		footer,
	)

	var modal = panelStyle.Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}
