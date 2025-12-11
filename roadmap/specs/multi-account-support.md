# Multi-Account Support

**ID**: AC-01
**Priority**: 🔴 High
**Estimated Effort**: 4-5 days
**Status**: Planned

## Overview

Enable users to manage multiple email accounts within a single miau instance, switching between accounts seamlessly without restarting the application.

## Current State Analysis

### What Already Works (85% Ready)

| Component | Status | Details |
|-----------|--------|---------|
| Database Schema | ✅ 100% | All tables have `account_id` foreign key |
| Config File | ✅ 100% | `config.yaml` supports `accounts: []` array |
| Storage Adapters | ✅ 100% | All methods accept `accountID` parameter |
| Service Layer | ✅ 100% | All services have `SetAccount()` method |
| Token Storage | ✅ 100% | Per-account OAuth2 tokens in `tokens/<email>.json` |
| Sync State | ✅ 100% | Per-account sync tracking |

### What Needs Implementation (15% Remaining)

| Component | Status | Details |
|-----------|--------|---------|
| Application Core | ❌ 0% | `SetCurrentAccount()` returns error |
| TUI Account Selector | ❌ 0% | No UI for switching accounts |
| Desktop Account Selector | ❌ 0% | No UI for switching accounts |
| CLI `--account` flag | ❌ 0% | Commands use first account only |

## Architecture

### Current Flow (Single Account)
```
Startup → Load cfg.Accounts[0] → Initialize Adapters → Run
```

### Target Flow (Multi Account)
```
Startup → Load all cfg.Accounts → Show Selector (if >1) → Select Account
       → Initialize Adapters for selected → Run
       → User switches account → Disconnect old adapters → Initialize new → Continue
```

## Implementation Plan

### Phase 1: Application Core (Day 1-2)

#### 1.1 Update `internal/app/app.go`

```go
type Application struct {
    cfg            *config.Config
    currentAccount *config.Account  // Currently active account
    accounts       []*config.Account // All available accounts
    // ... existing fields
}

func (a *Application) SetCurrentAccount(email string) error {
    // 1. Find account by email
    var newAccount *config.Account
    for i := range a.cfg.Accounts {
        if a.cfg.Accounts[i].Email == email {
            newAccount = &a.cfg.Accounts[i]
            break
        }
    }
    if newAccount == nil {
        return fmt.Errorf("account not found: %s", email)
    }

    // 2. Disconnect current IMAP adapter
    if a.imapAdapter != nil {
        a.imapAdapter.Disconnect()
    }

    // 3. Update current account
    a.currentAccount = newAccount

    // 4. Reinitialize adapters for new account
    a.reinitializeAdapters()

    // 5. Update all services with new account info
    accountInfo := a.buildAccountInfo(newAccount)
    a.emailService.SetAccount(accountInfo)
    a.syncService.SetAccount(accountInfo)
    a.sendService.SetAccount(accountInfo)
    // ... all other services

    // 6. Emit event for UI
    a.eventBus.Publish(events.NewAccountSwitched(email))

    return nil
}

func (a *Application) GetAllAccounts() []ports.AccountInfo {
    var accounts []ports.AccountInfo
    for _, acc := range a.cfg.Accounts {
        accounts = append(accounts, ports.AccountInfo{
            ID:    a.getAccountID(acc.Email),
            Email: acc.Email,
            Name:  acc.Name,
        })
    }
    return accounts
}
```

#### 1.2 Update `internal/ports/app.go`

```go
type App interface {
    // ... existing methods

    // Account management
    GetCurrentAccount() *AccountInfo
    GetAllAccounts() []AccountInfo
    SetCurrentAccount(email string) error
}
```

### Phase 2: CLI Support (Day 2)

#### 2.1 Update `cmd/miau/main.go`

```go
func main() {
    // Add --account flag
    var accountFlag string
    flag.StringVar(&accountFlag, "account", "", "Email address of account to use")

    // If multiple accounts and no flag, show selector
    if len(cfg.Accounts) > 1 && accountFlag == "" {
        accountFlag = selectAccountInteractively(cfg.Accounts)
    }

    // Find account by email or use first
    account := findAccountByEmail(cfg.Accounts, accountFlag)
}

func selectAccountInteractively(accounts []config.Account) string {
    fmt.Println("Multiple accounts configured. Select one:")
    for i, acc := range accounts {
        fmt.Printf("  [%d] %s <%s>\n", i+1, acc.Name, acc.Email)
    }
    // ... read selection
}
```

