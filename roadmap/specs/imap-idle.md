# TH-05: IMAP IDLE (Push Notifications)

## Overview

Implement IMAP IDLE for real-time email notifications instead of polling.

## User Stories

1. As a user, I want to see new emails instantly
2. As a user, I want to reduce battery/CPU usage vs polling
3. As a user, I want to know when connection is lost
4. As a user, I want fallback to polling if IDLE unavailable

## Technical Requirements

### IMAP IDLE Implementation

```go
package imap

import (
    "context"
    "github.com/emersion/go-imap/v2"
    "github.com/emersion/go-imap/v2/imapclient"
)

type IdleManager struct {
    client    *imapclient.Client
    mailbox   string
    updates   chan<- IdleUpdate
    stopCh    chan struct{}
    isRunning bool
    mu        sync.Mutex
}

type IdleUpdate struct {
    Type      IdleUpdateType
    Mailbox   string
    NewCount  int
    ExpungeSeq []uint32
    Timestamp time.Time
}

type IdleUpdateType string

const (
    UpdateNewMail    IdleUpdateType = "new"
    UpdateExpunge    IdleUpdateType = "expunge"
    UpdateFlagsChange IdleUpdateType = "flags"
    UpdateReconnect  IdleUpdateType = "reconnect"
    UpdateError      IdleUpdateType = "error"
)

func NewIdleManager(client *imapclient.Client, updates chan<- IdleUpdate) *IdleManager {
    return &IdleManager{
        client:  client,
        updates: updates,
        stopCh:  make(chan struct{}),
    }
}

func (m *IdleManager) Start(ctx context.Context, mailbox string) error {
    m.mu.Lock()
    if m.isRunning {
        m.mu.Unlock()
        return errors.New("idle already running")
    }
    m.isRunning = true
    m.mailbox = mailbox
    m.mu.Unlock()

    go m.idleLoop(ctx)
    return nil
}

func (m *IdleManager) idleLoop(ctx context.Context) {
    defer func() {
        m.mu.Lock()
        m.isRunning = false
        m.mu.Unlock()
    }()

    for {
        select {
        case <-ctx.Done():
            return
        case <-m.stopCh:
            return
        default:
            err := m.doIdle(ctx)
            if err != nil {
                m.updates <- IdleUpdate{
                    Type:      UpdateError,
                    Timestamp: time.Now(),
                }
                // Wait before retry
                time.Sleep(5 * time.Second)
            }
        }
    }
}

func (m *IdleManager) doIdle(ctx context.Context) error {
    // Select mailbox if not already selected
    _, err := m.client.Select(m.mailbox, nil)
    if err != nil {
        return err
    }

    // Start IDLE
    idleCmd, err := m.client.Idle()
    if err != nil {
        return err
    }

    // Handle updates
    for {
        select {
        case <-ctx.Done():
            idleCmd.Close()
            return ctx.Err()

        case <-time.After(25 * time.Minute):
            // RFC recommends refreshing IDLE every 29 minutes
            idleCmd.Close()
            return nil  // Will restart idle

        case update := <-m.client.Updates():
            switch u := update.(type) {
            case *imapclient.UnilateralDataMailbox:
                if u.NumMessages != nil {
                    m.updates <- IdleUpdate{
                        Type:      UpdateNewMail,
                        Mailbox:   m.mailbox,
                        NewCount:  int(*u.NumMessages),
                        Timestamp: time.Now(),
                    }
                }
            case *imapclient.UnilateralDataExpunge:
                m.updates <- IdleUpdate{
                    Type:       UpdateExpunge,
                    Mailbox:    m.mailbox,
                    ExpungeSeq: []uint32{u.SeqNum},
                    Timestamp:  time.Now(),
                }
            }
        }
    }
}

func (m *IdleManager) Stop() {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.isRunning {
        close(m.stopCh)
    }
}
```

### Sync Service Integration

```go
func (s *SyncService) StartRealTimeSync(ctx context.Context, accountID int64) error {
    // Check if IDLE is supported
    caps, err := s.imap.Capabilities()
    if err != nil {
        return err
    }

    hasIdle := false
    for _, cap := range caps {
        if cap == "IDLE" {
            hasIdle = true
            break
        }
    }

    if hasIdle {
        return s.startIdleSync(ctx, accountID)
    } else {
        // Fallback to polling
        return s.startPollingSync(ctx, accountID)
    }
}

func (s *SyncService) startIdleSync(ctx context.Context, accountID int64) error {
    updates := make(chan IdleUpdate, 100)
    idleManager := NewIdleManager(s.imapClient, updates)

    go func() {
        for update := range updates {
            switch update.Type {
            case UpdateNewMail:
                // Fetch new emails
                s.syncNewEmails(ctx, accountID, update.Mailbox)
                // Notify UI
                s.eventBus.Publish("email.new", update)

            case UpdateExpunge:
                // Remove deleted emails
                s.handleExpunge(ctx, accountID, update.ExpungeSeq)
                s.eventBus.Publish("email.expunge", update)

            case UpdateError:
                log.Printf("IDLE error, reconnecting...")
                s.eventBus.Publish("sync.reconnecting", nil)
            }
        }
    }()

    return idleManager.Start(ctx, "INBOX")
}
```

### Connection Management

```go
type ConnectionManager struct {
    client       *imapclient.Client
    idleManager  *IdleManager
    reconnectCh  chan struct{}
    maxRetries   int
    retryDelay   time.Duration
}

func (m *ConnectionManager) monitorConnection(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := m.client.Noop(); err != nil {
                // Connection lost, reconnect
                m.reconnect(ctx)
            }
        case <-m.reconnectCh:
            m.reconnect(ctx)
        }
    }
}

func (m *ConnectionManager) reconnect(ctx context.Context) {
    for i := 0; i < m.maxRetries; i++ {
        err := m.client.Connect(ctx)
        if err == nil {
            // Restart IDLE
            m.idleManager.Start(ctx, "INBOX")
            return
        }
        time.Sleep(m.retryDelay * time.Duration(i+1))
    }
}
```

## UI/UX

### Connection Status Indicator

```
┌─ INBOX ─────────────────────────────────────────────── 🟢 Live ───┐
│ ...emails...                                                       │
└────────────────────────────────────────────────────────────────────┘

Status indicators:
🟢 Live     - IDLE active, real-time updates
🟡 Polling  - Fallback to interval sync
🔴 Offline  - Connection lost, retrying
⏳ Syncing  - Currently fetching updates
```

### Notification

```
┌─ New Email ───────────────────────────────────────────────────────┐
│ 📬 From: john@example.com                                         │
│ Subject: Project Update                                           │
│                                                                   │
│ [Enter] View  [Esc] Dismiss                                       │
└───────────────────────────────────────────────────────────────────┘
```

## Testing

1. Test IDLE with Gmail
2. Test IDLE with other providers
3. Test reconnection logic
4. Test fallback to polling
5. Test notification delivery
6. Test with network interruptions

## Acceptance Criteria

- [ ] IDLE implemented for real-time updates
- [ ] New emails appear instantly
- [ ] Connection status shown
- [ ] Automatic reconnection
- [ ] Fallback to polling if IDLE unsupported
- [ ] Notifications for new emails
- [ ] Handles network interruptions
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
sync:
  use_idle: true
  idle_timeout: "25m"
  polling_interval: "5m"  # Fallback
  reconnect_retries: 5
  reconnect_delay: "5s"
```

## Estimated Complexity

Medium-High - IMAP IDLE protocol plus connection management
