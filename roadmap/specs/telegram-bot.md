# IN-09: Telegram Bot

## Overview
Telegram bot for email notifications and quick actions.

## Features
- New email notifications
- Reply to emails via Telegram
- Quick actions (archive, star)
- Search emails

## Commands
```
/inbox - Show recent emails
/search <query> - Search emails
/compose - Start composing (opens app)
/mute - Pause notifications
```

## Technical Requirements
```go
type TelegramBot struct {
    token  string
    chatID int64
    api    *tgbotapi.BotAPI
}

func (b *TelegramBot) HandleUpdate(update tgbotapi.Update) {
    switch update.Message.Command() {
    case "inbox":
        emails := b.emailService.GetRecent(5)
        b.sendEmailList(emails)
    case "search":
        query := update.Message.CommandArguments()
        results := b.emailService.Search(query)
        b.sendEmailList(results)
    }
}
```

## Estimated Complexity
Medium
