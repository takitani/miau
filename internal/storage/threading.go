package storage

import (
	"database/sql"
	"regexp"
	"strings"
)

// ThreadDetector handles thread detection and grouping logic
type ThreadDetector struct {
	db *sqlx.DB
}

// NewThreadDetector creates a new thread detector
func NewThreadDetector(db *sqlx.DB) *ThreadDetector {
	return &ThreadDetector{db: db}
}

// NormalizeSubject removes Re:, Fwd:, etc from subject for thread matching
func NormalizeSubject(subject string) string {
	// Remove common reply/forward prefixes (case-insensitive)
	var prefixRegex = regexp.MustCompile(`(?i)^(re|fwd|fw|aw|sv|ref):\s*`)
	var normalized = subject

	// Keep removing prefixes until no more found
	for prefixRegex.MatchString(normalized) {
		normalized = prefixRegex.ReplaceAllString(normalized, "")
		normalized = strings.TrimSpace(normalized)
	}

	// Remove extra whitespace
	normalized = strings.TrimSpace(normalized)
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	return strings.ToLower(normalized)
}

// GenerateThreadID generates a thread ID for an email
// Algorithm:
// 1. If email has In-Reply-To, use that Message-ID as thread root
// 2. Else if References exists, use first Message-ID as thread root
// 3. Else use email's own Message-ID as thread root (starts new thread)
// 4. Fallback: use normalized subject as thread ID
func GenerateThreadID(messageID, inReplyTo, references, subject string) string {
	// Strategy 1: In-Reply-To points to parent, find its thread
	if inReplyTo != "" {
		return cleanMessageID(inReplyTo)
	}

	// Strategy 2: References list - first ID is thread root
	if references != "" {
		var refList = strings.Fields(references)
		if len(refList) > 0 {
			return cleanMessageID(refList[0])
		}
	}

	// Strategy 3: Own Message-ID (new thread)
	if messageID != "" {
		return cleanMessageID(messageID)
	}

	// Strategy 4: Fallback to subject-based threading
	return "subject:" + NormalizeSubject(subject)
}

// cleanMessageID removes < > brackets and normalizes Message-ID
func cleanMessageID(msgID string) string {
	var cleaned = strings.Trim(msgID, "<> \t\r\n")
	return strings.ToLower(cleaned)
}

// DetectAndUpdateThread detects thread for a newly saved email and updates it
func (td *ThreadDetector) DetectAndUpdateThread(emailID int64, messageID, inReplyTo, references, subject string) error {
	// Generate thread ID
	var threadID = GenerateThreadID(messageID, inReplyTo, references, subject)

	// If In-Reply-To exists, try to find the parent's thread_id
	if inReplyTo != "" {
		var parentThreadID sql.NullString
		var err = td.db.Get(&parentThreadID,
			"SELECT thread_id FROM emails WHERE message_id = ? LIMIT 1",
			cleanMessageID(inReplyTo))

		if err == nil && parentThreadID.Valid && parentThreadID.String != "" {
			// Use parent's thread_id to keep thread continuity
			threadID = parentThreadID.String
		}
	}

	// Update email with thread_id
	var _, err = td.db.Exec(
		"UPDATE emails SET thread_id = ? WHERE id = ?",
		threadID, emailID)

	return err
}

// GetThreadEmails returns all emails in a thread, ordered by date DESC (newest first)
func (td *ThreadDetector) GetThreadEmails(threadID string, accountID int64) ([]Email, error) {
	var emails []Email
	var err = td.db.Select(&emails, `
		SELECT * FROM emails
		WHERE thread_id = ?
		  AND account_id = ?
		  AND is_deleted = 0
		ORDER BY date DESC
	`, threadID, accountID)

	return emails, err
}

// GetThreadForEmail returns all emails in the same thread as the given email
func (td *ThreadDetector) GetThreadForEmail(emailID int64) ([]Email, error) {
	// First get the thread_id and account_id of the email
	var email Email
	var err = td.db.Get(&email, "SELECT * FROM emails WHERE id = ?", emailID)
	if err != nil {
		return nil, err
	}

	if !email.ThreadID.Valid || email.ThreadID.String == "" {
		// No thread, return just this email
		return []Email{email}, nil
	}

	return td.GetThreadEmails(email.ThreadID.String, email.AccountID)
}

// CountThreadEmails returns the count of emails in a thread
func (td *ThreadDetector) CountThreadEmails(threadID string, accountID int64) (int, error) {
	var count int
	var err = td.db.Get(&count, `
		SELECT COUNT(*) FROM emails
		WHERE thread_id = ?
		  AND account_id = ?
		  AND is_deleted = 0
	`, threadID, accountID)

	return count, err
}

// GetThreadParticipants returns unique email addresses participating in a thread
func (td *ThreadDetector) GetThreadParticipants(threadID string, accountID int64) ([]string, error) {
	var participants []string
	var err = td.db.Select(&participants, `
		SELECT DISTINCT from_email FROM emails
		WHERE thread_id = ?
		  AND account_id = ?
		  AND is_deleted = 0
		ORDER BY from_email
	`, threadID, accountID)

	return participants, err
}
