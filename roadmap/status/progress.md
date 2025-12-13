# Status do Projeto miau

> Visão detalhada do progresso de implementação.

*Atualizado: 2025-12-12*

## Resumo Executivo

| Métrica | Valor |
|---------|-------|
| Features Implementadas | 38 |
| Features Pendentes | 59 |
| Progress Total | ~39% |
| Commits (últimos 30 dias) | 45+ |
| Linhas de Código Go | ~25.000 |
| Linhas de Código Svelte | ~15.000 |

---

## Por Categoria

### Core Features (100% ✅)

| Feature | Status | Commit | Data |
|---------|--------|--------|------|
| Estrutura inicial | ✅ | `a041592` | 2024-11-21 |
| Setup wizard | ✅ | `3827828` | 2024-11-22 |
| OAuth2 Gmail | ✅ | `8288f3c` | 2024-11-23 |
| IMAP client | ✅ | `45db4f1` | 2024-11-24 |
| SQLite + FTS5 | ✅ | `f7ac66b` | 2024-11-26 |
| Ports/Adapters | ✅ | `033e6a6` | 2024-12-04 |
| Multi-account (DB) | ✅ | `df40aaa` | 2024-12-11 |

### Email Management

| Feature | Status | Notes |
|---------|--------|-------|
| IMAP sync | ✅ | Incremental sync |
| SMTP send | ✅ | PLAIN/LOGIN auth |
| Gmail API send | ✅ | Bypasses DLP |
| Archive/Delete | ✅ | Gmail-style |
| Bounce detection | ✅ | 5 min tracking |
| Threading | ✅ | Via Gmail API |
| Attachments | ✅ | View, download, save |
| Snooze | ⏳ | Backend ready |
| Scheduled send | ⏳ | Backend ready |
| Templates | ❌ | Not started |
| Follow-up reminders | ❌ | Not started |

### TUI Interface (90%)

| Feature | Status | Notes |
|---------|--------|-------|
| Inbox view | ✅ | Vim-style navigation |
| Folder panel | ✅ | Collapsible |
| Email viewer | ✅ | HTML in browser |
| Compose modal | ✅ | With signatures |
| AI panel | ✅ | Press 'a' |
| Search (fuzzy) | ✅ | FTS5 trigram |
| Settings menu | ✅ | Indexer controls |
| Debug panel | ✅ | --debug flag |
| Image preview | ✅ | Press 'i' |
| Auto-refresh | ✅ | Timer visual |
| Mouse support | ❌ | Not started |
| Help overlay | ❌ | Not started |

### Desktop App (92%)

| Component | Status | Lines |
|-----------|--------|-------|
| App.svelte | ✅ | Main container |
| FolderList.svelte | ✅ | Sidebar |
| EmailList.svelte | ✅ | With multi-select |
| EmailRow.svelte | ✅ | Row component |
| EmailViewer.svelte | ✅ | HTML viewer |
| ComposeModal.svelte | ✅ | With autocomplete |
| ThreadView.svelte | ✅ | Timeline view |
| ThreadTimeline.svelte | ✅ | Collapsible |
| SearchPanel.svelte | ✅ | Real-time |
| SettingsModal.svelte | ✅ | 5 tabs |
| CalendarWidget.svelte | ✅ | Google Calendar |
| TasksWidget.svelte | ✅ | Local tasks |
| AnalyticsPanel.svelte | ✅ | Charts |
| ContactAutocomplete.svelte | ✅ | People API |
| SelectionBar.svelte | ✅ | Batch actions |
| StatusBar.svelte | ✅ | Sync status |
| HelpOverlay.svelte | ✅ | Shortcuts |
| ThemeToggle.svelte | ✅ | Light/dark/auto |
| AccountSelector.svelte | ✅ | Multi-account |
| AddAccountModal.svelte | ✅ | OAuth flow |
| ModernSidebar.svelte | ✅ | Gmail-style |
| SmartSelectMenu.svelte | ✅ | Smart select |
| AIChat.svelte | ✅ | Summarization |
| AuthOverlay.svelte | ✅ | Token refresh |
| LayoutToggle.svelte | ✅ | 2/3 panel |
| DebugPanel.svelte | ✅ | Dev tools |
| AboutScreen | ❌ | Not started |
| OnboardingTour | ❌ | Not started |

### AI Features (65%)

| Feature | Status | Notes |
|---------|--------|-------|
| AI panel | ✅ | TUI integration |
| Quick commands | ✅ | /dr, /resume, /action |
| Batch operations | ✅ | Via AI |
| Draft generation | ✅ | AI writes drafts |
| Email summarization | ⏳ | Backend WIP |
| Auto-categorization | ❌ | Not started |
| Smart reply | ❌ | Not started |
| Sentiment analysis | ❌ | Not started |
| Action extraction | ❌ | Not started |

### Integrations (80%)

| Integration | Status | Notes |
|-------------|--------|-------|
| Google People API | ✅ | Contact sync |
| Gmail API | ✅ | Send emails |
| Google Calendar | ✅ | Events sync |
| Basecamp plugin | ✅ | To-dos |
| Slack | ❌ | Not started |
| Todoist | ❌ | Not started |
| Notion | ❌ | Not started |

### Services Implementados

```
internal/services/
├── ai.go              ✅ 23.5 KB
├── analytics.go       ✅  4.5 KB
├── attachment_port.go ✅  8.3 KB
├── attachments.go     ✅  6.2 KB
├── basecamp.go        ✅ 12.2 KB
├── batch.go           ✅  5.6 KB
├── calendar.go        ✅ 20.3 KB
├── contact.go         ✅ 10.0 KB
├── draft.go           ✅  3.5 KB
├── email.go           ✅  8.6 KB
├── eventbus.go        ✅  2.2 KB
├── notification.go    ✅  4.7 KB
├── operations.go      ✅  9.2 KB
├── plugin.go          ✅ 12.4 KB
├── plugin_registry.go ✅ 12.5 KB
├── quickcmd.go        ✅ 11.8 KB
├── schedule.go        ✅  5.8 KB
├── search.go          ✅ 10.3 KB
├── send.go            ✅  6.5 KB
├── snooze.go          ✅  7.4 KB
├── sync.go            ✅ 19.8 KB
├── task.go            ✅  7.0 KB
├── thread.go          ✅  9.5 KB
└── undo.go            ✅ 10.0 KB
```

---

## Tech Debt

| Issue | Priority | Status |
|-------|----------|--------|
| Body not indexed | 🔴 High | Backlog |
| No IMAP IDLE | 🔴 High | Backlog |
| No retry logic | 🟡 Medium | Backlog |
| Virtual scrolling | 🟡 Medium | Backlog |
| Connection pooling | 🟡 Medium | Backlog |
| Delta sync | 🟢 Low | Backlog |

---

## Próximos Passos Recomendados

1. **Implementar IMAP IDLE** - Push notifications para novos emails
2. **Multi-account runtime** - Trocar contas em tempo de execução
3. **Email summarization** - AI resume emails longos
4. **Mouse support (TUI)** - Click e scroll
5. **Help overlay** - Documentação in-app

---

## Métricas de Qualidade

| Métrica | Valor | Status |
|---------|-------|--------|
| Test coverage | ~60% | 🟡 Needs improvement |
| SonarQube issues | 12 | 🟢 Good |
| Go vet warnings | 0 | ✅ Clean |
| Security vulns | 0 | ✅ Clean |

---

*Este documento é atualizado automaticamente a cada release.*
