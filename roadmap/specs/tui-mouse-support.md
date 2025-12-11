# UX-05: Mouse Support (TUI)

## Overview

Add mouse interaction support to the TUI for users who prefer clicking over keyboard navigation.

## User Stories

1. As a user, I want to click on emails to select them
2. As a user, I want to scroll with my mouse wheel
3. As a user, I want to double-click to open emails
4. As a user, I want context menus on right-click

## Technical Requirements

### Bubble Tea Mouse Setup

```go
// In main.go
func main() {
    p := tea.NewProgram(
        model,
        tea.WithAltScreen(),
        tea.WithMouseCellMotion(),  // Enable mouse tracking
    )
    p.Run()
}
```

### Mouse Event Handling

```go
// In Update function
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.MouseMsg:
        switch msg.Type {
        case tea.MouseLeft:
            // Single click - select
            if m.isInEmailList(msg.X, msg.Y) {
                index := m.getEmailIndexFromY(msg.Y)
                if index >= 0 && index < len(m.emails) {
                    m.selectedIndex = index
                }
            } else if m.isInFolderList(msg.X, msg.Y) {
                folder := m.getFolderFromY(msg.Y)
                if folder != "" {
                    m.currentFolder = folder
                    return m, m.loadFolder(folder)
                }
            }

        case tea.MouseRight:
            // Right click - context menu
            if m.isInEmailList(msg.X, msg.Y) {
                index := m.getEmailIndexFromY(msg.Y)
                m.selectedIndex = index
                m.showContextMenu = true
                m.contextMenuX = msg.X
                m.contextMenuY = msg.Y
            }

        case tea.MouseWheelUp:
            // Scroll up
            if m.isInEmailList(msg.X, msg.Y) {
                m.scrollUp()
            }

        case tea.MouseWheelDown:
            // Scroll down
            if m.isInEmailList(msg.X, msg.Y) {
                m.scrollDown()
            }

        case tea.MouseMotion:
            // Hover effects
            m.hoveredIndex = m.getEmailIndexFromY(msg.Y)
        }

    // ... existing keyboard handling
    }

    return m, nil
}
```

### Double-Click Detection

```go
type Model struct {
    // ...existing fields
    lastClickTime  time.Time
    lastClickIndex int
    doubleClickThreshold time.Duration
}

func (m *Model) handleLeftClick(x, y int) tea.Cmd {
    index := m.getEmailIndexFromY(y)
    if index < 0 {
        return nil
    }

    now := time.Now()
    isDoubleClick := m.lastClickIndex == index &&
                     now.Sub(m.lastClickTime) < m.doubleClickThreshold

    m.lastClickTime = now
    m.lastClickIndex = index
    m.selectedIndex = index

    if isDoubleClick {
        // Open email viewer
        return m.openEmail(index)
    }

    return nil
}
```

### Context Menu Component

```go
type ContextMenu struct {
    X, Y    int
    Items   []ContextMenuItem
    Selected int
}

type ContextMenuItem struct {
    Label    string
    Shortcut string
    Action   func() tea.Cmd
}

func (m *Model) getContextMenuItems() []ContextMenuItem {
    email := m.emails[m.selectedIndex]
    return []ContextMenuItem{
        {Label: "Open", Shortcut: "Enter", Action: m.openEmail},
        {Label: "Reply", Shortcut: "r", Action: m.reply},
        {Label: "Forward", Shortcut: "f", Action: m.forward},
        {Label: "---", Shortcut: "", Action: nil},  // Separator
        {Label: "Archive", Shortcut: "e", Action: m.archive},
        {Label: "Delete", Shortcut: "x", Action: m.delete},
        {Label: "Star", Shortcut: "s", Action: m.toggleStar},
        {Label: "---", Shortcut: "", Action: nil},
        {Label: "Mark as VIP", Shortcut: "V", Action: m.markVIP},
    }
}

func (m ContextMenu) View() string {
    var b strings.Builder
    b.WriteString("┌─────────────────┐\n")
    for i, item := range m.Items {
        if item.Label == "---" {
            b.WriteString("├─────────────────┤\n")
            continue
        }
        prefix := "│ "
        if i == m.Selected {
            prefix = "│▶"
        }
        b.WriteString(fmt.Sprintf("%s%-10s %5s │\n", prefix, item.Label, item.Shortcut))
    }
    b.WriteString("└─────────────────┘")
    return b.String()
}
```

### Hover Effects

```go
func (m Model) renderEmailItem(index int, email Email) string {
    style := lipgloss.NewStyle()

    if index == m.selectedIndex {
        style = style.Background(lipgloss.Color("240"))
    } else if index == m.hoveredIndex {
        style = style.Background(lipgloss.Color("236"))
    }

    return style.Render(formatEmail(email))
}
```

## UI/UX

### Mouse Interactions

| Action | Result |
|--------|--------|
| Click on email | Select email |
| Double-click | Open email |
| Right-click | Show context menu |
| Scroll wheel | Scroll email list |
| Click on folder | Switch to folder |
| Ctrl+Click | Add to selection |
| Shift+Click | Range selection |

### Context Menu

```
┌─────────────────┐
│ Open      Enter │
│ Reply         r │
│ Forward       f │
├─────────────────┤
│ Archive       e │
│ Delete        x │
│ Star          s │
├─────────────────┤
│ Mark as VIP   V │
│ Add label   ... │
└─────────────────┘
```

### Visual Feedback

- Hover: Subtle background highlight
- Selected: Strong background highlight
- Clicking: Brief animation/flash
- Scrollbar: Visual position indicator

## Testing

1. Test click selection
2. Test double-click timing
3. Test scroll behavior
4. Test context menu positioning
5. Test Ctrl/Shift+Click selection
6. Test in various terminal emulators

## Acceptance Criteria

- [ ] Single click selects email
- [ ] Double-click opens email
- [ ] Scroll wheel works
- [ ] Right-click shows context menu
- [ ] Hover effects visible
- [ ] Ctrl+Click adds to selection
- [ ] Shift+Click range selection
- [ ] Works in common terminals
- [ ] Can still use keyboard

## Configuration

```yaml
# config.yaml
tui:
  mouse_enabled: true
  double_click_ms: 500
  hover_highlight: true
```

## Estimated Complexity

Medium - Bubble Tea has built-in support
