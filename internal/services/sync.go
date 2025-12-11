package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/storage"
)

// EssentialFolders are folders that should always be synced
var EssentialFolders = []string{
	"INBOX",
	"[Gmail]/Sent Mail",
	"[Gmail]/Trash",
}

// SyncService implements ports.SyncService
type SyncService struct {
	mu       sync.RWMutex
	imap     ports.IMAPPort
	storage  ports.StoragePort
	events   ports.EventBus
	status   ports.ConnectionStatus
	account  *ports.AccountInfo
	folders  map[string]*ports.Folder
	config   ports.SyncConfig
}

// NewSyncService creates a new SyncService
func NewSyncService(imap ports.IMAPPort, storage ports.StoragePort, events ports.EventBus) *SyncService {
	return &SyncService{
		imap:    imap,
		storage: storage,
		events:  events,
		status:  ports.ConnectionStatusDisconnected,
		folders: make(map[string]*ports.Folder),
		config:  ports.DefaultSyncConfig(),
	}
}

// GetSyncConfig returns the current sync configuration
func (s *SyncService) GetSyncConfig() ports.SyncConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// SetSyncConfig sets the sync configuration
func (s *SyncService) SetSyncConfig(config ports.SyncConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

// SetAccount sets the current account for sync operations
func (s *SyncService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// SetIMAPAdapter updates the IMAP adapter (used when switching accounts)
func (s *SyncService) SetIMAPAdapter(imap ports.IMAPPort) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.imap = imap
	s.status = ports.ConnectionStatusDisconnected
	s.folders = make(map[string]*ports.Folder)
}

// Connect establishes connection to the email server
func (s *SyncService) Connect(ctx context.Context) error {
	s.mu.Lock()
	s.status = ports.ConnectionStatusConnecting
	s.mu.Unlock()

	var err = s.imap.Connect(ctx)
	if err != nil {
		s.mu.Lock()
		s.status = ports.ConnectionStatusError
		s.mu.Unlock()

		s.events.Publish(ports.ConnectErrorEvent{
			BaseEvent: ports.NewBaseEvent(ports.EventTypeConnectError),
			Error:     err,
		})
		return fmt.Errorf("failed to connect: %w", err)
	}

	s.mu.Lock()
	s.status = ports.ConnectionStatusConnected
	s.mu.Unlock()

	s.events.Publish(ports.ConnectedEvent{
		BaseEvent: ports.NewBaseEvent(ports.EventTypeConnected),
	})

	return nil
}

// Disconnect closes the connection
func (s *SyncService) Disconnect(ctx context.Context) error {
	var err = s.imap.Close()

	s.mu.Lock()
	s.status = ports.ConnectionStatusDisconnected
	s.mu.Unlock()

	s.events.Publish(ports.DisconnectedEvent{
		BaseEvent: ports.NewBaseEvent(ports.EventTypeDisconnected),
		Reason:    "user requested",
	})

	return err
}

// IsConnected returns true if connected
func (s *SyncService) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.imap.IsConnected()
}

