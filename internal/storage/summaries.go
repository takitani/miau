package storage

import (
	"database/sql"
	"encoding/json"
	"time"
)

// SummaryStyle defines the style of summary
type SummaryStyle string

const (
	SummaryStyleTLDR     SummaryStyle = "tldr"     // 1-2 sentences
	SummaryStyleBrief    SummaryStyle = "brief"    // 3-5 sentences
	SummaryStyleDetailed SummaryStyle = "detailed" // Full summary with sections
)

// EmailSummary represents a cached email summary
type EmailSummary struct {
	ID        int64        `db:"id"`
	EmailID   int64        `db:"email_id"`
	Style     SummaryStyle `db:"style"`
	Content   string       `db:"content"`
	KeyPoints string       `db:"key_points"` // JSON array
	CreatedAt time.Time    `db:"created_at"`
}

// ThreadSummary represents a cached thread summary
type ThreadSummary struct {
	ID           int64     `db:"id"`
	ThreadID     string    `db:"thread_id"`
	Participants string    `db:"participants"`  // JSON array
	Timeline     string    `db:"timeline"`
	KeyDecisions string    `db:"key_decisions"` // JSON array
	ActionItems  string    `db:"action_items"`  // JSON array
	CreatedAt    time.Time `db:"created_at"`
}

// GetEmailSummary retrieves a cached email summary
func GetEmailSummary(emailID int64) (*EmailSummary, error) {
	var summary EmailSummary
	var err = db.Get(&summary, `
		SELECT id, email_id, style, content, key_points, created_at
		FROM email_summaries
		WHERE email_id = ?
	`, emailID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

// SaveEmailSummary saves an email summary to cache
func SaveEmailSummary(emailID int64, style SummaryStyle, content string, keyPoints []string) error {
	var keyPointsJSON string
	if len(keyPoints) > 0 {
		var data, err = json.Marshal(keyPoints)
		if err == nil {
			keyPointsJSON = string(data)
		}
	}

	var _, err = db.Exec(`
		INSERT INTO email_summaries (email_id, style, content, key_points)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(email_id) DO UPDATE SET
			style = excluded.style,
			content = excluded.content,
			key_points = excluded.key_points,
			created_at = CURRENT_TIMESTAMP
	`, emailID, style, content, keyPointsJSON)
	return err
}

// DeleteEmailSummary removes a cached email summary
func DeleteEmailSummary(emailID int64) error {
	var _, err = db.Exec("DELETE FROM email_summaries WHERE email_id = ?", emailID)
	return err
}

// GetThreadSummary retrieves a cached thread summary
func GetThreadSummary(threadID string) (*ThreadSummary, error) {
	var summary ThreadSummary
	var err = db.Get(&summary, `
		SELECT id, thread_id, participants, timeline, key_decisions, action_items, created_at
		FROM thread_summaries
		WHERE thread_id = ?
	`, threadID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

// SaveThreadSummary saves a thread summary to cache
func SaveThreadSummary(threadID string, participants []string, timeline string, keyDecisions []string, actionItems []string) error {
	var participantsJSON, keyDecisionsJSON, actionItemsJSON string

	if len(participants) > 0 {
		var data, _ = json.Marshal(participants)
		participantsJSON = string(data)
	}
	if len(keyDecisions) > 0 {
		var data, _ = json.Marshal(keyDecisions)
		keyDecisionsJSON = string(data)
	}
	if len(actionItems) > 0 {
		var data, _ = json.Marshal(actionItems)
		actionItemsJSON = string(data)
	}

	var _, err = db.Exec(`
		INSERT INTO thread_summaries (thread_id, participants, timeline, key_decisions, action_items)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(thread_id) DO UPDATE SET
			participants = excluded.participants,
			timeline = excluded.timeline,
			key_decisions = excluded.key_decisions,
			action_items = excluded.action_items,
			created_at = CURRENT_TIMESTAMP
	`, threadID, participantsJSON, timeline, keyDecisionsJSON, actionItemsJSON)
	return err
}

// DeleteThreadSummary removes a cached thread summary
func DeleteThreadSummary(threadID string) error {
	var _, err = db.Exec("DELETE FROM thread_summaries WHERE thread_id = ?", threadID)
	return err
}

// GetKeyPointsFromSummary parses the key_points JSON array
func GetKeyPointsFromSummary(summary *EmailSummary) []string {
	if summary == nil || summary.KeyPoints == "" {
		return nil
	}
	var points []string
	json.Unmarshal([]byte(summary.KeyPoints), &points)
	return points
}

// ParseThreadSummaryParticipants parses the participants JSON array
func ParseThreadSummaryParticipants(summary *ThreadSummary) []string {
	if summary == nil || summary.Participants == "" {
		return nil
	}
	var participants []string
	json.Unmarshal([]byte(summary.Participants), &participants)
	return participants
}

// ParseThreadSummaryKeyDecisions parses the key_decisions JSON array
func ParseThreadSummaryKeyDecisions(summary *ThreadSummary) []string {
	if summary == nil || summary.KeyDecisions == "" {
		return nil
	}
	var decisions []string
	json.Unmarshal([]byte(summary.KeyDecisions), &decisions)
	return decisions
}

// ParseThreadSummaryActionItems parses the action_items JSON array
func ParseThreadSummaryActionItems(summary *ThreadSummary) []string {
	if summary == nil || summary.ActionItems == "" {
		return nil
	}
	var items []string
	json.Unmarshal([]byte(summary.ActionItems), &items)
	return items
}

// IsSummaryCacheFresh checks if the summary is still fresh (less than 7 days old)
func IsSummaryCacheFresh(createdAt time.Time) bool {
	return time.Since(createdAt) < 7*24*time.Hour
}
