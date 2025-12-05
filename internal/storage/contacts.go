package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/opik/miau/internal/ports"
)

// ContactStorageAdapter implements ports.ContactStoragePort
type ContactStorageAdapter struct{}

// NewContactStorageAdapter creates a new ContactStorageAdapter
func NewContactStorageAdapter() *ContactStorageAdapter {
	return &ContactStorageAdapter{}
}

// SaveContact saves or updates a contact
func (a *ContactStorageAdapter) SaveContact(ctx context.Context, contact *ports.ContactInfo) (int64, error) {
	var now = time.Now()

	if contact.ID == 0 {
		// Insert new contact
		var query = `
			INSERT INTO contacts (
				account_id, resource_name, display_name, given_name, family_name,
				photo_url, photo_path, is_starred, synced_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		var result, err = db.ExecContext(ctx, query,
			contact.AccountID, contact.ResourceName, contact.DisplayName,
			nullString(contact.GivenName), nullString(contact.FamilyName),
			nullString(contact.PhotoURL), nullString(contact.PhotoPath),
			contact.IsStarred, nullTime(contact.SyncedAt), now, now,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to insert contact: %w", err)
		}

		var id, err2 = result.LastInsertId()
		if err2 != nil {
			return 0, fmt.Errorf("failed to get insert ID: %w", err2)
		}

		return id, nil
	}

	// Update existing contact
	var query = `
		UPDATE contacts SET
			display_name = ?, given_name = ?, family_name = ?,
			photo_url = ?, photo_path = ?,
			is_starred = ?, synced_at = ?, updated_at = ?
		WHERE id = ?
	`
	var _, err = db.ExecContext(ctx, query,
		contact.DisplayName, nullString(contact.GivenName), nullString(contact.FamilyName),
		nullString(contact.PhotoURL), nullString(contact.PhotoPath),
		contact.IsStarred, nullTime(contact.SyncedAt), now,
		contact.ID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to update contact: %w", err)
	}

	return contact.ID, nil
}

// GetContact returns a contact by ID
func (a *ContactStorageAdapter) GetContact(ctx context.Context, id int64) (*ports.ContactInfo, error) {
	var contact Contact
	var query = `SELECT * FROM contacts WHERE id = ?`

	if err := db.GetContext(ctx, &contact, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("contact not found")
		}
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	return contactToPort(&contact), nil
}

// GetContactByResourceName finds a contact by Google resource name
func (a *ContactStorageAdapter) GetContactByResourceName(ctx context.Context, accountID int64, resourceName string) (*ports.ContactInfo, error) {
	var contact Contact
	var query = `SELECT * FROM contacts WHERE account_id = ? AND resource_name = ?`

	if err := db.GetContext(ctx, &contact, query, accountID, resourceName); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // not found, return nil without error
		}
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	return contactToPort(&contact), nil
}

// GetContactByEmail finds a contact by email address
func (a *ContactStorageAdapter) GetContactByEmail(ctx context.Context, accountID int64, email string) (*ports.ContactInfo, error) {
	var query = `
		SELECT c.* FROM contacts c
		INNER JOIN contact_emails ce ON c.id = ce.contact_id
		WHERE c.account_id = ? AND ce.email = ?
		LIMIT 1
	`

	var contact Contact
	if err := db.GetContext(ctx, &contact, query, accountID, email); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("contact not found")
		}
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}

	return contactToPort(&contact), nil
}

// ListContacts returns all contacts for an account
func (a *ContactStorageAdapter) ListContacts(ctx context.Context, accountID int64, limit int) ([]ports.ContactInfo, error) {
	var query = `
		SELECT * FROM contacts
		WHERE account_id = ?
		ORDER BY display_name ASC
		LIMIT ?
	`

	var contacts []Contact
	if err := db.SelectContext(ctx, &contacts, query, accountID, limit); err != nil {
		return nil, fmt.Errorf("failed to list contacts: %w", err)
	}

	var result []ports.ContactInfo
	for i := range contacts {
		result = append(result, *contactToPort(&contacts[i]))
	}

	return result, nil
}

// SearchContacts searches contacts by name or email
func (a *ContactStorageAdapter) SearchContacts(ctx context.Context, accountID int64, query string, limit int) ([]ports.ContactInfo, error) {
	var searchQuery = `
		SELECT DISTINCT c.* FROM contacts c
		LEFT JOIN contact_emails ce ON c.id = ce.contact_id
		WHERE c.account_id = ?
		AND (
			c.display_name LIKE ?
			OR c.given_name LIKE ?
			OR c.family_name LIKE ?
			OR ce.email LIKE ?
		)
		ORDER BY c.interaction_count DESC, c.display_name ASC
		LIMIT ?
	`

	var searchPattern = "%" + query + "%"
	var contacts []Contact
	if err := db.SelectContext(ctx, &contacts, searchQuery,
		accountID, searchPattern, searchPattern, searchPattern, searchPattern, limit); err != nil {
		return nil, fmt.Errorf("failed to search contacts: %w", err)
	}

	var result []ports.ContactInfo
	for i := range contacts {
		result = append(result, *contactToPort(&contacts[i]))
	}

	return result, nil
}

// SaveContactEmails saves email addresses for a contact
func (a *ContactStorageAdapter) SaveContactEmails(ctx context.Context, contactID int64, emails []ports.ContactEmailInfo) error {
	// Delete existing emails
	var _, err = db.ExecContext(ctx, "DELETE FROM contact_emails WHERE contact_id = ?", contactID)
	if err != nil {
		return fmt.Errorf("failed to delete existing emails: %w", err)
	}

	// Insert new emails
	for _, email := range emails {
		var query = `
			INSERT INTO contact_emails (contact_id, email, email_type, is_primary)
			VALUES (?, ?, ?, ?)
		`
		var _, err2 = db.ExecContext(ctx, query, contactID, email.Email, email.Type, email.IsPrimary)
		if err2 != nil {
			return fmt.Errorf("failed to insert email: %w", err2)
		}
	}

	return nil
}

// SaveContactPhones saves phone numbers for a contact
func (a *ContactStorageAdapter) SaveContactPhones(ctx context.Context, contactID int64, phones []ports.ContactPhoneInfo) error {
	// Delete existing phones
	var _, err = db.ExecContext(ctx, "DELETE FROM contact_phones WHERE contact_id = ?", contactID)
	if err != nil {
		return fmt.Errorf("failed to delete existing phones: %w", err)
	}

	// Insert new phones
	for _, phone := range phones {
		var query = `
			INSERT INTO contact_phones (contact_id, phone_number, phone_type, is_primary)
			VALUES (?, ?, ?, ?)
		`
		var _, err2 = db.ExecContext(ctx, query, contactID, phone.PhoneNumber, phone.Type, phone.IsPrimary)
		if err2 != nil {
			return fmt.Errorf("failed to insert phone: %w", err2)
		}
	}

	return nil
}

// GetContactEmails returns email addresses for a contact
func (a *ContactStorageAdapter) GetContactEmails(ctx context.Context, contactID int64) ([]ports.ContactEmailInfo, error) {
	var query = `SELECT * FROM contact_emails WHERE contact_id = ? ORDER BY is_primary DESC`

	var emails []ContactEmail
	if err := db.SelectContext(ctx, &emails, query, contactID); err != nil {
		return nil, fmt.Errorf("failed to get contact emails: %w", err)
	}

	var result []ports.ContactEmailInfo
	for i := range emails {
		result = append(result, ports.ContactEmailInfo{
			ID:        emails[i].ID,
			ContactID: emails[i].ContactID,
			Email:     emails[i].Email,
			Type:      emails[i].EmailType,
			IsPrimary: emails[i].IsPrimary,
		})
	}

	return result, nil
}

// GetContactPhones returns phone numbers for a contact
func (a *ContactStorageAdapter) GetContactPhones(ctx context.Context, contactID int64) ([]ports.ContactPhoneInfo, error) {
	var query = `SELECT * FROM contact_phones WHERE contact_id = ? ORDER BY is_primary DESC`

	var phones []ContactPhone
	if err := db.SelectContext(ctx, &phones, query, contactID); err != nil {
		return nil, fmt.Errorf("failed to get contact phones: %w", err)
	}

	var result []ports.ContactPhoneInfo
	for i := range phones {
		result = append(result, ports.ContactPhoneInfo{
			ID:          phones[i].ID,
			ContactID:   phones[i].ContactID,
			PhoneNumber: phones[i].PhoneNumber,
			Type:        phones[i].PhoneType,
			IsPrimary:   phones[i].IsPrimary,
		})
	}

	return result, nil
}

// RecordInteraction records an email interaction with a contact
func (a *ContactStorageAdapter) RecordInteraction(ctx context.Context, contactID int64, emailID int64, interactionType string, interactionDate time.Time) error {
	// Insert interaction
	var query = `
		INSERT INTO contact_interactions (contact_id, email_id, interaction_type, interaction_date)
		VALUES (?, ?, ?, ?)
	`
	var _, err = db.ExecContext(ctx, query, contactID, nullInt64(emailID), interactionType, interactionDate)
	if err != nil {
		return fmt.Errorf("failed to record interaction: %w", err)
	}

	// Update contact interaction stats
	var updateQuery = `
		UPDATE contacts SET
			interaction_count = interaction_count + 1,
			last_interaction_at = ?
		WHERE id = ?
	`
	var _, err2 = db.ExecContext(ctx, updateQuery, interactionDate, contactID)
	if err2 != nil {
		return fmt.Errorf("failed to update contact stats: %w", err2)
	}

	return nil
}

// GetSyncStatus returns the sync status for an account
func (a *ContactStorageAdapter) GetSyncStatus(ctx context.Context, accountID int64) (*ports.ContactSyncStatus, error) {
	var query = `SELECT * FROM contacts_sync_state WHERE account_id = ?`

	var state ContactsSyncState
	if err := db.GetContext(ctx, &state, query, accountID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // not found, return nil without error
		}
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	return &ports.ContactSyncStatus{
		AccountID:           state.AccountID,
		LastSyncToken:       state.LastSyncToken.String,
		LastFullSync:        nullTimeToPtr(state.LastFullSync),
		LastIncrementalSync: nullTimeToPtr(state.LastIncrementalSync),
		TotalContacts:       state.TotalContacts,
		Status:              state.Status,
		ErrorMessage:        state.ErrorMessage.String,
	}, nil
}

// UpdateSyncStatus updates the sync status
func (a *ContactStorageAdapter) UpdateSyncStatus(ctx context.Context, status *ports.ContactSyncStatus) error {
	var now = time.Now()

	// Try update first
	var updateQuery = `
		UPDATE contacts_sync_state SET
			last_sync_token = ?,
			last_full_sync = ?,
			last_incremental_sync = ?,
			total_contacts = ?,
			status = ?,
			error_message = ?,
			updated_at = ?
		WHERE account_id = ?
	`
	var result, err = db.ExecContext(ctx, updateQuery,
		nullString(status.LastSyncToken),
		nullTime(status.LastFullSync),
		nullTime(status.LastIncrementalSync),
		status.TotalContacts,
		status.Status,
		nullString(status.ErrorMessage),
		now,
		status.AccountID,
	)
	if err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	var rowsAffected, _ = result.RowsAffected()
	if rowsAffected > 0 {
		return nil
	}

	// Insert if not exists
	var insertQuery = `
		INSERT INTO contacts_sync_state (
			account_id, last_sync_token, last_full_sync, last_incremental_sync,
			total_contacts, status, error_message, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	var _, err2 = db.ExecContext(ctx, insertQuery,
		status.AccountID,
		nullString(status.LastSyncToken),
		nullTime(status.LastFullSync),
		nullTime(status.LastIncrementalSync),
		status.TotalContacts,
		status.Status,
		nullString(status.ErrorMessage),
		now, now,
	)
	if err2 != nil {
		return fmt.Errorf("failed to insert sync status: %w", err2)
	}

	return nil
}

// GetTopContacts returns contacts ordered by interaction frequency
func (a *ContactStorageAdapter) GetTopContacts(ctx context.Context, accountID int64, limit int) ([]ports.ContactInfo, error) {
	var query = `
		SELECT * FROM contacts
		WHERE account_id = ?
		ORDER BY interaction_count DESC, last_interaction_at DESC
		LIMIT ?
	`

	var contacts []Contact
	if err := db.SelectContext(ctx, &contacts, query, accountID, limit); err != nil {
		return nil, fmt.Errorf("failed to get top contacts: %w", err)
	}

	var result []ports.ContactInfo
	for i := range contacts {
		result = append(result, *contactToPort(&contacts[i]))
	}

	return result, nil
}

// DeleteContactsByAccount deletes all contacts for an account
func (a *ContactStorageAdapter) DeleteContactsByAccount(ctx context.Context, accountID int64) error {
	// Delete contacts (cascade will handle emails, phones, interactions)
	var _, err = db.ExecContext(ctx, "DELETE FROM contacts WHERE account_id = ?", accountID)
	if err != nil {
		return fmt.Errorf("failed to delete contacts: %w", err)
	}

	// Delete sync state
	var _, err2 = db.ExecContext(ctx, "DELETE FROM contacts_sync_state WHERE account_id = ?", accountID)
	if err2 != nil {
		return fmt.Errorf("failed to delete sync state: %w", err2)
	}

	return nil
}

// Helper functions

func contactToPort(c *Contact) *ports.ContactInfo {
	return &ports.ContactInfo{
		ID:                c.ID,
		AccountID:         c.AccountID,
		ResourceName:      c.ResourceName,
		DisplayName:       c.DisplayName,
		GivenName:         c.GivenName.String,
		FamilyName:        c.FamilyName.String,
		PhotoURL:          c.PhotoURL.String,
		PhotoPath:         c.PhotoPath.String,
		IsStarred:         c.IsStarred,
		InteractionCount:  c.InteractionCount,
		LastInteractionAt: nullTimeToPtr(c.LastInteractionAt),
		SyncedAt:          nullTimeToPtr(c.SyncedAt),
		CreatedAt:         c.CreatedAt.Time,
		UpdatedAt:         c.UpdatedAt.Time,
	}
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullInt64(i int64) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: i, Valid: true}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func nullTimeToPtr(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
