// Package app provides the main application core that wires all services together.
// This is the entry point for all UI layers (TUI, Web, Desktop).
package app

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/opik/miau/internal/adapters"
	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/imap"
	"github.com/opik/miau/internal/plugins/basecamp"
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
	eventBus          ports.EventBus
	syncService       *services.SyncService
	emailService      *services.EmailService
	sendService       *services.SendService
	draftService      *services.DraftService
	searchService     *services.SearchService
	batchService      *services.BatchService
	notifyService     *services.NotificationService
	analyticsService  *services.AnalyticsService
	attachmentService *services.AttachmentServicePort
	threadService     *services.ThreadService
	undoService       *services.UndoServiceImpl
	contactService    *services.ContactService
	taskService       *services.TaskService
	calendarService   *services.CalendarService
	aiService         *services.AIService
	basecampService   *services.BasecampService

	// Plugin system
	pluginStorage  *storage.PluginStorage
	pluginRegistry *services.PluginRegistry
	pluginService  *services.PluginService

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

	// Create undo service first (needed by other services)
	a.undoService = services.NewUndoService(a.storageAdapter, a.imapAdapter)
	a.undoService.SetAccount(accountInfo)

	// Create services
	a.syncService = services.NewSyncService(a.imapAdapter, a.storageAdapter, a.eventBus)
	a.syncService.SetAccount(accountInfo)

	a.emailService = services.NewEmailService(a.imapAdapter, a.storageAdapter, a.eventBus, a.undoService)
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
	a.searchService.SetIMAP(a.imapAdapter)

	a.batchService = services.NewBatchService(a.storageAdapter, a.eventBus)
	a.batchService.SetAccount(accountInfo)

	a.notifyService = services.NewNotificationService(a.storageAdapter, a.eventBus)
	a.notifyService.SetAccount(accountInfo)

	a.analyticsService = services.NewAnalyticsService(a.storageAdapter, a.eventBus)
	a.analyticsService.SetAccount(accountInfo)

	a.attachmentService = services.NewAttachmentServicePort(a.storageAdapter, a.imapAdapter)
	a.attachmentService.SetAccount(accountInfo)

	a.threadService = services.NewThreadService(a.storageAdapter, a.eventBus)
	a.threadService.SetAccount(accountInfo)
	a.threadService.SetEmailService(a.emailService)

	// Create contact service (needs ContactStorageAdapter and GmailContactsPort)
	var contactStorage = storage.NewContactStorageAdapter()
	var gmailContactsPort ports.GmailContactsPort
	if a.gmailAdapter != nil && a.gmailAdapter.Client() != nil {
		gmailContactsPort = a.gmailAdapter.ContactsAdapter()
		fmt.Printf("[App.Start] Gmail contacts port created successfully\n")
	} else {
		fmt.Printf("[App.Start] Gmail adapter or client is nil, contacts sync will not work\n")
	}
	var photoDir = filepath.Join(config.GetConfigPath(), "photos")
	a.contactService = services.NewContactService(contactStorage, gmailContactsPort, a.eventBus, photoDir)

	// Create task service
	a.taskService = services.NewTaskService()

	// Create calendar service (depends on task service for sync)
	a.calendarService = services.NewCalendarService(a.taskService)

	// Create AI service
	a.aiService = services.NewAIService(a.storageAdapter, a.eventBus)
	a.aiService.SetAccount(accountInfo)

	// Create Basecamp service
	a.basecampService = services.NewBasecampService(a.eventBus)

	// Wire up bidirectional Task ↔ Calendar sync
	a.taskService.SetCalendarSync(a.calendarService)

	// Wire up Google Calendar client if available
	if a.gmailAdapter != nil && a.gmailAdapter.CalendarClient() != nil {
		a.calendarService.SetGoogleCalendarClient(a.gmailAdapter.CalendarClient())
		fmt.Printf("[App.Start] Google Calendar client connected\n")
	}

	// Initialize plugin system
	a.pluginStorage = storage.NewPluginStorage()
	a.pluginRegistry = services.NewPluginRegistry(a.pluginStorage)
	a.pluginService = services.NewPluginService(a.pluginRegistry, a.pluginStorage, a.eventBus)
	a.pluginService.SetAccount(accountInfo)

	// Register built-in plugins
	a.pluginRegistry.Register(basecamp.New())

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

// AI returns the AI service
func (a *Application) AI() ports.AIService {
	return a.aiService
}

// Analytics returns the analytics service
func (a *Application) Analytics() ports.AnalyticsService {
	return a.analyticsService
}

// Attachment returns the attachment service
func (a *Application) Attachment() ports.AttachmentService {
	return a.attachmentService
}

// Thread returns the thread service
func (a *Application) Thread() ports.ThreadService {
	return a.threadService
}

// Undo returns the undo service
func (a *Application) Undo() ports.UndoService {
	return a.undoService
}

// Contacts returns the contact service
func (a *Application) Contacts() ports.ContactService {
	return a.contactService
}

// Tasks returns the task service
func (a *Application) Tasks() ports.TaskService {
	return a.taskService
}

