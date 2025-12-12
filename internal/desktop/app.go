package desktop

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"
	"time"

	"github.com/opik/miau/internal/app"
	"github.com/opik/miau/internal/auth"
	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/ports"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// App is the main Wails application struct
// All public methods are exposed to the frontend
type App struct {
	wailsApp      *application.App
	application   ports.App
	cfg           *config.Config
	account       *config.Account
	mu            sync.RWMutex
	connected     bool
	currentFolder string

	// Thread sync cancellation
	threadSyncCancel context.CancelFunc
}

// NewApp creates a new Wails App instance
func NewApp() *App {
	return &App{
		currentFolder: "INBOX",
	}
}

// SetApplication stores the Wails app reference for events/dialogs
func (a *App) SetApplication(wailsApp *application.App) {
	a.wailsApp = wailsApp

	// Initialize the application on startup
	a.startup()
}

// startup initializes the app (called from SetApplication)
func (a *App) startup() {
	// Load config
	var err error
	a.cfg, err = config.Load()
	if err != nil || a.cfg == nil || len(a.cfg.Accounts) == 0 {
		slog.Error("No config found, need setup")
		return
	}

	// Find the current account (saved preference or first account)
	a.account = &a.cfg.Accounts[0] // default to first
	if a.cfg.CurrentAccount != "" {
		for i := range a.cfg.Accounts {
			if a.cfg.Accounts[i].Email == a.cfg.CurrentAccount {
				a.account = &a.cfg.Accounts[i]
				slog.Info("Using saved current account", "email", a.account.Email)
				break
			}
		}
	}

	// Create application instance
	a.application, err = app.New(a.cfg, a.account, false)
	if err != nil {
		slog.Error("Failed to create app", "error", err)
		return
	}

	// Start application (initializes DB, services)
	if err := a.application.Start(); err != nil {
		slog.Error("Failed to start app", "error", err)
		return
	}

	// Pre-load signature to avoid crash when opening compose modal
	// This moves the API call to startup instead of UI interaction
	go func() {
		if err := a.application.Send().LoadSignature(context.Background()); err != nil {
			slog.Error("Failed to load signature", "error", err)
		} else {
			slog.Info("Signature loaded successfully")
		}
	}()

	// Setup event forwarding from Go to frontend
	a.setupEventForwarding()

	slog.Info("Desktop app started successfully")
}

// Shutdown is called when the app terminates
func (a *App) Shutdown() {
	if a.application != nil {
		a.application.Stop()
	}
	slog.Info("Desktop app shutdown")
}

// setupEventForwarding subscribes to app events and forwards to frontend
func (a *App) setupEventForwarding() {
	if a.application == nil || a.application.Events() == nil {
		return
	}

	a.application.Events().SubscribeAll(func(evt ports.Event) {
		if a.wailsApp == nil {
			return
		}

		switch e := evt.(type) {
		case *ports.NewEmailEvent:
			a.wailsApp.Event.Emit("email:new", a.emailMetadataToDTO(&e.Email))
		case *ports.EmailReadEvent:
			a.wailsApp.Event.Emit("email:read", e.EmailID, e.Read)
		case *ports.SyncStartedEvent:
			a.wailsApp.Event.Emit("sync:started", e.Folder)
		case *ports.SyncCompletedEvent:
			var newCount = 0
			if e.Result != nil {
				newCount = e.Result.NewEmails
			}
			a.wailsApp.Event.Emit("sync:completed", e.Folder, newCount)
		case *ports.SyncErrorEvent:
			a.wailsApp.Event.Emit("sync:error", e.Error.Error())
		case *ports.ConnectedEvent:
			a.mu.Lock()
			a.connected = true
			a.mu.Unlock()
			a.wailsApp.Event.Emit("connection:connected")
		case *ports.DisconnectedEvent:
			a.mu.Lock()
			a.connected = false
			a.mu.Unlock()
			a.wailsApp.Event.Emit("connection:disconnected", e.Reason)
		case *ports.ConnectErrorEvent:
			a.wailsApp.Event.Emit("connection:error", e.Error.Error())
		case *ports.SendCompletedEvent:
			var messageID = ""
			if e.Result != nil {
				messageID = e.Result.MessageID
			}
			a.wailsApp.Event.Emit("send:completed", messageID)
		case *ports.BounceEvent:
			a.wailsApp.Event.Emit("bounce:detected", e.Bounce.OriginalMessageID, e.Bounce.Reason)
		case *ports.BatchCreatedEvent:
			if e.Operation != nil {
				a.wailsApp.Event.Emit("batch:created", e.Operation.ID, e.Operation.Description)
			}
		case *ports.IndexProgressEvent:
			a.wailsApp.Event.Emit("index:progress", e.Current, e.Total)
		case ports.AccountSwitchedEvent:
			a.wailsApp.Event.Emit("account:switched", e.NewEmail, e.NewAccountID)
		}
	})
}

