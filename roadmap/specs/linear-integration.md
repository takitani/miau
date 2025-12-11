# IN-10: Linear Integration

## Overview
Create Linear issues from emails.

## Technical Requirements
```go
type LinearPlugin struct {
    apiKey string
    teamID string
}

func (p *LinearPlugin) CreateIssue(email *Email) error {
    issue := LinearIssue{
        Title:       email.Subject,
        Description: fmt.Sprintf("From: %s\n\n%s", email.FromEmail, email.BodyText),
        TeamID:      p.teamID,
    }
    return p.api.CreateIssue(issue)
}
```

## Estimated Complexity
Low
