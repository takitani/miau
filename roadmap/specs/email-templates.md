# EM-09: Email Templates

## Overview

Create, manage, and use email templates for quick composition of common emails.

## User Stories

1. As a user, I want to save frequently used emails as templates
2. As a user, I want to use variables in templates (recipient name, date, etc.)
3. As a user, I want to organize templates by category
4. As a user, I want to quickly insert a template when composing

## Technical Requirements

### Service Layer

Create `internal/services/template.go`:

```go
package services

type TemplateService interface {
    // CreateTemplate creates a new template
    CreateTemplate(ctx context.Context, template Template) (*Template, error)

    // GetTemplates returns all templates for account
    GetTemplates(ctx context.Context, accountID int64) ([]Template, error)

    // GetTemplate returns a specific template
    GetTemplate(ctx context.Context, id int64) (*Template, error)

    // UpdateTemplate updates a template
    UpdateTemplate(ctx context.Context, template Template) error

    // DeleteTemplate deletes a template
    DeleteTemplate(ctx context.Context, id int64) error

    // ApplyTemplate renders template with variables
    ApplyTemplate(ctx context.Context, id int64, vars TemplateVars) (*RenderedTemplate, error)

    // CreateFromEmail creates template from existing email
    CreateFromEmail(ctx context.Context, emailID int64, name string) (*Template, error)
}

type Template struct {
    ID          int64
    AccountID   int64
    Name        string
    Category    string
    Subject     string
    BodyText    string
    BodyHTML    string
    Variables   []TemplateVariable
    UsageCount  int
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type TemplateVariable struct {
    Name        string  // e.g., "recipient_name"
    Description string  // e.g., "Recipient's first name"
    Default     string  // Default value if not provided
    Required    bool
}

type TemplateVars map[string]string

type RenderedTemplate struct {
    Subject  string
    BodyText string
    BodyHTML string
}

// Built-in variables
var BuiltInVars = map[string]func(ctx context.Context) string{
    "{{today}}":     func(ctx context.Context) string { return time.Now().Format("January 2, 2006") },
    "{{tomorrow}}":  func(ctx context.Context) string { return time.Now().AddDate(0, 0, 1).Format("January 2, 2006") },
    "{{my_name}}":   func(ctx context.Context) string { /* from account */ return "" },
    "{{my_email}}":  func(ctx context.Context) string { /* from account */ return "" },
    "{{my_company}}": func(ctx context.Context) string { /* from config */ return "" },
}
```

### Database Schema

```sql
CREATE TABLE email_templates (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    name TEXT NOT NULL,
    category TEXT,
    subject TEXT,
    body_text TEXT,
    body_html TEXT,
    variables TEXT,  -- JSON array of TemplateVariable
    usage_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, name)
);

CREATE INDEX idx_templates_category ON email_templates(category);
```

### Template Syntax

```go
// Template example:
// Subject: Meeting follow-up with {{recipient_name}}
// Body:
// Hi {{recipient_name}},
//
// It was great meeting with you on {{meeting_date}}.
// As discussed, I'm sending over the {{document_type}}.
//
// Please let me know if you have any questions.
//
// Best regards,
// {{my_name}}

func (s *TemplateService) ApplyTemplate(ctx context.Context, id int64, vars TemplateVars) (*RenderedTemplate, error) {
    template, err := s.GetTemplate(ctx, id)
    if err != nil {
        return nil, err
    }

    // Merge built-in vars with provided vars
    allVars := s.getBuiltInVars(ctx)
    for k, v := range vars {
        allVars[k] = v
    }

    // Render subject and body
    subject := s.renderVars(template.Subject, allVars)
    bodyText := s.renderVars(template.BodyText, allVars)
    bodyHTML := s.renderVars(template.BodyHTML, allVars)

    return &RenderedTemplate{
        Subject:  subject,
        BodyText: bodyText,
        BodyHTML: bodyHTML,
    }, nil
}
```

## UI/UX

### TUI
- Press `T` in compose to select template
- Press `Ctrl+T` to save current compose as template

```
┌─ Select Template ─────────────────────────────────────────────────┐
│ Category: All ▼                                                   │
│                                                                   │
│ Work                                                              │
│   📋 Meeting Follow-up                              Used: 15x     │
│   📋 Project Update                                 Used: 8x      │
│   📋 Invoice Reminder                               Used: 5x      │
│                                                                   │
│ Personal                                                          │
│   📋 Thank You Note                                 Used: 12x     │
│   📋 Birthday Wishes                                Used: 3x      │
│                                                                   │
│ [Enter] Use template  [n] New  [e] Edit  [d] Delete              │
└───────────────────────────────────────────────────────────────────┘

┌─ Fill Variables ──────────────────────────────────────────────────┐
│ Template: Meeting Follow-up                                       │
│                                                                   │
│ recipient_name: [John                    ]                        │
│ meeting_date:   [December 15, 2024       ]                        │
│ document_type:  [project proposal        ]                        │
│                                                                   │
│ [Enter] Apply  [Esc] Cancel                                       │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Template picker in compose toolbar
- Template manager in settings
- Drag-and-drop template organization
- Template preview

## Testing

1. Test template CRUD operations
2. Test variable substitution
3. Test built-in variables
4. Test create from email
5. Test category filtering
6. Test HTML template rendering
7. Test missing variable handling

## Acceptance Criteria

- [ ] Can create new templates
- [ ] Can use variables in templates
- [ ] Built-in variables work (date, name, etc.)
- [ ] Can organize by category
- [ ] Can create template from existing email
- [ ] Quick template selection in compose
- [ ] Variable prompts when applying template
- [ ] Usage tracking
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
templates:
  default_category: "General"
  built_in_vars:
    my_company: "Example Corp"
```

## Estimated Complexity

Medium - CRUD plus variable rendering
