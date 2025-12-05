package ports

import (
	"context"
	"time"
)

// StoragePort defines the interface for data persistence.
// This abstracts the database layer, allowing different implementations.
type StoragePort interface {
	// Account operations
	GetOrCreateAccount(ctx context.Context, email, name string) (*AccountInfo, error)
	GetAccount(ctx context.Context, id int64) (*AccountInfo, error)

	// Folder operations
	UpsertFolder(ctx context.Context, accountID int64, folder *Folder) error
	GetFolders(ctx context.Context, accountID int64) ([]Folder, error)
	GetFolderByName(ctx context.Context, accountID int64, name string) (*Folder, error)
	UpdateFolderStats(ctx context.Context, folderID int64, total, unread int) error

	// Email operations
	UpsertEmail(ctx context.Context, accountID, folderID int64, email *EmailContent) error
	GetEmails(ctx context.Context, folderID int64, limit int) ([]EmailMetadata, error)
	GetEmail(ctx context.Context, id int64) (*EmailContent, error)
	GetEmailByUID(ctx context.Context, folderID int64, uid uint32) (*EmailContent, error)
	GetLatestUID(ctx context.Context, folderID int64) (uint32, error)
	GetAllUIDs(ctx context.Context, folderID int64) ([]uint32, error)

	// Email content updates
	UpdateEmailBody(ctx context.Context, id int64, bodyText, bodyHTML string) error

	// Email status updates
	MarkAsRead(ctx context.Context, id int64, read bool) error
	MarkAsStarred(ctx context.Context, id int64, starred bool) error
	MarkAsArchived(ctx context.Context, id int64, archived bool) error
	MarkAsDeleted(ctx context.Context, id int64, deleted bool) error
	MarkAsReplied(ctx context.Context, id int64, replied bool) error

	// Bulk operations
	MarkDeletedByUIDs(ctx context.Context, folderID int64, uids []uint32) error
	BulkMarkAsRead(ctx context.Context, ids []int64, read bool) error
	BulkMarkAsArchived(ctx context.Context, ids []int64) error
	BulkMarkAsDeleted(ctx context.Context, ids []int64) error

	// Search
	SearchEmails(ctx context.Context, accountID int64, query string, limit int) ([]EmailMetadata, error)
	SearchEmailsInFolder(ctx context.Context, folderID int64, query string, limit int) ([]EmailMetadata, error)

	// Draft operations
	CreateDraft(ctx context.Context, accountID int64, draft *Draft) (*Draft, error)
	UpdateDraft(ctx context.Context, draft *Draft) error
	GetDraft(ctx context.Context, id int64) (*Draft, error)
	GetDrafts(ctx context.Context, accountID int64) ([]Draft, error)
	GetPendingDrafts(ctx context.Context, accountID int64) ([]Draft, error)
	DeleteDraft(ctx context.Context, id int64) error
	UpdateDraftStatus(ctx context.Context, id int64, status DraftStatus) error

	// Batch operations
	CreateBatchOp(ctx context.Context, accountID int64, op *BatchOperation) (*BatchOperation, error)
	GetPendingBatchOp(ctx context.Context, accountID int64) (*BatchOperation, error)
	UpdateBatchOpStatus(ctx context.Context, id int64, status BatchOpStatus) error
	ExecuteBatchOp(ctx context.Context, id int64) error

	// Sent email tracking
	TrackSentEmail(ctx context.Context, accountID int64, messageID, to, subject string) error
	GetRecentSentEmails(ctx context.Context, accountID int64, since time.Duration) ([]SentEmailTrack, error)

	// Index state
	GetIndexState(ctx context.Context, accountID int64) (*IndexState, error)
	UpdateIndexState(ctx context.Context, accountID int64, state *IndexState) error

	// Settings
	GetSetting(ctx context.Context, accountID int64, key string) (string, error)
	SetSetting(ctx context.Context, accountID int64, key, value string) error

	// Analytics
	GetAnalyticsOverview(ctx context.Context, accountID int64) (*AnalyticsOverview, error)
	GetTopSenders(ctx context.Context, accountID int64, limit int, sinceDays int) ([]SenderStats, error)
	GetEmailCountByHour(ctx context.Context, accountID int64, sinceDays int) ([]HourlyStats, error)
	GetEmailCountByDay(ctx context.Context, accountID int64, sinceDays int) ([]DailyStats, error)
	GetEmailCountByWeekday(ctx context.Context, accountID int64, sinceDays int) ([]WeekdayStats, error)
	GetResponseStats(ctx context.Context, accountID int64) (*ResponseTimeStats, error)

	// Attachments
	GetAttachmentsByEmail(ctx context.Context, emailID int64) ([]Attachment, error)
	GetAttachment(ctx context.Context, id int64) (*Attachment, error)
	GetAttachmentContent(ctx context.Context, id int64) ([]byte, error)
	CacheAttachmentContent(ctx context.Context, id int64, content []byte) error
	UpsertAttachment(ctx context.Context, attachment *Attachment) (int64, error)

	// Undo/Redo operations
	SaveOperation(ctx context.Context, op *OperationRecord) error
	RemoveOperation(ctx context.Context, accountID int64, stackType, data string) error
	GetOperations(ctx context.Context, accountID int64, stackType string) ([]OperationRecord, error)
	ClearOperationsHistory(ctx context.Context, accountID int64) error
}