### Phase 3: TUI Account Selector (Day 3)

#### 3.1 Create `internal/tui/inbox/account_selector.go`

```go
type AccountSelectorModel struct {
    accounts []ports.AccountInfo
    selected int
    onSelect func(email string)
}

func (m AccountSelectorModel) View() string {
    var b strings.Builder
    b.WriteString(titleStyle.Render("Select Account"))
    b.WriteString("\n\n")

    for i, acc := range m.accounts {
        cursor := "  "
        if i == m.selected {
            cursor = "> "
        }
        b.WriteString(fmt.Sprintf("%s%s <%s>\n", cursor, acc.Name, acc.Email))
    }

    b.WriteString("\n[Enter] Select  [Esc] Cancel")
    return b.String()
}
```

#### 3.2 Update `internal/tui/inbox/inbox.go`

```go
// Add keyboard shortcut
case "ctrl+a":
    if len(m.accounts) > 1 {
        m.showAccountSelector = true
        return m, nil
    }

// Add account indicator in status bar
func (m Model) renderStatusBar() string {
    accountInfo := fmt.Sprintf("[%s]", m.currentAccount.Email)
    // ...
}
```

### Phase 4: Desktop Account Selector (Day 4)

#### 4.1 Create `AccountSelector.svelte`

```svelte
<script>
  import { GetAllAccounts, SelectAccount, GetCurrentAccount } from '../bindings/...';

  let accounts = [];
  let currentAccount = null;
  let showDropdown = false;

  onMount(async () => {
    accounts = await GetAllAccounts();
    currentAccount = await GetCurrentAccount();
  });

  async function switchAccount(email) {
    await SelectAccount(email);
    currentAccount = await GetCurrentAccount();
    showDropdown = false;
    // Reload emails for new account
    await loadEmails();
  }
</script>

<div class="account-selector">
  <button on:click={() => showDropdown = !showDropdown}>
    {currentAccount?.email || 'Select Account'}
    <ChevronDown />
  </button>

  {#if showDropdown}
    <div class="dropdown">
      {#each accounts as account}
        <button
          class:active={account.email === currentAccount?.email}
          on:click={() => switchAccount(account.email)}
        >
          <Avatar email={account.email} />
          <span>{account.name}</span>
          <span class="email">{account.email}</span>
        </button>
      {/each}
    </div>
  {/if}
</div>
```

#### 4.2 Add Desktop Bindings

```go
// internal/desktop/bindings.go

func (a *App) GetAllAccounts() []AccountDTO {
    accounts := a.application.GetAllAccounts()
    var result []AccountDTO
    for _, acc := range accounts {
        result = append(result, AccountDTO{
            ID:    acc.ID,
            Email: acc.Email,
            Name:  acc.Name,
        })
    }
    return result
}

func (a *App) SelectAccount(email string) error {
    return a.application.SetCurrentAccount(email)
}
```

### Phase 5: Testing & Polish (Day 5)

1. **Unit Tests**
   - Test `SetCurrentAccount()` with valid/invalid emails
   - Test adapter reinitialization
   - Test event emission on account switch

2. **Integration Tests**
   - Switch accounts while IMAP is connected
   - Verify sync state is per-account
   - Verify contacts/tasks are scoped to account

3. **Edge Cases**
   - Account with expired OAuth2 token
   - Switching during active sync
   - Network errors on switch

## Data Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         USER INTERFACE                          │
├─────────────────────────────────────────────────────────────────┤
│  TUI: Ctrl+A → Account Selector Modal                           │
│  Desktop: Header Dropdown → Account List                        │
│  CLI: --account flag or interactive selector                    │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    APPLICATION CORE                             │
├─────────────────────────────────────────────────────────────────┤
│  SetCurrentAccount(email)                                       │
│    1. Find account in cfg.Accounts                              │
│    2. Disconnect old IMAP                                       │
│    3. Update currentAccount                                     │
│    4. Reinitialize adapters (IMAP, SMTP, Gmail API)            │
│    5. Call SetAccount() on all services                         │
│    6. Emit AccountSwitched event                                │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      SERVICES LAYER                             │
├─────────────────────────────────────────────────────────────────┤
│  EmailService.SetAccount(info)                                  │
│  SyncService.SetAccount(info)                                   │
│  SendService.SetAccount(info)                                   │
│  ContactService.SetAccount(info)                                │
│  TaskService.SetAccount(info)                                   │
│  ... all services receive new account context                   │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      STORAGE LAYER                              │
├─────────────────────────────────────────────────────────────────┤
│  All queries now use new account.ID                             │
│  Data is automatically scoped to account                        │
└─────────────────────────────────────────────────────────────────┘
```

## Event Flow

```
User clicks "Switch Account"
       │
       ▼
