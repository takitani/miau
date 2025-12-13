# Prompt: Snippets/Templates (Trechos Reutilizáveis)

> Inspirado no Superhuman - Respostas rápidas com trechos salvos.

## Conceito

Salvar trechos de texto frequentemente usados para inserir rapidamente em emails. Shortcuts tipo `/meeting` expandem para texto completo.

```
Digitando: /meeting
Expande para:
─────────────────────────────────────
Olá,

Podemos agendar uma reunião para discutir este assunto?
Minha disponibilidade:
- Segunda a Sexta: 9h às 18h
- Prefiro chamadas de 30 minutos

Qual horário funciona para você?

Atenciosamente,
[Seu nome]
─────────────────────────────────────
```

## Tipos de Snippets

### 1. Texto Simples
```
/obrigado → "Obrigado pelo retorno!"
/assinatura → "Atenciosamente,\nJoão Silva"
```

### 2. Com Variáveis
```
/meeting → "Olá {{nome}}, podemos agendar..."
/followup → "Olá {{nome}}, gostaria de fazer um follow-up sobre {{assunto}}..."
```

### 3. Contextuais (Preenchimento Automático)
```
/reply-thanks → "Obrigado {{from_name}}, recebi seu email sobre {{subject}}..."
```

Variáveis disponíveis:
- `{{from_name}}` - Nome do remetente
- `{{from_email}}` - Email do remetente
- `{{subject}}` - Assunto do email
- `{{date}}` - Data atual
- `{{my_name}}` - Seu nome

## Database Schema

```sql
CREATE TABLE snippets (
    id INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    shortcut TEXT NOT NULL,          -- /meeting, /thanks, etc
    title TEXT NOT NULL,             -- "Agendar Reunião"
    content TEXT NOT NULL,           -- Texto do snippet
    category TEXT,                   -- work, personal, etc
    use_count INTEGER DEFAULT 0,     -- Para ordenar por frequência
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, shortcut)
);

-- Snippets compartilhados (todos os accounts)
CREATE TABLE shared_snippets (
    id INTEGER PRIMARY KEY,
    shortcut TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    category TEXT,
    is_builtin BOOLEAN DEFAULT 0     -- Snippets padrão do sistema
);

-- Index para busca rápida
CREATE INDEX idx_snippets_shortcut ON snippets(account_id, shortcut);
```

## Service Implementation

