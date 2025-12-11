package desktop

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"

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

	a.account = &a.cfg.Accounts[0]

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
