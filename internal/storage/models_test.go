package storage

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestEmailStructMatchesSchema ensures the Email struct has all columns from the emails table
// This prevents runtime errors like "missing destination name X in *storage.Email"
func TestEmailStructMatchesSchema(t *testing.T) {
	// Create temp database
	var tmpDir = t.TempDir()
	var dbPath = filepath.Join(tmpDir, "test.db")

	// Initialize database with schema
	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer Close()

	// Get actual columns from database
	var rows, err = db.Query("PRAGMA table_info(emails)")
	if err != nil {
		t.Fatalf("Failed to get table info: %v", err)
	}
	defer rows.Close()

	var dbColumns = make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		dbColumns[name] = true
	}

	// Get struct fields
	var emailType = reflect.TypeOf(Email{})
	var structFields = make(map[string]bool)
	for i := 0; i < emailType.NumField(); i++ {
		var field = emailType.Field(i)
		var dbTag = field.Tag.Get("db")
		if dbTag != "" && dbTag != "-" {
			structFields[dbTag] = true
		}
	}

	// Check for columns in DB but not in struct (this causes runtime errors!)
	var missing []string
	for col := range dbColumns {
		if !structFields[col] {
			missing = append(missing, col)
		}
	}

	if len(missing) > 0 {
		t.Errorf("Database columns missing from Email struct (will cause 'missing destination name' error): %s",
			strings.Join(missing, ", "))
		t.Error("Add these fields to internal/storage/models.go Email struct")
	}

	// Check for struct fields not in DB (less critical, but good to know)
	var extra []string
	for field := range structFields {
		if !dbColumns[field] {
			extra = append(extra, field)
		}
	}

	if len(extra) > 0 {
		t.Logf("Warning: Email struct has fields not in database: %s", strings.Join(extra, ", "))
	}
}

// TestEmailSummaryStructMatchesUsage ensures EmailSummary has expected fields
func TestEmailSummaryStructMatchesUsage(t *testing.T) {
	// EmailSummary is used in SELECT queries, so verify it has the common fields
	var summaryType = reflect.TypeOf(EmailSummary{})
	var requiredFields = []string{
		"id", "uid", "message_id", "subject", "from_name", "from_email",
		"date", "is_read", "is_starred", "is_replied", "has_attachments", "snippet",
	}

	var structFields = make(map[string]bool)
	for i := 0; i < summaryType.NumField(); i++ {
		var field = summaryType.Field(i)
		var dbTag = field.Tag.Get("db")
		if dbTag != "" {
			structFields[dbTag] = true
		}
	}

	for _, required := range requiredFields {
		if !structFields[required] {
			t.Errorf("EmailSummary missing required field: %s", required)
		}
	}
}

// TestContactStructMatchesSchema ensures Contact struct matches contacts table
func TestContactStructMatchesSchema(t *testing.T) {
	var tmpDir = t.TempDir()
	var dbPath = filepath.Join(tmpDir, "test.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer Close()

	var rows, err = db.Query("PRAGMA table_info(contacts)")
	if err != nil {
		t.Fatalf("Failed to get table info: %v", err)
	}
	defer rows.Close()

	var dbColumns = make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		dbColumns[name] = true
	}

	var contactType = reflect.TypeOf(Contact{})
	var structFields = make(map[string]bool)
	for i := 0; i < contactType.NumField(); i++ {
		var field = contactType.Field(i)
		var dbTag = field.Tag.Get("db")
		if dbTag != "" && dbTag != "-" {
			structFields[dbTag] = true
		}
	}

	var missing []string
	for col := range dbColumns {
		if !structFields[col] {
			missing = append(missing, col)
		}
	}

	if len(missing) > 0 {
		t.Errorf("Database columns missing from Contact struct: %s", strings.Join(missing, ", "))
	}
}

// TestFolderStructMatchesSchema ensures Folder struct matches folders table
func TestFolderStructMatchesSchema(t *testing.T) {
	var tmpDir = t.TempDir()
	var dbPath = filepath.Join(tmpDir, "test.db")

	if err := Init(dbPath); err != nil {
		t.Fatalf("Failed to init database: %v", err)
	}
	defer Close()

	var rows, err = db.Query("PRAGMA table_info(folders)")
	if err != nil {
		t.Fatalf("Failed to get table info: %v", err)
	}
	defer rows.Close()

	var dbColumns = make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		dbColumns[name] = true
	}

	var folderType = reflect.TypeOf(Folder{})
	var structFields = make(map[string]bool)
	for i := 0; i < folderType.NumField(); i++ {
		var field = folderType.Field(i)
		var dbTag = field.Tag.Get("db")
		if dbTag != "" && dbTag != "-" {
			structFields[dbTag] = true
		}
	}

	var missing []string
	for col := range dbColumns {
		if !structFields[col] {
			missing = append(missing, col)
		}
	}

	if len(missing) > 0 {
		t.Errorf("Database columns missing from Folder struct: %s", strings.Join(missing, ", "))
	}
}

// Ensure temp files are cleaned up
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
