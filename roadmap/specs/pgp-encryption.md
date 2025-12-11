# SC-04: PGP Encryption

## Overview

Support PGP encryption and signing for secure email.

## Technical Requirements

```go
type PGPService interface {
    // Encrypt email for recipients
    Encrypt(ctx context.Context, email *Email, recipients []string) (*Email, error)

    // Decrypt received email
    Decrypt(ctx context.Context, email *Email) (*Email, error)

    // Sign email
    Sign(ctx context.Context, email *Email) (*Email, error)

    // Verify signature
    Verify(ctx context.Context, email *Email) (*VerifyResult, error)

    // Key management
    ImportKey(ctx context.Context, armoredKey string) error
    ExportPublicKey(ctx context.Context) (string, error)
    GetRecipientKey(ctx context.Context, email string) (*PGPKey, error)
}
```

### Key Storage

```sql
CREATE TABLE pgp_keys (
    id INTEGER PRIMARY KEY,
    account_id INTEGER,
    email TEXT,
    public_key TEXT,
    key_id TEXT,
    fingerprint TEXT,
    expires_at DATETIME,
    is_own_key BOOLEAN DEFAULT 0
);
```

## UI Indicators

```
┌─ Email ─────────────────────────────────────────────────────────────┐
│ 🔒 From: john@example.com (Verified signature)                      │
│ Subject: Confidential Project Details                               │
├─────────────────────────────────────────────────────────────────────┤
│ [Decrypted content...]                                              │
└─────────────────────────────────────────────────────────────────────┘
```

## Acceptance Criteria

- [ ] Encrypt outgoing emails
- [ ] Decrypt incoming emails
- [ ] Sign emails
- [ ] Verify signatures
- [ ] Key management UI
- [ ] Keyserver integration

## Estimated Complexity

High - Crypto implementation
