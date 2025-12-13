# miau Roadmap

> Central de planejamento e documentação de features do miau.

## Estrutura desta Pasta

```
roadmap/
├── README.md           # Este arquivo - visão geral
├── specs/              # Especificações detalhadas de cada feature
│   └── *.md            # 68 specs prontos para implementação
├── prompts/            # Prompts prontos para usar com AI
│   ├── high-priority/  # Features críticas
│   ├── medium-priority/# Features de produtividade
│   └── low-priority/   # Nice-to-have
├── ideas/              # Novas ideias e brainstorming
│   └── new-features.md # Features não especificadas ainda
└── status/             # Status atual do projeto
    └── progress.md     # Progresso por categoria
```

## Status Geral do Projeto

```
Core Features     [████████████████████████] 100%  ✅
Email Sending     [████████████████████████] 100%  ✅
TUI Interface     [██████████████████████░░]  90%
Desktop App       [██████████████████████░░]  92%
AI Integration    [████████████████░░░░░░░░]  65%
Modular Arch      [████████████████████████] 100%  ✅
Contacts System   [████████████████████████] 100%  ✅
Tasks System      [████████████████████░░░░]  80%
Advanced Features [████████████░░░░░░░░░░░░]  45%
```

## Features Implementadas (38 total)

### Core (100%)
- [x] Estrutura modular (Ports/Adapters)
- [x] SQLite storage + FTS5 full-text search
- [x] IMAP sync com detecção de deleções
- [x] OAuth2 para Gmail/Google Workspace
- [x] Multi-account support (database ready)

### Email (100%)
- [x] SMTP send com autenticação
- [x] Gmail API send (bypass DLP)
- [x] Archive/Delete Gmail-style
- [x] Bounce detection
- [x] Email threading (Gmail API)
- [x] Attachments (view, download, save)

### TUI (90%)
- [x] Inbox com navegação vim-style
- [x] Folder navigation
- [x] Email viewer (HTML in browser)
- [x] Compose modal
- [x] AI panel integrado
- [x] Settings menu
- [x] Debug panel
- [x] Image preview (i key)
- [ ] Mouse support
- [ ] Help overlay (?)

### Desktop App (92%)
- [x] 3-panel layout
- [x] Thread view com timeline
- [x] Multi-select + batch operations
- [x] Contact autocomplete
- [x] Settings modal completo
- [x] Analytics dashboard
- [x] Undo/Redo infinito
- [x] Calendar widget
- [x] Tasks widget
- [x] Theme toggle (light/dark/auto)
- [ ] About screen
- [ ] Onboarding tour

### AI (65%)
- [x] AI panel integration
- [x] Quick commands (/dr, /resume, /action)
- [x] Batch operations via AI
- [x] Draft generation
- [ ] Email summarization
- [ ] Auto-categorization
- [ ] Smart reply suggestions

### Integrations (80%)
- [x] Google People API (contacts)
- [x] Gmail API (send)
- [x] Google Calendar
- [x] Basecamp plugin
- [ ] Slack
- [ ] Todoist

## Próximas Prioridades

### Sprint Atual (Q1 2025)

| ID | Feature | Priority | Spec | Prompt |
|----|---------|----------|------|--------|
| AC-01 | Multi-Account Runtime | 🔴 High | [spec](specs/multi-account-support.md) | [prompt](prompts/high-priority/multi-account.md) |
| AI-05 | Email Summarization | 🔴 High | [spec](specs/ai-email-summarization.md) | [prompt](prompts/high-priority/ai-summarization.md) |
| UX-05 | Mouse Support (TUI) | 🔴 High | [spec](specs/tui-mouse-support.md) | [prompt](prompts/high-priority/mouse-support.md) |
| UX-06 | Help Overlay | 🔴 High | [spec](specs/help-overlay.md) | [prompt](prompts/high-priority/help-overlay.md) |
| TH-05 | IMAP IDLE (Push) | 🔴 High | [spec](specs/imap-idle.md) | [prompt](prompts/high-priority/imap-idle.md) |

### Features Inspiradas em Concorrentes

Baseado na [análise competitiva](ideas/competitive-analysis.md), estas features foram identificadas como alta prioridade:

| Feature | Inspiração | Impacto | Prompt |
|---------|------------|---------|--------|
| The Screener | HEY.com | Alto | [prompt](prompts/high-priority/the-screener.md) |
| Split Inbox | Superhuman | Alto | [prompt](prompts/high-priority/split-inbox.md) |
| Ask AI | Superhuman | Alto | [prompt](prompts/high-priority/ask-ai-query.md) |
| Snippets/Templates | Superhuman | Alto | [prompt](prompts/high-priority/snippets-templates.md) |
| Spy Tracker Blocking | HEY.com | Medio | [prompt](prompts/high-priority/spy-tracker-blocking.md) |
| Mail Rules | Mailspring | Alto | [prompt](prompts/high-priority/mail-rules.md) |

### Backlog por Categoria

| Categoria | Total | Done | Pending | Progress |
|-----------|-------|------|---------|----------|
| AI/ML | 16 | 4 | 12 | 25% |
| Email Management | 17 | 6 | 11 | 35% |
| Platform | 9 | 2 | 7 | 22% |
| UX/UI | 14 | 4 | 10 | 29% |
| Performance | 14 | 4 | 10 | 29% |
| Security | 11 | 3 | 8 | 27% |
| Integrations | 11 | 4 | 7 | 36% |

## Como Usar os Prompts

### Para implementar uma feature:

1. Escolha uma feature do roadmap
2. Leia o spec em `specs/<feature>.md`
3. Use o prompt em `prompts/<priority>/<feature>.md`
4. Passe o prompt para o Claude Code

### Exemplo:

```bash
# No Claude Code, cole o conteúdo do prompt:
cat roadmap/prompts/high-priority/email-snooze.md
```

## Arquitetura (REGRA DE OURO)

**NUNCA** implemente lógica de negócio diretamente no TUI ou Desktop:

```
┌─────────────┐     ┌─────────────┐
│     TUI     │     │   Desktop   │
└──────┬──────┘     └──────┬──────┘
       │                   │
       └───────┬───────────┘
               │
       ┌───────▼───────┐
       │  Application  │  ← ÚNICO PONTO DE ENTRADA
       └───────┬───────┘
               │
       ┌───────▼───────┐
       │   Services    │  ← TODA LÓGICA AQUI
       └───────┬───────┘
               │
    ┌──────────┼──────────┐
    │          │          │
┌───▼───┐  ┌───▼───┐  ┌───▼───┐
│ IMAP  │  │Storage│  │ SMTP  │
└───────┘  └───────┘  └───────┘
```

## Links Úteis

- [docs/ROADMAP.md](../docs/ROADMAP.md) - Roadmap detalhado com datas
- [docs/IDEAS.md](../docs/IDEAS.md) - Ideias e brainstorming
- [docs/architecture.md](../docs/architecture.md) - Arquitetura do sistema
- [docs/database.md](../docs/database.md) - Schema do banco de dados
- [CLAUDE.md](../CLAUDE.md) - Instruções para AI assistants

---

*Última atualização: 2025-12-12*
