// Package app provides the main application core that wires all services together.
// This is the entry point for all UI layers (TUI, Web, Desktop).
package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/opik/miau/internal/adapters"
	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/services"
	"github.com/opik/miau/internal/storage"
)

// Application is the main application instance that wires all components together.
// It implements ports.App interface.
type Application struct {
	mu sync.RWMutex

	// Configuration
	cfg       *config.Config
	account   *config.Account
	appConfig ports.AppConfig

	// Ports (adapters)
	imapAdapter    *adapters.IMAPAdapter
	storageAdapter *adapters.StorageAdapter
	smtpAdapter    *adapters.SMTPAdapter
	gmailAdapter   *adapters.GmailAPIAdapter

	// Services
	eventBus     ports.EventBus
	syncService  *services.SyncService
	emailService *services.EmailService
	sendService  *services.SendService
	draftService *services.DraftService
	searchService *services.SearchService
	batchService *services.BatchService
	notifyService *services.NotificationService

	// State
	accountInfo *ports.AccountInfo
	started     bool
}

// New creates a new Application instance
func New(cfg *config.Config, account *config.Account, debugMode bool) (*Application, error) {
	if cfg == nil || account == nil {
		return nil, fmt.Errorf("config and account are required")
	}

	var app = &Application{
		cfg:     cfg,
		account: account,
		appConfig: ports.AppConfig{
			AccountEmail: account.Email,
			AccountName:  account.Name,
			IMAPHost:     account.IMAP.Host,
			IMAPPort:     account.IMAP.Port,
			DebugMode:    debugMode,
			DataPath:     config.GetConfigPath(),
		},
	}

	// Set auth type and send method
	if account.AuthType == config.AuthTypeOAuth2 {
		app.appConfig.AuthType = ports.AuthTypeOAuth2
	} else {
		app.appConfig.AuthType = ports.AuthTypePassword
	}

	if account.SendMethod == config.SendMethodGmailAPI {
		app.appConfig.SendMethod = ports.SendMethodGmailAPI
	} else {
		app.appConfig.SendMethod = ports.SendMethodSMTP
	}

	return app, nil
}

// Start initializes and starts all services
func (a *Application) Start() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.started {
		return nil
	}

	// Initialize database
	if err := storage.Init(a.cfg.Storage.Database); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create adapters
	a.imapAdapter = adapters.NewIMAPAdapter(a.account)
	a.storageAdapter = adapters.NewStorageAdapter()
	a.smtpAdapter = adapters.NewSMTPAdapter(a.account)
	a.gmailAdapter = adapters.NewGmailAPIAdapter(a.account, config.GetConfigPath())

	// Create event bus
	a.eventBus = services.NewEventBus()

	// Get or create account in storage
	var accountInfo, err = a.storageAdapter.GetOrCreateAccount(context.Background(), a.account.Email, a.account.Name)
	if err != nil {
		return fmt.Errorf("failed to get/create account: %w", err)
	}
	a.accountInfo = accountInfo

	// Create services
	a.syncService = services.NewSyncService(a.imapAdapter, a.storageAdapter, a.eventBus)
	a.syncService.SetAccount(accountInfo)

	a.emailService = services.NewEmailService(a.imapAdapter, a.storageAdapter, a.eventBus)
	a.emailService.SetAccount(accountInfo)

	// IMPORTANT: We must explicitly check for nil before assigning to interface
	// to avoid the "nil interface containing nil pointer" gotcha in Go.
	// An interface is only truly nil if both type and value are nil.
	var smtpPort ports.SMTPPort
	if a.smtpAdapter != nil {
		smtpPort = a.smtpAdapter
	}
	var gmailPort ports.GmailAPIPort
	if a.gmailAdapter != nil {
		gmailPort = a.gmailAdapter
	}
	a.sendService = services.NewSendService(smtpPort, gmailPort, a.storageAdapter, a.eventBus)
	a.sendService.SetAccount(accountInfo)
	a.sendService.SetSendMethod(a.appConfig.SendMethod)

	a.draftService = services.NewDraftService(a.storageAdapter, a.eventBus)
	a.draftService.SetAccount(accountInfo)

	a.searchService = services.NewSearchService(a.storageAdapter, a.eventBus)
	a.searchService.SetAccount(accountInfo)

	a.batchService = services.NewBatchService(a.storageAdapter, a.eventBus)
	a.batchService.SetAccount(accountInfo)

	a.notifyService = services.NewNotificationService(a.storageAdapter, a.eventBus)
	a.notifyService.SetAccount(accountInfo)

	a.started = true
	return nil
}

