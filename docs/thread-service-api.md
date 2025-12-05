# Thread Service API Documentation

## Overview

The `ThreadService` is the **single source of truth** for email threading logic in miau. All UI layers (TUI, Desktop) **MUST** use this service and **NEVER** implement threading logic directly.

## Core Principles

### REGRA DE OURO (Golden Rule)

```
❌ NEVER: TUI/Desktop implementing threading logic
✅ ALWAYS: Use ThreadService through app.Thread()
```

**Why?**
- Bug fix in one place = fixed for TUI + Desktop
- Unit tests on Services cover both interfaces
- Prevents behavior divergence between UIs

## API Reference

### Get Complete Thread

```go
// Returns all messages in a thread for a given email ID
// Messages ordered DESC by date (newest first)
thread, err := app.Thread().GetThread(ctx, emailID)

// Thread struct:
type Thread struct {
    ThreadID     string           // Unique thread identifier
    Subject      string           // Thread subject
    Participants []string         // All senders in thread
    MessageCount int              // Total messages
    Messages     []EmailContent   // Full message content (DESC by date)
    IsRead       bool             // True if all messages read
}
```

**Example:**
```go
var emailID int64 = 12345
var thread, err = app.Thread().GetThread(ctx, emailID)
if err != nil {
    // Handle error
    return err
}

fmt.Printf("Thread: %s\n", thread.Subject)
fmt.Printf("Messages: %d\n", thread.MessageCount)
fmt.Printf("Participants: %v\n", thread.Participants)

// Iterate messages (newest first)
for _, msg := range thread.Messages {
    fmt.Printf("From: %s, Date: %s\n", msg.FromName, msg.Date)
    fmt.Printf("Body: %s\n", msg.BodyText)
}
```

### Get Thread by ID

```go
// Returns thread by thread_id (useful when you know the thread ID)
thread, err := app.Thread().GetThreadByID(ctx, threadID)
```

**Example:**
```go
var threadID = "abc123xyz"
var thread, err = app.Thread().GetThreadByID(ctx, threadID)
if err != nil {
    return err
}
```

### Get Thread Summary

```go
// Returns lightweight metadata without full message content
// Perfect for inbox list display
summary, err := app.Thread().GetThreadSummary(ctx, threadID)

// ThreadSummary struct:
type ThreadSummary struct {
    ThreadID        string
    Subject         string
    LastSender      string    // Name of last sender
    LastSenderEmail string
    LastDate        time.Time // Date of most recent message
    MessageCount    int
    UnreadCount     int
    HasAttachments  bool
    Participants    []string
}
```

**Example:**
```go
var summary, err = app.Thread().GetThreadSummary(ctx, threadID)
if err != nil {
    return err
}

fmt.Printf("%s (%d messages, %d unread)\n",
    summary.Subject,
    summary.MessageCount,
    summary.UnreadCount)
```

### Mark Thread as Read

```go
// Marks all messages in thread as read
err := app.Thread().MarkThreadAsRead(ctx, threadID)
```

**Events Published:**
- `ports.EventTypeThreadMarkedRead`
- Includes `ThreadID` and `Count` (number of messages marked)

**Example:**
```go
// User presses 'r' on thread in TUI
if err := app.Thread().MarkThreadAsRead(ctx, currentThreadID); err != nil {
    showError("Failed to mark thread as read")
}
```

### Mark Thread as Unread

```go
// Marks the most recent message in thread as unread
// (Only the latest message, not all messages)
err := app.Thread().MarkThreadAsUnread(ctx, threadID)
```

**Events Published:**
- `ports.EventTypeThreadMarkedUnread`

### Count Thread Messages

```go
// Returns message count without loading full thread
count, err := app.Thread().CountThreadMessages(ctx, threadID)
```

**Example:**
```go
var count, err = app.Thread().CountThreadMessages(ctx, threadID)
if err != nil {
    return err
}
fmt.Printf("Thread has %d messages\n", count)
```

## Usage Examples

### Desktop UI: Display Thread View

```go
// User clicks on thread in inbox
func (v *ThreadView) LoadThread(emailID int64) error {
    var ctx = context.Background()

    // Get full thread
    var thread, err = v.app.Thread().GetThread(ctx, emailID)
    if err != nil {
        return fmt.Errorf("failed to load thread: %w", err)
    }

    // Update UI state
    v.thread = thread
    v.expandedIndex = 0 // Expand first (newest) message

    // Render minimap
    v.renderMinimap(thread)

    // Render messages (newest first)
    for i, msg := range thread.Messages {
        if i == v.expandedIndex {
            v.renderExpandedMessage(msg)
        } else {
            v.renderCollapsedMessage(msg)
        }
    }

    return nil
}
```

