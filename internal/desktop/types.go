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

// SearchResultDTO represents a search result
type SearchResultDTO struct {
	Emails     []EmailDTO `json:"emails"`
	TotalCount int        `json:"totalCount"`
	Query      string     `json:"query"`
}
