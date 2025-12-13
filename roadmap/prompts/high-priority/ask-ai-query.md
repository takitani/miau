# Prompt: Ask AI (Consultar Emails com IA)

> Inspirado no Superhuman - Pergunte qualquer coisa sobre seus emails.

## Conceito

Fazer perguntas em linguagem natural sobre seus emails e receber respostas precisas com referências aos emails relevantes.

```
┌─ Ask AI ─────────────────────────────────────────────────────┐
│                                                              │
│  💬 "Quando foi a última reunião que agendei com o João?"   │
│                                                              │
│  ─────────────────────────────────────────────────────────   │
│                                                              │
│  🤖 A última reunião com João Silva foi agendada para       │
│     15/12/2024 às 14:00, discutida no email de 10/12:       │
│                                                              │
│     📧 "Re: Reunião de alinhamento Q1"                      │
│        De: joao.silva@empresa.com                           │
│        Data: 10/12/2024                                     │
│        [Abrir email]                                         │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## Tipos de Perguntas

### 1. Busca Temporal
- "Qual foi o último email do João?"
- "Quando recebi a fatura da AWS?"
- "Emails desta semana sobre o projeto X"

### 2. Resumo/Agregação
- "Quanto gastei em faturas este mês?"
- "Quantos emails não lidos tenho do trabalho?"
- "Resuma as discussões sobre o orçamento"

### 3. Extração de Informação
- "Qual é o número do pedido da Amazon?"
- "Qual o link do documento que o Pedro enviou?"
- "Quais são os deadlines mencionados?"

### 4. Análise
- "Quem mais me enviou emails sobre o projeto?"
- "Tenho algum email que precisa de resposta urgente?"
- "Há alguma reunião marcada para amanhã?"

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      User Query                              │
│              "último email do João"                          │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│                   Query Analyzer                             │
│  - Extrair entidades (João)                                 │
│  - Identificar tipo (busca temporal)                         │
│  - Gerar SQL/filtros                                        │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│                  Context Builder                             │
│  - Buscar emails relevantes (FTS5 + filtros)                │
│  - Limitar a N emails mais relevantes                        │
│  - Preparar contexto para LLM                                │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│                    LLM Processing                            │
│  - Contexto: emails relevantes                               │
│  - Query: pergunta do usuário                                │
│  - Output: resposta + referências                            │
└─────────────────┬───────────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────────┐
│                   Response Formatter                         │
│  - Formatar resposta                                        │
│  - Incluir links para emails                                 │
│  - Estruturar dados extraídos                                │
└─────────────────────────────────────────────────────────────┘
```

## Service Implementation

