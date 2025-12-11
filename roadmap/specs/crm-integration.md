# IN-11: CRM Integration (HubSpot)

## Overview
Sync email activity with HubSpot CRM.

## Features
- Log emails to contact timeline
- Create contacts from emails
- Link deals to email threads
- Sync contact info

## Technical Requirements
```go
type HubSpotPlugin struct {
    apiKey string
}

func (p *HubSpotPlugin) OnEmailReceived(email *Email) error {
    // Find or create contact
    contact, _ := p.api.FindContactByEmail(email.FromEmail)
    if contact == nil {
        contact = p.api.CreateContact(Contact{
            Email: email.FromEmail,
            Name:  email.FromName,
        })
    }

    // Log email engagement
    return p.api.CreateEngagement(Engagement{
        Type:      "EMAIL",
        ContactID: contact.ID,
        Subject:   email.Subject,
        Body:      email.Snippet,
    })
}
```

## Estimated Complexity
Medium
