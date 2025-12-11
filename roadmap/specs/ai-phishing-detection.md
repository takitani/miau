# AI-14: AI Phishing Detection

## Overview

Use AI to detect potential phishing emails and warn users about suspicious content.

## User Stories

1. As a user, I want warnings about suspicious emails
2. As a user, I want to see why an email is flagged
3. As a user, I want links checked for safety
4. As a user, I want to report false positives

## Technical Requirements

### Service Layer

Create `internal/services/phishing.go`:

```go
package services

type PhishingDetectionService interface {
    // AnalyzeEmail checks email for phishing indicators
    AnalyzeEmail(ctx context.Context, emailID int64) (*PhishingAnalysis, error)

    // AnalyzeLink checks a URL for safety
    AnalyzeLink(ctx context.Context, url string) (*LinkAnalysis, error)

    // ReportFalsePositive marks email as safe
    ReportFalsePositive(ctx context.Context, emailID int64) error

    // GetSuspiciousEmails returns flagged emails
    GetSuspiciousEmails(ctx context.Context, accountID int64) ([]Email, error)
}

type PhishingAnalysis struct {
    EmailID      int64
    RiskLevel    RiskLevel
    Score        float64  // 0-100 (higher = more suspicious)
    Indicators   []PhishingIndicator
    Links        []LinkAnalysis
    Recommendation string
}

type RiskLevel string

const (
    RiskSafe     RiskLevel = "safe"
    RiskLow      RiskLevel = "low"
    RiskMedium   RiskLevel = "medium"
    RiskHigh     RiskLevel = "high"
    RiskCritical RiskLevel = "critical"
)

type PhishingIndicator struct {
    Type        IndicatorType
    Description string
    Severity    string
    Evidence    string
}

type IndicatorType string

const (
    IndicatorSpoofedSender   IndicatorType = "spoofed_sender"
    IndicatorUrgency         IndicatorType = "urgency_language"
    IndicatorSuspiciousLink  IndicatorType = "suspicious_link"
    IndicatorMismatchDomain  IndicatorType = "domain_mismatch"
    IndicatorRequestsInfo    IndicatorType = "requests_sensitive_info"
    IndicatorGenericGreeting IndicatorType = "generic_greeting"
    IndicatorThreats         IndicatorType = "threat_language"
)

type LinkAnalysis struct {
    URL            string
    DisplayText    string
    ActualDomain   string
    IsSuspicious   bool
    Reason         string
    SafeBrowsing   string  // Google Safe Browsing result
}
```

### Database Schema

```sql
CREATE TABLE phishing_analysis (
    id INTEGER PRIMARY KEY,
    email_id INTEGER REFERENCES emails(id) UNIQUE,
    risk_level TEXT NOT NULL,
    score REAL NOT NULL,
    indicators TEXT,  -- JSON array
    links TEXT,  -- JSON array
    is_false_positive BOOLEAN DEFAULT 0,
    analyzed_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE phishing_whitelist (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    domain TEXT,
    email TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### AI Prompt Template

```go
var phishingPrompt = `Analyze this email for phishing indicators.

From: {{.From}}
Reply-To: {{.ReplyTo}}
Subject: {{.Subject}}
Body: {{.Body}}
Links: {{.Links}}

Check for:
1. Sender spoofing (display name vs actual email)
2. Urgency/threat language
3. Requests for sensitive information
4. Suspicious links (domain mismatch)
5. Generic greetings ("Dear Customer")
6. Grammar/spelling typical of phishing
7. Too-good-to-be-true offers

Output JSON:
{
  "risk_level": "high",
  "score": 75,
  "indicators": [
    {
      "type": "spoofed_sender",
      "description": "Display name 'PayPal' but sender is @random-domain.com",
      "severity": "high",
      "evidence": "From: PayPal <support@random-domain.com>"
    }
  ],
  "recommendation": "Do not click links. This appears to be a phishing attempt."
}`
```

## UI/UX

### TUI
- Warning banner on suspicious emails
- Don't open links without confirmation
- Risk indicator in email list

```
┌─ ⚠️  SECURITY WARNING ─────────────────────────────────────────────┐
│ This email may be a phishing attempt!                              │
│                                                                    │
│ Risk Level: HIGH (75/100)                                          │
│                                                                    │
│ Suspicious indicators found:                                       │
│ • Sender claims to be "PayPal" but email is from random-domain.com │
│ • Contains urgent language: "Account will be suspended"            │
│ • Link goes to different domain than displayed                     │
│                                                                    │
│ Recommendation: Do not click any links in this email.              │
├────────────────────────────────────────────────────────────────────┤
│ [Enter] View anyway  [d] Delete  [r] Report false positive         │
└────────────────────────────────────────────────────────────────────┘
```

### Desktop
- Red warning banner
- Link hover shows actual URL
- Block link opening without confirmation
- Settings for sensitivity level

## Testing

1. Test with known phishing patterns
2. Test legitimate emails (false positives)
3. Test link analysis
4. Test whitelist functionality
5. Test various phishing techniques

## Acceptance Criteria

- [ ] Detects common phishing patterns
- [ ] Warning shown before viewing suspicious email
- [ ] Links analyzed for domain mismatch
- [ ] Users can report false positives
- [ ] Whitelist prevents repeated warnings
- [ ] Low false positive rate
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
security:
  phishing_detection:
    enabled: true
    sensitivity: "medium"  # low, medium, high
    block_suspicious_links: true
    warn_before_open: true
```

## Estimated Complexity

Medium-High - Security analysis plus UI warnings
