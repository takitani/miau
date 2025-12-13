# Prompt: Split Inbox (Inbox Inteligente)

> Inspirado no Superhuman - Inbox dividido em seções automáticas.

## Conceito

Ao invés de um inbox único com centenas de emails misturados, dividir em seções inteligentes que categorizam automaticamente:

```
┌─ INBOX ─────────────────────────────────────────────────────┐
│                                                              │
│ ⭐ VIP (3)                                                   │
│   ├─ Boss <ceo@company.com> - Q4 Budget Review              │
│   ├─ Wife <amor@gmail.com> - Jantar hoje?                   │
│   └─ Client <john@bigclient.com> - Contract signed!         │
│                                                              │
│ 👥 Team (12)                                                 │
│   ├─ Dev Team <dev@company.com> - Sprint planning           │
│   ├─ HR <hr@company.com> - Holiday schedule                 │
│   └─ ... mais 10 emails                                      │
│                                                              │
│ 📰 Newsletters (25)                                          │
│   ├─ Medium Daily Digest                                     │
│   ├─ Hacker News Weekly                                      │
│   └─ ... mais 23 emails                                      │
│                                                              │
│ 📦 Other (45)                                                │
│   ├─ GitHub notifications                                    │
│   └─ ... mais 44 emails                                      │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## Como Funciona

### Categorização Automática

1. **VIP**: Contatos marcados como importantes pelo usuário
2. **Team**: Emails do mesmo domínio (@company.com)
3. **Newsletters**: Detectados por headers (List-Unsubscribe, etc)
4. **Transactional**: Recibos, confirmações, notificações
5. **Other**: Todo o resto

### Detecção de Newsletters

```go
// Headers que indicam newsletter
func isNewsletter(headers map[string]string) bool {
    indicators := []string{
        "List-Unsubscribe",
        "List-Id",
        "X-Mailchimp-Campaign",
        "X-Campaign",
        "X-Mailer: mailchimp",
        "X-Mailer: sendgrid",
    }
    for _, h := range indicators {
        if _, ok := headers[h]; ok {
            return true
        }
    }
    // Também checar from_email
    newsletterDomains := []string{
        "mailchimp.com", "sendgrid.net", "amazonses.com",
        "mailgun.org", "constantcontact.com",
    }
    // ...
    return false
}
```

## Database Schema

```sql
-- Categorias de inbox
CREATE TABLE inbox_categories (
    id INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    name TEXT NOT NULL,           -- vip, team, newsletter, transactional, other
    display_name TEXT NOT NULL,   -- "VIP", "Team", etc
    icon TEXT,                    -- emoji ou icon name
    sort_order INTEGER DEFAULT 0,
    is_collapsed BOOLEAN DEFAULT 0,
    UNIQUE(account_id, name)
);

-- Regras de categorização
CREATE TABLE category_rules (
    id INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    category_id INTEGER NOT NULL REFERENCES inbox_categories(id),
    rule_type TEXT NOT NULL,      -- domain, email, header, keyword
    rule_value TEXT NOT NULL,     -- @company.com, boss@x.com, List-Unsubscribe, etc
    priority INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- VIPs (contatos importantes)
CREATE TABLE vip_contacts (
    id INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    email TEXT NOT NULL,
    name TEXT,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, email)
);

-- Cache de categoria por email (para performance)
ALTER TABLE emails ADD COLUMN category TEXT DEFAULT 'other';
CREATE INDEX idx_emails_category ON emails(account_id, category, is_deleted);
```

## Service Implementation

```go
// internal/services/inbox_category.go

type InboxCategoryService struct {
    storage ports.InboxCategoryStorage
    email   ports.EmailService
}

type Category struct {
    ID          int64
    Name        string
    DisplayName string
    Icon        string
    Count       int
    IsCollapsed bool
}

// CategorizeEmail determina a categoria de um email
func (s *InboxCategoryService) CategorizeEmail(ctx context.Context, email *Email) (string, error) {
    accountID := email.AccountID

    // 1. Verificar se é VIP
    isVIP, _ := s.storage.IsVIP(ctx, accountID, email.FromEmail)
    if isVIP {
        return "vip", nil
    }

    // 2. Verificar regras customizadas
    rules, _ := s.storage.GetRules(ctx, accountID)
    for _, rule := range rules {
        if s.matchesRule(email, rule) {
            return rule.CategoryName, nil
        }
    }

    // 3. Detectar newsletter automaticamente
    if s.isNewsletter(email) {
        return "newsletter", nil
    }

    // 4. Detectar team (mesmo domínio)
    userDomain := s.extractDomain(s.getUserEmail(ctx, accountID))
    emailDomain := s.extractDomain(email.FromEmail)
    if userDomain == emailDomain && userDomain != "" {
        return "team", nil
    }

    // 5. Detectar transacional
    if s.isTransactional(email) {
        return "transactional", nil
    }

    return "other", nil
}

