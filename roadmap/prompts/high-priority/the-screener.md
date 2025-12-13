# Prompt: The Screener (Triagem de Remetentes)

> Inspirado no HEY.com - Você controla quem pode te enviar emails.

## Conceito

Quando alguém te envia email pela primeira vez, o email vai para "Triagem" (Screener). Você decide se quer receber emails dessa pessoa/serviço.

## Como Funciona

```
┌─ Triagem (5 novos) ─────────────────────────────────────────┐
│                                                             │
│ newsletter@medium.com                                       │
│ "Welcome to Medium Daily Digest"                            │
│ [✅ Aceitar] [❌ Bloquear] [📰 Newsletter] [🧾 Recibo]       │
│                                                             │
│ john.doe@company.com                                        │
│ "Introduction - Partnership Opportunity"                    │
│ [✅ Aceitar] [❌ Bloquear] [📰 Newsletter] [🧾 Recibo]       │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Decisões

| Ação | Resultado |
|------|-----------|
| ✅ Aceitar | Emails vão para Inbox, futuros passam direto |
| ❌ Bloquear | Email deletado, futuros auto-deletados |
| 📰 Newsletter | Email vai para Feed, futuros também |
| 🧾 Recibo | Email vai para Paper Trail, futuros também |

## Database Schema

```sql
-- Tabela de decisões do Screener
CREATE TABLE screener_decisions (
    id INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    email_address TEXT NOT NULL,      -- Email ou domínio
    decision TEXT NOT NULL,            -- accept, block, newsletter, receipt
    decided_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, email_address)
);

