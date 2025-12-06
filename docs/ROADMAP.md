# miau Roadmap

Roadmap de desenvolvimento do miau com status visual de progresso.

> Para detalhes de cada feature, veja [IDEAS.md](IDEAS.md)

---

## Progresso Geral

```
Core Features     [████████████████████████] 100%
Email Sending     [████████████████████████] 100%
TUI Interface     [██████████████████████░░] 90%
Desktop App       [██████████████████████░░] 92%
AI Integration    [████████████████░░░░░░░░] 65%
Modular Arch      [████████████████████████] 100%
Contacts System   [████████████████████████] 100%
Tasks System      [████████████████████░░░░] 80%
Advanced Features [████████████░░░░░░░░░░░░] 45%
```

---

## ✅ Concluído

| Feature | Data | Commit |
|---------|------|--------|
| ✅ Estrutura inicial do projeto | 2024-11-21 | `a041592` |
| ✅ Setup wizard com auto-detecção | 2024-11-22 | `3827828` |
| ✅ OAuth2 para Gmail/Workspace | 2024-11-23 | `8288f3c` |
| ✅ Cliente IMAP + TUI inbox | 2024-11-24 | `45db4f1` |
| ✅ Melhorias de autenticação | 2024-11-25 | `9114f40` |
| ✅ SQLite storage + FTS5 | 2024-11-26 | `f7ac66b` |
| ✅ Painel de AI integrado | 2024-11-27 | `817633e` |
| ✅ Sync configurável + trigram | 2024-11-29 | `87d62ac` |
| ✅ Spinner + HTML viewer | 2024-12-01 | `a50db5a` |
| ✅ SMTP + composição + assinaturas | 2024-12-02 | `0266c7c` |
| ✅ Bounce detection | 2024-12-03 | `1de7fd0` |
| ✅ Gmail API send + boot otimizado | 2024-12-03 | `356cf65` |
| ✅ Archive/delete Gmail-style | 2024-12-04 | `de0d314` |
| ✅ Operações em lote via AI | 2024-12-04 | `de0d314` |
| ✅ Drafts via AI | 2024-12-04 | `de0d314` |
| ✅ Retenção permanente de dados | 2024-12-04 | `de0d314` |
| ✅ Menu de configurações | 2024-12-04 | merged |
| ✅ Documentação (arch + schema) | 2024-12-04 | merged |
| ✅ Image Preview no TUI | 2024-12-04 | `7243d38` |
| ✅ Fix delete/archive sync Gmail | 2024-12-04 | `fcb23e8` |
| ✅ Arquitetura Modular (Ports/Adapters) | 2024-12-04 | `033e6a6` |
| ✅ Auto-refresh com timer visual | 2024-12-04 | pending |
| ✅ Sync logs para contagem correta | 2024-12-04 | pending |
| ✅ Multi-select com batch operations | 2024-12-05 | `9d44fda` |
| ✅ Gmail thread sync (API) | 2024-12-05 | `00f9c14` |
| ✅ Contacts system + Google People API | 2024-12-05 | `1e3eca6` |
| ✅ Contact autocomplete no compose | 2024-12-06 | `6a0be8d` |
| ✅ Tasks system (desktop) | 2024-12-06 | pending |
| ✅ OtherContacts sync (Gmail auto-suggest) | 2024-12-06 | `6a0be8d` |
| ✅ SQLite busy_timeout fix | 2024-12-06 | pending |

---

## 🚧 Em Desenvolvimento

```
┌─────────────────────────────────────────────────────────────────┐
│  🔄 CURRENT SPRINT                                              │
├─────────────────────────────────────────────────────────────────┤
│  [ ] Resumo automático de emails via IA                         │
│  [ ] Categorização automática de emails                         │
│  [ ] Busca fuzzy nativa (tecla F)                               │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📋 Backlog (Fila de Prioridades)

### 🔴 Alta Prioridade — Próxima Release

```
┌─ QUEUE ─────────────────────────────────────────────────────────┐
│                                                                 │
│  1. [x] Multi-Select ✅                                         │
│         └─ Shift+Click, Ctrl+Click para selecionar              │
│         └─ Batch actions: archive, delete, mark read            │
│         └─ Implementado no Desktop                              │
│                                                                 │
│  2. [ ] Mouse Support (TUI)                                     │
│         └─ Click, scroll, double-click, context menu            │
│         └─ Bubble Tea: WithMouseCellMotion()                    │
│         └─ Ver: IDEAS.md#multi-select--mouse-support            │
│                                                                 │
│  3. [ ] Help Overlay                                            │
│         └─ Tecla ? abre painel com todos os atalhos             │
│         └─ Tips & tricks section                                │
│         └─ Ver: IDEAS.md#help-overlay                           │
│                                                                 │
│  4. [ ] About Screen                                            │
│         └─ Info do autor, LinkedIn, GitHub, Exato               │
│         └─ Versão, créditos, licença                            │
│         └─ Ver: IDEAS.md#about-screen                           │
│                                                                 │
│  5. [ ] Quick Commands (/dr, /resume, /action)                  │
│         └─ Comandos rápidos estilo Slack                        │
│         └─ Ver: IDEAS.md#quick-commands                         │
│                                                                 │
│  6. [x] Attachments ✅                                          │
│         └─ Listar, baixar, salvar, abrir anexos                 │
│         └─ Implementado no Desktop                              │
│                                                                 │
│  7. [x] Threading/Conversas ✅                                  │
│         └─ Gmail thread sync via API                            │
│         └─ Thread view com timeline colapsável                  │
│         └─ Implementado no Desktop                              │
│                                                                 │
│  8. [x] Contact Autocomplete ✅                                 │
│         └─ Sync Google People API                               │
│         └─ Search + autocomplete no compose                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 🟡 Média Prioridade

