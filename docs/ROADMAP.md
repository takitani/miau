# miau Roadmap

Roadmap de desenvolvimento do miau com status visual de progresso.

> Para detalhes de cada feature, veja [IDEAS.md](IDEAS.md)
> Para specs de implementação, veja [roadmap/specs/](../roadmap/specs/)

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
| ✅ Quick Commands (/dr, /resume, /action) | 2024-12-07 | merged |
| ✅ Plugin Architecture + Basecamp | 2024-12-07 | merged |
| ✅ Calendar Integration (Google Calendar) | 2024-12-08 | merged |
| ✅ Analytics Dashboard (TUI + Desktop) | 2024-12-08 | merged |
| ✅ Undo/Redo System (infinite) | 2024-12-08 | merged |

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

## 📋 Backlog Completo

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
│         └─ Spec: roadmap/specs/tui-mouse-support.md             │
│                                                                 │
│  3. [ ] Help Overlay                                            │
│         └─ Tecla ? abre painel com todos os atalhos             │
│         └─ Tips & tricks section                                │
│         └─ Spec: roadmap/specs/help-overlay.md                  │
│                                                                 │
│  4. [ ] About Screen                                            │
│         └─ Info do autor, LinkedIn, GitHub, Exato               │
│         └─ Versão, créditos, licença                            │
│         └─ Spec: roadmap/specs/about-screen.md                  │
│                                                                 │
│  5. [x] Quick Commands (/dr, /resume, /action) ✅               │
│         └─ Comandos rápidos estilo Slack                        │
│         └─ Implementado!                                        │
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

---

## 🤖 AI/ML Features

### Implemented ✅
| ID | Feature | Status | Spec |
|----|---------|--------|------|
| AI-01 | AI Panel Integration | ✅ Done | - |
| AI-02 | Quick Commands (/dr, /resume) | ✅ Done | - |
| AI-03 | Batch Operations via AI | ✅ Done | - |
| AI-04 | Draft Generation | ✅ Done | - |

### Backlog
| ID | Feature | Priority | Spec |
|----|---------|----------|------|
| AI-05 | AI Email Summarization | 🔴 High | [spec](../roadmap/specs/ai-email-summarization.md) |
| AI-06 | AI Auto-Categorization | 🔴 High | [spec](../roadmap/specs/ai-auto-categorization.md) |
| AI-07 | AI Smart Reply | 🔴 High | [spec](../roadmap/specs/ai-smart-reply.md) |
| AI-08 | AI Sentiment Analysis | 🟡 Medium | [spec](../roadmap/specs/ai-sentiment-analysis.md) |
| AI-09 | AI Action Items Extraction | 🟡 Medium | [spec](../roadmap/specs/ai-action-items.md) |
| AI-10 | AI Email Prioritization | 🟡 Medium | [spec](../roadmap/specs/ai-email-prioritization.md) |
| AI-11 | AI Smart Search (NLP) | 🟡 Medium | [spec](../roadmap/specs/ai-smart-search.md) |
| AI-12 | AI Translation | 🟡 Medium | [spec](../roadmap/specs/ai-translation.md) |
| AI-13 | AI Grammar Check | 🟢 Low | [spec](../roadmap/specs/ai-grammar-check.md) |
| AI-14 | AI Phishing Detection | 🟢 Low | [spec](../roadmap/specs/ai-phishing-detection.md) |
| AI-15 | AI Meeting Notes Extraction | 🟢 Low | [spec](../roadmap/specs/ai-meeting-notes.md) |
| AI-16 | Multi-AI Provider Support | 🟢 Low | [spec](../roadmap/specs/ai-multi-provider.md) |

**[autogenerated]** Items AI-05 to AI-16 were auto-generated based on product analysis.

---

## 📧 Email Management Features

### Implemented ✅
| ID | Feature | Status | Spec |
|----|---------|--------|------|
| EM-01 | IMAP Sync | ✅ Done | - |
| EM-02 | SMTP/Gmail API Send | ✅ Done | - |
| EM-03 | Archive/Delete/Star | ✅ Done | - |
| EM-04 | Thread Grouping | ✅ Done | - |
| EM-05 | FTS5 Search | ✅ Done | - |
| EM-06 | Bounce Detection | ✅ Done | - |

