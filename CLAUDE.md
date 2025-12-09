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
- Scopes: `mail.google.com` (IMAP) + `gmail.send` (API) + `contacts.readonly` (People API)

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
- People API integration for contacts sync

**internal/services/contact** - Contact management service
- Implements `ports.ContactService` following REGRA DE OURO
- Syncs contacts from Google People API
- Supports full and incremental sync with sync tokens
- Auto-downloads and caches contact profile photos
- Extracts contacts from emails automatically
- Tracks interaction frequency for smart suggestions
- Photos stored at `~/.config/miau/photos/`

**internal/storage** - SQLite persistence
- Schema includes: accounts, folders, emails, contacts, attachments tables
- FTS5 full-text search on emails
- Repository pattern in `repository.go`
- Database at `~/.config/miau/data/miau.db`
- Soft delete: `is_deleted=1` (never hard deletes)
- PurgeDeletedFromServer: syncs deletions from IMAP server
- Contact storage adapter implements `ports.ContactStoragePort`

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

## Contacts Management

### Overview
miau includes a comprehensive contacts system that syncs with Google Contacts via the People API. Contacts are stored locally in SQLite and automatically enriched with interaction data from emails.

### Database Schema

**contacts** - Main contacts table
- `id`, `account_id`, `resource_name` (Google People ID)
- `display_name`, `given_name`, `family_name`
- `photo_url`, `photo_path` (local cache)
- `is_starred`, `interaction_count`, `last_interaction_at`
- `synced_at`, `created_at`, `updated_at`

**contact_emails** - Email addresses (N:N relationship)
- `contact_id`, `email`, `email_type` (home/work/other)
- `is_primary`

**contact_phones** - Phone numbers
- `contact_id`, `phone_number`, `phone_type`
- `is_primary`

**contact_interactions** - Interaction history
- `contact_id`, `email_id`, `interaction_type` (sent/received)
- `interaction_date`

**contacts_sync_state** - Sync tracking
- `account_id`, `last_sync_token`, `last_full_sync`
- `total_contacts`, `status`, `error_message`

### Sync Flow

1. **Initial Sync** (full sync)
   - `ContactService.SyncContacts(accountID, fullSync=true)`
   - Fetches all contacts from Google People API
   - Downloads profile photos in background
   - Stores contacts + emails + phones
   - Saves sync token for incremental updates

2. **Incremental Sync** (delta updates)
   - Uses `syncToken` from previous sync
   - Only fetches changed/new contacts
   - More efficient for regular updates

3. **Automatic Extraction**
   - When emails are synced, extract sender/recipient info
   - Auto-create/update contacts from email headers
   - Track interactions (sent/received) for frequency ranking

### Usage Examples

```go
// Via ContactService (REGRA DE OURO - always use service!)
contactService := app.Contacts()

// Full sync
err := contactService.SyncContacts(ctx, accountID, true)

// Search contacts
contacts, err := contactService.SearchContacts(ctx, accountID, "john", 10)

// Get contact by email
contact, err := contactService.GetContactByEmail(ctx, accountID, "john@example.com")

// Get top frequent contacts
topContacts, err := contactService.GetTopContacts(ctx, accountID, 20)
```

### Photo Storage
- Photos cached at `~/.config/miau/photos/contact_{id}.jpg`
- Downloaded asynchronously during sync
- Photo path stored in `contacts.photo_path`

### Integration Points
- `internal/services/contact.go` - Business logic (REGRA DE OURO)
- `internal/storage/contacts.go` - Storage adapter
- `internal/gmail/api.go` - People API client (`ListContacts`, `GetContact`, `DownloadPhoto`)
- `internal/gmail/contacts_adapter.go` - Adapter to `ports.GmailContactsPort`
- `internal/ports/contact.go` - Service interfaces

### OAuth Scopes Required
- `https://www.googleapis.com/auth/contacts.readonly` - Read-only access to contacts
- `https://www.googleapis.com/auth/contacts.other.readonly` - Access to "Other Contacts" (auto-suggested from emails)
- Automatically included in `internal/auth/oauth2.go`

