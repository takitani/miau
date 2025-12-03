package inbox

import (
	"bytes"
	"database/sql"
	"encoding/base64"
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
	"github.com/opik/miau/internal/config"
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
	showAI          bool
	aiInput         textinput.Model
	aiResponse      string
	aiLastQuestion  string
	aiLoading       bool
	aiScrollOffset  int
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
	synced int
	total  int
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

type emailSentMsg struct {
	err    error
	host   string
	port   int
	to     string
	msgID  string
}

type markReadMsg struct {
	emailID int64
	uid     uint32
}

type debugLogMsg struct {
	msg string
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

		// Atualiza stats da pasta
		var total, unread, _ = storage.CountEmails(m.dbAccount.ID, m.dbFolder.ID)
		storage.UpdateFolderStats(m.dbFolder.ID, total, unread)

		// Retorna total da caixa para mostrar na UI
		var totalInBox uint32
		if selectData != nil {
			totalInBox = selectData.NumMessages
		}

		return syncDoneMsg{synced: len(emails), total: int(totalInBox)}
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

		var client = smtp.NewClient(m.account)
		var email = &smtp.Email{
			To:             []string{to},
			Subject:        subject,
			Body:           emailBody,
			Classification: smtp.Classifications[m.composeClassification],
			IsHTML:         useHTML,
		}

		// Se for reply, adiciona headers de threading
		if m.composeReplyTo != nil && m.composeReplyTo.MessageID.Valid {
			var originalMsgID = m.composeReplyTo.MessageID.String
			email.InReplyTo = originalMsgID
			email.References = originalMsgID
		}

		var result, err = client.Send(email)
		if err != nil {
			return emailSentMsg{err: err, to: to}
		}

		return emailSentMsg{
			host:  result.Host,
			port:  result.Port,
			to:    to,
			msgID: result.MessageID,
		}
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
				// Envia email
				if m.composeSending {
					return m, nil
				}
				var to = strings.TrimSpace(m.composeTo.Value())
				var subject = strings.TrimSpace(m.composeSubject.Value())
				if to == "" || subject == "" {
					return m, nil
				}
				m.composeSending = true
				return m, m.sendEmail()
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
			m.showAI = true
			m.aiInput.Focus()
			m.aiScrollOffset = 0
			return m, textinput.Blink

		case "c":
			// Novo email
			m.showCompose = true
			m.composeTo.SetValue("")
			m.composeSubject.SetValue("")
			m.composeBodyText = ""
			m.composeFocus = 0
			m.composeTo.Focus()
			m.composeReplyTo = nil
			return m, textinput.Blink

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
		m.log("🔄 Iniciando sync...")
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
		m.log("✅ Sync completo: %d emails (total servidor: %d)", msg.synced, msg.total)
		// Recarrega emails do DB após sync
		if m.state != stateReady {
			m.state = stateLoadingEmails
		}
		return m, m.loadEmailsFromDB()

	case emailsLoadedMsg:
		m.log("📧 %d emails carregados do cache", len(msg.emails))
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
			m.showCompose = false
			m.showAI = true
			// Mensagem detalhada para o usuário saber exatamente o que aconteceu
			var details = fmt.Sprintf(`✅ Email aceito pelo servidor SMTP

📤 Para: %s
🖥️  Servidor: %s:%d

⚠️  IMPORTANTE: O servidor aceitou a mensagem.
Se não chegar ao destinatário, verifique:
- Pasta de spam do destinatário
- Se o email está correto
- Logs do servidor (Google Workspace Admin)`, msg.to, msg.host, msg.port)
			m.aiResponse = infoStyle.Render(details)
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
		// Se já temos emails no cache, mostra a quantidade
		if len(m.emails) > 0 {
			msg = fmt.Sprintf("Sincronizando %s (%d emails em cache)...", m.currentBox, len(m.emails))
		}
		return m.viewLoading(msg)
	case stateLoadingEmails:
		return m.viewLoading("Carregando emails do banco local...")
	case stateNeedsAppPassword:
		return m.viewAppPasswordPrompt()
	case stateError:
		return m.viewError()
	case stateReady:
		if m.showCompose {
			return m.viewCompose()
		}
		if m.showViewer {
			return m.viewEmailViewer()
		}
		return m.viewInbox()
	}
	return ""
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
		footer = subtitleStyle.Render(" ↑↓:navegar  Enter:ver  c:novo  R:reply  Tab:pastas  a:AI  q:sair ")
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
