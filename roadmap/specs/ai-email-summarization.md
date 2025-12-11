# AI-05: AI Email Summarization

## Overview

Implement automatic email summarization using AI to help users quickly understand email content without reading the full body.

## User Stories

1. As a user, I want to see a brief AI-generated summary of long emails
2. As a user, I want to summarize entire email threads into key points
3. As a user, I want to configure summary length (tldr, brief, detailed)
4. As a user, I want summaries to be cached so they load instantly on repeat views

## Technical Requirements

### Service Layer

Create `internal/services/summarization.go`:

```go
package services

type SummarizationService interface {
    // SummarizeEmail generates a summary for a single email
    SummarizeEmail(ctx context.Context, emailID int64, style SummaryStyle) (*Summary, error)

    // SummarizeThread generates a summary for an entire thread
    SummarizeThread(ctx context.Context, threadID string) (*ThreadSummary, error)

    // GetCachedSummary retrieves cached summary if exists
    GetCachedSummary(ctx context.Context, emailID int64) (*Summary, error)

    // InvalidateSummary removes cached summary
    InvalidateSummary(ctx context.Context, emailID int64) error
}

type SummaryStyle string

const (
    SummaryTLDR     SummaryStyle = "tldr"     // 1-2 sentences
    SummaryBrief    SummaryStyle = "brief"    // 3-5 sentences
    SummaryDetailed SummaryStyle = "detailed" // Full summary with sections
)

type Summary struct {
    EmailID   int64
    Style     SummaryStyle
    Content   string
    KeyPoints []string
    CreatedAt time.Time
}

type ThreadSummary struct {
    ThreadID     string
    Participants []string
    Timeline     string
    KeyDecisions []string
    ActionItems  []string
    CreatedAt    time.Time
}
```

### Database Schema

```sql
CREATE TABLE email_summaries (
    id INTEGER PRIMARY KEY,
    email_id INTEGER REFERENCES emails(id) UNIQUE,
    style TEXT NOT NULL,
    content TEXT NOT NULL,
    key_points TEXT,  -- JSON array
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE thread_summaries (
    id INTEGER PRIMARY KEY,
    thread_id TEXT UNIQUE NOT NULL,
    participants TEXT,  -- JSON array
    timeline TEXT,
    key_decisions TEXT,  -- JSON array
    action_items TEXT,  -- JSON array
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### AI Prompt Templates

```go
var emailSummaryPrompt = `Summarize this email in {{.Style}} style:

From: {{.From}}
To: {{.To}}
Subject: {{.Subject}}
Date: {{.Date}}

{{.Body}}

Output format:
- Summary: [your summary]
- Key points: [bullet list if brief/detailed]`

var threadSummaryPrompt = `Summarize this email thread:

{{range .Emails}}
---
From: {{.From}}
Date: {{.Date}}
{{.Body}}
{{end}}

Output:
- Timeline: brief chronological summary
- Key decisions: what was decided
- Action items: tasks mentioned
- Participants: list of people involved`
```

## UI/UX

### TUI
- Press `s` on an email to show/toggle summary panel
- Summary appears above email body
- Loading indicator while AI generates

```
┌─ Summary ─────────────────────────────────────────────────────────┐
│ TL;DR: John proposes Q1 budget increase of 15% for marketing.     │
│ Key points:                                                       │
│ • Current spend: $50k/month                                       │
│ • Proposed: $57.5k/month                                          │
│ • ROI projection: 2.3x                                            │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Summary section in email viewer (collapsible)
- "Summarize Thread" button in thread view
- Summary style selector in settings

## Testing

1. Unit tests for SummarizationService
2. Mock AI responses for deterministic testing
3. Test summary caching behavior
4. Test thread with 1, 5, 10+ emails
5. Test with HTML-heavy emails (strip HTML first)

## Acceptance Criteria

- [ ] Single email summarization works with all 3 styles
- [ ] Thread summarization correctly identifies participants
- [ ] Summaries are cached and retrieved on repeat views
- [ ] Cache invalidation works when email is updated
- [ ] Works in both TUI and Desktop
- [ ] Handles emails in multiple languages
- [ ] Graceful error handling when AI unavailable

## Dependencies

- Existing AI service (`internal/services/ai.go`)
- Email service for fetching content
- Storage adapter for caching

## Estimated Complexity

Medium - Mostly integration work with existing AI infrastructure