// GetConnectionStatus returns the current connection status
func (s *SyncService) GetConnectionStatus() ports.ConnectionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// SyncFolder syncs a specific folder using OPTIMIZED batch operations
// This is for incremental sync - uses FetchNewEmailsBatch (1 request for N emails)
// Does NOT run purge - call PurgeDeletedEmails separately
func (s *SyncService) SyncFolder(ctx context.Context, folderName string) (*ports.SyncResult, error) {
	s.mu.RLock()
	var account = s.account
	var config = s.config
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	s.events.Publish(ports.SyncStartedEvent{
		BaseEvent: ports.NewBaseEvent(ports.EventTypeSyncStarted),
		Folder:    folderName,
	})

	// Get folder from storage
	var folder, err = s.storage.GetFolderByName(ctx, account.ID, folderName)
	if err != nil {
		return nil, fmt.Errorf("folder not found: %w", err)
	}

	// Registra início do sync
	var syncID, _ = storage.LogSyncStart(account.ID, folder.ID)

	// Select mailbox on IMAP
	var status, err2 = s.imap.SelectMailbox(ctx, folderName)
	if err2 != nil {
		storage.LogSyncComplete(syncID, 0, 0, err2)
		s.events.Publish(ports.SyncErrorEvent{
			BaseEvent: ports.NewBaseEvent(ports.EventTypeSyncError),
			Folder:    folderName,
			Error:     err2,
		})
		return nil, fmt.Errorf("failed to select mailbox: %w", err2)
	}

	// Update folder stats
	s.storage.UpdateFolderStats(ctx, folder.ID, int(status.NumMessages), int(status.NumUnseen))

	// Get latest UID from storage
	var latestUID, _ = s.storage.GetLatestUID(ctx, folder.ID)

	// Check if this is initial sync (no emails yet)
	var isInitialSync = latestUID == 0

	var result = &ports.SyncResult{
		LatestUID: latestUID,
	}

	// Use appropriate sync method
	if isInitialSync {
		// Initial sync: use date-based fetch (last N days)
		result, err = s.initialSyncFolder(ctx, account, folder, config, syncID)
		if err != nil {
			return nil, err
		}
	} else {
		// Incremental sync: use batch fetch (1 request for all new emails)
		var batchSize = config.IncrementalBatchSize
		if batchSize == 0 {
			batchSize = 100
		}

		// OPTIMIZED: Single request for envelope + bodystructure
		var newEmails, err3 = s.imap.FetchNewEmailsBatch(ctx, latestUID, batchSize)
		if err3 != nil {
			storage.LogSyncComplete(syncID, 0, 0, err3)
			s.events.Publish(ports.SyncErrorEvent{
				BaseEvent: ports.NewBaseEvent(ports.EventTypeSyncError),
				Folder:    folderName,
				Error:     err3,
			})
			return nil, fmt.Errorf("failed to fetch new emails: %w", err3)
		}

		// Store new emails (attachments already included from batch fetch!)
		s.storeEmailsBatch(ctx, account, folder, newEmails, result)
	}

	// NOTE: Purge is now SEPARATE - not called during sync
	// Call PurgeDeletedEmails periodically instead

	// Conta novos emails desde o último sync
	var newCount, _ = storage.CountNewEmailsSinceLastSync(account.ID, folder.ID)
	result.NewEmails = newCount

	// Registra conclusão do sync
	storage.LogSyncComplete(syncID, newCount, 0, nil)

	s.events.Publish(ports.SyncCompletedEvent{
		BaseEvent: ports.NewBaseEvent(ports.EventTypeSyncCompleted),
		Folder:    folderName,
		Result:    result,
	})

	return result, nil
}

// initialSyncFolder performs optimized first-time sync using date-based search
func (s *SyncService) initialSyncFolder(ctx context.Context, account *ports.AccountInfo, folder *ports.Folder, config ports.SyncConfig, syncID int64) (*ports.SyncResult, error) {
	var days = config.InitialSyncDays
	if days == 0 {
		days = 30
	}
	var maxEmails = config.InitialMaxPerFolder
	if maxEmails == 0 {
		maxEmails = 500
	}

	var result = &ports.SyncResult{}

	// OPTIMIZED: Fetch by date (last N days) with batch operation
	var emails, err = s.imap.FetchEmailsSinceDateBatch(ctx, days, maxEmails)
	if err != nil {
		storage.LogSyncComplete(syncID, 0, 0, err)
		s.events.Publish(ports.SyncErrorEvent{
			BaseEvent: ports.NewBaseEvent(ports.EventTypeSyncError),
			Folder:    folder.Name,
			Error:     err,
		})
		return nil, fmt.Errorf("failed to fetch emails: %w", err)
	}

	// Store emails (attachments included from batch!)
	s.storeEmailsBatch(ctx, account, folder, emails, result)

	return result, nil
}

