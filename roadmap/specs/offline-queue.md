# TH-12: Offline Queue

## Overview

Queue email operations when offline and sync when connection restored.

## User Stories

1. As a user, I want to compose emails offline
2. As a user, I want actions queued when offline
3. As a user, I want queued actions synced when back online

## Technical Requirements

### Queue Storage

```sql
CREATE TABLE offline_queue (
    id INTEGER PRIMARY KEY,
    account_id INTEGER,
    action TEXT NOT NULL,  -- send, archive, delete, mark_read, etc.
    payload TEXT NOT NULL,  -- JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'pending',
    synced_at DATETIME,
    error TEXT
);
```

### Service Layer

```go
type OfflineQueueService interface {
    QueueAction(ctx context.Context, action QueuedAction) error
    ProcessQueue(ctx context.Context) error
    GetPendingActions(ctx context.Context) ([]QueuedAction, error)
    IsOnline() bool
}

type QueuedAction struct {
    Action  string  // "send", "archive", "delete"
    Payload interface{}
}
```

## Acceptance Criteria

- [ ] Actions queued when offline
- [ ] Visual indicator of pending actions
- [ ] Auto-process when online
- [ ] Conflict resolution

## Estimated Complexity

Medium
