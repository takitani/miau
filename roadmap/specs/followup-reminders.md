# EM-10: Follow-up Reminders

## Overview

Remind users to follow up on emails that haven't received a response within a specified time.

## User Stories

1. As a user, I want to be reminded if I don't get a response to an important email
2. As a user, I want to set follow-up reminders when sending emails
3. As a user, I want to see all pending follow-ups in one view
4. As a user, I want the reminder cancelled if a response arrives

## Technical Requirements

### Service Layer

Create `internal/services/followup.go`:

```go
package services

type FollowUpService interface {
    // CreateFollowUp sets a follow-up reminder
    CreateFollowUp(ctx context.Context, emailID int64, remindAt time.Time) (*FollowUp, error)

    // CreateFollowUpPreset creates with preset duration
    CreateFollowUpPreset(ctx context.Context, emailID int64, preset FollowUpPreset) (*FollowUp, error)

    // GetPendingFollowUps returns all pending reminders
    GetPendingFollowUps(ctx context.Context, accountID int64) ([]FollowUp, error)

    // CancelFollowUp cancels a reminder
    CancelFollowUp(ctx context.Context, id int64) error

    // CheckForResponses cancels follow-ups that received responses
    CheckForResponses(ctx context.Context) error

    // ProcessDueFollowUps notifies about due follow-ups
    ProcessDueFollowUps(ctx context.Context) error
}

type FollowUp struct {
    ID            int64
    EmailID       int64
    Email         *Email
    CreatedAt     time.Time
    RemindAt      time.Time
    Status        FollowUpStatus
    ResponseEmail *Email  // If response received
}

type FollowUpStatus string

const (
    FollowUpPending   FollowUpStatus = "pending"
    FollowUpResponded FollowUpStatus = "responded"
    FollowUpCanceled  FollowUpStatus = "canceled"
    FollowUpReminded  FollowUpStatus = "reminded"
)

type FollowUpPreset string

const (
    FollowUp1Day   FollowUpPreset = "1_day"
    FollowUp3Days  FollowUpPreset = "3_days"
    FollowUp1Week  FollowUpPreset = "1_week"
    FollowUp2Weeks FollowUpPreset = "2_weeks"
)
```

### Database Schema

```sql
CREATE TABLE follow_ups (
    id INTEGER PRIMARY KEY,
    email_id INTEGER REFERENCES emails(id),
    account_id INTEGER REFERENCES accounts(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    remind_at DATETIME NOT NULL,
    status TEXT DEFAULT 'pending',
    response_email_id INTEGER REFERENCES emails(id),
    reminded_at DATETIME
);

CREATE INDEX idx_followup_remind_at ON follow_ups(remind_at);
CREATE INDEX idx_followup_status ON follow_ups(status);
```

### Response Detection

```go
func (s *FollowUpService) CheckForResponses(ctx context.Context) error {
    pendingFollowUps, err := s.storage.GetPendingFollowUps(ctx)
    if err != nil {
        return err
    }

    for _, followUp := range pendingFollowUps {
        // Check if any email in inbox is a response to this email
        response, err := s.emailService.FindResponse(ctx, followUp.EmailID)
        if err != nil {
            continue
        }

        if response != nil {
            s.storage.MarkResponded(ctx, followUp.ID, response.ID)
        }
    }
    return nil
}

func (s *EmailService) FindResponse(ctx context.Context, sentEmailID int64) (*Email, error) {
    sentEmail, err := s.GetEmail(ctx, sentEmailID)
    if err != nil {
        return nil, err
    }

    // Find email that:
    // 1. Has In-Reply-To matching sent email's Message-ID
    // 2. OR has References containing sent email's Message-ID
    // 3. OR subject starts with "Re:" and matches
    // 4. AND is from one of the original recipients
    // 5. AND was received after the sent email

    return s.storage.FindResponseEmail(ctx, sentEmail)
}
```

## UI/UX

### TUI
- When sending, prompt for follow-up reminder
- Press `F` on sent email to add follow-up

```
┌─ Email Sent ──────────────────────────────────────────────────────┐
│ ✓ Email sent to john@example.com                                  │
│                                                                   │
│ Set follow-up reminder?                                           │
│ [1] If no reply in 1 day                                          │
│ [2] If no reply in 3 days                                         │
│ [3] If no reply in 1 week                                         │
│ [4] If no reply in 2 weeks                                        │
│ [c] Custom...                                                     │
│ [n] No reminder                                                   │
└───────────────────────────────────────────────────────────────────┘

┌─ Follow-up Reminders (3) ─────────────────────────────────────────┐
│ ⏰ To: john@example.com   │ Project Proposal │ Due in 2 days      │
│ ⏰ To: client@co.com      │ Invoice #123     │ Due tomorrow       │
│ ✓  To: boss@company.com   │ Budget Review    │ Responded!         │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Follow-up checkbox in compose
- Follow-up indicator on sent emails
- Follow-up panel/view
- Notification when reminder triggers

## Testing

1. Test reminder creation
2. Test response detection via Message-ID
3. Test response detection via subject match
4. Test automatic cancellation on response
5. Test notification on due reminder
6. Test manual cancellation

## Acceptance Criteria

- [ ] Can set follow-up when sending
- [ ] Can add follow-up to sent emails
- [ ] Reminder automatically cancelled when response received
- [ ] Notification sent when follow-up is due
- [ ] Can view all pending follow-ups
- [ ] Can manually cancel follow-up
- [ ] Shows "responded" status when applicable
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
follow_up:
  enabled: true
  prompt_on_send: true
  default_duration: "3_days"
  check_responses_interval: "5m"
```

## Estimated Complexity

Medium-High - Response detection logic
