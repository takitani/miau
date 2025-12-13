# Análise Competitiva - Email Clients

> Features de concorrentes que podemos aprender e implementar no miau.

*Pesquisado: 2025-12-12*

## Concorrentes Analisados

### Terminal/TUI Clients
- [NeoMutt](https://neomutt.org/) - Fork do Mutt com mais features
- [aerc](https://aerc-mail.org/) - TUI moderno em Go
- [himalaya](https://github.com/pimalaya/himalaya) - CLI em Rust
- [meli](https://meli-email.org/) - TUI em Rust

### Desktop Premium
- [Superhuman](https://superhuman.com/) - $30/mês, foco em velocidade
- [HEY](https://hey.com/) - $99/ano, da 37signals
- [Mailspring](https://getmailspring.com/) - Open source, Electron

### Privacy-Focused
- [Proton Mail](https://proton.me/) - E2E encryption
- [Thunderbird](https://www.thunderbird.net/) - Open source, OpenPGP
- [Edison Mail](https://mail.edison.tech/) - AI local (Llama)

---

## Features para Copiar/Melhorar

### De Superhuman ($30/mês)

| Feature | Status miau | Prioridade | Notas |
|---------|-------------|------------|-------|
| **Split Inbox** | ❌ | 🔴 Alta | Separar emails por categoria automaticamente |
| **Predictive Typing** | ❌ | 🟡 Média | Completar frases enquanto digita |
| **Instant Reply** | ⏳ Parcial | 🔴 Alta | AI sugere respostas rápidas |
| **Ask AI** | ❌ | 🔴 Alta | Perguntar coisas sobre seus emails |
| **Team Comments** | ❌ | 🟢 Baixa | Comentários internos em emails |
| **Follow-up Reminders** | ❌ | 🔴 Alta | Lembrar de responder |
| **Snippets/Templates** | ❌ | 🔴 Alta | Trechos reutilizáveis |
| **Send Later** | ⏳ Backend | 🔴 Alta | Agendar envio |
| **Undo Send** | ✅ Done | - | Já temos |

**Insight Superhuman:** O grande diferencial é VELOCIDADE. Emails carregam instantaneamente, ações são imediatas. Precisamos otimizar performance.

---

### De HEY (37signals) ($99/ano)

| Feature | Status miau | Prioridade | Notas |
|---------|-------------|------------|-------|
| **The Screener** | ❌ | 🔴 Alta | Aprovar remetentes antes de entrar no inbox |
| **Imbox** (não Inbox) | ❌ | 🟡 Média | Apenas emails importantes |
| **The Feed** | ❌ | 🟡 Média | Newsletters separadas |
| **Paper Trail** | ❌ | 🟡 Média | Recibos e transacionais |
| **Spy Tracker Blocking** | ❌ | 🔴 Alta | Bloquear pixels de tracking |
| **Merge Emails** | ❌ | 🟢 Baixa | Combinar threads relacionadas |
| **Workflows/Stages** | ❌ | 🟡 Média | Kanban para emails |
| **Publish to Web** | ❌ | 🟢 Baixa | Publicar email como webpage |
| **Selective Notifications** | ❌ | 🔴 Alta | Push só para VIPs |

**Insight HEY:** Filosofia de "você controla quem te manda email". O Screener é genial.

---

### De Mailspring (Open Source)

| Feature | Status miau | Prioridade | Notas |
|---------|-------------|------------|-------|
| **Link Tracking** | ❌ | 🟢 Baixa | Saber quando link foi clicado |
| **Read Receipts** | ❌ | 🟢 Baixa | Saber quando email foi lido |
| **Contact Profiles** | ❌ | 🟡 Média | Enriquecimento de contatos |
| **Company Info** | ❌ | 🟡 Média | Info da empresa do contato |
| **Mailbox Analytics** | ✅ Done | - | Já temos |
| **Translation** | ❌ | 🟡 Média | Traduzir emails |
| **Mail Rules** | ❌ | 🔴 Alta | Regras automáticas |
| **Unified Inbox** | ❌ | 🟡 Média | Todas contas em um inbox |

---

### De aerc (TUI em Go)

| Feature | Status miau | Prioridade | Notas |
|---------|-------------|------------|-------|
| **Tab Support** | ❌ | 🟡 Média | Múltiplas views em tabs |
| **Embedded Terminal** | ❌ | 🟢 Baixa | Terminal dentro do client |
| **Pipe to Command** | ❌ | 🟡 Média | Pipe email para comando |
| **Notmuch Integration** | ❌ | 🟢 Baixa | Integração com notmuch |
| **HTML Filter** | ✅ Done | - | Já temos |
| **PGP Support** | ❌ | 🟡 Média | Criptografia |

**Insight aerc:** Simplicidade de config e extensibilidade via comandos shell.

---

### De Proton Mail (Privacy)

| Feature | Status miau | Prioridade | Notas |
|---------|-------------|------------|-------|
| **Local AI Processing** | ⏳ Parcial | 🔴 Alta | AI sem enviar dados |
| **E2E Encryption** | ❌ | 🟡 Média | PGP nativo |
| **Zero-Access** | ✅ Done | - | Local-first |
| **Quantum-Safe** | ❌ | 🟢 Baixa | Criptografia pós-quântica |

---

## Top 15 Features para Implementar

Baseado na análise, estas são as features mais impactantes que NÃO temos:

| # | Feature | De | Impacto | Esforço |
|---|---------|----|---------|---------|
| 1 | **The Screener** | HEY | Alto | Médio |
| 2 | **Split Inbox** | Superhuman | Alto | Alto |
| 3 | **Ask AI** (sobre seus emails) | Superhuman | Alto | Médio |
| 4 | **Snippets/Templates** | Superhuman | Alto | Baixo |
| 5 | **Mail Rules** | Mailspring | Alto | Alto |
| 6 | **Follow-up Reminders** | Superhuman | Alto | Médio |
| 7 | **Spy Tracker Blocking** | HEY | Médio | Baixo |
| 8 | **Instant Reply Suggestions** | Superhuman | Alto | Alto |
| 9 | **The Feed** (newsletters) | HEY | Médio | Médio |
| 10 | **Workflows/Stages** | HEY | Médio | Alto |
| 11 | **Contact Enrichment** | Mailspring | Médio | Médio |
| 12 | **Selective Notifications** | HEY | Alto | Baixo |
| 13 | **Translation** | Mailspring | Médio | Baixo |
| 14 | **Predictive Typing** | Superhuman | Alto | Alto |
| 15 | **Tab Support** | aerc | Médio | Médio |

---

## Diferenciais do miau

Features que JÁ TEMOS e são diferenciais:

1. **100% Local** - Nenhum dado sai da máquina
2. **TUI + Desktop** - Duas interfaces, mesma lógica
3. **AI Integration** - Claude Code direto no app
4. **SQLite FTS5** - Busca full-text rápida
5. **Gmail API** - Bypass DLP
6. **Ports/Adapters** - Arquitetura modular
7. **Undo/Redo Infinito** - Histórico completo
8. **Calendar Integration** - Google Calendar
9. **Contact Autocomplete** - Via People API
10. **Multi-Account Ready** - Schema pronto

---

## Proposta de Novas Features

### 1. The Screener (inspirado HEY)
Primeira vez que alguém te envia email, vai para "Triagem". Você decide:
- ✅ Aceitar → vai para Inbox normal
- ❌ Rejeitar → nunca mais aparece
- 📰 Newsletter → vai para Feed
- 🧾 Recibo → vai para Paper Trail

### 2. Split Inbox (inspirado Superhuman)
Inbox dividido em seções automáticas:
- **VIP** - Pessoas importantes
- **Team** - Colegas de trabalho
- **News** - Newsletters
- **Other** - Resto

### 3. Ask AI (inspirado Superhuman)
Perguntas em linguagem natural sobre seus emails:
- "Quando foi o último email do João?"
- "Qual o valor total das faturas deste mês?"
- "Quais meetings tenho marcados via email?"

### 4. Spy Tracker Blocking (inspirado HEY)
Bloquear pixels de tracking:
- Detectar `<img>` de 1x1 pixel
- Bloquear domínios conhecidos
- Mostrar badge "🕵️ Tracking blocked"

### 5. Email Workflows (inspirado HEY)
Kanban para emails:
```
Triagem → Em análise → Aguardando → Resolvido
```

---

## Fontes

- [Superhuman Review 2024](https://francescod.medium.com/superhuman-email-review-a-2024-must-have-or-overpriced-hype-78735ea1ac09)
- [HEY.com Features](https://www.hey.com/)
- [Mailspring GitHub](https://github.com/Foundry376/Mailspring)
- [aerc Blog](https://blog.sergeantbiggs.net/posts/aerc-a-well-crafted-tui-for-email/)
- [Best AI Email Assistants 2025](https://zapier.com/blog/best-ai-email-assistant/)
- [Terminal Email Clients 2025](https://forwardemail.net/en/blog/open-source/terminal-email-clients)
- [Proton Mail Desktop](https://www.getmailbird.com/proton-mail-desktop-client-comparison/)

---

*Este documento deve ser atualizado regularmente com novos insights.*
