# TH-06: Email Body Indexing

## Overview

Index full email body content in FTS5 for comprehensive search.

## User Stories

1. As a user, I want to search within email body text
2. As a user, I want search to find phrases in attachments
3. As a user, I want fast search even with thousands of emails
4. As a user, I want to search HTML content (extracted text)

## Technical Requirements

### Database Schema Update

```sql
-- Update FTS5 to include body_text
DROP TABLE IF EXISTS emails_fts;

CREATE VIRTUAL TABLE emails_fts USING fts5(
    subject,
    from_name,
    from_email,
    to_addresses,
    snippet,
    body_text,  -- Full body content
    content=emails,
    content_rowid=id,
    tokenize='trigram'
);

-- Triggers to keep FTS in sync
CREATE TRIGGER emails_ai AFTER INSERT ON emails BEGIN
    INSERT INTO emails_fts(rowid, subject, from_name, from_email, to_addresses, snippet, body_text)
    VALUES (new.id, new.subject, new.from_name, new.from_email, new.to_addresses, new.snippet, new.body_text);
END;

CREATE TRIGGER emails_ad AFTER DELETE ON emails BEGIN
    INSERT INTO emails_fts(emails_fts, rowid, subject, from_name, from_email, to_addresses, snippet, body_text)
    VALUES ('delete', old.id, old.subject, old.from_name, old.from_email, old.to_addresses, old.snippet, old.body_text);
END;

CREATE TRIGGER emails_au AFTER UPDATE ON emails BEGIN
    INSERT INTO emails_fts(emails_fts, rowid, subject, from_name, from_email, to_addresses, snippet, body_text)
    VALUES ('delete', old.id, old.subject, old.from_name, old.from_email, old.to_addresses, old.snippet, old.body_text);
    INSERT INTO emails_fts(rowid, subject, from_name, from_email, to_addresses, snippet, body_text)
    VALUES (new.id, new.subject, new.from_name, new.from_email, new.to_addresses, new.snippet, new.body_text);
END;
```

### Body Text Extraction

```go
package extraction

import (
    "strings"
    "github.com/PuerkitoBio/goquery"
    "golang.org/x/net/html"
)

// ExtractTextFromHTML extracts readable text from HTML email
func ExtractTextFromHTML(htmlContent string) string {
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
    if err != nil {
        return ""
    }

    // Remove script, style, head
    doc.Find("script, style, head, noscript").Remove()

    // Get text
    text := doc.Text()

    // Clean up whitespace
    text = strings.Join(strings.Fields(text), " ")

    return text
}

// ExtractTextFromEmail extracts searchable text from email
func ExtractTextFromEmail(email *Email) string {
    var parts []string

    // Plain text body
    if email.BodyText != "" {
        parts = append(parts, email.BodyText)
    }

    // HTML body (extract text)
    if email.BodyHTML != "" {
        htmlText := ExtractTextFromHTML(email.BodyHTML)
        if htmlText != "" {
            parts = append(parts, htmlText)
        }
    }

    return strings.Join(parts, " ")
}
```

### Incremental Indexing

```go
func (s *SyncService) indexEmailBodies(ctx context.Context, accountID int64) error {
    // Get emails with empty body_text that have body content
    emails, err := s.storage.GetUnindexedEmails(ctx, accountID, 100)
    if err != nil {
        return err
    }

    for _, email := range emails {
        // Extract text
        bodyText := extraction.ExtractTextFromEmail(&email)

        // Update database
        err = s.storage.UpdateEmailBodyText(ctx, email.ID, bodyText)
        if err != nil {
            log.Printf("Failed to index email %d: %v", email.ID, err)
            continue
        }
    }

    return nil
}

// Background indexer
func (s *SyncService) startBodyIndexer(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.indexEmailBodies(ctx, s.accountID)
        }
    }
}
```

### Search Enhancement

```go
func (r *Repository) Search(ctx context.Context, query string, limit int) ([]Email, error) {
    // Use FTS5 match with ranking
    sql := `
        SELECT e.*, bm25(emails_fts) as rank
        FROM emails e
        JOIN emails_fts fts ON e.id = fts.rowid
        WHERE emails_fts MATCH ?
          AND e.is_deleted = 0
        ORDER BY rank
        LIMIT ?
    `

    // Escape FTS5 special characters
    safeQuery := escapeFTS5Query(query)

    rows, err := r.db.QueryContext(ctx, sql, safeQuery, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    return scanEmails(rows)
}

func escapeFTS5Query(query string) string {
    // Handle phrases
    if strings.Contains(query, " ") && !strings.HasPrefix(query, "\"") {
        // Wrap in quotes for phrase search
        return fmt.Sprintf("\"%s\"", query)
    }
    return query
}
```

### Migration Strategy

```go
func (s *Storage) MigrateBodyIndexing(ctx context.Context) error {
    // 1. Create new FTS table
    _, err := s.db.ExecContext(ctx, createFTSTableSQL)
    if err != nil {
        return err
    }

    // 2. Create triggers
    for _, trigger := range ftsTriggersSQL {
        _, err = s.db.ExecContext(ctx, trigger)
        if err != nil {
            return err
        }
    }

    // 3. Populate FTS from existing emails
    _, err = s.db.ExecContext(ctx, `
        INSERT INTO emails_fts(rowid, subject, from_name, from_email, to_addresses, snippet, body_text)
        SELECT id, subject, from_name, from_email, to_addresses, snippet, body_text
        FROM emails
    `)
    if err != nil {
        return err
    }

    // 4. Start background indexer for emails without body_text
    go s.backgroundIndexer(ctx)

    return nil
}
```

## UI/UX

### Search Results with Highlights

```
┌─ Search: "project budget" ────────────────────────────────────────┐
│ Found 12 results                                                  │
│                                                                   │
│ John Smith - Q4 Budget Review                          Dec 15    │
│ ...discussing the **project budget** for next quarter...          │
│                                                                   │
│ Finance Team - Budget Approval                         Dec 10    │
│ ...approved the **project budget** of $50,000...                  │
│                                                                   │
│ Client - RE: **Project** Discussion                    Dec 8     │
│ ...need to review the **budget** allocation...                    │
└───────────────────────────────────────────────────────────────────┘
```

## Testing

1. Test HTML text extraction
2. Test FTS5 queries
3. Test phrase search
4. Test special character handling
5. Test indexing performance
6. Test migration from old schema

## Acceptance Criteria

- [ ] Body text extracted from plain and HTML emails
- [ ] FTS5 indexes body content
- [ ] Search finds text in body
- [ ] Phrase search works
- [ ] Incremental indexing for existing emails
- [ ] Migration doesn't break existing data
- [ ] Search performance acceptable

## Configuration

```yaml
# config.yaml
search:
  index_body: true
  batch_size: 100
  max_body_length: 100000  # Truncate very long bodies
```

## Estimated Complexity

Medium - Database migration plus extraction logic
