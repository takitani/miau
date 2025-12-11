# IN-05: Slack Integration

## Overview

Forward important emails to Slack channels and send email summaries.

## Technical Requirements

```go
type SlackPlugin struct {
    webhookURL string
    channel    string
}

func (p *SlackPlugin) OnEmailReceived(email *Email) error {
    if !p.shouldNotify(email) {
        return nil
    }

    msg := SlackMessage{
        Channel: p.channel,
        Text:    fmt.Sprintf("New email from %s: %s", email.FromName, email.Subject),
        Attachments: []Attachment{{
            Color: "#4ECDC4",
            Fields: []Field{
                {Title: "From", Value: email.FromEmail},
                {Title: "Subject", Value: email.Subject},
                {Title: "Snippet", Value: email.Snippet},
            },
        }},
    }

    return p.sendToSlack(msg)
}
```

### Config

```yaml
plugins:
  slack:
    enabled: true
    webhook_url: "${SLACK_WEBHOOK_URL}"
    channel: "#email-alerts"
    notify_vip: true
    notify_keywords: ["urgent", "invoice"]
```

## Acceptance Criteria

- [ ] Forward emails to Slack
- [ ] Filter by VIP/keywords
- [ ] Rich message formatting
- [ ] Configurable triggers

## Estimated Complexity

Low-Medium (uses existing plugin system)
