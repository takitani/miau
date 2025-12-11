# UX-08: Dark/Light Themes

## Overview

Support multiple color themes including dark mode, light mode, and custom themes.

## User Stories

1. As a user, I want to switch between dark and light mode
2. As a user, I want high-contrast themes for accessibility
3. As a user, I want to customize colors
4. As a user, I want themes to persist across sessions

## Technical Requirements

### Theme System

```go
package theme

type Theme struct {
    Name        string
    Background  lipgloss.Color
    Foreground  lipgloss.Color
    Primary     lipgloss.Color
    Secondary   lipgloss.Color
    Accent      lipgloss.Color
    Error       lipgloss.Color
    Warning     lipgloss.Color
    Success     lipgloss.Color
    Muted       lipgloss.Color
    Border      lipgloss.Color
    Selection   lipgloss.Color
    Highlight   lipgloss.Color
}

var Themes = map[string]Theme{
    "dark": {
        Name:       "Dark",
        Background: lipgloss.Color("#1a1a1a"),
        Foreground: lipgloss.Color("#ffffff"),
        Primary:    lipgloss.Color("#4ECDC4"),
        Secondary:  lipgloss.Color("#45B7D1"),
        Accent:     lipgloss.Color("#96CEB4"),
        Error:      lipgloss.Color("#FF6B6B"),
        Warning:    lipgloss.Color("#FFE66D"),
        Success:    lipgloss.Color("#4ECDC4"),
        Muted:      lipgloss.Color("#666666"),
        Border:     lipgloss.Color("#333333"),
        Selection:  lipgloss.Color("#2d2d2d"),
        Highlight:  lipgloss.Color("#3d3d3d"),
    },
    "light": {
        Name:       "Light",
        Background: lipgloss.Color("#ffffff"),
        Foreground: lipgloss.Color("#1a1a1a"),
        Primary:    lipgloss.Color("#0066cc"),
        Secondary:  lipgloss.Color("#0099cc"),
        Accent:     lipgloss.Color("#00aa66"),
        Error:      lipgloss.Color("#cc0000"),
        Warning:    lipgloss.Color("#cc9900"),
        Success:    lipgloss.Color("#00aa66"),
        Muted:      lipgloss.Color("#999999"),
        Border:     lipgloss.Color("#dddddd"),
        Selection:  lipgloss.Color("#e6f2ff"),
        Highlight:  lipgloss.Color("#f0f0f0"),
    },
    "high-contrast": {
        Name:       "High Contrast",
        Background: lipgloss.Color("#000000"),
        Foreground: lipgloss.Color("#ffffff"),
        Primary:    lipgloss.Color("#00ff00"),
        Secondary:  lipgloss.Color("#00ffff"),
        Accent:     lipgloss.Color("#ffff00"),
        Error:      lipgloss.Color("#ff0000"),
        Warning:    lipgloss.Color("#ffff00"),
        Success:    lipgloss.Color("#00ff00"),
        Muted:      lipgloss.Color("#888888"),
        Border:     lipgloss.Color("#ffffff"),
        Selection:  lipgloss.Color("#0000ff"),
        Highlight:  lipgloss.Color("#333333"),
    },
    "solarized-dark": {
        Name:       "Solarized Dark",
        Background: lipgloss.Color("#002b36"),
        Foreground: lipgloss.Color("#839496"),
        Primary:    lipgloss.Color("#268bd2"),
        Secondary:  lipgloss.Color("#2aa198"),
        Accent:     lipgloss.Color("#859900"),
        Error:      lipgloss.Color("#dc322f"),
        Warning:    lipgloss.Color("#b58900"),
        Success:    lipgloss.Color("#859900"),
        Muted:      lipgloss.Color("#586e75"),
        Border:     lipgloss.Color("#073642"),
        Selection:  lipgloss.Color("#073642"),
        Highlight:  lipgloss.Color("#073642"),
    },
    "dracula": {
        Name:       "Dracula",
        Background: lipgloss.Color("#282a36"),
        Foreground: lipgloss.Color("#f8f8f2"),
        Primary:    lipgloss.Color("#bd93f9"),
        Secondary:  lipgloss.Color("#8be9fd"),
        Accent:     lipgloss.Color("#50fa7b"),
        Error:      lipgloss.Color("#ff5555"),
        Warning:    lipgloss.Color("#ffb86c"),
        Success:    lipgloss.Color("#50fa7b"),
        Muted:      lipgloss.Color("#6272a4"),
        Border:     lipgloss.Color("#44475a"),
        Selection:  lipgloss.Color("#44475a"),
        Highlight:  lipgloss.Color("#44475a"),
    },
}

var currentTheme = Themes["dark"]

func GetTheme() Theme {
    return currentTheme
}

func SetTheme(name string) error {
    theme, ok := Themes[name]
    if !ok {
        return fmt.Errorf("theme not found: %s", name)
    }
    currentTheme = theme
    return nil
}
```

### Styled Components

```go
func (m Model) renderEmailList() string {
    theme := GetTheme()

    listStyle := lipgloss.NewStyle().
        Background(theme.Background).
        Foreground(theme.Foreground)

    selectedStyle := lipgloss.NewStyle().
        Background(theme.Selection).
        Foreground(theme.Foreground).
        Bold(true)

    unreadStyle := lipgloss.NewStyle().
        Foreground(theme.Primary).
        Bold(true)

    var b strings.Builder
    for i, email := range m.emails {
        style := listStyle
        if i == m.selectedIndex {
            style = selectedStyle
        }
        if !email.IsRead {
            style = style.Inherit(unreadStyle)
        }
        b.WriteString(style.Render(formatEmail(email)))
        b.WriteString("\n")
    }

    return b.String()
}
```

### Theme Configuration

```yaml
# config.yaml
theme:
  name: "dark"  # dark, light, high-contrast, solarized-dark, dracula
  custom:
    primary: "#4ECDC4"
    accent: "#96CEB4"
```

## UI/UX

### Theme Selector in Settings

```
┌─ Theme Settings ──────────────────────────────────────────────────┐
│                                                                   │
│ Select theme:                                                     │
│                                                                   │
│ [●] Dark (default)                                                │
│ [ ] Light                                                         │
│ [ ] High Contrast                                                 │
│ [ ] Solarized Dark                                                │
│ [ ] Dracula                                                       │
│ [ ] Custom...                                                     │
│                                                                   │
│ Preview:                                                          │
│ ┌─────────────────────────────────────────────────────────────┐   │
│ │ ● John Smith    Project Update         10:30 AM            │   │
│ │   Newsletter    Weekly Digest          Yesterday           │   │
│ │   Amazon        Your order shipped     Dec 13              │   │
│ └─────────────────────────────────────────────────────────────┘   │
│                                                                   │
│ [Enter] Apply  [Esc] Cancel                                       │
└───────────────────────────────────────────────────────────────────┘
```

### Quick Theme Toggle

- Press `Ctrl+T` to cycle through themes
- Or `Ctrl+D` for dark/light toggle

## Testing

1. Test all built-in themes
2. Test custom color configuration
3. Test persistence across restarts
4. Test readability in all themes
5. Test terminal compatibility

## Acceptance Criteria

- [ ] Dark theme (default)
- [ ] Light theme
- [ ] High contrast theme
- [ ] Popular themes (Solarized, Dracula)
- [ ] Custom color support
- [ ] Theme persists in config
- [ ] Quick toggle shortcut
- [ ] Works in Desktop app too

## Estimated Complexity

Low-Medium - Lipgloss styling system