```
┌─ BACKLOG ───────────────────────────────────────────────────────┐
│                                                                 │
│  8.  [x] Image Preview no TUI ✅                                │
│          └─ chafa/viu para gráficos, ASCII art como fallback    │
│          └─ Tecla i no viewer, ←→ navega, s salva               │
│                                                                 │
│  9.  [x] Auto-refresh com timer visual ✅                       │
│          └─ Barra de progresso animada no footer                │
│          └─ Indicador de novos emails após sync                 │
│                                                                 │
│  10. [ ] Web Interface (Go + HTMX)                              │
│          └─ miau serve --port 8080                              │
│          └─ Arquitetura modular já suporta ✅                   │
│          └─ Ver: IDEAS.md#multi-platform-ui                     │
│                                                                 │
│  11. [ ] Offline Queue                                          │
│          └─ Fila de ações quando offline                        │
│          └─ Ver: IDEAS.md#offline-mode--sync                    │
│                                                                 │
│  12. [ ] Rules Engine                                           │
│          └─ Filtros automáticos YAML                            │
│          └─ Ver: IDEAS.md#smart-notifications--alerts           │
│                                                                 │
│  13. [x] Analytics Dashboard ✅                                 │
│          └─ Estatísticas de email (TUI e Desktop)               │
│          └─ Top senders, trends, response time                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 🟢 Baixa Prioridade — Futuro

```
┌─ ICEBOX ────────────────────────────────────────────────────────┐
│                                                                 │
│  14. [x] Desktop App (Wails + Svelte) ✅                        │
│          └─ Implementado! Layout 3 painéis                      │
│          └─ `make desktop-build && make desktop-run`            │
│                                                                 │
│  15. [ ] Calendar Integration                                   │
│          └─ ICS, accept/decline                                 │
│          └─ Ver: IDEAS.md#calendar-integration                  │
│                                                                 │
│  16. [ ] Plugin System                                          │
│          └─ CRM, Todoist, Slack integrations                    │
│          └─ Ver: IDEAS.md#plugin-system                         │
│                                                                 │
│  17. [ ] Encryption (PGP/S-MIME)                                │
│          └─ Ver: IDEAS.md#security--privacy                     │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🔧 Dívida Técnica

```
┌─ TECH DEBT ─────────────────────────────────────────────────────┐
│                                                                 │
│  [ ] Body content não indexado (só metadata sincronizado)       │
│  [ ] Sem IMAP IDLE (push notifications)                         │
│  [ ] Sem operações multi-folder                                 │
│  [ ] Recuperação de erros limitada                              │
│  [ ] Sem retry logic para syncs falhados                        │
│                                                                 │
│  PERFORMANCE:                                                   │
│  [ ] Virtual scrolling para mailboxes grandes                   │
│  [ ] Lazy loading de email bodies                               │
│  [ ] Connection pooling para IMAP                               │
│  [ ] Background sync worker                                     │
│  [ ] Delta sync (apenas mudanças)                               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📊 Timeline Visual

```
Nov 2024                              Dez 2024
   │                                     │
   ▼                                     ▼
   ┌──────────────────────────────────────────────────────────────┐
   │░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│
   │▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓                     │
   └──────────────────────────────────────────────────────────────┘
    21 22 23 24 25 26 27 28 29 30 01 02 03 04
    ▲  ▲  ▲  ▲  ▲  ▲  ▲     ▲     ▲  ▲  ▲  ▲
    │  │  │  │  │  │  │     │     │  │  │  └─ Archive/Batch/Drafts
    │  │  │  │  │  │  │     │     │  │  └──── Gmail API + Bounce
    │  │  │  │  │  │  │     │     │  └─────── SMTP + Compose
    │  │  │  │  │  │  │     │     └────────── HTML Viewer
    │  │  │  │  │  │  │     └──────────────── FTS5 Trigram
    │  │  │  │  │  │  └────────────────────── AI Panel
    │  │  │  │  │  └───────────────────────── SQLite
    │  │  │  │  └──────────────────────────── Auth
    │  │  │  └─────────────────────────────── IMAP + TUI
    │  │  └────────────────────────────────── OAuth2
    │  └───────────────────────────────────── Setup Wizard
    └──────────────────────────────────────── Project Init
```

---

## Legenda

| Símbolo | Significado |
|---------|-------------|
| ✅ | Concluído |
| 🔄 | Em desenvolvimento |
| [ ] | Pendente |
| 🔴 | Alta prioridade |
| 🟡 | Média prioridade |
| 🟢 | Baixa prioridade |
| ████ | Progresso visual |

---

*Última atualização: 2025-12-06*
