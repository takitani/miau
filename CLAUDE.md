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
go run ./cmd/miau --debug       # Run with debug logging
```

## CLI Commands

```bash
miau              # Start TUI
miau --debug      # Start with debug panel
miau auth         # OAuth2 authentication flow
miau signature    # Show configured signature
```

## Architecture

### ⚠️ CRITICAL: Single Source of Truth (NUNCA DUPLICAR LÓGICA!)

**REGRA DE OURO**: TUI e Desktop NUNCA implementam lógica de negócio diretamente.
Toda operação DEVE passar pelos Services centralizados em `internal/services/`.

```
┌─────────────┐     ┌─────────────┐
│     TUI     │     │   Desktop   │
│ (bubbletea) │     │  (wails)    │
└──────┬──────┘     └──────┬──────┘
       │                   │
       └───────┬───────────┘
               │
       ┌───────▼───────┐
       │  Application  │  ← ÚNICO PONTO DE ENTRADA
       │ internal/app  │
       └───────┬───────┘
               │
       ┌───────▼───────┐
       │   Services    │  ← TODA LÓGICA DE NEGÓCIO AQUI
       │   (ports.*)   │
       └───────┬───────┘
               │
    ┌──────────┼──────────┐
    │          │          │
┌───▼───┐  ┌───▼───┐  ┌───▼───┐
│ IMAP  │  │Storage│  │ SMTP  │
│Adapter│  │Adapter│  │Adapter│
└───────┘  └───────┘  └───────┘
```

**PROIBIDO** (TUI/Desktop chamando diretamente):
- ❌ `imap.Client.FetchEmailRaw()`
- ❌ `storage.GetEmails()` para operações complexas
- ❌ `smtp.Send()` diretamente
- ❌ Qualquer parsing/extração de dados duplicado

**CORRETO** (via Application/Services):
- ✅ `app.Email().GetEmail(id)` → retorna email + attachments
- ✅ `app.Email().GetAttachments(emailID)` → retorna attachments
- ✅ `app.Email().Send(email)` → envia via SMTP ou Gmail API
- ✅ `app.Sync().SyncFolder(folder)` → sincroniza pasta

**Por quê?**
- Bug fix em um lugar = corrigido para TUI + Desktop
- Testes unitários nos Services cobrem ambas interfaces
- Evita divergência de comportamento (ex: TUI parseando anexos diferente do Desktop)

### Application Flow (cmd/miau/main.go)
The app uses Bubble Tea as its TUI framework with a state machine pattern:
- `stateSetup` → First-run setup wizard (internal/tui/setup)
- `stateInbox` → Main email interface (internal/tui/inbox)

### Key Packages

**internal/config** - Configuration via Viper
- Config stored at `~/.config/miau/config.yaml`
- Supports multiple accounts with two auth types: `password` or `oauth2`
- SendMethod: `smtp` or `gmail_api` for email sending

**internal/auth** - OAuth2 authentication
- Browser-based OAuth2 flow for Gmail/Google Workspace
- Tokens stored at `~/.config/miau/tokens/{account}.json`
- Implements XOAUTH2 SASL mechanism for IMAP
- Scopes: `mail.google.com` (IMAP) + `gmail.send` (API)

**internal/imap** - IMAP client wrapper
- Wraps go-imap/v2 library
- Supports both password and OAuth2 authentication
- Auto-triggers browser auth when token missing
- Key methods: `Connect()`, `FetchNewEmails()`, `GetAllUIDs()`, `ListMailboxes()`
- Optimized boot: only fetches INBOX status, skips other folders

**internal/smtp** - SMTP client for sending
- Supports PLAIN and LOGIN authentication
- Email classification headers for Google Workspace
- Signature injection (HTML/text)

**internal/gmail** - Gmail API client
- OAuth2-based REST API client
- SendMessage with classificationLabelValues (bypasses DLP)
- GetSignature, GetSendAsConfig for account info

**internal/storage** - SQLite persistence
- Schema includes: accounts, folders, emails tables
- FTS5 full-text search on emails
- Repository pattern in `repository.go`
- Database at `~/.config/miau/data/miau.db`
- Soft delete: `is_deleted=1` (never hard deletes)
- PurgeDeletedFromServer: syncs deletions from IMAP server

**internal/tui/inbox** - Main inbox interface
- State machine: `stateInitDB` → `stateConnecting` → `stateLoadingFolders` → `stateSyncing` → `stateReady`
- Handles App Password prompts on authentication failures
- Bounce detection after sending emails
- Keyboard: j/k navigate, Tab switch panels, r refresh, c compose, a AI assistant, q quit

**internal/tui/setup** - First-run wizard
- Multi-step flow: Welcome → Email → AuthType → IMAP config → Password/OAuth2 → Confirm
- Auto-detects IMAP hosts for common providers

### Data Flow
1. Config loads from YAML
2. SQLite initializes (creates schema if needed)
3. Cache loads from local DB (instant display)
4. IMAP connects (OAuth2 tokens refresh automatically)
5. Folders load (only INBOX status for fast boot)
6. Sync: fetch new emails (UID > lastUID) + purge deleted
7. TUI displays from local database

### Email Sending Flow
1. User composes email (c key)
2. Check `send_method` in config
3. If `gmail_api`: use Gmail REST API with OAuth2
4. If `smtp`: use SMTP with password/OAuth2
5. Track sent email for bounce detection (5 min)
6. Show success/error overlay

## AI Email Management

When invoked from the miau TUI (pressing 'a'), you have direct access to the SQLite database at `~/.config/miau/data/miau.db`. Use sqlite3 to query and manipulate emails.

### Escrevendo Respostas de Email (IMPORTANTE!)

Quando o usuário pedir para responder/escrever um email, você deve retornar **APENAS o corpo do email** - texto puro, pronto para enviar:

**CORRETO:**
```
Oi André,

