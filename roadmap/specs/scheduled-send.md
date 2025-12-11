# EM-08: Scheduled Send

## Overview

Allow users to compose emails and schedule them to be sent at a later time.

## User Stories

1. As a user, I want to write an email now and send it tomorrow morning
2. As a user, I want to see and edit scheduled emails before they send
3. As a user, I want to cancel a scheduled send
4. As a user, I want notifications when scheduled emails are sent

## Technical Requirements

### Service Layer

Extend `internal/services/send.go`:

```go
package services

type ScheduledSendService interface {
    // ScheduleEmail schedules an email for later sending
    ScheduleEmail(ctx context.Context, draft Draft, sendAt time.Time) (*ScheduledEmail, error)

    // GetScheduledEmails returns all scheduled emails
    GetScheduledEmails(ctx context.Context, accountID int64) ([]ScheduledEmail, error)

    // UpdateScheduledEmail updates a scheduled email
    UpdateScheduledEmail(ctx context.Context, id int64, draft Draft, sendAt *time.Time) error

    // CancelScheduledEmail cancels and converts back to draft
    CancelScheduledEmail(ctx context.Context, id int64) (*Draft, error)

    // SendNow sends a scheduled email immediately
    SendNow(ctx context.Context, id int64) error

    // ProcessDueSchedules sends emails that are due (background job)
    ProcessDueSchedules(ctx context.Context) error
}

type ScheduledEmail struct {
    ID          int64
    AccountID   int64
    DraftID     int64
    Draft       *Draft
    ScheduledAt time.Time
    SendAt      time.Time
    Status      ScheduleStatus
    SentAt      *time.Time
    Error       string
}

type ScheduleStatus string

const (
    SchedulePending   ScheduleStatus = "pending"
    ScheduleSending   ScheduleStatus = "sending"
    ScheduleSent      ScheduleStatus = "sent"
    ScheduleFailed    ScheduleStatus = "failed"
    ScheduleCanceled  ScheduleStatus = "canceled"
)
```

### Database Schema

```sql
CREATE TABLE scheduled_emails (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    draft_id INTEGER REFERENCES drafts(id),
    scheduled_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    send_at DATETIME NOT NULL,
    status TEXT DEFAULT 'pending',
    sent_at DATETIME,
    error_message TEXT
);

CREATE INDEX idx_scheduled_send_at ON scheduled_emails(send_at);
CREATE INDEX idx_scheduled_status ON scheduled_emails(status);
```

### Background Worker

```go
func (s *ScheduledSendService) ProcessDueSchedules(ctx context.Context) error {
    dueEmails, err := s.storage.GetDueScheduledEmails(ctx, time.Now())
    if err != nil {
        return err
    }

    for _, scheduled := range dueEmails {
        // Mark as sending
        s.storage.UpdateStatus(ctx, scheduled.ID, ScheduleSending)

        // Get draft content
        draft, err := s.draftService.GetDraft(ctx, scheduled.DraftID)
        if err != nil {
            s.storage.MarkFailed(ctx, scheduled.ID, err.Error())
            continue
        }

        // Send the email
        err = s.sendService.SendEmail(ctx, draft.ToEmail())
        if err != nil {
            s.storage.MarkFailed(ctx, scheduled.ID, err.Error())
            // Notify user of failure
            s.notifications.Notify(ctx, "Scheduled email failed", draft.Subject)
            continue
        }

        // Mark as sent
        s.storage.MarkSent(ctx, scheduled.ID)

        // Clean up draft
        s.draftService.DeleteDraft(ctx, scheduled.DraftID)

        // Notify user
        s.notifications.Notify(ctx, "Scheduled email sent", draft.Subject)
    }
    return nil
}
```

## UI/UX

### TUI
- In compose, press `S` (shift+s) for schedule options
- Show scheduled emails in drafts panel

```
┌─ Schedule Send ───────────────────────────────────────────────────┐
│ "Project Proposal" to john@example.com                            │
│                                                                   │
│ Send at:                                                          │
│ [1] Tomorrow morning (9:00 AM)                                    │
│ [2] Tomorrow afternoon (2:00 PM)                                  │
│ [3] Monday morning (9:00 AM)                                      │
│ [c] Custom date/time...                                           │
│                                                                   │
│ ┌─ Custom ───────────────────────────────────────────────────┐    │
│ │ Date: [2024-12-16]  Time: [09:30]                          │    │
│ │                                                            │    │
│ │ Scheduled for: Monday, Dec 16 at 9:30 AM                   │    │
│ └────────────────────────────────────────────────────────────┘    │
│                                                                   │
│ [Enter] Schedule  [Esc] Cancel                                    │
└───────────────────────────────────────────────────────────────────┘

┌─ Scheduled (2) ───────────────────────────────────────────────────┐
│ ⏰ To: john@example.com   │ Project Proposal │ Mon 9:30 AM       │
│ ⏰ To: team@company.com   │ Weekly Report    │ Fri 8:00 AM       │
└───────────────────────────────────────────────────────────────────┘
  Enter: edit  d: delete/cancel  s: send now
```

### Desktop
- "Schedule Send" button next to "Send"
- Calendar/time picker modal
- Scheduled tab in sidebar
- Edit scheduled emails

## Testing

1. Test scheduling with presets
2. Test custom date/time scheduling
3. Test background worker sends at correct time
4. Test cancellation before send
5. Test edit before send
6. Test send now (immediate)
7. Test failure handling
8. Test timezone handling

## Acceptance Criteria

- [ ] Can schedule email with presets
- [ ] Can schedule with custom date/time
- [ ] Scheduled emails shown in dedicated view
- [ ] Can edit scheduled email before send time
- [ ] Can cancel scheduled email
- [ ] Can send immediately
- [ ] Background job sends at scheduled time
- [ ] Notification when sent or failed
- [ ] Handles timezone correctly
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
send:
  schedule:
    enabled: true
    check_interval: "1m"
    default_morning: "09:00"
    default_afternoon: "14:00"
    notify_on_send: true
    notify_on_fail: true
```

## Estimated Complexity

Medium - Similar to snooze with send integration
