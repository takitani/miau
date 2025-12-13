# Prompt: Multi-Account Runtime Support

> Use este prompt com Claude Code para implementar troca de contas em runtime.

## Contexto

O miau já tem suporte a múltiplas contas no banco de dados. Precisamos implementar a troca de contas em tempo de execução sem reiniciar o app.

## Status Atual

- [x] Database schema suporta múltiplas contas
- [x] Storage adapters suportam account_id
- [x] Config suporta múltiplas contas
- [x] Desktop UI tem AccountSelector
- [ ] Runtime switch (IMAP disconnect/reconnect)
- [ ] TUI account selector

## Arquivos Relevantes

```
internal/app/app.go                    # Application struct
internal/config/config.go              # Config com accounts
internal/imap/client.go                # IMAP connection
internal/services/sync.go              # Sync service
cmd/miau-desktop/frontend/src/lib/components/AccountSelector.svelte
internal/tui/inbox/inbox.go            # TUI principal
```

## Spec Detalhado

Leia: `roadmap/specs/multi-account-support.md`

## Tasks

### 1. Application - SetCurrentAccount

Adicionar ao `internal/app/app.go`:

```go
// Troca a conta ativa
func (a *Application) SetCurrentAccount(email string) error {
    // 1. Verificar se conta existe
    account := a.config.GetAccount(email)
    if account == nil {
        return fmt.Errorf("account not found: %s", email)
    }

    // 2. Desconectar IMAP atual (graceful)
    if a.imapClient != nil && a.imapClient.IsConnected() {
        a.imapClient.Logout()
    }

    // 3. Atualizar account atual
    a.currentAccount = account

    // 4. Emitir evento
    a.eventBus.Publish(AccountChangedEvent{Email: email})

    // 5. Reconectar IMAP (async)
    go func() {
        if err := a.Connect(); err != nil {
            a.eventBus.Publish(ConnectionErrorEvent{Error: err})
        }
    }()

    return nil
}

func (a *Application) GetCurrentAccount() *config.Account {
    return a.currentAccount
}

func (a *Application) GetAccounts() []config.Account {
    return a.config.Accounts
}
```

### 2. Desktop - Binding

Adicionar ao `internal/desktop/app.go`:

```go
func (a *App) SwitchAccount(email string) error {
    return a.app.SetCurrentAccount(email)
}

func (a *App) GetCurrentAccount() *AccountDTO {
    acc := a.app.GetCurrentAccount()
    if acc == nil {
        return nil
    }
    return &AccountDTO{
        Email: acc.Email,
        Name:  acc.Name,
    }
}

func (a *App) GetAccounts() []AccountDTO {
    accounts := a.app.GetAccounts()
    result := make([]AccountDTO, len(accounts))
    for i, acc := range accounts {
        result[i] = AccountDTO{Email: acc.Email, Name: acc.Name}
    }
    return result
}
```

### 3. Desktop - Frontend Update

Atualizar `AccountSelector.svelte` para usar os bindings:

```svelte
<script>
  import { onMount } from 'svelte';
  import { GetAccounts, GetCurrentAccount, SwitchAccount } from '../wailsjs/go/desktop/App';
  import { currentAccount, accounts } from '../stores/accounts.js';

  onMount(async () => {
    $accounts = await GetAccounts();
    $currentAccount = await GetCurrentAccount();
  });

  async function handleSwitch(email) {
    await SwitchAccount(email);
    $currentAccount = await GetCurrentAccount();
    // Reload emails for new account
    window.dispatchEvent(new CustomEvent('account-changed'));
  }
</script>
```

### 4. TUI - Account Selector

Adicionar ao TUI (`internal/tui/inbox/inbox.go`):

```go
// Ctrl+A abre seletor de conta
case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+a"))):
    return m.showAccountSelector()

func (m *Model) showAccountSelector() (tea.Model, tea.Cmd) {
    accounts := m.app.GetAccounts()
    current := m.app.GetCurrentAccount()

    // Criar lista de seleção
    items := make([]list.Item, len(accounts))
    for i, acc := range accounts {
        items[i] = accountItem{
            email: acc.Email,
            name:  acc.Name,
            current: acc.Email == current.Email,
        }
    }

    m.state = stateAccountSelector
    m.accountList.SetItems(items)
    return m, nil
}

func (m *Model) handleAccountSelect(email string) tea.Cmd {
    return func() tea.Msg {
        if err := m.app.SetCurrentAccount(email); err != nil {
            return errMsg{err}
        }
        return accountChangedMsg{email}
    }
}
```

### 5. Event Handling

Eventos para sincronizar UI:

```go
// internal/ports/events.go
type AccountChangedEvent struct {
    Email string
}

// Handlers devem:
// 1. Limpar cache de emails da conta anterior
// 2. Carregar emails da nova conta
// 3. Atualizar status de conexão
```

## Critérios de Aceitação

- [ ] Desktop: Dropdown funciona e troca conta
- [ ] TUI: Ctrl+A abre seletor
- [ ] IMAP desconecta/reconecta corretamente
- [ ] Emails recarregam após troca
- [ ] Status de conexão atualiza
- [ ] Sem memory leaks (goroutines antigas)
- [ ] Funciona com OAuth2 e password

## Cuidados

1. **Graceful disconnect**: Não interromper sync em andamento
2. **Token refresh**: OAuth2 pode precisar refresh
3. **Event cleanup**: Limpar listeners da conta anterior
4. **State reset**: Limpar selection, scroll position, etc.

---

*Prompt criado: 2025-12-12*
