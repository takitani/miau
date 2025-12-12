package storage

import (
	"time"
)

// SnoozedEmail represents a snoozed email in storage
type SnoozedEmail struct {
	ID          int64     `db:"id"`
	EmailID     int64     `db:"email_id"`
	AccountID   int64     `db:"account_id"`
	SnoozedAt   time.Time `db:"snoozed_at"`
	SnoozeUntil time.Time `db:"snooze_until"`
	Preset      string    `db:"preset"`
	Processed   bool      `db:"processed"`
}

// SnoozeEmail snoozes an email until specified time
func SnoozeEmail(emailID, accountID int64, until time.Time, preset string) error {
	_, err := db.Exec(`
		INSERT INTO snoozed_emails (email_id, account_id, snooze_until, preset)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(email_id) DO UPDATE SET
			snooze_until = excluded.snooze_until,
			preset = excluded.preset,
			snoozed_at = CURRENT_TIMESTAMP,
			processed = 0`,
		emailID, accountID, until, preset)
	return err
}

// UnsnoozeEmail removes snooze from an email
func UnsnoozeEmail(emailID int64) error {
	_, err := db.Exec("DELETE FROM snoozed_emails WHERE email_id = ?", emailID)
	return err
}

// GetSnoozedEmails returns all snoozed emails for an account
func GetSnoozedEmails(accountID int64) ([]SnoozedEmail, error) {
	var snoozes []SnoozedEmail
	err := db.Select(&snoozes, `
		SELECT * FROM snoozed_emails
		WHERE account_id = ? AND processed = 0
		ORDER BY snooze_until ASC`,
		accountID)
	return snoozes, err
}

// GetSnoozedEmailsCount returns the count of snoozed emails for an account
func GetSnoozedEmailsCount(accountID int64) (int, error) {
	var count int
	err := db.Get(&count, `
		SELECT COUNT(*) FROM snoozed_emails
		WHERE account_id = ? AND processed = 0`,
		accountID)
	return count, err
}

// GetDueSnoozes returns snoozes that are due (snooze_until <= now)
func GetDueSnoozes() ([]SnoozedEmail, error) {
	var snoozes []SnoozedEmail
	err := db.Select(&snoozes, `
		SELECT * FROM snoozed_emails
		WHERE processed = 0 AND snooze_until <= CURRENT_TIMESTAMP
		ORDER BY snooze_until ASC`)
	return snoozes, err
}

// MarkSnoozeProcessed marks a snooze as processed
func MarkSnoozeProcessed(id int64) error {
	_, err := db.Exec(`
		UPDATE snoozed_emails SET processed = 1
		WHERE id = ?`, id)
	return err
}

// IsEmailSnoozed checks if an email is currently snoozed
func IsEmailSnoozed(emailID int64) (bool, error) {
	var count int
	err := db.Get(&count, `
		SELECT COUNT(*) FROM snoozed_emails
		WHERE email_id = ? AND processed = 0`,
		emailID)
	return count > 0, err
}

// GetSnoozeByEmailID returns the snooze for an email if exists
func GetSnoozeByEmailID(emailID int64) (*SnoozedEmail, error) {
	var snooze SnoozedEmail
	err := db.Get(&snooze, `
		SELECT * FROM snoozed_emails
		WHERE email_id = ? AND processed = 0`,
		emailID)
	if err != nil {
		return nil, err
	}
	return &snooze, nil
}

// MarkEmailUnread marks an email as unread (used when snooze triggers)
func MarkEmailUnread(emailID int64) error {
	_, err := db.Exec(`
		UPDATE emails SET is_read = 0, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, emailID)
	return err
}

// BumpEmailDate updates the email date to now (so it appears at top)
func BumpEmailDate(emailID int64) error {
	_, err := db.Exec(`
		UPDATE emails SET date = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, emailID)
	return err
}
