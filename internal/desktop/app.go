package desktop

import (
	"context"
	"encoding/base64"
	"fmt"
	"os/exec"
	"sync"

	"github.com/opik/miau/internal/app"
	"github.com/opik/miau/internal/auth"
	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/ports"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is the main Wails application struct
// All public methods are exposed to the frontend
type App struct {
	ctx           context.Context
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

// startup is called when the app starts
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Load config
	var err error
	a.cfg, err = config.Load()
	if err != nil || a.cfg == nil || len(a.cfg.Accounts) == 0 {
		runtime.LogError(ctx, "No config found, need setup")
		return
	}

	a.account = &a.cfg.Accounts[0]

	// Create application instance
	a.application, err = app.New(a.cfg, a.account, false)
	if err != nil {
		runtime.LogErrorf(ctx, "Failed to create app: %v", err)
		return
	}

	// Start application (initializes DB, services)
	if err := a.application.Start(); err != nil {
		runtime.LogErrorf(ctx, "Failed to start app: %v", err)
		return
	}

	// Pre-load signature to avoid crash when opening compose modal
	// This moves the API call to startup instead of UI interaction
	go func() {
		if err := a.application.Send().LoadSignature(ctx); err != nil {
			runtime.LogErrorf(ctx, "Failed to load signature: %v", err)
		} else {
			runtime.LogInfo(ctx, "Signature loaded successfully")
		}
	}()

	// Setup event forwarding from Go to frontend
	a.setupEventForwarding()

	runtime.LogInfo(ctx, "Desktop app started successfully")
}

// shutdown is called when the app terminates
func (a *App) Shutdown(ctx context.Context) {
	if a.application != nil {
		a.application.Stop()
	}
	runtime.LogInfo(ctx, "Desktop app shutdown")
}

// domReady is called after the frontend DOM is ready
func (a *App) DomReady(ctx context.Context) {
	runtime.LogInfo(ctx, "Frontend DOM ready")
}

// beforeClose is called when the user tries to close the window
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	return false // Allow close
}

// setupEventForwarding subscribes to app events and forwards to frontend
func (a *App) setupEventForwarding() {
	if a.application == nil || a.application.Events() == nil {
		return
	}

	a.application.Events().SubscribeAll(func(evt ports.Event) {
		switch e := evt.(type) {
		case *ports.NewEmailEvent:
			runtime.EventsEmit(a.ctx, "email:new", a.emailMetadataToDTO(&e.Email))
		case *ports.EmailReadEvent:
			runtime.EventsEmit(a.ctx, "email:read", e.EmailID, e.Read)
		case *ports.SyncStartedEvent:
			runtime.EventsEmit(a.ctx, "sync:started", e.Folder)
		case *ports.SyncCompletedEvent:
			var newCount = 0
			if e.Result != nil {
				newCount = e.Result.NewEmails
			}
			runtime.EventsEmit(a.ctx, "sync:completed", e.Folder, newCount)
		case *ports.SyncErrorEvent:
			runtime.EventsEmit(a.ctx, "sync:error", e.Error.Error())
		case *ports.ConnectedEvent:
			a.mu.Lock()
			a.connected = true
			a.mu.Unlock()
			runtime.EventsEmit(a.ctx, "connection:connected")
		case *ports.DisconnectedEvent:
			a.mu.Lock()
			a.connected = false
			a.mu.Unlock()
			runtime.EventsEmit(a.ctx, "connection:disconnected", e.Reason)
		case *ports.ConnectErrorEvent:
			runtime.EventsEmit(a.ctx, "connection:error", e.Error.Error())
		case *ports.SendCompletedEvent:
			var messageID = ""
			if e.Result != nil {
				messageID = e.Result.MessageID
			}
			runtime.EventsEmit(a.ctx, "send:completed", messageID)
		case *ports.BounceEvent:
			runtime.EventsEmit(a.ctx, "bounce:detected", e.Bounce.OriginalMessageID, e.Bounce.Reason)
		case *ports.BatchCreatedEvent:
			if e.Operation != nil {
				runtime.EventsEmit(a.ctx, "batch:created", e.Operation.ID, e.Operation.Description)
			}
		case *ports.IndexProgressEvent:
			runtime.EventsEmit(a.ctx, "index:progress", e.Current, e.Total)
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
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.ErrorDialog,
		Title:   title,
		Message: message,
	})
}

// ShowInfo displays an info dialog
func (a *App) ShowInfo(title, message string) {
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   title,
		Message: message,
	})
}

// Confirm shows a confirmation dialog
func (a *App) Confirm(title, message string) bool {
	result, _ := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         title,
		Message:       message,
		Buttons:       []string{"Yes", "No"},
		DefaultButton: "No",
	})
	return result == "Yes"
}

// OpenURL opens a URL in the default browser
func (a *App) OpenURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
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
			runtime.LogInfo(a.ctx, "Launched TUI in "+t.cmd)
			return nil
		}
	}

	// If no terminal found, show error
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.ErrorDialog,
		Title:   "Terminal não encontrado",
		Message: "Não foi possível encontrar um terminal. Instale gnome-terminal, konsole, alacritty ou xterm.",
	})

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

	runtime.LogInfo(a.ctx, "Starting OAuth2 authentication flow...")

	var oauth2Cfg = auth.GetOAuth2Config(a.account.OAuth2.ClientID, a.account.OAuth2.ClientSecret)

	// This will open browser and wait for callback
	token, err := auth.AuthenticateWithBrowser(oauth2Cfg)
	if err != nil {
		runtime.LogErrorf(a.ctx, "OAuth2 authentication failed: %v", err)
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Save token
	var tokenPath = auth.GetTokenPath(config.GetConfigPath(), a.account.Email)
	if err := auth.SaveToken(tokenPath, token); err != nil {
		runtime.LogErrorf(a.ctx, "Failed to save token: %v", err)
		return fmt.Errorf("failed to save token: %w", err)
	}

	runtime.LogInfo(a.ctx, "OAuth2 token saved successfully")

	// Reinitialize Gmail adapter in the application
	if coreApp, ok := a.application.(*app.Application); ok {
		if err := coreApp.ReinitializeGmailAdapter(); err != nil {
			runtime.LogErrorf(a.ctx, "Failed to reinitialize Gmail adapter: %v", err)
			return fmt.Errorf("failed to initialize Gmail API: %w", err)
		}
		runtime.LogInfo(a.ctx, "Gmail API adapter reinitialized successfully")

		// Reload signature with new adapter
		go func() {
			if err := a.application.Send().LoadSignature(a.ctx); err != nil {
				runtime.LogErrorf(a.ctx, "Failed to reload signature: %v", err)
			} else {
				runtime.LogInfo(a.ctx, "Signature reloaded successfully")
			}
		}()
	}

	return nil
}
