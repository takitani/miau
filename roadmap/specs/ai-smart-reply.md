# AI-07: AI Smart Reply

## Overview

Generate context-aware reply suggestions that users can quickly select, similar to Gmail's Smart Reply feature.

## User Stories

1. As a user, I want to see 2-3 quick reply suggestions when viewing an email
2. As a user, I want to click a suggestion to start composing with that text
3. As a user, I want suggestions to match my writing style over time
4. As a user, I want to customize suggestion tones (formal, casual, brief)

## Technical Requirements

### Service Layer

Create `internal/services/smartreply.go`:

```go
package services

type SmartReplyService interface {
    // GenerateReplies generates reply suggestions for an email
    GenerateReplies(ctx context.Context, emailID int64, opts ReplyOptions) ([]SmartReply, error)

    // AcceptReply records when user selects a reply (for learning)
    AcceptReply(ctx context.Context, emailID int64, replyIndex int) error

    // GetUserStyle analyzes user's writing style from sent emails
    GetUserStyle(ctx context.Context, accountID int64) (*WritingStyle, error)

    // RefreshSuggestions regenerates suggestions with different parameters
    RefreshSuggestions(ctx context.Context, emailID int64) ([]SmartReply, error)
}

type ReplyOptions struct {
    Count     int           // Number of suggestions (default 3)
    Tone      ReplyTone     // Desired tone
    MaxLength int           // Max characters per suggestion
    Language  string        // Response language
}

type ReplyTone string

const (
    ToneAuto     ReplyTone = "auto"     // Detect from context
    ToneFormal   ReplyTone = "formal"
    ToneCasual   ReplyTone = "casual"
    ToneFriendly ReplyTone = "friendly"
    ToneBrief    ReplyTone = "brief"
)

type SmartReply struct {
    Text       string
    Tone       ReplyTone
    Confidence float64
    Category   string  // "positive", "negative", "question", "confirmation"
}

type WritingStyle struct {
    PreferredTone    ReplyTone
    AvgReplyLength   int
    CommonGreetings  []string
    CommonSignoffs   []string
    FormalityScore   float64  // 0-1
}
```

### Database Schema

```sql
CREATE TABLE smart_reply_cache (
    id INTEGER PRIMARY KEY,
    email_id INTEGER REFERENCES emails(id) UNIQUE,
    suggestions TEXT NOT NULL,  -- JSON array of SmartReply
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE smart_reply_usage (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    email_id INTEGER REFERENCES emails(id),
    selected_index INTEGER,
    selected_text TEXT,
    was_edited BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE writing_style (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id) UNIQUE,
    style_data TEXT NOT NULL,  -- JSON WritingStyle
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### AI Prompt Template

```go
var smartReplyPrompt = `Generate {{.Count}} short reply suggestions for this email.

Email context:
From: {{.From}}
Subject: {{.Subject}}
Body: {{.Body}}

User's writing style:
- Tone: {{.Style.PreferredTone}}
- Typical greeting: {{.Style.CommonGreetings}}
- Typical signoff: {{.Style.CommonSignoffs}}

Requirements:
- Each reply should be 1-3 sentences
- Vary the responses (positive, neutral, question)
- Match the language of the original email
- Tone: {{.Tone}}

Output JSON array:
[
  {"text": "reply text", "tone": "casual", "category": "positive"},
  {"text": "reply text", "tone": "casual", "category": "question"},
  {"text": "reply text", "tone": "casual", "category": "confirmation"}
]`
```

## UI/UX

### TUI
- Smart replies shown below email content when pressing `r`
- Number keys (1, 2, 3) to select a reply
- Selected reply populates compose

```
┌─ Quick Replies ───────────────────────────────────────────────────┐
│ [1] Sounds good! I'll review and get back to you by EOD.          │
│ [2] Thanks for sending this. Let me check with the team first.    │
│ [3] Can we schedule a call to discuss this further?               │
└───────────────────────────────────────────────────────────────────┘
  Press 1-3 to use, Enter to compose custom, Esc to cancel
```

### Desktop
- Smart reply chips below email viewer
- Click to start compose with that text
- Hover to see full text if truncated
- "Regenerate" button for new suggestions

## Testing

1. Test suggestion generation for various email types
2. Test tone detection and matching
3. Test writing style learning from sent emails
4. Test cache behavior
5. Test with non-English emails
6. Test with thread context

## Acceptance Criteria

- [ ] Shows 3 relevant reply suggestions
- [ ] Clicking suggestion opens compose with text
- [ ] Learns from user's writing style over time
- [ ] Suggestions match email language
- [ ] Handles different email types (questions, info, requests)
- [ ] Cache prevents re-generation on same email
- [ ] Regenerate button works
- [ ] Works in both TUI and Desktop

## Configuration

```yaml
# config.yaml
ai:
  smart_reply:
    enabled: true
    count: 3
    default_tone: "auto"
    max_length: 200
    learn_style: true
```

## Dependencies

- AI service for generation
- Email service for context
- Compose service for integration
- Storage for caching and learning

## Estimated Complexity

Medium - AI integration plus learning component