// storeEmailsBatch stores emails from batch fetch (includes attachment metadata)
func (s *SyncService) storeEmailsBatch(ctx context.Context, account *ports.AccountInfo, folder *ports.Folder, emails []ports.IMAPEmail, result *ports.SyncResult) {
	for _, email := range emails {
		var content = &ports.EmailContent{
			EmailMetadata: ports.EmailMetadata{
				UID:        email.UID,
				MessageID:  email.MessageID,
				Subject:    email.Subject,
				FromName:   email.FromName,
				FromEmail:  email.FromEmail,
				ToAddress:  email.To,
				Date:       email.Date,
				IsRead:     email.Seen,
				IsStarred:  email.Flagged,
				Size:       email.Size,
				InReplyTo:  email.InReplyTo,
				References: email.References,
			},
			BodyText:       email.BodyText,
			HasAttachments: email.HasAttachments,
		}

		var emailID, messageID, upsertErr = s.storage.UpsertEmail(ctx, account.ID, folder.ID, content)
		if upsertErr != nil {
			result.Errors = append(result.Errors, upsertErr)
			continue
		}

		// Collect ID for thread sync (only emails with message_id can have thread_id)
		if messageID != "" {
			result.NewEmailIDs = append(result.NewEmailIDs, emailID)
		}

		// Store attachments if any (already fetched in batch!)
		if email.HasAttachments && len(email.Attachments) > 0 {
			for _, att := range email.Attachments {
				var contentID = att.ContentID
				if len(contentID) > 2 && contentID[0] == '<' && contentID[len(contentID)-1] == '>' {
					contentID = contentID[1 : len(contentID)-1]
				}

				var attachment = &ports.Attachment{
					EmailID:     emailID,
					Filename:    att.Filename,
					ContentType: att.ContentType,
					ContentID:   contentID,
					Size:        att.Size,
					IsInline:    att.IsInline,
					PartNumber:  att.PartNumber,
					Encoding:    att.Encoding,
				}

				s.storage.UpsertAttachment(ctx, attachment)
			}
		}

		if email.UID > result.LatestUID {
			result.LatestUID = email.UID
		}

		// Publish new email event
		s.events.Publish(ports.NewEmailEvent{
			BaseEvent: ports.NewBaseEvent(ports.EventTypeNewEmail),
			Email:     content.EmailMetadata,
		})
	}
}

// InitialSync performs optimized first-time sync for a folder
// Uses date-based search (last N days) instead of full UID scan
func (s *SyncService) InitialSync(ctx context.Context, folderName string) (*ports.SyncResult, error) {
	s.mu.RLock()
	var account = s.account
	var config = s.config
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	s.events.Publish(ports.SyncStartedEvent{
		BaseEvent: ports.NewBaseEvent(ports.EventTypeSyncStarted),
		Folder:    folderName,
	})

	var folder, err = s.storage.GetFolderByName(ctx, account.ID, folderName)
	if err != nil {
		return nil, fmt.Errorf("folder not found: %w", err)
	}

	var syncID, _ = storage.LogSyncStart(account.ID, folder.ID)

	if _, err := s.imap.SelectMailbox(ctx, folderName); err != nil {
		storage.LogSyncComplete(syncID, 0, 0, err)
		return nil, fmt.Errorf("failed to select mailbox: %w", err)
	}

	var result, syncErr = s.initialSyncFolder(ctx, account, folder, config, syncID)
	if syncErr != nil {
		return nil, syncErr
	}

	var newCount, _ = storage.CountNewEmailsSinceLastSync(account.ID, folder.ID)
	result.NewEmails = newCount
	storage.LogSyncComplete(syncID, newCount, 0, nil)

	s.events.Publish(ports.SyncCompletedEvent{
		BaseEvent: ports.NewBaseEvent(ports.EventTypeSyncCompleted),
		Folder:    folderName,
		Result:    result,
	})

	return result, nil
}

