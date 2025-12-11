# AI-06: AI Auto-Categorization

## Overview

Automatically categorize incoming emails using AI to help users organize their inbox without manual labeling.

## User Stories

1. As a user, I want new emails to be automatically categorized (Work, Personal, Newsletters, etc.)
2. As a user, I want to define custom categories
3. As a user, I want to train the AI by correcting miscategorizations
4. As a user, I want to see category suggestions before applying

## Technical Requirements

### Service Layer

Create `internal/services/categorization.go`:

```go
package services

type CategorizationService interface {
    // CategorizeEmail suggests categories for an email
    CategorizeEmail(ctx context.Context, emailID int64) (*CategorySuggestion, error)

    // AutoCategorize categorizes and applies labels to new emails
    AutoCategorize(ctx context.Context, emailID int64) error

    // GetCategories returns all available categories
    GetCategories(ctx context.Context, accountID int64) ([]Category, error)

    // CreateCategory creates a custom category
    CreateCategory(ctx context.Context, cat Category) error

    // TrainCategory improves categorization by user feedback
    TrainCategory(ctx context.Context, emailID int64, correctCategory string) error

    // GetCategoryStats returns categorization statistics
    GetCategoryStats(ctx context.Context, accountID int64) (*CategoryStats, error)
}

type Category struct {
    ID          int64
    AccountID   int64
    Name        string
    Color       string
    Icon        string
    Description string
    IsSystem    bool  // true for default categories
    EmailCount  int
}

type CategorySuggestion struct {
    EmailID     int64
    Suggestions []SuggestedCategory
    Confidence  float64
}

type SuggestedCategory struct {
    Category   string
    Confidence float64
    Reason     string
}

// Default categories
var DefaultCategories = []string{
    "Work",
    "Personal",
    "Newsletters",
    "Social",
    "Promotions",
    "Updates",
    "Finance",
    "Travel",
}
```

### Database Schema

```sql
CREATE TABLE categories (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    name TEXT NOT NULL,
    color TEXT DEFAULT '#808080',
    icon TEXT,
    description TEXT,
    is_system BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, name)
);

CREATE TABLE email_categories (
    id INTEGER PRIMARY KEY,
    email_id INTEGER REFERENCES emails(id),
    category_id INTEGER REFERENCES categories(id),
    confidence REAL,
    is_manual BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(email_id, category_id)
);

CREATE TABLE categorization_training (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    from_pattern TEXT,  -- e.g., "*@newsletter.com"
    subject_pattern TEXT,
    category_id INTEGER REFERENCES categories(id),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### AI Prompt Template

```go
var categorizationPrompt = `Categorize this email into one or more categories.

Available categories: {{.Categories}}

Email:
From: {{.From}}
Subject: {{.Subject}}
Snippet: {{.Snippet}}

Output JSON:
{
  "primary": "category_name",
  "secondary": ["other", "categories"],
  "confidence": 0.95,
  "reason": "Brief explanation"
}`
```

### Auto-categorization Hook

```go
// In sync service, after new email is saved
func (s *SyncService) onNewEmail(ctx context.Context, email *Email) {
    if s.config.AutoCategorize {
        go s.categorization.AutoCategorize(ctx, email.ID)
    }
}
```

## UI/UX

### TUI
- Category badges next to email subjects
- Press `l` to see/change category
- Filter by category in folder view

```
┌─ INBOX ───────────────────────────────────────────────────────────┐
│ [Work]    John Smith     │ Q4 Budget Review        │ 10:30 AM    │
│ [Social]  LinkedIn       │ 5 new connections       │ 10:15 AM    │
│ [Promo]   Amazon         │ Your order shipped      │ 09:45 AM    │
│ [News]    TechCrunch     │ Daily Digest           │ 08:00 AM    │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Category chips in email list
- Category filter sidebar
- Drag-and-drop to change category
- Settings page for managing categories

## Testing

1. Test categorization with various email types
2. Test custom category creation
3. Test training/feedback loop
4. Test category migration when deleting category
5. Test sync with Gmail labels (optional integration)

## Acceptance Criteria

- [ ] Default categories are created on first run
- [ ] New emails are automatically categorized
- [ ] Users can create custom categories
- [ ] Users can correct/train categorization
- [ ] Category badges visible in email list
- [ ] Can filter inbox by category
- [ ] Categories persist across sessions
- [ ] Works with large mailboxes (>10k emails)

## Configuration

```yaml
# config.yaml
ai:
  auto_categorize: true
  categorize_on_sync: true
  min_confidence: 0.7  # Don't apply below this threshold
```

## Dependencies

- AI service for categorization
- Storage for categories and mappings
- Sync service for hook integration

## Estimated Complexity

Medium-High - Requires AI integration, training loop, and UI updates
