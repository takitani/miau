package inbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/opik/miau/internal/auth"
	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/gmail"
	"github.com/opik/miau/internal/image"
	"github.com/opik/miau/internal/imap"
	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/services"
	"github.com/opik/miau/internal/smtp"
	"github.com/opik/miau/internal/storage"
)

// New creates a new inbox Model
// app is optional - if nil, attachments will be loaded via direct IMAP (legacy mode)
func New(account *config.Account, debug bool, app ...ports.App) Model {
	var input = textinput.New()
	input.Placeholder = "xxxx xxxx xxxx xxxx"
	input.CharLimit = 20
	input.Width = 25
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '•'

	var aiInput = textinput.New()
	aiInput.Placeholder = "Pergunte algo ou digite / para comandos..."
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

	// Detect image rendering capabilities
	var imgCaps = image.DetectCapabilities()

	// Get app if provided (for centralized services)
	var application ports.App
	if len(app) > 0 {
		application = app[0]
	}

	return Model{
		account:           account,
		state:             stateInitDB,
		currentBox:        "INBOX",
		showFolders:       false,
		foldersWidth:      25, // Default folders panel width
		passwordInput:     input,
		aiInput:           aiInput,
		spinner:           s,
		composeTo:         composeTo,
		composeSubject:    composeSubject,
		searchInput:       searchInput,
		debugMode:         debug,
		debugLogs:         debugLogs,
		imageCapabilities: &imgCaps,
		imageAttachments:  []Attachment{},
		app:               application,
		selectedEmails:    make(map[int64]bool), // Initialize multi-selection map
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

		// Registra início do sync
		var syncID, _ = storage.LogSyncStart(m.dbAccount.ID, m.dbFolder.ID)

		// Seleciona a mailbox
		var selectData, err = m.client.SelectMailbox(m.currentBox)
		if err != nil {
			storage.LogSyncComplete(syncID, 0, 0, err)
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
			storage.LogSyncComplete(syncID, 0, 0, err2)
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
			_, _, _ = storage.UpsertEmail(dbEmail)
		}

		// Detecta emails deletados no servidor
		var serverUIDs, errUIDs = m.client.GetAllUIDs()
		if errUIDs != nil {
			serverUIDs = nil
		}
		var purged, _ = storage.PurgeDeletedFromServer(m.dbAccount.ID, m.dbFolder.ID, serverUIDs)

		// Move emails deletados há mais de 30 dias para arquivo permanente
		var archived, _ = storage.PurgeToArchive(m.dbAccount.ID, 30)

		// Atualiza stats da pasta
		var total, unread, _ = storage.CountEmails(m.dbAccount.ID, m.dbFolder.ID)
		storage.UpdateFolderStats(m.dbFolder.ID, total, unread)

		// Conta novos emails desde o último sync (baseado em created_at no DB)
		var newCount, _ = storage.CountNewEmailsSinceLastSync(m.dbAccount.ID, m.dbFolder.ID)

		// Registra conclusão do sync
		storage.LogSyncComplete(syncID, newCount, purged, nil)

		// Retorna total da caixa para mostrar na UI
		var totalInBox uint32
		if selectData != nil {
			totalInBox = selectData.NumMessages
		}

		return syncDoneMsg{synced: newCount, total: int(totalInBox), purged: purged, archived: archived}
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

// === SETTINGS & INDEXER COMMANDS ===

func (m Model) loadIndexState() tea.Cmd {
	var accountID = m.dbAccount.ID
	return func() tea.Msg {
		var state, err = storage.GetOrCreateIndexState(accountID)
		if err != nil {
			return indexStateLoadedMsg{err: err}
		}
		// Atualiza contagem de emails a indexar
		var toIndex, _ = storage.CountEmailsToIndex(accountID)
		var indexed, _ = storage.CountIndexedEmails(accountID)
		state.TotalEmails = toIndex + indexed
		state.IndexedEmails = indexed
		return indexStateLoadedMsg{state: state}
	}
}

func (m Model) loadSettingsFolders() tea.Cmd {
	var accountID = m.dbAccount.ID
	var mailboxes = m.mailboxes
	return func() tea.Msg {
		// Get configured folders from app_settings
		var allSettings, _ = storage.GetAllSettings(accountID)
		var syncFolders []string
		if val, ok := allSettings["sync_folders"]; ok && val != "" {
			json.Unmarshal([]byte(val), &syncFolders)
		}
		if len(syncFolders) == 0 {
			// Default to essential folders
			syncFolders = []string{"INBOX", "[Gmail]/Sent Mail", "[Gmail]/Trash"}
		}

		// Build folder list with selection state
		var folders []SettingsFolder
		for _, mb := range mailboxes {
			var selected = false
			for _, sf := range syncFolders {
				if sf == mb.Name {
					selected = true
					break
				}
			}
			folders = append(folders, SettingsFolder{
				Name:     mb.Name,
				Selected: selected,
			})
		}
		return settingsFoldersLoadedMsg{folders: folders}
	}
}

func (m Model) saveSettingsFolders() tea.Cmd {
	var accountID = m.dbAccount.ID
	var folders = m.settingsFolders
	return func() tea.Msg {
		// Collect selected folders
		var selected []string
		for _, f := range folders {
			if f.Selected {
				selected = append(selected, f.Name)
			}
		}
		// Save to app_settings
		var data, _ = json.Marshal(selected)
		var err = storage.SetSetting(accountID, "sync_folders", string(data))
		return settingsSavedMsg{err: err}
	}
}

func (m Model) loadAnalytics() tea.Cmd {
	var accountID = m.dbAccount.ID
	var period = m.analyticsPeriod
	return func() tea.Msg {
		// Converte período para dias
		var sinceDays int
		switch period {
		case "7d":
			sinceDays = 7
		case "30d":
			sinceDays = 30
		case "90d":
			sinceDays = 90
		case "all":
			sinceDays = 0
		default:
			sinceDays = 30
		}

		var data = &AnalyticsData{}
		var err error

		// Carrega overview
		data.Overview, err = storage.GetAnalyticsOverview(accountID)
		if err != nil {
			return analyticsLoadedMsg{err: err}
		}

		// Carrega top senders
		data.TopSenders, _ = storage.GetTopSenders(accountID, 10, sinceDays)

		// Carrega distribuição por hora
		data.Hourly, _ = storage.GetEmailCountByHour(accountID, sinceDays)

		// Carrega distribuição por dia da semana
		data.Weekday, _ = storage.GetEmailCountByWeekday(accountID, sinceDays)

		// Carrega estatísticas de resposta
		data.ResponseTime, _ = storage.GetResponseStats(accountID)

		return analyticsLoadedMsg{data: data}
	}
}

func (m Model) handleSettingsAction() tea.Cmd {
	if m.indexState == nil || m.dbAccount == nil {
		return nil
	}

	switch m.settingsSelection {
	case 0: // Iniciar/Parar indexação
		if m.indexState.Status == storage.IndexStatusRunning {
			// Pausar
			storage.PauseIndexer(m.dbAccount.ID)
			m.indexState.Status = storage.IndexStatusPaused
			m.indexerRunning = false
			m.log("⏸️ Indexador pausado")
		} else if m.indexState.Status == storage.IndexStatusPaused {
			// Retomar
			storage.ResumeIndexer(m.dbAccount.ID)
			m.indexState.Status = storage.IndexStatusRunning
			m.indexerRunning = true
			m.log("▶️ Indexador retomado")
			return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
				return indexerTickMsg{}
			})
		} else {
			// Iniciar
			var toIndex, _ = storage.CountEmailsToIndex(m.dbAccount.ID)
			if toIndex == 0 {
				m.log("✅ Todos os emails já foram indexados!")
				return nil
			}
			storage.StartIndexer(m.dbAccount.ID, toIndex+m.indexState.IndexedEmails)
			m.indexState.Status = storage.IndexStatusRunning
			m.indexState.TotalEmails = toIndex + m.indexState.IndexedEmails
			m.indexerRunning = true
			m.log("▶️ Indexador iniciado: %d emails para processar", toIndex)
			return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
				return indexerTickMsg{}
			})
		}

	case 1: // Velocidade (usa +/- para ajustar)
		// Nada aqui, ação tratada pelo +/-

	case 2: // Cancelar indexação
		if m.indexState.Status == storage.IndexStatusRunning || m.indexState.Status == storage.IndexStatusPaused {
			storage.UpdateIndexState(m.dbAccount.ID, storage.IndexStatusIdle, m.indexState.IndexedEmails, m.indexState.LastIndexedUID, "")
			m.indexState.Status = storage.IndexStatusIdle
			m.indexerRunning = false
			m.log("🛑 Indexação cancelada")
		}

	case 3: // Fechar menu
		m.showSettings = false

	case 4: // Sobre
		// Nada aqui, apenas info
	}

	return nil
}