### Contact Autocomplete (Desktop)
The desktop app includes a contact autocomplete component for composing emails:
- Type 2+ characters in "Para:", "Cc:" or "Bcc:" fields to search contacts
- Contacts are searched by name AND email
- Shows contact photo (if available) and primary email
- Arrow keys to navigate, Enter/Tab to select, Escape to close
- Automatically loads top contacts on focus
- Component: `cmd/miau-desktop/frontend/src/lib/components/ContactAutocomplete.svelte`

### Future Enhancements
- Contact avatar display in email list
- Contact groups/labels sync
- Contact birthday/event reminders
- Merge duplicate contacts detection

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

### Go (Backend)
- Code and comments in English
- User-facing documentation in Portuguese (pt-BR)
- Commit messages in English
- Use `var` for declarations when possible
- One-line conditionals when possible (no braces for single statements)
- Follow Go conventions (gofmt, go vet)

### JavaScript/Svelte (Frontend)
- Use `const` for all declarations (preferred) or `let` when mutation is needed
- NEVER use `var` in JavaScript - it's hoisted and can cause bugs
- Use optional chaining (`?.`) instead of `&&` chains for property access
- Remove unused imports
- Use arrow functions for callbacks
- Follow ES6+ modern patterns

```javascript
// ERRADO
var emails = writable([]);
export var selectedEmail = writable(null);
if (data && data.user && data.user.name) { ... }

// CORRETO
const emails = writable([]);
export const selectedEmail = writable(null);
if (data?.user?.name) { ... }
```

**Nota**: A regra "use var" do arquivo global se aplica apenas a Go, não JavaScript!

## Desktop App (Wails + Svelte)

The desktop app provides a modern GUI alternative to the TUI, built with Wails and Svelte.

### Running
```bash
cd cmd/miau-desktop
wails dev --devtools  # Development mode with hot reload
wails build           # Production build
```

### Features Implemented
- **3-panel layout**: Folders | Email List | Email Viewer
- **Thread view**: Collapsible thread timeline with all messages
- **Multi-select**: Shift+Click, Ctrl+Click for batch operations
- **Contact autocomplete**: Search contacts while composing
- **Contact sync**: Full and incremental sync from Google People API
- **Settings modal**: Configure sync folders, UI preferences
- **Analytics dashboard**: Email statistics and trends
- **Attachments**: View, download, open attachments
- **Keyboard shortcuts**: Same shortcuts as TUI (j/k, c, r, etc.)
- **Undo/Redo**: For email operations

### Architecture
- Frontend: Svelte + TypeScript (in `cmd/miau-desktop/frontend/`)
- Backend: Go bindings via Wails (in `internal/desktop/`)
- Bindings auto-generated in `frontend/src/lib/wailsjs/`

### Key Components
- `App.svelte` - Main app container with layout
- `FolderList.svelte` - Folder navigation sidebar
- `EmailList.svelte` - Email list with selection
- `EmailViewer.svelte` - Email content viewer
- `ComposeModal.svelte` - Email composition with autocomplete
- `ThreadView.svelte` - Thread conversation view
- `ContactAutocomplete.svelte` - Contact search for compose
- `SettingsModal.svelte` - App settings

## Tasks System

The app includes a task management system integrated with emails.

### Database Schema
```sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    is_completed BOOLEAN DEFAULT 0,
    priority INTEGER DEFAULT 0,  -- 0=normal, 1=high, 2=urgent
    due_date DATETIME,
    email_id INTEGER,  -- optional link to email
    source TEXT DEFAULT 'manual',  -- 'manual' or 'ai_suggestion'
    created_at DATETIME,
    updated_at DATETIME
);
```

### Task Sources
- `manual` - Created by user
- `ai_suggestion` - Suggested by AI from email content

### Integration Points
- `internal/services/task.go` - Task business logic
- `internal/storage/tasks.go` - Task storage adapter
- `internal/ports/task.go` - Task service interface

## Debug

- Run with `--debug` flag to show debug panel in TUI
- Desktop: `wails dev --devtools` opens browser DevTools
- Bounce detection logs to `/tmp/miau-bounce.log`
- Sync logs latestUID and found UIDs for troubleshooting
- SQLite: `busy_timeout(5000)` prevents "database is locked" errors
