// Package adapters provides implementations of the port interfaces.
package adapters

import (
	"context"
	"sync"

	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/imap"
	"github.com/opik/miau/internal/ports"
)

// IMAPAdapter implements ports.IMAPPort using the existing imap package
type IMAPAdapter struct {
	mu      sync.RWMutex
	client  *imap.Client
	account *config.Account
}

// NewIMAPAdapter creates a new IMAPAdapter
func NewIMAPAdapter(account *config.Account) *IMAPAdapter {
	return &IMAPAdapter{
		account: account,
	}
}

// Connect establishes connection to the IMAP server
func (a *IMAPAdapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	var client, err = imap.Connect(a.account)
	if err != nil {
		return err
	}
	a.client = client
	return nil
}

// Close closes the connection
func (a *IMAPAdapter) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.client == nil {
		return nil
	}
	return a.client.Close()
}

// IsConnected returns true if connected
func (a *IMAPAdapter) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.client != nil
}

// ListMailboxes lists all mailboxes
func (a *IMAPAdapter) ListMailboxes(ctx context.Context) ([]ports.MailboxInfo, error) {
	a.mu.RLock()
	var client = a.client
	a.mu.RUnlock()

	if client == nil {
		return nil, ErrNotConnected
	}

	var mailboxes, err = client.ListMailboxes()
	if err != nil {
		return nil, err
	}

	var result = make([]ports.MailboxInfo, len(mailboxes))
	for i, mb := range mailboxes {
		result[i] = ports.MailboxInfo{
			Name:     mb.Name,
			Messages: mb.Messages,
			Unseen:   mb.Unseen,
		}
	}
	return result, nil
}

// SelectMailbox selects a mailbox
func (a *IMAPAdapter) SelectMailbox(ctx context.Context, name string) (*ports.MailboxStatus, error) {
	a.mu.RLock()
	var client = a.client
	a.mu.RUnlock()

	if client == nil {
		return nil, ErrNotConnected
	}

	var data, err = client.SelectMailbox(name)
	if err != nil {
		return nil, err
	}

	return &ports.MailboxStatus{
		Name:        name,
		NumMessages: data.NumMessages,
		UIDNext:     data.UIDNext,
		UIDValidity: data.UIDValidity,
	}, nil
}

// FetchEmails fetches emails
func (a *IMAPAdapter) FetchEmails(ctx context.Context, limit int) ([]ports.IMAPEmail, error) {
	a.mu.RLock()
	var client = a.client
	a.mu.RUnlock()

	if client == nil {
		return nil, ErrNotConnected
	}

	var emails, err = client.FetchEmails(limit)
	if err != nil {
		return nil, err
	}

	return convertIMAPEmails(emails), nil
}

// FetchNewEmails fetches new emails since a UID
func (a *IMAPAdapter) FetchNewEmails(ctx context.Context, sinceUID uint32, limit int) ([]ports.IMAPEmail, error) {
	a.mu.RLock()
	var client = a.client
	a.mu.RUnlock()

	if client == nil {
		return nil, ErrNotConnected
	}

	var emails, err = client.FetchNewEmails(sinceUID, limit)
	if err != nil {
		return nil, err
	}

	return convertIMAPEmails(emails), nil
}

// FetchEmailRaw fetches raw email data
func (a *IMAPAdapter) FetchEmailRaw(ctx context.Context, uid uint32) ([]byte, error) {
	a.mu.RLock()
	var client = a.client
	a.mu.RUnlock()

	if client == nil {
		return nil, ErrNotConnected
	}

	return client.FetchEmailRaw(uid)
}

// FetchEmailBody fetches email body
func (a *IMAPAdapter) FetchEmailBody(ctx context.Context, uid uint32) (string, error) {
	a.mu.RLock()
	var client = a.client
	a.mu.RUnlock()

	if client == nil {
		return "", ErrNotConnected
	}

	return client.FetchEmailBody(uid)
}

// GetAllUIDs returns all UIDs in the current mailbox
func (a *IMAPAdapter) GetAllUIDs(ctx context.Context) ([]uint32, error) {
	a.mu.RLock()
	var client = a.client
	a.mu.RUnlock()

	if client == nil {
		return nil, ErrNotConnected
	}

	return client.GetAllUIDs()
}

// MarkAsRead marks an email as read
func (a *IMAPAdapter) MarkAsRead(ctx context.Context, uid uint32) error {
	a.mu.RLock()
	var client = a.client
	a.mu.RUnlock()

	if client == nil {
		return ErrNotConnected
	}

	return client.MarkAsRead(uid)
}

// MarkAsUnread marks an email as unread (not implemented in current imap package)
func (a *IMAPAdapter) MarkAsUnread(ctx context.Context, uid uint32) error {
	// The current imap package doesn't have MarkAsUnread
	// This would need to be implemented
	return nil
}

// Archive archives an email
func (a *IMAPAdapter) Archive(ctx context.Context, uid uint32) error {
	a.mu.RLock()
	var client = a.client
	a.mu.RUnlock()

	if client == nil {
		return ErrNotConnected
	}

	return client.ArchiveEmail(uid)
}

// MoveToFolder moves an email to a folder
func (a *IMAPAdapter) MoveToFolder(ctx context.Context, uid uint32, folder string) error {
	a.mu.RLock()
	var client = a.client
	a.mu.RUnlock()

	if client == nil {
		return ErrNotConnected
	}

	return client.MoveToFolder(uid, folder)
}

// Delete deletes an email (moves to trash)
func (a *IMAPAdapter) Delete(ctx context.Context, uid uint32) error {
	a.mu.RLock()
	var client = a.client
	a.mu.RUnlock()

	if client == nil {
		return ErrNotConnected
	}

	var trashFolder = client.GetTrashFolder()
	return client.TrashEmail(uid, trashFolder)
}

// GetTrashFolder returns the trash folder name
func (a *IMAPAdapter) GetTrashFolder() string {
	a.mu.RLock()
	var client = a.client
	a.mu.RUnlock()

	if client == nil {
		return "[Gmail]/Trash"
	}

	return client.GetTrashFolder()
}

// convertIMAPEmails converts imap.Email to ports.IMAPEmail
func convertIMAPEmails(emails []imap.Email) []ports.IMAPEmail {
	var result = make([]ports.IMAPEmail, len(emails))
	for i, e := range emails {
		result[i] = ports.IMAPEmail{
			UID:       e.UID,
			MessageID: e.MessageID,
			Subject:   e.Subject,
			FromName:  e.From,
			FromEmail: e.FromEmail,
			To:        e.To,
			Date:      e.Date,
			Seen:      e.Seen,
			Flagged:   e.Flagged,
			Size:      e.Size,
			BodyText:  e.BodyText,
		}
	}
	return result
}
