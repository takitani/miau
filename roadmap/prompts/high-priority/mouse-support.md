# Prompt: Mouse Support no TUI

> Use este prompt com Claude Code para implementar suporte a mouse no TUI.

## Contexto

O TUI do miau usa Bubble Tea que já tem suporte a mouse. Precisamos habilitar e implementar interações de mouse.

## Objetivo

Implementar suporte completo a mouse:
1. Click para selecionar email
2. Double-click para abrir
3. Scroll para navegar
4. Ctrl+Click para multi-select
5. Shift+Click para range select

## Arquivos Relevantes

```
internal/tui/inbox/inbox.go      # TUI principal
cmd/miau/main.go                 # Entry point (onde criar Program)
```

## Tasks

### 1. Habilitar Mouse no Program

Atualizar `cmd/miau/main.go`:

```go
import tea "github.com/charmbracelet/bubbletea"

func main() {
    // ... código existente ...

    p := tea.NewProgram(
        model,
        tea.WithAltScreen(),
        tea.WithMouseCellMotion(),  // Habilitar mouse
    )

    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### 2. Tratar Eventos de Mouse

Adicionar ao `internal/tui/inbox/inbox.go`:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    // ... casos existentes ...

    case tea.MouseMsg:
        return m.handleMouse(msg)
    }
    // ...
}

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
    switch msg.Type {
    case tea.MouseLeft:
        return m.handleMouseClick(msg)

    case tea.MouseWheelUp:
        return m.scrollUp(3), nil

    case tea.MouseWheelDown:
        return m.scrollDown(3), nil

    case tea.MouseMotion:
        // Highlight on hover (opcional)
        return m.handleMouseHover(msg), nil
    }
    return m, nil
}

func (m *Model) handleMouseClick(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
    // Calcular qual email foi clicado baseado na posição Y
    emailIndex := m.getEmailIndexFromY(msg.Y)
    if emailIndex < 0 || emailIndex >= len(m.emails) {
        return m, nil
    }

    // Verificar modificadores
    if msg.Ctrl {
        // Ctrl+Click: toggle selection
        m.toggleSelection(emailIndex)
        return m, nil
    }

    if msg.Shift && m.lastClickIndex >= 0 {
        // Shift+Click: range selection
        m.selectRange(m.lastClickIndex, emailIndex)
        return m, nil
    }

    // Click normal: selecionar email
    m.selectedIndex = emailIndex
    m.lastClickIndex = emailIndex

    // Double-click detection
    now := time.Now()
    if m.lastClickTime.Add(300*time.Millisecond).After(now) &&
       m.lastClickY == msg.Y {
        // Double-click: abrir email
        return m, m.openSelectedEmail()
    }
    m.lastClickTime = now
    m.lastClickY = msg.Y

    return m, nil
}

func (m *Model) getEmailIndexFromY(y int) int {
    // Calcular baseado no layout
    // headerHeight + (y - headerHeight) / rowHeight + scrollOffset
    headerHeight := 3 // Ajustar baseado no layout real
    rowHeight := 1

    if y < headerHeight {
        return -1 // Clicou no header
    }

    relativeY := y - headerHeight
    index := relativeY/rowHeight + m.scrollOffset

    return index
}

func (m *Model) scrollUp(lines int) Model {
    m.scrollOffset = max(0, m.scrollOffset-lines)
    return *m
}

func (m *Model) scrollDown(lines int) Model {
    maxOffset := max(0, len(m.emails)-m.visibleRows)
    m.scrollOffset = min(maxOffset, m.scrollOffset+lines)
    return *m
}

// Multi-select helpers
func (m *Model) toggleSelection(index int) {
    email := m.emails[index]
    if m.selectedIDs[email.ID] {
        delete(m.selectedIDs, email.ID)
    } else {
        m.selectedIDs[email.ID] = true
    }
}

func (m *Model) selectRange(from, to int) {
    start := min(from, to)
    end := max(from, to)
    for i := start; i <= end; i++ {
        if i >= 0 && i < len(m.emails) {
            m.selectedIDs[m.emails[i].ID] = true
        }
    }
}
```

### 3. Estado para Mouse

Adicionar campos ao Model:

```go
type Model struct {
    // ... campos existentes ...

    // Mouse state
    lastClickTime  time.Time
    lastClickY     int
    lastClickIndex int

    // Multi-select
    selectedIDs    map[int64]bool
    selectionMode  bool
}
```

### 4. Visual Feedback

Atualizar o render para mostrar seleções:

```go
func (m Model) renderEmailRow(email Email, index int, isSelected bool) string {
    // Determinar estilo
    style := m.styles.Normal
    if isSelected {
        style = m.styles.Selected
    }
    if m.selectedIDs[email.ID] {
        style = m.styles.MultiSelected
    }

    // Checkbox para multi-select
    checkbox := "  "
    if m.selectedIDs[email.ID] {
        checkbox = "✓ "
    }

    return style.Render(fmt.Sprintf("%s%s", checkbox, email.Subject))
}
```

### 5. Ações em Batch

Quando há seleção múltipla, ações aplicam a todos:

```go
func (m *Model) archiveSelected() tea.Cmd {
    if len(m.selectedIDs) == 0 {
        // Arquivar apenas o selecionado
        return m.archiveEmail(m.emails[m.selectedIndex].ID)
    }

    // Arquivar todos selecionados
    return func() tea.Msg {
        for id := range m.selectedIDs {
            m.app.Email().Archive(context.Background(), id)
        }
        return batchCompleteMsg{action: "archive", count: len(m.selectedIDs)}
    }
}
```

## Critérios de Aceitação

- [ ] Click seleciona email
- [ ] Double-click abre email
- [ ] Scroll wheel funciona
- [ ] Ctrl+Click toggle selection
- [ ] Shift+Click range selection
- [ ] Visual feedback de hover (opcional)
- [ ] Batch actions funcionam
- [ ] Não quebra navegação por teclado

## Testes Manuais

1. Abrir TUI
2. Clicar em um email - deve selecionar
3. Scroll up/down - deve navegar
4. Ctrl+Click em vários - deve multi-selecionar
5. Pressionar `e` - deve arquivar todos selecionados
6. Double-click - deve abrir email

---

*Prompt criado: 2025-12-12*
