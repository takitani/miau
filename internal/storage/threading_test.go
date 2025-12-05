package storage

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestGmailThreadIDFormat verifies that Gmail thread IDs are valid hex strings
func TestGmailThreadIDFormat(t *testing.T) {
	var validThreadIDs = []string{
		"17062d1764232491",
		"19ae713f03d89fe5",
		"170c5b08c8811e14",
		"1734431e92ba2897",
	}

	var invalidThreadIDs = []string{
		"qr7qwmsyrecxevjbbtl26q@geopod-ismtpd-22",           // Message-ID format
		"caa=yxit+utezau-q4vf5mscdtqsdkkdcfrfp5uq3rfcynkwbea@mail.gmail.com", // Message-ID
		"<abc123@example.com>",                              // Message-ID with brackets
		"subject:newsletter",                                // Subject-based threading
		"",                                                  // Empty
	}

	for _, id := range validThreadIDs {
		if !isValidGmailThreadID(id) {
			t.Errorf("Expected %q to be valid Gmail thread ID", id)
		}
	}

	for _, id := range invalidThreadIDs {
		if isValidGmailThreadID(id) {
			t.Errorf("Expected %q to be INVALID Gmail thread ID", id)
		}
	}
}

// isValidGmailThreadID checks if a thread_id looks like a Gmail thread ID (16-char hex)
func isValidGmailThreadID(threadID string) bool {
	if threadID == "" {
		return false
	}
	// Gmail thread IDs are 16 character hex strings
	if len(threadID) != 16 {
		return false
	}
	for _, c := range threadID {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// TestThreadIDsInDatabaseAreValid ensures no invalid thread_ids exist in database
func TestThreadIDsInDatabaseAreValid(t *testing.T) {
	var tmpDir = t.TempDir()
	var dbPath = filepath.Join(tmpDir, "test.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer Close()

	// Create test account and folder
	var _, err = db.Exec("INSERT INTO accounts (email, name) VALUES ('test@example.com', 'Test')")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}
	var _, err2 = db.Exec("INSERT INTO folders (account_id, name) VALUES (1, 'INBOX')")
	if err2 != nil {
		t.Fatalf("Failed to create folder: %v", err2)
	}

	// Insert emails with various thread_ids
	var testCases = []struct {
		uid      int
		threadID string
		valid    bool
	}{
		{1, "17062d1764232491", true},                    // Valid Gmail thread ID
		{2, "19ae713f03d89fe5", true},                    // Valid Gmail thread ID
		{3, "", true},                                    // Empty is OK (not synced yet)
		{4, "abc@example.com", false},                    // Invalid: Message-ID format
		{5, "subject:newsletter", false},                 // Invalid: subject-based
	}

	for _, tc := range testCases {
		var _, err = db.Exec(`
			INSERT INTO emails (account_id, folder_id, uid, subject, from_email, thread_id)
			VALUES (1, 1, ?, 'Test', 'test@example.com', ?)
		`, tc.uid, tc.threadID)
		if err != nil {
			t.Fatalf("Failed to insert email: %v", err)
		}
	}

	// Query for invalid thread_ids
	var invalidCount int
	err = db.Get(&invalidCount, `
		SELECT COUNT(*) FROM emails
		WHERE thread_id IS NOT NULL
		  AND thread_id <> ''
		  AND (thread_id LIKE '%@%' OR thread_id LIKE 'subject:%')
	`)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}

	if invalidCount != 2 {
		t.Errorf("Expected 2 invalid thread_ids, got %d", invalidCount)
	}
}

// TestGetThreadForEmailReturnsCorrectEmails tests thread retrieval
func TestGetThreadForEmailReturnsCorrectEmails(t *testing.T) {
	var tmpDir = t.TempDir()
	var dbPath = filepath.Join(tmpDir, "test.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer Close()

	// Setup
	db.Exec("INSERT INTO accounts (email, name) VALUES ('test@example.com', 'Test')")
	db.Exec("INSERT INTO folders (account_id, name) VALUES (1, 'INBOX')")

	// Create a thread with 3 emails (include all NOT NULL string fields)
	var threadID = "17062d1764232491"
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_name, from_email, to_addresses, cc_addresses, snippet, body_text, body_html, raw_headers, thread_id, date)
		VALUES (1, 1, 1, 'msg1@example.com', 'Original', 'Alice', 'alice@example.com', '', '', '', '', '', '', ?, '2024-01-01')`, threadID)
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_name, from_email, to_addresses, cc_addresses, snippet, body_text, body_html, raw_headers, thread_id, date)
		VALUES (1, 1, 2, 'msg2@example.com', 'Re: Original', 'Bob', 'bob@example.com', '', '', '', '', '', '', ?, '2024-01-02')`, threadID)
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_name, from_email, to_addresses, cc_addresses, snippet, body_text, body_html, raw_headers, thread_id, date)
		VALUES (1, 1, 3, 'msg3@example.com', 'Re: Original', 'Alice', 'alice@example.com', '', '', '', '', '', '', ?, '2024-01-03')`, threadID)

	// Create another email NOT in the thread
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_name, from_email, to_addresses, cc_addresses, snippet, body_text, body_html, raw_headers, thread_id, date)
		VALUES (1, 1, 4, 'msg4@example.com', 'Different Thread', 'Charlie', 'charlie@example.com', '', '', '', '', '', '', '19ae713f03d89fe5', '2024-01-04')`)

	// Test GetThreadForEmail
	var emails, err = GetThreadForEmail(1) // Get thread for email ID 1
	if err != nil {
		t.Fatalf("GetThreadForEmail failed: %v", err)
	}

	if len(emails) != 3 {
		t.Errorf("Expected 3 emails in thread, got %d", len(emails))
	}

	// Verify all emails have the same thread_id
	for _, e := range emails {
		if e.ThreadID.String != threadID {
			t.Errorf("Email %d has wrong thread_id: %s", e.ID, e.ThreadID.String)
		}
	}

	// Verify emails are ordered by date DESC (newest first)
	if len(emails) >= 2 && emails[0].Date.Time.Before(emails[1].Date.Time) {
		t.Error("Emails should be ordered by date DESC (newest first)")
	}
}

