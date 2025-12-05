// Package ports defines the interfaces for the application's core services.
// These interfaces allow different implementations (TUI, Web, Desktop) to share
// the same business logic while providing their own UI layer.
package ports

import (
	"database/sql"
	"time"
)

// EmailMetadata contains basic email information for listing
type EmailMetadata struct {
	ID             int64
	UID            uint32
	MessageID      string
	Subject        string
	FromName       string
	FromEmail      string
	ToAddress      string
	Date           time.Time
	IsRead         bool
	IsStarred      bool
	IsReplied      bool
	HasAttachments bool
	Snippet        string
	Size           int64
	InReplyTo      string
	References     string
	ThreadID       string
	ThreadCount    int // Number of emails in thread (for grouped view)
}

// EmailContent contains full email content
type EmailContent struct {
	EmailMetadata
	FolderID       int64  // needed for IMAP fetch
	FolderName     string // folder name for IMAP select
	ToAddresses    string
	CcAddresses    string
	BodyText       string
	BodyHTML       string
	RawHeaders     string
	HasAttachments bool
	Attachments    []Attachment
}

// Attachment represents an email attachment
type Attachment struct {
	ID          int64
	EmailID     int64
	Filename    string
	ContentType string
	Size        int64
	ContentID   string // for inline images
	IsInline    bool
	PartNumber  string // MIME part number for IMAP fetch
	Encoding    string // content-transfer-encoding
	IsCached    bool   // whether content is cached in DB
	Data        []byte // populated only when downloaded
}

// Folder represents an email folder/mailbox
type Folder struct {
	ID             int64
	Name           string
	TotalMessages  int
	UnreadMessages int
	LastSync       *time.Time
}