```go
// internal/services/ai_query.go

type AIQueryService struct {
    storage ports.EmailStorage
    llm     ports.LLMService
    fts     ports.FullTextSearch
}

type QueryResult struct {
    Answer         string
    ReferencedEmails []EmailReference
    Confidence     float64
    QueryType      string
}

type EmailReference struct {
    ID      int64
    Subject string
    From    string
    Date    time.Time
    Snippet string
}

// Ask processa uma pergunta sobre emails
func (s *AIQueryService) Ask(ctx context.Context, accountID int64, question string) (*QueryResult, error) {
    // 1. Analisar a pergunta
    analysis := s.analyzeQuery(question)

    // 2. Buscar emails relevantes
    emails, err := s.findRelevantEmails(ctx, accountID, analysis)
    if err != nil {
        return nil, err
    }

    if len(emails) == 0 {
        return &QueryResult{
            Answer:     "Não encontrei emails relevantes para sua pergunta.",
            Confidence: 1.0,
            QueryType:  analysis.Type,
        }, nil
    }

    // 3. Preparar contexto para LLM
    context := s.buildContext(emails)

    // 4. Processar com LLM
    answer, refs := s.processWithLLM(ctx, question, context, emails)

    return &QueryResult{
        Answer:           answer,
        ReferencedEmails: refs,
        Confidence:       s.calculateConfidence(answer, emails),
        QueryType:        analysis.Type,
    }, nil
}

type QueryAnalysis struct {
    Type        string   // search, summarize, extract, analyze
    Entities    []string // nomes, empresas, etc
    TimeRange   *TimeRange
    Keywords    []string
    Filters     map[string]string
}

func (s *AIQueryService) analyzeQuery(question string) *QueryAnalysis {
    analysis := &QueryAnalysis{
        Entities: make([]string, 0),
        Keywords: make([]string, 0),
        Filters:  make(map[string]string),
    }

    questionLower := strings.ToLower(question)

    // Detectar tipo de query
    if containsAny(questionLower, []string{"último", "última", "quando", "mais recente"}) {
        analysis.Type = "search_temporal"
    } else if containsAny(questionLower, []string{"quanto", "quantos", "total", "soma"}) {
        analysis.Type = "aggregate"
    } else if containsAny(questionLower, []string{"resuma", "resumo", "principais"}) {
        analysis.Type = "summarize"
    } else if containsAny(questionLower, []string{"qual", "onde", "link", "número"}) {
        analysis.Type = "extract"
    } else {
        analysis.Type = "search"
    }

    // Extrair time range
    analysis.TimeRange = s.extractTimeRange(question)

    // Extrair entidades (nomes próprios)
    analysis.Entities = s.extractEntities(question)

    // Extrair keywords
    analysis.Keywords = s.extractKeywords(question)

    return analysis
}

func (s *AIQueryService) findRelevantEmails(ctx context.Context, accountID int64, analysis *QueryAnalysis) ([]*Email, error) {
    var allEmails []*Email

    // 1. Busca por FTS5 com keywords
    if len(analysis.Keywords) > 0 {
        query := strings.Join(analysis.Keywords, " OR ")
        ftsResults, _ := s.fts.Search(ctx, accountID, query, 50)
        allEmails = append(allEmails, ftsResults...)
    }

    // 2. Busca por entidades (from_name, from_email)
    for _, entity := range analysis.Entities {
        entityResults, _ := s.storage.SearchByParticipant(ctx, accountID, entity, 20)
        allEmails = append(allEmails, entityResults...)
    }

    // 3. Aplicar filtro de tempo
    if analysis.TimeRange != nil {
        allEmails = filterByTimeRange(allEmails, analysis.TimeRange)
    }

    // 4. Ordenar por relevância e data
    sort.Slice(allEmails, func(i, j int) bool {
        // Mais recentes primeiro
        return allEmails[i].Date.After(allEmails[j].Date)
    })

    // 5. Remover duplicatas e limitar
    unique := removeDuplicates(allEmails)
    if len(unique) > 20 {
        unique = unique[:20]
    }

    return unique, nil
}

func (s *AIQueryService) buildContext(emails []*Email) string {
    var b strings.Builder

    b.WriteString("EMAILS RELEVANTES:\n\n")

    for i, email := range emails {
        b.WriteString(fmt.Sprintf("--- Email %d ---\n", i+1))
        b.WriteString(fmt.Sprintf("ID: %d\n", email.ID))
        b.WriteString(fmt.Sprintf("De: %s <%s>\n", email.FromName, email.FromEmail))
        b.WriteString(fmt.Sprintf("Para: %s\n", email.ToAddresses))
        b.WriteString(fmt.Sprintf("Data: %s\n", email.Date.Format("02/01/2006 15:04")))
        b.WriteString(fmt.Sprintf("Assunto: %s\n", email.Subject))
        b.WriteString(fmt.Sprintf("Corpo:\n%s\n\n", truncate(email.BodyText, 1000)))
    }

    return b.String()
}

func (s *AIQueryService) processWithLLM(ctx context.Context, question, context string, emails []*Email) (string, []EmailReference) {
    prompt := fmt.Sprintf(`Você é um assistente de email. Analise os emails fornecidos e responda à pergunta do usuário.

REGRAS:
1. Responda de forma direta e concisa
2. Cite o email específico que contém a informação (use o ID)
3. Se não encontrar a informação, diga claramente
4. Para valores monetários, seja preciso
5. Para datas, use formato brasileiro (DD/MM/AAAA)

%s

PERGUNTA: %s

Responda no formato:
RESPOSTA: [sua resposta]
EMAILS_CITADOS: [lista de IDs dos emails usados, separados por vírgula]`, context, question)

    response, err := s.llm.Complete(ctx, prompt)
    if err != nil {
        return "Desculpe, ocorreu um erro ao processar sua pergunta.", nil
    }

    // Parse response
    answer, citedIDs := s.parseResponse(response)

    // Build references
    refs := make([]EmailReference, 0)
    for _, id := range citedIDs {
        for _, email := range emails {
            if email.ID == id {
                refs = append(refs, EmailReference{
                    ID:      email.ID,
                    Subject: email.Subject,
                    From:    email.FromEmail,
                    Date:    email.Date,
                    Snippet: truncate(email.Snippet, 100),
                })
                break
            }
        }
    }

    return answer, refs
}

// Queries pré-definidas para ações rápidas
func (s *AIQueryService) GetQuickQueries() []QuickQuery {
    return []QuickQuery{
        {Label: "Emails não respondidos", Query: "Quais emails preciso responder?"},
        {Label: "Reuniões desta semana", Query: "Tenho reuniões marcadas para esta semana?"},
        {Label: "Faturas pendentes", Query: "Há faturas ou cobranças pendentes?"},
        {Label: "Últimos anexos", Query: "Quais foram os últimos anexos que recebi?"},
        {Label: "Follow-ups necessários", Query: "Há emails aguardando meu follow-up?"},
    }
}
```

