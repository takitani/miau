# IN-06: Todoist Integration

## Overview
Create Todoist tasks from emails.

## Technical Requirements
```go
type TodoistPlugin struct {
    apiToken string
}

func (p *TodoistPlugin) CreateTask(email *Email) error {
    task := TodoistTask{
        Content:     fmt.Sprintf("Reply to: %s", email.Subject),
        Description: fmt.Sprintf("From: %s\n\n%s", email.FromEmail, email.Snippet),
        DueString:   "tomorrow",
        Priority:    2,
    }
    return p.api.CreateTask(task)
}
```

## Actions
- Create task from email (keyboard shortcut)
- Link task to email
- AI extract tasks from email content

## Estimated Complexity
Low-Medium
