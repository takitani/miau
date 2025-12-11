# TH-10: Lazy Body Loading

## Overview

Load email bodies on-demand instead of fetching all content at sync.

## Technical Requirements

```go
// Only fetch envelope during sync
type EmailEnvelope struct {
    ID, UID, MessageID string
    Subject, From, To string
    Date time.Time
    HasAttachments bool
    Size int
}

// Fetch body on view
func (s *EmailService) GetEmailBody(ctx context.Context, emailID int64) (*EmailBody, error) {
    // Check if already cached
    body, err := s.storage.GetEmailBody(ctx, emailID)
    if err == nil && body != "" {
        return body, nil
    }

    // Fetch from IMAP
    body, err = s.imap.FetchBody(ctx, emailID)
    if err != nil {
        return nil, err
    }

    // Cache locally
    s.storage.CacheEmailBody(ctx, emailID, body)
    return body, nil
}
```

## Acceptance Criteria

- [ ] Sync only fetches metadata
- [ ] Body fetched on email open
- [ ] Bodies cached after fetch
- [ ] Faster initial sync

## Estimated Complexity

Medium
