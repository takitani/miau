# miau

**M**ail **I**ntelligence **A**ssistant **U**tility - Your local-first email client with AI integration.

> A terminal-based email client powered by Claude AI for intelligent email management, search, and automation.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Claude](https://img.shields.io/badge/AI-Claude-orange?style=flat)](https://claude.ai/code)

## What is miau?

**miau** is a privacy-focused, local-first email client that runs entirely in your terminal. It combines the power of IMAP email synchronization with **Claude AI** to create an intelligent email management experience. All your emails are stored locally in SQLite, giving you full control over your data.

### Key Features

- **AI-Powered Email Management** - Natural language queries, AI-generated responses, and intelligent batch operations via Claude Code
- **Local-First Architecture** - All emails stored in SQLite, works offline, your data stays on your machine
- **Fuzzy Search** - Fast trigram-based full-text search across all emails
- **Terminal UI** - Beautiful TUI built with Bubble Tea, vim-style keybindings
- **Desktop App** - Modern GUI with Wails + Svelte (3-panel layout, threads, multi-select)
- **Multi-Account** - Support for Gmail, Google Workspace, and any IMAP provider
- **Gmail Integration** - OAuth2 authentication, Gmail API for sending (bypasses DLP)
- **Contact Sync** - Sync contacts from Google People API with autocomplete in compose
- **Thread View** - Gmail-style conversation view with collapsible messages

## Why "miau"?

- It's short and easy to type in the terminal
- It has "**AI**" hidden in the middle (m-**ia**-u)
- It sounds like a cat asking for attention... just like your unread emails

## Screenshots

```
┌─ miau   demo@example.com  [INBOX] (15 emails) ───────────────────────────────┐
│ ★ miau Team          │ Welcome to miau!                         │ 12/03 14:30 │
│ ● Maria Silva        │ Re: Q4 2025 Commercial Proposal           │ 12/03 13:30 │
│ ● John Santos        │ Meeting tomorrow at 2pm confirmed         │ 12/03 12:30 │
│   Finance            │ Invoice #12345 - December/2025            │ 12/03 11:30 │
│   Tech Weekly        │ Newsletter: AI News                       │ 12/03 10:30 │
│ ★ Security           │ Alert: Login detected from new device...  │ 12/03 09:30 │
├─ AI ─────────────────────────────────────────────────────────────────────────┤
│  AI: how many unread emails?                                                │
│ > how many unread emails?                                                    │
│                                                                              │
│ You have 5 unread emails in your inbox.                                      │
└──────────────────────────────────────────────────────────────────────────────┘
 ↑↓:navigate  Tab:folders  r:sync  /:search  a:AI  c:compose  S:settings  q:quit
```

## Features

### Email Client
- [x] IMAP connection with multiple accounts
- [x] Local email storage in SQLite
- [x] Configurable sync (last X days or all)
- [x] Full-text fuzzy search with FTS5 trigram
- [x] Server deletion detection
- [x] Gmail-style archive (e: archive, x: trash)
- [x] Permanent data retention (never deletes anything)

### Email Composition
- [x] Send via SMTP with authentication
- [x] Send via Gmail API (bypasses DLP/classification)
- [x] Configurable HTML and text signatures
- [x] Email classification (Google Workspace)
- [x] Bounce detection after sending

### Terminal UI (TUI)
- [x] Folder/label navigation
- [x] Email list with indicators (read/unread/starred)
- [x] Vim-style keyboard shortcuts (j/k)
- [x] Email body viewer (HTML opens in browser)
- [x] Email composition and replies
- [x] Integrated AI panel
- [x] Settings menu with indexer controls

### Authentication
- [x] Login with password/App Password
- [x] OAuth2 for Gmail/Google Workspace
- [x] `miau auth` command for token management

### Desktop App (Wails + Svelte)
- [x] 3-panel layout (folders, emails, viewer)
- [x] Thread view with collapsible messages
- [x] Multi-select with batch operations
- [x] Contact sync from Google People API
- [x] Contact autocomplete in compose
- [x] Analytics dashboard
- [x] Settings modal
- [x] Undo/Redo for email operations

### AI Integration (via Claude Code)
- [x] Integrated chat in TUI (press `a`)
- [x] Natural language database queries
- [x] AI-generated draft responses
- [x] Batch operations with preview (archive/delete multiple)
- [ ] Email summarization
- [ ] Automatic categorization

## Documentation

- [Roadmap](docs/ROADMAP.md) - Development progress and feature queue
- [Ideas & Features](docs/IDEAS.md) - Future features and proposals
- [System Architecture](docs/architecture.md) - Component diagrams and data flow
- [Database Schema](docs/database.md) - ERD and table descriptions

## Technology Stack

- **Language**: Go
- **TUI**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) (Charm.sh)
- **Database**: SQLite ([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite))
- **Search**: FTS5 with trigram tokenizer
- **IMAP**: [go-imap/v2](https://github.com/emersion/go-imap)
- **SMTP**: net/smtp with PLAIN/LOGIN auth
- **Gmail API**: REST API for sending (DLP bypass)
- **Config**: [Viper](https://github.com/spf13/viper)
- **AI**: [Claude Code](https://claude.ai/code) integration

## Requirements

- **Go** 1.21+
- **Claude Code** - Claude CLI for AI integration ([install](https://claude.ai/code))
- **sqlite3** - SQLite driver for CLI queries

```bash
# Fedora/RHEL
sudo dnf install sqlite

# Ubuntu/Debian
sudo apt install sqlite3

# macOS
brew install sqlite3

# Windows (via winget)
winget install SQLite.SQLite

# Windows (via choco)
choco install sqlite
```

## Installation

```bash
git clone https://github.com/takitani/miau.git
cd miau
go build ./cmd/miau/
./miau
```

## Usage

### Terminal UI (TUI)
```bash
# Run main TUI
miau

# Run in debug mode
miau --debug

# OAuth2 authentication (for Gmail API)
miau auth

# Show configured signature
miau signature
```

### Desktop App
```bash
cd cmd/miau-desktop

# Development mode (hot reload)
wails dev --devtools

# Build production binary
wails build
./build/bin/miau-desktop
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate list |
| `Enter` | Open email in browser |
| `Tab` | Toggle folder panel |
| `/` | Fuzzy search |
| `c` | Compose new email |
| `r` | Sync emails |
| `a` | Open AI panel |
| `d` | View pending drafts |
| `e` | Archive email |
| `x` or `#` | Move to trash |
| `i` | Image preview (in viewer) |
| `S` | Open settings |
| `q` | Quit |

### Configuration

Configuration file is at `~/.config/miau/config.yaml`:

```yaml
accounts:
  - name: my-account
    email: user@example.com
    auth_type: oauth2  # or "password"
    oauth2:
      client_id: "your-client-id.apps.googleusercontent.com"
      client_secret: "your-client-secret"
    send_method: gmail_api  # or "smtp"
    imap:
      host: imap.gmail.com
      port: 993
      tls: true
    signature:
      enabled: true
      html: |
        <p>Best regards,<br>Your Name</p>
      text: |
        Best regards,
        Your Name
sync:
  interval: 5m
  initial_days: 30
ui:
  theme: dark
  page_size: 50
compose:
  format: html
```

## Gmail API vs SMTP

miau supports two sending methods:

| Method | Advantages | Disadvantages |
|--------|------------|---------------|
| **SMTP** | Works with any provider | May have DLP/classification issues |
| **Gmail API** | DLP bypass, better integration | Requires OAuth2, Google only |

To use Gmail API, set `send_method: gmail_api` and run `miau auth` to authenticate.

## Google Cloud Setup (Gmail API / OAuth2)

To use miau with Gmail API, OAuth2 authentication, Google Contacts sync, or Google Calendar sync, you need to create a Google Cloud project and enable the required APIs.

### Step 1: Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Note down your **Project ID** (you'll see it in the URL and dashboard)

### Step 2: Enable Required APIs

Enable the following APIs in your project (APIs & Services > Library):

| API | Required For | Link |
|-----|--------------|------|
| **Gmail API** | Sending emails, reading signatures | [Enable Gmail API](https://console.cloud.google.com/apis/library/gmail.googleapis.com) |
| **People API** | Contact sync, autocomplete | [Enable People API](https://console.cloud.google.com/apis/library/people.googleapis.com) |
| **Google Calendar API** | Calendar sync | [Enable Calendar API](https://console.cloud.google.com/apis/library/calendar-json.googleapis.com) |

**Important**: If you see a "403 - API not enabled" error, click the link in the error message to enable the API, wait a few minutes, then retry.

### Step 3: Configure OAuth Consent Screen

1. Go to **APIs & Services > OAuth consent screen**
2. Choose **External** (or Internal if using Google Workspace)
3. Fill in the required fields:
   - App name: `miau` (or any name you prefer)
   - User support email: your email
   - Developer contact: your email
4. Add the following **Scopes**:
   - `https://mail.google.com/` - Full IMAP access
   - `https://www.googleapis.com/auth/gmail.send` - Send emails
   - `https://www.googleapis.com/auth/contacts.readonly` - Read contacts
   - `https://www.googleapis.com/auth/contacts.other.readonly` - Read "Other Contacts"
   - `https://www.googleapis.com/auth/calendar.readonly` - Read calendar events
5. Add your email as a **Test user** (required while app is in "Testing" status)
6. Save and continue

### Step 4: Create OAuth Credentials

1. Go to **APIs & Services > Credentials**
2. Click **Create Credentials > OAuth client ID**
3. Application type: **Desktop app**
4. Name: `miau-desktop` (or any name)
5. Click **Create**
6. Download or copy the **Client ID** and **Client Secret**

### Step 5: Configure miau

Add the credentials to your `~/.config/miau/config.yaml`:

```yaml
accounts:
  - name: Your Name
    email: your.email@gmail.com
    auth_type: oauth2
    oauth2:
      client_id: "YOUR_CLIENT_ID.apps.googleusercontent.com"
      client_secret: "YOUR_CLIENT_SECRET"
    send_method: gmail_api
    imap:
      host: imap.gmail.com
      port: 993
      tls: true
```

### Step 6: Authenticate

Run `miau auth` to complete the OAuth2 flow:

```bash
miau auth
```

This will:
1. Open your browser to Google's consent screen
2. Ask you to grant permissions for the enabled APIs
3. Save the token locally at `~/.config/miau/tokens/your.email@gmail.com.json`

### Troubleshooting

| Error | Solution |
|-------|----------|
| `403: Access Not Configured` | Enable the required API in Google Cloud Console |
| `403: The caller does not have permission` | Add yourself as a test user in OAuth consent screen |
| `Token has been revoked` | Delete `~/.config/miau/tokens/*.json` and run `miau auth` again |
| `redirect_uri_mismatch` | Ensure you created a "Desktop app" OAuth client, not "Web application" |

### Features by API

| Feature | Required API |
|---------|--------------|
| OAuth2 login (IMAP) | Gmail API |
| Send via Gmail API | Gmail API |
| Contact autocomplete | People API |
| Contact sync | People API |
| Google Calendar sync | Calendar API |

## AI Commands

When the AI panel is open (press `a`), you can ask natural language questions about your emails:

```
> how many unread emails do I have?
> show me emails from newsletter@example.com
> archive all promotional emails older than 30 days
> draft a reply to the last email from John
> summarize my inbox by sender
```

The AI has direct access to your local SQLite database and can perform complex queries and batch operations with preview before execution.

## Privacy & Security

- **Local-first**: All emails stored locally in SQLite
- **No cloud sync**: Your data never leaves your machine
- **OAuth2**: Secure token-based authentication for Gmail
- **Permanent retention**: Deleted emails are archived, never truly deleted
- **AI integration**: Claude Code runs locally, queries your local database

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT

---

*Built for developers who want control over their email with AI assistance.*

## Keywords

email client, terminal email, TUI email, AI email assistant, Claude email, email automation, IMAP client, Gmail client, local email, privacy email, email agent, AI email management, Claude Code integration, intelligent email, email search, fuzzy search email