```go
// internal/services/snippet.go

type SnippetService struct {
    storage ports.SnippetStorage
}

type Snippet struct {
    ID        int64
    Shortcut  string
    Title     string
    Content   string
    Category  string
    UseCount  int
    Variables []string // Extraídas do content
}

// ExpandSnippet expande um shortcut para texto completo
func (s *SnippetService) ExpandSnippet(ctx context.Context, accountID int64, shortcut string, context *EmailContext) (string, error) {
    // Buscar snippet
    snippet, err := s.storage.GetByShortcut(ctx, accountID, shortcut)
    if err != nil {
        // Tentar snippets compartilhados
        snippet, err = s.storage.GetSharedByShortcut(ctx, shortcut)
        if err != nil {
            return "", fmt.Errorf("snippet not found: %s", shortcut)
        }
    }

    // Incrementar contador de uso
    go s.storage.IncrementUseCount(ctx, snippet.ID)

    // Substituir variáveis
    content := s.replaceVariables(snippet.Content, context)

    return content, nil
}

func (s *SnippetService) replaceVariables(content string, ctx *EmailContext) string {
    if ctx == nil {
        return content
    }

    replacements := map[string]string{
        "{{from_name}}":  ctx.FromName,
        "{{from_email}}": ctx.FromEmail,
        "{{subject}}":    ctx.Subject,
        "{{date}}":       time.Now().Format("02/01/2006"),
        "{{my_name}}":    ctx.MyName,
        "{{my_email}}":   ctx.MyEmail,
    }

    result := content
    for placeholder, value := range replacements {
        result = strings.ReplaceAll(result, placeholder, value)
    }

    return result
}

// SearchSnippets busca snippets por texto
func (s *SnippetService) SearchSnippets(ctx context.Context, accountID int64, query string) ([]Snippet, error) {
    return s.storage.Search(ctx, accountID, query)
}

// GetFrequentSnippets retorna snippets mais usados
func (s *SnippetService) GetFrequentSnippets(ctx context.Context, accountID int64, limit int) ([]Snippet, error) {
    return s.storage.GetByUseCount(ctx, accountID, limit)
}

// CreateSnippet cria novo snippet
func (s *SnippetService) CreateSnippet(ctx context.Context, accountID int64, snippet *Snippet) error {
    // Validar shortcut (deve começar com /)
    if !strings.HasPrefix(snippet.Shortcut, "/") {
        snippet.Shortcut = "/" + snippet.Shortcut
    }

    // Extrair variáveis do content
    snippet.Variables = s.extractVariables(snippet.Content)

    return s.storage.Create(ctx, accountID, snippet)
}

func (s *SnippetService) extractVariables(content string) []string {
    re := regexp.MustCompile(`\{\{(\w+)\}\}`)
    matches := re.FindAllStringSubmatch(content, -1)

    vars := make([]string, 0)
    seen := make(map[string]bool)
    for _, m := range matches {
        if !seen[m[1]] {
            vars = append(vars, m[1])
            seen[m[1]] = true
        }
    }
    return vars
}

// EmailContext contém dados para substituição de variáveis
type EmailContext struct {
    FromName  string
    FromEmail string
    Subject   string
    MyName    string
    MyEmail   string
}
```

## Default Snippets

```go
// Snippets padrão do sistema
var builtinSnippets = []Snippet{
    {
        Shortcut: "/thanks",
        Title:    "Agradecimento",
        Content:  "Obrigado pelo retorno!\n\nAtenciosamente,\n{{my_name}}",
    },
    {
        Shortcut: "/meeting",
        Title:    "Agendar Reunião",
        Content: `Olá {{from_name}},

Podemos agendar uma reunião para discutir este assunto?

Minha disponibilidade esta semana:
- Segunda a Sexta: 9h às 18h
- Prefiro chamadas de 30 minutos

Qual horário funciona para você?

Atenciosamente,
{{my_name}}`,
    },
    {
        Shortcut: "/followup",
        Title:    "Follow-up",
        Content: `Olá {{from_name}},

Gostaria de fazer um follow-up sobre nosso último contato.

Você teve chance de analisar a proposta?

Fico à disposição para quaisquer dúvidas.

Atenciosamente,
{{my_name}}`,
    },
    {
        Shortcut: "/ack",
        Title:    "Confirmação de Recebimento",
        Content:  "Recebi seu email sobre {{subject}}. Vou analisar e retorno em breve.",
    },
    {
        Shortcut: "/ooo",
        Title:    "Fora do Escritório",
        Content: `Olá,

Obrigado pelo seu email. Estou fora do escritório até [DATA] com acesso limitado ao email.

Para assuntos urgentes, por favor entre em contato com [NOME] em [EMAIL].

Retornarei seu email assim que possível.

Atenciosamente,
{{my_name}}`,
    },
    {
        Shortcut: "/intro",
        Title:    "Introdução",
        Content: `Olá {{from_name}},

Prazer em conhecê-lo! Meu nome é {{my_name}}.

[BREVE DESCRIÇÃO]

Seria ótimo conectarmos. Você teria disponibilidade para uma breve conversa?

Atenciosamente,
{{my_name}}`,
    },
}
```

## Desktop UI

