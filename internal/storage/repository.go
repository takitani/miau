package storage

import (
	"database/sql"
	"time"
)

// === ACCOUNTS ===

func GetOrCreateAccount(email, name string) (*Account, error) {
	var account Account

	err := db.Get(&account, "SELECT * FROM accounts WHERE email = ?", email)
	if err == nil {
		return &account, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Cria nova conta
	var result, err2 = db.Exec("INSERT INTO accounts (email, name) VALUES (?, ?)", email, name)
	if err2 != nil {
		return nil, err2
	}

	var id, _ = result.LastInsertId()
	account = Account{
		ID:        id,
		Email:     email,
		Name:      name,
		CreatedAt: SQLiteTime{time.Now()},
	}

	return &account, nil
}

// === FOLDERS ===

func GetOrCreateFolder(accountID int64, name string) (*Folder, error) {
	var folder Folder

	err := db.Get(&folder, "SELECT * FROM folders WHERE account_id = ? AND name = ?", accountID, name)
	if err == nil {
		return &folder, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Cria nova pasta
	var result, err2 = db.Exec("INSERT INTO folders (account_id, name) VALUES (?, ?)", accountID, name)
	if err2 != nil {
		return nil, err2
	}

	var id, _ = result.LastInsertId()
	folder = Folder{
		ID:        id,
		AccountID: accountID,
		Name:      name,
	}

	return &folder, nil
}

func GetFolders(accountID int64) ([]Folder, error) {
	var folders []Folder
	err := db.Select(&folders, "SELECT * FROM folders WHERE account_id = ? ORDER BY name", accountID)
	return folders, err
}

func UpdateFolderStats(folderID int64, total, unread int) error {
	_, err := db.Exec(`
		UPDATE folders
		SET total_messages = ?, unread_messages = ?, last_sync = CURRENT_TIMESTAMP
		WHERE id = ?`,
		total, unread, folderID)
	return err
}

// === EMAILS ===

func UpsertEmail(e *Email) error {
	_, err := db.Exec(`
		INSERT INTO emails (
			account_id, folder_id, uid, message_id, subject,
			from_name, from_email, to_addresses, cc_addresses, date,
			is_read, is_starred, is_deleted, has_attachments, snippet,
			body_text, body_html, raw_headers, size, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(account_id, folder_id, uid) DO UPDATE SET
			subject = excluded.subject,
			from_name = excluded.from_name,
			from_email = excluded.from_email,
			is_read = excluded.is_read,
			is_starred = excluded.is_starred,
			is_deleted = excluded.is_deleted,
			body_text = excluded.body_text,
			body_html = excluded.body_html,
			updated_at = CURRENT_TIMESTAMP`,
		e.AccountID, e.FolderID, e.UID, e.MessageID, e.Subject,
		e.FromName, e.FromEmail, e.ToAddresses, e.CcAddresses, e.Date,
		e.IsRead, e.IsStarred, e.IsDeleted, e.HasAttachments, e.Snippet,
		e.BodyText, e.BodyHTML, e.RawHeaders, e.Size)
	return err
}

func GetEmails(accountID, folderID int64, limit, offset int) ([]EmailSummary, error) {
	var emails []EmailSummary
	err := db.Select(&emails, `
		SELECT id, uid, subject, from_name, from_email, date, is_read, is_starred, snippet
		FROM emails
		WHERE account_id = ? AND folder_id = ? AND is_deleted = 0
		ORDER BY date DESC
		LIMIT ? OFFSET ?`,
		accountID, folderID, limit, offset)
	return emails, err
}

func GetEmailByID(id int64) (*Email, error) {
	var email Email
	err := db.Get(&email, "SELECT * FROM emails WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return &email, nil
}

func GetEmailByUID(accountID, folderID int64, uid uint32) (*Email, error) {
	var email Email
	err := db.Get(&email, "SELECT * FROM emails WHERE account_id = ? AND folder_id = ? AND uid = ?", accountID, folderID, uid)
	if err != nil {
		return nil, err
	}
	return &email, nil
}

func GetLatestUID(accountID, folderID int64) (uint32, error) {
	var uid uint32
	err := db.Get(&uid, "SELECT COALESCE(MAX(uid), 0) FROM emails WHERE account_id = ? AND folder_id = ?", accountID, folderID)
	return uid, err
}

func MarkAsRead(id int64, read bool) error {
	_, err := db.Exec("UPDATE emails SET is_read = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", read, id)
	return err
}

func MarkAsStarred(id int64, starred bool) error {
	_, err := db.Exec("UPDATE emails SET is_starred = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", starred, id)
	return err
}

func DeleteEmail(id int64) error {
	_, err := db.Exec("UPDATE emails SET is_deleted = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	return err
}

func CountEmails(accountID, folderID int64) (total int, unread int, err error) {
	err = db.Get(&total, "SELECT COUNT(*) FROM emails WHERE account_id = ? AND folder_id = ? AND is_deleted = 0", accountID, folderID)
	if err != nil {
		return
	}
	err = db.Get(&unread, "SELECT COUNT(*) FROM emails WHERE account_id = ? AND folder_id = ? AND is_deleted = 0 AND is_read = 0", accountID, folderID)
	return
}

// === SEARCH ===

func SearchEmails(accountID int64, query string, limit int) ([]EmailSummary, error) {
	var emails []EmailSummary
	err := db.Select(&emails, `
		SELECT e.id, e.uid, e.subject, e.from_name, e.from_email, e.date, e.is_read, e.is_starred, e.snippet
		FROM emails e
		JOIN emails_fts fts ON e.id = fts.rowid
		WHERE e.account_id = ? AND e.is_deleted = 0 AND emails_fts MATCH ?
		ORDER BY e.date DESC
		LIMIT ?`,
		accountID, query, limit)
	return emails, err
}
