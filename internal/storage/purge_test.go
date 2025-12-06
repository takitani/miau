package storage

import (
	"path/filepath"
	"testing"
	"time"
)

// TestPurgeDeletedFromServer tests that emails not on server are marked as deleted
func TestPurgeDeletedFromServer(t *testing.T) {
	var tmpDir = t.TempDir()
	var dbPath = filepath.Join(tmpDir, "test.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer Close()

	// Create test account and folder
	var account, err = GetOrCreateAccount("test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	var folder, err2 = GetOrCreateFolder(account.ID, "INBOX")
	if err2 != nil {
		t.Fatalf("Failed to create folder: %v", err2)
	}

	// Insert test emails with UIDs 1, 2, 3, 4, 5
	for uid := uint32(1); uid <= 5; uid++ {
		var email = Email{
			AccountID: account.ID,
			FolderID:  folder.ID,
			UID:       uid,
			Subject:   "Test email",
			FromEmail: "sender@example.com",
			Date:      SQLiteTime{time.Now()},
			IsDeleted: false,
		}
		if err := UpsertEmail(&email); err != nil {
			t.Fatalf("Failed to insert email UID %d: %v", uid, err)
		}
	}

	// Verify 5 emails exist
	var emails, _ = GetEmails(account.ID, folder.ID, 100, 0)
	if len(emails) != 5 {
		t.Fatalf("Expected 5 emails, got %d", len(emails))
	}

	// Simulate server returning only UIDs 1, 3, 5 (2 and 4 were deleted on server)
	var serverUIDs = []uint32{1, 3, 5}

	// Run purge
	var purged, err3 = PurgeDeletedFromServer(account.ID, folder.ID, serverUIDs)
	if err3 != nil {
		t.Fatalf("PurgeDeletedFromServer failed: %v", err3)
	}

	// Should have purged 2 emails (UIDs 2 and 4)
	if purged != 2 {
		t.Errorf("Expected 2 purged emails, got %d", purged)
	}

	// Verify only 3 emails are not deleted
	emails, _ = GetEmails(account.ID, folder.ID, 100, 0)
	if len(emails) != 3 {
		t.Errorf("Expected 3 remaining emails, got %d", len(emails))
	}

	// Verify the correct UIDs remain
	var remainingUIDs = make(map[uint32]bool)
	for _, e := range emails {
		remainingUIDs[e.UID] = true
	}

	for _, expected := range []uint32{1, 3, 5} {
		if !remainingUIDs[expected] {
			t.Errorf("Expected UID %d to remain, but it was deleted", expected)
		}
	}
}

// TestPurgeDeletedFromServerEmptyServerList tests that empty server list doesn't delete anything
func TestPurgeDeletedFromServerEmptyServerList(t *testing.T) {
	var tmpDir = t.TempDir()
	var dbPath = filepath.Join(tmpDir, "test.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer Close()

	var account, _ = GetOrCreateAccount("test@example.com", "Test User")
	var folder, _ = GetOrCreateFolder(account.ID, "INBOX")

	// Insert test email
	var email = Email{
		AccountID: account.ID,
		FolderID:  folder.ID,
		UID:       1,
		Subject:   "Test email",
		FromEmail: "sender@example.com",
		Date:      SQLiteTime{time.Now()},
		IsDeleted: false,
	}
	UpsertEmail(&email)

	// Purge with empty server list (simulates connection error - should NOT delete)
	var purged, _ = PurgeDeletedFromServer(account.ID, folder.ID, []uint32{})

	if purged != 0 {
		t.Errorf("Expected 0 purged with empty server list, got %d", purged)
	}

	// Email should still exist
	var emails, _ = GetEmails(account.ID, folder.ID, 100, 0)
	if len(emails) != 1 {
		t.Errorf("Email should not be deleted when server list is empty")
	}
}

// TestPurgeDeletedFromServerNilServerList tests that nil server list doesn't delete anything
func TestPurgeDeletedFromServerNilServerList(t *testing.T) {
	var tmpDir = t.TempDir()
	var dbPath = filepath.Join(tmpDir, "test.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer Close()

	var account, _ = GetOrCreateAccount("test@example.com", "Test User")
	var folder, _ = GetOrCreateFolder(account.ID, "INBOX")

	var email = Email{
		AccountID: account.ID,
		FolderID:  folder.ID,
		UID:       1,
		Subject:   "Test email",
		FromEmail: "sender@example.com",
		Date:      SQLiteTime{time.Now()},
		IsDeleted: false,
	}
	UpsertEmail(&email)

	// Purge with nil server list
	var purged, _ = PurgeDeletedFromServer(account.ID, folder.ID, nil)

	if purged != 0 {
		t.Errorf("Expected 0 purged with nil server list, got %d", purged)
	}

	var emails, _ = GetEmails(account.ID, folder.ID, 100, 0)
	if len(emails) != 1 {
		t.Errorf("Email should not be deleted when server list is nil")
	}
}
