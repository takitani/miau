# AI-10: AI Email Prioritization

## Overview

Automatically prioritize emails based on importance, urgency, and user patterns to help users focus on what matters most.

## User Stories

1. As a user, I want my inbox sorted by AI-determined priority
2. As a user, I want VIP contacts to always be high priority
3. As a user, I want to train the AI by marking emails as important/not important
4. As a user, I want to see why an email was prioritized

## Technical Requirements

### Service Layer

Create `internal/services/prioritization.go`:

```go
package services

type PrioritizationService interface {
    // PrioritizeEmail calculates priority for an email
    PrioritizeEmail(ctx context.Context, emailID int64) (*PriorityScore, error)

    // BatchPrioritize prioritizes multiple emails
    BatchPrioritize(ctx context.Context, emailIDs []int64) error

    // GetPrioritizedInbox returns emails sorted by priority
    GetPrioritizedInbox(ctx context.Context, accountID int64, limit int) ([]Email, error)

    // TrainPriority records user feedback for learning
    TrainPriority(ctx context.Context, emailID int64, isImportant bool) error

    // AddVIP marks a contact as VIP (always high priority)
    AddVIP(ctx context.Context, accountID int64, email string) error

    // RemoveVIP removes VIP status
    RemoveVIP(ctx context.Context, accountID int64, email string) error

    // GetVIPs returns all VIP contacts
    GetVIPs(ctx context.Context, accountID int64) ([]string, error)
}

type PriorityScore struct {
    EmailID     int64
    Score       float64  // 0-100
    Level       PriorityLevel
    Factors     []PriorityFactor
    IsVIP       bool
    NeedsReply  bool
    HasDeadline bool
}

type PriorityLevel string

const (
    PriorityUrgent PriorityLevel = "urgent"
    PriorityHigh   PriorityLevel = "high"
    PriorityMedium PriorityLevel = "medium"
    PriorityLow    PriorityLevel = "low"
)

type PriorityFactor struct {
    Name        string  // "vip_sender", "urgent_language", "deadline_mentioned"
    Weight      float64
    Description string
}

// Scoring weights
var PriorityWeights = map[string]float64{
    "vip_sender":        30,
    "direct_to_me":      20,
    "reply_to_my_email": 15,
    "urgent_language":   15,
    "deadline_detected": 10,
    "question_to_me":    10,
    "thread_active":     5,
    "recent_interaction": 5,
}
```

### Database Schema

```sql
CREATE TABLE email_priority (
    id INTEGER PRIMARY KEY,
    email_id INTEGER REFERENCES emails(id) UNIQUE,
    score REAL NOT NULL,
    level TEXT NOT NULL,
    factors TEXT,  -- JSON array of PriorityFactor
    calculated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE vip_contacts (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    email TEXT NOT NULL,
    name TEXT,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, email)
);

CREATE TABLE priority_training (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    email_id INTEGER REFERENCES emails(id),
    is_important BOOLEAN NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_priority_score ON email_priority(score DESC);
CREATE INDEX idx_priority_level ON email_priority(level);
```

### AI Prompt Template

```go
var priorityPrompt = `Analyze this email's priority and urgency.

From: {{.From}}
To: {{.To}}
Subject: {{.Subject}}
Snippet: {{.Snippet}}

Context:
- Is from VIP: {{.IsVIP}}
- Is direct to me: {{.IsDirect}}
- Thread depth: {{.ThreadDepth}}

Evaluate these factors:
1. Urgency language (ASAP, urgent, deadline)
2. Question directed at recipient
3. Action required
4. Time sensitivity
5. Business importance

Output JSON:
{
  "score": 75,  // 0-100
  "level": "high",
  "needs_reply": true,
  "deadline_mentioned": true,
  "factors": [
    {"name": "urgent_language", "description": "Contains 'ASAP'"},
    {"name": "question_to_me", "description": "Asks for budget approval"}
  ]
}`
```

## UI/UX

### TUI
- Priority indicator in email list (!, !!, !!!)
- Toggle priority sort with `P`
- Quick VIP toggle with `V`

```
┌─ INBOX (Priority View) ───────────────────────────────────────────┐
│ !!! Boss            │ URGENT: Board meeting     │ VIP  10:30 AM  │
│ !!  Client ABC      │ Contract review needed    │      09:45 AM  │
│ !   Team            │ RE: Project update        │      09:00 AM  │
│     Newsletter      │ Weekly digest             │      08:00 AM  │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Priority badges/colors in email list
- Priority sort option
- VIP management in settings
- Priority explanation tooltip

## Testing

1. Test scoring algorithm
2. Test VIP handling
3. Test training feedback loop
4. Test with large inbox
5. Test performance of batch prioritization

## Acceptance Criteria

- [ ] Emails scored and ranked by priority
- [ ] VIP contacts always prioritized
- [ ] User can train by marking important/not important
- [ ] Priority reasons visible
- [ ] Sort by priority works
- [ ] Handles new emails quickly
- [ ] Works in both TUI and Desktop

## Configuration

```yaml
# config.yaml
ai:
  prioritization:
    enabled: true
    default_sort: true  # Use as default inbox sort
    weights:
      vip_sender: 30
      urgent_language: 15
```

## Estimated Complexity

Medium - Scoring algorithm plus AI analysis
