package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/opik/miau/internal/ports"
)

// ContactService implements ports.ContactService
type ContactService struct {
	mu       sync.RWMutex
	storage  ports.ContactStoragePort
	gmail    ports.GmailContactsPort
	events   ports.EventBus
	photoDir string
}

// NewContactService creates a new ContactService
func NewContactService(storage ports.ContactStoragePort, gmail ports.GmailContactsPort, events ports.EventBus, photoDir string) *ContactService {
	// Ensure photo directory exists
	if err := os.MkdirAll(photoDir, 0700); err != nil {
		log.Printf("Warning: failed to create photo directory: %v", err)
	}

	return &ContactService{
		storage:  storage,
		gmail:    gmail,
		events:   events,
		photoDir: photoDir,
	}
}

// SyncContacts performs a full or incremental sync of contacts from Gmail
func (s *ContactService) SyncContacts(ctx context.Context, accountID int64, fullSync bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get current sync status
	var syncStatus, err = s.storage.GetSyncStatus(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get sync status: %w", err)
	}

	// Initialize sync status if needed
	if syncStatus == nil {
		syncStatus = &ports.ContactSyncStatus{
			AccountID: accountID,
			Status:    "never_synced",
		}
	}

	// Update status to syncing
	syncStatus.Status = "syncing"
	if err := s.storage.UpdateSyncStatus(ctx, syncStatus); err != nil {
		return fmt.Errorf("failed to update sync status: %w", err)
	}

	// Publish event
	s.events.Publish(ports.ContactSyncStartedEvent{
		BaseEvent: ports.NewBaseEvent(ports.EventTypeContactSyncStarted),
		AccountID: accountID,
		FullSync:  fullSync,
	})

	// Determine sync token
	var syncToken string
	if !fullSync && syncStatus.LastSyncToken != "" {
		syncToken = syncStatus.LastSyncToken
	}

	var totalSynced = 0
	var nextPageToken = ""
	var newSyncToken = ""

	for {
		// Fetch page of contacts
		var contacts, pageToken, token, err2 = s.gmail.ListContacts(100, nextPageToken, syncToken)
		if err2 != nil {
			syncStatus.Status = "error"
			syncStatus.ErrorMessage = err2.Error()
			s.storage.UpdateSyncStatus(ctx, syncStatus)
			s.events.Publish(ports.ContactSyncFailedEvent{
				BaseEvent: ports.NewBaseEvent(ports.EventTypeContactSyncFailed),
				AccountID: accountID,
				Error:     err2.Error(),
			})
			return fmt.Errorf("failed to fetch contacts: %w", err2)
		}

		// Save sync token from first response
		if newSyncToken == "" && token != "" {
			newSyncToken = token
		}

		// Process each contact
		for _, person := range contacts {
			if err := s.savePersonContact(ctx, accountID, &person); err != nil {
				log.Printf("Warning: failed to save contact %s: %v", person.ResourceName, err)
			} else {
				totalSynced++
			}
		}

		// Check if there are more pages
		if pageToken == "" {
			break
		}
		nextPageToken = pageToken
	}

	// Update sync status
	var now = time.Now()
	syncStatus.Status = "synced"
	syncStatus.LastSyncToken = newSyncToken
	syncStatus.TotalContacts = totalSynced
	syncStatus.ErrorMessage = ""

	if fullSync {
		syncStatus.LastFullSync = &now
	} else {
		syncStatus.LastIncrementalSync = &now
	}

	if err := s.storage.UpdateSyncStatus(ctx, syncStatus); err != nil {
		log.Printf("Warning: failed to update sync status: %v", err)
	}

	// Publish success event
	s.events.Publish(ports.ContactSyncCompletedEvent{
		BaseEvent:   ports.NewBaseEvent(ports.EventTypeContactSyncCompleted),
		AccountID:   accountID,
		TotalSynced: totalSynced,
		FullSync:    fullSync,
	})

	log.Printf("Contact sync completed: %d contacts synced", totalSynced)
	return nil
}

