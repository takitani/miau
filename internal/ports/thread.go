package ports

import "context"

// ThreadService provides operations for email threading
// CRITICAL: This is the SINGLE SOURCE OF TRUTH for threading logic
// TUI and Desktop MUST use this service, NEVER implement threading directly
type ThreadService interface {
	// GetThread returns a complete thread with all messages for a given email ID
	// Messages are ordered DESC by date (newest first)
	GetThread(ctx context.Context, emailID int64) (*Thread, error)

	// GetThreadByID returns a thread by its thread_id
	GetThreadByID(ctx context.Context, threadID string) (*Thread, error)

	// GetThreadSummary returns thread metadata without full message content
	// Useful for inbox display
	GetThreadSummary(ctx context.Context, threadID string) (*ThreadSummary, error)

	// MarkThreadAsRead marks all messages in a thread as read
	MarkThreadAsRead(ctx context.Context, threadID string) error

	// MarkThreadAsUnread marks the most recent message in a thread as unread
	MarkThreadAsUnread(ctx context.Context, threadID string) error

	// CountThreadMessages returns the number of messages in a thread
	CountThreadMessages(ctx context.Context, threadID string) (int, error)
}