### Backlog
| ID | Feature | Priority | Spec |
|----|---------|----------|------|
| EM-07 | Email Snooze | 🔴 High | [spec](../roadmap/specs/email-snooze.md) |
| EM-08 | Scheduled Send | 🔴 High | [spec](../roadmap/specs/scheduled-send.md) |
| EM-09 | Email Templates | 🔴 High | [spec](../roadmap/specs/email-templates.md) |
| EM-10 | Follow-up Reminders | 🟡 Medium | [spec](../roadmap/specs/followup-reminders.md) |
| EM-11 | Unsubscribe Manager | 🟡 Medium | [spec](../roadmap/specs/unsubscribe-manager.md) |
| EM-12 | VIP Inbox | 🟡 Medium | [spec](../roadmap/specs/vip-inbox.md) |
| EM-13 | Focus Mode | 🟡 Medium | [spec](../roadmap/specs/focus-mode.md) |
| EM-14 | Canned Responses | 🟡 Medium | [spec](../roadmap/specs/canned-responses.md) |
| EM-15 | Email Digest (Newsletter Summary) | 🟢 Low | [spec](../roadmap/specs/email-digest.md) |
| EM-16 | Read Receipts (opt-in) | 🟢 Low | [spec](../roadmap/specs/read-receipts.md) |
| EM-17 | Email Delegation (Team) | 🟢 Low | [spec](../roadmap/specs/email-delegation.md) |

**[autogenerated]** Items EM-07 to EM-17 were auto-generated based on product analysis.

---

## 🖥️ Platform & Interfaces

### Implemented ✅
| ID | Feature | Status | Spec |
|----|---------|--------|------|
| PL-01 | TUI (Bubble Tea) | ✅ Done | - |
| PL-02 | Desktop App (Wails + Svelte) | ✅ 92% | - |

### Backlog
| ID | Feature | Priority | Spec |
|----|---------|----------|------|
| PL-03 | Web Interface (HTMX) | 🔴 High | [spec](../roadmap/specs/web-interface.md) |
| PL-04 | CLI Commands (miau ls, send) | 🟡 Medium | [spec](../roadmap/specs/cli-commands.md) |
| PL-05 | REST API Server | 🟡 Medium | [spec](../roadmap/specs/api-server.md) |
| PL-06 | Mobile PWA | 🟡 Medium | [spec](../roadmap/specs/mobile-pwa.md) |
| PL-07 | Browser Extension | 🟢 Low | [spec](../roadmap/specs/browser-extension.md) |
| PL-08 | Raycast/Alfred Integration | 🟢 Low | [spec](../roadmap/specs/launcher-integration.md) |
| PL-09 | Zapier/n8n Connector | 🟢 Low | [spec](../roadmap/specs/automation-connector.md) |

**[autogenerated]** Items PL-03 to PL-09 were auto-generated based on product analysis.

---

## 🎨 UX/UI Features

### Implemented ✅
| ID | Feature | Status | Spec |
|----|---------|--------|------|
| UX-01 | Image Preview (TUI) | ✅ Done | - |
| UX-02 | Settings Modal | ✅ Done | - |
| UX-03 | Analytics Dashboard | ✅ Done | - |
| UX-04 | Undo/Redo System | ✅ Done | - |