// SentEmailTrack represents a tracked sent email for bounce detection
type SentEmailTrack struct {
	MessageID string
	To        string
	Subject   string
	SentAt    time.Time
}

// AttachmentInfo contains attachment metadata from IMAP BODYSTRUCTURE
type AttachmentInfo struct {
	PartNumber  string
	Filename    string
	ContentType string
	ContentID   string
	Encoding    string
	Size        int64
	IsInline    bool
	Charset     string
}

// IMAPPort defines the interface for IMAP operations.
// This abstracts the IMAP client implementation.
type IMAPPort interface {
	// Connection
	Connect(ctx context.Context) error
	Close() error
	IsConnected() bool

	// Mailbox operations
	ListMailboxes(ctx context.Context) ([]MailboxInfo, error)
	SelectMailbox(ctx context.Context, name string) (*MailboxStatus, error)

	// Email fetching (legacy - prefer batch methods)
	FetchEmails(ctx context.Context, limit int) ([]IMAPEmail, error)
	FetchNewEmails(ctx context.Context, sinceUID uint32, limit int) ([]IMAPEmail, error)
	FetchEmailRaw(ctx context.Context, uid uint32) ([]byte, error)
	FetchEmailBody(ctx context.Context, uid uint32) (string, error)
	GetAllUIDs(ctx context.Context) ([]uint32, error)

	// Batch email fetching (optimized - 1 request for N emails with attachments)
	SearchSince(ctx context.Context, sinceDate time.Time) ([]uint32, error)
	FetchEmailsBatch(ctx context.Context, uids []uint32) ([]IMAPEmail, error)
	FetchNewEmailsBatch(ctx context.Context, sinceUID uint32, limit int) ([]IMAPEmail, error)
	FetchEmailsSinceDateBatch(ctx context.Context, sinceDays int, limit int) ([]IMAPEmail, error)

	// Attachments
	FetchAttachmentMetadata(ctx context.Context, uid uint32) ([]AttachmentInfo, bool, error)
	FetchAttachmentPart(ctx context.Context, uid uint32, partNumber string) ([]byte, error)

	// Email actions
	MarkAsRead(ctx context.Context, uid uint32) error
	MarkAsUnread(ctx context.Context, uid uint32) error
	Archive(ctx context.Context, uid uint32) error
	MoveToFolder(ctx context.Context, uid uint32, folder string) error
	Delete(ctx context.Context, uid uint32) error
	Undelete(ctx context.Context, uid uint32) error

	// Utility
	GetTrashFolder() string
}

// MailboxInfo contains basic mailbox information
type MailboxInfo struct {
	Name     string
	Messages uint32
	Unseen   uint32
}

// MailboxStatus contains detailed mailbox status
type MailboxStatus struct {
	Name        string
	NumMessages uint32
	NumUnseen   uint32
	UIDNext     uint32
	UIDValidity uint32
}

// IMAPEmail represents an email fetched from IMAP
type IMAPEmail struct {
	UID        uint32
	MessageID  string
	Subject    string
	FromName   string
	FromEmail  string
	To         string
	Date       time.Time
	Seen       bool
	Flagged    bool
	Size       int64
	BodyText   string
	InReplyTo  string
	References string
	// Attachment metadata (populated by batch fetch methods)
	HasAttachments bool
	Attachments    []AttachmentInfo
}

// SMTPPort defines the interface for SMTP operations.
type SMTPPort interface {
	// Send sends an email via SMTP
	Send(ctx context.Context, req *SendRequest) (*SendResult, error)
}

// GmailAPIPort defines the interface for Gmail API operations.
type GmailAPIPort interface {
	// Send sends an email via Gmail API
	Send(ctx context.Context, req *SendRequest) (*SendResult, error)

	// GetSignature retrieves the user's signature
	GetSignature(ctx context.Context) (string, error)

	// Archive archives an email
	Archive(ctx context.Context, messageID string) error
}
