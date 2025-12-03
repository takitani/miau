# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## About

**miau** (Mail Intelligence Assistant Utility) is a local email client with TUI interface via IMAP and AI integration for email management assistance. The goal is local control of emails with privacy focus - everything runs locally.

## Build Commands

```bash
make run              # Run without compiling
make build            # Build binary
make test             # Run tests
make lint             # Format + vet
go test ./internal/storage/...  # Run single package tests
```

## Architecture

### Application Flow (cmd/miau/main.go)
The app uses Bubble Tea as its TUI framework with a state machine pattern:
- `stateSetup` → First-run setup wizard (internal/tui/setup)
- `stateInbox` → Main email interface (internal/tui/inbox)

### Key Packages

**internal/config** - Configuration via Viper
- Config stored at `~/.config/miau/config.yaml`
- Supports multiple accounts with two auth types: `password` or `oauth2`

**internal/auth** - OAuth2 authentication
- Browser-based OAuth2 flow for Gmail/Google Workspace
- Tokens stored at `~/.config/miau/tokens/{account}.json`
- Implements XOAUTH2 SASL mechanism for IMAP

**internal/imap** - IMAP client wrapper
- Wraps go-imap/v2 library
- Supports both password and OAuth2 authentication
- Key methods: `Connect()`, `FetchEmailsSeqNum()`, `ListMailboxes()`

**internal/storage** - SQLite persistence
- Schema includes: accounts, folders, emails tables
- FTS5 full-text search on emails
- Repository pattern in `repository.go`
- Database at `~/.config/miau/data/miau.db`

**internal/tui/inbox** - Main inbox interface
- State machine: `stateInitDB` → `stateConnecting` → `stateLoadingFolders` → `stateSyncing` → `stateReady`
- Handles App Password prompts on authentication failures
- Keyboard: j/k navigate, Tab switch panels, r refresh, a AI assistant, q quit

**internal/tui/setup** - First-run wizard
- Multi-step flow: Welcome → Email → AuthType → IMAP config → Password/OAuth2 → Confirm
- Auto-detects IMAP hosts for common providers

### Data Flow
1. Config loads from YAML
2. SQLite initializes (creates schema if needed)
3. IMAP connects (OAuth2 tokens refresh automatically)
4. Emails sync from server to local SQLite
5. TUI displays from local database

## AI Email Management

When invoked from the miau TUI (pressing 'a'), you have direct access to the SQLite database at `~/.config/miau/data/miau.db`. Use sqlite3 to query and manipulate emails.

### Database Schema

```sql
-- accounts table
CREATE TABLE accounts (
    id INTEGER PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- folders table
CREATE TABLE folders (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    name TEXT NOT NULL,
    total_messages INTEGER DEFAULT 0,
    unread_messages INTEGER DEFAULT 0,
    last_sync DATETIME,
    UNIQUE(account_id, name)
);

-- emails table
CREATE TABLE emails (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    folder_id INTEGER REFERENCES folders(id),
    uid INTEGER NOT NULL,
    message_id TEXT,
    subject TEXT,
    from_name TEXT,
    from_email TEXT,
    to_addresses TEXT,
    cc_addresses TEXT,
    date DATETIME,
    is_read BOOLEAN DEFAULT 0,
    is_starred BOOLEAN DEFAULT 0,
    is_deleted BOOLEAN DEFAULT 0,
    has_attachments BOOLEAN DEFAULT 0,
    snippet TEXT,
    body_text TEXT,
    body_html TEXT,
    raw_headers TEXT,
    size INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(account_id, folder_id, uid)
);

-- FTS5 full-text search
CREATE VIRTUAL TABLE emails_fts USING fts5(subject, from_name, from_email, snippet, body_text, content=emails, content_rowid=id);
```

### Example Queries

```bash
# List unread emails
sqlite3 ~/.config/miau/data/miau.db "SELECT from_email, subject, date FROM emails WHERE is_read=0 ORDER BY date DESC"

# Search emails by sender
sqlite3 ~/.config/miau/data/miau.db "SELECT subject, date FROM emails WHERE from_email LIKE '%newsletter%'"

# Mark emails as read
sqlite3 ~/.config/miau/data/miau.db "UPDATE emails SET is_read=1 WHERE from_email LIKE '%newsletter%'"

# Full-text search
sqlite3 ~/.config/miau/data/miau.db "SELECT e.subject, e.from_email FROM emails e JOIN emails_fts fts ON e.id=fts.rowid WHERE emails_fts MATCH 'invoice'"

# Count by sender
sqlite3 ~/.config/miau/data/miau.db "SELECT from_email, COUNT(*) as cnt FROM emails GROUP BY from_email ORDER BY cnt DESC LIMIT 10"

# Archive (soft delete)
sqlite3 ~/.config/miau/data/miau.db "UPDATE emails SET is_deleted=1 WHERE subject LIKE '%unsubscribe%'"
```

### Important Notes
- Changes to the database are reflected when returning to miau (it reloads from DB)
- Use `is_deleted=1` to soft-delete emails (archive)
- FTS5 table is synced automatically with the emails table

## Code Conventions

- Code and comments in English
- User-facing documentation in Portuguese (pt-BR)
- Commit messages in English
- Use `var` for declarations when possible
- One-line conditionals when possible (no braces for single statements)
- Follow Go conventions (gofmt, go vet)
