package storage

import (
	"database/sql"
	"time"
)

type Account struct {
	ID        int64     `db:"id"`
	Email     string    `db:"email"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}

type Folder struct {
	ID             int64          `db:"id"`
	AccountID      int64          `db:"account_id"`
	Name           string         `db:"name"`
	TotalMessages  int            `db:"total_messages"`
	UnreadMessages int            `db:"unread_messages"`
	LastSync       sql.NullTime   `db:"last_sync"`
}

type Email struct {
	ID             int64          `db:"id"`
	AccountID      int64          `db:"account_id"`
	FolderID       int64          `db:"folder_id"`
	UID            uint32         `db:"uid"`
	MessageID      sql.NullString `db:"message_id"`
	Subject        string         `db:"subject"`
	FromName       string         `db:"from_name"`
	FromEmail      string         `db:"from_email"`
	ToAddresses    string         `db:"to_addresses"`
	CcAddresses    string         `db:"cc_addresses"`
	Date           time.Time      `db:"date"`
	IsRead         bool           `db:"is_read"`
	IsStarred      bool           `db:"is_starred"`
	IsDeleted      bool           `db:"is_deleted"`
	HasAttachments bool           `db:"has_attachments"`
	Snippet        string         `db:"snippet"`
	BodyText       string         `db:"body_text"`
	BodyHTML       string         `db:"body_html"`
	RawHeaders     string         `db:"raw_headers"`
	Size           int64          `db:"size"`
	CreatedAt      time.Time      `db:"created_at"`
	UpdatedAt      time.Time      `db:"updated_at"`
}

// EmailSummary é uma versão resumida para listagem
type EmailSummary struct {
	ID        int64     `db:"id"`
	UID       uint32    `db:"uid"`
	Subject   string    `db:"subject"`
	FromName  string    `db:"from_name"`
	FromEmail string    `db:"from_email"`
	Date      time.Time `db:"date"`
	IsRead    bool      `db:"is_read"`
	IsStarred bool      `db:"is_starred"`
	Snippet   string    `db:"snippet"`
}