// Calendar returns the calendar service
func (a *Application) Calendar() ports.CalendarService {
	return a.calendarService
}

// Basecamp returns the Basecamp service
func (a *Application) Basecamp() ports.BasecampService {
	return a.basecampService
}

// Plugins returns the plugin service
func (a *Application) Plugins() ports.PluginService {
	return a.pluginService
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

// GetAllAccounts returns all configured accounts
func (a *Application) GetAllAccounts() []ports.AccountInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var accounts []ports.AccountInfo
	for _, acc := range a.cfg.Accounts {
		// Get the account ID from storage if available
		var id int64
		if acc.Email == a.account.Email && a.accountInfo != nil {
			id = a.accountInfo.ID
		} else {
			// Try to get from storage
			var accInfo, err = a.storageAdapter.GetOrCreateAccount(context.Background(), acc.Email, acc.Name)
			if err == nil && accInfo != nil {
				id = accInfo.ID
			}
		}
		accounts = append(accounts, ports.AccountInfo{
			ID:    id,
			Email: acc.Email,
			Name:  acc.Name,
		})
	}
	return accounts
}

// SetCurrentAccount switches to a different account
// This disconnects the current IMAP connection and reinitializes all adapters
func (a *Application) SetCurrentAccount(email string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Find account by email
	var newAccount *config.Account
	for i := range a.cfg.Accounts {
		if a.cfg.Accounts[i].Email == email {
			newAccount = &a.cfg.Accounts[i]
			break
		}
	}
	if newAccount == nil {
		return fmt.Errorf("account not found: %s", email)
	}

	// Check if it's the same account
	if a.account.Email == email {
		return nil // Already on this account
	}

	// Save previous email for event
	var previousEmail = a.account.Email

	// Step 1: Disconnect current IMAP
	if a.imapAdapter != nil {
		a.imapAdapter.Close()
	}

	// Step 2: Update current account reference
	a.account = newAccount

	// Step 3: Update app config
	a.appConfig.AccountEmail = newAccount.Email
	a.appConfig.AccountName = newAccount.Name
	a.appConfig.IMAPHost = newAccount.IMAP.Host
	a.appConfig.IMAPPort = newAccount.IMAP.Port
	if newAccount.AuthType == config.AuthTypeOAuth2 {
		a.appConfig.AuthType = ports.AuthTypeOAuth2
	} else {
		a.appConfig.AuthType = ports.AuthTypePassword
	}
	if newAccount.SendMethod == config.SendMethodGmailAPI {
		a.appConfig.SendMethod = ports.SendMethodGmailAPI
	} else {
		a.appConfig.SendMethod = ports.SendMethodSMTP
	}

	// Step 4: Reinitialize adapters for new account
	a.imapAdapter = adapters.NewIMAPAdapter(newAccount)
	a.smtpAdapter = adapters.NewSMTPAdapter(newAccount)
	a.gmailAdapter = adapters.NewGmailAPIAdapter(newAccount, config.GetConfigPath())

	// Step 5: Get or create account info in storage
	var accountInfo, err = a.storageAdapter.GetOrCreateAccount(context.Background(), newAccount.Email, newAccount.Name)
	if err != nil {
		return fmt.Errorf("failed to get/create account in storage: %w", err)
	}
	a.accountInfo = accountInfo

	// Step 6: Update all services with new account
	a.undoService.SetAccount(accountInfo)
	a.syncService.SetAccount(accountInfo)
	a.emailService.SetAccount(accountInfo)
	a.sendService.SetAccount(accountInfo)
	a.sendService.SetSendMethod(a.appConfig.SendMethod)
	a.draftService.SetAccount(accountInfo)
	a.searchService.SetAccount(accountInfo)
	a.batchService.SetAccount(accountInfo)
	a.notifyService.SetAccount(accountInfo)
	a.analyticsService.SetAccount(accountInfo)
	a.attachmentService.SetAccount(accountInfo)
	a.threadService.SetAccount(accountInfo)
	a.aiService.SetAccount(accountInfo)
	a.pluginService.SetAccount(accountInfo)

	// Step 7: Update IMAP and Gmail in services that need them
	a.syncService.SetIMAPAdapter(a.imapAdapter)
	a.searchService.SetIMAP(a.imapAdapter)
	a.attachmentService.SetIMAPAdapter(a.imapAdapter)

	// Update Gmail adapter in send service
	var gmailPort ports.GmailAPIPort
	if a.gmailAdapter != nil {
		gmailPort = a.gmailAdapter
	}
	a.sendService.SetGmailAPI(gmailPort)

	// Update calendar client if available
	if a.gmailAdapter != nil && a.gmailAdapter.CalendarClient() != nil {
		a.calendarService.SetGoogleCalendarClient(a.gmailAdapter.CalendarClient())
	}

	// Step 8: Emit account switched event
	if a.eventBus != nil {
		a.eventBus.Publish(ports.NewAccountSwitchedEvent(previousEmail, email, accountInfo.ID))
	}

	fmt.Printf("[Application] Switched account from %s to %s\n", previousEmail, email)
	return nil
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

// GetGmailAdapter returns the Gmail API adapter for direct access
func (a *Application) GetGmailAdapter() *adapters.GmailAPIAdapter {
	return a.gmailAdapter
}

// SetIMAPClient sets an external IMAP client (for TUI to share connection with services)
// This allows the TUI's IMAP connection to be shared with AttachmentService and others.
func (a *Application) SetIMAPClient(client interface{}) {
	var imapClient, ok = client.(*imap.Client)
	if !ok {
		return
	}
	if a.imapAdapter == nil {
		return
	}
	a.imapAdapter.SetClient(imapClient)
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

	// Update the calendar service with the new calendar client
	if a.gmailAdapter.CalendarClient() != nil {
		a.calendarService.SetGoogleCalendarClient(a.gmailAdapter.CalendarClient())
		fmt.Printf("[ReinitializeGmailAdapter] Google Calendar client updated\n")
	}

	// Update contact service with new contacts adapter
	if a.gmailAdapter.Client() != nil && a.contactService != nil {
		// Contact adapter is already wired in Start()
		fmt.Printf("[ReinitializeGmailAdapter] Gmail adapter reinitialized with all services\n")
	}

	return nil
}

// SyncThreadIDsFromGmail syncs thread IDs from Gmail API for existing emails
// Returns the number of emails updated
// Uses TRUE incremental sync - only queries Gmail for emails that haven't been checked yet
// Supports cancellation via context
// maxEmails: limit how many emails to process (0 = no limit, but uses default of 500)
func (a *Application) SyncThreadIDsFromGmail(ctx context.Context, progressCallback func(processed, total int)) (int, error) {
	return a.SyncThreadIDsFromGmailWithLimit(ctx, 500, progressCallback)
}

// SyncThreadIDsFromGmailWithLimit syncs thread IDs with a configurable limit
func (a *Application) SyncThreadIDsFromGmailWithLimit(ctx context.Context, maxEmails int, progressCallback func(processed, total int)) (int, error) {
	a.mu.RLock()
	var gmailAdapter = a.gmailAdapter
	var account = a.accountInfo
	a.mu.RUnlock()

	if gmailAdapter == nil {
		return 0, fmt.Errorf("Gmail API not configured")
	}
	if account == nil {
		return 0, fmt.Errorf("no account configured")
	}

	// Check for cancellation
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	// Get emails that need thread sync (haven't been checked yet)
	// This returns emails where thread_synced_at IS NULL
	var emails, err = storage.GetEmailsForThreadSync(account.ID)
	if err != nil {
		return 0, fmt.Errorf("failed to get emails for thread sync: %w", err)
	}

	if len(emails) == 0 {
		return 0, nil
	}

	// Separate emails that already have thread_id from those that need API lookup
	var needAPILookup []storage.EmailForThreadSync
	var alreadyHaveThread []int64

	for _, e := range emails {
		if e.ThreadID != "" {
			// Already has thread_id, just mark as synced (no API call needed)
			alreadyHaveThread = append(alreadyHaveThread, e.ID)
		} else {
			needAPILookup = append(needAPILookup, e)
		}
	}

	// Batch mark emails that already have thread_id as synced
	if len(alreadyHaveThread) > 0 {
		storage.MarkEmailsThreadSynced(alreadyHaveThread)
		fmt.Printf("[SyncThreadIDs] Marked %d emails as synced (already had thread_id)\n", len(alreadyHaveThread))
	}

	if len(needAPILookup) == 0 {
		return 0, nil
	}

	// Apply limit to avoid processing too many at once
	if maxEmails > 0 && len(needAPILookup) > maxEmails {
		fmt.Printf("[SyncThreadIDs] Limiting from %d to %d emails (run again for more)\n", len(needAPILookup), maxEmails)
		needAPILookup = needAPILookup[:maxEmails]
	}

	var total = len(needAPILookup)
	var updated = 0

	fmt.Printf("[SyncThreadIDs] Processing %d emails that need API lookup\n", total)

	// Process each email individually - TRUE incremental sync
	for i, email := range needAPILookup {
		// Check for cancellation
		if ctx.Err() != nil {
			return updated, ctx.Err()
		}

		// Progress callback
		if progressCallback != nil {
			progressCallback(i+1, total)
		}

		if email.MessageID == "" {
			// Mark as synced even if no message_id (won't retry)
			storage.UpdateEmailThreadID(email.ID, "")
			continue
		}

		// Query Gmail API for this specific message
		var msgInfo, apiErr = gmailAdapter.GetMessageInfoByRFC822MsgID(email.MessageID)
		if apiErr != nil {
			// Mark as synced to avoid retrying failed lookups
			storage.UpdateEmailThreadID(email.ID, "")
			continue
		}

		if msgInfo != nil && msgInfo.ThreadID != "" {
			if updateErr := storage.UpdateEmailThreadID(email.ID, msgInfo.ThreadID); updateErr == nil {
				updated++
			}
		} else {
			// Mark as synced even if no thread found
			storage.UpdateEmailThreadID(email.ID, "")
		}
	}

	return updated, nil
}
