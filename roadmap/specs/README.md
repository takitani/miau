# Roadmap Specs

Esta pasta contém especificações detalhadas para cada item do roadmap do miau. Cada arquivo `.md` serve como prompt para agentes AI implementarem a funcionalidade.

## Como Usar

1. Escolha um spec da lista abaixo
2. Passe o conteúdo do arquivo como prompt para o agente AI
3. O agente deve seguir a arquitetura existente (REGRA DE OURO - via Services)
4. Teste a implementação antes de considerar completa

## Estrutura dos Specs

Cada spec contém:
- **Overview**: Descrição do feature
- **User Stories**: Casos de uso
- **Technical Requirements**: Requisitos técnicos
- **Database Schema**: Mudanças no banco (se aplicável)
- **API/Service Interface**: Interfaces Go
- **UI/UX**: Mockups e comportamento
- **Testing**: Requisitos de teste
- **Acceptance Criteria**: Critérios de aceitação

## Specs por Categoria

### AI/ML Features (AI-*)
- [ai-email-summarization.md](ai-email-summarization.md) - AI-05
- [ai-auto-categorization.md](ai-auto-categorization.md) - AI-06
- [ai-smart-reply.md](ai-smart-reply.md) - AI-07
- [ai-sentiment-analysis.md](ai-sentiment-analysis.md) - AI-08
- [ai-action-items.md](ai-action-items.md) - AI-09
- [ai-email-prioritization.md](ai-email-prioritization.md) - AI-10
- [ai-smart-search.md](ai-smart-search.md) - AI-11
- [ai-translation.md](ai-translation.md) - AI-12
- [ai-grammar-check.md](ai-grammar-check.md) - AI-13
- [ai-phishing-detection.md](ai-phishing-detection.md) - AI-14
- [ai-meeting-notes.md](ai-meeting-notes.md) - AI-15
- [ai-multi-provider.md](ai-multi-provider.md) - AI-16

### Email Management (EM-*)
- [email-snooze.md](email-snooze.md) - EM-07
- [scheduled-send.md](scheduled-send.md) - EM-08
- [email-templates.md](email-templates.md) - EM-09
- [followup-reminders.md](followup-reminders.md) - EM-10
- [unsubscribe-manager.md](unsubscribe-manager.md) - EM-11
- [vip-inbox.md](vip-inbox.md) - EM-12
- [focus-mode.md](focus-mode.md) - EM-13
- [canned-responses.md](canned-responses.md) - EM-14
- [email-digest.md](email-digest.md) - EM-15
- [read-receipts.md](read-receipts.md) - EM-16
- [email-delegation.md](email-delegation.md) - EM-17

### Platform & Interfaces (PL-*)
- [web-interface.md](web-interface.md) - PL-03
- [cli-commands.md](cli-commands.md) - PL-04
- [api-server.md](api-server.md) - PL-05
- [mobile-pwa.md](mobile-pwa.md) - PL-06
- [browser-extension.md](browser-extension.md) - PL-07
- [launcher-integration.md](launcher-integration.md) - PL-08
- [automation-connector.md](automation-connector.md) - PL-09

### UX/UI Features (UX-*)
- [tui-mouse-support.md](tui-mouse-support.md) - UX-05
- [help-overlay.md](help-overlay.md) - UX-06
- [about-screen.md](about-screen.md) - UX-07
- [themes.md](themes.md) - UX-08
- [custom-shortcuts.md](custom-shortcuts.md) - UX-09
- [i18n.md](i18n.md) - UX-10
- [view-density.md](view-density.md) - UX-11
- [accessibility.md](accessibility.md) - UX-12
- [onboarding-tour.md](onboarding-tour.md) - UX-13
- [notification-prefs.md](notification-prefs.md) - UX-14

### Performance & Technical (TH-*)
- [imap-idle.md](imap-idle.md) - TH-05
- [body-indexing.md](body-indexing.md) - TH-06
- [background-sync.md](background-sync.md) - TH-07
- [connection-pooling.md](connection-pooling.md) - TH-08
- [virtual-scrolling.md](virtual-scrolling.md) - TH-09
- [lazy-loading.md](lazy-loading.md) - TH-10
- [retry-logic.md](retry-logic.md) - TH-11
- [offline-queue.md](offline-queue.md) - TH-12
- [delta-sync.md](delta-sync.md) - TH-13
- [attachment-caching.md](attachment-caching.md) - TH-14

### Security & Privacy (SC-*)
- [pgp-encryption.md](pgp-encryption.md) - SC-04
- [smime-support.md](smime-support.md) - SC-05
- [phishing-detection.md](phishing-detection.md) - SC-06
- [link-safety.md](link-safety.md) - SC-07
- [spf-dkim-display.md](spf-dkim-display.md) - SC-08
- [audit-logs.md](audit-logs.md) - SC-09
- [data-export.md](data-export.md) - SC-10
- [2fa-app.md](2fa-app.md) - SC-11

### Integrations (IN-*)
- [slack-integration.md](slack-integration.md) - IN-05
- [todoist-integration.md](todoist-integration.md) - IN-06
- [notion-integration.md](notion-integration.md) - IN-07
- [discord-integration.md](discord-integration.md) - IN-08
- [telegram-bot.md](telegram-bot.md) - IN-09
- [linear-integration.md](linear-integration.md) - IN-10
- [crm-integration.md](crm-integration.md) - IN-11

## Priority Guide

| Priority | Description | Implementation Order |
|----------|-------------|---------------------|
| High | Core functionality, user-requested | Implement first |
| Medium | Productivity enhancements | After high priority |
| Low | Nice-to-have, future | When time permits |

## Architecture Reminder (REGRA DE OURO)

**NUNCA** implemente lógica de negócio diretamente no TUI ou Desktop:

```
❌ PROIBIDO:
   TUI → imap.Client.FetchEmail()
   Desktop → storage.GetEmails()

✅ CORRETO:
   TUI → app.Email().GetEmail(id)
   Desktop → app.Email().GetEmail(id)
```

Toda lógica deve estar em `internal/services/` e ser acessada via `internal/app/`.
