# UX-07: About Screen

## Overview

Display application information, credits, and author details.

## User Stories

1. As a user, I want to see the app version
2. As a user, I want to know who made the app
3. As a user, I want links to documentation and support
4. As a user, I want to see technology credits

## Technical Requirements

### About Component

```go
type AboutScreen struct {
    Visible bool
}

var AppInfo = struct {
    Name        string
    Version     string
    Description string
    Author      Author
    Links       []Link
    Credits     []Credit
    License     string
}{
    Name:        "miau",
    Version:     "1.0.0",
    Description: "Mail Intelligence Assistant Utility",
    Author: Author{
        Name:     "André Takitani",
        Email:    "andre@exato.digital",
        LinkedIn: "linkedin.com/in/takitani",
        GitHub:   "github.com/takitani",
        Company:  "Exato Digital",
        Website:  "exato.digital",
    },
    Links: []Link{
        {Label: "GitHub", URL: "github.com/takitani/miau"},
        {Label: "Documentation", URL: "github.com/takitani/miau/wiki"},
        {Label: "Report Issue", URL: "github.com/takitani/miau/issues"},
    },
    Credits: []Credit{
        {Name: "Go", URL: "golang.org"},
        {Name: "Bubble Tea", URL: "github.com/charmbracelet/bubbletea"},
        {Name: "Lipgloss", URL: "github.com/charmbracelet/lipgloss"},
        {Name: "SQLite", URL: "sqlite.org"},
        {Name: "go-imap", URL: "github.com/emersion/go-imap"},
        {Name: "Claude AI", URL: "anthropic.com"},
    },
    License: "MIT",
}

type Author struct {
    Name     string
    Email    string
    LinkedIn string
    GitHub   string
    Company  string
    Website  string
}

type Link struct {
    Label string
    URL   string
}

type Credit struct {
    Name string
    URL  string
}
```

### About View

```go
func (a AboutScreen) View() string {
    if !a.Visible {
        return ""
    }

    logo := `
    ╭──────────╮
    │  ┌────┐  │
    │  │ 🐱 │  │
    │  └────┘  │
    ╰──────────╯
    `

    var b strings.Builder

    b.WriteString("┌─ About miau ────────────────────────────────────────────────────────┐\n")
    b.WriteString("│                                                                      │\n")
    b.WriteString(centerLogo(logo))
    b.WriteString("│                                                                      │\n")
    b.WriteString(center(fmt.Sprintf("miau v%s", AppInfo.Version)))
    b.WriteString(center("Mail Intelligence Assistant Utility"))
    b.WriteString("│                                                                      │\n")
    b.WriteString("│  ────────────────────────────────────────────────────────────────── │\n")
    b.WriteString("│                                                                      │\n")
    b.WriteString(fmt.Sprintf("│  Created by: %-55s │\n", AppInfo.Author.Name))
    b.WriteString("│                                                                      │\n")
    b.WriteString(fmt.Sprintf("│  🔗 LinkedIn: %-54s │\n", AppInfo.Author.LinkedIn))
    b.WriteString(fmt.Sprintf("│  🌐 Website:  %-54s │\n", AppInfo.Author.Website))
    b.WriteString(fmt.Sprintf("│  📧 GitHub:   %-54s │\n", AppInfo.Author.GitHub))
    b.WriteString("│                                                                      │\n")
    b.WriteString("│  ────────────────────────────────────────────────────────────────── │\n")
    b.WriteString("│                                                                      │\n")
    b.WriteString("│  Built with:                                                         │\n")
    b.WriteString("│  • Go + Bubble Tea                                                   │\n")
    b.WriteString("│  • SQLite + FTS5                                                     │\n")
    b.WriteString("│  • Claude AI                                                         │\n")
    b.WriteString("│  • Wails + Svelte (Desktop)                                          │\n")
    b.WriteString("│                                                                      │\n")
    b.WriteString(fmt.Sprintf("│  License: %-58s │\n", AppInfo.License))
    b.WriteString("│                                                                      │\n")
    b.WriteString("└──────────────────────────────────────────────────────────────────────┘\n")
    b.WriteString("                              [Press any key to close]                  ")

    return b.String()
}
```

## UI/UX

### TUI About Screen

```
┌─ About miau ────────────────────────────────────────────────────────┐
│                                                                      │
│                          ╭──────────╮                                │
│                          │  ┌────┐  │                                │
│                          │  │ 🐱 │  │                                │
│                          │  └────┘  │                                │
│                          ╰──────────╯                                │
│                                                                      │
│                     miau v1.0.0                                      │
│           Mail Intelligence Assistant Utility                        │
│                                                                      │
│  ────────────────────────────────────────────────────────────────── │
│                                                                      │
│  Created by: André Takitani                                          │
│                                                                      │
│  🔗 LinkedIn: linkedin.com/in/takitani                               │
│  🌐 Website:  exato.digital                                          │
│  📧 GitHub:   github.com/takitani/miau                               │
│                                                                      │
│  ────────────────────────────────────────────────────────────────── │
│                                                                      │
│  Built with:                                                         │
│  • Go + Bubble Tea                                                   │
│  • SQLite + FTS5                                                     │
│  • Claude AI                                                         │
│  • Wails + Svelte (Desktop)                                          │
│                                                                      │
│  License: MIT                                                        │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
                              [Press any key to close]
```

### Access Methods

- Press `Ctrl+?` or from Settings menu
- `miau --version` in CLI shows version + basic info

## Testing

1. Test display on various terminal sizes
2. Test clickable links (OSC 8 support)
3. Test version number accuracy
4. Test from Settings menu access

## Acceptance Criteria

- [ ] Shows version number
- [ ] Shows author information
- [ ] Shows contact links
- [ ] Shows technology credits
- [ ] Shows license
- [ ] Accessible from Settings
- [ ] CLI version flag works
- [ ] Links work in supporting terminals

## Estimated Complexity

Low - Static content display
