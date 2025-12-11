# SC-07: Link Safety Check

## Overview
Check URLs in emails against safety databases before opening.

## Technical Requirements
```go
type LinkSafetyService interface {
    CheckURL(ctx context.Context, url string) (*SafetyResult, error)
    CheckEmail(ctx context.Context, emailID int64) ([]LinkCheck, error)
}

type SafetyResult struct {
    URL        string
    IsSafe     bool
    Threats    []string
    Source     string  // "google", "virustotal", etc.
}
```

## Integration Options
- Google Safe Browsing API
- VirusTotal API
- Local blocklist

## UI
```
⚠️ This link may be unsafe:
https://suspicious-site.com/login

Threats detected:
• Phishing
• Malware distribution

[Open anyway] [Cancel]
```

## Estimated Complexity
Medium
