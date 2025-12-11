# UX-13: Onboarding Tour

## Overview
Guide new users through key features on first launch.

## Technical Requirements
```go
type OnboardingStep struct {
    ID          string
    Title       string
    Description string
    Highlight   string  // UI element to highlight
    Action      string  // Key to press
}

var OnboardingSteps = []OnboardingStep{
    {ID: "welcome", Title: "Welcome to miau!", Description: "Let's take a quick tour..."},
    {ID: "navigation", Title: "Navigation", Description: "Use j/k or arrow keys to navigate", Highlight: "email_list"},
    {ID: "open", Title: "Open Email", Description: "Press Enter to open an email", Action: "Enter"},
    {ID: "compose", Title: "Compose", Description: "Press c to compose a new email", Action: "c"},
    {ID: "ai", Title: "AI Assistant", Description: "Press a to open the AI assistant", Action: "a"},
}
```

## Acceptance Criteria
- [ ] Shows on first launch
- [ ] Can skip/dismiss
- [ ] Highlights UI elements
- [ ] Saved as completed

## Estimated Complexity
Low-Medium
