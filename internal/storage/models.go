package storage

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"
)

// SQLiteTime handles time parsing from SQLite strings
type SQLiteTime struct {
	time.Time
}

func (t *SQLiteTime) Scan(value interface{}) error {
	if value == nil {
		t.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		t.Time = v
		return nil
	case string:
		var formats = []string{
			"2006-01-02 15:04:05.999999999-07:00",
			"2006-01-02 15:04:05.999999999",
			"2006-01-02T15:04:05.999999999-07:00",
			"2006-01-02T15:04:05.999999999Z",
			"2006-01-02T15:04:05-07:00",
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05 -0700 -0700",
			"2006-01-02 15:04:05 -0700 MST",
			"2006-01-02 15:04:05 -0700",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if parsed, err := time.Parse(format, v); err == nil {
				t.Time = parsed
				return nil
			}
		}
		return fmt.Errorf("cannot parse time: %s", v)
	default:
		return fmt.Errorf("unsupported type for SQLiteTime: %T", value)
	}
}

func (t SQLiteTime) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.Time.Format("2006-01-02 15:04:05"), nil
}

type Account struct {
	ID        int64      `db:"id"`
	Email     string     `db:"email"`
	Name      string     `db:"name"`
	CreatedAt SQLiteTime `db:"created_at"`
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
	Date           SQLiteTime     `db:"date"`
	IsRead         bool           `db:"is_read"`
	IsStarred      bool           `db:"is_starred"`
	IsDeleted      bool           `db:"is_deleted"`
	IsReplied      bool           `db:"is_replied"`
	HasAttachments bool           `db:"has_attachments"`
	Snippet        string         `db:"snippet"`
	BodyText       string         `db:"body_text"`
	BodyHTML       string         `db:"body_html"`
	RawHeaders     string         `db:"raw_headers"`
	Size           int64          `db:"size"`
	CreatedAt      SQLiteTime     `db:"created_at"`
	UpdatedAt      SQLiteTime     `db:"updated_at"`
}

// EmailSummary é uma versão resumida para listagem
type EmailSummary struct {
	ID        int64          `db:"id"`
	UID       uint32         `db:"uid"`
	MessageID sql.NullString `db:"message_id"`
	Subject   string         `db:"subject"`
	FromName  string         `db:"from_name"`
	FromEmail string         `db:"from_email"`
	Date      SQLiteTime     `db:"date"`
	IsRead    bool           `db:"is_read"`
	IsStarred bool           `db:"is_starred"`
	IsReplied bool           `db:"is_replied"`
	Snippet   string         `db:"snippet"`
}
