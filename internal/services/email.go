package services

import (
	"context"
	"fmt"
	"log"
	"sync"

	emailparser "github.com/opik/miau/internal/email"
	"github.com/opik/miau/internal/ports"
)

// EmailService implements ports.EmailService
type EmailService struct {
	mu      sync.RWMutex
	imap    ports.IMAPPort
	storage ports.StoragePort
	events  ports.EventBus
	account *ports.AccountInfo
	folder  *ports.Folder
}

// NewEmailService creates a new EmailService
func NewEmailService(imap ports.IMAPPort, storage ports.StoragePort, events ports.EventBus) *EmailService {
	return &EmailService{
		imap:    imap,
		storage: storage,
		events:  events,
	}
}

// SetAccount sets the current account
func (s *EmailService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// GetFolders returns all folders for the current account
func (s *EmailService) GetFolders(ctx context.Context) ([]ports.Folder, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	return s.storage.GetFolders(ctx, account.ID)
}

// SelectFolder selects a folder for subsequent operations
func (s *EmailService) SelectFolder(ctx context.Context, name string) (*ports.Folder, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	var folder, err = s.storage.GetFolderByName(ctx, account.ID, name)
	if err != nil {
		return nil, err
	}

	// Select on IMAP
	var _, err2 = s.imap.SelectMailbox(ctx, name)
	if err2 != nil {
		return nil, fmt.Errorf("failed to select mailbox: %w", err2)
	}

	s.mu.Lock()
	s.folder = folder
	s.mu.Unlock()

	return folder, nil
}

// GetEmails returns emails from the current folder
func (s *EmailService) GetEmails(ctx context.Context, folder string, limit int) ([]ports.EmailMetadata, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	var f, err = s.storage.GetFolderByName(ctx, account.ID, folder)
	if err != nil {
		return nil, err
	}

	return s.storage.GetEmails(ctx, f.ID, limit)
}

// GetEmail returns a single email by ID with full content
func (s *EmailService) GetEmail(ctx context.Context, id int64) (*ports.EmailContent, error) {
	var email, err = s.storage.GetEmail(ctx, id)
	if err != nil {
		return nil, err
	}

	// Load attachments from database
	var attachments, attErr = s.storage.GetAttachmentsByEmail(ctx, id)
	if attErr == nil && len(attachments) > 0 {
		email.Attachments = attachments
		email.HasAttachments = true
	}

	// Select the correct folder in IMAP before fetching
	if email.FolderName != "" {
		if _, err := s.imap.SelectMailbox(ctx, email.FolderName); err != nil {
			// Log but continue - the mailbox might already be selected
			log.Printf("[GetEmail] Failed to select mailbox %s: %v", email.FolderName, err)
		}
	}

	// If no attachments from DB but has_attachments flag is set, fetch from IMAP
	if len(email.Attachments) == 0 && email.HasAttachments {
		var imapAtts, hasAtts, attErr = s.imap.FetchAttachmentMetadata(ctx, email.UID)
		if attErr == nil && hasAtts {
			for _, att := range imapAtts {
				var contentID = att.ContentID
				if len(contentID) > 2 && contentID[0] == '<' && contentID[len(contentID)-1] == '>' {
					contentID = contentID[1 : len(contentID)-1]
				}
				email.Attachments = append(email.Attachments, ports.Attachment{
					EmailID:     id,
					Filename:    att.Filename,
					ContentType: att.ContentType,
					ContentID:   contentID,
					Size:        att.Size,
					IsInline:    att.IsInline,
					PartNumber:  att.PartNumber,
					Encoding:    att.Encoding,
				})
			}
		}
	}

	// If body is empty, fetch from IMAP
	if email.BodyText == "" && email.BodyHTML == "" {
		var rawData, fetchErr = s.imap.FetchEmailRaw(ctx, email.UID)
		if fetchErr != nil {
			log.Printf("[GetEmail] Failed to fetch email body: %v", fetchErr)
			return email, nil // Return without body
		}

		// Parse email content
		var parsed, _ = emailparser.Parse(rawData)
		if parsed != nil {
			email.BodyText = parsed.TextBody
			email.BodyHTML = parsed.HTMLBody
		}
	}

	return email, nil
}

// GetEmailByUID returns an email by UID
func (s *EmailService) GetEmailByUID(ctx context.Context, folder string, uid uint32) (*ports.EmailContent, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	var f, err = s.storage.GetFolderByName(ctx, account.ID, folder)
	if err != nil {
		return nil, err
	}

	return s.storage.GetEmailByUID(ctx, f.ID, uid)
}

// MarkAsRead marks an email as read/unread
func (s *EmailService) MarkAsRead(ctx context.Context, id int64, read bool) error {
	// Get email to get UID
	var email, err = s.storage.GetEmail(ctx, id)
	if err != nil {
		return err
	}

	// Mark on IMAP server
	if read {
		if err := s.imap.MarkAsRead(ctx, email.UID); err != nil {
			return err
		}
	} else {
		if err := s.imap.MarkAsUnread(ctx, email.UID); err != nil {
			return err
		}
	}

	// Mark in storage
	if err := s.storage.MarkAsRead(ctx, id, read); err != nil {
		return err
	}

	s.events.Publish(ports.EmailReadEvent{
		BaseEvent: ports.NewBaseEvent(ports.EventTypeEmailRead),
		EmailID:   id,
		Read:      read,
	})

	return nil
}

// MarkAsStarred marks an email as starred/unstarred
func (s *EmailService) MarkAsStarred(ctx context.Context, id int64, starred bool) error {
	return s.storage.MarkAsStarred(ctx, id, starred)
}

// Archive archives an email
func (s *EmailService) Archive(ctx context.Context, id int64) error {
	var email, err = s.storage.GetEmail(ctx, id)
	if err != nil {
		return err
	}

	// Archive on IMAP
	if err := s.imap.Archive(ctx, email.UID); err != nil {
		return err
	}

	// Mark as archived in storage
	return s.storage.MarkAsArchived(ctx, id, true)
}

// Delete marks an email as deleted
func (s *EmailService) Delete(ctx context.Context, id int64) error {
	var email, err = s.storage.GetEmail(ctx, id)
	if err != nil {
		return err
	}

	// Delete on IMAP
	if err := s.imap.Delete(ctx, email.UID); err != nil {
		return err
	}

	// Mark as deleted in storage
	return s.storage.MarkAsDeleted(ctx, id, true)
}

// MoveToFolder moves an email to another folder
func (s *EmailService) MoveToFolder(ctx context.Context, id int64, folder string) error {
	var email, err = s.storage.GetEmail(ctx, id)
	if err != nil {
		return err
	}

	return s.imap.MoveToFolder(ctx, email.UID, folder)
}

// Sync syncs emails for a folder
func (s *EmailService) Sync(ctx context.Context, folder string) (*ports.SyncResult, error) {
	// This is delegated to SyncService
	return nil, fmt.Errorf("use SyncService for sync operations")
}

// GetLatestUID returns the latest UID for a folder
func (s *EmailService) GetLatestUID(ctx context.Context, folder string) (uint32, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return 0, fmt.Errorf("no account set")
	}

	var f, err = s.storage.GetFolderByName(ctx, account.ID, folder)
	if err != nil {
		return 0, err
	}

	return s.storage.GetLatestUID(ctx, f.ID)
}