// Helper to convert ports.EmailMetadata to EmailDTO
func (a *App) emailMetadataToDTO(email *ports.EmailMetadata) EmailDTO {
	if email == nil {
		return EmailDTO{}
	}
	return EmailDTO{
		ID:             email.ID,
		UID:            email.UID,
		Subject:        email.Subject,
		FromName:       email.FromName,
		FromEmail:      email.FromEmail,
		Date:           email.Date,
		IsRead:         email.IsRead,
		IsStarred:      email.IsStarred,
		HasAttachments: email.HasAttachments,
		Snippet:        email.Snippet,
		ThreadID:       email.ThreadID,
		ThreadCount:    email.ThreadCount,
	}
}

// Helper to convert ports.EmailContent to EmailDetailDTO
func (a *App) emailContentToDTO(email *ports.EmailContent) *EmailDetailDTO {
	if email == nil {
		return nil
	}
	var attachments []AttachmentDTO
	for _, att := range email.Attachments {
		var dataStr string
		if att.IsInline && len(att.Data) > 0 {
			dataStr = base64.StdEncoding.EncodeToString(att.Data)
		}
		attachments = append(attachments, AttachmentDTO{
			ID:          att.ID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			ContentID:   att.ContentID,
			Size:        att.Size,
			Data:        dataStr,
			IsInline:    att.IsInline,
			PartNumber:  att.PartNumber,
		})
	}
	return &EmailDetailDTO{
		EmailDTO: EmailDTO{
			ID:             email.ID,
			UID:            email.UID,
			Subject:        email.Subject,
			FromName:       email.FromName,
			FromEmail:      email.FromEmail,
			Date:           email.Date,
			IsRead:         email.IsRead,
			IsStarred:      email.IsStarred,
			HasAttachments: email.HasAttachments,
			Snippet:        email.Snippet,
		},
		ToAddresses:  email.ToAddresses,
		CcAddresses:  email.CcAddresses,
		BodyText:     email.BodyText,
		BodyHTML:     email.BodyHTML,
		Attachments:  attachments,
	}
}

// Helper to convert ports.Folder to FolderDTO
func (a *App) folderToDTO(folder *ports.Folder) FolderDTO {
	if folder == nil {
		return FolderDTO{}
	}
	return FolderDTO{
		ID:             folder.ID,
		Name:           folder.Name,
		TotalMessages:  folder.TotalMessages,
		UnreadMessages: folder.UnreadMessages,
	}
}

// IsReady returns true if the application is ready to use
func (a *App) IsReady() bool {
	return a.application != nil
}

// NeedsSetup returns true if no account is configured
func (a *App) NeedsSetup() bool {
	return a.cfg == nil || len(a.cfg.Accounts) == 0
}

// GetAppInfo returns basic app information
func (a *App) GetAppInfo() map[string]string {
	return map[string]string{
		"name":    "miau",
		"version": "1.0.0",
	}
}

// ShowError displays an error dialog
func (a *App) ShowError(title, message string) {
	if a.wailsApp == nil {
		return
	}
	a.wailsApp.Dialog.Error().
		SetTitle(title).
		SetMessage(message).
		Show()
}

// ShowInfo displays an info dialog
func (a *App) ShowInfo(title, message string) {
	if a.wailsApp == nil {
		return
	}
	a.wailsApp.Dialog.Info().
		SetTitle(title).
		SetMessage(message).
		Show()
}

