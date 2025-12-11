# AI-08: AI Sentiment Analysis

## Overview

Analyze the emotional tone and sentiment of emails to help users prioritize responses and understand context.

## User Stories

1. As a user, I want to see sentiment indicators on emails (positive, negative, urgent)
2. As a user, I want to filter emails by sentiment
3. As a user, I want to be alerted to negative/urgent emails
4. As a user, I want sentiment trends in analytics

## Technical Requirements

### Service Layer

Create `internal/services/sentiment.go`:

```go
package services

type SentimentService interface {
    // AnalyzeEmail analyzes sentiment of a single email
    AnalyzeEmail(ctx context.Context, emailID int64) (*SentimentResult, error)

    // BatchAnalyze analyzes multiple emails
    BatchAnalyze(ctx context.Context, emailIDs []int64) ([]SentimentResult, error)

    // GetSentimentStats returns sentiment statistics
    GetSentimentStats(ctx context.Context, accountID int64, period string) (*SentimentStats, error)

    // GetNegativeEmails returns emails needing attention
    GetNegativeEmails(ctx context.Context, accountID int64, limit int) ([]Email, error)
}

type Sentiment string

const (
    SentimentPositive Sentiment = "positive"
    SentimentNeutral  Sentiment = "neutral"
    SentimentNegative Sentiment = "negative"
    SentimentUrgent   Sentiment = "urgent"
    SentimentAngry    Sentiment = "angry"
)

type SentimentResult struct {
    EmailID    int64
    Sentiment  Sentiment
    Score      float64  // -1 to 1
    Confidence float64  // 0 to 1
    Emotions   []Emotion
    Keywords   []string // Words that influenced the analysis
}

type Emotion struct {
    Name  string  // "happy", "frustrated", "anxious", etc.
    Score float64
}

type SentimentStats struct {
    Period       string
    TotalEmails  int
    ByCategory   map[Sentiment]int
    TrendChange  float64  // Compared to previous period
    TopNegative  []Email
    MoodTimeline []MoodPoint
}

type MoodPoint struct {
    Date      time.Time
    AvgScore  float64
    Count     int
}
```

### Database Schema

```sql
CREATE TABLE email_sentiment (
    id INTEGER PRIMARY KEY,
    email_id INTEGER REFERENCES emails(id) UNIQUE,
    sentiment TEXT NOT NULL,
    score REAL NOT NULL,
    confidence REAL NOT NULL,
    emotions TEXT,  -- JSON array
    keywords TEXT,  -- JSON array
    analyzed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sentiment_score ON email_sentiment(score);
CREATE INDEX idx_sentiment_type ON email_sentiment(sentiment);
```

### AI Prompt Template

```go
var sentimentPrompt = `Analyze the sentiment and emotions in this email.

From: {{.From}}
Subject: {{.Subject}}
Body: {{.Body}}

Output JSON:
{
  "sentiment": "positive|neutral|negative|urgent|angry",
  "score": 0.5,  // -1 (very negative) to 1 (very positive)
  "confidence": 0.9,
  "emotions": [
    {"name": "frustrated", "score": 0.7},
    {"name": "anxious", "score": 0.3}
  ],
  "keywords": ["deadline", "urgent", "disappointed"],
  "summary": "Sender appears frustrated about missed deadline"
}`
```

## UI/UX

### TUI
- Sentiment emoji/icon in email list
- Color coding (green=positive, yellow=neutral, red=negative)

```
┌─ INBOX ───────────────────────────────────────────────────────────┐
│ 😊 John Smith     │ Great news about Q4!        │ 10:30 AM       │
│ 😐 Newsletter     │ Weekly digest               │ 10:15 AM       │
│ 😠 Client ABC     │ RE: Delayed shipment        │ 09:45 AM       │
│ ⚠️  Boss          │ URGENT: Need response       │ 08:00 AM       │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Sentiment badge/chip in email list
- Sentiment filter in sidebar
- Sentiment graph in analytics
- Notification for negative emails (optional)

## Testing

1. Test various sentiment types
2. Test confidence scoring
3. Test batch analysis performance
4. Test with sarcasm/irony (known limitation)
5. Test multilingual emails
6. Test statistics calculation

## Acceptance Criteria

- [ ] Sentiment analyzed on email sync
- [ ] Visual indicators in email list
- [ ] Filter by sentiment works
- [ ] Analytics shows sentiment trends
- [ ] Handles non-English emails
- [ ] Caches results to avoid re-analysis
- [ ] Works with both TUI and Desktop

## Configuration

```yaml
# config.yaml
ai:
  sentiment:
    enabled: true
    analyze_on_sync: true
    notify_negative: false
    show_indicators: true
```

## Estimated Complexity

Medium - AI integration with optional notifications
