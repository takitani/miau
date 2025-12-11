# EM-17: Email Delegation (Team)

## Overview

Allow team members to access and manage emails on behalf of others (Google Workspace delegation).

## User Stories

1. As an executive, I want my assistant to manage my inbox
2. As an assistant, I want to read and respond to emails on behalf of my boss
3. As a team lead, I want to delegate inbox access during vacation
4. As a user, I want to see who has access to my inbox

## Technical Requirements

### Service Layer

Create `internal/services/delegation.go`:

```go
package services

type DelegationService interface {
    // AddDelegate grants inbox access to another user
    AddDelegate(ctx context.Context, accountID int64, delegateEmail string, perms DelegatePermissions) error

    // RemoveDelegate revokes inbox access
    RemoveDelegate(ctx context.Context, accountID int64, delegateEmail string) error

    // GetDelegates returns all delegates for an account
    GetDelegates(ctx context.Context, accountID int64) ([]Delegate, error)

    // GetDelegatedAccounts returns accounts user has access to
    GetDelegatedAccounts(ctx context.Context, userEmail string) ([]DelegatedAccount, error)

    // SwitchAccount switches to delegated account
    SwitchAccount(ctx context.Context, targetAccountID int64) error

    // SendAsDelegate sends email on behalf of another user
    SendAsDelegate(ctx context.Context, delegatorID int64, email *Email) error

    // SyncDelegates syncs delegation from Gmail API
    SyncDelegates(ctx context.Context, accountID int64) error
}

type Delegate struct {
    ID              int64
    AccountID       int64
    DelegateEmail   string
    DelegateName    string
    Permissions     DelegatePermissions
    AddedAt         time.Time
    ExpiresAt       *time.Time
    Source          string  // "local" or "gmail"
}

type DelegatePermissions struct {
    Read      bool
    Compose   bool
    Send      bool
    Delete    bool
    Manage    bool  // Can add/remove other delegates
}

type DelegatedAccount struct {
    AccountID    int64
    Email        string
    Name         string
    Permissions  DelegatePermissions
}
```

### Database Schema

```sql
CREATE TABLE delegates (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    delegate_email TEXT NOT NULL,
    delegate_name TEXT,
    can_read BOOLEAN DEFAULT 1,
    can_compose BOOLEAN DEFAULT 1,
    can_send BOOLEAN DEFAULT 0,
    can_delete BOOLEAN DEFAULT 0,
    can_manage BOOLEAN DEFAULT 0,
    added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME,
    source TEXT DEFAULT 'local',
    UNIQUE(account_id, delegate_email)
);
```

### Gmail API Integration

```go
// Sync delegates from Gmail API
func (s *DelegationService) SyncDelegates(ctx context.Context, accountID int64) error {
    // Get account
    account, err := s.accountService.GetAccount(ctx, accountID)
    if err != nil {
        return err
    }

    // Only for Gmail/Workspace accounts
    if account.AuthType != "oauth2" {
        return nil
    }

    // Call Gmail API for delegates
    // GET https://gmail.googleapis.com/gmail/v1/users/me/settings/delegates
    delegates, err := s.gmailAPI.ListDelegates(ctx, account.Email)
    if err != nil {
        return err
    }

    // Sync to local database
    for _, delegate := range delegates {
        s.storage.UpsertDelegate(ctx, &Delegate{
            AccountID:     accountID,
            DelegateEmail: delegate.DelegateEmail,
            Permissions:   DelegatePermissions{Read: true, Send: true},
            Source:        "gmail",
        })
    }

    return nil
}

// Send as delegate
func (s *DelegationService) SendAsDelegate(ctx context.Context, delegatorID int64, email *Email) error {
    delegator, err := s.accountService.GetAccount(ctx, delegatorID)
    if err != nil {
        return err
    }

    // Set From to delegator's email
    email.From = delegator.Email
    email.FromName = delegator.Name

    // Add Sender header with delegate's email
    email.Headers["Sender"] = s.currentUser.Email

    return s.sendService.SendEmail(ctx, email)
}
```

## UI/UX

### TUI
- Account switcher when delegates available
- Indicator showing current account context

```
┌─ Account ─────────────────────────────────────────────────────────┐
│ Current: you@company.com                                          │
│                                                                   │
│ Switch to:                                                        │
│ [1] boss@company.com (delegated)                                  │
│ [2] team@company.com (shared)                                     │
│                                                                   │
│ [Esc] Cancel                                                      │
└───────────────────────────────────────────────────────────────────┘

┌─ INBOX (boss@company.com) ────────────────────────────────────────┐
│ ⚠️  Viewing as delegate for: Boss Name                            │
├───────────────────────────────────────────────────────────────────┤
│   Client        │ Contract ready        │ 10:30 AM                │
│   Partner       │ Meeting request       │ 09:45 AM                │
│   HR            │ Benefits update       │ Yesterday               │
└───────────────────────────────────────────────────────────────────┘
  [Tab] Switch account  [c] Compose as boss  [r] Reply as boss
```

### Delegation Management

```
┌─ Manage Delegates ────────────────────────────────────────────────┐
│                                                                   │
│ People with access to your inbox:                                 │
│                                                                   │
│ assistant@company.com                                             │
│   Permissions: Read, Compose, Send                                │
│   Added: Dec 1, 2024                                              │
│   [e] Edit  [r] Remove                                            │
│                                                                   │
│ [+] Add delegate                                                  │
│                                                                   │
│ ─────────────────────────────────────────────────────────────────│
│                                                                   │
│ Inboxes you have access to:                                       │
│                                                                   │
│ boss@company.com                                                  │
│   Permissions: Read only                                          │
│   [s] Switch to this account                                      │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Account dropdown in header
- Delegation settings page
- Visual indicator for delegated context
- Send-as selector in compose

## Testing

1. Test adding/removing delegates
2. Test permission enforcement
3. Test send-as functionality
4. Test Gmail API sync
5. Test account switching
6. Test expiration handling

## Acceptance Criteria

- [ ] Can add delegates with specific permissions
- [ ] Can view delegated accounts
- [ ] Can switch between accounts
- [ ] Can compose/send on behalf of delegator
- [ ] Syncs with Gmail delegation settings
- [ ] Clear visual indicator of current context
- [ ] Respects permission levels
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
delegation:
  enabled: true
  sync_gmail_delegates: true
  default_permissions:
    read: true
    compose: true
    send: false
```

## Security Considerations

- Delegates must be authenticated users
- Audit log for delegated actions
- Permission changes logged
- Expiration support for temporary access

## Estimated Complexity

High - Multi-account handling plus API integration