-- Emails pendentes de triagem
CREATE TABLE screener_pending (
    id INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    email_id INTEGER NOT NULL REFERENCES emails(id),
    from_email TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Index para lookup rápido
CREATE INDEX idx_screener_email ON screener_decisions(account_id, email_address);
```

## Service Implementation

```go
// internal/services/screener.go

type ScreenerService struct {
    storage ports.ScreenerStorage
    email   ports.EmailService
}

type ScreenerDecision string

const (
    DecisionAccept     ScreenerDecision = "accept"
    DecisionBlock      ScreenerDecision = "block"
    DecisionNewsletter ScreenerDecision = "newsletter"
    DecisionReceipt    ScreenerDecision = "receipt"
)

// ProcessIncomingEmail verifica se email precisa de triagem
func (s *ScreenerService) ProcessIncomingEmail(ctx context.Context, email *Email) error {
    // Verificar se já temos decisão para este remetente
    decision, err := s.storage.GetDecision(ctx, email.AccountID, email.FromEmail)
    if err != nil {
        return err
    }

    if decision != nil {
        // Já temos decisão, aplicar
        return s.applyDecision(ctx, email, *decision)
    }

    // Primeira vez - adicionar à triagem
    return s.storage.AddToPending(ctx, email.AccountID, email.ID, email.FromEmail)
}

func (s *ScreenerService) applyDecision(ctx context.Context, email *Email, decision ScreenerDecision) error {
    switch decision {
    case DecisionAccept:
        // Deixar no inbox normal
        return nil
    case DecisionBlock:
        // Deletar email
        return s.email.Delete(ctx, email.ID)
    case DecisionNewsletter:
        // Mover para Feed
        return s.email.MoveToFolder(ctx, email.ID, "Feed")
    case DecisionReceipt:
        // Mover para Paper Trail
        return s.email.MoveToFolder(ctx, email.ID, "Paper Trail")
    }
    return nil
}

// Decide registra decisão do usuário
func (s *ScreenerService) Decide(ctx context.Context, accountID int64, emailAddress string, decision ScreenerDecision) error {
    // Salvar decisão
    if err := s.storage.SaveDecision(ctx, accountID, emailAddress, decision); err != nil {
        return err
    }

    // Aplicar a todos os emails pendentes deste remetente
    pending, err := s.storage.GetPendingByEmail(ctx, accountID, emailAddress)
    if err != nil {
        return err
    }

    for _, p := range pending {
        email, _ := s.email.GetEmail(ctx, p.EmailID)
        if email != nil {
            s.applyDecision(ctx, email, decision)
        }
        s.storage.RemoveFromPending(ctx, p.ID)
    }

    return nil
}

// GetPendingCount retorna quantidade de emails pendentes
func (s *ScreenerService) GetPendingCount(ctx context.Context, accountID int64) (int, error) {
    return s.storage.CountPending(ctx, accountID)
}

// GetPendingEmails retorna emails pendentes de triagem
func (s *ScreenerService) GetPendingEmails(ctx context.Context, accountID int64) ([]PendingEmail, error) {
    return s.storage.GetPending(ctx, accountID)
}
```

## Desktop UI

Adicionar ao sidebar um item "Triagem" com badge:

```svelte
<!-- ModernSidebar.svelte -->
<script>
  import { onMount } from 'svelte';
  import { GetScreenerCount } from '../wailsjs/go/desktop/App';

  let screenerCount = 0;

  onMount(async () => {
    screenerCount = await GetScreenerCount();
  });
</script>

<nav>
  <!-- ... outras pastas ... -->

  {#if screenerCount > 0}
    <button
      class="folder-item screener"
      on:click={() => selectFolder('Screener')}
    >
      <span class="icon">🛡️</span>
      <span class="name">Triagem</span>
      <span class="badge">{screenerCount}</span>
    </button>
  {/if}
</nav>
```

Criar componente de triagem:

```svelte
<!-- ScreenerPanel.svelte -->
<script>
  import { GetPendingEmails, ScreenerDecide } from '../wailsjs/go/desktop/App';

  let pendingEmails = [];

  async function loadPending() {
    pendingEmails = await GetPendingEmails();
  }

  async function decide(email, decision) {
    await ScreenerDecide(email.fromEmail, decision);
    await loadPending();
  }
</script>

<div class="screener-panel">
  <h2>Triagem de Remetentes</h2>
  <p>Primeira vez que estes remetentes te enviaram email.</p>

  {#each pendingEmails as email}
    <div class="pending-email">
      <div class="sender">
        <strong>{email.fromName || email.fromEmail}</strong>
        <span class="email">{email.fromEmail}</span>
      </div>
      <div class="subject">{email.subject}</div>
      <div class="actions">
        <button on:click={() => decide(email, 'accept')} title="Aceitar">
          ✅ Aceitar
        </button>
        <button on:click={() => decide(email, 'block')} title="Bloquear">
          ❌ Bloquear
        </button>
        <button on:click={() => decide(email, 'newsletter')} title="Newsletter">
          📰 Newsletter
        </button>
        <button on:click={() => decide(email, 'receipt')} title="Recibo">
          🧾 Recibo
        </button>
      </div>
    </div>
  {/each}
</div>
```

## TUI Implementation

```go
// Mostrar badge no folder list
func (m *Model) renderFolders() string {
    // ...
    if m.screenerCount > 0 {
        folders = append(folders, fmt.Sprintf("🛡️ Triagem (%d)", m.screenerCount))
    }
    // ...
}

// Tecla 'T' abre triagem
case "T":
    return m.showScreener()
```

## Integration Points

1. **Sync Service**: Após baixar email, chamar `ProcessIncomingEmail`
2. **Folders**: Criar pastas virtuais "Feed" e "Paper Trail"
3. **Notifications**: Não notificar emails em triagem

## Critérios de Aceitação

- [ ] Emails de novos remetentes vão para Triagem
- [ ] 4 opções de decisão funcionam
- [ ] Decisão aplica a todos emails pendentes do remetente
- [ ] Futuros emails seguem a decisão
- [ ] Badge mostra quantidade pendente
- [ ] Desktop e TUI funcionam
- [ ] Performance: lookup < 1ms

---

*Inspirado em: [HEY.com - The Screener](https://hey.com/features/the-screener/)*
