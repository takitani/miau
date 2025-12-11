# EM-14: Canned Responses

## Overview

Quick pre-written responses that can be inserted with a few keystrokes, simpler than full templates.

## User Stories

1. As a user, I want to quickly insert common phrases while composing
2. As a user, I want to create and manage my canned responses
3. As a user, I want keyboard shortcuts for frequent responses
4. As a user, I want canned responses to expand inline

## Technical Requirements

### Service Layer

Create `internal/services/cannedresponses.go`:

```go
package services

type CannedResponseService interface {
    // GetResponses returns all canned responses
    GetResponses(ctx context.Context, accountID int64) ([]CannedResponse, error)

    // GetResponse returns a specific response
    GetResponse(ctx context.Context, id int64) (*CannedResponse, error)

    // CreateResponse creates a new canned response
    CreateResponse(ctx context.Context, resp CannedResponse) (*CannedResponse, error)

    // UpdateResponse updates a response
    UpdateResponse(ctx context.Context, resp CannedResponse) error

    // DeleteResponse deletes a response
    DeleteResponse(ctx context.Context, id int64) error

    // SearchResponses searches by keyword or shortcut
    SearchResponses(ctx context.Context, accountID int64, query string) ([]CannedResponse, error)
}

type CannedResponse struct {
    ID          int64
    AccountID   int64
    Shortcut    string  // e.g., "/ty" expands to full text
    Title       string  // For display in list
    Content     string  // The actual text
    Category    string
    UsageCount  int
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### Database Schema

```sql
CREATE TABLE canned_responses (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    shortcut TEXT,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    category TEXT,
    usage_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, shortcut)
);
```

### Default Responses

```go
var DefaultResponses = []CannedResponse{
    {Shortcut: "/ty", Title: "Thank you", Content: "Thank you for your email. I appreciate you reaching out."},
    {Shortcut: "/ack", Title: "Acknowledge", Content: "I received your message and will get back to you shortly."},
    {Shortcut: "/ooo", Title: "Out of office", Content: "I'm currently out of the office and will respond when I return."},
    {Shortcut: "/lmk", Title: "Let me know", Content: "Please let me know if you have any questions."},
    {Shortcut: "/br", Title: "Best regards", Content: "Best regards,"},
    {Shortcut: "/ty+", Title: "Thank you extended", Content: "Thank you for your email. I appreciate you taking the time to reach out. I'll review this and get back to you as soon as possible."},
}
```

### Inline Expansion

```go
// In compose component
func (c *Compose) handleInput(input string) {
    // Check if input starts with shortcut prefix
    if strings.HasPrefix(input, "/") {
        response, err := c.cannedService.SearchResponses(ctx, c.accountID, input)
        if err == nil && len(response) == 1 {
            // Auto-expand if exact match
            c.insertText(response[0].Content)
            c.cannedService.IncrementUsage(ctx, response[0].ID)
            return
        }
        // Show suggestions if partial match
        if len(response) > 1 {
            c.showSuggestions(response)
        }
    }
}
```

## UI/UX

### TUI
- Type `/shortcut` to expand
- Press `Ctrl+/` to browse all responses

```
┌─ Compose ─────────────────────────────────────────────────────────┐
│ To: john@example.com                                              │
│ Subject: RE: Meeting                                              │
├───────────────────────────────────────────────────────────────────┤
│ Hi John,                                                          │
│                                                                   │
│ /ty_                                                              │
│ ┌─ Suggestions ─────────────────────────────────┐                 │
│ │ /ty  - Thank you                              │                 │
│ │ /ty+ - Thank you extended                     │                 │
│ └───────────────────────────────────────────────┘                 │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘

After pressing Enter or Tab:

│ Hi John,                                                          │
│                                                                   │
│ Thank you for your email. I appreciate you reaching out.          │
│ _                                                                 │
```

### Canned Response Manager

```
┌─ Canned Responses ────────────────────────────────────────────────┐
│                                                                   │
│ Greetings                                                         │
│   /ty   - Thank you                              Used: 45x        │
│   /ty+  - Thank you extended                     Used: 12x        │
│                                                                   │
│ Closings                                                          │
│   /br   - Best regards                           Used: 89x        │
│   /lmk  - Let me know                            Used: 34x        │
│                                                                   │
│ Status                                                            │
│   /ack  - Acknowledge                            Used: 23x        │
│   /ooo  - Out of office                          Used: 5x         │
│                                                                   │
├───────────────────────────────────────────────────────────────────┤
│ [n] New  [e] Edit  [d] Delete  [Enter] Preview                    │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Autocomplete dropdown while typing
- Response manager in settings
- Drag to reorder
- Import/export

## Testing

1. Test shortcut expansion
2. Test partial match suggestions
3. Test CRUD operations
4. Test usage tracking
5. Test search functionality

## Acceptance Criteria

- [ ] Shortcuts expand inline when typed
- [ ] Partial matches show suggestions
- [ ] Can create/edit/delete responses
- [ ] Can assign custom shortcuts
- [ ] Usage tracked
- [ ] Can organize by category
- [ ] Default responses provided
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
canned_responses:
  enabled: true
  shortcut_prefix: "/"
  expand_on_space: true  # Expand when space pressed after shortcut
```

## Estimated Complexity

Low-Medium - Simple CRUD plus inline expansion