// InitialSyncEssentialFolders performs optimized first-time sync for essential folders
func (s *SyncService) InitialSyncEssentialFolders(ctx context.Context) ([]ports.SyncResult, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	var foldersToSync = GetConfiguredFolders(account.ID)
	var results []ports.SyncResult

	for _, folderName := range foldersToSync {
		var result, err = s.InitialSync(ctx, folderName)
		if err != nil {
			continue // Skip folders that don't exist
		}
		if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

// PurgeDeletedEmails checks for deleted emails and marks them locally
// This is a SEPARATE job that should run periodically, not on every sync
func (s *SyncService) PurgeDeletedEmails(ctx context.Context, folderName string) (int, error) {
	log.Printf("[PurgeDeletedEmails] starting for folder: %s", folderName)

	s.mu.RLock()
	var account = s.account
	var config = s.config
	s.mu.RUnlock()

	if account == nil {
		log.Printf("[PurgeDeletedEmails] error: no account set")
		return 0, fmt.Errorf("no account set")
	}

	if !config.PurgeEnabled {
		log.Printf("[PurgeDeletedEmails] purge disabled in config")
		return 0, nil
	}

	var folder, err = s.storage.GetFolderByName(ctx, account.ID, folderName)
	if err != nil {
		log.Printf("[PurgeDeletedEmails] error getting folder: %v", err)
		return 0, err
	}

	// Select mailbox
	if _, err := s.imap.SelectMailbox(ctx, folderName); err != nil {
		log.Printf("[PurgeDeletedEmails] error selecting mailbox: %v", err)
		return 0, err
	}

	var purged, purgeErr = s.purgeDeleted(ctx, folder.ID)
	log.Printf("[PurgeDeletedEmails] completed: purged=%d, err=%v", purged, purgeErr)
	return purged, purgeErr
}

// purgeDeleted marks emails as deleted that were removed from server
// NOTE: This is expensive for large mailboxes, so we skip if too many local emails
func (s *SyncService) purgeDeleted(ctx context.Context, folderID int64) (int, error) {
	// Get local email count first
	var localUIDs, err2 = s.storage.GetAllUIDs(ctx, folderID)
	if err2 != nil {
		return 0, err2
	}

	// Skip purge check for very large mailboxes (> 10k emails) to avoid slowdown
	// User can manually trigger full purge if needed
	if len(localUIDs) > 10000 {
		return 0, nil
	}

	// Get all UIDs from server
	var serverUIDs, err = s.imap.GetAllUIDs(ctx)
	if err != nil {
		return 0, err
	}

	// Create a set for fast lookup
	var serverUIDSet = make(map[uint32]bool)
	for _, uid := range serverUIDs {
		serverUIDSet[uid] = true
	}

	// Find UIDs that exist locally but not on server
	var deletedUIDs []uint32
	for _, uid := range localUIDs {
		if !serverUIDSet[uid] {
			deletedUIDs = append(deletedUIDs, uid)
		}
	}

	if len(deletedUIDs) > 0 {
		if err := s.storage.MarkDeletedByUIDs(ctx, folderID, deletedUIDs); err != nil {
			return 0, err
		}
	}

	return len(deletedUIDs), nil
}

// PurgeSpecificUIDs checks if specific UIDs still exist on server and marks deleted ones
// Returns the list of UIDs that were marked as deleted
func (s *SyncService) PurgeSpecificUIDs(ctx context.Context, folderName string, uids []uint32) ([]uint32, error) {
	if len(uids) == 0 {
		return nil, nil
	}

	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	var folder, err = s.storage.GetFolderByName(ctx, account.ID, folderName)
	if err != nil {
		return nil, err
	}

	// Select mailbox
	if _, err := s.imap.SelectMailbox(ctx, folderName); err != nil {
		return nil, err
	}

	// Get all UIDs from server (we need to check against them)
	var serverUIDs, err2 = s.imap.GetAllUIDs(ctx)
	if err2 != nil {
		return nil, err2
	}

	// Create a set for fast lookup
	var serverUIDSet = make(map[uint32]bool)
	for _, uid := range serverUIDs {
		serverUIDSet[uid] = true
	}

	// Find which of the requested UIDs don't exist on server
	var deletedUIDs []uint32
	for _, uid := range uids {
		if !serverUIDSet[uid] {
			deletedUIDs = append(deletedUIDs, uid)
		}
	}

	// Mark them as deleted in storage
	if len(deletedUIDs) > 0 {
		log.Printf("[PurgeSpecificUIDs] marking %d emails as deleted in folder %s", len(deletedUIDs), folderName)
		if err := s.storage.MarkDeletedByUIDs(ctx, folder.ID, deletedUIDs); err != nil {
			return nil, err
		}
	}

	return deletedUIDs, nil
}

// SyncAll syncs all folders
func (s *SyncService) SyncAll(ctx context.Context) ([]ports.SyncResult, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	// Get all folders
	var folders, err = s.storage.GetFolders(ctx, account.ID)
	if err != nil {
		return nil, err
	}

	var results []ports.SyncResult
	for _, folder := range folders {
		var result, err = s.SyncFolder(ctx, folder.Name)
		if err != nil {
			results = append(results, ports.SyncResult{
				Errors: []error{err},
			})
		} else if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

// GetConfiguredFolders returns the folders configured for sync from app_settings
// Falls back to EssentialFolders if not configured
func GetConfiguredFolders(accountID int64) []string {
	var foldersJSON, err = storage.GetSetting(accountID, "sync_folders")
	if err != nil || foldersJSON == "" {
		return EssentialFolders
	}

	var folders []string
	if err := json.Unmarshal([]byte(foldersJSON), &folders); err != nil {
		return EssentialFolders
	}

	if len(folders) == 0 {
		return EssentialFolders
	}

	return folders
}

// SyncEssentialFolders syncs essential folders (INBOX, Sent, Trash) or configured folders
func (s *SyncService) SyncEssentialFolders(ctx context.Context) ([]ports.SyncResult, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	// Get folders to sync from settings, or use defaults
	var foldersToSync = GetConfiguredFolders(account.ID)

	var results []ports.SyncResult
	for _, folderName := range foldersToSync {
		var result, err = s.SyncFolder(ctx, folderName)
		if err != nil {
			// Skip folders that don't exist (e.g., non-Gmail accounts)
			continue
		}
		if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

// LoadFolders loads folders from IMAP and stores them
func (s *SyncService) LoadFolders(ctx context.Context) ([]ports.Folder, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	var mailboxes, err = s.imap.ListMailboxes(ctx)
	if err != nil {
		return nil, err
	}

	var folders []ports.Folder
	for _, mb := range mailboxes {
		var folder = &ports.Folder{
			Name:           mb.Name,
			TotalMessages:  int(mb.Messages),
			UnreadMessages: int(mb.Unseen),
		}

		// Store in database
		if err := s.storage.UpsertFolder(ctx, account.ID, folder); err != nil {
			return nil, err
		}

		folders = append(folders, *folder)

		// Cache locally
		s.mu.Lock()
		s.folders[mb.Name] = folder
		s.mu.Unlock()
	}

	return folders, nil
}

// SyncAttachmentsForFolder syncs attachment metadata for existing emails in a folder
// This is useful for backfilling attachments for emails that were synced before attachment support
func (s *SyncService) SyncAttachmentsForFolder(ctx context.Context, folderName string, limit int) (int, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return 0, fmt.Errorf("no account set")
	}

	// Get folder
	var folder, err = s.storage.GetFolderByName(ctx, account.ID, folderName)
	if err != nil {
		return 0, err
	}

	// Select mailbox
	if _, err := s.imap.SelectMailbox(ctx, folderName); err != nil {
		return 0, err
	}

	// Get emails that don't have attachments checked yet
	var emails, err2 = s.storage.GetEmails(ctx, folder.ID, limit)
	if err2 != nil {
		return 0, err2
	}

	var synced = 0
	for _, email := range emails {
		// Check if already has attachments in DB
		var existing, _ = s.storage.GetAttachmentsByEmail(ctx, email.ID)
		if len(existing) > 0 {
			continue // Already has attachments
		}

		// Fetch attachment metadata from IMAP
		var attachments, hasAttachments, fetchErr = s.imap.FetchAttachmentMetadata(ctx, email.UID)
		if fetchErr != nil {
			continue
		}

		if hasAttachments && len(attachments) > 0 {
			for _, att := range attachments {
				var contentID = att.ContentID
				if len(contentID) > 2 && contentID[0] == '<' && contentID[len(contentID)-1] == '>' {
					contentID = contentID[1 : len(contentID)-1]
				}

				var attachment = &ports.Attachment{
					EmailID:     email.ID,
					Filename:    att.Filename,
					ContentType: att.ContentType,
					ContentID:   contentID,
					Size:        att.Size,
					IsInline:    att.IsInline,
					PartNumber:  att.PartNumber,
					Encoding:    att.Encoding,
				}

				s.storage.UpsertAttachment(ctx, attachment)
			}
			// Update has_attachments flag on the email
			storage.UpdateHasAttachments(email.ID, true)
			synced++
		}
	}

	return synced, nil
}