## Desktop UI

```svelte
<!-- AskAI.svelte -->
<script>
  import { AskAI, GetQuickQueries } from '../wailsjs/go/desktop/App';
  import { goto } from '$app/navigation';

  let query = '';
  let result = null;
  let loading = false;
  let quickQueries = [];
  let history = [];

  onMount(async () => {
    quickQueries = await GetQuickQueries();
    // Carregar histórico do localStorage
    history = JSON.parse(localStorage.getItem('askAIHistory') || '[]');
  });

  async function ask() {
    if (!query.trim()) return;

    loading = true;
    result = null;

    try {
      result = await AskAI(query);

      // Salvar no histórico
      history = [{ query, timestamp: new Date() }, ...history.slice(0, 9)];
      localStorage.setItem('askAIHistory', JSON.stringify(history));
    } catch (error) {
      result = { answer: 'Erro ao processar pergunta: ' + error.message };
    }

    loading = false;
  }

  function openEmail(emailId) {
    goto(`/email/${emailId}`);
  }

  function useQuickQuery(q) {
    query = q;
    ask();
  }
</script>

<div class="ask-ai">
  <div class="input-section">
    <div class="input-wrapper">
      <span class="icon">💬</span>
      <input
        type="text"
        bind:value={query}
        on:keydown={(e) => e.key === 'Enter' && ask()}
        placeholder="Pergunte qualquer coisa sobre seus emails..."
        disabled={loading}
      />
      <button on:click={ask} disabled={loading || !query.trim()}>
        {loading ? '...' : 'Perguntar'}
      </button>
    </div>

    <!-- Quick queries -->
    <div class="quick-queries">
      {#each quickQueries as q}
        <button class="quick-query" on:click={() => useQuickQuery(q.query)}>
          {q.label}
        </button>
      {/each}
    </div>
  </div>

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
      <p>Analisando seus emails...</p>
    </div>
  {/if}

  {#if result}
    <div class="result">
      <div class="answer">
        <h3>Resposta</h3>
        <p>{result.answer}</p>
      </div>

      {#if result.referencedEmails?.length > 0}
        <div class="references">
          <h4>Emails referenciados:</h4>
          {#each result.referencedEmails as ref}
            <button class="email-ref" on:click={() => openEmail(ref.id)}>
              <div class="subject">{ref.subject}</div>
              <div class="meta">
                <span class="from">{ref.from}</span>
                <span class="date">{formatDate(ref.date)}</span>
              </div>
              <div class="snippet">{ref.snippet}</div>
            </button>
          {/each}
        </div>
      {/if}

      <div class="confidence">
        Confiança: {Math.round(result.confidence * 100)}%
      </div>
    </div>
  {/if}

  {#if !result && !loading && history.length > 0}
    <div class="history">
      <h4>Perguntas recentes</h4>
      {#each history as item}
        <button class="history-item" on:click={() => useQuickQuery(item.query)}>
          <span class="query">{item.query}</span>
          <span class="time">{formatRelativeTime(item.timestamp)}</span>
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .ask-ai {
    padding: var(--space-lg);
    max-width: 800px;
    margin: 0 auto;
  }

  .input-wrapper {
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    background: var(--bg-secondary);
    border-radius: var(--radius-lg);
    padding: var(--space-sm) var(--space-md);
    border: 2px solid var(--border-subtle);
  }

  .input-wrapper:focus-within {
    border-color: var(--accent-primary);
  }

  .icon {
    font-size: 1.5em;
  }

  input {
    flex: 1;
    border: none;
    background: transparent;
    color: var(--text-primary);
    font-size: 1.1rem;
    padding: var(--space-sm);
  }

  input:focus {
    outline: none;
  }

  button {
    padding: var(--space-sm) var(--space-md);
    background: var(--accent-primary);
    color: white;
    border: none;
    border-radius: var(--radius-md);
    cursor: pointer;
    font-weight: 500;
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .quick-queries {
    display: flex;
    gap: var(--space-sm);
    margin-top: var(--space-md);
    flex-wrap: wrap;
  }

  .quick-query {
    padding: var(--space-xs) var(--space-sm);
    background: var(--bg-tertiary);
    color: var(--text-secondary);
    font-size: 0.85em;
  }

  .quick-query:hover {
    background: var(--bg-secondary);
    color: var(--text-primary);
  }

  .loading {
    text-align: center;
    padding: var(--space-xl);
    color: var(--text-secondary);
  }

  .spinner {
    width: 40px;
    height: 40px;
    border: 3px solid var(--border-subtle);
    border-top-color: var(--accent-primary);
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin: 0 auto var(--space-md);
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .result {
    margin-top: var(--space-lg);
    background: var(--bg-secondary);
    border-radius: var(--radius-lg);
    padding: var(--space-lg);
  }

  .answer {
    margin-bottom: var(--space-lg);
  }

  .answer h3 {
    margin: 0 0 var(--space-sm);
    color: var(--text-secondary);
    font-size: 0.9em;
    text-transform: uppercase;
  }

  .answer p {
    font-size: 1.1rem;
    line-height: 1.6;
  }

  .references h4 {
    margin: 0 0 var(--space-sm);
    font-size: 0.9em;
    color: var(--text-secondary);
  }

  .email-ref {
    width: 100%;
    text-align: left;
    padding: var(--space-md);
    margin-bottom: var(--space-sm);
    background: var(--bg-primary);
    border-radius: var(--radius-md);
  }

  .email-ref:hover {
    background: var(--bg-tertiary);
  }

  .email-ref .subject {
    font-weight: 500;
    color: var(--text-primary);
  }

  .email-ref .meta {
    display: flex;
    gap: var(--space-md);
    font-size: 0.85em;
    color: var(--text-secondary);
    margin: var(--space-xs) 0;
  }

  .email-ref .snippet {
    font-size: 0.9em;
    color: var(--text-tertiary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .confidence {
    text-align: right;
    font-size: 0.85em;
    color: var(--text-tertiary);
    margin-top: var(--space-md);
  }

  .history {
    margin-top: var(--space-lg);
  }

  .history h4 {
    color: var(--text-secondary);
    font-size: 0.9em;
    margin-bottom: var(--space-sm);
  }

  .history-item {
    width: 100%;
    display: flex;
    justify-content: space-between;
    padding: var(--space-sm) var(--space-md);
    background: transparent;
    text-align: left;
    color: var(--text-secondary);
  }

  .history-item:hover {
    background: var(--bg-secondary);
    color: var(--text-primary);
  }

  .history-item .time {
    font-size: 0.85em;
    color: var(--text-tertiary);
  }
</style>
```