// Confirm shows a confirmation dialog
func (a *App) Confirm(title, message string) bool {
	if a.wailsApp == nil {
		return false
	}

	resultChan := make(chan bool, 1)

	dialog := a.wailsApp.Dialog.Question().
		SetTitle(title).
		SetMessage(message)

	yesBtn := dialog.AddButton("Yes")
	yesBtn.OnClick(func() {
		resultChan <- true
	})

	noBtn := dialog.AddButton("No")
	noBtn.OnClick(func() {
		resultChan <- false
	})
	noBtn.SetAsCancel()

	dialog.Show()

	return <-resultChan
}

// OpenURL opens a URL in the default browser
func (a *App) OpenURL(url string) {
	// Use exec to open URL in default browser
	exec.Command("xdg-open", url).Start()
}

// SwitchToTerminal opens the TUI version in a terminal
func (a *App) SwitchToTerminal() error {
	var err error

	// Try different terminal emulators
	terminals := []struct {
		cmd  string
		args []string
	}{
		{"gnome-terminal", []string{"--", "miau"}},
		{"konsole", []string{"-e", "miau"}},
		{"xfce4-terminal", []string{"-e", "miau"}},
		{"alacritty", []string{"-e", "miau"}},
		{"kitty", []string{"miau"}},
		{"xterm", []string{"-e", "miau"}},
	}

	for _, t := range terminals {
		var cmd = exec.Command(t.cmd, t.args...)
		err = cmd.Start()
		if err == nil {
			slog.Info("Launched TUI", "terminal", t.cmd)
			return nil
		}
	}

	// If no terminal found, show error
	a.ShowError("Terminal não encontrado", "Não foi possível encontrar um terminal. Instale gnome-terminal, konsole, alacritty ou xterm.")

	return fmt.Errorf("no terminal emulator found")
}

// GetError returns a formatted error for the frontend
func (a *App) getError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}

// NeedsOAuth2Auth returns true if OAuth2 authentication is required
func (a *App) NeedsOAuth2Auth() bool {
	if a.account == nil || a.account.AuthType != config.AuthTypeOAuth2 {
		return false
	}
	if a.account.OAuth2 == nil {
		return false
	}
	// Check if token exists
	var tokenPath = auth.GetTokenPath(config.GetConfigPath(), a.account.Email)
	var oauth2Cfg = auth.GetOAuth2Config(a.account.OAuth2.ClientID, a.account.OAuth2.ClientSecret)
	_, err := auth.GetValidToken(oauth2Cfg, tokenPath)
	return err != nil
}

