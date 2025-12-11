# EM-15: Email Digest (Newsletter Summary)

## Overview

Consolidate newsletters and low-priority emails into a periodic digest to reduce inbox clutter.

## User Stories

1. As a user, I want newsletters grouped into a daily digest
2. As a user, I want AI summaries of each newsletter in the digest
3. As a user, I want to configure which senders go into digest
4. As a user, I want to read the full email from the digest

## Technical Requirements

### Service Layer

Create `internal/services/digest.go`:

```go
package services

type DigestService interface {
    // GenerateDigest creates a digest for the specified period
    GenerateDigest(ctx context.Context, accountID int64, period DigestPeriod) (*Digest, error)

    // GetLatestDigest returns the most recent digest
    GetLatestDigest(ctx context.Context, accountID int64) (*Digest, error)

    // GetDigests returns digest history
    GetDigests(ctx context.Context, accountID int64, limit int) ([]Digest, error)

    // ConfigureDigest sets up digest preferences
    ConfigureDigest(ctx context.Context, accountID int64, config DigestConfig) error

    // AddToDigest marks a sender for digest inclusion
    AddToDigest(ctx context.Context, accountID int64, senderEmail string) error

    // RemoveFromDigest removes a sender from digest
    RemoveFromDigest(ctx context.Context, accountID int64, senderEmail string) error
}

type Digest struct {
    ID          int64
    AccountID   int64
    Period      DigestPeriod
    StartTime   time.Time
    EndTime     time.Time
    Items       []DigestItem
    TotalEmails int
    ReadCount   int
    GeneratedAt time.Time
}

type DigestItem struct {
    EmailID     int64
    SenderName  string
    SenderEmail string
    Subject     string
    Summary     string  // AI-generated summary
    Date        time.Time
    IsRead      bool
}

type DigestPeriod string

const (
    DigestDaily   DigestPeriod = "daily"
    DigestWeekly  DigestPeriod = "weekly"
)

type DigestConfig struct {
    Enabled       bool
    Period        DigestPeriod
    SendTime      string  // "08:00"
    IncludeCategories []string
    ExcludeSenders    []string
    GenerateSummaries bool
}
```

### Database Schema

```sql
CREATE TABLE digest_config (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id) UNIQUE,
    enabled BOOLEAN DEFAULT 1,
    period TEXT DEFAULT 'daily',
    send_time TEXT DEFAULT '08:00',
    include_categories TEXT,  -- JSON array
    exclude_senders TEXT,  -- JSON array
    generate_summaries BOOLEAN DEFAULT 1
);

CREATE TABLE digest_senders (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    sender_email TEXT NOT NULL,
    sender_name TEXT,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, sender_email)
);

CREATE TABLE digests (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    period TEXT,
    start_time DATETIME,
    end_time DATETIME,
    items TEXT,  -- JSON array of DigestItem
    total_emails INTEGER,
    generated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Digest Generation

```go
func (s *DigestService) GenerateDigest(ctx context.Context, accountID int64, period DigestPeriod) (*Digest, error) {
    config, err := s.getConfig(ctx, accountID)
    if err != nil {
        return nil, err
    }

    // Calculate time range
    var startTime time.Time
    endTime := time.Now()
    switch period {
    case DigestDaily:
        startTime = endTime.AddDate(0, 0, -1)
    case DigestWeekly:
        startTime = endTime.AddDate(0, 0, -7)
    }

    // Get digest senders
    digestSenders, err := s.getDigestSenders(ctx, accountID)
    if err != nil {
        return nil, err
    }

    // Fetch emails from digest senders in time range
    emails, err := s.emailService.GetEmailsBySenders(ctx, accountID, digestSenders, startTime, endTime)
    if err != nil {
        return nil, err
    }

    // Generate summaries if enabled
    items := make([]DigestItem, len(emails))
    for i, email := range emails {
        item := DigestItem{
            EmailID:     email.ID,
            SenderName:  email.FromName,
            SenderEmail: email.FromEmail,
            Subject:     email.Subject,
            Date:        email.Date,
            IsRead:      email.IsRead,
        }

        if config.GenerateSummaries {
            summary, _ := s.aiService.SummarizeEmail(ctx, email.ID, SummaryTLDR)
            item.Summary = summary.Content
        }

        items[i] = item
    }

    // Mark emails as part of digest (hide from main inbox)
    for _, email := range emails {
        s.emailService.MarkInDigest(ctx, email.ID)
    }

    digest := &Digest{
        AccountID:   accountID,
        Period:      period,
        StartTime:   startTime,
        EndTime:     endTime,
        Items:       items,
        TotalEmails: len(items),
        GeneratedAt: time.Now(),
    }

    // Save digest
    err = s.storage.SaveDigest(ctx, digest)
    return digest, err
}
```

## UI/UX

### TUI
- Press `D` for digest view
- Digest shown as special folder

```
┌─ Today's Digest (12 emails) ──────────────────────────────────────┐
│ Generated: Today at 8:00 AM                                       │
│                                                                   │
│ TechCrunch Daily                                         09:15 AM │
│   → AI startup raises $50M, Google announces new features...      │
│                                                                   │
│ Medium Digest                                            08:30 AM │
│   → 5 articles about productivity, software architecture...       │
│                                                                   │
│ GitHub Weekly                                            Yesterday │
│   → New releases: React 19, TypeScript 5.4, trending repos...     │
│                                                                   │
│ LinkedIn Updates                                         Yesterday │
│   → 3 connection requests, 5 job suggestions, 2 mentions...       │
│                                                                   │
│ Unread: 8  Total: 12                                              │
├───────────────────────────────────────────────────────────────────┤
│ [Enter] Read full email  [a] Archive all  [m] Mark all read       │
└───────────────────────────────────────────────────────────────────┘
```

### Digest Settings

```
┌─ Digest Settings ─────────────────────────────────────────────────┐
│                                                                   │
│ [✓] Enable email digest                                           │
│                                                                   │
│ Frequency: [Daily ▼]                                              │
│ Generate at: [08:00]                                              │
│                                                                   │
│ Include in digest:                                                │
│ [✓] TechCrunch (45 emails/month)                                  │
│ [✓] Medium Digest (30 emails/month)                               │
│ [✓] GitHub (15 emails/month)                                      │
│ [✓] LinkedIn (22 emails/month)                                    │
│ [ ] Amazon (promotions)                                           │
│                                                                   │
│ [✓] Generate AI summaries for each email                          │
│                                                                   │
│ [a] Auto-detect newsletters  [+] Add sender manually              │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Digest tab/folder in sidebar
- Card-based digest view
- One-click subscribe to digest
- Digest history

## Testing

1. Test digest generation
2. Test time range filtering
3. Test sender filtering
4. Test AI summary generation
5. Test mark as digest
6. Test periodic generation (cron)

## Acceptance Criteria

- [ ] Generates daily/weekly digest
- [ ] Groups emails by sender
- [ ] AI summaries included (optional)
- [ ] Can configure which senders
- [ ] Digest emails hidden from main inbox
- [ ] Can read full email from digest
- [ ] Digest history available
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
digest:
  enabled: true
  period: "daily"
  generate_at: "08:00"
  generate_summaries: true
  auto_detect_newsletters: true
```

## Estimated Complexity

Medium-High - Scheduling, AI integration, special folder handling