### Backlog
| ID | Feature | Priority | Spec |
|----|---------|----------|------|
| UX-05 | Mouse Support (TUI) | 🔴 High | [spec](../roadmap/specs/tui-mouse-support.md) |
| UX-06 | Help Overlay (?) | 🔴 High | [spec](../roadmap/specs/help-overlay.md) |
| UX-07 | About Screen | 🔴 High | [spec](../roadmap/specs/about-screen.md) |
| UX-08 | Dark/Light Themes | 🟡 Medium | [spec](../roadmap/specs/themes.md) |
| UX-09 | Custom Keyboard Shortcuts | 🟡 Medium | [spec](../roadmap/specs/custom-shortcuts.md) |
| UX-10 | Multi-Language (i18n) | 🟡 Medium | [spec](../roadmap/specs/i18n.md) |
| UX-11 | Compact/Comfortable View | 🟢 Low | [spec](../roadmap/specs/view-density.md) |
| UX-12 | Accessibility (a11y) | 🟢 Low | [spec](../roadmap/specs/accessibility.md) |
| UX-13 | Onboarding Tour | 🟢 Low | [spec](../roadmap/specs/onboarding-tour.md) |
| UX-14 | Notification Preferences | 🟢 Low | [spec](../roadmap/specs/notification-prefs.md) |

**[autogenerated]** Items UX-05 to UX-14 were auto-generated based on product analysis.

---

## ⚡ Performance & Technical

### Implemented ✅
| ID | Feature | Status | Spec |
|----|---------|--------|------|
| TH-01 | SQLite Storage | ✅ Done | - |
| TH-02 | FTS5 Full-Text Search | ✅ Done | - |
| TH-03 | Ports/Adapters Architecture | ✅ Done | - |
| TH-04 | Event Bus | ✅ Done | - |

### Backlog (Tech Debt + Performance)
| ID | Feature | Priority | Spec |
|----|---------|----------|------|
| TH-05 | IMAP IDLE (Push) | 🔴 High | [spec](../roadmap/specs/imap-idle.md) |
| TH-06 | Email Body Indexing | 🔴 High | [spec](../roadmap/specs/body-indexing.md) |
| TH-07 | Background Sync Worker | 🔴 High | [spec](../roadmap/specs/background-sync.md) |
| TH-08 | Connection Pooling | 🟡 Medium | [spec](../roadmap/specs/connection-pooling.md) |
| TH-09 | Virtual Scrolling | 🟡 Medium | [spec](../roadmap/specs/virtual-scrolling.md) |
| TH-10 | Lazy Body Loading | 🟡 Medium | [spec](../roadmap/specs/lazy-loading.md) |
| TH-11 | Retry Logic & Error Recovery | 🟡 Medium | [spec](../roadmap/specs/retry-logic.md) |
| TH-12 | Offline Queue | 🟡 Medium | [spec](../roadmap/specs/offline-queue.md) |
| TH-13 | Delta Sync | 🟢 Low | [spec](../roadmap/specs/delta-sync.md) |
| TH-14 | Attachment Caching | 🟢 Low | [spec](../roadmap/specs/attachment-caching.md) |

**[autogenerated]** Items TH-05 to TH-14 were auto-generated based on product analysis.

---

## 👤 Account Management

### Backlog
| ID | Feature | Priority | Spec |
|----|---------|----------|------|
| AC-01 | **Multi-Account Support** | 🔴 High | [spec](../roadmap/specs/multi-account-support.md) |

**Status**: Architecture is 85% ready. Database schema, storage adapters, and services already support multiple accounts. Only runtime/UI layer needs implementation.

**Key Implementation Points**:
- [ ] `Application.SetCurrentAccount(email)` - switch between accounts at runtime
- [ ] TUI account selector (Ctrl+A shortcut)
- [ ] Desktop account dropdown in header
- [ ] CLI `--account <email>` flag
- [ ] Graceful IMAP disconnect/reconnect on switch

---

## 🔒 Security & Privacy

### Implemented ✅
| ID | Feature | Status | Spec |
|----|---------|--------|------|
| SC-01 | OAuth2 Authentication | ✅ Done | - |
| SC-02 | Local-First Storage | ✅ Done | - |
| SC-03 | Token Management | ✅ Done | - |

