# Prompt: Help Overlay (Tecla ?)

> Use este prompt com Claude Code para implementar overlay de ajuda no TUI e Desktop.

## Contexto

Usuários precisam de uma forma fácil de ver todos os atalhos de teclado. Pressionar `?` deve abrir um overlay com documentação.

## Objetivo

Implementar help overlay que:
1. Mostra todos os atalhos de teclado
2. Organizado por categoria
3. Tips & tricks
4. Fecha com qualquer tecla

## TUI Implementation

### 1. Help Model

Criar `internal/tui/help/help.go`:

```go
package help

import (
    "strings"

    "github.com/charmbracelet/bubbles/key"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type Model struct {
    width  int
    height int
    styles Styles
}

type Styles struct {
    Title    lipgloss.Style
    Category lipgloss.Style
    Key      lipgloss.Style
    Desc     lipgloss.Style
    Tip      lipgloss.Style
    Border   lipgloss.Style
}

func New() Model {
    return Model{
        styles: DefaultStyles(),
    }
}

func DefaultStyles() Styles {
    return Styles{
        Title:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")),
        Category: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")),
        Key:      lipgloss.NewStyle().Background(lipgloss.Color("237")).Padding(0, 1),
        Desc:     lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
        Tip:      lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("244")),
        Border:   lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2),
    }
}

type Shortcut struct {
    Key  string
    Desc string
}

type Category struct {
    Name      string
    Shortcuts []Shortcut
}

func (m Model) GetCategories() []Category {
    return []Category{
        {
            Name: "NAVIGATION",
            Shortcuts: []Shortcut{
                {"j/k or ↑/↓", "Navigate list"},
                {"Tab", "Toggle folder panel"},
                {"Enter", "Open email"},
                {"/", "Fuzzy search"},
                {"g", "Go to folder"},
                {"Home/End", "First/last email"},
            },
        },
        {
            Name: "EMAIL ACTIONS",
            Shortcuts: []Shortcut{
                {"e", "Archive"},
                {"x or #", "Move to trash"},
                {"s", "Star/unstar"},
                {"u", "Mark unread"},
                {"r", "Sync (refresh)"},
            },
        },
        {
            Name: "COMPOSE",
            Shortcuts: []Shortcut{
                {"c", "New email"},
                {"R", "Reply"},
                {"Shift+R", "Reply all"},
                {"f", "Forward"},
            },
        },
        {
            Name: "AI ASSISTANT",
            Shortcuts: []Shortcut{
                {"a", "Open AI panel"},
                {"/dr", "Draft reply"},
                {"/resume", "Summarize email"},
                {"/action", "Extract actions"},
            },
        },
        {
            Name: "GENERAL",
            Shortcuts: []Shortcut{
                {"S", "Settings"},
                {"d", "View drafts"},
                {"A", "Analytics"},
                {"i", "Image preview"},
                {"?", "This help"},
                {"q", "Quit"},
            },
        },
    }
}

func (m Model) GetTips() []string {
    return []string{
        "Use AI for batch operations: 'archive all newsletters'",
        "Fuzzy search with partial words: 'joh' finds 'John'",
        "Quick commands: /dr formal, /resume, /action",
        "Image preview works in terminals with Sixel/iTerm2",
    }
}

func (m Model) View() string {
    var b strings.Builder

    // Title
    title := m.styles.Title.Render("⌨ miau Help")
    b.WriteString(title + "\n\n")

    // Categories (2 columns)
    categories := m.GetCategories()
    leftCol := categories[:3]
    rightCol := categories[3:]

    leftContent := m.renderCategories(leftCol)
    rightContent := m.renderCategories(rightCol)

    // Side by side
    cols := lipgloss.JoinHorizontal(
        lipgloss.Top,
        leftContent,
        "    ",
        rightContent,
    )
    b.WriteString(cols + "\n\n")

    // Tips
    b.WriteString(m.styles.Category.Render("TIPS & TRICKS") + "\n")
    for _, tip := range m.GetTips() {
        b.WriteString(m.styles.Tip.Render("• " + tip) + "\n")
    }

    // Footer
    b.WriteString("\n" + m.styles.Tip.Render("[Press any key to close]"))

    return m.styles.Border.Render(b.String())
}

func (m Model) renderCategories(cats []Category) string {
    var b strings.Builder
    for _, cat := range cats {
        b.WriteString(m.styles.Category.Render(cat.Name) + "\n")
        for _, s := range cat.Shortcuts {
            key := m.styles.Key.Render(s.Key)
            desc := m.styles.Desc.Render(s.Desc)
            b.WriteString(fmt.Sprintf("  %s  %s\n", key, desc))
        }
        b.WriteString("\n")
    }
    return b.String()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
    switch msg.(type) {
    case tea.KeyMsg:
        // Qualquer tecla fecha
        return m, func() tea.Msg { return CloseHelpMsg{} }
    }
    return m, nil
}

type CloseHelpMsg struct{}
```

