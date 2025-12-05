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

	// Multi-selection operations (direct execution)
	ArchiveSelected(ctx context.Context, emailIDs []int64) error
	DeleteSelected(ctx context.Context, emailIDs []int64) error
	MarkReadSelected(ctx context.Context, emailIDs []int64, read bool) error
	StarSelected(ctx context.Context, emailIDs []int64, starred bool) error
	ForwardSelected(ctx context.Context, emailIDs []int64, forwardTo string) error
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

// SyncConfig contains sync configuration parameters
type SyncConfig struct {
	// Initial sync settings
	InitialSyncDays     int // Days to fetch on first sync (default: 30)
	InitialMaxPerFolder int // Max emails per folder on first sync (default: 500)

	// Incremental sync settings
	IncrementalBatchSize int // Emails per batch (default: 100)

	// Purge settings
	PurgeEnabled       bool // Enable purge job (default: true)
	PurgeMaxFolderSize int  // Skip purge if folder > this size (default: 10000)

	// Folders to skip during sync
	SkipFolders []string // e.g., "[Gmail]/All Mail", "[Gmail]/Spam"
}

// DefaultSyncConfig returns the default sync configuration
func DefaultSyncConfig() SyncConfig {
	return SyncConfig{
		InitialSyncDays:      30,
		InitialMaxPerFolder:  500,
		IncrementalBatchSize: 100,
		PurgeEnabled:         true,
		PurgeMaxFolderSize:   10000,
		SkipFolders:          []string{"[Gmail]/All Mail", "[Gmail]/Spam"},
	}
}

// SyncService defines operations for synchronization.
type SyncService interface {
	// Connect establishes connection to the email server
	Connect(ctx context.Context) error

	// Disconnect closes the connection
	Disconnect(ctx context.Context) error

	// IsConnected returns true if connected
	IsConnected() bool

	// SyncFolder syncs a specific folder (incremental, uses batch operations)
	SyncFolder(ctx context.Context, folder string) (*SyncResult, error)

	// SyncAll syncs all folders
	SyncAll(ctx context.Context) ([]SyncResult, error)

	// SyncEssentialFolders syncs essential folders (INBOX, Sent, Trash)
	SyncEssentialFolders(ctx context.Context) ([]SyncResult, error)

	// InitialSync performs optimized first-time sync for a folder
	// Uses date-based search (last N days) instead of full UID scan
	InitialSync(ctx context.Context, folder string) (*SyncResult, error)

	// InitialSyncEssentialFolders performs optimized first-time sync for essential folders
	InitialSyncEssentialFolders(ctx context.Context) ([]SyncResult, error)

	// PurgeDeletedEmails checks for deleted emails and marks them locally
	// This is a separate job that should run periodically, not on every sync
	PurgeDeletedEmails(ctx context.Context, folder string) (int, error)

	// GetConnectionStatus returns the current connection status
	GetConnectionStatus() ConnectionStatus

	// GetSyncConfig returns the current sync configuration
	GetSyncConfig() SyncConfig

	// SetSyncConfig sets the sync configuration
	SetSyncConfig(config SyncConfig)
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

// AnalyticsService defines operations for email analytics and statistics.
type AnalyticsService interface {
	// GetAnalytics returns comprehensive analytics for a time period
	// period can be: "7d", "30d", "90d", "all"
	GetAnalytics(ctx context.Context, period string) (*AnalyticsResult, error)

	// GetOverview returns basic email statistics
	GetOverview(ctx context.Context) (*AnalyticsOverview, error)

	// GetTopSenders returns top email senders
	GetTopSenders(ctx context.Context, limit int, period string) ([]SenderStats, error)

	// GetEmailTrends returns email volume trends
	GetEmailTrends(ctx context.Context, period string) (*EmailTrends, error)

	// GetResponseStats returns response time statistics
	GetResponseStats(ctx context.Context) (*ResponseTimeStats, error)
}

// AttachmentService defines operations for email attachments.
type AttachmentService interface {
	// GetAttachments returns all attachments for an email
	GetAttachments(ctx context.Context, emailID int64) ([]Attachment, error)

	// GetAttachment returns a single attachment by ID
	GetAttachment(ctx context.Context, id int64) (*Attachment, error)

	// Download downloads an attachment and returns its content
	Download(ctx context.Context, id int64) ([]byte, error)

	// SaveToFile downloads an attachment and saves it to a file
	SaveToFile(ctx context.Context, id int64, path string) error

	// DownloadByPart downloads an attachment by email ID and MIME part number
	DownloadByPart(ctx context.Context, emailID int64, partNumber string) ([]byte, error)

	// SaveToFileByPart downloads by part number and saves to a file
	SaveToFileByPart(ctx context.Context, emailID int64, partNumber, path string) error
}