### Backlog
| ID | Feature | Priority | Spec |
|----|---------|----------|------|
| SC-04 | PGP Encryption | 🟡 Medium | [spec](../roadmap/specs/pgp-encryption.md) |
| SC-05 | S/MIME Support | 🟡 Medium | [spec](../roadmap/specs/smime-support.md) |
| SC-06 | Phishing Detection | 🟡 Medium | [spec](../roadmap/specs/phishing-detection.md) |
| SC-07 | Link Safety Check | 🟡 Medium | [spec](../roadmap/specs/link-safety.md) |
| SC-08 | SPF/DKIM Display | 🟢 Low | [spec](../roadmap/specs/spf-dkim-display.md) |
| SC-09 | Audit Logs | 🟢 Low | [spec](../roadmap/specs/audit-logs.md) |
| SC-10 | Data Export (GDPR) | 🟢 Low | [spec](../roadmap/specs/data-export.md) |
| SC-11 | 2FA for App | 🟢 Low | [spec](../roadmap/specs/2fa-app.md) |

**[autogenerated]** Items SC-04 to SC-11 were auto-generated based on product analysis.

---

## 🔌 Integrations

### Implemented ✅
| ID | Feature | Status | Spec |
|----|---------|--------|------|
| IN-01 | Google People API (Contacts) | ✅ Done | - |
| IN-02 | Gmail API (Send) | ✅ Done | - |
| IN-03 | Google Calendar | ✅ Done | - |
| IN-04 | Basecamp Plugin | ✅ Done | - |

### Backlog
| ID | Feature | Priority | Spec |
|----|---------|----------|------|
| IN-05 | Slack Integration | 🟡 Medium | [spec](../roadmap/specs/slack-integration.md) |
| IN-06 | Todoist Integration | 🟡 Medium | [spec](../roadmap/specs/todoist-integration.md) |
| IN-07 | Notion Integration | 🟢 Low | [spec](../roadmap/specs/notion-integration.md) |
| IN-08 | Discord Integration | 🟢 Low | [spec](../roadmap/specs/discord-integration.md) |
| IN-09 | Telegram Bot | 🟢 Low | [spec](../roadmap/specs/telegram-bot.md) |
| IN-10 | Linear Integration | 🟢 Low | [spec](../roadmap/specs/linear-integration.md) |
| IN-11 | CRM Integration (HubSpot) | 🟢 Low | [spec](../roadmap/specs/crm-integration.md) |

**[autogenerated]** Items IN-05 to IN-11 were auto-generated based on product analysis.

---

## 📊 Full Roadmap Summary

### By Priority

| Priority | Count | Categories |
|----------|-------|------------|
| 🔴 High | 13 | Core UX, Critical AI, Performance, Account Mgmt |
| 🟡 Medium | 24 | Productivity, Integrations, Security |
| 🟢 Low | 22 | Nice-to-have, Future |
| ✅ Done | 38 | Completed features |

### Phase Planning

#### Phase 1: Core Polish (Q1 2025)
- [ ] **Multi-Account Support** ⭐ (AC-01)
- [ ] Mouse Support (TUI)
- [ ] Help Overlay
- [ ] About Screen
- [ ] Email Snooze
- [ ] Scheduled Send
- [ ] AI Summarization
- [ ] IMAP IDLE

#### Phase 2: Productivity (Q2 2025)
- [ ] Email Templates
- [ ] AI Smart Reply
- [ ] AI Auto-Categorization
- [ ] Web Interface
- [ ] Background Sync
- [ ] Themes

#### Phase 3: Advanced (Q3 2025)
- [ ] Multi-AI Providers
- [ ] CLI Commands
- [ ] API Server
- [ ] PGP Encryption
- [ ] Slack Integration
- [ ] Mobile PWA

#### Phase 4: Enterprise (Q4 2025)
- [ ] Email Delegation
- [ ] Audit Logs
- [ ] CRM Integration
- [ ] Advanced Analytics
- [ ] Custom Plugins

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

## Legenda

| Símbolo | Significado |
|---------|-------------|
| ✅ | Concluído |
| 🔄 | Em desenvolvimento |
| [ ] | Pendente |
| 🔴 | Alta prioridade |
| 🟡 | Média prioridade |
| 🟢 | Baixa prioridade |
| [autogenerated] | Item gerado automaticamente |
| ████ | Progresso visual |

---

*Última atualização: 2025-12-11*
*Items [autogenerated] foram gerados por análise de produto*