func (s *InboxCategoryService) isNewsletter(email *Email) bool {
    // Checar headers
    headers := email.ParsedHeaders()
    newsletterHeaders := []string{
        "List-Unsubscribe", "List-Id", "X-Mailchimp-Campaign",
        "X-Campaign", "Precedence: bulk", "Precedence: list",
    }
    for _, h := range newsletterHeaders {
        if _, ok := headers[h]; ok {
            return true
        }
    }

    // Checar domínio do remetente
    newsletterDomains := []string{
        "mailchimp.com", "sendgrid.net", "amazonses.com",
        "mailgun.org", "constantcontact.com", "hubspot.com",
        "mailerlite.com", "convertkit.com", "substack.com",
    }
    for _, domain := range newsletterDomains {
        if strings.Contains(email.FromEmail, domain) {
            return true
        }
    }

    return false
}

func (s *InboxCategoryService) isTransactional(email *Email) bool {
    // Keywords em subject que indicam transacional
    transactionalKeywords := []string{
        "order confirmation", "payment received", "invoice",
        "receipt", "shipping", "delivery", "password reset",
        "verify your email", "your purchase", "booking confirmed",
    }
    subjectLower := strings.ToLower(email.Subject)
    for _, kw := range transactionalKeywords {
        if strings.Contains(subjectLower, kw) {
            return true
        }
    }

    // Remetentes típicos de transacional
    transactionalDomains := []string{
        "noreply@", "no-reply@", "notifications@",
        "alerts@", "support@", "billing@",
    }
    for _, prefix := range transactionalDomains {
        if strings.HasPrefix(email.FromEmail, prefix) {
            return true
        }
    }

    return false
}

// GetCategorizedInbox retorna emails agrupados por categoria
func (s *InboxCategoryService) GetCategorizedInbox(ctx context.Context, accountID int64) ([]CategoryWithEmails, error) {
    categories := []string{"vip", "team", "newsletter", "transactional", "other"}
    result := make([]CategoryWithEmails, 0, len(categories))

    for _, cat := range categories {
        emails, _ := s.storage.GetEmailsByCategory(ctx, accountID, cat, 50)
        if len(emails) > 0 {
            result = append(result, CategoryWithEmails{
                Category: s.getCategoryInfo(cat),
                Emails:   emails,
                Total:    len(emails),
            })
        }
    }

    return result, nil
}

// AddVIP marca um contato como VIP
func (s *InboxCategoryService) AddVIP(ctx context.Context, accountID int64, email string) error {
    if err := s.storage.AddVIP(ctx, accountID, email); err != nil {
        return err
    }
    // Recategorizar emails existentes desse remetente
    return s.recategorizeFromSender(ctx, accountID, email, "vip")
}
```

## Desktop UI

```svelte
<!-- SplitInbox.svelte -->
<script>
  import { onMount } from 'svelte';
  import { GetCategorizedInbox, ToggleCategoryCollapse } from '../wailsjs/go/desktop/App';

  let categories = [];
  let selectedCategory = 'all';

  onMount(async () => {
    categories = await GetCategorizedInbox();
  });

  function toggleCollapse(category) {
    category.isCollapsed = !category.isCollapsed;
    ToggleCategoryCollapse(category.name, category.isCollapsed);
  }

  const categoryIcons = {
    vip: '⭐',
    team: '👥',
    newsletter: '📰',
    transactional: '🧾',
    other: '📦'
  };
</script>