// StartOAuth2Auth initiates the OAuth2 authentication flow
// Opens browser for user to authenticate and waits for callback
func (a *App) StartOAuth2Auth() error {
	if a.account == nil {
		return fmt.Errorf("no account configured")
	}
	if a.account.OAuth2 == nil {
		return fmt.Errorf("OAuth2 not configured for this account")
	}

	slog.Info("Starting OAuth2 authentication flow...")

	var oauth2Cfg = auth.GetOAuth2Config(a.account.OAuth2.ClientID, a.account.OAuth2.ClientSecret)

	// This will open browser and wait for callback
	token, err := auth.AuthenticateWithBrowser(oauth2Cfg)
	if err != nil {
		slog.Error("OAuth2 authentication failed", "error", err)
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Save token
	var tokenPath = auth.GetTokenPath(config.GetConfigPath(), a.account.Email)
	if err := auth.SaveToken(tokenPath, token); err != nil {
		slog.Error("Failed to save token", "error", err)
		return fmt.Errorf("failed to save token: %w", err)
	}

	slog.Info("OAuth2 token saved successfully")

	// Reinitialize Gmail adapter in the application
	if coreApp, ok := a.application.(*app.Application); ok {
		if err := coreApp.ReinitializeGmailAdapter(); err != nil {
			slog.Error("Failed to reinitialize Gmail adapter", "error", err)
			return fmt.Errorf("failed to initialize Gmail API: %w", err)
		}
		slog.Info("Gmail API adapter reinitialized successfully")

		// Reload signature with new adapter
		go func() {
			if err := a.application.Send().LoadSignature(context.Background()); err != nil {
				slog.Error("Failed to reload signature", "error", err)
			} else {
				slog.Info("Signature reloaded successfully")
			}
		}()
	}

	return nil
}

// ============================================================================
// ACCOUNT OPERATIONS
// ============================================================================

// GetCurrentAccount returns the currently active account
func (a *App) GetCurrentAccount() *AccountDTO {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.account == nil {
		return nil
	}

	return &AccountDTO{
		Email: a.account.Email,
		Name:  a.account.Name,
	}
}

// GetAllAccounts returns all configured accounts
func (a *App) GetAllAccounts() []AccountDTO {
	if a.application == nil {
		return nil
	}

	var accounts = a.application.GetAllAccounts()
	var result []AccountDTO
	for _, acc := range accounts {
		result = append(result, AccountDTO{
			Email: acc.Email,
			Name:  acc.Name,
		})
	}
	return result
}

// SetCurrentAccount switches to a different account
func (a *App) SetCurrentAccount(email string) error {
	if a.application == nil {
		return fmt.Errorf("application not initialized")
	}

	slog.Info("Switching account", "email", email)

	// Call the application layer to switch accounts
	if err := a.application.SetCurrentAccount(email); err != nil {
		slog.Error("Failed to switch account", "error", err)
		return err
	}

	// Update local reference
	a.mu.Lock()
	for i := range a.cfg.Accounts {
		if a.cfg.Accounts[i].Email == email {
			a.account = &a.cfg.Accounts[i]
			break
		}
	}
	// Save current account preference to config
	a.cfg.CurrentAccount = email
	a.mu.Unlock()

	// Persist to config file
	if err := config.Save(a.cfg); err != nil {
		slog.Error("Failed to save current account preference", "error", err)
		// Don't return error - account switch worked, just preference not saved
	}

	slog.Info("Account switched successfully", "email", email)
	return nil
}

// AddAccount adds a new email account to the configuration
func (a *App) AddAccount(newAccount NewAccountConfigDTO) error {
	slog.Info("Adding new account", "email", newAccount.Email)

	// Validate required fields
	if newAccount.Email == "" {
		return fmt.Errorf("email is required")
	}
	if newAccount.ImapHost == "" {
		return fmt.Errorf("IMAP host is required")
	}
	if newAccount.ImapPort <= 0 {
		newAccount.ImapPort = 993
	}

	// Load current config or create new
	var cfg *config.Config
	var err error
	if a.cfg != nil {
		cfg = a.cfg
	} else {
		cfg, err = config.Load()
		if err != nil || cfg == nil {
			cfg = config.DefaultConfig()
		}
	}

	// Check if account already exists
	for _, acc := range cfg.Accounts {
		if acc.Email == newAccount.Email {
			return fmt.Errorf("account %s already exists", newAccount.Email)
		}
	}

	// Create the new account
	var account = config.Account{
		Name:  newAccount.Name,
		Email: newAccount.Email,
		IMAP: config.ImapConfig{
			Host: newAccount.ImapHost,
			Port: newAccount.ImapPort,
			TLS:  newAccount.ImapPort == 993,
		},
	}

	// Set auth type and credentials
	if newAccount.AuthType == "oauth2" {
		account.AuthType = config.AuthTypeOAuth2
		account.OAuth2 = &config.OAuth2Config{
			ClientID:     newAccount.ClientID,
			ClientSecret: newAccount.ClientSecret,
		}
		// Default to Gmail API for OAuth2 accounts
		if newAccount.SendMethod == "" {
			account.SendMethod = config.SendMethodGmailAPI
		} else {
			account.SendMethod = config.SendMethod(newAccount.SendMethod)
		}
	} else {
		account.AuthType = config.AuthTypePassword
		account.Password = newAccount.Password
		account.SendMethod = config.SendMethodSMTP
		// Set SMTP config for password auth
		if newAccount.SmtpHost != "" {
			account.SMTP = config.SMTPConfig{
				Host: newAccount.SmtpHost,
				Port: newAccount.SmtpPort,
			}
		}
	}

	// Add account to config
	cfg.Accounts = append(cfg.Accounts, account)

	// Save config
	if err := config.Save(cfg); err != nil {
		slog.Error("Failed to save config", "error", err)
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Update local reference
	a.cfg = cfg

	slog.Info("Account added successfully", "email", newAccount.Email)
	return nil
}

// StartOAuth2AuthForNewAccount initiates OAuth2 authentication for a new account
// This should be called after AddAccount for OAuth2 accounts
func (a *App) StartOAuth2AuthForNewAccount(email, clientID, clientSecret string) error {
	slog.Info("Starting OAuth2 authentication for new account", "email", email)

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("OAuth2 credentials required")
	}

	var oauth2Cfg = auth.GetOAuth2Config(clientID, clientSecret)

	// This will open browser and wait for callback
	token, err := auth.AuthenticateWithBrowser(oauth2Cfg)
	if err != nil {
		slog.Error("OAuth2 authentication failed", "error", err)
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Save token
	var tokenPath = auth.GetTokenPath(config.GetConfigPath(), email)
	if err := auth.SaveToken(tokenPath, token); err != nil {
		slog.Error("Failed to save token", "error", err)
		return fmt.Errorf("failed to save token: %w", err)
	}

	slog.Info("OAuth2 token saved successfully for new account", "email", email)
	return nil
}

// GetKnownImapHost returns the known IMAP host for a domain
func (a *App) GetKnownImapHost(email string) map[string]interface{} {
	var result = map[string]interface{}{
		"imapHost":   "imap.gmail.com",
		"imapPort":   993,
		"smtpHost":   "smtp.gmail.com",
		"smtpPort":   587,
		"isGoogle":   false,
		"sendMethod": "smtp",
	}

	if email == "" {
		return result
	}

	var parts = make([]string, 0)
	for _, ch := range email {
		if ch == '@' {
			parts = append(parts, "")
		} else if len(parts) > 0 {
			parts[len(parts)-1] += string(ch)
		}
	}

	if len(parts) == 0 {
		return result
	}

	var domain = parts[0]

	// Known hosts mapping
	var knownHosts = map[string]map[string]interface{}{
		"gmail.com": {
			"imapHost":   "imap.gmail.com",
			"imapPort":   993,
			"smtpHost":   "smtp.gmail.com",
			"smtpPort":   587,
			"isGoogle":   true,
			"sendMethod": "gmail_api",
		},
		"googlemail.com": {
			"imapHost":   "imap.gmail.com",
			"imapPort":   993,
			"smtpHost":   "smtp.gmail.com",
			"smtpPort":   587,
			"isGoogle":   true,
			"sendMethod": "gmail_api",
		},
		"outlook.com": {
			"imapHost":   "outlook.office365.com",
			"imapPort":   993,
			"smtpHost":   "smtp.office365.com",
			"smtpPort":   587,
			"isGoogle":   false,
			"sendMethod": "smtp",
		},
		"hotmail.com": {
			"imapHost":   "outlook.office365.com",
			"imapPort":   993,
			"smtpHost":   "smtp.office365.com",
			"smtpPort":   587,
			"isGoogle":   false,
			"sendMethod": "smtp",
		},
		"yahoo.com": {
			"imapHost":   "imap.mail.yahoo.com",
			"imapPort":   993,
			"smtpHost":   "smtp.mail.yahoo.com",
			"smtpPort":   587,
			"isGoogle":   false,
			"sendMethod": "smtp",
		},
		"icloud.com": {
			"imapHost":   "imap.mail.me.com",
			"imapPort":   993,
			"smtpHost":   "smtp.mail.me.com",
			"smtpPort":   587,
			"isGoogle":   false,
			"sendMethod": "smtp",
		},
	}

	if hostConfig, ok := knownHosts[domain]; ok {
		return hostConfig
	}

	// Default assumes Google Workspace for unknown domains
	result["isGoogle"] = true
	result["sendMethod"] = "gmail_api"
	return result
}

// ============================================================================
// SNOOZE OPERATIONS
// ============================================================================

// GetSnoozePresets returns available snooze presets
func (a *App) GetSnoozePresets() []SnoozePresetDTO {
	if a.application == nil {
		return nil
	}

	var presets = a.application.Snooze().GetSnoozePresets()
	var result []SnoozePresetDTO
	for _, p := range presets {
		result = append(result, SnoozePresetDTO{
			Preset:      string(p.Preset),
			Label:       p.Label,
			Description: p.Description,
			Time:        p.Time,
		})
	}
	return result
}

// SnoozeEmail snoozes an email with a preset
func (a *App) SnoozeEmail(emailID int64, preset string) error {
	if a.application == nil {
		return fmt.Errorf("application not initialized")
	}

	var snoozePreset = ports.SnoozePreset(preset)
	return a.application.Snooze().SnoozeEmailPreset(context.Background(), emailID, snoozePreset)
}

// SnoozeEmailCustom snoozes an email until a custom time
func (a *App) SnoozeEmailCustom(emailID int64, untilTimeStr string) error {
	if a.application == nil {
		return fmt.Errorf("application not initialized")
	}

	var until, err = parseTime(untilTimeStr)
	if err != nil {
		return fmt.Errorf("invalid time format: %w", err)
	}

	return a.application.Snooze().SnoozeEmail(context.Background(), emailID, until)
}

// UnsnoozeEmail removes snooze from an email
func (a *App) UnsnoozeEmail(emailID int64) error {
	if a.application == nil {
		return fmt.Errorf("application not initialized")
	}

	return a.application.Snooze().UnsnoozeEmail(context.Background(), emailID)
}

// GetSnoozedEmails returns all snoozed emails
func (a *App) GetSnoozedEmails() ([]SnoozedEmailDTO, error) {
	if a.application == nil {
		return nil, fmt.Errorf("application not initialized")
	}

	var snoozes, err = a.application.Snooze().GetSnoozedEmails(context.Background())
	if err != nil {
		return nil, err
	}

	var result []SnoozedEmailDTO
	for _, s := range snoozes {
		result = append(result, SnoozedEmailDTO{
			ID:          s.ID,
			EmailID:     s.EmailID,
			SnoozedAt:   s.SnoozedAt,
			SnoozeUntil: s.SnoozeUntil,
			Preset:      string(s.Preset),
		})
	}
	return result, nil
}

// GetSnoozedEmailsCount returns the count of snoozed emails
func (a *App) GetSnoozedEmailsCount() (int, error) {
	if a.application == nil {
		return 0, fmt.Errorf("application not initialized")
	}

	return a.application.Snooze().GetSnoozedEmailsCount(context.Background())
}

// IsEmailSnoozed checks if an email is currently snoozed
func (a *App) IsEmailSnoozed(emailID int64) (bool, error) {
	if a.application == nil {
		return false, fmt.Errorf("application not initialized")
	}

	return a.application.Snooze().IsEmailSnoozed(context.Background(), emailID)
}

// ============================================================================
// SCHEDULE SEND OPERATIONS
// ============================================================================

// GetSchedulePresets returns available schedule send presets
func (a *App) GetSchedulePresets() []SchedulePresetDTO {
	if a.application == nil {
		return nil
	}

	var presets = a.application.Schedule().GetSchedulePresets()
	var result []SchedulePresetDTO
	for _, p := range presets {
		result = append(result, SchedulePresetDTO{
			Preset:      string(p.Preset),
			Label:       p.Label,
			Description: p.Description,
			Time:        p.Time,
		})
	}
	return result
}

// GetScheduledDrafts returns all scheduled drafts
func (a *App) GetScheduledDrafts() ([]ScheduledDraftDTO, error) {
	if a.application == nil {
		return nil, fmt.Errorf("application not initialized")
	}

	var drafts, err = a.application.Schedule().GetScheduledDrafts(context.Background())
	if err != nil {
		return nil, err
	}

	var result []ScheduledDraftDTO
	for _, d := range drafts {
		result = append(result, ScheduledDraftDTO{
			ID:              d.ID,
			To:              d.ToAddresses,
			Subject:         d.Subject,
			ScheduledSendAt: d.ScheduledSendAt,
			Status:          string(d.Status),
			CreatedAt:       d.CreatedAt,
		})
	}
	return result, nil
}

// GetScheduledDraftsCount returns the count of scheduled drafts
func (a *App) GetScheduledDraftsCount() (int, error) {
	if a.application == nil {
		return 0, fmt.Errorf("application not initialized")
	}

	return a.application.Schedule().GetScheduledDraftsCount(context.Background())
}

// Helper to parse time from frontend format
func parseTime(timeStr string) (time.Time, error) {
	// Try multiple formats
	var formats = []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time: %s", timeStr)
}
