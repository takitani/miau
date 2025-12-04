package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/storage"
)

// SyncService implements ports.SyncService
type SyncService struct {
	mu       sync.RWMutex
	imap     ports.IMAPPort
	storage  ports.StoragePort
	events   ports.EventBus
	status   ports.ConnectionStatus
	account  *ports.AccountInfo
	folders  map[string]*ports.Folder
}

// NewSyncService creates a new SyncService
func NewSyncService(imap ports.IMAPPort, storage ports.StoragePort, events ports.EventBus) *SyncService {
	return &SyncService{
		imap:    imap,
		storage: storage,
		events:  events,
		status:  ports.ConnectionStatusDisconnected,
		folders: make(map[string]*ports.Folder),
	}
}

// SetAccount sets the current account for sync operations
func (s *SyncService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
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

// SyncFolder syncs a specific folder
func (s *SyncService) SyncFolder(ctx context.Context, folderName string) (*ports.SyncResult, error) {
	s.mu.RLock()
	var account = s.account
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

	// Fetch new emails
	var newEmails, err3 = s.imap.FetchNewEmails(ctx, latestUID, 100)
	if err3 != nil {
		storage.LogSyncComplete(syncID, 0, 0, err3)
		s.events.Publish(ports.SyncErrorEvent{
			BaseEvent: ports.NewBaseEvent(ports.EventTypeSyncError),
			Folder:    folderName,
			Error:     err3,
		})
		return nil, fmt.Errorf("failed to fetch new emails: %w", err3)
	}

	var result = &ports.SyncResult{
		LatestUID: latestUID,
	}

	// Store new emails
	for _, email := range newEmails {
		var content = &ports.EmailContent{
			EmailMetadata: ports.EmailMetadata{
				UID:       email.UID,
				MessageID: email.MessageID,
				Subject:   email.Subject,
				FromName:  email.FromName,
				FromEmail: email.FromEmail,
				ToAddress: email.To,
				Date:      email.Date,
				IsRead:    email.Seen,
				IsStarred: email.Flagged,
				Size:      email.Size,
			},
			BodyText: email.BodyText,
		}

		if err := s.storage.UpsertEmail(ctx, account.ID, folder.ID, content); err != nil {
			result.Errors = append(result.Errors, err)
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

	// Purge deleted emails (sync deletions from server)
	var deletedCount, err4 = s.purgeDeleted(ctx, folder.ID)
	if err4 == nil {
		result.DeletedEmails = deletedCount
	}

	// Conta novos emails desde o último sync (baseado em created_at no DB)
	var newCount, _ = storage.CountNewEmailsSinceLastSync(account.ID, folder.ID)
	result.NewEmails = newCount

	// Registra conclusão do sync
	storage.LogSyncComplete(syncID, newCount, deletedCount, nil)

	s.events.Publish(ports.SyncCompletedEvent{
		BaseEvent: ports.NewBaseEvent(ports.EventTypeSyncCompleted),
		Folder:    folderName,
		Result:    result,
	})

	return result, nil
}

// purgeDeleted marks emails as deleted that were removed from server
func (s *SyncService) purgeDeleted(ctx context.Context, folderID int64) (int, error) {
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

	// Get all UIDs from storage
	var localUIDs, err2 = s.storage.GetAllUIDs(ctx, folderID)
	if err2 != nil {
		return 0, err2
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
