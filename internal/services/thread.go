package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/storage"
)

// ThreadService handles email threading operations
// CRITICAL: This is the SINGLE SOURCE OF TRUTH for threading logic
// TUI and Desktop MUST use this service, NEVER implement threading directly
type ThreadService struct {
	mu      sync.RWMutex
	storage ports.StoragePort
	events  ports.EventBus
	account *ports.AccountInfo
}

// NewThreadService creates a new ThreadService
func NewThreadService(storage ports.StoragePort, events ports.EventBus) *ThreadService {
	return &ThreadService{
		storage: storage,
		events:  events,
	}
}

// SetAccount sets the current account
func (s *ThreadService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// GetThread returns a complete thread with all messages for a given email ID
// Messages are ordered DESC by date (newest first)
func (s *ThreadService) GetThread(ctx context.Context, emailID int64) (*ports.Thread, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	// Get all emails in the thread
	var emails, err = storage.GetThreadForEmail(emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	if len(emails) == 0 {
		return nil, fmt.Errorf("email not found")
	}

	// Convert storage.Email to ports.EmailContent
	var messages = make([]ports.EmailContent, len(emails))
	for i, e := range emails {
		messages[i] = ports.EmailContent{
			EmailMetadata: ports.EmailMetadata{
				ID:         e.ID,
				UID:        e.UID,
				MessageID:  e.MessageID.String,
				Subject:    e.Subject,
				FromName:   e.FromName,
				FromEmail:  e.FromEmail,
				Date:       e.Date.Time,
				IsRead:     e.IsRead,
				IsStarred:  e.IsStarred,
				IsReplied:  e.IsReplied,
				Snippet:    e.Snippet,
				Size:       e.Size,
				InReplyTo:  e.InReplyTo.String,
				References: e.References.String,
				ThreadID:   e.ThreadID.String,
			},
			BodyText:       e.BodyText,
			BodyHTML:       e.BodyHTML,
			RawHeaders:     e.RawHeaders,
			HasAttachments: e.HasAttachments,
		}
	}

	// Get thread metadata
	var threadID = emails[0].ThreadID.String
	var subject = emails[0].Subject

	// Get participants
	var participants []string
	if threadID != "" {
		participants, _ = storage.GetThreadParticipants(threadID, account.ID)
	}

	// Check if all messages are read
	var allRead = true
	for _, e := range emails {
		if !e.IsRead {
			allRead = false
			break
		}
	}

	var thread = &ports.Thread{
		ThreadID:     threadID,
		Subject:      subject,
		Participants: participants,
		MessageCount: len(emails),
		Messages:     messages,
		IsRead:       allRead,
	}

	return thread, nil
}

// GetThreadByID returns a thread by its thread_id
func (s *ThreadService) GetThreadByID(ctx context.Context, threadID string) (*ports.Thread, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	// Get all emails in the thread
	var emails, err = storage.GetThreadEmails(threadID, account.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	if len(emails) == 0 {
		return nil, fmt.Errorf("thread not found")
	}

	// Convert storage.Email to ports.EmailContent
	var messages = make([]ports.EmailContent, len(emails))
	for i, e := range emails {
		messages[i] = ports.EmailContent{
			EmailMetadata: ports.EmailMetadata{
				ID:         e.ID,
				UID:        e.UID,
				MessageID:  e.MessageID.String,
				Subject:    e.Subject,
				FromName:   e.FromName,
				FromEmail:  e.FromEmail,
				Date:       e.Date.Time,
				IsRead:     e.IsRead,
				IsStarred:  e.IsStarred,
				IsReplied:  e.IsReplied,
				Snippet:    e.Snippet,
				Size:       e.Size,
				InReplyTo:  e.InReplyTo.String,
				References: e.References.String,
				ThreadID:   e.ThreadID.String,
			},
			BodyText:       e.BodyText,
			BodyHTML:       e.BodyHTML,
			RawHeaders:     e.RawHeaders,
			HasAttachments: e.HasAttachments,
		}
	}

	// Get participants
	var participants []string
	participants, _ = storage.GetThreadParticipants(threadID, account.ID)

	// Check if all messages are read
	var allRead = true
	for _, e := range emails {
		if !e.IsRead {
			allRead = false
			break
		}
	}

	var thread = &ports.Thread{
		ThreadID:     threadID,
		Subject:      emails[0].Subject,
		Participants: participants,
		MessageCount: len(emails),
		Messages:     messages,
		IsRead:       allRead,
	}

	return thread, nil
}

// GetThreadSummary returns thread metadata without full message content
// Useful for inbox display
func (s *ThreadService) GetThreadSummary(ctx context.Context, threadID string) (*ports.ThreadSummary, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	// Get thread emails (we need at least metadata)
	var emails, err = storage.GetThreadEmails(threadID, account.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	if len(emails) == 0 {
		return nil, fmt.Errorf("thread not found")
	}

	// Most recent email is first (DESC order)
	var latest = emails[0]

	// Count unread
	var unreadCount = 0
	var hasAttachments = false
	for _, e := range emails {
		if !e.IsRead {
			unreadCount++
		}
		if e.HasAttachments {
			hasAttachments = true
		}
	}

	// Get participants
	var participants []string
	participants, _ = storage.GetThreadParticipants(threadID, account.ID)

	var summary = &ports.ThreadSummary{
		ThreadID:       threadID,
		Subject:        latest.Subject,
		LastSender:     latest.FromName,
		LastSenderEmail: latest.FromEmail,
		LastDate:       latest.Date.Time,
		MessageCount:   len(emails),
		UnreadCount:    unreadCount,
		HasAttachments: hasAttachments,
		Participants:   participants,
	}

	return summary, nil
}

// MarkThreadAsRead marks all messages in a thread as read
func (s *ThreadService) MarkThreadAsRead(ctx context.Context, threadID string) error {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return fmt.Errorf("no account set")
	}

	// Get all emails in thread
	var emails, err = storage.GetThreadEmails(threadID, account.ID)
	if err != nil {
		return fmt.Errorf("failed to get thread: %w", err)
	}

	// Mark each as read
	for _, email := range emails {
		if !email.IsRead {
			if err := storage.MarkAsRead(email.ID); err != nil {
				return fmt.Errorf("failed to mark email %d as read: %w", email.ID, err)
			}
		}
	}

	// Publish event
	s.events.Publish(ports.ThreadMarkedReadEvent{
		BaseEvent: ports.NewBaseEvent(ports.EventTypeThreadMarkedRead),
		ThreadID:  threadID,
		Count:     len(emails),
	})

	return nil
}

// MarkThreadAsUnread marks the most recent message in a thread as unread
func (s *ThreadService) MarkThreadAsUnread(ctx context.Context, threadID string) error {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return fmt.Errorf("no account set")
	}

	// Get thread emails
	var emails, err = storage.GetThreadEmails(threadID, account.ID)
	if err != nil {
		return fmt.Errorf("failed to get thread: %w", err)
	}

	if len(emails) == 0 {
		return fmt.Errorf("thread not found")
	}

	// Mark most recent (first) as unread
	var latest = emails[0]
	if err := storage.MarkAsUnread(latest.ID); err != nil {
		return fmt.Errorf("failed to mark email as unread: %w", err)
	}

	// Publish event
	s.events.Publish(ports.ThreadMarkedUnreadEvent{
		BaseEvent: ports.NewBaseEvent(ports.EventTypeThreadMarkedUnread),
		ThreadID:  threadID,
	})

	return nil
}

// CountThreadMessages returns the number of messages in a thread
func (s *ThreadService) CountThreadMessages(ctx context.Context, threadID string) (int, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return 0, fmt.Errorf("no account set")
	}

	return storage.CountThreadEmails(threadID, account.ID)
}
