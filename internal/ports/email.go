package ports

import "context"

// EmailService defines operations for email management.
// This is the main interface that UI layers use to interact with emails.
type EmailService interface {
	// Folder operations
	GetFolders(ctx context.Context) ([]Folder, error)
	SelectFolder(ctx context.Context, name string) (*Folder, error)

	// Email listing
	GetEmails(ctx context.Context, folder string, limit int) ([]EmailMetadata, error)
	GetEmail(ctx context.Context, id int64) (*EmailContent, error)
	GetEmailByUID(ctx context.Context, folder string, uid uint32) (*EmailContent, error)

	// Email actions
	MarkAsRead(ctx context.Context, id int64, read bool) error
	MarkAsStarred(ctx context.Context, id int64, starred bool) error
	Archive(ctx context.Context, id int64) error
	Delete(ctx context.Context, id int64) error
	MoveToFolder(ctx context.Context, id int64, folder string) error

	// Sync
	Sync(ctx context.Context, folder string) (*SyncResult, error)
	GetLatestUID(ctx context.Context, folder string) (uint32, error)
}

// SendService defines operations for sending emails.
type SendService interface {
	// Send sends an email immediately
	Send(ctx context.Context, req *SendRequest) (*SendResult, error)

	// SendDraft sends a draft
	SendDraft(ctx context.Context, draftID int64) (*SendResult, error)

	// GetSignature returns the configured email signature
	GetSignature(ctx context.Context) (string, error)

	// LoadSignature pre-loads and caches the signature
	LoadSignature(ctx context.Context) error
}

// DraftService defines operations for draft management.
type DraftService interface {
	// CreateDraft creates a new draft
	CreateDraft(ctx context.Context, draft *Draft) (*Draft, error)

	// UpdateDraft updates an existing draft
	UpdateDraft(ctx context.Context, draft *Draft) error

	// GetDraft gets a draft by ID
	GetDraft(ctx context.Context, id int64) (*Draft, error)

	// ListDrafts lists all drafts
	ListDrafts(ctx context.Context) ([]Draft, error)

	// DeleteDraft deletes a draft
	DeleteDraft(ctx context.Context, id int64) error

	// ScheduleDraft schedules a draft for sending
	ScheduleDraft(ctx context.Context, id int64, sendAt *int64) error

	// CancelScheduledDraft cancels a scheduled draft
	CancelScheduledDraft(ctx context.Context, id int64) error
}

// SearchService defines operations for searching emails.
type SearchService interface {
	// Search performs a full-text search on emails
	Search(ctx context.Context, query string, limit int) (*SearchResult, error)

	// SearchInFolder searches within a specific folder
	SearchInFolder(ctx context.Context, folder, query string, limit int) (*SearchResult, error)

	// GetIndexState returns the current indexing state
	GetIndexState(ctx context.Context) (*IndexState, error)

	// StartIndexing starts/resumes background indexing
	StartIndexing(ctx context.Context) error

	// PauseIndexing pauses background indexing
	PauseIndexing(ctx context.Context) error

	// IndexEmail indexes a single email's content
	IndexEmail(ctx context.Context, emailID int64, content string) error
}

// BatchService defines operations for batch email operations.
type BatchService interface {
	// CreateBatchOp creates a pending batch operation
	CreateBatchOp(ctx context.Context, op *BatchOperation) (*BatchOperation, error)

	// GetPendingBatchOp returns the current pending batch operation if any
	GetPendingBatchOp(ctx context.Context) (*BatchOperation, error)

	// ConfirmBatchOp confirms and executes a batch operation
	ConfirmBatchOp(ctx context.Context, id int64) error

	// CancelBatchOp cancels a batch operation
	CancelBatchOp(ctx context.Context, id int64) error

	// GetBatchOpEmails returns the emails affected by a batch operation
	GetBatchOpEmails(ctx context.Context, id int64) ([]EmailMetadata, error)
}

// NotificationService defines operations for notifications and alerts.
type NotificationService interface {
	// CheckBounces checks for bounce notifications
	CheckBounces(ctx context.Context) ([]BounceInfo, error)

	// GetAlerts returns pending alerts
	GetAlerts(ctx context.Context) ([]Alert, error)

	// DismissAlert dismisses an alert
	DismissAlert(ctx context.Context, alertID string) error

	// TrackSentEmail tracks a sent email for bounce detection
	TrackSentEmail(ctx context.Context, messageID, recipient string) error
}

// SyncService defines operations for synchronization.
type SyncService interface {
	// Connect establishes connection to the email server
	Connect(ctx context.Context) error

	// Disconnect closes the connection
	Disconnect(ctx context.Context) error

	// IsConnected returns true if connected
	IsConnected() bool

	// SyncFolder syncs a specific folder
	SyncFolder(ctx context.Context, folder string) (*SyncResult, error)

	// SyncAll syncs all folders
	SyncAll(ctx context.Context) ([]SyncResult, error)

	// GetConnectionStatus returns the current connection status
	GetConnectionStatus() ConnectionStatus
}

// ConnectionStatus represents the connection state
type ConnectionStatus string

const (
	ConnectionStatusDisconnected ConnectionStatus = "disconnected"
	ConnectionStatusConnecting   ConnectionStatus = "connecting"
	ConnectionStatusConnected    ConnectionStatus = "connected"
	ConnectionStatusError        ConnectionStatus = "error"
)

// AIService defines operations for AI-assisted features.
type AIService interface {
	// GenerateReply generates a reply draft using AI
	GenerateReply(ctx context.Context, emailID int64, prompt string) (*Draft, error)

	// Summarize summarizes an email or thread
	Summarize(ctx context.Context, emailID int64) (string, error)

	// ExtractActions extracts action items from an email
	ExtractActions(ctx context.Context, emailID int64) ([]string, error)

	// ClassifyEmail classifies an email (spam, important, etc.)
	ClassifyEmail(ctx context.Context, emailID int64) (string, error)
}
