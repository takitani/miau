package storage

import (
	"database/sql"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
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

// GenerateThreadID generates a thread ID for an email based on headers.
//
// DEPRECATED for Gmail accounts! This function creates thread_ids that look like
// Message-IDs (xxx@yyy.com), not Gmail's hex thread IDs (17062d1764232491).
// For Gmail accounts, use SyncThreadIDsFromGmail to get proper thread IDs.
//
// This function is kept for non-Gmail IMAP servers that don't have native threading.
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

	// Strategy 3: If subject has reply/forward prefix but no threading headers,
	// use subject-based threading (common for emails with stripped headers)
	var prefixRegex = regexp.MustCompile(`(?i)^(re|fwd|fw|aw|sv|ref):\s*`)
	if prefixRegex.MatchString(subject) {
		return "subject:" + NormalizeSubject(subject)
	}

	// Strategy 4: Own Message-ID (new thread)
	if messageID != "" {
		return cleanMessageID(messageID)
	}

	// Strategy 5: Fallback to subject-based threading
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
// Deduplicates by message_id and excludes Trash/Spam folders
func (td *ThreadDetector) GetThreadEmails(threadID string, accountID int64) ([]Email, error) {
	var emails []Email
	var err = td.db.Select(&emails, `
		SELECT e.* FROM emails e
		JOIN folders f ON e.folder_id = f.id
		WHERE e.thread_id = ?
		  AND e.account_id = ?
		  AND e.is_deleted = 0
		  AND f.name NOT LIKE '%Trash%'
		  AND f.name NOT LIKE '%Spam%'
		  AND f.name NOT LIKE '%Lixeira%'
		  AND e.id IN (
		    SELECT MIN(e2.id) FROM emails e2
		    JOIN folders f2 ON e2.folder_id = f2.id
		    WHERE e2.thread_id = ?
		      AND e2.account_id = ?
		      AND e2.is_deleted = 0
		      AND f2.name NOT LIKE '%Trash%'
		      AND f2.name NOT LIKE '%Spam%'
		      AND f2.name NOT LIKE '%Lixeira%'
		    GROUP BY COALESCE(NULLIF(e2.message_id, ''), CAST(e2.id AS TEXT))
		  )
		ORDER BY e.date DESC
	`, threadID, accountID, threadID, accountID)

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
// Excludes Trash/Spam and deduplicates by message_id
func (td *ThreadDetector) CountThreadEmails(threadID string, accountID int64) (int, error) {
	var count int
	var err = td.db.Get(&count, `
		SELECT COUNT(DISTINCT COALESCE(NULLIF(e.message_id, ''), e.id))
		FROM emails e
		JOIN folders f ON e.folder_id = f.id
		WHERE e.thread_id = ?
		  AND e.account_id = ?
		  AND e.is_deleted = 0
		  AND f.name NOT LIKE '%Trash%'
		  AND f.name NOT LIKE '%Spam%'
		  AND f.name NOT LIKE '%Lixeira%'
	`, threadID, accountID)

	return count, err
}

// GetThreadParticipants returns unique email addresses participating in a thread
// Excludes Trash/Spam folders
func (td *ThreadDetector) GetThreadParticipants(threadID string, accountID int64) ([]string, error) {
	var participants []string
	var err = td.db.Select(&participants, `
		SELECT DISTINCT e.from_email FROM emails e
		JOIN folders f ON e.folder_id = f.id
		WHERE e.thread_id = ?
		  AND e.account_id = ?
		  AND e.is_deleted = 0
		  AND f.name NOT LIKE '%Trash%'
		  AND f.name NOT LIKE '%Spam%'
		  AND f.name NOT LIKE '%Lixeira%'
		ORDER BY e.from_email
	`, threadID, accountID)

	return participants, err
}

// ReprocessAllThreads is DEPRECATED - do not use!
// Thread IDs should come from Gmail sync (SyncThreadIDsFromGmail), not local header analysis.
// Local threading based on In-Reply-To/References creates invalid thread_ids (Message-IDs).
// Use SyncThreadIDsFromGmail instead.
func ReprocessAllThreads() (int, error) {
	// DO NOT USE - returns 0 intentionally
	return 0, nil
}
