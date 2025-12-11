# UX-06: Help Overlay

## Overview

A comprehensive keyboard shortcut reference accessible via `?` key.

## User Stories

1. As a new user, I want to see all available keyboard shortcuts
2. As a user, I want tips for using miau effectively
3. As a user, I want context-sensitive help
4. As a user, I want to search for commands

## Technical Requirements

### Help Component

```go
type HelpOverlay struct {
    Visible    bool
    Tab        int  // 0: shortcuts, 1: tips, 2: search
    SearchQuery string
    Categories []HelpCategory
}

type HelpCategory struct {
    Name      string
    Shortcuts []Shortcut
}

type Shortcut struct {
    Key         string
    Description string
    Context     string  // "global", "inbox", "compose", etc.
}

var AllShortcuts = []HelpCategory{
    {
        Name: "Navigation",
        Shortcuts: []Shortcut{
            {Key: "j/↓", Description: "Move down", Context: "global"},
            {Key: "k/↑", Description: "Move up", Context: "global"},
            {Key: "Tab", Description: "Switch panels", Context: "global"},
            {Key: "g", Description: "Go to folder", Context: "inbox"},
            {Key: "/", Description: "Search", Context: "inbox"},
            {Key: "Home", Description: "First email", Context: "inbox"},
            {Key: "End", Description: "Last email", Context: "inbox"},
        },
    },
    {
        Name: "Email Actions",
        Shortcuts: []Shortcut{
            {Key: "Enter", Description: "Open email", Context: "inbox"},
            {Key: "e", Description: "Archive", Context: "inbox"},
            {Key: "x/#", Description: "Delete/Trash", Context: "inbox"},
            {Key: "s", Description: "Toggle star", Context: "inbox"},
            {Key: "m", Description: "Toggle read/unread", Context: "inbox"},
            {Key: "l", Description: "Add label", Context: "inbox"},
        },
    },
    {
        Name: "Compose",
        Shortcuts: []Shortcut{
            {Key: "c", Description: "New email", Context: "inbox"},
            {Key: "r", Description: "Reply", Context: "inbox"},
            {Key: "R", Description: "Reply all", Context: "inbox"},
            {Key: "f", Description: "Forward", Context: "inbox"},
            {Key: "Ctrl+Enter", Description: "Send", Context: "compose"},
            {Key: "Esc", Description: "Save draft & close", Context: "compose"},
        },
    },
    {
        Name: "AI Assistant",
        Shortcuts: []Shortcut{
            {Key: "a", Description: "Open AI panel", Context: "inbox"},
            {Key: "/dr", Description: "Draft reply", Context: "ai"},
            {Key: "/resume", Description: "Summarize", Context: "ai"},
            {Key: "/action", Description: "Extract action items", Context: "ai"},
        },
    },
    {
        Name: "Views",
        Shortcuts: []Shortcut{
            {Key: "V", Description: "VIP inbox", Context: "inbox"},
            {Key: "D", Description: "Digest view", Context: "inbox"},
            {Key: "A", Description: "Analytics", Context: "inbox"},
            {Key: "S", Description: "Settings", Context: "global"},
        },
    },
    {
        Name: "Selection",
        Shortcuts: []Shortcut{
            {Key: "Space", Description: "Toggle selection", Context: "inbox"},
            {Key: "Shift+j/k", Description: "Extend selection", Context: "inbox"},
            {Key: "Ctrl+a", Description: "Select all", Context: "inbox"},
            {Key: "Esc", Description: "Clear selection", Context: "inbox"},
        },
    },
}
```

### Help View

