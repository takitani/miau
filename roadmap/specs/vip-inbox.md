# EM-12: VIP Inbox

## Overview

Create a separate inbox view showing only emails from important/VIP contacts.

## User Stories

1. As a user, I want to mark contacts as VIP
2. As a user, I want a dedicated VIP inbox view
3. As a user, I want notifications only for VIP emails
4. As a user, I want VIP emails visually distinguished

## Technical Requirements

### Service Layer

Extend existing services:

```go
package services

type VIPService interface {
    // AddVIP marks a contact as VIP
    AddVIP(ctx context.Context, accountID int64, email string, name string) error

    // RemoveVIP removes VIP status
    RemoveVIP(ctx context.Context, accountID int64, email string) error

    // GetVIPs returns all VIP contacts
    GetVIPs(ctx context.Context, accountID int64) ([]VIPContact, error)

    // IsVIP checks if email is from VIP
    IsVIP(ctx context.Context, accountID int64, email string) bool

    // GetVIPEmails returns emails from VIPs
    GetVIPEmails(ctx context.Context, accountID int64, opts EmailListOpts) ([]Email, error)

    // GetVIPUnreadCount returns unread count for VIP inbox
    GetVIPUnreadCount(ctx context.Context, accountID int64) (int, error)
}

type VIPContact struct {
    ID          int64
    AccountID   int64
    Email       string
    Name        string
    ContactID   *int64  // Link to contacts table
    AddedAt     time.Time
    EmailCount  int
    LastEmail   *time.Time
}
```

### Database Schema

```sql
CREATE TABLE vip_contacts (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    email TEXT NOT NULL,
    name TEXT,
    contact_id INTEGER REFERENCES contacts(id),
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, email)
);

-- Add is_vip column to emails for fast filtering
ALTER TABLE emails ADD COLUMN is_vip BOOLEAN DEFAULT 0;

CREATE INDEX idx_emails_vip ON emails(is_vip);
```

### VIP Detection on Sync

```go
func (s *SyncService) onNewEmail(ctx context.Context, email *Email) {
    // Check if sender is VIP
    isVIP := s.vipService.IsVIP(ctx, email.AccountID, email.FromEmail)
    if isVIP {
        email.IsVIP = true
        s.storage.UpdateEmailVIP(ctx, email.ID, true)

        // Send notification for VIP email
        s.notifications.Notify(ctx,
            fmt.Sprintf("VIP: %s", email.FromName),
            email.Subject,
        )
    }
}
```

## UI/UX

### TUI
- Press `V` to toggle VIP inbox view
- Press `v` on email to toggle sender as VIP
- VIP indicator (★) in email list

```
┌─ INBOX ───────────────────────────────────────────────────────────┐
│ ★ Boss           │ URGENT: Board meeting     │ VIP    10:30 AM   │
│   Newsletter     │ Weekly digest             │        10:15 AM   │
│ ★ Client ABC     │ Contract review           │ VIP    09:45 AM   │
│   Promo          │ 50% off sale              │        08:00 AM   │
└───────────────────────────────────────────────────────────────────┘

┌─ VIP INBOX (2 unread) ────────────────────────────────────────────┐
│ ★ Boss           │ URGENT: Board meeting     │ unread 10:30 AM   │
│ ★ Client ABC     │ Contract review           │ unread 09:45 AM   │
│ ★ Boss           │ RE: Budget proposal       │        Yesterday  │
│ ★ Client ABC     │ Thanks for the call       │        Dec 10     │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- VIP folder in sidebar with unread count
- Star/VIP button on emails and contacts
- VIP management in settings
- Visual distinction (gold star, different color)

### Folders Panel

```
┌─ Folders ────────┐
│ 📥 Inbox    (12) │
│ ★  VIP      (2)  │  <- VIP "folder"
│ 📤 Sent          │
│ 📝 Drafts   (1)  │
│ 🗑  Trash         │
└──────────────────┘
```

## Testing

1. Test adding/removing VIP
2. Test VIP detection on new emails
3. Test VIP inbox filtering
4. Test notifications for VIP only
5. Test sync with contact system

## Acceptance Criteria

- [ ] Can mark contacts as VIP
- [ ] VIP inbox shows only VIP emails
- [ ] VIP indicator visible in main inbox
- [ ] Unread count for VIP inbox
- [ ] Optional: Notifications only for VIP
- [ ] Can manage VIPs from settings
- [ ] Links to contact system
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
vip:
  enabled: true
  notify_only_vip: false  # Only notify for VIP emails
  show_vip_badge: true
  vip_color: "gold"
```

## Estimated Complexity

Low-Medium - Filtering plus UI changes
