# SC-08: SPF/DKIM Display

## Overview
Show email authentication status (SPF, DKIM, DMARC).

## Technical Requirements
```go
type AuthResult struct {
    SPF   AuthStatus
    DKIM  AuthStatus
    DMARC AuthStatus
}

type AuthStatus struct {
    Pass   bool
    Result string
}

func ParseAuthResults(headers map[string]string) *AuthResult {
    // Parse Authentication-Results header
    authResults := headers["Authentication-Results"]
    // ...
}
```

## UI Display
```
From: john@example.com ✓ Verified
SPF: ✓ pass | DKIM: ✓ pass | DMARC: ✓ pass

From: suspicious@bad-domain.com ⚠️
SPF: ✗ fail | DKIM: ✗ none | DMARC: ✗ fail
```

## Estimated Complexity
Low
