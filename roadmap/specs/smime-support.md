# SC-05: S/MIME Support

## Overview
Support S/MIME certificates for email signing and encryption.

## Technical Requirements
```go
type SMIMEService interface {
    Sign(ctx context.Context, email *Email, cert *Certificate) (*Email, error)
    Encrypt(ctx context.Context, email *Email, recipientCerts []*Certificate) (*Email, error)
    Decrypt(ctx context.Context, email *Email) (*Email, error)
    Verify(ctx context.Context, email *Email) (*VerifyResult, error)
    ImportCertificate(ctx context.Context, p12 []byte, password string) error
}
```

## Certificate Storage
```sql
CREATE TABLE smime_certs (
    id INTEGER PRIMARY KEY,
    account_id INTEGER,
    email TEXT,
    certificate BLOB,
    private_key BLOB,  -- Encrypted
    expires_at DATETIME
);
```

## Estimated Complexity
High
