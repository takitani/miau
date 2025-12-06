package ports

import (
	"context"
	"time"
)

// ContactService defines the interface for contact operations
type ContactService interface {
	// SyncContacts performs a full or incremental sync of contacts from Gmail
	SyncContacts(ctx context.Context, accountID int64, fullSync bool) error

	// GetContact returns a contact by ID
	GetContact(ctx context.Context, id int64) (*ContactInfo, error)

	// GetContactByEmail finds a contact by email address
	GetContactByEmail(ctx context.Context, accountID int64, email string) (*ContactInfo, error)

	// ListContacts returns all contacts for an account
	ListContacts(ctx context.Context, accountID int64, limit int) ([]ContactInfo, error)

	// SearchContacts searches contacts by name or email
	SearchContacts(ctx context.Context, accountID int64, query string, limit int) ([]ContactInfo, error)

	// ExtractAndSaveContactFromEmail extracts contact info from an email and saves it
	ExtractAndSaveContactFromEmail(ctx context.Context, accountID int64, emailID int64) error

	// GetContactPhoto returns the photo data for a contact
	GetContactPhoto(ctx context.Context, contactID int64) ([]byte, error)

	// GetSyncStatus returns the sync status for an account
	GetSyncStatus(ctx context.Context, accountID int64) (*ContactSyncStatus, error)

	// GetTopContacts returns contacts ordered by interaction frequency
	GetTopContacts(ctx context.Context, accountID int64, limit int) ([]ContactInfo, error)
}

// ContactInfo represents contact information
type ContactInfo struct {
	ID                int64
	AccountID         int64
	ResourceName      string
	DisplayName       string
	GivenName         string
	FamilyName        string
	PhotoURL          string
	PhotoPath         string
	IsStarred         bool
	InteractionCount  int
	LastInteractionAt *time.Time
	SyncedAt          *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time

	// Related data
	Emails []ContactEmailInfo
	Phones []ContactPhoneInfo
}

// ContactEmailInfo represents an email address for a contact
type ContactEmailInfo struct {
	ID        int64
	ContactID int64
	Email     string
	Type      string
	IsPrimary bool
}

// ContactPhoneInfo represents a phone number for a contact
type ContactPhoneInfo struct {
	ID          int64
	ContactID   int64
	PhoneNumber string
	Type        string
	IsPrimary   bool
}

// ContactSyncStatus represents the sync status for contacts
type ContactSyncStatus struct {
	AccountID           int64
	LastSyncToken       string
	LastFullSync        *time.Time
	LastIncrementalSync *time.Time
	TotalContacts       int
	Status              string
	ErrorMessage        string
}

// ContactStoragePort defines the interface for contact storage operations
type ContactStoragePort interface {
	// SaveContact saves or updates a contact
	SaveContact(ctx context.Context, contact *ContactInfo) (int64, error)

	// GetContact returns a contact by ID
	GetContact(ctx context.Context, id int64) (*ContactInfo, error)

	// GetContactByResourceName finds a contact by Google resource name
	GetContactByResourceName(ctx context.Context, accountID int64, resourceName string) (*ContactInfo, error)

	// GetContactByEmail finds a contact by email address
	GetContactByEmail(ctx context.Context, accountID int64, email string) (*ContactInfo, error)

	// ListContacts returns all contacts for an account
	ListContacts(ctx context.Context, accountID int64, limit int) ([]ContactInfo, error)

	// SearchContacts searches contacts by name or email
	SearchContacts(ctx context.Context, accountID int64, query string, limit int) ([]ContactInfo, error)

	// SaveContactEmails saves email addresses for a contact
	SaveContactEmails(ctx context.Context, contactID int64, emails []ContactEmailInfo) error

	// SaveContactPhones saves phone numbers for a contact
	SaveContactPhones(ctx context.Context, contactID int64, phones []ContactPhoneInfo) error

	// GetContactEmails returns email addresses for a contact
	GetContactEmails(ctx context.Context, contactID int64) ([]ContactEmailInfo, error)

	// GetContactPhones returns phone numbers for a contact
	GetContactPhones(ctx context.Context, contactID int64) ([]ContactPhoneInfo, error)

	// RecordInteraction records an email interaction with a contact
	RecordInteraction(ctx context.Context, contactID int64, emailID int64, interactionType string, interactionDate time.Time) error

	// GetSyncStatus returns the sync status for an account
	GetSyncStatus(ctx context.Context, accountID int64) (*ContactSyncStatus, error)

	// UpdateSyncStatus updates the sync status
	UpdateSyncStatus(ctx context.Context, status *ContactSyncStatus) error

	// GetTopContacts returns contacts ordered by interaction frequency
	GetTopContacts(ctx context.Context, accountID int64, limit int) ([]ContactInfo, error)

	// DeleteContactsByAccount deletes all contacts for an account
	DeleteContactsByAccount(ctx context.Context, accountID int64) error
}

// GmailContactsPort defines the interface for Gmail People API operations
type GmailContactsPort interface {
	// ListContacts fetches contacts from Gmail People API
	ListContacts(pageSize int, pageToken string, syncToken string) ([]PersonContact, string, string, error)

	// ListOtherContacts fetches "Other Contacts" (auto-suggested from emails)
	ListOtherContacts(pageSize int, pageToken string) ([]PersonContact, string, error)

	// GetContact fetches a single contact by resource name
	GetContact(resourceName string) (*PersonContact, error)

	// DownloadPhoto downloads a contact's profile photo
	DownloadPhoto(photoURL string) ([]byte, error)
}

// PersonContact represents a contact from Google People API
type PersonContact struct {
	ResourceName   string
	DisplayName    string
	GivenName      string
	FamilyName     string
	EmailAddresses []PersonEmail
	PhoneNumbers   []PersonPhone
	Photos         []PersonPhoto
}

// PersonEmail represents an email from People API
type PersonEmail struct {
	Value     string
	Type      string
	IsPrimary bool
}

// PersonPhone represents a phone from People API
type PersonPhone struct {
	Value     string
	Type      string
	IsPrimary bool
}

// PersonPhoto represents a photo from People API
type PersonPhoto struct {
	URL       string
	IsDefault bool
}
