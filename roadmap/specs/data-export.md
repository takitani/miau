# SC-10: Data Export (GDPR)

## Overview

Export all user data in portable formats for GDPR compliance.

## Technical Requirements

```go
type ExportService interface {
    ExportAll(ctx context.Context, accountID int64, format ExportFormat) (*ExportResult, error)
    ExportEmails(ctx context.Context, accountID int64, format ExportFormat) ([]byte, error)
    ExportContacts(ctx context.Context, accountID int64) ([]byte, error)
    ExportSettings(ctx context.Context, accountID int64) ([]byte, error)
}

type ExportFormat string

const (
    FormatMBOX ExportFormat = "mbox"
    FormatEML  ExportFormat = "eml"
    FormatJSON ExportFormat = "json"
)
```

### CLI Command

```bash
miau export --format mbox --output ~/my-emails.mbox
miau export --format json --output ~/my-data.json
```

## Acceptance Criteria

- [ ] Export all emails
- [ ] Export contacts
- [ ] Export settings
- [ ] Multiple formats (mbox, eml, json)
- [ ] Progress indicator for large exports

## Estimated Complexity

Medium
