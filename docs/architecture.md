# miau Architecture

## System Overview

```mermaid
graph TB
    subgraph "User Interface"
        TUI[Terminal UI<br/>Bubble Tea]
        CLI[CLI Commands<br/>miau, miau auth]
    end

    subgraph "Core Application"
        SM[State Machine<br/>Setup → Inbox]

        subgraph "TUI Components"
            INBOX[Inbox View]
            COMPOSE[Compose View]
            AI[AI Panel]
            SETTINGS[Settings Menu]
            SEARCH[Fuzzy Search<br/>FTS5 Trigram]
        end
    end

    subgraph "Services"
        IMAP[IMAP Client<br/>go-imap/v2]
        SMTP[SMTP Client<br/>net/smtp]
        GMAIL[Gmail API<br/>REST Client]
        AUTH[OAuth2 Manager<br/>Token Storage]
    end

    subgraph "Data Layer"
        REPO[Repository<br/>CRUD Operations]
        DB[(SQLite<br/>modernc.org/sqlite)]
        FTS[(FTS5 Index<br/>Trigram Search)]
        CONFIG[Config Manager<br/>Viper/YAML]
    end

    subgraph "External Services"
        IMAP_SERVER[IMAP Server<br/>Gmail, Outlook, etc.]
        SMTP_SERVER[SMTP Server]
        GMAIL_API[Gmail API<br/>Google Cloud]
        CLAUDE[Claude Code<br/>AI Assistant]
    end

    TUI --> SM
    CLI --> AUTH
    SM --> INBOX
    SM --> SETTINGS

    INBOX --> COMPOSE
    INBOX --> AI
    INBOX --> SEARCH

    SEARCH --> FTS

    INBOX --> REPO
    COMPOSE --> SMTP
    COMPOSE --> GMAIL
    AI --> CLAUDE

    REPO --> DB
    DB --> FTS

    IMAP --> IMAP_SERVER
    SMTP --> SMTP_SERVER
    GMAIL --> GMAIL_API
    AUTH --> GMAIL_API

    CONFIG --> AUTH
```

## Data Flow

```mermaid
sequenceDiagram
    participant User
    participant TUI
    participant StateM as State Machine
    participant IMAP
    participant SQLite
    participant Claude

    User->>TUI: Start miau
    TUI->>StateM: Initialize
    StateM->>SQLite: Load cached emails
    SQLite-->>TUI: Display immediately

    StateM->>IMAP: Connect (OAuth2/Password)
    IMAP-->>StateM: Connected

    StateM->>IMAP: Fetch new emails (UID > lastUID)
    IMAP-->>StateM: New emails
    StateM->>SQLite: Store emails
    SQLite-->>TUI: Update display

    User->>TUI: Press 'a' (AI)
    TUI->>Claude: Query via Claude Code
    Claude->>SQLite: sqlite3 queries
    SQLite-->>Claude: Results
    Claude-->>TUI: Response

    User->>TUI: Press '/' (Search)
    TUI->>SQLite: FTS5 trigram search
    SQLite-->>TUI: Search results
```

## Component Responsibilities

### TUI Layer (`internal/tui/`)
- **inbox/** - Main email interface with state machine
- **setup/** - First-run configuration wizard

### Service Layer (`internal/`)
- **imap/** - IMAP protocol wrapper (go-imap/v2)
- **smtp/** - Email sending via SMTP
- **gmail/** - Gmail REST API client
- **auth/** - OAuth2 authentication flow

### Data Layer (`internal/storage/`)
- **db.go** - SQLite initialization, schema, migrations
- **models.go** - Data structures (Email, Draft, Account, etc.)
- **repository.go** - CRUD operations, search, batch ops

### Configuration (`internal/config/`)
- YAML-based configuration
- Multi-account support
- OAuth2 credentials management

## State Machine Flow

```mermaid
stateDiagram-v2
    [*] --> stateInitDB: Start
    stateInitDB --> stateConnecting: DB Ready
    stateConnecting --> stateNeedsAppPassword: Auth Failed
    stateConnecting --> stateLoadingFolders: Connected
    stateNeedsAppPassword --> stateConnecting: Password Entered
    stateLoadingFolders --> stateSyncing: Folders Loaded
    stateSyncing --> stateLoadingEmails: Sync Complete
    stateLoadingEmails --> stateReady: Emails Loaded
    stateReady --> stateReady: User Actions
    stateReady --> [*]: Quit
```

## Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| TUI Framework | Bubble Tea + Lip Gloss | Terminal UI rendering |
| Database | SQLite (modernc.org) | Local email storage |
| Full-text Search | FTS5 + Trigram | Fuzzy email search |
| IMAP Client | go-imap/v2 | Email retrieval |
| SMTP Client | net/smtp | Email sending |
| Gmail API | REST + OAuth2 | Gmail integration |
| Config | Viper | YAML configuration |
| AI Integration | Claude Code | Natural language queries |
