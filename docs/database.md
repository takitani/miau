# miau Database Schema

## Entity Relationship Diagram

```mermaid
erDiagram
    accounts ||--o{ folders : has
    accounts ||--o{ emails : contains
    accounts ||--o{ drafts : creates
    accounts ||--o{ sent_emails : sends
    accounts ||--o{ emails_archive : archives
    accounts ||--o{ drafts_history : logs
    accounts ||--o{ pending_batch_ops : queues
    accounts ||--o{ content_index_state : tracks
    accounts ||--o{ app_settings : configures
    folders ||--o{ emails : contains
    emails ||--o| emails_fts : indexes

    accounts {
        int id PK
        text email UK
        text name
        datetime created_at
    }

    folders {
        int id PK
        int account_id FK
        text name
        int total_messages
        int unread_messages
        datetime last_sync
    }

    emails {
        int id PK
        int account_id FK
        int folder_id FK
        int uid
        text message_id
        text subject
        text from_name
        text from_email
        text to_addresses
        text cc_addresses
        datetime date
        bool is_read
        bool is_starred
        bool is_archived
        bool is_deleted
        bool is_replied
        bool has_attachments
        bool body_indexed
        text snippet
        text body_text
        text body_html
        text raw_headers
        int size
        datetime created_at
        datetime updated_at
    }

    emails_fts {
        int rowid PK
        text subject
        text from_name
        text from_email
        text body_text
    }

    drafts {
        int id PK
        int account_id FK
        text to_addresses
        text cc_addresses
        text bcc_addresses
        text subject
        text body_html
        text body_text
        text classification
        text in_reply_to
        text reference_ids
        int reply_to_email_id FK
        text status
        datetime scheduled_send_at
        datetime sent_at
        text generation_source
        text ai_prompt
        text error_message
        datetime created_at
        datetime updated_at
    }

    sent_emails {
        int id PK
        int account_id FK
        text message_id
        text to_addresses
        text cc_addresses
        text bcc_addresses
        text subject
        text body_html
        text body_text
        text in_reply_to
        text reference_ids
        int reply_to_email_id FK
        datetime sent_at
        text send_method
        int draft_id FK
    }

    emails_archive {
        int id PK
        int original_id
        int account_id FK
        int folder_id
        int uid
        text message_id
        text subject
        text from_name
        text from_email
        text to_addresses
        text cc_addresses
        datetime date
        bool is_read
        bool is_starred
        bool has_attachments
        text snippet
        text body_text
        text body_html
        text raw_headers
        int size
        datetime original_created_at
        datetime original_updated_at
        datetime archived_at
        text archive_reason
    }

    drafts_history {
        int id PK
        int original_id
        int account_id FK
        text to_addresses
        text cc_addresses
        text bcc_addresses
        text subject
        text body_html
        text body_text
        text classification
        text in_reply_to
        text reference_ids
        int reply_to_email_id FK
        text final_status
        datetime scheduled_send_at
        datetime sent_at
        text generation_source
        text ai_prompt
        text error_message
        datetime original_created_at
        datetime original_updated_at
        datetime archived_at
    }

    pending_batch_ops {
        int id PK
        int account_id FK
        text operation
        text description
        text filter_query
        text email_ids
        int email_count
        text preview_data
        text status
        datetime created_at
        datetime executed_at
    }

    content_index_state {
        int id PK
        int account_id FK
        text status
        int total_emails
        int indexed_emails
        int last_indexed_uid
        int speed
        text last_error
        datetime started_at
        datetime paused_at
        datetime completed_at
        datetime created_at
        datetime updated_at
    }

    app_settings {
        int id PK
        int account_id FK
        text key
        text value
        datetime created_at
        datetime updated_at
    }
```

## Table Descriptions

### Core Tables

| Table | Purpose |
|-------|---------|
| `accounts` | User email accounts |
| `folders` | IMAP folders/labels |
| `emails` | Email messages (cached from IMAP) |
| `emails_fts` | Full-text search index (FTS5 trigram) |

### Composition & Sending

| Table | Purpose |
|-------|---------|
| `drafts` | Draft emails awaiting send |
| `sent_emails` | Permanent record of sent emails |

### Archive Tables (Permanent Storage)

| Table | Purpose |
|-------|---------|
| `emails_archive` | Archived emails (after server deletion) |
| `drafts_history` | Historical draft records |

### Operations & State

| Table | Purpose |
|-------|---------|
| `pending_batch_ops` | Queued bulk operations with preview |
| `content_index_state` | Background indexer progress |
| `app_settings` | Per-account settings |

## Key Indexes

```sql
-- Email retrieval
idx_emails_account_folder ON emails(account_id, folder_id)
idx_emails_date ON emails(date DESC)
idx_emails_from ON emails(from_email)
idx_emails_subject ON emails(subject)
idx_emails_is_read ON emails(is_read)
idx_emails_is_archived ON emails(is_archived)
idx_emails_body_indexed ON emails(body_indexed)

-- Drafts
idx_drafts_account_status ON drafts(account_id, status)
idx_drafts_scheduled ON drafts(status, scheduled_send_at)

-- Archive
idx_emails_archive_account ON emails_archive(account_id)
idx_emails_archive_date ON emails_archive(date DESC)
idx_emails_archive_from ON emails_archive(from_email)

-- Operations
idx_pending_batch_ops_status ON pending_batch_ops(account_id, status)
idx_app_settings_account_key ON app_settings(account_id, key)
```

## FTS5 Full-Text Search

The `emails_fts` virtual table uses **trigram tokenization** for fuzzy partial matching:

```sql
CREATE VIRTUAL TABLE emails_fts USING fts5(
    subject,
    from_name,
    from_email,
    body_text,
    content='emails',
    content_rowid='id',
    tokenize='trigram'
);
```

**Features:**
- Partial word matching (e.g., "inv" matches "invoice")
- Case-insensitive search
- Multi-field search (subject, from, body)
- Auto-sync via triggers

**Example Queries:**
```sql
-- Fuzzy search
SELECT * FROM emails_fts WHERE emails_fts MATCH 'invoice';

-- With email metadata
SELECT e.* FROM emails e
JOIN emails_fts fts ON e.id = fts.rowid
WHERE emails_fts MATCH 'newsletter' AND e.is_deleted = 0;
```

## Soft Delete Strategy

miau never hard-deletes data:

| Flag | Meaning | Recovery |
|------|---------|----------|
| `is_archived` | Hidden from inbox, kept in DB | Toggle back |
| `is_deleted` | In trash, pending archive | Restore within 30 days |
| `emails_archive` | Permanently archived | Search archive table |

## Draft Status Flow

```mermaid
stateDiagram-v2
    [*] --> draft: Create
    draft --> scheduled: Approve (AI drafts)
    draft --> cancelled: Cancel
    scheduled --> sending: Timer triggers
    sending --> sent: Success
    sending --> failed: Error
    sent --> [*]: Move to drafts_history
    cancelled --> [*]: Move to drafts_history
    failed --> draft: Retry
```

## Data Retention Policy

| Data Type | Retention | Storage |
|-----------|-----------|---------|
| Active emails | Synced with server | `emails` |
| Deleted by server | Permanent | `emails_archive` |
| User-deleted | 30 days → archive | `emails` → `emails_archive` |
| Sent emails | Permanent | `sent_emails` |
| Drafts | Until sent/cancelled | `drafts` → `drafts_history` |