Tudo bem sim! E você?

Abraço
```

**ERRADO:**
```
**Assunto:** Re: E ai tudo bem?

---

Ok!

---

Quer que eu envie essa resposta?
```

**Regras:**
- NÃO use markdown (**, ---, ##, etc)
- NÃO inclua "Assunto:", "Para:", "De:" no corpo
- NÃO pergunte "Quer que eu envie?" - o sistema já cuida disso
- NÃO inclua explicações ou meta-texto
- APENAS o texto que será enviado no email
- Mantenha o tom solicitado (formal, informal, etc)
- Se o usuário não especificar o tom, use informal em português

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
    is_replied BOOLEAN DEFAULT 0,
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
sqlite3 ~/.config/miau/data/miau.db "SELECT from_email, subject, date FROM emails WHERE is_read=0 AND is_deleted=0 ORDER BY date DESC"

# Search emails by sender
sqlite3 ~/.config/miau/data/miau.db "SELECT subject, date FROM emails WHERE from_email LIKE '%newsletter%' AND is_deleted=0"

# Mark emails as read
sqlite3 ~/.config/miau/data/miau.db "UPDATE emails SET is_read=1 WHERE from_email LIKE '%newsletter%'"

# Full-text search
sqlite3 ~/.config/miau/data/miau.db "SELECT e.subject, e.from_email FROM emails e JOIN emails_fts fts ON e.id=fts.rowid WHERE emails_fts MATCH 'invoice' AND e.is_deleted=0"

# Count by sender
sqlite3 ~/.config/miau/data/miau.db "SELECT from_email, COUNT(*) as cnt FROM emails WHERE is_deleted=0 GROUP BY from_email ORDER BY cnt DESC LIMIT 10"

# Archive (soft delete)
sqlite3 ~/.config/miau/data/miau.db "UPDATE emails SET is_deleted=1 WHERE subject LIKE '%unsubscribe%'"

# Count emails by folder
sqlite3 ~/.config/miau/data/miau.db "SELECT f.name, COUNT(*) FROM emails e JOIN folders f ON e.folder_id=f.id WHERE e.is_deleted=0 GROUP BY f.name"
```

### Important Notes
- Changes to the database are reflected when returning to miau (it reloads from DB)
- Use `is_archived=1` for archiving (hide from inbox, keep email)
- Use `is_deleted=1` for trash (will be permanently archived after 30 days)
- NEVER hard delete - all data is preserved forever
- FTS5 table is synced automatically with the emails table
- Sync detects server deletions and marks local copies as deleted

### Batch Operations (IMPORTANTE!)

Para operações em lote (arquivar, deletar, marcar como lido), **SEMPRE** crie uma `pending_batch_ops`. O TUI automaticamente detecta e mostra o preview na interface principal do inbox.

**Operações suportadas:** `archive`, `delete`, `mark_read`, `mark_unread`

```bash
# Criar operação pendente - TUI mostra preview automaticamente
sqlite3 ~/.config/miau/data/miau.db "
INSERT INTO pending_batch_ops (account_id, operation, description, filter_query, email_ids, email_count, status)
SELECT
    1,
    'archive',
    'Arquivar ' || COUNT(*) || ' emails de zaqueu',
    'from_email LIKE ''%zaqueu%''',
    '[' || GROUP_CONCAT(id) || ']',
    COUNT(*),
    'pending'
FROM emails
WHERE from_email LIKE '%zaqueu%' AND is_archived=0 AND is_deleted=0;
"
```

**Fluxo automático:**
1. Usuário pede: "arquive todos emails do Zaqueu" ou "filtre emails do xpto"
2. AI cria `pending_batch_ops` com status='pending'
3. TUI detecta automaticamente e exibe os emails filtrados na interface principal
4. Banner mostra: "⚡ Arquivar 15 emails de zaqueu | y:confirmar n:cancelar"
5. Usuário navega pelos emails com ↑↓ e confirma com `y` ou cancela com `n`

**Exemplos de queries para filtros:**
```bash
# Por remetente
WHERE from_email LIKE '%newsletter%'

# Por assunto
WHERE subject LIKE '%promoção%'

# Por data (emails antigos)
WHERE date < date('now', '-90 days')

# Não lidos de um remetente
WHERE from_email LIKE '%xpto%' AND is_read=0

# Combinações
WHERE from_email LIKE '%marketing%' AND subject LIKE '%oferta%'
```

### Archive Tables (Permanent Storage)

```sql
-- Emails permanentemente arquivados (após purge do servidor)
SELECT * FROM emails_archive WHERE from_email LIKE '%importante%';

-- Histórico de drafts (enviados, cancelados, deletados)
SELECT * FROM drafts_history ORDER BY archived_at DESC;

-- Todos emails enviados
SELECT * FROM sent_emails ORDER BY sent_at DESC;
```

## Code Conventions

- Code and comments in English
- User-facing documentation in Portuguese (pt-BR)
- Commit messages in English
- Use `var` for declarations when possible
- One-line conditionals when possible (no braces for single statements)
- Follow Go conventions (gofmt, go vet)

## Debug

- Run with `--debug` flag to show debug panel in TUI
- Bounce detection logs to `/tmp/miau-bounce.log`
- Sync logs latestUID and found UIDs for troubleshooting
