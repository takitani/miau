# TH-07: Background Sync Worker

## Overview

Implement a background worker that syncs emails independently of the UI.

## User Stories

1. As a user, I want email sync to continue while I compose
2. As a user, I want sync status visible but non-blocking
3. As a user, I want to pause/resume background sync
4. As a user, I want sync to run even when app is minimized

## Technical Requirements

### Worker Architecture

```go
package worker

import (
    "context"
    "sync"
    "time"
)

type SyncWorker struct {
    app          *app.Application
    eventBus     *EventBus
    interval     time.Duration
    isRunning    bool
    isPaused     bool
    mu           sync.RWMutex
    stopCh       chan struct{}
    pauseCh      chan struct{}
    resumeCh     chan struct{}
    currentJob   *SyncJob
}

type SyncJob struct {
    ID         string
    AccountID  int64
    Folder     string
    StartedAt  time.Time
    Status     SyncStatus
    Progress   SyncProgress
    Error      error
}

type SyncStatus string

const (
    SyncStatusIdle      SyncStatus = "idle"
    SyncStatusRunning   SyncStatus = "running"
    SyncStatusPaused    SyncStatus = "paused"
    SyncStatusError     SyncStatus = "error"
    SyncStatusCompleted SyncStatus = "completed"
)

type SyncProgress struct {
    Folder          string
    TotalFolders    int
    CurrentFolder   int
    NewEmails       int
    ProcessedEmails int
    DeletedEmails   int
}

func NewSyncWorker(app *app.Application, eventBus *EventBus, interval time.Duration) *SyncWorker {
    return &SyncWorker{
        app:      app,
        eventBus: eventBus,
        interval: interval,
        stopCh:   make(chan struct{}),
        pauseCh:  make(chan struct{}),
        resumeCh: make(chan struct{}),
    }
}

func (w *SyncWorker) Start(ctx context.Context) {
    w.mu.Lock()
    if w.isRunning {
        w.mu.Unlock()
        return
    }
    w.isRunning = true
    w.mu.Unlock()

    go w.run(ctx)
}

func (w *SyncWorker) run(ctx context.Context) {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()

    // Initial sync
    w.doSync(ctx)

    for {
        select {
        case <-ctx.Done():
            return
        case <-w.stopCh:
            return
        case <-w.pauseCh:
            w.mu.Lock()
            w.isPaused = true
            w.mu.Unlock()
            w.eventBus.Publish("sync.paused", nil)
            <-w.resumeCh
            w.mu.Lock()
            w.isPaused = false
            w.mu.Unlock()
            w.eventBus.Publish("sync.resumed", nil)
        case <-ticker.C:
            w.mu.RLock()
            paused := w.isPaused
            w.mu.RUnlock()
            if !paused {
                w.doSync(ctx)
            }
        }
    }
}

func (w *SyncWorker) doSync(ctx context.Context) {
    job := &SyncJob{
        ID:        generateID(),
        StartedAt: time.Now(),
        Status:    SyncStatusRunning,
    }

    w.mu.Lock()
    w.currentJob = job
    w.mu.Unlock()

    w.eventBus.Publish("sync.started", job)

    // Get all accounts
    accounts, err := w.app.Account().GetAccounts(ctx)
    if err != nil {
        job.Status = SyncStatusError
        job.Error = err
        w.eventBus.Publish("sync.error", job)
        return
    }

    for i, account := range accounts {
        job.AccountID = account.ID
        job.Progress.CurrentFolder = i + 1
        job.Progress.TotalFolders = len(accounts)

        // Sync each folder
        folders := w.app.Config().GetSyncFolders()
        for _, folder := range folders {
            job.Progress.Folder = folder

            result, err := w.app.Sync().SyncFolder(ctx, account.ID, folder)
            if err != nil {
                log.Printf("Sync error for %s/%s: %v", account.Email, folder, err)
                continue
            }

            job.Progress.NewEmails += result.NewCount
            job.Progress.DeletedEmails += result.DeletedCount

            w.eventBus.Publish("sync.progress", job)
        }
    }

    job.Status = SyncStatusCompleted
    w.eventBus.Publish("sync.completed", job)

    w.mu.Lock()
    w.currentJob = nil
    w.mu.Unlock()
}

func (w *SyncWorker) Pause() {
    w.pauseCh <- struct{}{}
}

func (w *SyncWorker) Resume() {
    w.resumeCh <- struct{}{}
}

func (w *SyncWorker) Stop() {
    close(w.stopCh)
}

func (w *SyncWorker) GetStatus() *SyncJob {
    w.mu.RLock()
    defer w.mu.RUnlock()
    return w.currentJob
}
```

### Event Bus Integration

```go
// In TUI model
func (m Model) handleSyncEvents() {
    m.eventBus.Subscribe("sync.started", func(data interface{}) {
        m.syncStatus = "Syncing..."
    })

    m.eventBus.Subscribe("sync.progress", func(data interface{}) {
        job := data.(*SyncJob)
        m.syncStatus = fmt.Sprintf("Syncing %s (%d/%d)",
            job.Progress.Folder,
            job.Progress.CurrentFolder,
            job.Progress.TotalFolders)
    })

    m.eventBus.Subscribe("sync.completed", func(data interface{}) {
        job := data.(*SyncJob)
        m.syncStatus = fmt.Sprintf("Synced: %d new", job.Progress.NewEmails)
        // Refresh email list
        m.loadEmails()
    })

    m.eventBus.Subscribe("sync.error", func(data interface{}) {
        job := data.(*SyncJob)
        m.syncStatus = fmt.Sprintf("Sync error: %v", job.Error)
    })
}
```

## UI/UX

### Status Bar

```
┌─ INBOX ─────────────────────────────────────────────────────────────┐
│ ...emails...                                                        │
├─────────────────────────────────────────────────────────────────────┤
│ 🔄 Syncing INBOX (2/5)... │ 3 new │ Last: 2 min ago │ [P]ause      │
└─────────────────────────────────────────────────────────────────────┘

States:
🔄 Syncing INBOX (2/5)...    - Active sync
✓ Synced: 3 new emails       - Completed
⏸ Sync paused                - Paused
❌ Sync error: connection    - Error with message
⏳ Next sync in 4:30         - Idle countdown
```

## Testing

1. Test background sync timing
2. Test pause/resume
3. Test error recovery
4. Test UI updates during sync
5. Test concurrent operations

## Acceptance Criteria

- [ ] Sync runs in background
- [ ] UI remains responsive during sync
- [ ] Progress visible in status bar
- [ ] Can pause/resume sync
- [ ] Errors don't crash app
- [ ] New emails trigger refresh
- [ ] Works when app minimized

## Configuration

```yaml
# config.yaml
sync:
  background: true
  interval: "5m"
  sync_folders: ["INBOX", "Sent"]
```

## Estimated Complexity

Medium - Goroutines plus event system
