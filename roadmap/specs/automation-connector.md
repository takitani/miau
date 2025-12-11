# PL-09: Zapier/n8n Connector

## Overview
Connect miau to automation platforms.

## Triggers
- New email received
- Email archived
- Email sent
- New contact added

## Actions
- Send email
- Archive email
- Create draft
- Add contact

## Webhook Format
```json
{
    "event": "email.received",
    "timestamp": "2024-12-15T10:30:00Z",
    "data": {
        "id": 123,
        "from": "john@example.com",
        "subject": "Hello",
        "snippet": "..."
    }
}
```

## Implementation
Uses existing webhook system from API server.

## Estimated Complexity
Low (uses existing API)
