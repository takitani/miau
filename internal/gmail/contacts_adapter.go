package gmail

import (
	"github.com/opik/miau/internal/ports"
)

// ContactsAdapter adapts Gmail Client to ports.GmailContactsPort
type ContactsAdapter struct {
	client *Client
}

// NewContactsAdapter creates a new ContactsAdapter
func NewContactsAdapter(client *Client) *ContactsAdapter {
	return &ContactsAdapter{client: client}
}

// ListContacts fetches contacts from Gmail People API
func (a *ContactsAdapter) ListContacts(pageSize int, pageToken string, syncToken string) ([]ports.PersonContact, string, string, error) {
	var req = &ListContactsRequest{
		PageSize:     pageSize,
		PageToken:    pageToken,
		SyncToken:    syncToken,
		PersonFields: "names,emailAddresses,phoneNumbers,photos",
	}

	var resp, err = a.client.ListContacts(req)
	if err != nil {
		return nil, "", "", err
	}

	var contacts []ports.PersonContact
	for _, person := range resp.Connections {
		var contact = ports.PersonContact{
			ResourceName: person.ResourceName,
		}

		// Extract names
		if len(person.Names) > 0 {
			var name = person.Names[0]
			contact.DisplayName = name.DisplayName
			contact.GivenName = name.GivenName
			contact.FamilyName = name.FamilyName
		}

		// Extract emails
		for _, email := range person.EmailAddresses {
			contact.EmailAddresses = append(contact.EmailAddresses, ports.PersonEmail{
				Value:     email.Value,
				Type:      email.Type,
				IsPrimary: email.Metadata.Primary,
			})
		}

		// Extract phones
		for _, phone := range person.PhoneNumbers {
			contact.PhoneNumbers = append(contact.PhoneNumbers, ports.PersonPhone{
				Value:     phone.Value,
				Type:      phone.Type,
				IsPrimary: phone.Metadata.Primary,
			})
		}

		// Extract photos
		for _, photo := range person.Photos {
			contact.Photos = append(contact.Photos, ports.PersonPhoto{
				URL:       photo.URL,
				IsDefault: photo.Default,
			})
		}

		contacts = append(contacts, contact)
	}

	return contacts, resp.NextPageToken, resp.NextSyncToken, nil
}

// GetContact fetches a single contact by resource name
func (a *ContactsAdapter) GetContact(resourceName string) (*ports.PersonContact, error) {
	var person, err = a.client.GetContact(resourceName, "names,emailAddresses,phoneNumbers,photos")
	if err != nil {
		return nil, err
	}

	var contact = &ports.PersonContact{
		ResourceName: person.ResourceName,
	}

	// Extract names
	if len(person.Names) > 0 {
		var name = person.Names[0]
		contact.DisplayName = name.DisplayName
		contact.GivenName = name.GivenName
		contact.FamilyName = name.FamilyName
	}

	// Extract emails
	for _, email := range person.EmailAddresses {
		contact.EmailAddresses = append(contact.EmailAddresses, ports.PersonEmail{
			Value:     email.Value,
			Type:      email.Type,
			IsPrimary: email.Metadata.Primary,
		})
	}

	// Extract phones
	for _, phone := range person.PhoneNumbers {
		contact.PhoneNumbers = append(contact.PhoneNumbers, ports.PersonPhone{
			Value:     phone.Value,
			Type:      phone.Type,
			IsPrimary: phone.Metadata.Primary,
		})
	}

	// Extract photos
	for _, photo := range person.Photos {
		contact.Photos = append(contact.Photos, ports.PersonPhoto{
			URL:       photo.URL,
			IsDefault: photo.Default,
		})
	}

	return contact, nil
}

// DownloadPhoto downloads a contact's profile photo
func (a *ContactsAdapter) DownloadPhoto(photoURL string) ([]byte, error) {
	return a.client.DownloadPhoto(photoURL)
}
