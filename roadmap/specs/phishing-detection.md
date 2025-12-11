# SC-06: Phishing Detection

## Overview
Detect potential phishing emails using pattern matching and AI.

## Indicators Checked
- Sender domain mismatch
- Urgency language
- Suspicious links
- Generic greetings
- Request for credentials

## Implementation
See ai-phishing-detection.md for full AI implementation.

## Simple Pattern Detection
```go
func (s *PhishingService) QuickCheck(email *Email) []Warning {
    warnings := []Warning{}

    // Check sender mismatch
    if email.FromName != "" && !strings.Contains(email.FromEmail, extractDomain(email.FromName)) {
        warnings = append(warnings, Warning{Type: "sender_mismatch"})
    }

    // Check urgency words
    urgencyWords := []string{"urgent", "immediately", "suspended", "verify"}
    // ...

    return warnings
}
```

## Estimated Complexity
Medium (with AI), Low (pattern only)
