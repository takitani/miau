# UX-09: Custom Keyboard Shortcuts

## Overview

Allow users to customize keyboard shortcuts.

## Technical Requirements

```go
type KeyBinding struct {
    Action  string
    Key     string
    Ctrl    bool
    Shift   bool
    Alt     bool
}

var DefaultBindings = map[string]KeyBinding{
    "archive": {Action: "archive", Key: "e"},
    "delete":  {Action: "delete", Key: "x"},
    "reply":   {Action: "reply", Key: "r"},
    "compose": {Action: "compose", Key: "c"},
}

func (m Model) handleKey(msg tea.KeyMsg) tea.Cmd {
    action := m.keymap.GetAction(msg)
    switch action {
    case "archive":
        return m.archiveEmail()
    case "reply":
        return m.reply()
    }
    return nil
}
```

### Config

```yaml
keybindings:
  archive: "a"        # Changed from 'e'
  delete: "d"         # Changed from 'x'
  reply: "r"
  compose: "n"        # Changed from 'c'
```

## Acceptance Criteria

- [ ] All shortcuts customizable
- [ ] Persisted in config
- [ ] Reset to defaults option
- [ ] Conflict detection

## Estimated Complexity

Low-Medium