### 2. Integrar no Inbox

Atualizar `internal/tui/inbox/inbox.go`:

```go
import "miau/internal/tui/help"

type Model struct {
    // ... campos existentes ...
    helpModel    help.Model
    showHelp     bool
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Se help está aberto, delegar para help model
    if m.showHelp {
        hm, cmd := m.helpModel.Update(msg)
        m.helpModel = hm
        if _, ok := msg.(help.CloseHelpMsg); ok {
            m.showHelp = false
        }
        return m, cmd
    }

    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "?":
            m.showHelp = true
            return m, nil
        }
    }
    // ... resto do código ...
}

func (m Model) View() string {
    if m.showHelp {
        // Overlay centralizado
        return m.renderWithOverlay(m.helpModel.View())
    }
    // ... view normal ...
}

func (m Model) renderWithOverlay(overlay string) string {
    // Centralizar overlay na tela
    overlayWidth := lipgloss.Width(overlay)
    overlayHeight := lipgloss.Height(overlay)

    x := (m.width - overlayWidth) / 2
    y := (m.height - overlayHeight) / 2

    // Renderizar background dimmed + overlay
    return lipgloss.Place(
        m.width, m.height,
        lipgloss.Center, lipgloss.Center,
        overlay,
    )
}
```

## Desktop Implementation

O Desktop já tem `HelpOverlay.svelte`. Verificar se está completo:

```svelte
<!-- HelpOverlay.svelte -->
<script>
  import { onMount, onDestroy } from 'svelte';

  export let show = false;

  function handleKeydown(e) {
    if (show) {
      show = false;
      e.preventDefault();
    } else if (e.key === '?') {
      show = true;
      e.preventDefault();
    }
  }

  onMount(() => {
    document.addEventListener('keydown', handleKeydown);
  });

  onDestroy(() => {
    document.removeEventListener('keydown', handleKeydown);
  });

  const categories = [
    {
      name: "Navigation",
      shortcuts: [
        { key: "j/k", desc: "Navigate list" },
        { key: "Tab", desc: "Toggle folders" },
        { key: "Enter", desc: "Open email" },
        { key: "/", desc: "Search" },
      ]
    },
    // ... mais categorias
  ];
</script>

{#if show}
  <div class="overlay" on:click={() => show = false}>
    <div class="help-panel" on:click|stopPropagation>
      <h2>Keyboard Shortcuts</h2>

      <div class="columns">
        {#each categories as cat}
          <div class="category">
            <h3>{cat.name}</h3>
            {#each cat.shortcuts as s}
              <div class="shortcut">
                <kbd>{s.key}</kbd>
                <span>{s.desc}</span>
              </div>
            {/each}
          </div>
        {/each}
      </div>

      <div class="tips">
        <h3>Tips</h3>
        <ul>
          <li>Use AI for batch operations</li>
          <li>Fuzzy search with partial words</li>
        </ul>
      </div>

      <p class="footer">Press any key to close</p>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .help-panel {
    background: var(--bg-primary);
    border-radius: var(--radius-lg);
    padding: var(--space-xl);
    max-width: 800px;
    max-height: 80vh;
    overflow-y: auto;
  }

  .columns {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: var(--space-lg);
  }

  kbd {
    background: var(--bg-tertiary);
    padding: 2px 8px;
    border-radius: 4px;
    font-family: monospace;
    margin-right: var(--space-sm);
  }

  .shortcut {
    padding: var(--space-xs) 0;
  }
</style>
```

## Critérios de Aceitação

- [ ] TUI: `?` abre overlay
- [ ] TUI: Qualquer tecla fecha
- [ ] TUI: Centralizado na tela
- [ ] Desktop: `?` abre overlay
- [ ] Desktop: Click fora fecha
- [ ] Todas as categorias presentes
- [ ] Tips incluídos
- [ ] Visual bonito e legível

---

*Prompt criado: 2025-12-12*