// Draft represents a draft email
type Draft struct {
	ID              int64
	ToAddresses     string
	CcAddresses     string
	BccAddresses    string
	Subject         string
	BodyHTML        string
	BodyText        string
	Classification  string
	InReplyTo       string
	ReferenceIDs    string
	ReplyToEmailID  *int64
	Status          DraftStatus
	ScheduledSendAt *time.Time
	SentAt          *time.Time
	Source          string // "manual" or "ai"
	AIPrompt        string
	ErrorMessage    string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// DraftStatus represents the state of a draft
type DraftStatus string

const (
	DraftStatusDraft     DraftStatus = "draft"
	DraftStatusScheduled DraftStatus = "scheduled"
	DraftStatusSending   DraftStatus = "sending"
	DraftStatusSent      DraftStatus = "sent"
	DraftStatusCancelled DraftStatus = "cancelled"
	DraftStatusFailed    DraftStatus = "failed"
)

// SyncResult contains the result of a sync operation
type SyncResult struct {
	NewEmails     int
	DeletedEmails int
	LatestUID     uint32
	Errors        []error
}

// SendRequest contains all data needed to send an email
type SendRequest struct {
	To             []string
	Cc             []string
	Bcc            []string
	Subject        string
	BodyText       string
	BodyHTML       string
	InReplyTo      string
	ReferenceIDs   string
	ReplyToEmailID *int64
	Classification string // for Gmail API classification
	Attachments    []Attachment
}

// SendResult contains the result of sending an email
type SendResult struct {
	Success   bool
	MessageID string
	Error     error
	SentAt    time.Time
}

// BatchOperation represents a batch operation on emails
type BatchOperation struct {
	ID          int64
	Operation   BatchOpType
	Description string
	FilterQuery string
	EmailIDs    []int64
	EmailCount  int
	Status      BatchOpStatus
	CreatedAt   time.Time
	ExecutedAt  *time.Time
}

// BatchOpType defines the type of batch operation
type BatchOpType string

const (
	BatchOpArchive    BatchOpType = "archive"
	BatchOpDelete     BatchOpType = "delete"
	BatchOpMarkRead   BatchOpType = "mark_read"
	BatchOpMarkUnread BatchOpType = "mark_unread"
	BatchOpStar       BatchOpType = "star"
	BatchOpUnstar     BatchOpType = "unstar"
)

// BatchOpStatus defines the status of a batch operation
type BatchOpStatus string

const (
	BatchOpStatusPending   BatchOpStatus = "pending"
	BatchOpStatusConfirmed BatchOpStatus = "confirmed"
	BatchOpStatusCancelled BatchOpStatus = "cancelled"
	BatchOpStatusExecuted  BatchOpStatus = "executed"
)

// SearchResult contains search results
type SearchResult struct {
	Emails     []EmailMetadata
	TotalCount int
	Query      string
}

// IndexState represents the state of content indexing
type IndexState struct {
	Status         IndexStatus
	TotalEmails    int
	IndexedEmails  int
	LastIndexedUID int64
	Speed          int // emails per minute
	LastError      string
	StartedAt      *time.Time
	CompletedAt    *time.Time
}

// IndexStatus defines the status of the indexer
type IndexStatus string

const (
	IndexStatusIdle      IndexStatus = "idle"
	IndexStatusRunning   IndexStatus = "running"
	IndexStatusPaused    IndexStatus = "paused"
	IndexStatusCompleted IndexStatus = "completed"
	IndexStatusError     IndexStatus = "error"
)

// Alert represents a notification/alert to show to the user
type Alert struct {
	Type    AlertType
	Title   string
	Message string
	Data    interface{} // additional data specific to alert type
}

// AlertType defines the type of alert
type AlertType string

const (
	AlertTypeBounce  AlertType = "bounce"
	AlertTypeError   AlertType = "error"
	AlertTypeSuccess AlertType = "success"
	AlertTypeInfo    AlertType = "info"
)

// BounceInfo contains information about a bounced email
type BounceInfo struct {
	OriginalMessageID string
	Recipient         string
	Reason            string
	BouncedAt         time.Time
}

// ============================================================================
// THREADING TYPES
// ============================================================================

// Thread represents a complete email conversation/thread
type Thread struct {
	ThreadID     string
	Subject      string
	Participants []string
	MessageCount int
	Messages     []EmailContent // Ordered DESC by date (newest first)
	IsRead       bool           // All messages read?
}

// ThreadSummary contains thread metadata for inbox display
type ThreadSummary struct {
	ThreadID        string
	Subject         string
	LastSender      string
	LastSenderEmail string
	LastDate        time.Time
	MessageCount    int
	UnreadCount     int
	HasAttachments  bool
	Participants    []string
}

// ============================================================================
// ANALYTICS TYPES
// ============================================================================

// AnalyticsOverview contains general email statistics
type AnalyticsOverview struct {
	TotalEmails    int     `json:"totalEmails"`
	UnreadEmails   int     `json:"unreadEmails"`
	StarredEmails  int     `json:"starredEmails"`
	ArchivedEmails int     `json:"archivedEmails"`
	SentEmails     int     `json:"sentEmails"`
	DraftCount     int     `json:"draftCount"`
	StorageUsedMB  float64 `json:"storageUsedMb"`
}

// SenderStats contains statistics for a sender
type SenderStats struct {
	Email      string  `json:"email"`
	Name       string  `json:"name"`
	Count      int     `json:"count"`
	UnreadCount int    `json:"unreadCount"`
	Percentage float64 `json:"percentage"`
}

// HourlyStats contains email count per hour
type HourlyStats struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

// DailyStats contains email count per day
type DailyStats struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// WeekdayStats contains email count per weekday
type WeekdayStats struct {
	Weekday int    `json:"weekday"` // 0=Sunday, 6=Saturday
	Name    string `json:"name"`
	Count   int    `json:"count"`
}

// EmailTrends contains email volume trends over time
type EmailTrends struct {
	Daily   []DailyStats   `json:"daily"`
	Hourly  []HourlyStats  `json:"hourly"`
	Weekday []WeekdayStats `json:"weekday"`
}

// ResponseTimeStats contains response time statistics
type ResponseTimeStats struct {
	AvgResponseMinutes float64 `json:"avgResponseMinutes"`
	MedianMinutes      float64 `json:"medianMinutes"`
	ResponseRate       float64 `json:"responseRate"` // percentage of emails replied
}

// AnalyticsResult contains all analytics data
type AnalyticsResult struct {
	Overview     AnalyticsOverview   `json:"overview"`
	TopSenders   []SenderStats       `json:"topSenders"`
	Trends       EmailTrends         `json:"trends"`
	ResponseTime ResponseTimeStats   `json:"responseTime"`
	Period       string              `json:"period"` // "7d", "30d", "90d", "all"
	GeneratedAt  time.Time           `json:"generatedAt"`
}

// AccountInfo contains account information
type AccountInfo struct {
	ID        int64
	Email     string
	Name      string
	CreatedAt time.Time
}

// NullString is a helper for optional strings
type NullString struct {
	sql.NullString
}

// NewNullString creates a NullString from a string
func NewNullString(s string) NullString {
	if s == "" {
		return NullString{}
	}
	return NullString{sql.NullString{String: s, Valid: true}}
}

// String returns the string value or empty string
func (n NullString) String() string {
	if n.Valid {
		return n.NullString.String
	}
	return ""
}