```svelte
<!-- SnippetPicker.svelte -->
<script>
  import { createEventDispatcher } from 'svelte';
  import { SearchSnippets, GetFrequentSnippets } from '../wailsjs/go/desktop/App';

  export let emailContext = null;

  const dispatch = createEventDispatcher();

  let query = '';
  let snippets = [];
  let selectedIndex = 0;

  async function search() {
    if (query.length < 1) {
      snippets = await GetFrequentSnippets(10);
    } else {
      snippets = await SearchSnippets(query);
    }
    selectedIndex = 0;
  }

  function selectSnippet(snippet) {
    dispatch('select', { snippet, context: emailContext });
  }

  function handleKeydown(e) {
    switch (e.key) {
      case 'ArrowDown':
        selectedIndex = Math.min(selectedIndex + 1, snippets.length - 1);
        e.preventDefault();
        break;
      case 'ArrowUp':
        selectedIndex = Math.max(selectedIndex - 1, 0);
        e.preventDefault();
        break;
      case 'Enter':
        if (snippets[selectedIndex]) {
          selectSnippet(snippets[selectedIndex]);
        }
        e.preventDefault();
        break;
      case 'Escape':
        dispatch('close');
        break;
    }
  }
</script>

<div class="snippet-picker" on:keydown={handleKeydown}>
  <input
    type="text"
    bind:value={query}
    on:input={search}
    placeholder="Buscar snippets... (ex: /meeting)"
    autofocus
  />

  <div class="snippet-list">
    {#each snippets as snippet, i}
      <button
        class="snippet-item"
        class:selected={i === selectedIndex}
        on:click={() => selectSnippet(snippet)}
      >
        <div class="shortcut">{snippet.shortcut}</div>
        <div class="title">{snippet.title}</div>
        <div class="preview">{snippet.content.slice(0, 50)}...</div>
      </button>
    {/each}
  </div>

  <div class="footer">
    <span>↑↓ navegar</span>
    <span>Enter selecionar</span>
    <span>Esc fechar</span>
  </div>
</div>

<style>
  .snippet-picker {
    background: var(--bg-primary);
    border: 1px solid var(--border-subtle);
    border-radius: var(--radius-lg);
    width: 400px;
    max-height: 400px;
    overflow: hidden;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
  }

  input {
    width: 100%;
    padding: var(--space-md);
    border: none;
    border-bottom: 1px solid var(--border-subtle);
    background: transparent;
    color: var(--text-primary);
    font-size: 1rem;
  }

  .snippet-list {
    max-height: 300px;
    overflow-y: auto;
  }

  .snippet-item {
    width: 100%;
    padding: var(--space-sm) var(--space-md);
    border: none;
    background: transparent;
    text-align: left;
    cursor: pointer;
    display: block;
  }

  .snippet-item:hover,
  .snippet-item.selected {
    background: var(--bg-secondary);
  }

  .shortcut {
    font-family: monospace;
    color: var(--accent-primary);
    font-size: 0.9em;
  }

  .title {
    font-weight: 500;
    color: var(--text-primary);
  }

  .preview {
    font-size: 0.85em;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .footer {
    display: flex;
    gap: var(--space-md);
    padding: var(--space-sm) var(--space-md);
    background: var(--bg-secondary);
    font-size: 0.8em;
    color: var(--text-tertiary);
  }
</style>
```

### Integração no Compose

```svelte
<!-- ComposeModal.svelte -->
<script>
  import SnippetPicker from './SnippetPicker.svelte';

  let showSnippetPicker = false;
  let bodyContent = '';

  function handleBodyKeydown(e) {
    // Detectar / no início da linha
    if (e.key === '/') {
      const textarea = e.target;
      const cursorPos = textarea.selectionStart;
      const textBefore = bodyContent.slice(0, cursorPos);

      // Se está no início ou após quebra de linha
      if (cursorPos === 0 || textBefore.endsWith('\n')) {
        showSnippetPicker = true;
        e.preventDefault();
      }
    }
  }

  function insertSnippet(event) {
    const { snippet, context } = event.detail;
    // ExpandSnippet já faz substituição de variáveis no backend
    ExpandSnippet(snippet.shortcut, context).then(expandedText => {
      // Inserir na posição atual do cursor
      bodyContent = bodyContent + expandedText;
      showSnippetPicker = false;
    });
  }
</script>

<textarea
  bind:value={bodyContent}
  on:keydown={handleBodyKeydown}
></textarea>

{#if showSnippetPicker}
  <div class="snippet-overlay">
    <SnippetPicker
      emailContext={getEmailContext()}
      on:select={insertSnippet}
      on:close={() => showSnippetPicker = false}
    />
  </div>
{/if}
```

