# AI-09: AI Action Items Extraction

## Overview

Automatically extract action items, tasks, and deadlines from emails to help users track what needs to be done.

## User Stories

1. As a user, I want to see action items extracted from emails
2. As a user, I want extracted items to create tasks automatically
3. As a user, I want to see deadlines and due dates highlighted
4. As a user, I want a consolidated view of all pending action items

## Technical Requirements

### Service Layer

Create `internal/services/actionitems.go`:

```go
package services

type ActionItemsService interface {
    // ExtractFromEmail extracts action items from an email
    ExtractFromEmail(ctx context.Context, emailID int64) ([]ActionItem, error)

    // ExtractFromThread extracts action items from a thread
    ExtractFromThread(ctx context.Context, threadID string) ([]ActionItem, error)

    // GetPendingItems returns all unresolved action items
    GetPendingItems(ctx context.Context, accountID int64) ([]ActionItem, error)

    // ConvertToTask creates a task from an action item
    ConvertToTask(ctx context.Context, itemID int64) (*Task, error)

    // MarkComplete marks an action item as done
    MarkComplete(ctx context.Context, itemID int64) error

    // GetUpcoming returns action items with approaching deadlines
    GetUpcoming(ctx context.Context, accountID int64, days int) ([]ActionItem, error)
}

type ActionItem struct {
    ID          int64
    EmailID     int64
    ThreadID    string
    Description string
    Assignee    string      // Who should do this
    Deadline    *time.Time
    Priority    Priority
    Status      ItemStatus
    Source      string      // Quote from email
    CreatedAt   time.Time
}

type Priority string

const (
    PriorityHigh   Priority = "high"
    PriorityMedium Priority = "medium"
    PriorityLow    Priority = "low"
)

type ItemStatus string

const (
    StatusPending   ItemStatus = "pending"
    StatusCompleted ItemStatus = "completed"
    StatusSkipped   ItemStatus = "skipped"
)
```

### Database Schema

```sql
CREATE TABLE action_items (
    id INTEGER PRIMARY KEY,
    email_id INTEGER REFERENCES emails(id),
    thread_id TEXT,
    description TEXT NOT NULL,
    assignee TEXT,
    deadline DATETIME,
    priority TEXT DEFAULT 'medium',
    status TEXT DEFAULT 'pending',
    source_quote TEXT,
    task_id INTEGER REFERENCES tasks(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_action_items_status ON action_items(status);
CREATE INDEX idx_action_items_deadline ON action_items(deadline);
```

### AI Prompt Template

```go
var actionItemsPrompt = `Extract action items, tasks, and deadlines from this email.

From: {{.From}}
To: {{.To}}
Subject: {{.Subject}}
Body: {{.Body}}

Look for:
- Explicit requests ("please do X", "can you Y")
- Implicit tasks (things that need follow-up)
- Deadlines and due dates
- Assignments (who should do what)

Output JSON array:
[
  {
    "description": "Review the proposal document",
    "assignee": "me",  // "me" if assigned to recipient, sender name otherwise
    "deadline": "2024-12-15T17:00:00Z",  // null if no deadline
    "priority": "high",
    "source": "Please review the attached proposal by Friday"
  }
]`
```

## UI/UX

### TUI
- Press `A` (capital) to view action items panel
- Action items shown in email viewer footer
- Quick key to create task from item

```
┌─ Action Items (from: john@example.com) ───────────────────────────┐
│ ☐ Review the proposal document              Due: Dec 15          │
│ ☐ Schedule follow-up meeting                Due: Dec 12          │
│ ☐ Send updated budget numbers               No deadline          │
└───────────────────────────────────────────────────────────────────┘
  Space: mark done   t: create task   Enter: view email   q: close
```

### Desktop
- Action items panel in email viewer
- Consolidated action items view (all emails)
- Integration with tasks panel
- Calendar view of deadlines

## Testing

1. Test extraction from various email formats
2. Test deadline parsing (relative and absolute dates)
3. Test assignee detection
4. Test task conversion
5. Test with thread context
6. Test with non-English emails

## Acceptance Criteria

- [ ] Extracts action items from emails on view/sync
- [ ] Detects deadlines and due dates
- [ ] Identifies who should complete the action
- [ ] Can convert action items to tasks
- [ ] Shows consolidated view of pending items
- [ ] Handles threads (doesn't duplicate items)
- [ ] Works in both TUI and Desktop

## Configuration

```yaml
# config.yaml
ai:
  action_items:
    enabled: true
    extract_on_view: true
    auto_create_tasks: false
    notify_deadlines: true
```

## Dependencies

- AI service for extraction
- Tasks service for conversion
- Email service for content
- Notification service for deadline alerts

## Estimated Complexity

Medium-High - AI extraction plus task integration
