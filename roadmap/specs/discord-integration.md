# IN-08: Discord Integration

## Overview
Forward emails to Discord channels via webhooks.

## Technical Requirements
```go
type DiscordPlugin struct {
    webhookURL string
}

func (p *DiscordPlugin) Notify(email *Email) error {
    msg := DiscordMessage{
        Embeds: []Embed{{
            Title:       email.Subject,
            Description: email.Snippet,
            Color:       0x4ECDC4,
            Fields: []Field{
                {Name: "From", Value: email.FromEmail},
                {Name: "Date", Value: email.Date.Format(time.RFC1123)},
            },
        }},
    }
    return p.sendWebhook(msg)
}
```

## Estimated Complexity
Low