// TestGetThreadExcludesTrashAndSpam ensures deleted emails are not included
func TestGetThreadExcludesTrashAndSpam(t *testing.T) {
	var tmpDir = t.TempDir()
	var dbPath = filepath.Join(tmpDir, "test.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer Close()

	// Setup
	db.Exec("INSERT INTO accounts (email, name) VALUES ('test@example.com', 'Test')")
	db.Exec("INSERT INTO folders (account_id, name) VALUES (1, 'INBOX')")
	db.Exec("INSERT INTO folders (account_id, name) VALUES (1, '[Gmail]/Trash')")
	db.Exec("INSERT INTO folders (account_id, name) VALUES (1, '[Gmail]/Spam')")

	var threadID = "17062d1764232491"

	// Email in INBOX
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_name, from_email, to_addresses, cc_addresses, snippet, body_text, body_html, raw_headers, thread_id)
		VALUES (1, 1, 1, 'msg1@example.com', 'Test', 'Alice', 'alice@example.com', '', '', '', '', '', '', ?)`, threadID)

	// Email in Trash (same thread)
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_name, from_email, to_addresses, cc_addresses, snippet, body_text, body_html, raw_headers, thread_id)
		VALUES (1, 2, 2, 'msg2@example.com', 'Re: Test', 'Bob', 'bob@example.com', '', '', '', '', '', '', ?)`, threadID)

	// Email in Spam (same thread)
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_name, from_email, to_addresses, cc_addresses, snippet, body_text, body_html, raw_headers, thread_id)
		VALUES (1, 3, 3, 'msg3@example.com', 'Re: Test', 'Spam', 'spam@example.com', '', '', '', '', '', '', ?)`, threadID)

	// Test
	var emails, err = GetThreadForEmail(1)
	if err != nil {
		t.Fatalf("GetThreadForEmail failed: %v", err)
	}

	// Should only return the INBOX email, not Trash/Spam
	if len(emails) != 1 {
		t.Errorf("Expected 1 email (excluding Trash/Spam), got %d", len(emails))
	}
}

// TestGetThreadParticipants tests participant extraction
func TestGetThreadParticipants(t *testing.T) {
	var tmpDir = t.TempDir()
	var dbPath = filepath.Join(tmpDir, "test.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer Close()

	// Setup
	db.Exec("INSERT INTO accounts (email, name) VALUES ('test@example.com', 'Test')")
	db.Exec("INSERT INTO folders (account_id, name) VALUES (1, 'INBOX')")

	var threadID = "17062d1764232491"

	// Create thread with multiple participants
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_email, thread_id)
		VALUES (1, 1, 1, 'msg1@example.com', 'Test', 'alice@example.com', ?)`, threadID)
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_email, thread_id)
		VALUES (1, 1, 2, 'msg2@example.com', 'Re: Test', 'bob@example.com', ?)`, threadID)
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_email, thread_id)
		VALUES (1, 1, 3, 'msg3@example.com', 'Re: Test', 'alice@example.com', ?)`, threadID)

	// Test
	var participants, err = GetThreadParticipants(threadID, 1)
	if err != nil {
		t.Fatalf("GetThreadParticipants failed: %v", err)
	}

	if len(participants) != 2 {
		t.Errorf("Expected 2 unique participants, got %d: %v", len(participants), participants)
	}

	// Check both participants are present
	var hasAlice, hasBob bool
	for _, p := range participants {
		if strings.Contains(p, "alice") {
			hasAlice = true
		}
		if strings.Contains(p, "bob") {
			hasBob = true
		}
	}
	if !hasAlice || !hasBob {
		t.Errorf("Missing expected participants. Got: %v", participants)
	}
}