## TUI Implementation

```go
// Tecla '/' ou 'A' (Shift+A) para Ask AI
case "A":
    return m.showAskAI()

// Ask AI Model
type AskAIModel struct {
    input    textinput.Model
    result   *QueryResult
    loading  bool
    quick    []QuickQuery
    selected int
}

func (m AskAIModel) View() string {
    var b strings.Builder

    b.WriteString("💬 Ask AI\n")
    b.WriteString("─────────────────────────────────────\n\n")

    // Input
    b.WriteString(m.input.View() + "\n\n")

    // Quick queries
    b.WriteString("Sugestões:\n")
    for i, q := range m.quick {
        cursor := "  "
        if i == m.selected {
            cursor = "▸ "
        }
        b.WriteString(fmt.Sprintf("%s%s\n", cursor, q.Label))
    }

    // Loading
    if m.loading {
        b.WriteString("\n⏳ Analisando emails...\n")
    }

    // Result
    if m.result != nil {
        b.WriteString("\n" + strings.Repeat("─", 40) + "\n")
        b.WriteString("\n📝 Resposta:\n")
        b.WriteString(wordwrap.String(m.result.Answer, 60) + "\n")

        if len(m.result.ReferencedEmails) > 0 {
            b.WriteString("\n📧 Emails referenciados:\n")
            for _, ref := range m.result.ReferencedEmails {
                b.WriteString(fmt.Sprintf("  • %s\n", ref.Subject))
                b.WriteString(fmt.Sprintf("    %s - %s\n", ref.From, ref.Date.Format("02/01")))
            }
        }
    }

    b.WriteString("\n[Enter] Perguntar  [↑↓] Sugestões  [Esc] Fechar")

    return lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        Padding(1, 2).
        Width(70).
        Render(b.String())
}
```

## Integração com AI existente

```go
// No internal/tui/ai/panel.go, adicionar modo "ask"
const (
    ModeChat    AIMode = "chat"
    ModeCommand AIMode = "command"
    ModeAsk     AIMode = "ask"  // Novo modo
)

// O modo Ask usa o AIQueryService ao invés do chat genérico
func (m *Model) handleAskMode(input string) tea.Cmd {
    return func() tea.Msg {
        result, err := m.app.AIQuery().Ask(ctx, m.accountID, input)
        if err != nil {
            return errorMsg{err}
        }
        return askResultMsg{result}
    }
}
```

## Critérios de Aceitação

- [ ] Perguntas em linguagem natural funcionam
- [ ] Emails relevantes são encontrados via FTS5
- [ ] Respostas incluem referências aos emails
- [ ] Quick queries disponíveis
- [ ] Histórico de perguntas salvo
- [ ] Performance: resposta < 5 segundos
- [ ] Funciona offline (se LLM local disponível)
- [ ] UI Desktop e TUI implementadas

---

*Inspirado em: Superhuman Ask AI*
