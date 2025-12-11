# EM-11: Unsubscribe Manager

## Overview

Help users manage newsletter subscriptions and easily unsubscribe from unwanted emails.

## User Stories

1. As a user, I want to see all my newsletter subscriptions
2. As a user, I want to easily unsubscribe with one click
3. As a user, I want to identify emails I can unsubscribe from
4. As a user, I want bulk unsubscribe options

## Technical Requirements

### Service Layer

Create `internal/services/unsubscribe.go`:

```go
package services

type UnsubscribeService interface {
    // GetSubscriptions returns detected subscriptions
    GetSubscriptions(ctx context.Context, accountID int64) ([]Subscription, error)

    // DetectUnsubscribe checks if email has unsubscribe option
    DetectUnsubscribe(ctx context.Context, emailID int64) (*UnsubscribeInfo, error)

    // Unsubscribe attempts to unsubscribe from a sender
    Unsubscribe(ctx context.Context, subscriptionID int64) error

    // BulkUnsubscribe unsubscribes from multiple senders
    BulkUnsubscribe(ctx context.Context, ids []int64) error

    // ScanForSubscriptions scans inbox for newsletters
    ScanForSubscriptions(ctx context.Context, accountID int64) error
}

type Subscription struct {
    ID              int64
    AccountID       int64
    SenderEmail     string
    SenderName      string
    UnsubscribeURL  string
    UnsubscribeEmail string
    EmailCount      int
    LastReceived    time.Time
    Status          SubscriptionStatus
}

type SubscriptionStatus string

const (
    SubscriptionActive      SubscriptionStatus = "active"
    SubscriptionUnsubscribed SubscriptionStatus = "unsubscribed"
    SubscriptionPending     SubscriptionStatus = "pending"
    SubscriptionFailed      SubscriptionStatus = "failed"
)

type UnsubscribeInfo struct {
    HasUnsubscribe   bool
    UnsubscribeURL   string
    UnsubscribeEmail string
    ListUnsubscribe  string  // RFC 2369 header
    OneClickSupport  bool    // RFC 8058
}
```

### Database Schema

```sql
CREATE TABLE subscriptions (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    sender_email TEXT NOT NULL,
    sender_name TEXT,
    unsubscribe_url TEXT,
    unsubscribe_email TEXT,
    email_count INTEGER DEFAULT 1,
    last_received DATETIME,
    status TEXT DEFAULT 'active',
    unsubscribed_at DATETIME,
    UNIQUE(account_id, sender_email)
);
```

### Unsubscribe Detection

```go
func (s *UnsubscribeService) DetectUnsubscribe(ctx context.Context, emailID int64) (*UnsubscribeInfo, error) {
    email, err := s.emailService.GetEmailWithHeaders(ctx, emailID)
    if err != nil {
        return nil, err
    }

    info := &UnsubscribeInfo{}

    // 1. Check List-Unsubscribe header (RFC 2369)
    if listUnsub := email.GetHeader("List-Unsubscribe"); listUnsub != "" {
        info.ListUnsubscribe = listUnsub
        info.HasUnsubscribe = true

        // Parse URL or mailto
        if url := extractURL(listUnsub); url != "" {
            info.UnsubscribeURL = url
        }
        if mailto := extractMailto(listUnsub); mailto != "" {
            info.UnsubscribeEmail = mailto
        }
    }

    // 2. Check List-Unsubscribe-Post header (RFC 8058 one-click)
    if post := email.GetHeader("List-Unsubscribe-Post"); post == "List-Unsubscribe=One-Click" {
        info.OneClickSupport = true
    }

    // 3. Scan body for unsubscribe links
    if !info.HasUnsubscribe {
        if url := s.findUnsubscribeInBody(email.BodyHTML); url != "" {
            info.UnsubscribeURL = url
            info.HasUnsubscribe = true
        }
    }

    return info, nil
}

func (s *UnsubscribeService) Unsubscribe(ctx context.Context, subscriptionID int64) error {
    sub, err := s.storage.GetSubscription(ctx, subscriptionID)
    if err != nil {
        return err
    }

    // Method 1: One-click unsubscribe (preferred)
    if sub.UnsubscribeURL != "" {
        // POST to URL with List-Unsubscribe=One-Click
        err = s.oneClickUnsubscribe(ctx, sub.UnsubscribeURL)
        if err == nil {
            return s.storage.MarkUnsubscribed(ctx, subscriptionID)
        }
    }

    // Method 2: Open URL in browser
    if sub.UnsubscribeURL != "" {
        s.openBrowser(sub.UnsubscribeURL)
        return s.storage.MarkPending(ctx, subscriptionID)
    }

    // Method 3: Send email to unsubscribe address
    if sub.UnsubscribeEmail != "" {
        err = s.sendUnsubscribeEmail(ctx, sub.UnsubscribeEmail)
        if err == nil {
            return s.storage.MarkPending(ctx, subscriptionID)
        }
    }

    return fmt.Errorf("no unsubscribe method available")
}
```

## UI/UX

### TUI
- Press `u` on email to show unsubscribe option
- Press `U` for unsubscribe manager

```
┌─ Unsubscribe Manager ─────────────────────────────────────────────┐
│ Found 12 newsletters/subscriptions                                │
│                                                                   │
│ [ ] TechCrunch Daily        │ 45 emails │ Last: Today           │
│ [ ] Medium Digest           │ 30 emails │ Last: Yesterday       │
│ [ ] GitHub Notifications    │ 128 emails │ Last: Today          │
│ [✓] Random Promo Site       │ 15 emails │ Last: 3 days ago     │
│ [ ] LinkedIn Updates        │ 22 emails │ Last: 2 days ago     │
│ [✓] Spam Newsletter         │ 8 emails  │ Last: 1 week ago     │
│                                                                   │
│ Selected: 2                                                       │
├───────────────────────────────────────────────────────────────────┤
│ [Space] Select  [a] Select all  [u] Unsubscribe selected         │
│ [Enter] Open in browser  [Esc] Close                             │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Unsubscribe button on newsletter emails
- Subscription manager view
- Bulk actions
- Status tracking

## Testing

1. Test List-Unsubscribe header parsing
2. Test one-click unsubscribe
3. Test URL extraction from body
4. Test mailto unsubscribe
5. Test bulk unsubscribe
6. Test subscription detection

## Acceptance Criteria

- [ ] Detects emails with unsubscribe options
- [ ] Parses List-Unsubscribe header correctly
- [ ] One-click unsubscribe works when supported
- [ ] Opens browser for URL-based unsubscribe
- [ ] Sends email for mailto unsubscribe
- [ ] Shows all detected subscriptions
- [ ] Bulk unsubscribe works
- [ ] Tracks unsubscribe status
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
unsubscribe:
  enabled: true
  scan_on_sync: true
  one_click_preferred: true
```

## Estimated Complexity

Medium - Header parsing and HTTP requests