```go
func (h HelpOverlay) View() string {
    if !h.Visible {
        return ""
    }

    var b strings.Builder

    // Header
    b.WriteString("┌─ miau Help ────────────────────────────────────────────────────────┐\n")
    b.WriteString("│                                                                     │\n")

    // Tabs
    tabs := []string{"[1] Shortcuts", "[2] Tips", "[3] Search"}
    for i, tab := range tabs {
        if i == h.Tab {
            b.WriteString(fmt.Sprintf("│ %s ", lipgloss.NewStyle().Bold(true).Render(tab)))
        } else {
            b.WriteString(fmt.Sprintf("│ %s ", tab))
        }
    }
    b.WriteString("│\n")
    b.WriteString("│─────────────────────────────────────────────────────────────────────│\n")

    switch h.Tab {
    case 0:
        b.WriteString(h.renderShortcuts())
    case 1:
        b.WriteString(h.renderTips())
    case 2:
        b.WriteString(h.renderSearch())
    }

    b.WriteString("│                                                                     │\n")
    b.WriteString("│             Press ? or Esc to close  │  Tab to switch sections     │\n")
    b.WriteString("└─────────────────────────────────────────────────────────────────────┘")

    return b.String()
}

func (h HelpOverlay) renderShortcuts() string {
    var b strings.Builder

    // Two-column layout
    leftCol := h.Categories[:len(h.Categories)/2]
    rightCol := h.Categories[len(h.Categories)/2:]

    for i := 0; i < max(len(leftCol), len(rightCol)); i++ {
        left := ""
        right := ""

        if i < len(leftCol) {
            left = formatCategory(leftCol[i])
        }
        if i < len(rightCol) {
            right = formatCategory(rightCol[i])
        }

        b.WriteString(fmt.Sprintf("│ %-32s │ %-32s │\n", left, right))
    }

    return b.String()
}
```

## UI/UX

### Shortcuts Tab

```
┌─ miau Help ────────────────────────────────────────────────────────┐
│                                                                     │
│ [1] Shortcuts    [2] Tips    [3] Search                            │
│─────────────────────────────────────────────────────────────────────│
│                                                                     │
│  NAVIGATION                      EMAIL ACTIONS                      │
│  ───────────                     ─────────────                      │
│  j/k or ↑/↓   Navigate list      Enter   Open email                │
│  Tab          Toggle panels      e       Archive                   │
│  /            Fuzzy search       x/#     Move to trash             │
│  g            Go to folder       s       Star/unstar               │
│  Home/End     First/last         m       Mark read/unread          │
│                                                                     │
│  COMPOSE                         AI ASSISTANT                       │
│  ───────                         ────────────                       │
│  c            New email          a       Open AI panel             │
│  r            Reply              /dr     Draft reply               │
│  R            Reply all          /resume Summarize                 │
│  f            Forward            /action Extract actions           │
│                                                                     │
│  SELECTION                       GENERAL                            │
│  ─────────                       ───────                            │
│  Space        Toggle select      S       Settings                  │
│  Shift+j/k    Extend selection   ?       This help                 │
│  Ctrl+a       Select all         q       Quit                      │
│                                                                     │
│             Press ? or Esc to close  │  Tab to switch sections     │
└─────────────────────────────────────────────────────────────────────┘
```

### Tips Tab

```
│─────────────────────────────────────────────────────────────────────│
│                                                                     │
│  TIPS & TRICKS                                                      │
│                                                                     │
│  • Use AI for batch operations: "archive all emails from X"         │
│  • Fuzzy search works with partial words                            │
│  • Press / to start searching immediately                           │
│  • Use VIP inbox (V) for important contacts only                    │
│  • Snooze emails (z) to deal with them later                        │
│  • Quick commands: /dr for draft reply, /resume for summary         │
│  • Ctrl+Z to undo most actions                                      │
│  • Use templates (T) for repetitive emails                          │
│                                                                     │
```

### Search Tab

```
│─────────────────────────────────────────────────────────────────────│
│                                                                     │
│  Search: [archive_____________]                                     │
│                                                                     │
│  Results:                                                           │
│  • e - Archive email                                                │
│  • "archive all from X" - AI batch archive                          │
│  • Archive folder - View archived emails                            │
│                                                                     │
```

## Testing

1. Test overlay display
2. Test tab navigation
3. Test search functionality
4. Test context-sensitive help
5. Test keyboard dismiss

## Acceptance Criteria

- [ ] `?` opens help overlay
- [ ] Shows all keyboard shortcuts
- [ ] Organized by category
- [ ] Tips section included
- [ ] Search works
- [ ] Esc closes overlay
- [ ] Two-column layout fits terminal
- [ ] Context-sensitive (shows relevant shortcuts)

## Estimated Complexity

Low-Medium - Static content with search
