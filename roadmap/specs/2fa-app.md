# SC-11: 2FA for App

## Overview
Optional two-factor authentication to access the miau application.

## Technical Requirements
```go
type TwoFactorService interface {
    Enable(ctx context.Context, accountID int64) (*TOTPSecret, error)
    Verify(ctx context.Context, accountID int64, code string) bool
    Disable(ctx context.Context, accountID int64, code string) error
    GenerateBackupCodes(ctx context.Context, accountID int64) ([]string, error)
}
```

## Storage
```sql
CREATE TABLE two_factor (
    id INTEGER PRIMARY KEY,
    account_id INTEGER UNIQUE,
    secret TEXT NOT NULL,
    backup_codes TEXT,
    enabled_at DATETIME
);
```

## UI Flow
1. Settings > Security > Enable 2FA
2. Show QR code for authenticator app
3. Verify with code
4. Generate backup codes

## Estimated Complexity
Medium
