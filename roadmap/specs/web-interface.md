# PL-03: Web Interface (HTMX)

## Overview

Create a web-based interface for miau using Go + HTMX for server-side rendering with minimal JavaScript.

## User Stories

1. As a user, I want to access my email from any browser
2. As a user, I want the same features as TUI/Desktop
3. As a user, I want a responsive design for mobile
4. As a user, I want fast page loads without heavy JavaScript

## Technical Requirements

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Web Server (Go)                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   Templ     │  │    HTMX     │  │   Static    │              │
│  │  Templates  │  │  Handlers   │  │   Assets    │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│                         │                                       │
│                         ▼                                       │
│              ┌─────────────────────┐                            │
│              │    Application      │  ← Existing services       │
│              │    (internal/app)   │                            │
│              └─────────────────────┘                            │
└─────────────────────────────────────────────────────────────────┘
```

### Project Structure

```
cmd/miau-web/
├── main.go
├── handlers/
│   ├── inbox.go
│   ├── compose.go
│   ├── settings.go
│   └── auth.go
├── templates/
│   ├── layout.templ
│   ├── inbox.templ
│   ├── email.templ
│   ├── compose.templ
│   └── components/
│       ├── email_list.templ
│       ├── folder_list.templ
│       └── email_viewer.templ
└── static/
    ├── css/
    │   └── styles.css
    └── js/
        └── htmx.min.js
```

### Server Implementation

```go
package main

import (
    "net/http"
    "github.com/takitani/miau/internal/app"
)

func main() {
    // Initialize application (same as TUI/Desktop)
    application, err := app.NewApplication(cfg)
    if err != nil {
        log.Fatal(err)
    }

    // Setup handlers
    h := handlers.New(application)

    mux := http.NewServeMux()

    // HTMX endpoints
    mux.HandleFunc("GET /", h.Index)
    mux.HandleFunc("GET /inbox", h.Inbox)
    mux.HandleFunc("GET /email/{id}", h.ViewEmail)
    mux.HandleFunc("GET /compose", h.ComposeForm)
    mux.HandleFunc("POST /send", h.SendEmail)
    mux.HandleFunc("POST /email/{id}/archive", h.ArchiveEmail)
    mux.HandleFunc("POST /email/{id}/delete", h.DeleteEmail)
    mux.HandleFunc("GET /search", h.Search)
    mux.HandleFunc("GET /folders", h.Folders)
    mux.HandleFunc("GET /settings", h.Settings)

    // Static files
    mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

    // Start server
    log.Printf("Starting web server on :8080")
    http.ListenAndServe(":8080", mux)
}
```

### HTMX Handlers

```go
package handlers

type Handlers struct {
    app *app.Application
}

func (h *Handlers) Inbox(w http.ResponseWriter, r *http.Request) {
    accountID := h.getAccountID(r)
    folder := r.URL.Query().Get("folder")
    if folder == "" {
        folder = "INBOX"
    }

    emails, err := h.app.Email().GetEmails(r.Context(), accountID, folder, 50, 0)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    // Check if HTMX request (partial update)
    if r.Header.Get("HX-Request") == "true" {
        // Return only email list component
        templates.EmailList(emails).Render(r.Context(), w)
        return
    }

    // Full page render
    templates.InboxPage(emails, folder).Render(r.Context(), w)
}

func (h *Handlers) ArchiveEmail(w http.ResponseWriter, r *http.Request) {
    emailID := r.PathValue("id")

    err := h.app.Email().Archive(r.Context(), emailID)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    // Return empty with HX-Trigger for list refresh
    w.Header().Set("HX-Trigger", "email-archived")
    w.WriteHeader(200)
}
```

### Templ Templates

```go
// templates/inbox.templ
package templates

templ InboxPage(emails []Email, currentFolder string) {
    @Layout("Inbox - miau") {
        <div class="flex h-screen">
            <!-- Sidebar -->
            <aside class="w-64 border-r" hx-get="/folders" hx-trigger="load">
                @FolderList(currentFolder)
            </aside>

            <!-- Email List -->
            <main class="flex-1 flex">
                <div class="w-1/3 border-r overflow-y-auto"
                     id="email-list"
                     hx-get="/inbox"
                     hx-trigger="email-archived from:body">
                    @EmailList(emails)
                </div>

                <!-- Email Viewer -->
                <div class="w-2/3 overflow-y-auto" id="email-viewer">
                    <p class="p-4 text-gray-500">Select an email to view</p>
                </div>
            </main>
        </div>
    }
}

templ EmailList(emails []Email) {
    for _, email := range emails {
        <div class="email-item p-3 border-b hover:bg-gray-50 cursor-pointer"
             hx-get={ "/email/" + email.ID }
             hx-target="#email-viewer"
             hx-swap="innerHTML">
            <div class="font-medium">{ email.FromName }</div>
            <div class="text-sm">{ email.Subject }</div>
            <div class="text-xs text-gray-500">{ email.Date }</div>
        </div>
    }
}
```

## UI/UX

### Responsive Layout

```
Desktop (>1024px):
┌────────┬─────────────────┬──────────────────────┐
│Folders │  Email List     │   Email Viewer       │
│        │                 │                      │
│ INBOX  │ ● John Smith    │ From: John Smith     │
│ Sent   │   Project...    │ Subject: Project     │
│ Drafts │ ○ Newsletter    │                      │
│ Trash  │   Weekly...     │ Hi,                  │
│        │ ○ Amazon        │ ...email content...  │
│        │   Order...      │                      │
└────────┴─────────────────┴──────────────────────┘

Mobile (<768px):
┌────────────────────────┐
│ ☰ miau         🔍 ✏️   │
├────────────────────────┤
│ ● John Smith           │
│   Project Update       │
│   Dec 15, 10:30 AM     │
├────────────────────────┤
│ ○ Newsletter           │
│   Weekly Digest        │
│   Dec 14, 8:00 AM      │
├────────────────────────┤
│ ○ Amazon               │
│   Your order shipped   │
│   Dec 13, 2:15 PM      │
└────────────────────────┘
```

### Features

- Real-time updates via HTMX polling or SSE
- Keyboard shortcuts (with htmx-ext-head-support)
- Progressive enhancement (works without JS)
- Dark/light mode support
- Mobile-responsive

## CLI Command

```bash
miau serve --port 8080     # Start web server
miau serve --host 0.0.0.0  # Allow external access
miau serve --tls           # Enable HTTPS
```

## Testing

1. Test all HTMX endpoints
2. Test full page vs partial renders
3. Test mobile responsiveness
4. Test keyboard navigation
5. Test without JavaScript (fallback)
6. Test concurrent users

## Acceptance Criteria

- [ ] Web server starts with `miau serve`
- [ ] Inbox displays emails with HTMX updates
- [ ] Can read individual emails
- [ ] Can compose and send emails
- [ ] Can archive/delete emails
- [ ] Search works
- [ ] Mobile responsive
- [ ] Keyboard shortcuts work
- [ ] Uses existing Application services

## Configuration

```yaml
# config.yaml
web:
  enabled: true
  port: 8080
  host: "127.0.0.1"
  tls:
    enabled: false
    cert: ""
    key: ""
  session_secret: "${MIAU_SESSION_SECRET}"
```

## Estimated Complexity

High - Full web stack but reuses existing services
