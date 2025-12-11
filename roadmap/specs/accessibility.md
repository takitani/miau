# UX-12: Accessibility (a11y)

## Overview
Ensure miau is accessible to users with disabilities.

## Technical Requirements
- High contrast theme
- Screen reader compatibility
- Keyboard-only navigation
- Configurable font sizes
- Color blind friendly indicators

```go
type A11ySettings struct {
    HighContrast      bool
    LargeText         bool
    ReduceMotion      bool
    AnnounceNewEmails bool
    UseSymbols        bool  // Use symbols instead of colors
}
```

## Acceptance Criteria
- [ ] High contrast mode
- [ ] Works without mouse
- [ ] Screen reader friendly output
- [ ] Color blind indicators

## Estimated Complexity
Medium
