# Novas Ideias para o miau

> Features inovadoras e melhorias técnicas ainda não especificadas.

## Novas Features de Produto

### NF-01: Smart Inbox Zero
**Prioridade:** Alta
**Complexidade:** Média

AI analisa inbox e sugere ações para chegar a zero emails não processados:
- Agrupa emails similares para ação em lote
- Sugere "archive all newsletters older than 7 days"
- Identifica emails que precisam de resposta urgente
- Mostra progresso: "15 emails → Inbox Zero"

```
┌─ Smart Inbox Zero ──────────────────────────────────────────┐
│ 📊 Current: 127 emails in inbox                             │
│                                                             │
│ Suggestions:                                                │
│ 1. Archive 45 newsletters (older than 7 days)        [y/n]  │
│ 2. Mark 23 notifications as read                     [y/n]  │
│ 3. Reply to 5 urgent emails from VIPs               [view]  │
│ 4. Unsubscribe from 12 low-value senders            [view]  │
│                                                             │
│ Estimated time to Inbox Zero: 15 minutes                    │
└─────────────────────────────────────────────────────────────┘
```

---

### NF-02: Email Health Score
**Prioridade:** Média
**Complexidade:** Baixa

Score de 0-100 indicando organização do email:
- Tempo médio de resposta
- Taxa de emails não lidos
- Inbox size vs archived
- Follow-ups pendentes

```
┌─ Email Health ──────────────────────────────────────────────┐
│                                                             │
│         ████████████████████░░░░░░░░░░                      │
│                    72/100                                   │
│                                                             │
│ ✅ Response time: 4.2h (good)                               │
│ ⚠️  Unread rate: 23% (could improve)                        │
│ ✅ Inbox size: 45 (healthy)                                 │
│ ❌ Pending follow-ups: 8 (needs attention)                  │
│                                                             │
│ Tip: Clear your 8 pending follow-ups to reach 85+          │
└─────────────────────────────────────────────────────────────┘
```

---

### NF-03: Email Snippets (Text Expander)
**Prioridade:** Alta
**Complexidade:** Baixa

Trechos de texto reutilizáveis com expansão automática:
- `;sig` → expande para assinatura
- `;addr` → expande para endereço
- `;meet` → expande para link de meeting
- Suporta variáveis: `{date}`, `{name}`, `{email}`

```yaml
# ~/.config/miau/snippets.yaml
snippets:
  sig: |
    Atenciosamente,
    André Takitani
  addr: |
    Rua Example, 123
    São Paulo, SP
  meet: |
    Link para reunião: https://meet.google.com/xxx-yyyy-zzz
    Horário: {date}
  followup: |
    Olá {name},

    Gostaria de fazer um follow-up sobre nosso último contato.
    Podemos agendar uma conversa?
```

---

### NF-04: Quiet Hours
**Prioridade:** Média
**Complexidade:** Baixa

Período sem notificações e sync pausado:
- Configurável por dia da semana
- Exceções para VIPs
- Auto-ativa em eventos de calendário

```yaml
quiet_hours:
  enabled: true
  schedule:
    weekdays: "22:00-08:00"
    weekends: "20:00-10:00"
  exceptions:
    - from: boss@company.com
    - subject_contains: "URGENT"
  calendar_integration: true  # Quiet during meetings
```

---

### NF-05: Email Workflows
**Prioridade:** Média
**Complexidade:** Alta

Automações locais tipo Zapier:
- Trigger: novo email matching condição
- Action: archive, label, forward, reply, etc.

```yaml
workflows:
  - name: "Auto-archive newsletters"
    trigger:
      from_contains: ["newsletter", "digest", "weekly"]
    actions:
      - label: "Newsletters"
      - archive_after: "7 days"

  - name: "VIP notifications"
    trigger:
      from: ["boss@company.com", "client@vip.com"]
    actions:
      - notify: true
      - label: "VIP"
      - star: true

  - name: "Invoice auto-organize"
    trigger:
      subject_contains: ["invoice", "fatura", "nota fiscal"]
      has_attachment: true
    actions:
      - label: "Finance"
      - save_attachments: "~/Documents/Invoices/{year}/{month}/"
```

---

### NF-06: Contact Enrichment
**Prioridade:** Baixa
**Complexidade:** Média

Enriquecer contatos com dados públicos:
- LinkedIn profile
- Company info
- Timezone
- Social links

```
┌─ Contact: John Smith ───────────────────────────────────────┐
│ john.smith@company.com                                      │
│                                                             │
│ 🏢 Company: TechCorp Inc (Senior Developer)                 │
│ 🌐 LinkedIn: linkedin.com/in/johnsmith                      │
│ 🕐 Timezone: PST (UTC-8) - Currently 2:30 PM                │
│ 📧 Last interaction: 3 days ago                             │
│ 📊 Response rate: Usually replies within 24h                │
│                                                             │
│ Notes: Prefers email over calls                             │
└─────────────────────────────────────────────────────────────┘
```

---

### NF-07: Keyboard Macro Recording
**Prioridade:** Baixa
**Complexidade:** Alta

Gravar sequências de teclas para replay:
- `Ctrl+Shift+R` inicia recording
- Salva como comando customizado
- Replay com atalho personalizado

```
Recording: archive_and_next
Keys: e → j → Enter
Saved! Use with: Ctrl+1
```

---