// savePersonContact saves a Person contact to storage
func (s *ContactService) savePersonContact(ctx context.Context, accountID int64, person *ports.PersonContact) error {
	// Check if contact already exists
	var existing, _ = s.storage.GetContactByResourceName(ctx, accountID, person.ResourceName)

	var contactInfo = &ports.ContactInfo{
		AccountID:    accountID,
		ResourceName: person.ResourceName,
		DisplayName:  person.DisplayName,
		GivenName:    person.GivenName,
		FamilyName:   person.FamilyName,
	}

	// Handle photos
	if len(person.Photos) > 0 {
		for _, photo := range person.Photos {
			if !photo.IsDefault {
				contactInfo.PhotoURL = photo.URL
				break
			}
		}
		// Fallback to default photo if no custom photo
		if contactInfo.PhotoURL == "" && len(person.Photos) > 0 {
			contactInfo.PhotoURL = person.Photos[0].URL
		}
	}

	// If contact exists, preserve ID
	if existing != nil {
		contactInfo.ID = existing.ID
	}

	// Set synced timestamp
	var now = time.Now()
	contactInfo.SyncedAt = &now

	// Save contact
	var contactID, err = s.storage.SaveContact(ctx, contactInfo)
	if err != nil {
		return fmt.Errorf("failed to save contact: %w", err)
	}

	// Save emails
	if len(person.EmailAddresses) > 0 {
		var emails []ports.ContactEmailInfo
		for _, email := range person.EmailAddresses {
			emails = append(emails, ports.ContactEmailInfo{
				ContactID: contactID,
				Email:     email.Value,
				Type:      email.Type,
				IsPrimary: email.IsPrimary,
			})
		}
		if err := s.storage.SaveContactEmails(ctx, contactID, emails); err != nil {
			return fmt.Errorf("failed to save contact emails: %w", err)
		}
	}

	// Save phones
	if len(person.PhoneNumbers) > 0 {
		var phones []ports.ContactPhoneInfo
		for _, phone := range person.PhoneNumbers {
			phones = append(phones, ports.ContactPhoneInfo{
				ContactID:   contactID,
				PhoneNumber: phone.Value,
				Type:        phone.Type,
				IsPrimary:   phone.IsPrimary,
			})
		}
		if err := s.storage.SaveContactPhones(ctx, contactID, phones); err != nil {
			return fmt.Errorf("failed to save contact phones: %w", err)
		}
	}

	// Download photo if available and not cached
	if contactInfo.PhotoURL != "" && (existing == nil || existing.PhotoPath == "") {
		go s.downloadAndSavePhoto(contactID, contactInfo.PhotoURL)
	}

	return nil
}

// downloadAndSavePhoto downloads and saves a contact photo in background
func (s *ContactService) downloadAndSavePhoto(contactID int64, photoURL string) {
	var data, err = s.gmail.DownloadPhoto(photoURL)
	if err != nil {
		log.Printf("Warning: failed to download photo for contact %d: %v", contactID, err)
		return
	}

	// Save to file
	var filename = fmt.Sprintf("contact_%d.jpg", contactID)
	var photoPath = filepath.Join(s.photoDir, filename)

	if err := os.WriteFile(photoPath, data, 0600); err != nil {
		log.Printf("Warning: failed to save photo for contact %d: %v", contactID, err)
		return
	}

	// Update contact with photo path (best effort, no error handling)
	var ctx = context.Background()
	var contact, err2 = s.storage.GetContact(ctx, contactID)
	if err2 == nil {
		contact.PhotoPath = photoPath
		s.storage.SaveContact(ctx, contact)
	}
}

// GetContact returns a contact by ID
func (s *ContactService) GetContact(ctx context.Context, id int64) (*ports.ContactInfo, error) {
	return s.storage.GetContact(ctx, id)
}

// GetContactByEmail finds a contact by email address
func (s *ContactService) GetContactByEmail(ctx context.Context, accountID int64, email string) (*ports.ContactInfo, error) {
	return s.storage.GetContactByEmail(ctx, accountID, email)
}

// ListContacts returns all contacts for an account
func (s *ContactService) ListContacts(ctx context.Context, accountID int64, limit int) ([]ports.ContactInfo, error) {
	return s.storage.ListContacts(ctx, accountID, limit)
}

// SearchContacts searches contacts by name or email
func (s *ContactService) SearchContacts(ctx context.Context, accountID int64, query string, limit int) ([]ports.ContactInfo, error) {
	return s.storage.SearchContacts(ctx, accountID, query, limit)
}

// ExtractAndSaveContactFromEmail extracts contact info from an email and saves it
func (s *ContactService) ExtractAndSaveContactFromEmail(ctx context.Context, accountID int64, emailID int64) error {
	// TODO: implement email parsing to extract from/to contacts
	// This will be called automatically when syncing emails
	return nil
}

// GetContactPhoto returns the photo data for a contact
func (s *ContactService) GetContactPhoto(ctx context.Context, contactID int64) ([]byte, error) {
	var contact, err = s.storage.GetContact(ctx, contactID)
	if err != nil {
		return nil, err
	}

	if contact.PhotoPath == "" {
		return nil, fmt.Errorf("no photo available")
	}

	return os.ReadFile(contact.PhotoPath)
}

// GetSyncStatus returns the sync status for an account
func (s *ContactService) GetSyncStatus(ctx context.Context, accountID int64) (*ports.ContactSyncStatus, error) {
	return s.storage.GetSyncStatus(ctx, accountID)
}

// GetTopContacts returns contacts ordered by interaction frequency
func (s *ContactService) GetTopContacts(ctx context.Context, accountID int64, limit int) ([]ports.ContactInfo, error) {
	return s.storage.GetTopContacts(ctx, accountID, limit)
}