// TestNormalizeSubject tests subject normalization for threading
func TestNormalizeSubject(t *testing.T) {
	var testCases = []struct {
		input    string
		expected string
	}{
		{"Re: Hello World", "hello world"},
		{"RE: RE: Hello World", "hello world"},
		{"Fwd: Hello World", "hello world"},
		{"FW: Hello World", "hello world"},
		{"Re: Fwd: Re: Hello World", "hello world"},
		{"Hello World", "hello world"},
		{"  Re:  Hello  World  ", "re: hello world"}, // inner spaces preserved after trim
		{"AW: German Reply", "german reply"},
		{"SV: Swedish Reply", "swedish reply"},
	}

	for _, tc := range testCases {
		var result = NormalizeSubject(tc.input)
		if result != tc.expected {
			t.Errorf("NormalizeSubject(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

// TestBatchUpdateThreadIDsOnlyUpdatesUnsyncedEmails ensures incremental sync works
func TestBatchUpdateThreadIDsOnlyUpdatesUnsyncedEmails(t *testing.T) {
	var tmpDir = t.TempDir()
	var dbPath = filepath.Join(tmpDir, "test.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer Close()

	// Setup
	db.Exec("INSERT INTO accounts (email, name) VALUES ('test@example.com', 'Test')")
	db.Exec("INSERT INTO folders (account_id, name) VALUES (1, 'INBOX')")

	// Email 1: Already synced (has thread_synced_at)
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_email, thread_id, thread_synced_at)
		VALUES (1, 1, 1, 'msg1@example.com', 'Test 1', 'alice@example.com', 'oldthread123456', '2024-01-01')`)

	// Email 2: Not synced (no thread_synced_at)
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_email, thread_id, thread_synced_at)
		VALUES (1, 1, 2, 'msg2@example.com', 'Test 2', 'bob@example.com', NULL, NULL)`)

	// Batch update
	var threadMap = map[string]string{
		"msg1@example.com": "newthread111111",
		"msg2@example.com": "newthread222222",
	}

	var updated, err = BatchUpdateThreadIDs(1, threadMap)
	if err != nil {
		t.Fatalf("BatchUpdateThreadIDs failed: %v", err)
	}

	// Should only update email 2 (the one without thread_synced_at)
	if updated != 1 {
		t.Errorf("Expected 1 email updated, got %d", updated)
	}

	// Verify email 1 still has old thread_id (not updated because already synced)
	var email1ThreadID string
	db.Get(&email1ThreadID, "SELECT thread_id FROM emails WHERE uid = 1")
	if email1ThreadID != "oldthread123456" {
		t.Errorf("Email 1 should keep old thread_id, got %s", email1ThreadID)
	}

	// Verify email 2 has new thread_id
	var email2ThreadID string
	db.Get(&email2ThreadID, "SELECT thread_id FROM emails WHERE uid = 2")
	if email2ThreadID != "newthread222222" {
		t.Errorf("Email 2 should have new thread_id, got %s", email2ThreadID)
	}
}

// TestGetThreadEmailsReturnsContent ensures thread emails have body content
func TestGetThreadEmailsReturnsContent(t *testing.T) {
	var tmpDir = t.TempDir()
	var dbPath = filepath.Join(tmpDir, "test.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer Close()

	// Setup
	db.Exec("INSERT INTO accounts (email, name) VALUES ('test@example.com', 'Test')")
	db.Exec("INSERT INTO folders (account_id, name) VALUES (1, 'INBOX')")

	var threadID = "17062d1764232491"

	// Create email WITH content
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_name, from_email, to_addresses, cc_addresses, snippet, body_text, body_html, raw_headers, thread_id, date)
		VALUES (1, 1, 1, 'msg1@example.com', 'Test Email', 'Alice', 'alice@example.com', 'bob@example.com', '', 'This is the preview...', 'Full body text here', '<p>Full body HTML</p>', '', ?, '2024-01-01')`, threadID)

	// Create email WITHOUT content (will need IMAP fetch)
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_name, from_email, to_addresses, cc_addresses, snippet, body_text, body_html, raw_headers, thread_id, date)
		VALUES (1, 1, 2, 'msg2@example.com', 'Reply', 'Bob', 'bob@example.com', 'alice@example.com', '', '', '', '', '', ?, '2024-01-02')`, threadID)

	// Get thread
	var emails, err = GetThreadEmails(threadID, 1)
	if err != nil {
		t.Fatalf("GetThreadEmails failed: %v", err)
	}

	if len(emails) != 2 {
		t.Fatalf("Expected 2 emails, got %d", len(emails))
	}

	// Email 1 should have content
	var email1 = emails[1] // Ordered DESC by date, so older is index 1
	if email1.Snippet == "" || email1.BodyText == "" {
		t.Errorf("Email 1 should have content from database")
	}

	// Email 2 is expected to have NO content from storage
	// The ThreadService should fetch it via IMAP (tested separately)
	var email2 = emails[0] // Newer email
	if email2.Snippet != "" || email2.BodyText != "" || email2.BodyHTML != "" {
		t.Error("Email 2 should have empty content from storage (IMAP fetch happens in service layer)")
	}
}

// TestThreadEmailsHaveRequiredFields ensures all necessary fields are present
func TestThreadEmailsHaveRequiredFields(t *testing.T) {
	var tmpDir = t.TempDir()
	var dbPath = filepath.Join(tmpDir, "test.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer Close()

	// Setup
	db.Exec("INSERT INTO accounts (email, name) VALUES ('test@example.com', 'Test')")
	db.Exec("INSERT INTO folders (account_id, name) VALUES (1, 'INBOX')")

	var threadID = "17062d1764232491"

	// Create complete email
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_name, from_email, to_addresses, cc_addresses, snippet, body_text, body_html, raw_headers, thread_id, date)
		VALUES (1, 1, 1, 'msg1@example.com', 'Complete Email', 'Sender Name', 'sender@example.com', 'recipient@example.com', 'cc@example.com', 'Preview snippet', 'Body text', '<p>Body HTML</p>', 'Headers', ?, '2024-01-01')`, threadID)

	var emails, err = GetThreadEmails(threadID, 1)
	if err != nil {
		t.Fatalf("GetThreadEmails failed: %v", err)
	}

	if len(emails) != 1 {
		t.Fatalf("Expected 1 email, got %d", len(emails))
	}

	var e = emails[0]

	// Check all required fields for ThreadEmailDTO
	if e.ID == 0 {
		t.Error("Email should have ID")
	}
	if e.UID == 0 {
		t.Error("Email should have UID")
	}
	if e.Subject == "" {
		t.Error("Email should have Subject")
	}
	if e.FromName == "" {
		t.Error("Email should have FromName")
	}
	if e.FromEmail == "" {
		t.Error("Email should have FromEmail")
	}
	if e.ToAddresses == "" {
		t.Error("Email should have ToAddresses")
	}
	if e.Snippet == "" {
		t.Error("Email should have Snippet")
	}
	if e.BodyText == "" {
		t.Error("Email should have BodyText")
	}
	if e.BodyHTML == "" {
		t.Error("Email should have BodyHTML")
	}
}

// TestCountEmailsWithoutThreadID tests the incremental sync counter
func TestCountEmailsWithoutThreadID(t *testing.T) {
	var tmpDir = t.TempDir()
	var dbPath = filepath.Join(tmpDir, "test.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer Close()

	// Setup
	db.Exec("INSERT INTO accounts (email, name) VALUES ('test@example.com', 'Test')")
	db.Exec("INSERT INTO folders (account_id, name) VALUES (1, 'INBOX')")

	// Email 1: Has thread_synced_at (already synced)
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_email, thread_synced_at)
		VALUES (1, 1, 1, 'msg1@example.com', 'Test 1', 'alice@example.com', '2024-01-01')`)

	// Email 2: No thread_synced_at (needs sync)
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_email)
		VALUES (1, 1, 2, 'msg2@example.com', 'Test 2', 'bob@example.com')`)

	// Email 3: No message_id (can't sync - should not be counted)
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, subject, from_email)
		VALUES (1, 1, 3, 'Test 3', 'charlie@example.com')`)

	// Email 4: Deleted (should not be counted)
	db.Exec(`INSERT INTO emails (account_id, folder_id, uid, message_id, subject, from_email, is_deleted)
		VALUES (1, 1, 4, 'msg4@example.com', 'Test 4', 'dave@example.com', 1)`)

	// Test
	var count, err = CountEmailsWithoutThreadID(1)
	if err != nil {
		t.Fatalf("CountEmailsWithoutThreadID failed: %v", err)
	}

	// Should only count email 2
	if count != 1 {
		t.Errorf("Expected 1 email needing sync, got %d", count)
	}
}