## TUI Implementation

```go
// No compose, detectar /
func (m *ComposeModel) handleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "/":
        // Abrir snippet picker
        if m.isAtLineStart() {
            return m.showSnippetPicker()
        }
    }
    // ...
}

// Snippet picker como submenu
type SnippetPickerModel struct {
    snippets []Snippet
    filtered []Snippet
    query    string
    selected int
}

func (m SnippetPickerModel) View() string {
    var b strings.Builder

    b.WriteString("📝 Snippets\n")
    b.WriteString("─────────────\n")

    for i, s := range m.filtered {
        cursor := "  "
        if i == m.selected {
            cursor = "▸ "
        }
        b.WriteString(fmt.Sprintf("%s%s - %s\n", cursor, s.Shortcut, s.Title))
    }

    b.WriteString("\n[Enter] Inserir  [Esc] Fechar")

    return lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        Padding(1).
        Render(b.String())
}
```

## Gerenciador de Snippets

```svelte
<!-- SnippetManager.svelte -->
<script>
  import { GetAllSnippets, CreateSnippet, UpdateSnippet, DeleteSnippet } from '../wailsjs/go/desktop/App';

  let snippets = [];
  let editing = null;

  async function loadSnippets() {
    snippets = await GetAllSnippets();
  }

  async function save() {
    if (editing.id) {
      await UpdateSnippet(editing);
    } else {
      await CreateSnippet(editing);
    }
    editing = null;
    await loadSnippets();
  }
</script>

<div class="snippet-manager">
  <h2>Gerenciar Snippets</h2>

  <button on:click={() => editing = { shortcut: '/', title: '', content: '' }}>
    + Novo Snippet
  </button>

  <div class="snippet-list">
    {#each snippets as snippet}
      <div class="snippet-card">
        <div class="header">
          <code>{snippet.shortcut}</code>
          <span>{snippet.title}</span>
          <span class="uses">{snippet.useCount}x usado</span>
        </div>
        <pre class="content">{snippet.content}</pre>
        <div class="actions">
          <button on:click={() => editing = {...snippet}}>Editar</button>
          <button on:click={() => DeleteSnippet(snippet.id)}>Excluir</button>
        </div>
      </div>
    {/each}
  </div>

  {#if editing}
    <div class="edit-modal">
      <h3>{editing.id ? 'Editar' : 'Novo'} Snippet</h3>

      <label>
        Shortcut (ex: /meeting)
        <input bind:value={editing.shortcut} />
      </label>

      <label>
        Título
        <input bind:value={editing.title} />
      </label>

      <label>
        Conteúdo
        <textarea bind:value={editing.content} rows="10"></textarea>
      </label>

      <p class="help">
        Variáveis: {{from_name}}, {{from_email}}, {{subject}}, {{date}}, {{my_name}}
      </p>

      <div class="buttons">
        <button on:click={() => editing = null}>Cancelar</button>
        <button on:click={save}>Salvar</button>
      </div>
    </div>
  {/if}
</div>
```

## Critérios de Aceitação

- [ ] Digitar `/` no compose abre snippet picker
- [ ] Snippets podem ser criados/editados/deletados
- [ ] Variáveis são substituídas automaticamente
- [ ] Snippets mais usados aparecem primeiro
- [ ] Busca funciona por shortcut e título
- [ ] Snippets padrão disponíveis
- [ ] Funciona no Desktop e TUI
- [ ] Sync entre dispositivos (opcional)

---

*Inspirado em: Superhuman Snippets*