<div class="split-inbox">
  <!-- Category tabs -->
  <div class="category-tabs">
    <button
      class:active={selectedCategory === 'all'}
      on:click={() => selectedCategory = 'all'}
    >
      All
    </button>
    {#each categories as cat}
      <button
        class:active={selectedCategory === cat.name}
        on:click={() => selectedCategory = cat.name}
      >
        {categoryIcons[cat.name]} {cat.displayName}
        <span class="count">{cat.total}</span>
      </button>
    {/each}
  </div>

  <!-- Categorized view -->
  {#if selectedCategory === 'all'}
    {#each categories as cat}
      <div class="category-section">
        <button
          class="category-header"
          on:click={() => toggleCollapse(cat)}
        >
          <span class="icon">{categoryIcons[cat.name]}</span>
          <span class="name">{cat.displayName}</span>
          <span class="count">({cat.total})</span>
          <span class="chevron">{cat.isCollapsed ? '▶' : '▼'}</span>
        </button>

        {#if !cat.isCollapsed}
          <div class="emails">
            {#each cat.emails.slice(0, 5) as email}
              <EmailRow {email} />
            {/each}
            {#if cat.total > 5}
              <button class="show-more">
                Ver mais {cat.total - 5} emails...
              </button>
            {/if}
          </div>
        {/if}
      </div>
    {/each}
  {:else}
    <!-- Single category view -->
    <EmailList category={selectedCategory} />
  {/if}
</div>

<style>
  .category-tabs {
    display: flex;
    gap: var(--space-sm);
    padding: var(--space-md);
    border-bottom: 1px solid var(--border-subtle);
    overflow-x: auto;
  }

  .category-tabs button {
    padding: var(--space-sm) var(--space-md);
    border-radius: var(--radius-md);
    background: transparent;
    border: none;
    cursor: pointer;
    white-space: nowrap;
  }

  .category-tabs button.active {
    background: var(--accent-primary);
    color: white;
  }

  .category-section {
    border-bottom: 1px solid var(--border-subtle);
  }

  .category-header {
    width: 100%;
    display: flex;
    align-items: center;
    gap: var(--space-sm);
    padding: var(--space-md);
    background: var(--bg-secondary);
    border: none;
    cursor: pointer;
    text-align: left;
  }

  .category-header:hover {
    background: var(--bg-tertiary);
  }

  .count {
    color: var(--text-secondary);
    font-size: 0.85em;
  }

  .chevron {
    margin-left: auto;
    color: var(--text-tertiary);
  }

  .show-more {
    width: 100%;
    padding: var(--space-sm);
    background: transparent;
    border: none;
    color: var(--accent-primary);
    cursor: pointer;
  }
</style>
```

## TUI Implementation

```go
// Mostrar categorias no inbox
func (m *Model) renderSplitInbox() string {
    var b strings.Builder

    for _, cat := range m.categories {
        // Header da categoria
        header := fmt.Sprintf("%s %s (%d)", cat.Icon, cat.DisplayName, cat.Total)
        if cat.IsCollapsed {
            header += " ▶"
        } else {
            header += " ▼"
        }
        b.WriteString(m.styles.CategoryHeader.Render(header) + "\n")

        if !cat.IsCollapsed {
            // Emails da categoria
            for i, email := range cat.Emails[:min(5, len(cat.Emails))] {
                b.WriteString(m.renderEmailRow(email, i) + "\n")
            }
            if cat.Total > 5 {
                b.WriteString(m.styles.ShowMore.Render(
                    fmt.Sprintf("  ... mais %d emails", cat.Total-5),
                ) + "\n")
            }
        }
        b.WriteString("\n")
    }

    return b.String()
}

// Teclas para navegação
case "Tab":
    // Próxima categoria
    m.currentCategory = (m.currentCategory + 1) % len(m.categories)
case "v":
    // Marcar remetente como VIP
    if email := m.selectedEmail(); email != nil {
        m.app.InboxCategory().AddVIP(ctx, email.AccountID, email.FromEmail)
    }
```

## Sync Integration

```go
// No sync service, categorizar novos emails
func (s *SyncService) onNewEmail(ctx context.Context, email *Email) error {
    // Categorizar email
    category, _ := s.categoryService.CategorizeEmail(ctx, email)
    email.Category = category

    // Salvar com categoria
    return s.storage.SaveEmail(ctx, email)
}
```

## Critérios de Aceitação

- [ ] Emails são categorizados automaticamente no sync
- [ ] VIPs podem ser adicionados/removidos
- [ ] Newsletters detectadas por headers
- [ ] Team detectado por domínio
- [ ] UI Desktop mostra categorias colapsáveis
- [ ] TUI mostra categorias com navegação
- [ ] Performance: categorização < 10ms por email
- [ ] Categorias customizáveis pelo usuário

---

*Inspirado em: Superhuman Split Inbox*