func (m Model) indexNextBatch() tea.Cmd {
	if m.client == nil || m.dbAccount == nil {
		return nil
	}

	var accountID = m.dbAccount.ID
	var client = m.client
	var currentBox = m.currentBox

	return func() tea.Msg {
		// Busca emails para indexar (em lote pequeno para não travar)
		var emails, err = storage.GetEmailsToIndex(accountID, 5)
		if err != nil {
			return indexBatchDoneMsg{err: err}
		}

		if len(emails) == 0 {
			return indexBatchDoneMsg{indexed: 0}
		}

		// Seleciona mailbox
		if _, err := client.SelectMailbox(currentBox); err != nil {
			return indexBatchDoneMsg{err: fmt.Errorf("erro ao selecionar mailbox: %w", err)}
		}

		var indexed = 0
		var lastUID int64 = 0

		for _, email := range emails {
			// Busca corpo do email
			var rawData, err = client.FetchEmailRaw(email.UID)
			if err != nil {
				// Marca como indexado mesmo com erro para não travar
				storage.MarkEmailIndexed(email.ID, "")
				continue
			}

			// Extrai texto
			var textContent = extractText(rawData)
			if textContent == "" {
				var htmlContent = extractHTML(rawData)
				if htmlContent != "" {
					textContent = htmlToText(htmlContent)
				}
			}

			// Salva
			if err := storage.MarkEmailIndexed(email.ID, textContent); err != nil {
				continue
			}

			indexed++
			lastUID = int64(email.UID)
		}

		return indexBatchDoneMsg{indexed: indexed, lastUID: lastUID}
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

// runQuickCommand executes a quick command via the AI service
func (m Model) runQuickCommand(cmd *ports.QuickCommand) tea.Cmd {
	var emailContext = m.aiEmailContext

	return func() tea.Msg {
		if m.app == nil {
			return aiResponseMsg{err: fmt.Errorf("application not initialized")}
		}

		var aiService = m.app.AI()
		if aiService == nil {
			return aiResponseMsg{err: fmt.Errorf("AI service not available")}
		}

		// Get email ID from context if available
		var emailID int64
		if emailContext != nil {
			emailID = emailContext.ID
		}

		var response, err = aiService.ExecuteQuickCommand(context.Background(), cmd, emailID)
		if err != nil {
			return aiResponseMsg{err: err}
		}
		return aiResponseMsg{response: response}
	}
}

// summarizeEmail summarizes the selected email using the AI service
func (m Model) summarizeEmail(emailID int64) tea.Cmd {
	return func() tea.Msg {
		if m.app == nil {
			return aiResponseMsg{err: fmt.Errorf("application not initialized")}
		}

		var aiService = m.app.AI()
		if aiService == nil {
			return aiResponseMsg{err: fmt.Errorf("AI service not available")}
		}

		var summary, err = aiService.Summarize(context.Background(), emailID)
		if err != nil {
			return aiResponseMsg{err: err}
		}
		return aiResponseMsg{response: summary}
	}
}

// summarizeThread summarizes the entire thread for the selected email using the AI service
func (m Model) summarizeThread(emailID int64) tea.Cmd {
	return func() tea.Msg {
		if m.app == nil {
			return aiResponseMsg{err: fmt.Errorf("application not initialized")}
		}

		var aiService = m.app.AI()
		if aiService == nil {
			return aiResponseMsg{err: fmt.Errorf("AI service not available")}
		}

		var summary, err = aiService.SummarizeThread(context.Background(), emailID)
		if err != nil {
			return aiResponseMsg{err: err}
		}
		return aiResponseMsg{response: summary}
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
	var tokenPath = auth.GetTokenPath(config.GetConfigPath(), m.account.Email)
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
						_, _, _ = storage.UpsertEmail(dbEmail)
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
		"classificação":            "Requer classificação de email",
		"classification":           "Requires email classification",
		"spam":                     "Marcado como spam",
		"rejected":                 "Rejeitado pelo servidor",
		"user unknown":             "Usuário desconhecido",
		"mailbox full":             "Caixa de correio cheia",
		"quota exceeded":           "Cota excedida",
		"does not exist":           "Endereço não existe",
		"address rejected":         "Endereço rejeitado",
		"policy":                   "Violação de política",
		"blocked":                  "Bloqueado",
		"blacklist":                "Na lista negra",
		"administrador":            "Bloqueado pelo administrador",
		"administrator":            "Blocked by administrator",
		"enterprise administrator": "Bloqueado pela política corporativa",
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

// scheduleAutoRefresh agenda o próximo auto-refresh
func scheduleAutoRefresh() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return autoRefreshTickMsg{}
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

		// 2. Arquiva no servidor (Gmail API se OAuth2, senão IMAP)
		var serverErr error

		// Gmail API se OAuth2 configurado (mais confiável, funciona com DLP)
		if m.account.AuthType == config.AuthTypeOAuth2 {
			var tokenPath = auth.GetTokenPath(config.GetConfigPath(), m.account.Email)
			var oauthCfg = auth.GetOAuth2Config(m.account.OAuth2.ClientID, m.account.OAuth2.ClientSecret)
			if token, err := auth.GetValidToken(oauthCfg, tokenPath); err == nil {
				var gmailClient = gmail.NewClient(token, oauthCfg, m.account.Email)
				if gmailMsgID, err := gmailClient.GetMessageIDByRFC822MsgID(messageID); err == nil {
					serverErr = gmailClient.ArchiveMessage(gmailMsgID)
				} else {
					serverErr = err
				}
			} else {
				serverErr = err
			}
		}

		// Fallback para IMAP se Gmail API não disponível ou falhou
		if serverErr != nil || m.account.AuthType != config.AuthTypeOAuth2 {
			if m.client != nil {
				serverErr = m.client.ArchiveEmail(uid)
			}
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

		// 2. Move para lixeira no servidor (Gmail API se OAuth2, senão IMAP)
		var serverErr error

		// Gmail API se OAuth2 configurado (mais confiável, funciona com DLP)
		if m.account.AuthType == config.AuthTypeOAuth2 {
			var tokenPath = auth.GetTokenPath(config.GetConfigPath(), m.account.Email)
			var oauthCfg = auth.GetOAuth2Config(m.account.OAuth2.ClientID, m.account.OAuth2.ClientSecret)
			if token, err := auth.GetValidToken(oauthCfg, tokenPath); err == nil {
				var gmailClient = gmail.NewClient(token, oauthCfg, m.account.Email)
				if gmailMsgID, err := gmailClient.GetMessageIDByRFC822MsgID(messageID); err == nil {
					serverErr = gmailClient.TrashMessage(gmailMsgID)
				} else {
					serverErr = err
				}
			} else {
				serverErr = err
			}
		}

		// Fallback para IMAP se Gmail API não disponível ou falhou
		if serverErr != nil || m.account.AuthType != config.AuthTypeOAuth2 {
			if m.client != nil {
				var trashFolder = m.client.GetTrashFolder()
				serverErr = m.client.TrashEmail(uid, trashFolder)
			}
		}

		// Retorna sucesso mesmo se servidor falhou (email já marcado local)
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
			var tokenPath = auth.GetTokenPath(config.GetConfigPath(), m.account.Email)
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

// loadImageAttachments extrai imagens do email selecionado
func (m Model) loadImageAttachments() tea.Cmd {
	// Captura valores antes de retornar a closure
	var uid uint32
	if m.viewerEmail != nil {
		uid = m.viewerEmail.UID
	} else if len(m.emails) > 0 && m.selectedEmail < len(m.emails) {
		uid = m.emails[m.selectedEmail].UID
	} else {
		return func() tea.Msg {
			return imageAttachmentsMsg{err: fmt.Errorf("nenhum email selecionado")}
		}
	}

	if m.client == nil {
		return func() tea.Msg {
			return imageAttachmentsMsg{err: fmt.Errorf("não conectado ao servidor")}
		}
	}

	var client = m.client
	var currentBox = m.currentBox

	return func() tea.Msg {
		// Seleciona a mailbox antes de buscar
		if _, err := client.SelectMailbox(currentBox); err != nil {
			return imageAttachmentsMsg{err: fmt.Errorf("erro ao selecionar pasta: %w", err)}
		}

		var rawData, err = client.FetchEmailRaw(uid)
		if err != nil {
			return imageAttachmentsMsg{err: err}
		}

		var attachments = extractAttachments(rawData)

		// Filtra apenas imagens
		var images []Attachment
		for _, att := range attachments {
			if strings.HasPrefix(att.ContentType, "image/") {
				images = append(images, att)
			}
		}

		return imageAttachmentsMsg{attachments: images}
	}
}

// loadAllAttachments extrai todos os anexos do email selecionado
func (m Model) loadAllAttachments() tea.Cmd {
	// Get UID from current email
	var uid uint32
	if m.viewerEmail != nil {
		uid = m.viewerEmail.UID
	} else if len(m.emails) > 0 && m.selectedEmail < len(m.emails) {
		uid = m.emails[m.selectedEmail].UID
	} else {
		return func() tea.Msg {
			return allAttachmentsMsg{err: fmt.Errorf("nenhum email selecionado")}
		}
	}

	if m.client == nil {
		return func() tea.Msg {
			return allAttachmentsMsg{err: fmt.Errorf("não conectado ao servidor IMAP")}
		}
	}

	var client = m.client
	var currentBox = m.currentBox

	return func() tea.Msg {
		// Seleciona a mailbox antes de buscar
		if _, err := client.SelectMailbox(currentBox); err != nil {
			return allAttachmentsMsg{err: fmt.Errorf("erro ao selecionar pasta: %w", err)}
		}

		var rawData, err = client.FetchEmailRaw(uid)
		if err != nil {
			return allAttachmentsMsg{err: err}
		}

		var attachments = extractAllAttachments(rawData)
		return allAttachmentsMsg{attachments: attachments}
	}
}

// saveCurrentAttachment salva o anexo selecionado no disco
func (m Model) saveCurrentAttachment() tea.Cmd {
	if len(m.viewerAttachments) == 0 || m.selectedAttachment >= len(m.viewerAttachments) {
		return nil
	}

	var att = m.viewerAttachments[m.selectedAttachment]

	return func() tea.Msg {
		// Determina o diretório de downloads
		var homeDir, _ = os.UserHomeDir()
		var downloadDir = filepath.Join(homeDir, "Downloads")

		// Cria o diretório se não existir
		os.MkdirAll(downloadDir, 0755)

		// Gera um nome único para evitar sobrescrita
		var filename = att.Filename
		var fullPath = filepath.Join(downloadDir, filename)

		// Se arquivo já existe, adiciona sufixo numérico
		var counter = 1
		for {
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				break
			}
			var ext = filepath.Ext(filename)
			var base = strings.TrimSuffix(filename, ext)
			fullPath = filepath.Join(downloadDir, fmt.Sprintf("%s_%d%s", base, counter, ext))
			counter++
		}

		// Salva o arquivo
		if err := os.WriteFile(fullPath, att.Data, 0644); err != nil {
			return attachmentSavedMsg{err: err}
		}

		return attachmentSavedMsg{path: fullPath}
	}
}

// renderCurrentImage renderiza a imagem atual usando chafa/viu
func (m Model) renderCurrentImage() tea.Cmd {
	if len(m.imageAttachments) == 0 || m.selectedImage >= len(m.imageAttachments) {
		return nil
	}

	var caps = m.imageCapabilities
	var att = m.imageAttachments[m.selectedImage]

	// Captura dimensões do terminal (overlay: -6, padding: -4, border: -2)
	var width = m.width - 14
	var height = m.height - 14 // header + footer + info + padding
	if width < 40 {
		width = 40
	}
	if height < 10 {
		height = 10
	}

	return func() tea.Msg {
		var opts = image.RenderOptions{
			Width:  width,
			Height: height,
			Data:   att.Data,
		}

		var output, err = image.Render(*caps, opts)
		if err != nil {
			return imageRenderedMsg{err: err}
		}

		return imageRenderedMsg{output: output}
	}
}

// openImageExternal abre a imagem no viewer externo do sistema
func (m Model) openImageExternal() tea.Cmd {
	if len(m.imageAttachments) == 0 || m.selectedImage >= len(m.imageAttachments) {
		return nil
	}

	var att = m.imageAttachments[m.selectedImage]

	return func() tea.Msg {
		var tmpFile = filepath.Join(os.TempDir(), "miau-image-"+att.Filename)
		if err := os.WriteFile(tmpFile, att.Data, 0600); err != nil {
			return imageSavedMsg{err: err}
		}

		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "linux":
			cmd = exec.Command("xdg-open", tmpFile)
		case "darwin":
			cmd = exec.Command("open", tmpFile)
		case "windows":
			cmd = exec.Command("cmd", "/c", "start", tmpFile)
		default:
			return imageSavedMsg{err: fmt.Errorf("sistema operacional não suportado")}
		}

		cmd.Start()
		return nil
	}
}

// saveImage salva a imagem atual na pasta Downloads
func (m Model) saveImage() tea.Cmd {
	if len(m.imageAttachments) == 0 || m.selectedImage >= len(m.imageAttachments) {
		return nil
	}

	var att = m.imageAttachments[m.selectedImage]

	return func() tea.Msg {
		var homeDir, err = os.UserHomeDir()
		if err != nil {
			return imageSavedMsg{err: err}
		}

		var downloadDir = filepath.Join(homeDir, "Downloads")
		// Cria o diretório se não existir
		if err := os.MkdirAll(downloadDir, 0755); err != nil {
			return imageSavedMsg{err: err}
		}

		var savePath = filepath.Join(downloadDir, att.Filename)

		// Evita sobrescrever arquivo existente
		if _, err := os.Stat(savePath); err == nil {
			var ext = filepath.Ext(att.Filename)
			var base = strings.TrimSuffix(att.Filename, ext)
			savePath = filepath.Join(downloadDir, fmt.Sprintf("%s_%d%s", base, time.Now().Unix(), ext))
		}

		if err := os.WriteFile(savePath, att.Data, 0644); err != nil {
			return imageSavedMsg{err: err}
		}

		return imageSavedMsg{path: savePath}
	}
}

// switchToDesktop launches the desktop GUI version of miau
func (m Model) switchToDesktop() tea.Cmd {
	return func() tea.Msg {
		m.log("🖥️ Abrindo versão desktop...")

		// Try to find miau-desktop binary
		var paths = []string{
			"miau-desktop",
			"./miau-desktop",
			"./build/bin/miau-desktop",
			"~/.local/bin/miau-desktop",
			"/usr/local/bin/miau-desktop",
		}

		for _, p := range paths {
			// Expand ~ to home dir
			if strings.HasPrefix(p, "~") {
				if home, err := os.UserHomeDir(); err == nil {
					p = filepath.Join(home, p[1:])
				}
			}

			var cmd = exec.Command(p)
			if err := cmd.Start(); err == nil {
				return desktopLaunchedMsg{success: true}
			}
		}

		return desktopLaunchedMsg{success: false, err: fmt.Errorf("miau-desktop não encontrado")}
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

		// Extract attachments and append to content if any
		if email.HasAttachments {
			var attachments = extractAllAttachments(rawData)
			if len(attachments) > 0 {
				textContent += "\n\n" + renderAttachmentList(attachments)
			}
		}

		return emailContentMsg{content: textContent}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Delegate to thread view if active
	if m.state == stateViewingThread {
		return m.updateThreadView(msg)
	}

	switch msg := msg.(type) {
	case undoCompleteMsg:
		if msg.success {
			m.log("✅ %s", msg.description)
			m.aiResponse = infoStyle.Render(msg.description)
			// Recarrega emails para refletir mudanças
			return m, m.loadEmailsFromDB()
		} else {
			m.log("⚠️ %s", msg.description)
			m.aiResponse = errorStyle.Render(msg.description)
		}
		return m, nil

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

		// Verifica attachments panel mode
		if m.showAttachments {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc", "q", "x":
				m.showAttachments = false
				m.viewerAttachments = []Attachment{}
				m.selectedAttachment = 0
				return m, nil
			case "up", "k":
				if m.selectedAttachment > 0 {
					m.selectedAttachment--
				}
				return m, nil
			case "down", "j":
				if m.selectedAttachment < len(m.viewerAttachments)-1 {
					m.selectedAttachment++
				}
				return m, nil
			case "enter", "s":
				// Salva o anexo selecionado
				return m, m.saveCurrentAttachment()
			}
			return m, nil // Bloqueia outras teclas no modo attachments
		}

		// Verifica image preview mode
		if m.showImagePreview {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc", "q", "i":
				m.showImagePreview = false
				m.imageAttachments = []Attachment{}
				m.selectedImage = 0
				m.imageRenderOutput = ""
				return m, nil
			case "left", "h":
				if m.selectedImage > 0 {
					m.selectedImage--
					m.imageLoading = true
					return m, m.renderCurrentImage()
				}
				return m, nil
			case "right", "l":
				if m.selectedImage < len(m.imageAttachments)-1 {
					m.selectedImage++
					m.imageLoading = true
					return m, m.renderCurrentImage()
				}
				return m, nil
			case "enter":
				// Abre no viewer externo
				return m, m.openImageExternal()
			case "s":
				// Salva na pasta Downloads
				return m, m.saveImage()
			}
			return m, nil // Bloqueia outras teclas no modo image preview
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

		// Analytics mode
		if m.showAnalytics {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc", "p", "q":
				m.showAnalytics = false
				m.log("📊 Analytics fechado")
				return m, nil
			case "1":
				m.analyticsPeriod = "7d"
				m.analyticsLoading = true
				return m, m.loadAnalytics()
			case "2":
				m.analyticsPeriod = "30d"
				m.analyticsLoading = true
				return m, m.loadAnalytics()
			case "3":
				m.analyticsPeriod = "90d"
				m.analyticsLoading = true
				return m, m.loadAnalytics()
			case "4":
				m.analyticsPeriod = "all"
				m.analyticsLoading = true
				return m, m.loadAnalytics()
			}
			return m, nil // Bloqueia outras teclas no modo analytics
		}

		// Settings mode
		if m.showSettings {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc", "q":
				m.showSettings = false
				m.settingsTab = 0
				m.settingsSelection = 0
				m.log("⚙️ Configurações fechadas")
				return m, nil
			case "tab", "right", "l":
				// Next tab
				m.settingsTab = (m.settingsTab + 1) % 4 // 4 tabs: Folders, Sync, Indexer, About
				m.settingsSelection = 0
				return m, nil
			case "shift+tab", "left", "h":
				// Previous tab
				m.settingsTab = (m.settingsTab + 3) % 4 // +3 same as -1 mod 4
				m.settingsSelection = 0
				return m, nil
			case "up", "k":
				if m.settingsTab == 0 { // Folders tab
					if m.settingsSelection > 0 {
						m.settingsSelection--
					}
				} else if m.settingsTab == 2 { // Indexer tab
					if m.settingsSelection > 0 {
						m.settingsSelection--
					}
				}
				return m, nil
			case "down", "j":
				if m.settingsTab == 0 { // Folders tab
					if m.settingsSelection < len(m.settingsFolders)-1 {
						m.settingsSelection++
					}
				} else if m.settingsTab == 2 { // Indexer tab
					if m.settingsSelection < 4 { // 5 opções no indexer
						m.settingsSelection++
					}
				}
				return m, nil
			case "enter", " ":
				if m.settingsTab == 0 { // Folders tab - toggle selection
					if m.settingsSelection < len(m.settingsFolders) {
						m.settingsFolders[m.settingsSelection].Selected = !m.settingsFolders[m.settingsSelection].Selected
					}
					return m, nil
				} else if m.settingsTab == 2 { // Indexer tab
					return m, m.handleSettingsAction()
				}
				return m, nil
			case "s":
				// Save settings
				if m.settingsTab == 0 { // Folders tab
					m.log("💾 Salvando configurações de pastas...")
					return m, m.saveSettingsFolders()
				}
				return m, nil
			case "+", "=":
				// Aumenta velocidade do indexador
				if m.settingsTab == 2 && m.indexState != nil && m.settingsSelection == 1 {
					var newSpeed = m.indexState.Speed + 50
					if newSpeed > 500 {
						newSpeed = 500
					}
					storage.SetIndexerSpeed(m.dbAccount.ID, newSpeed)
					m.indexState.Speed = newSpeed
					m.log("⚡ Velocidade: %d emails/min", newSpeed)
				}
				return m, nil
			case "-", "_":
				// Diminui velocidade do indexador
				if m.settingsTab == 2 && m.indexState != nil && m.settingsSelection == 1 {
					var newSpeed = m.indexState.Speed - 50
					if newSpeed < 10 {
						newSpeed = 10
					}
					storage.SetIndexerSpeed(m.dbAccount.ID, newSpeed)
					m.indexState.Speed = newSpeed
					m.log("⚡ Velocidade: %d emails/min", newSpeed)
				}
				return m, nil
			}
			return m, nil // Bloqueia outras teclas no settings
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
				// Check for quick command
				if quickCmd, ok := services.ParseQuickCommand(prompt); ok {
					return m, m.runQuickCommand(quickCmd)
				}
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
			case "i":
				// Abre preview de imagens
				m.log("📷 Tecla 'i' pressionada no viewer")
				m.imageLoading = true
				return m, m.loadImageAttachments()
			case "x":
				// Abre painel de anexos
				m.log("📎 Tecla 'x' pressionada no viewer - abrindo anexos (app=%v)", m.app != nil)
				m.attachmentsLoading = true
				return m, m.loadAllAttachments()
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

		case "<", ",":
			// Diminui largura do painel de pastas
			if m.showFolders && m.foldersWidth > 15 {
				m.foldersWidth -= 5
				m.log("📐 Folders width: %d", m.foldersWidth)
			}

		case ">", ".":
			// Aumenta largura do painel de pastas
			if m.showFolders && m.foldersWidth < 50 {
				m.foldersWidth += 5
				m.log("📐 Folders width: %d", m.foldersWidth)
			}

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
			// Abre thread view do email (usa app.Thread service)
			if !m.showFolders && len(m.emails) > 0 && m.app != nil {
				return m.openThreadView()
			}

		case "V":
			// Entra no modo de seleção múltipla (visual mode)
			if !m.showFolders && len(m.emails) > 0 {
				m.multiSelectMode = !m.multiSelectMode
				if !m.multiSelectMode {
					// Saiu do visual mode - limpa seleções
					m.selectedEmails = make(map[int64]bool)
					m.log("✖️ Modo visual desativado - seleções limpas")
				} else {
					m.selectionAnchor = m.selectedEmail
					m.log("✅ Modo visual ativado - use Space para selecionar")
				}
				return m, nil
			}

		case " ":
			// Seleciona/deseleciona email atual (só em visual mode)
			if m.multiSelectMode && !m.showFolders && len(m.emails) > 0 {
				var email = m.emails[m.selectedEmail]
				if m.selectedEmails[email.ID] {
					delete(m.selectedEmails, email.ID)
					m.log("➖ Email desmarcado: %s", email.Subject)
				} else {
					m.selectedEmails[email.ID] = true
					m.log("➕ Email marcado: %s", email.Subject)
				}
				// Avança para o próximo email
				if m.selectedEmail < len(m.emails)-1 {
					m.selectedEmail++
				}
				return m, nil
			}

		case "ctrl+a":
			// Seleciona todos os emails visíveis
			if !m.showFolders && len(m.emails) > 0 {
				if !m.multiSelectMode {
					m.multiSelectMode = true
					m.log("✅ Modo visual ativado")
				}
				m.selectedEmails = make(map[int64]bool)
				for _, email := range m.emails {
					m.selectedEmails[email.ID] = true
				}
				m.log("✅ Selecionados %d emails", len(m.emails))
				return m, nil
			}

		case "v":
			// Abre viewer simples (legacy - mantido para compatibilidade)
			if !m.showFolders && len(m.emails) > 0 {
				m.viewerEmail = &m.emails[m.selectedEmail]
				m.viewerLoading = true
				m.showViewer = true
				return m, m.loadEmailContent()
			}

		case "r":
			m.state = stateSyncing
			return m, m.syncEmails()

		case "ctrl+z":
			// Undo
			return m, m.performUndo()

		case "ctrl+y":
			// Redo
			return m, m.performRedo()

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

		case "s":
			// Resumir email selecionado
			if !m.showFolders && len(m.emails) > 0 {
				var email = m.emails[m.selectedEmail]
				m.showAI = true
				m.aiLoading = true
				m.aiLastQuestion = "Resumir email"
				m.aiResponse = statusStyle.Render("Gerando resumo do email...")
				m.aiScrollOffset = 0
				return m, m.summarizeEmail(email.ID)
			}

		case "S":
			// Resumir thread/conversa do email selecionado
			if !m.showFolders && len(m.emails) > 0 {
				var email = m.emails[m.selectedEmail]
				m.showAI = true
				m.aiLoading = true
				m.aiLastQuestion = "Resumir conversa"
				m.aiResponse = statusStyle.Render("Gerando resumo da conversa...")
				m.aiScrollOffset = 0
				return m, m.summarizeThread(email.ID)
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

		case "G":
			// Switch to Desktop GUI
			return m, m.switchToDesktop()

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
			// Archive email (Gmail style) - batch if in visual mode with selections
			if !m.showFolders && len(m.emails) > 0 {
				if m.multiSelectMode && len(m.selectedEmails) > 0 {
					// Batch archive via service
					m.log("📦 Arquivando %d emails selecionados", len(m.selectedEmails))
					return m, m.batchArchiveSelected()
				}
				var email = m.emails[m.selectedEmail]
				var messageID = ""
				if email.MessageID.Valid {
					messageID = email.MessageID.String
				}
				m.log("📦 Arquivando email: %s", email.Subject)
				return m, m.archiveEmail(email.ID, email.UID, messageID)
			}

		case "x", "#":
			// Delete email (move to trash) - batch if in visual mode with selections
			if !m.showFolders && len(m.emails) > 0 {
				if m.multiSelectMode && len(m.selectedEmails) > 0 {
					// Batch delete via service
					m.log("🗑️ Deletando %d emails selecionados", len(m.selectedEmails))
					return m, m.batchDeleteSelected()
				}
				var email = m.emails[m.selectedEmail]
				var messageID = ""
				if email.MessageID.Valid {
					messageID = email.MessageID.String
				}
				m.log("🗑️ Deletando email: %s", email.Subject)
				return m, m.deleteEmail(email.ID, email.UID, messageID)
			}

		case "m":
			// Mark as read/unread - batch if in visual mode with selections
			if !m.showFolders && len(m.emails) > 0 {
				if m.multiSelectMode && len(m.selectedEmails) > 0 {
					// Batch mark as read
					m.log("📧 Marcando %d emails como lidos", len(m.selectedEmails))
					return m, m.batchMarkReadSelected()
				}
				// Single email toggle read status
				var email = m.emails[m.selectedEmail]
				m.log("📧 Alternando status de leitura: %s", email.Subject)
				return m, m.markAsRead(email.ID, email.UID)
			}

		case "*":
			// Star/unstar email - batch if in visual mode with selections
			if !m.showFolders && len(m.emails) > 0 {
				if m.multiSelectMode && len(m.selectedEmails) > 0 {
					// Batch star
					m.log("⭐ Estrelando %d emails", len(m.selectedEmails))
					return m, m.batchStarSelected()
				}
				// Single email toggle star status
				var email = m.emails[m.selectedEmail]
				var newStarStatus = !email.IsStarred
				m.log("⭐ Alternando estrela: %s", email.Subject)
				return m, m.starEmail(email.ID, newStarStatus)
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

		case ":":
			// Abre menu de configurações
			if m.state == stateReady && !m.showFolders && !m.showViewer && !m.showCompose && !m.showDrafts && !m.showAI && !m.searchMode {
				m.showSettings = true
				m.settingsTab = 0
				m.settingsSelection = 0
				m.log("⚙️ Abrindo configurações")
				return m, tea.Batch(m.loadIndexState(), m.loadSettingsFolders())
			}

		case "p":
			// Abre painel de analytics
			if m.state == stateReady && !m.showFolders && !m.showViewer && !m.showCompose && !m.showDrafts && !m.showAI && !m.searchMode && !m.showSettings {
				m.showAnalytics = !m.showAnalytics
				if m.showAnalytics {
					m.analyticsPeriod = "30d"
					m.analyticsLoading = true
					m.log("📊 Abrindo analytics")
					return m, m.loadAnalytics()
				}
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.MouseMsg:
		// Handle mouse events for panel resizing
		if m.showFolders && m.state == stateReady {
			// Check if mouse is near the folder panel border (foldersWidth area)
			var borderX = m.foldersWidth + 1 // Approximate border position

			switch msg.Action {
			case tea.MouseActionPress:
				// Start dragging if clicking near the border (±3 pixels)
				if msg.X >= borderX-2 && msg.X <= borderX+2 {
					m.resizingPanel = true
					m.log("🖱️ Started resizing folders panel")
				}
			case tea.MouseActionRelease:
				if m.resizingPanel {
					m.resizingPanel = false
					m.log("🖱️ Stopped resizing. Width: %d", m.foldersWidth)
				}
			case tea.MouseActionMotion:
				// Drag to resize
				if m.resizingPanel {
					var newWidth = msg.X - 1
					if newWidth < 15 {
						newWidth = 15
					} else if newWidth > 50 {
						newWidth = 50
					}
					m.foldersWidth = newWidth
				}
			}
		}

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
		// Share IMAP connection with Application services
		if m.app != nil {
			m.app.SetIMAPClient(m.client)
			m.log("🔗 IMAP client compartilhado com Application")
		}
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
		// Mostra notificação de emails por 3 segundos (inclusive 0)
		m.newEmailCount = msg.synced
		m.newEmailShowTime = time.Now().Add(3 * time.Second)
		// Recarrega emails do DB após sync
		if m.state != stateReady {
			m.state = stateLoadingEmails
		}
		// Reinicia timer de auto-refresh
		m.autoRefreshStart = time.Now()
		m.autoRefreshEnabled = true
		return m, tea.Batch(m.loadEmailsFromDB(), scheduleAutoRefresh())

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

	case imageAttachmentsMsg:
		m.imageLoading = false
		if msg.err != nil {
			m.showAI = true
			m.aiResponse = errorStyle.Render("Erro ao carregar imagens: " + msg.err.Error())
			return m, nil
		}
		if len(msg.attachments) == 0 {
			m.showAI = true
			m.aiResponse = infoStyle.Render("Este email não contém imagens.")
			return m, nil
		}
		m.imageAttachments = msg.attachments
		m.selectedImage = 0
		m.showImagePreview = true
		m.imageLoading = true
		m.log("📷 %d imagens encontradas", len(msg.attachments))
		return m, m.renderCurrentImage()

	case imageRenderedMsg:
		m.imageLoading = false
		if msg.err != nil {
			m.imageRenderOutput = errorStyle.Render("Erro ao renderizar: " + msg.err.Error())
		} else {
			m.imageRenderOutput = msg.output
		}
		return m, nil

	case imageSavedMsg:
		if msg.err != nil {
			m.alerts = append(m.alerts, Alert{
				Type:      "error",
				Title:     "Erro ao salvar",
				Message:   msg.err.Error(),
				Timestamp: time.Now(),
			})
			m.showAlert = true
		} else if msg.path != "" {
			m.alerts = append(m.alerts, Alert{
				Type:      "success",
				Title:     "Imagem salva",
				Message:   "Salvo em: " + msg.path,
				Timestamp: time.Now(),
			})
			m.showAlert = true
			m.log("💾 Imagem salva: %s", msg.path)
		}
		return m, nil

	case allAttachmentsMsg:
		m.log("📎 allAttachmentsMsg recebida: err=%v, attachments=%d", msg.err != nil, len(msg.attachments))
		m.attachmentsLoading = false
		if msg.err != nil {
			m.log("📎 Erro ao carregar anexos: %v", msg.err)
			m.showAI = true
			m.aiResponse = errorStyle.Render("Erro ao carregar anexos: " + msg.err.Error())
			return m, nil
		}
		if len(msg.attachments) == 0 {
			m.log("📎 Nenhum anexo encontrado")
			m.showAI = true
			m.aiResponse = infoStyle.Render("Este email não contém anexos.")
			return m, nil
		}
		m.viewerAttachments = msg.attachments
		m.selectedAttachment = 0
		m.showAttachments = true
		m.log("📎 %d anexos encontrados, showAttachments=%v", len(msg.attachments), m.showAttachments)
		return m, nil

	case attachmentSavedMsg:
		if msg.err != nil {
			m.alerts = append(m.alerts, Alert{
				Type:      "error",
				Title:     "Erro ao salvar anexo",
				Message:   msg.err.Error(),
				Timestamp: time.Now(),
			})
			m.showAlert = true
		} else if msg.path != "" {
			m.alerts = append(m.alerts, Alert{
				Type:      "success",
				Title:     "Anexo salvo",
				Message:   "Salvo em: " + msg.path,
				Timestamp: time.Now(),
			})
			m.showAlert = true
			m.log("💾 Anexo salvo: %s", msg.path)
		}
		return m, nil

	case desktopLaunchedMsg:
		if msg.err != nil {
			m.alerts = append(m.alerts, Alert{
				Type:      "error",
				Title:     "Desktop GUI",
				Message:   "Não foi possível abrir: " + msg.err.Error(),
				Timestamp: time.Now(),
			})
			m.showAlert = true
		} else if msg.success {
			m.log("🖥️ Desktop GUI iniciado")
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

	case batchOperationCompleteMsg:
		// Multi-selection batch operation completed
		if msg.err != nil {
			m.log("❌ Erro na operação %s: %v", msg.operation, msg.err)
			return m, nil
		}
		m.log("✅ Operação %s concluída: %d emails processados", msg.operation, msg.count)
		// Sai do visual mode e limpa seleções
		m.multiSelectMode = false
		m.selectedEmails = make(map[int64]bool)
		// Recarrega emails do banco
		return m, m.loadEmailsFromDB()

	case emailStarredMsg:
		// Atualiza o email na lista local
		for i := range m.emails {
			if m.emails[i].ID == msg.emailID {
				m.emails[i].IsStarred = msg.starred
				break
			}
		}
		var starStatus = "estrelado"
		if !msg.starred {
			starStatus = "desestrelado"
		}
		m.log("⭐ Email %s", starStatus)
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

	// === AUTO-REFRESH HANDLER ===

	case autoRefreshTickMsg:
		if !m.autoRefreshEnabled || m.state != stateReady {
			return m, scheduleAutoRefresh()
		}

		// Calcula tempo desde último refresh (adiciona 1s de buffer para barra completar)
		var elapsed = time.Since(m.autoRefreshStart)
		if elapsed >= autoRefreshInterval+time.Second {
			// Hora de fazer refresh!
			m.log("⏰ Auto-refresh iniciado")
			m.state = stateSyncing
			m.autoRefreshStart = time.Now()
			return m, m.syncEmails()
		}

		// Continua aguardando
		return m, scheduleAutoRefresh()

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

	case analyticsLoadedMsg:
		m.analyticsLoading = false
		if msg.err != nil {
			m.log("❌ Erro ao carregar analytics: %v", msg.err)
			return m, nil
		}
		m.analyticsData = msg.data
		m.log("📊 Analytics carregados")
		return m, nil

	// === SETTINGS & INDEXER HANDLERS ===

	case indexStateLoadedMsg:
		if msg.err != nil {
			m.log("❌ Erro ao carregar estado do indexador: %v", msg.err)
			return m, nil
		}
		m.indexState = msg.state
		m.log("⚙️ Estado do indexador carregado: %s", msg.state.Status)
		return m, nil

	case settingsFoldersLoadedMsg:
		if msg.err != nil {
			m.log("❌ Erro ao carregar pastas: %v", msg.err)
			return m, nil
		}
		m.settingsFolders = msg.folders
		m.log("📁 Pastas para settings carregadas: %d", len(msg.folders))
		return m, nil

	case settingsSavedMsg:
		if msg.err != nil {
			m.log("❌ Erro ao salvar configurações: %v", msg.err)
		} else {
			m.log("✅ Configurações salvas com sucesso!")
		}
		return m, nil

	case indexerTickMsg:
		// Tick do indexador em background
		if !m.indexerRunning || m.indexState == nil || m.indexState.Status != storage.IndexStatusRunning {
			return m, nil
		}
		// Processa próximo lote
		return m, m.indexNextBatch()

	case indexBatchDoneMsg:
		if msg.err != nil {
			m.log("❌ Erro no indexador: %v", msg.err)
			storage.UpdateIndexState(m.dbAccount.ID, storage.IndexStatusError, m.indexState.IndexedEmails, msg.lastUID, msg.err.Error())
			m.indexState.Status = storage.IndexStatusError
			m.indexerRunning = false
			return m, nil
		}

		// Atualiza estado
		m.indexState.IndexedEmails += msg.indexed
		m.indexState.LastIndexedUID = msg.lastUID
		storage.UpdateIndexState(m.dbAccount.ID, storage.IndexStatusRunning, m.indexState.IndexedEmails, msg.lastUID, "")

		// Verifica se terminou
		if m.indexState.IndexedEmails >= m.indexState.TotalEmails || msg.indexed == 0 {
			storage.CompleteIndexer(m.dbAccount.ID)
			m.indexState.Status = storage.IndexStatusCompleted
			m.indexerRunning = false
			m.log("✅ Indexação completa: %d emails", m.indexState.IndexedEmails)
			return m, nil
		}

		m.log("📊 Indexados: %d/%d", m.indexState.IndexedEmails, m.indexState.TotalEmails)

		// Agenda próximo tick baseado na velocidade
		var interval = time.Minute / time.Duration(m.indexState.Speed)
		if interval < 100*time.Millisecond {
			interval = 100 * time.Millisecond
		}
		return m, tea.Tick(interval, func(time.Time) tea.Msg {
			return indexerTickMsg{}
		})

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
	case stateViewingThread:
		baseView = m.viewThreadView()
	case stateReady:
		if m.showCompose {
			baseView = m.viewCompose()
		} else if m.showViewer {
			baseView = m.viewEmailViewer()
		} else if m.showSettings {
			baseView = m.viewSettings()
		} else if m.showAnalytics {
			baseView = m.viewAnalytics()
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

	// Overlay de Image Preview
	if m.showImagePreview && len(m.imageAttachments) > 0 {
		return m.viewImagePreview()
	}

	// Overlay de Attachments
	if m.showAttachments && len(m.viewerAttachments) > 0 {
		return m.viewAttachmentsPanel()
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

	// Indicador de novos emails (mostra por 3 segundos após sync)
	var newEmailIndicator = ""
	if time.Now().Before(m.newEmailShowTime) {
		if m.newEmailCount > 0 {
			var newEmailStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#00FF00")).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Padding(0, 1).
				Blink(true)
			if m.newEmailCount == 1 {
				newEmailIndicator = newEmailStyle.Render("📬 1 NOVO!")
			} else {
				newEmailIndicator = newEmailStyle.Render(fmt.Sprintf("📬 %d NOVOS!", m.newEmailCount))
			}
		} else {
			var noNewStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#666666")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Padding(0, 1)
			newEmailIndicator = noNewStyle.Render("✓ 0 novos")
		}
	}

	var header = headerStyle.Render(fmt.Sprintf(" miau 🐱  %s  [%s]%s ",
		m.account.Email,
		m.currentBox,
		stats,
	)) + newEmailIndicator + draftIndicator + monitorIndicator + alertIndicator

	// Visual mode banner (quando em modo de seleção múltipla)
	var visualModeBanner = ""
	if m.multiSelectMode {
		var bannerStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#9D4EDD")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 1).
			Width(m.width)
		var selectedCount = len(m.selectedEmails)
		var helpText = "Space:selecionar  V:sair  a:arquivar  d:deletar  m:marcar_lido  s:estrelar"
		visualModeBanner = bannerStyle.Render(fmt.Sprintf("✓ VISUAL MODE - %d selecionados  |  %s", selectedCount, helpText))
	}

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

	// Auto-refresh timer indicator
	var timerIndicator = ""
	if m.autoRefreshEnabled && m.state == stateReady {
		var elapsed = time.Since(m.autoRefreshStart)
		var progress = float64(elapsed) / float64(autoRefreshInterval)
		if progress > 1 {
			progress = 1
		}
		// Barra de progresso visual com 10 caracteres
		var filled = int(progress * 10)
		var bar = ""
		for i := 0; i < 10; i++ {
			if i < filled {
				bar += "█"
			} else {
				bar += "░"
			}
		}
		var remaining = autoRefreshInterval - elapsed
		if remaining < 0 {
			remaining = 0
		}
		var timerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		timerIndicator = timerStyle.Render(fmt.Sprintf(" ⏱ %s %ds ", bar, int(remaining.Seconds())))
	}

	// Footer
	var footer string
	if m.searchMode {
		footer = subtitleStyle.Render(" ↑↓:navegar  Enter:selecionar  Esc:cancelar  /:buscar ")
	} else if m.filterActive {
		footer = subtitleStyle.Render(" y:CONFIRMAR operação  n/Esc:CANCELAR e voltar  ↑↓:navegar preview ")
	} else if m.showAI {
		var contextHint = ""
		if m.aiEmailContext != nil {
			contextHint = " [📧]"
		}
		footer = subtitleStyle.Render(fmt.Sprintf(" Enter:enviar  /:comandos  ↑↓:scroll  Esc:fechar%s ", contextHint))
	} else if activeAlerts > 0 {
		footer = subtitleStyle.Render(" ↑↓:navegar  Enter:ver  x:limpar alertas  c:novo  R:reply  a:AI  q:sair ")
	} else if m.showFolders {
		footer = subtitleStyle.Render(" ↑↓:navegar  Enter:selecionar  Tab:emails  </>:resize  q:sair ")
	} else {
		footer = subtitleStyle.Render(" ↑↓:navegar  Enter:ver  e:arquivar  x:lixo  c:novo  d:drafts  /:buscar  a:AI  q:sair ")
	}

	// Adiciona timer ao footer se ativo
	if timerIndicator != "" {
		footer = lipgloss.JoinHorizontal(lipgloss.Left, footer, timerIndicator)
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
	} else if visualModeBanner != "" {
		view = lipgloss.JoinVertical(lipgloss.Left,
			header,
			visualModeBanner,
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
	return boxStyle.Width(m.foldersWidth).Render(content)
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
		emailWidth -= m.foldersWidth + 2 // folders panel width + borders
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
	// Selection indicator (multi-select mode)
	var selectionPrefix = ""
	if m.multiSelectMode {
		if m.selectedEmails[email.ID] {
			selectionPrefix = "[✓] "
		} else {
			selectionPrefix = "[ ] "
		}
	}

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
	// formato: " X from(18) │ [📎 ]subject │ dd/mm hh:mm "
	// fixo: 1 + 1 + 18 + 3 + 3 + 11 + 1 = 38
	var subjectWidth = width - 38
	if subjectWidth < 10 {
		subjectWidth = 10
	}

	// Attachment indicator (📎 takes 2 columns + 1 space)
	var attachmentIcon = "  " // 2 spaces to match emoji width
	if email.HasAttachments {
		attachmentIcon = "📎 "
		subjectWidth -= 1 // Already accounted for 2 cols, just need 1 more for space
	}

	var subject = truncateWidth(email.Subject, subjectWidth)
	var date = email.Date.Format("02/01 15:04")

	// Pad subject to align (use visual width)
	var currentWidth = runewidth.StringWidth(subject)
	for currentWidth < subjectWidth {
		subject += " "
		currentWidth++
	}

	return fmt.Sprintf("%s %s %-18s │ %s%s │ %s ", selectionPrefix, indicator, from, attachmentIcon, subject, date)
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

// truncateWidth truncates a string to fit within max visual width (handles emojis/unicode)
func truncateWidth(s string, max int) string {
	if runewidth.StringWidth(s) <= max {
		return s
	}
	if max <= 3 {
		return runewidth.Truncate(s, max, "")
	}
	return runewidth.Truncate(s, max-3, "") + "..."
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

	// Quick command suggestions - show when input starts with /
	var suggestions string
	var currentInput = m.aiInput.Value()
	if strings.HasPrefix(currentInput, "/") && !m.aiLoading {
		var cmds = services.ParseQuickCommand(currentInput)
		if cmds == nil || cmds.Name == "" {
			// Show all available commands when just "/"
			suggestions = m.renderQuickCommandSuggestions(currentInput)
		} else {
			// Show matching suggestions
			suggestions = m.renderQuickCommandSuggestions(currentInput)
		}
	}

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
	if suggestions != "" {
		content += "\n" + suggestions
	}
	if lastQ != "" {
		content += "\n" + lastQ
	}
	if response != "" {
		content += "\n\n" + response
	}

	return aiBoxStyle.Width(width).Render(content)
}

// renderQuickCommandSuggestions renders the quick command suggestions dropdown
func (m Model) renderQuickCommandSuggestions(prefix string) string {
	var suggestionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))
	var highlightStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4ECDC4")).
		Bold(true)

	// Get matching suggestions
	var suggestions []ports.QuickCommandInfo
	if m.app != nil && m.app.AI() != nil {
		suggestions = m.app.AI().GetCommandSuggestions(prefix)
	} else {
		// Fallback to local function
		suggestions = services.GetCommandSuggestions(prefix)
	}

	if len(suggestions) == 0 {
		return suggestionStyle.Render("  Comando não encontrado. Digite /help")
	}

	// Limit to 5 suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	var sb strings.Builder
	sb.WriteString(suggestionStyle.Render("  Comandos:"))
	for _, cmd := range suggestions {
		sb.WriteString("\n  ")
		sb.WriteString(highlightStyle.Render(cmd.Usage))
		sb.WriteString(suggestionStyle.Render(" - " + cmd.Description))
		if cmd.NeedsEmail {
			sb.WriteString(suggestionStyle.Render(" [email]"))
		}
	}

	return sb.String()
}

func (m Model) viewEmailViewer() string {
	var viewerBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF6B6B")).
		Padding(0, 1)

	// Header com info do email
	var header string
	if m.viewerEmail != nil {
		var attachmentIndicator = ""
		if m.viewerEmail.HasAttachments {
			attachmentIndicator = " 📎"
		}
		header = titleStyle.Render("miau 🐱") + " - " + subtitleStyle.Render(m.viewerEmail.Subject) + attachmentIndicator + "\n"
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

	// Footer - highlight x:attachments if email has attachments
	var attachmentHint = "x:anexos"
	if m.viewerEmail != nil && m.viewerEmail.HasAttachments {
		attachmentHint = infoStyle.Render("x: 📎 anexos")
	}
	var footer = subtitleStyle.Render(" ↑↓:scroll  h:browser  i:images  ") + attachmentHint + subtitleStyle.Render("  q/Esc:voltar ")

	// Scroll info
	var scrollInfo = subtitleStyle.Render(fmt.Sprintf(" %d%% ", int(m.viewerViewport.ScrollPercent()*100)))

	// Ajusta largura se debug mode ativo
	var viewerWidth = m.width - 4
	if m.debugMode {
		viewerWidth = m.width - 48 // 44 para debug panel + margem
	}

	var body = viewerBoxStyle.Width(viewerWidth).Height(m.height - 8).Render(content)

	var viewerContent = lipgloss.JoinVertical(lipgloss.Left,
		header,
		body,
		lipgloss.JoinHorizontal(lipgloss.Left, footer, scrollInfo),
	)

	// Debug panel sempre visível em debug mode
	if m.debugMode {
		var debugPanel = m.viewDebugPanel()
		return lipgloss.JoinHorizontal(lipgloss.Top, viewerContent, debugPanel)
	}

	return viewerContent
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

func (m Model) viewAnalytics() string {
	var analyticsBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4ECDC4")).
		Padding(1, 2)

	var header = titleStyle.Render("miau 🐱") + " - " + infoStyle.Render("Analytics") + " " + subtitleStyle.Render("("+m.analyticsPeriod+")")

	var lines []string

	// Period selector
	var periodLine = "  Período: "
	var periods = []struct {
		key    string
		label  string
		period string
	}{
		{"1", "7d", "7d"},
		{"2", "30d", "30d"},
		{"3", "90d", "90d"},
		{"4", "Todos", "all"},
	}
	for _, p := range periods {
		if p.period == m.analyticsPeriod {
			periodLine += selectedStyle.Render("["+p.key+"]"+p.label) + "  "
		} else {
			periodLine += subtitleStyle.Render("["+p.key+"]"+p.label) + "  "
		}
	}
	lines = append(lines, periodLine)
	lines = append(lines, "")

	if m.analyticsLoading {
		lines = append(lines, "")
		lines = append(lines, infoStyle.Render("  ⏳ Carregando estatísticas..."))
		lines = append(lines, "")
	} else if m.analyticsData != nil && m.analyticsData.Overview != nil {
		var o = m.analyticsData.Overview

		// Overview cards
		lines = append(lines, infoStyle.Render("  📊 Visão Geral"))
		lines = append(lines, fmt.Sprintf("     Total:     %s emails", formatNumber(o.TotalEmails)))
		lines = append(lines, fmt.Sprintf("     Não lidos: %s", successStyle.Render(formatNumber(o.UnreadEmails))))
		lines = append(lines, fmt.Sprintf("     Enviados:  %s", infoStyle.Render(formatNumber(o.SentEmails))))
		lines = append(lines, fmt.Sprintf("     Arquivos:  %s", subtitleStyle.Render(formatNumber(o.ArchivedEmails))))
		lines = append(lines, fmt.Sprintf("     Storage:   %.1f MB", o.StorageUsedMB))
		lines = append(lines, "")

		// Response stats
		if m.analyticsData.ResponseTime != nil {
			var r = m.analyticsData.ResponseTime
			lines = append(lines, infoStyle.Render("  ⏱️  Tempo de Resposta"))
			lines = append(lines, fmt.Sprintf("     Média:         %s", formatDuration(r.AvgResponseMinutes)))
			lines = append(lines, fmt.Sprintf("     Taxa resposta: %.1f%%", r.ResponseRate))
			lines = append(lines, "")
		}

		// Top senders
		if len(m.analyticsData.TopSenders) > 0 {
			lines = append(lines, infoStyle.Render("  👤 Top Remetentes"))
			for i, s := range m.analyticsData.TopSenders {
				if i >= 5 {
					break // Mostra apenas top 5
				}
				var name = s.Name
				if name == "" {
					name = s.Email
				}
				// Trunca nome se muito longo
				if len(name) > 20 {
					name = name[:17] + "..."
				}
				var bar = renderMiniBar(s.Count, m.analyticsData.TopSenders[0].Count, 10)
				var unreadInfo = ""
				if s.UnreadCount > 0 {
					unreadInfo = successStyle.Render(fmt.Sprintf(" +%d", s.UnreadCount))
				}
				lines = append(lines, fmt.Sprintf("     %d. %-20s %s %3d%s",
					i+1, name, bar, s.Count, unreadInfo))
			}
			lines = append(lines, "")
		}

		// Weekday distribution
		if len(m.analyticsData.Weekday) > 0 {
			var weekdayNames = []string{"Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"}
			var maxCount = 1
			for _, w := range m.analyticsData.Weekday {
				if w.Count > maxCount {
					maxCount = w.Count
				}
			}
			lines = append(lines, infoStyle.Render("  📅 Por Dia da Semana"))
			var weekLine = "     "
			for i, w := range m.analyticsData.Weekday {
				var bar = renderVerticalBar(w.Count, maxCount, 5)
				weekLine += fmt.Sprintf("%s ", bar)
				_ = weekdayNames[i] // usado abaixo
			}
			lines = append(lines, weekLine)
			var labelLine = "     "
			for _, name := range weekdayNames {
				labelLine += fmt.Sprintf("%-4s", name)
			}
			lines = append(lines, subtitleStyle.Render(labelLine))
			lines = append(lines, "")
		}

		// Hourly distribution (simplified)
		if len(m.analyticsData.Hourly) > 0 {
			var maxCount = 1
			for _, h := range m.analyticsData.Hourly {
				if h.Count > maxCount {
					maxCount = h.Count
				}
			}
			lines = append(lines, infoStyle.Render("  ⏰ Por Hora (pico)"))
			// Find peak hour
			var peakHour = 0
			var peakCount = 0
			for _, h := range m.analyticsData.Hourly {
				if h.Count > peakCount {
					peakCount = h.Count
					peakHour = h.Hour
				}
			}
			lines = append(lines, fmt.Sprintf("     Horário de pico: %02d:00 (%d emails)", peakHour, peakCount))
		}
	} else {
		lines = append(lines, "")
		lines = append(lines, subtitleStyle.Render("  Nenhum dado disponível"))
		lines = append(lines, "")
	}

	var content = strings.Join(lines, "\n")

	// Footer
	var footer = subtitleStyle.Render(" [1-4]:período  p/Esc:fechar ")

	var box = analyticsBoxStyle.Width(m.width - 4).Render(header + "\n" + content + "\n" + footer)

	// Debug panel se ativo
	if m.debugMode && m.width > 0 {
		var debugPanel = m.viewDebugPanel()
		return lipgloss.JoinHorizontal(lipgloss.Top, box, debugPanel)
	}

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}

// formatNumber formata número para exibição (1234 -> 1.2k)
func formatNumber(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// formatDuration formata minutos para exibição (247 -> 4h 7m)
func formatDuration(minutes float64) string {
	if minutes < 60 {
		return fmt.Sprintf("%.0f min", minutes)
	}
	var hours = int(minutes / 60)
	var mins = int(minutes) % 60
	if hours < 24 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	var days = hours / 24
	return fmt.Sprintf("%dd %dh", days, hours%24)
}

// renderMiniBar renderiza uma barra horizontal proporcional
func renderMiniBar(value, max, width int) string {
	if max == 0 {
		return strings.Repeat("░", width)
	}
	var filled = value * width / max
	if filled < 1 && value > 0 {
		filled = 1
	}
	return infoStyle.Render(strings.Repeat("█", filled)) + strings.Repeat("░", width-filled)
}

// renderVerticalBar renderiza uma barra vertical (usada para weekday chart)
func renderVerticalBar(value, max, height int) string {
	if max == 0 {
		return "·"
	}
	var filled = value * height / max
	if filled < 1 && value > 0 {
		filled = 1
	}
	var bars = []string{"·", "▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	if filled >= len(bars) {
		filled = len(bars) - 1
	}
	return infoStyle.Render(bars[filled])
}

func (m Model) viewSettings() string {
	var settingsBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFD93D")).
		Padding(1, 2)

	var header = titleStyle.Render("miau 🐱") + " - " + infoStyle.Render("Configurações")

	// Tabs
	var tabs = []string{"📁 Folders", "🔄 Sync", "📚 Indexer", "ℹ About"}
	var tabLine = "  "
	for i, tab := range tabs {
		if i == m.settingsTab {
			tabLine += selectedStyle.Render(" " + tab + " ")
		} else {
			tabLine += subtitleStyle.Render(" " + tab + " ")
		}
		if i < len(tabs)-1 {
			tabLine += " │ "
		}
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, tabLine)
	lines = append(lines, "  "+strings.Repeat("─", 50))
	lines = append(lines, "")

	switch m.settingsTab {
	case 0: // Folders tab
		lines = append(lines, infoStyle.Render("  Pastas para sincronização:"))
		lines = append(lines, "")

		// Calculate visible area for scrolling (max 12 folders visible)
		var maxVisible = 12
		var startIdx = 0
		if m.settingsSelection >= maxVisible {
			startIdx = m.settingsSelection - maxVisible + 1
		}
		var endIdx = startIdx + maxVisible
		if endIdx > len(m.settingsFolders) {
			endIdx = len(m.settingsFolders)
		}

		for i := startIdx; i < endIdx; i++ {
			var f = m.settingsFolders[i]
			var checkbox = "☐"
			if f.Selected {
				checkbox = "☑"
			}
			var line = fmt.Sprintf("  %s %s", checkbox, f.Name)
			if i == m.settingsSelection {
				lines = append(lines, selectedStyle.Render(" ➤"+line))
			} else {
				lines = append(lines, subtitleStyle.Render("   "+line))
			}
		}

		if len(m.settingsFolders) > maxVisible {
			lines = append(lines, "")
			lines = append(lines, subtitleStyle.Render(fmt.Sprintf("   Mostrando %d-%d de %d pastas", startIdx+1, endIdx, len(m.settingsFolders))))
		}

		lines = append(lines, "")
		lines = append(lines, infoStyle.Render("  Space: toggle  s: salvar  Tab: próxima aba"))

	case 1: // Sync tab
		lines = append(lines, infoStyle.Render("  Configurações de Sincronização:"))
		lines = append(lines, "")
		lines = append(lines, subtitleStyle.Render("   Auto-refresh: "+fmt.Sprintf("%v", m.autoRefreshEnabled)))
		lines = append(lines, subtitleStyle.Render("   Intervalo: 1 minuto"))
		lines = append(lines, "")
		lines = append(lines, subtitleStyle.Render("   As pastas selecionadas na aba Folders"))
		lines = append(lines, subtitleStyle.Render("   serão sincronizadas automaticamente."))
		lines = append(lines, "")
		lines = append(lines, infoStyle.Render("  Tab: próxima aba  Esc: fechar"))

	case 2: // Indexer tab
		var indexStatus = "Carregando..."
		var indexProgress = ""
		var indexAction = "▶ Iniciar indexação"

		if m.indexState != nil {
			switch m.indexState.Status {
			case storage.IndexStatusIdle:
				indexStatus = subtitleStyle.Render("Parado")
				indexAction = "▶ Iniciar indexação"
			case storage.IndexStatusRunning:
				indexStatus = successStyle.Render("Executando")
				indexAction = "⏸ Pausar indexação"
			case storage.IndexStatusPaused:
				indexStatus = infoStyle.Render("Pausado")
				indexAction = "▶ Retomar indexação"
			case storage.IndexStatusCompleted:
				indexStatus = successStyle.Render("Completo ✓")
				indexAction = "Já indexado"
			case storage.IndexStatusError:
				indexStatus = errorStyle.Render("Erro")
				indexAction = "▶ Reiniciar indexação"
			}

			if m.indexState.TotalEmails > 0 {
				var percent = float64(m.indexState.IndexedEmails) / float64(m.indexState.TotalEmails) * 100
				indexProgress = fmt.Sprintf("%d/%d (%.1f%%)", m.indexState.IndexedEmails, m.indexState.TotalEmails, percent)
				var barWidth = 20
				var filled = int(percent / 100 * float64(barWidth))
				var bar = strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
				indexProgress += " " + infoStyle.Render("["+bar+"]")
			}
		}

		lines = append(lines, infoStyle.Render("  Indexação de Conteúdo"))
		lines = append(lines, fmt.Sprintf("     Status: %s", indexStatus))
		if indexProgress != "" {
			lines = append(lines, fmt.Sprintf("     Progresso: %s", indexProgress))
		}
		lines = append(lines, "")

		var options = []string{
			indexAction,
			fmt.Sprintf("⚡ Velocidade: %d emails/min  [+/-]", func() int {
				if m.indexState != nil {
					return m.indexState.Speed
				}
				return 100
			}()),
			"🛑 Cancelar indexação",
			"← Fechar configurações",
			"ℹ Sobre o miau",
		}

		for i, opt := range options {
			var prefix = "   "
			if i == m.settingsSelection {
				prefix = " ➤ "
				lines = append(lines, selectedStyle.Render(prefix+opt))
			} else {
				lines = append(lines, subtitleStyle.Render(prefix+opt))
			}
		}

		lines = append(lines, "")
		lines = append(lines, subtitleStyle.Render("  A indexação permite busca no conteúdo completo"))
		lines = append(lines, subtitleStyle.Render("  dos emails, não apenas assunto e remetente."))

	case 3: // About tab
		lines = append(lines, "")
		lines = append(lines, titleStyle.Render("  miau 🐱"))
		lines = append(lines, infoStyle.Render("  Mail Intelligence Assistant Utility"))
		lines = append(lines, "")
		lines = append(lines, subtitleStyle.Render("  Versão: 1.0.0"))
		lines = append(lines, subtitleStyle.Render("  Desenvolvido com ❤️ usando Go + Bubble Tea"))
		lines = append(lines, "")
		lines = append(lines, subtitleStyle.Render("  Cliente de email local com interface TUI,"))
		lines = append(lines, subtitleStyle.Render("  integração IMAP e suporte a IA."))
		lines = append(lines, "")
		lines = append(lines, subtitleStyle.Render("  Funcionalidades:"))
		lines = append(lines, subtitleStyle.Render("  • Sincronização IMAP"))
		lines = append(lines, subtitleStyle.Render("  • Busca em texto completo (FTS5)"))
		lines = append(lines, subtitleStyle.Render("  • Composição de emails"))
		lines = append(lines, subtitleStyle.Render("  • Detecção de bounces"))
		lines = append(lines, subtitleStyle.Render("  • Analytics de emails"))
		lines = append(lines, subtitleStyle.Render("  • Preview de imagens"))
		lines = append(lines, "")
		lines = append(lines, infoStyle.Render("  Tab: próxima aba  Esc: fechar"))
	}

	var content = strings.Join(lines, "\n")

	// Footer based on current tab
	var footer string
	switch m.settingsTab {
	case 0:
		footer = subtitleStyle.Render(" ↑↓:navegar  Space:toggle  s:salvar  Tab/←→:aba  Esc:fechar ")
	case 2:
		footer = subtitleStyle.Render(" ↑↓:navegar  Enter:selecionar  +/-:velocidade  Tab/←→:aba  Esc:fechar ")
	default:
		footer = subtitleStyle.Render(" Tab/←→:navegar abas  Esc:fechar ")
	}

	var box = settingsBoxStyle.Width(m.width - 4).Render(header + "\n" + content)

	if m.width > 0 && m.height > 0 {
		var centered = lipgloss.Place(m.width, m.height-2, lipgloss.Center, lipgloss.Center, box)
		return lipgloss.JoinVertical(lipgloss.Left, centered, footer)
	}
	return box + "\n" + footer
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

// viewImagePreview renderiza o overlay de preview de imagem
func (m Model) viewImagePreview() string {
	if len(m.imageAttachments) == 0 {
		return ""
	}

	var currentImage = m.imageAttachments[m.selectedImage]

	// Estilo do overlay - sem Background para não interferir com cores ANSI do chafa
	var overlayStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4ECDC4")).
		Padding(1, 2)

	// Header com info da imagem
	var header = titleStyle.Render("Image Preview") + " " +
		subtitleStyle.Render(fmt.Sprintf("(%d/%d)", m.selectedImage+1, len(m.imageAttachments)))

	// Info do arquivo
	var info = infoStyle.Render(fmt.Sprintf("%s (%s)",
		currentImage.Filename,
		image.FormatSize(currentImage.Size)))

	// Tipo de imagem
	var typeInfo string
	if currentImage.IsInline {
		typeInfo = subtitleStyle.Render("Inline image")
	} else {
		typeInfo = subtitleStyle.Render("Attachment")
	}

	// Conteúdo renderizado ou loading
	var imageContent string
	if m.imageLoading {
		imageContent = statusStyle.Render("Rendering image...")
	} else if m.imageRenderOutput != "" {
		imageContent = m.imageRenderOutput
	} else {
		imageContent = subtitleStyle.Render("[Image will appear here]")
	}

	// Instruções de navegação
	var footer string
	if len(m.imageAttachments) > 1 {
		footer = subtitleStyle.Render("←→/h l:navigate  Enter:open  s:save  Esc:close")
	} else {
		footer = subtitleStyle.Render("Enter:open externally  s:save  Esc:close")
	}

	// Info sobre o renderer e dica de instalação
	var rendererInfo string
	if m.imageCapabilities != nil {
		if m.imageCapabilities.Renderer == image.RendererASCII {
			rendererInfo = subtitleStyle.Render("Tip: Install chafa for better graphics (brew/apt/dnf install chafa)")
		} else {
			rendererInfo = subtitleStyle.Render(fmt.Sprintf("Renderer: %s", m.imageCapabilities.String()))
		}
	}

	var content = lipgloss.JoinVertical(lipgloss.Left,
		header,
		info,
		typeInfo,
		"",
		imageContent,
		"",
		footer,
		rendererInfo,
	)

	// Tamanho do overlay baseado no terminal - sem limitar para aproveitar espaço
	var width = m.width - 6
	if width < 50 {
		width = 50
	}

	var modal = overlayStyle.Width(width).Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal)
}

// viewAttachmentsPanel renderiza o painel de anexos
func (m Model) viewAttachmentsPanel() string {
	if len(m.viewerAttachments) == 0 {
		return ""
	}

	// Estilo do overlay
	var overlayStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF6B6B")).
		Background(lipgloss.Color("#1a1a1a")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Padding(1, 2)

	// Header
	var header = titleStyle.Render("📎 Attachments") + " " +
		subtitleStyle.Render(fmt.Sprintf("(%d files)", len(m.viewerAttachments)))

	// Lista de anexos
	var attachmentLines []string
	var maxVisible = 15
	var start = 0
	if m.selectedAttachment >= maxVisible {
		start = m.selectedAttachment - maxVisible + 1
	}

	for i, att := range m.viewerAttachments {
		if i < start {
			continue
		}
		if i >= start+maxVisible {
			break
		}

		var icon = getAttachmentIcon(att.ContentType)
		var size = formatAttachmentSize(att.Size)
		var line string

		// Trunca filename se muito longo
		var filename = att.Filename
		if len(filename) > 40 {
			filename = filename[:37] + "..."
		}

		if i == m.selectedAttachment {
			var selectedItemStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#FF6B6B")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)
			line = selectedItemStyle.Render(fmt.Sprintf(" ▶ %s %s (%s) ", icon, filename, size))
		} else {
			var typeInfo = ""
			if att.IsInline {
				typeInfo = " [inline]"
			}
			line = fmt.Sprintf("   %s %s (%s)%s", icon, filename, size, subtitleStyle.Render(typeInfo))
		}
		attachmentLines = append(attachmentLines, line)
	}

	// Footer com instruções
	var footer = subtitleStyle.Render("↑↓/jk:navigate  Enter/s:save  Esc/x:close")

	// Info sobre o anexo selecionado
	var selectedInfo = ""
	if m.selectedAttachment < len(m.viewerAttachments) {
		var att = m.viewerAttachments[m.selectedAttachment]
		selectedInfo = infoStyle.Render(fmt.Sprintf("Type: %s", att.ContentType))
	}

	var content = lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		strings.Join(attachmentLines, "\n"),
		"",
		selectedInfo,
		"",
		footer,
	)

	// Tamanho do overlay
	var width = m.width - 20
	if width < 50 {
		width = 50
	}
	if width > 80 {
		width = 80
	}

	var modal = overlayStyle.Width(width).Render(content)

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

// ============================================================================
// Batch Operations (Multi-selection)
// ============================================================================

type batchOperationCompleteMsg struct {
	operation string
	count     int
	err       error
}

func (m Model) batchArchiveSelected() tea.Cmd {
	return func() tea.Msg {
		if m.app == nil {
			return batchOperationCompleteMsg{operation: "archive", err: fmt.Errorf("app not initialized")}
		}

		var emailIDs = make([]int64, 0, len(m.selectedEmails))
		for id := range m.selectedEmails {
			emailIDs = append(emailIDs, id)
		}

		var ctx = context.Background()
		var err = m.app.Batch().ArchiveSelected(ctx, emailIDs)
		return batchOperationCompleteMsg{operation: "archive", count: len(emailIDs), err: err}
	}
}

func (m Model) batchDeleteSelected() tea.Cmd {
	return func() tea.Msg {
		if m.app == nil {
			return batchOperationCompleteMsg{operation: "delete", err: fmt.Errorf("app not initialized")}
		}

		var emailIDs = make([]int64, 0, len(m.selectedEmails))
		for id := range m.selectedEmails {
			emailIDs = append(emailIDs, id)
		}

		var ctx = context.Background()
		var err = m.app.Batch().DeleteSelected(ctx, emailIDs)
		return batchOperationCompleteMsg{operation: "delete", count: len(emailIDs), err: err}
	}
}

func (m Model) batchMarkReadSelected() tea.Cmd {
	return func() tea.Msg {
		if m.app == nil {
			return batchOperationCompleteMsg{operation: "mark_read", err: fmt.Errorf("app not initialized")}
		}

		var emailIDs = make([]int64, 0, len(m.selectedEmails))
		for id := range m.selectedEmails {
			emailIDs = append(emailIDs, id)
		}

		var ctx = context.Background()
		var err = m.app.Batch().MarkReadSelected(ctx, emailIDs, true)
		return batchOperationCompleteMsg{operation: "mark_read", count: len(emailIDs), err: err}
	}
}

func (m Model) batchStarSelected() tea.Cmd {
	return func() tea.Msg {
		if m.app == nil {
			return batchOperationCompleteMsg{operation: "star", err: fmt.Errorf("app not initialized")}
		}

		var emailIDs = make([]int64, 0, len(m.selectedEmails))
		for id := range m.selectedEmails {
			emailIDs = append(emailIDs, id)
		}

		var ctx = context.Background()
		var err = m.app.Batch().StarSelected(ctx, emailIDs, true)
		return batchOperationCompleteMsg{operation: "star", count: len(emailIDs), err: err}
	}
}

func (m Model) starEmail(emailID int64, starred bool) tea.Cmd {
	return func() tea.Msg {
		storage.MarkAsStarred(emailID, starred)
		return emailStarredMsg{emailID: emailID, starred: starred}
	}
}

type emailStarredMsg struct {
	emailID int64
	starred bool
}

// ============================================================================
// Undo/Redo Operations
// ============================================================================

// performUndo performs an undo operation
func (m Model) performUndo() tea.Cmd {
	return func() tea.Msg {
		var ctx = context.Background()
		var undo = m.app.Undo()

		if !undo.CanUndo(ctx) {
			return undoCompleteMsg{
				success:     false,
				description: "Nada para desfazer",
			}
		}

		var description = undo.GetUndoDescription(ctx)
		if err := undo.Undo(ctx); err != nil {
			return undoCompleteMsg{
				success:     false,
				description: fmt.Sprintf("Erro ao desfazer: %v", err),
			}
		}

		return undoCompleteMsg{
			success:     true,
			description: fmt.Sprintf("Desfeito: %s", description),
		}
	}
}

// performRedo performs a redo operation
func (m Model) performRedo() tea.Cmd {
	return func() tea.Msg {
		var ctx = context.Background()
		var undo = m.app.Undo()

		if !undo.CanRedo(ctx) {
			return undoCompleteMsg{
				success:     false,
				description: "Nada para refazer",
			}
		}

		var description = undo.GetRedoDescription(ctx)
		if err := undo.Redo(ctx); err != nil {
			return undoCompleteMsg{
				success:     false,
				description: fmt.Sprintf("Erro ao refazer: %v", err),
			}
		}

		return undoCompleteMsg{
			success:     true,
			description: fmt.Sprintf("Refeito: %s", description),
		}
	}
}

// undoCompleteMsg is sent when an undo/redo operation completes
type undoCompleteMsg struct {
	success     bool
	description string
}
