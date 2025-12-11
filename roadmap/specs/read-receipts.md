# EM-16: Read Receipts (Opt-in)

## Overview

Track when recipients open emails with opt-in read receipts for important messages.

## User Stories

1. As a user, I want to know when my important email was read
2. As a user, I want to request read receipts when sending
3. As a user, I want to see read status on sent emails
4. As a user, I want to control whether I send receipts to others

## Technical Requirements

### Service Layer

Create `internal/services/readreceipt.go`:

```go
package services

type ReadReceiptService interface {
    // RequestReceipt marks email to request read receipt
    RequestReceipt(ctx context.Context, emailID int64) error

    // ProcessReceipt handles incoming MDN (message disposition notification)
    ProcessReceipt(ctx context.Context, mdn *MDN) error

    // GetReceiptStatus returns read status for sent email
    GetReceiptStatus(ctx context.Context, emailID int64) (*ReceiptStatus, error)

    // GetTrackedEmails returns emails with receipt tracking
    GetTrackedEmails(ctx context.Context, accountID int64) ([]TrackedEmail, error)

    // SetReceiptPolicy sets how to handle receipt requests
    SetReceiptPolicy(ctx context.Context, accountID int64, policy ReceiptPolicy) error
}

type ReceiptStatus struct {
    EmailID      int64
    Requested    bool
    Recipients   []RecipientStatus
}

type RecipientStatus struct {
    Email        string
    Status       ReadStatus
    ReadAt       *time.Time
    UserAgent    string
}

type ReadStatus string

const (
    StatusPending   ReadStatus = "pending"
    StatusDelivered ReadStatus = "delivered"
    StatusRead      ReadStatus = "read"
    StatusFailed    ReadStatus = "failed"
)

type TrackedEmail struct {
    EmailID      int64
    Subject      string
    SentAt       time.Time
    Recipients   []RecipientStatus
    AllRead      bool
}

type ReceiptPolicy string

const (
    PolicyNever  ReceiptPolicy = "never"   // Never send receipts
    PolicyAsk    ReceiptPolicy = "ask"     // Ask each time
    PolicyAlways ReceiptPolicy = "always"  // Always send
)

type MDN struct {
    OriginalMessageID string
    Disposition       string  // "displayed", "deleted", etc.
    RecipientEmail    string
    ReceivedAt        time.Time
}
```

### Database Schema

```sql
CREATE TABLE read_receipts (
    id INTEGER PRIMARY KEY,
    email_id INTEGER REFERENCES emails(id),
    recipient_email TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    read_at DATETIME,
    user_agent TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(email_id, recipient_email)
);

CREATE TABLE receipt_settings (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id) UNIQUE,
    send_policy TEXT DEFAULT 'ask',
    request_default BOOLEAN DEFAULT 0
);
```

### Email Headers

```go
// When sending with receipt request
func (s *SendService) addReceiptHeaders(email *Email) {
    // RFC 8098 - Message Disposition Notification
    email.AddHeader("Disposition-Notification-To", s.config.Email)

    // RFC 3798 - Original-Recipient
    email.AddHeader("Return-Receipt-To", s.config.Email)
}

// When processing incoming email
func (s *SyncService) checkReceiptRequest(ctx context.Context, email *Email) {
    dnt := email.GetHeader("Disposition-Notification-To")
    if dnt == "" {
        return
    }

    // Check user's policy
    policy, _ := s.receiptService.GetPolicy(ctx, email.AccountID)

    switch policy {
    case PolicyAlways:
        s.sendReceipt(ctx, email, dnt)
    case PolicyAsk:
        // Queue for user decision
        s.queueReceiptRequest(ctx, email, dnt)
    case PolicyNever:
        // Do nothing
    }
}

// Sending MDN
func (s *ReadReceiptService) sendReceipt(ctx context.Context, originalEmail *Email, to string) error {
    mdn := &Email{
        To:      to,
        Subject: fmt.Sprintf("Read: %s", originalEmail.Subject),
        Headers: map[string]string{
            "Content-Type": "message/disposition-notification",
        },
        Body: fmt.Sprintf(mdnTemplate,
            originalEmail.MessageID,
            "displayed",
            time.Now().Format(time.RFC1123Z),
        ),
    }

    return s.sendService.SendEmail(ctx, mdn)
}
```

## UI/UX

### TUI
- Checkbox in compose for receipt request
- Read status indicator on sent emails

```
┌─ Compose ─────────────────────────────────────────────────────────┐
│ To: important-client@company.com                                  │
│ Subject: Contract Proposal                                        │
├───────────────────────────────────────────────────────────────────┤
│ ...email content...                                               │
├───────────────────────────────────────────────────────────────────┤
│ [✓] Request read receipt                                          │
│ [s] Send  [d] Save draft  [Esc] Cancel                           │
└───────────────────────────────────────────────────────────────────┘

┌─ Sent ────────────────────────────────────────────────────────────┐
│ ✓  Contract Proposal │ client@co.com │ Read: Dec 15, 2:30 PM     │
│ ⏳ Project Update    │ team@co.com   │ Pending (3 of 5 read)     │
│    Weekly Report     │ boss@co.com   │ Sent: Dec 14              │
└───────────────────────────────────────────────────────────────────┘
```

### Receipt Request Prompt

```
┌─ Read Receipt Request ────────────────────────────────────────────┐
│                                                                   │
│ john@example.com is requesting a read receipt for:                │
│ "RE: Project Proposal"                                            │
│                                                                   │
│ Send receipt when you open this email?                            │
│                                                                   │
│ [y] Yes  [n] No  [a] Always for this sender  [N] Never for this  │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Read receipt checkbox in compose
- Track status in sent folder
- Notification when receipt received
- Settings for receipt policy

## Testing

1. Test receipt request header addition
2. Test MDN parsing
3. Test policy handling
4. Test receipt status tracking
5. Test multiple recipients

## Acceptance Criteria

- [ ] Can request read receipt when sending
- [ ] Receipt status shown on sent emails
- [ ] Handles incoming receipt requests per policy
- [ ] Can configure default policy
- [ ] Tracks multiple recipients separately
- [ ] Notification when receipt received
- [ ] Privacy-respecting (opt-in)
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
read_receipts:
  enabled: true
  request_default: false
  send_policy: "ask"  # never, ask, always
```

## Privacy Considerations

- Read receipts are opt-in only
- Users can choose not to send receipts
- No tracking pixels (uses standard MDN)
- Respects user's policy preferences

## Estimated Complexity

Medium - Email headers plus MDN handling