### TUI: Navigate Thread

```go
// Handle keyboard input in thread view
func (m *ThreadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "j", "down":
            // Navigate to next message
            if m.selectedIndex < len(m.thread.Messages)-1 {
                m.selectedIndex++
            }

        case "k", "up":
            // Navigate to previous message
            if m.selectedIndex > 0 {
                m.selectedIndex--
            }

        case "enter":
            // Expand/collapse current message
            m.toggleExpanded(m.selectedIndex)

        case "r":
            // Mark thread as read
            var ctx = context.Background()
            if err := m.app.Thread().MarkThreadAsRead(ctx, m.thread.ThreadID); err != nil {
                return m, showError(err)
            }
        }
    }
    return m, nil
}
```

### Inbox List: Show Thread Summaries

```go
// Display threads in inbox (collapsed view)
func (v *InboxView) RenderThreadList() error {
    var ctx = context.Background()

    // Get emails from folder
    var emails, err = v.app.Email().GetEmails(ctx, "INBOX", 50)
    if err != nil {
        return err
    }

    // Group by thread_id and show summaries
    var threadsMap = make(map[string][]ports.EmailMetadata)
    for _, email := range emails {
        if email.ThreadID != "" {
            threadsMap[email.ThreadID] = append(threadsMap[email.ThreadID], email)
        }
    }

    // Render each thread
    for threadID, threadEmails := range threadsMap {
        var summary, err = v.app.Thread().GetThreadSummary(ctx, threadID)
        if err != nil {
            continue
        }

        // Render thread row
        v.renderThreadRow(summary)
    }

    return nil
}
```

### Listen to Thread Events

```go
// Subscribe to thread events
var unsubscribe = app.Events().Subscribe(
    ports.EventTypeThreadMarkedRead,
    func(event ports.Event) {
        var e = event.(ports.ThreadMarkedReadEvent)
        fmt.Printf("Thread %s marked as read (%d messages)\n",
            e.ThreadID, e.Count)

        // Update UI to reflect read status
        v.refreshThreadRow(e.ThreadID)
    },
)

// Don't forget to unsubscribe when done
defer unsubscribe()
```

## Error Handling

All methods return errors that should be handled appropriately:

```go
thread, err := app.Thread().GetThread(ctx, emailID)
if err != nil {
    // Common errors:
    // - "no account set" - Need to call SetAccount first
    // - "email not found" - Invalid email ID
    // - "thread not found" - Invalid thread ID
    // - Database errors

    log.Printf("Failed to get thread: %v", err)
    return err
}
```

## Testing

When testing UI components, mock the `ThreadService`:

```go
type MockThreadService struct {
    GetThreadFunc func(ctx context.Context, emailID int64) (*ports.Thread, error)
}

func (m *MockThreadService) GetThread(ctx context.Context, emailID int64) (*ports.Thread, error) {
    if m.GetThreadFunc != nil {
        return m.GetThreadFunc(ctx, emailID)
    }
    return nil, fmt.Errorf("not implemented")
}

// In test:
var mockThread = &ports.Thread{
    ThreadID:     "test-thread",
    Subject:      "Test Subject",
    MessageCount: 3,
    Messages:     []ports.EmailContent{/* ... */},
}

var mockService = &MockThreadService{
    GetThreadFunc: func(ctx context.Context, emailID int64) (*ports.Thread, error) {
        return mockThread, nil
    },
}
```

## Best Practices

1. **Always use ThreadService** - Never query `storage.GetThreadEmails()` directly from UI
2. **Handle errors gracefully** - Network/DB errors can occur
3. **Use summaries for lists** - Don't load full threads in inbox view
4. **Subscribe to events** - Keep UI in sync with thread state changes
5. **Cache smartly** - Cache thread data in UI state to avoid repeated queries
6. **Order awareness** - Remember messages are DESC by date (newest first)

## Migration Guide

If you have existing code calling storage directly:

### Before (❌ Wrong)
```go
// TUI directly calling storage
var emails, err = storage.GetThreadEmails(threadID, accountID)
for _, email := range emails {
    // Process email...
}
```

### After (✅ Correct)
```go
// TUI using ThreadService
var thread, err = app.Thread().GetThreadByID(ctx, threadID)
if err != nil {
    return err
}
for _, msg := range thread.Messages {
    // Process message...
}
```

## Performance Tips

- Use `GetThreadSummary()` for inbox lists (lighter weight)
- Use `GetThread()` only when displaying full thread view
- Cache thread data in UI state to avoid repeated fetches
- Subscribe to events to invalidate cache when threads change