// Stop shuts down all services
func (a *Application) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.started {
		return nil
	}

	// Disconnect IMAP
	if a.imapAdapter != nil {
		a.imapAdapter.Close()
	}

	a.started = false
	return nil
}

// Email returns the email service
func (a *Application) Email() ports.EmailService {
	return a.emailService
}

// Send returns the send service
func (a *Application) Send() ports.SendService {
	return a.sendService
}

// Draft returns the draft service
func (a *Application) Draft() ports.DraftService {
	return a.draftService
}

// Search returns the search service
func (a *Application) Search() ports.SearchService {
	return a.searchService
}

// Batch returns the batch service
func (a *Application) Batch() ports.BatchService {
	return a.batchService
}

// Notification returns the notification service
func (a *Application) Notification() ports.NotificationService {
	return a.notifyService
}

// Sync returns the sync service
func (a *Application) Sync() ports.SyncService {
	return a.syncService
}

// AI returns the AI service (not implemented yet)
func (a *Application) AI() ports.AIService {
	return nil
}

// Events returns the event bus
func (a *Application) Events() ports.EventBus {
	return a.eventBus
}

// GetCurrentAccount returns the current account info
func (a *Application) GetCurrentAccount() *ports.AccountInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.accountInfo
}

// SetCurrentAccount sets the current account (for multi-account support)
func (a *Application) SetCurrentAccount(email string) error {
	// For now, we only support single account
	return fmt.Errorf("multi-account not supported yet")
}

// GetConfig returns the app configuration
func (a *Application) GetConfig() ports.AppConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.appConfig
}

// GetRawConfig returns the raw config.Account for backward compatibility
func (a *Application) GetRawConfig() *config.Account {
	return a.account
}

// GetIMAPAdapter returns the IMAP adapter for direct access (backward compatibility)
func (a *Application) GetIMAPAdapter() *adapters.IMAPAdapter {
	return a.imapAdapter
}

// GetStorageAdapter returns the storage adapter for direct access (backward compatibility)
func (a *Application) GetStorageAdapter() *adapters.StorageAdapter {
	return a.storageAdapter
}

// Connect connects to the IMAP server
func (a *Application) Connect(ctx context.Context) error {
	return a.syncService.Connect(ctx)
}

// LoadFolders loads folders from IMAP
func (a *Application) LoadFolders(ctx context.Context) ([]ports.Folder, error) {
	return a.syncService.LoadFolders(ctx)
}

// SyncFolder syncs a specific folder
func (a *Application) SyncFolder(ctx context.Context, folder string) (*ports.SyncResult, error) {
	return a.syncService.SyncFolder(ctx, folder)
}

// ReinitializeGmailAdapter reinitializes the Gmail API adapter after OAuth2 authentication
// This is called after the user completes the browser-based OAuth2 flow
func (a *Application) ReinitializeGmailAdapter() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Create new Gmail adapter with fresh tokens
	a.gmailAdapter = adapters.NewGmailAPIAdapter(a.account, config.GetConfigPath())
	if a.gmailAdapter == nil {
		return fmt.Errorf("failed to initialize Gmail API - token may be missing")
	}

	// Update the send service with the new adapter
	// We need to check for nil before passing to avoid the nil interface gotcha
	var gmailPort ports.GmailAPIPort = a.gmailAdapter
	a.sendService.SetGmailAPI(gmailPort)

	return nil
}
