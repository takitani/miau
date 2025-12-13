# Prompt: AI Email Summarization

> Use este prompt com Claude Code para implementar resumo de emails via AI.

## Contexto

O miau é um email client local com TUI e Desktop (Wails + Svelte). Precisamos implementar resumo automático de emails usando AI (Claude).

## Objetivo

Implementar feature de resumo de emails que:
1. Gera resumo de um email individual
2. Gera resumo de uma thread inteira
3. Cache de resumos para não reprocessar
4. UI no Desktop e TUI

## Arquivos Relevantes

```
internal/services/ai.go          # Service de AI existente
internal/desktop/app.go          # Bindings desktop
internal/storage/repository.go   # Storage
cmd/miau-desktop/frontend/src/lib/components/EmailViewer.svelte
cmd/miau-desktop/frontend/src/lib/components/AIChat.svelte
```

## Spec Detalhado

Leia o spec completo em: `roadmap/specs/ai-email-summarization.md`

## Tasks

### 1. Backend - Service

Adicionar ao `internal/services/ai.go`:

```go
type SummaryResult struct {
    EmailID    int64    `json:"emailId"`
    Summary    string   `json:"summary"`
    KeyPoints  []string `json:"keyPoints"`
    Style      string   `json:"style"` // brief, detailed, tldr
    Cached     bool     `json:"cached"`
    CreatedAt  time.Time `json:"createdAt"`
}

func (s *AIService) SummarizeEmail(ctx context.Context, emailID int64, style string) (*SummaryResult, error)
func (s *AIService) SummarizeThread(ctx context.Context, threadID string, style string) (*SummaryResult, error)
```

### 2. Backend - Cache

Criar tabela para cache de resumos:

```sql
CREATE TABLE email_summaries (
    id INTEGER PRIMARY KEY,
    email_id INTEGER REFERENCES emails(id),
    thread_id TEXT,
    style TEXT NOT NULL,
    summary TEXT NOT NULL,
    key_points TEXT, -- JSON array
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(email_id, style)
);
```

### 3. Backend - Bindings

Adicionar ao `internal/desktop/app.go`:

```go
func (a *App) SummarizeEmail(emailID int64, style string) (*SummaryResult, error)
func (a *App) SummarizeThread(threadID string, style string) (*SummaryResult, error)
func (a *App) GetCachedSummary(emailID int64) (*SummaryResult, error)
func (a *App) InvalidateSummary(emailID int64) error
```

### 4. Frontend - EmailViewer

Adicionar botão de resumo no `EmailViewer.svelte`:

```svelte
<script>
  let summary = null;
  let summaryLoading = false;
  let summaryStyle = 'brief'; // brief, detailed, tldr

  async function generateSummary() {
    summaryLoading = true;
    summary = await SummarizeEmail(email.id, summaryStyle);
    summaryLoading = false;
  }
</script>

<!-- No toolbar -->
<button on:click={generateSummary} title="Resumir (AI)">
  {#if summaryLoading}
    <Spinner />
  {:else}
    <SummaryIcon />
  {/if}
</button>

<!-- Panel de resumo -->
{#if summary}
  <div class="summary-panel">
    <h4>Resumo ({summary.style})</h4>
    <p>{summary.summary}</p>
    {#if summary.keyPoints?.length}
      <ul>
        {#each summary.keyPoints as point}
          <li>{point}</li>
        {/each}
      </ul>
    {/if}
    <div class="summary-actions">
      <select bind:value={summaryStyle} on:change={generateSummary}>
        <option value="tldr">TL;DR</option>
        <option value="brief">Breve</option>
        <option value="detailed">Detalhado</option>
      </select>
      <button on:click={() => InvalidateSummary(email.id)}>Regenerar</button>
    </div>
  </div>
{/if}
```

### 5. Prompt para AI

O prompt para gerar resumos deve ser:

```
Resuma o email abaixo de forma {style}.

{style == 'tldr'}: Máximo 2 frases.
{style == 'brief'}: 3-5 frases com pontos principais.
{style == 'detailed'}: Resumo completo com todos os detalhes relevantes.

Extraia também os key points (máximo 5) se houver.

---
De: {from}
Para: {to}
Assunto: {subject}
Data: {date}

{body}
---

Responda em JSON:
{
  "summary": "...",
  "keyPoints": ["...", "..."]
}
```

## Critérios de Aceitação

- [ ] Botão de resumo no EmailViewer
- [ ] 3 estilos: TL;DR, Breve, Detalhado
- [ ] Cache funciona (não reprocessa)
- [ ] Pode regenerar resumo
- [ ] Funciona offline (mostra cache)
- [ ] Loading state visual
- [ ] Erro handling

## Importante

Siga a REGRA DE OURO:
- Toda lógica no service (`internal/services/ai.go`)
- Desktop chama via `app.AI().SummarizeEmail()`
- Nunca implemente lógica diretamente no frontend

---

*Prompt criado: 2025-12-12*
