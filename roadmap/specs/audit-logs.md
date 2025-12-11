# SC-09: Audit Logs

## Overview

Track all actions for compliance and debugging.

## Technical Requirements

```sql
CREATE TABLE audit_logs (
    id INTEGER PRIMARY KEY,
    account_id INTEGER,
    action TEXT NOT NULL,
    entity_type TEXT,
    entity_id INTEGER,
    details TEXT,  -- JSON
    ip_address TEXT,
    user_agent TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

```go
type AuditService interface {
    Log(ctx context.Context, action string, entity Entity, details map[string]interface{}) error
    GetLogs(ctx context.Context, filters AuditFilters) ([]AuditLog, error)
    Export(ctx context.Context, format string) ([]byte, error)
}

// Usage
auditService.Log(ctx, "email.archive", email, map[string]interface{}{
    "folder": "Archive",
    "previous_folder": "INBOX",
})
```

## Acceptance Criteria

- [ ] All actions logged
- [ ] Filterable by action/date
- [ ] Exportable to CSV/JSON
- [ ] Retention policy configurable

## Estimated Complexity

Low-Medium
