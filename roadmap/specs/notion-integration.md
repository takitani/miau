# IN-07: Notion Integration

## Overview
Save emails to Notion databases.

## Features
- Save email to Notion page
- Create task in Notion
- Link emails to Notion pages

## Technical Requirements
```go
type NotionPlugin struct {
    apiKey     string
    databaseID string
}

func (p *NotionPlugin) SaveEmail(email *Email) error {
    page := NotionPage{
        Parent:     DatabaseParent{ID: p.databaseID},
        Properties: map[string]Property{
            "Title":   {Title: []Text{{Content: email.Subject}}},
            "From":    {Email: email.FromEmail},
            "Date":    {Date: email.Date},
        },
        Children: []Block{
            {Type: "paragraph", Text: email.BodyText},
        },
    }
    return p.api.CreatePage(page)
}
```

## Estimated Complexity
Medium
