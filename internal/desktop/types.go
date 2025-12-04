package desktop

import (
	"time"
)

// EmailDTO represents an email for the frontend
type EmailDTO struct {
	ID             int64     `json:"id"`
	UID            uint32    `json:"uid"`
	Subject        string    `json:"subject"`
	FromName       string    `json:"fromName"`
	FromEmail      string    `json:"fromEmail"`
	Date           time.Time `json:"date"`
	IsRead         bool      `json:"isRead"`
	IsStarred      bool      `json:"isStarred"`
	HasAttachments bool      `json:"hasAttachments"`
	Snippet        string    `json:"snippet"`
}

// EmailDetailDTO represents full email details for the frontend
type EmailDetailDTO struct {
	EmailDTO
	ToAddresses  string          `json:"toAddresses"`
	CcAddresses  string          `json:"ccAddresses"`
	BodyText     string          `json:"bodyText"`
	BodyHTML     string          `json:"bodyHtml"`
	Attachments  []AttachmentDTO `json:"attachments"`
}

// AttachmentDTO represents an email attachment
type AttachmentDTO struct {
	ID          int64  `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	ContentID   string `json:"contentId,omitempty"`
	Size        int64  `json:"size"`
	Data        string `json:"data,omitempty"` // base64 encoded for inline images
	IsInline    bool   `json:"isInline"`
	PartNumber  string `json:"partNumber,omitempty"`
}

// FolderDTO represents a mail folder for the frontend
type FolderDTO struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	TotalMessages int    `json:"totalMessages"`
	UnreadMessages int   `json:"unreadMessages"`
}

// AccountDTO represents an email account
type AccountDTO struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// SendRequest represents an email to send
type SendRequest struct {
	To      []string `json:"to"`
	Cc      []string `json:"cc"`
	Bcc     []string `json:"bcc"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	IsHTML  bool     `json:"isHtml"`
	ReplyTo int64    `json:"replyTo,omitempty"`
}

// SendResult represents the result of sending an email
type SendResult struct {
	Success   bool   `json:"success"`
	MessageID string `json:"messageId"`
	Error     string `json:"error,omitempty"`
}

// DraftDTO represents a draft email
type DraftDTO struct {
	ID          int64    `json:"id,omitempty"`
	To          []string `json:"to"`
	Cc          []string `json:"cc"`
	Bcc         []string `json:"bcc"`
	Subject     string   `json:"subject"`
	BodyHTML    string   `json:"bodyHtml"`
	BodyText    string   `json:"bodyText"`
	ReplyToID   int64    `json:"replyToId,omitempty"`
}

// ConnectionStatus represents IMAP connection status
type ConnectionStatus struct {
	Connected    bool      `json:"connected"`
	LastSync     time.Time `json:"lastSync"`
	Error        string    `json:"error,omitempty"`
}

// SyncResultDTO represents the result of a sync operation
type SyncResultDTO struct {
	NewEmails     int `json:"newEmails"`
	DeletedEmails int `json:"deletedEmails"`
}

// SearchResultDTO represents a search result
type SearchResultDTO struct {
	Emails     []EmailDTO `json:"emails"`
	TotalCount int        `json:"totalCount"`
	Query      string     `json:"query"`
}

// ============================================================================
// ANALYTICS DTOs
// ============================================================================

// AnalyticsOverviewDTO contains general email statistics
type AnalyticsOverviewDTO struct {
	TotalEmails    int     `json:"totalEmails"`
	UnreadEmails   int     `json:"unreadEmails"`
	StarredEmails  int     `json:"starredEmails"`
	ArchivedEmails int     `json:"archivedEmails"`
	SentEmails     int     `json:"sentEmails"`
	DraftCount     int     `json:"draftCount"`
	StorageUsedMB  float64 `json:"storageUsedMb"`
}

// SenderStatsDTO contains statistics for a sender
type SenderStatsDTO struct {
	Email       string  `json:"email"`
	Name        string  `json:"name"`
	Count       int     `json:"count"`
	UnreadCount int     `json:"unreadCount"`
	Percentage  float64 `json:"percentage"`
}

// HourlyStatsDTO contains email count per hour
type HourlyStatsDTO struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

// DailyStatsDTO contains email count per day
type DailyStatsDTO struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// WeekdayStatsDTO contains email count per weekday
type WeekdayStatsDTO struct {
	Weekday int    `json:"weekday"`
	Name    string `json:"name"`
	Count   int    `json:"count"`
}

// EmailTrendsDTO contains email volume trends
type EmailTrendsDTO struct {
	Daily   []DailyStatsDTO   `json:"daily"`
	Hourly  []HourlyStatsDTO  `json:"hourly"`
	Weekday []WeekdayStatsDTO `json:"weekday"`
}

// ResponseTimeStatsDTO contains response time statistics
type ResponseTimeStatsDTO struct {
	AvgResponseMinutes float64 `json:"avgResponseMinutes"`
	MedianMinutes      float64 `json:"medianMinutes"`
	ResponseRate       float64 `json:"responseRate"`
}

// AnalyticsResultDTO contains all analytics data
type AnalyticsResultDTO struct {
	Overview     AnalyticsOverviewDTO   `json:"overview"`
	TopSenders   []SenderStatsDTO       `json:"topSenders"`
	Trends       EmailTrendsDTO         `json:"trends"`
	ResponseTime ResponseTimeStatsDTO   `json:"responseTime"`
	Period       string                 `json:"period"`
	GeneratedAt  time.Time              `json:"generatedAt"`
}

// ============================================================================
// SETTINGS DTOs
// ============================================================================

// SettingsDTO contains all application settings
type SettingsDTO struct {
	SyncFolders      []string `json:"syncFolders"`
	UITheme          string   `json:"uiTheme"`
	UIShowPreview    bool     `json:"uiShowPreview"`
	UIPageSize       int      `json:"uiPageSize"`
	ComposeFormat    string   `json:"composeFormat"`
	ComposeSendDelay int      `json:"composeSendDelay"`
	SyncInterval     string   `json:"syncInterval"`
}

// AvailableFolderDTO represents a folder with its sync status
type AvailableFolderDTO struct {
	Name       string `json:"name"`
	IsSelected bool   `json:"isSelected"`
}