Application.SetCurrentAccount("new@email.com")
       │
       ├── Disconnect IMAP ────────────────────► Old connection closed
       │
       ├── Reinitialize Adapters ──────────────► New IMAP/SMTP/Gmail clients
       │
       ├── Update Services ────────────────────► All services get new account
       │
       └── Emit Event ─────────────────────────► AccountSwitched{email}
              │
              ▼
       UI receives event
              │
              ├── TUI: Refresh inbox, update status bar
              │
              └── Desktop: Reload emails, folders, contacts
```

## Configuration Example

```yaml
# ~/.config/miau/config.yaml
accounts:
  - name: "Work"
    email: "andre@company.com"
    auth_type: oauth2
    imap:
      host: imap.gmail.com
      port: 993
    smtp:
      host: smtp.gmail.com
      port: 587
    send_method: gmail_api

  - name: "Personal"
    email: "andre@gmail.com"
    auth_type: oauth2
    imap:
      host: imap.gmail.com
      port: 993
    smtp:
      host: smtp.gmail.com
      port: 587
    send_method: gmail_api

  - name: "Freelance"
    email: "andre@freelance.io"
    auth_type: password
    password: "app-specific-password"
    imap:
      host: mail.freelance.io
      port: 993
    smtp:
      host: mail.freelance.io
      port: 587
    send_method: smtp
```

## UI Mockups

### TUI Account Selector
```
┌─────────────────────────────────────────┐
│         Select Account                  │
├─────────────────────────────────────────┤
│                                         │
│  > Work <andre@company.com>       ✓     │
│    Personal <andre@gmail.com>           │
│    Freelance <andre@freelance.io>       │
│                                         │
├─────────────────────────────────────────┤
│  [Enter] Select  [Esc] Cancel           │
└─────────────────────────────────────────┘
```

### Desktop Account Selector
```
┌──────────────────────────────────────────────────────────────┐
│ ⚙  miau                           [andre@company.com ▼]  ✕  │
├──────────────────────────────────────────────────────────────┤
│                                   ┌─────────────────────────┐│
│                                   │ ● Work                  ││
│                                   │   andre@company.com  ✓  ││
│                                   ├─────────────────────────┤│
│                                   │ ○ Personal              ││
│                                   │   andre@gmail.com       ││
│                                   ├─────────────────────────┤│
│                                   │ ○ Freelance             ││
│                                   │   andre@freelance.io    ││
│                                   └─────────────────────────┘│
└──────────────────────────────────────────────────────────────┘
```

## Potential Issues & Solutions

### 1. IMAP Connection Management
**Issue**: Switching accounts requires disconnecting from one IMAP server and connecting to another.
**Solution**: Graceful disconnect with timeout, retry logic for new connection.

### 2. OAuth2 Token Refresh
**Issue**: Account being switched to may have expired token.
**Solution**: Check token expiry before switch, trigger refresh flow if needed.

### 3. Active Sync During Switch
**Issue**: User switches account while sync is in progress.
**Solution**: Cancel current sync gracefully, emit SyncCancelled event.

### 4. In-Memory Caches
**Issue**: Caches may contain data from previous account.
**Solution**: Clear all caches on account switch.

### 5. Unsaved Drafts
**Issue**: User has unsaved draft when switching.
**Solution**: Prompt to save or discard draft before switch.

## Success Metrics

- [ ] User can configure multiple accounts in config.yaml
- [ ] TUI shows account selector when Ctrl+A pressed
- [ ] Desktop shows account dropdown in header
- [ ] CLI accepts --account flag
- [ ] Account switch completes in <2 seconds
- [ ] All data is correctly scoped to selected account
- [ ] Sync state persists per-account across restarts

## Dependencies

- None (architecture already supports this)

## Related Features

- **IN-05** Slack Integration (may need per-account config)
- **TH-08** Connection Pooling (may optimize multi-account connections)

---

*Created: 2025-12-11*
*Author: Claude + Andre*
