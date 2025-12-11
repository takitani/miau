# EM-07: Email Snooze

## Overview

Allow users to temporarily hide emails and have them reappear at a specified time.

## User Stories

1. As a user, I want to snooze an email until later today
2. As a user, I want to snooze an email until a specific date/time
3. As a user, I want snoozed emails to reappear as unread at the top
4. As a user, I want to see and manage all snoozed emails

## Technical Requirements

### Service Layer

Create `internal/services/snooze.go`:

```go
package services

type SnoozeService interface {
    // SnoozeEmail snoozes an email until specified time
    SnoozeEmail(ctx context.Context, emailID int64, until time.Time) error

    // SnoozeEmailPreset snoozes using a preset duration
    SnoozeEmailPreset(ctx context.Context, emailID int64, preset SnoozePreset) error

    // UnsnoozeEmail removes snooze before it triggers
    UnsnoozeEmail(ctx context.Context, emailID int64) error

    // GetSnoozedEmails returns all currently snoozed emails
    GetSnoozedEmails(ctx context.Context, accountID int64) ([]SnoozedEmail, error)

    // ProcessDueSnoozes processes snoozes that are due (background job)
    ProcessDueSnoozes(ctx context.Context) error
}

type SnoozePreset string

const (
    SnoozeLaterToday    SnoozePreset = "later_today"    // +4 hours
    SnoozeTomorrow      SnoozePreset = "tomorrow"       // Tomorrow 9 AM
    SnoozeNextWeek      SnoozePreset = "next_week"      // Next Monday 9 AM
    SnoozeThisWeekend   SnoozePreset = "this_weekend"   // Saturday 9 AM
    SnoozeNextMonth     SnoozePreset = "next_month"     // 1st of next month
)

type SnoozedEmail struct {
    ID        int64
    EmailID   int64
    Email     *Email
    SnoozedAt time.Time
    SnoozeUntil time.Time
    Preset    SnoozePreset
}
```

### Database Schema

```sql
CREATE TABLE snoozed_emails (
    id INTEGER PRIMARY KEY,
    email_id INTEGER REFERENCES emails(id) UNIQUE,
    account_id INTEGER REFERENCES accounts(id),
    snoozed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    snooze_until DATETIME NOT NULL,
    preset TEXT,
    processed BOOLEAN DEFAULT 0
);

CREATE INDEX idx_snooze_until ON snoozed_emails(snooze_until);
CREATE INDEX idx_snooze_processed ON snoozed_emails(processed);
```

### Background Worker

```go
// In main.go or app startup
func startSnoozeWorker(ctx context.Context, snoozeService SnoozeService) {
    ticker := time.NewTicker(1 * time.Minute)
    go func() {
        for {
            select {
            case <-ticker.C:
                snoozeService.ProcessDueSnoozes(ctx)
            case <-ctx.Done():
                return
            }
        }
    }()
}

func (s *SnoozeService) ProcessDueSnoozes(ctx context.Context) error {
    dueSnoozes, err := s.storage.GetDueSnoozes(ctx, time.Now())
    if err != nil {
        return err
    }

    for _, snooze := range dueSnoozes {
        // 1. Mark email as unread
        s.emailService.MarkUnread(ctx, snooze.EmailID)

        // 2. Update email timestamp to appear at top
        s.storage.UpdateEmailDate(ctx, snooze.EmailID, time.Now())

        // 3. Mark snooze as processed
        s.storage.MarkSnoozeProcessed(ctx, snooze.ID)

        // 4. Send notification
        s.notifications.Notify(ctx, "Snoozed email reminder", snooze.Email.Subject)
    }
    return nil
}
```

## UI/UX

### TUI
- Press `z` to snooze an email
- Quick presets or custom time

```
┌─ Snooze Email ────────────────────────────────────────────────────┐
│ "RE: Project Update" from john@example.com                        │
│                                                                   │
│ Remind me:                                                        │
│ [1] Later today (4:00 PM)                                         │
│ [2] Tomorrow morning (9:00 AM)                                    │
│ [3] This weekend (Sat 9:00 AM)                                    │
│ [4] Next week (Mon 9:00 AM)                                       │
│ [5] Next month (Jan 1 9:00 AM)                                    │
│ [c] Custom date/time...                                           │
└───────────────────────────────────────────────────────────────────┘

┌─ Snoozed (3) ─────────────────────────────────────────────────────┐
│ 💤 John Smith   │ RE: Project Update  │ Due: Today 4:00 PM        │
│ 💤 Newsletter   │ Weekly Digest       │ Due: Tomorrow 9:00 AM     │
│ 💤 HR Team      │ Benefits Update     │ Due: Mon 9:00 AM          │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Snooze button/icon
- Calendar picker for custom time
- Snoozed folder in sidebar
- Notification when snooze triggers

## Testing

1. Test preset time calculations
2. Test custom time snooze
3. Test background worker processing
4. Test notification on unsnooze
5. Test unsnooze before due
6. Test timezone handling

## Acceptance Criteria

- [ ] Can snooze with presets (later, tomorrow, etc.)
- [ ] Can snooze with custom date/time
- [ ] Snoozed emails hidden from inbox
- [ ] Email reappears as unread at scheduled time
- [ ] Can view all snoozed emails
- [ ] Can cancel snooze early
- [ ] Notification sent when email unsnoozes
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
snooze:
  default_time_morning: "09:00"
  default_time_afternoon: "14:00"
  notifications: true
```

## Estimated Complexity

Medium - Background worker plus time handling