### NF-08: Email Deduplication
**Prioridade:** Média
**Complexidade:** Média

Detectar e gerenciar emails duplicados:
- Mesmo Message-ID em múltiplas pastas
- Emails similares (forwarded, replied)
- Merge ou delete duplicatas

---

### NF-09: Smart Meeting Scheduler
**Prioridade:** Alta
**Complexidade:** Alta

Integração profunda com calendário:
- Detecta pedidos de reunião em emails
- Sugere horários disponíveis
- Gera link de calendly/meet
- Reply automático com opções

```
┌─ Meeting Detected ──────────────────────────────────────────┐
│ John wants to schedule a meeting about "Project Review"     │
│                                                             │
│ Your available slots this week:                             │
│ [1] Tue 2:00 PM - 3:00 PM                                   │
│ [2] Wed 10:00 AM - 11:00 AM                                 │
│ [3] Thu 4:00 PM - 5:00 PM                                   │
│                                                             │
│ [r] Reply with options  [c] Create event  [i] Ignore        │
└─────────────────────────────────────────────────────────────┘
```

---

### NF-10: Thread Summary
**Prioridade:** Alta
**Complexidade:** Média

Resumo automático de threads longas:
- Mostra participantes
- Key points de cada mensagem
- Decisões tomadas
- Action items pendentes

```
┌─ Thread Summary: "Q4 Budget Discussion" (12 messages) ──────┐
│                                                             │
│ Participants: John, Maria, Carlos, You                      │
│ Duration: Dec 1 - Dec 10                                    │
│                                                             │
│ Key Points:                                                 │
│ • Initial budget proposal: $50k (John)                      │
│ • Maria requested increase for marketing: +$10k             │
│ • Carlos approved with conditions                           │
│ • Final approved budget: $55k                               │
│                                                             │
│ Decisions:                                                  │
│ ✅ Budget approved at $55k                                  │
│ ✅ Marketing gets extra $5k                                 │
│                                                             │
│ Pending Actions:                                            │
│ • You: Send final report to finance                         │
│ • Maria: Submit marketing plan by Dec 15                    │
└─────────────────────────────────────────────────────────────┘
```

---

## Melhorias Técnicas

### TH-15: WebSocket Sync
**Prioridade:** Alta
**Complexidade:** Alta

Sync em tempo real via WebSocket para web interface:
- Eventos push do servidor
- Reconexão automática
- Offline queue

---

### TH-16: SQLite WAL Mode
**Prioridade:** Alta
**Complexidade:** Baixa

Habilitar WAL mode para melhor concorrência:
```go
db.Exec("PRAGMA journal_mode=WAL")
db.Exec("PRAGMA synchronous=NORMAL")
```

---

### TH-17: Incremental FTS Indexing
**Prioridade:** Média
**Complexidade:** Média

Indexação incremental do body:
- Background worker
- Prioriza emails recentes
- Pausa durante uso intenso

---

### TH-18: Connection Health Monitor
**Prioridade:** Média
**Complexidade:** Baixa

Monitoramento de saúde das conexões:
- Ping periódico
- Reconnect automático
- Métricas de latência

---

### TH-19: Rate Limiter
**Prioridade:** Média
**Complexidade:** Média

Limitar requests para evitar bloqueios:
- Gmail API: 250 quota units/second
- IMAP: configurable
- Exponential backoff

---

### TH-20: GraphQL API
**Prioridade:** Baixa
**Complexidade:** Alta

API GraphQL para integrações externas:
```graphql
query {
  emails(folder: "INBOX", limit: 10) {
    id
    subject
    from { name email }
    thread { messageCount }
  }
}

mutation {
  archiveEmail(id: 123)
}
```

---

### TH-21: Telemetry (Opt-in)
**Prioridade:** Baixa
**Complexidade:** Média

Métricas de uso para melhorar o produto:
- Totalmente opt-in
- Dados anonimizados
- Open source dashboard

---

### TH-22: Plugin Hot Reload
**Prioridade:** Baixa
**Complexidade:** Alta

Reload de plugins sem reiniciar:
- Watch de diretório
- Graceful shutdown
- State preservation

---

## Features de Segurança

### SC-12: Email Forensics
**Prioridade:** Baixa
**Complexidade:** Alta

Análise detalhada de headers:
- Full header inspection
- Routing visualization
- SPF/DKIM/DMARC status
- IP geolocation

---

### SC-13: Secure Attachment Viewer
**Prioridade:** Média
**Complexidade:** Média

Visualizar anexos em sandbox:
- Render PDFs in-app
- Image preview
- No execution of files

---

### SC-14: Email Expiration
**Prioridade:** Baixa
**Complexidade:** Média

Auto-delete de emails antigos (GDPR compliance):
- Configurável por folder
- Warnings antes de delete
- Export antes de purge

---

## Como Contribuir com Ideias

### Template para novas ideias:

```markdown
### NF-XX: Nome da Feature
**Prioridade:** Alta/Média/Baixa
**Complexidade:** Baixa/Média/Alta

Descrição breve da feature.

**User Story:**
Como usuário, eu quero [ação] para [benefício].

**UI/UX:**
[Mockup ASCII ou descrição]

**Implementação:**
- [ ] Backend changes
- [ ] TUI changes
- [ ] Desktop changes
- [ ] Database changes
```

---

*Documento vivo - adicione suas ideias!*
*Última atualização: 2025-12-12*
