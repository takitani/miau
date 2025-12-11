# PL-05: REST API Server

## Overview

Expose miau functionality via REST API for integration with external tools and services.

## User Stories

1. As a developer, I want to integrate miau with other applications
2. As a user, I want to use Alfred/Raycast plugins
3. As a user, I want browser extensions
4. As a developer, I want webhook notifications for new emails

## Technical Requirements

### API Endpoints

```
Authentication:
POST   /api/auth/token          # Get API token

Emails:
GET    /api/emails              # List emails
GET    /api/emails/:id          # Get email
POST   /api/emails              # Send email
DELETE /api/emails/:id          # Delete email
POST   /api/emails/:id/archive  # Archive email
POST   /api/emails/:id/star     # Toggle star
POST   /api/emails/:id/read     # Mark as read

Folders:
GET    /api/folders             # List folders
GET    /api/folders/:name/emails # Get folder emails

Search:
GET    /api/search?q=<query>    # Search emails

Contacts:
GET    /api/contacts            # List contacts
GET    /api/contacts/:id        # Get contact

Sync:
POST   /api/sync                # Trigger sync

Webhooks:
POST   /api/webhooks            # Register webhook
DELETE /api/webhooks/:id        # Remove webhook
GET    /api/webhooks            # List webhooks
```

### Server Implementation

```go
package api

import (
    "encoding/json"
    "net/http"
    "github.com/takitani/miau/internal/app"
)

type Server struct {
    app      *app.Application
    webhooks *WebhookManager
}

func NewServer(app *app.Application) *Server {
    return &Server{
        app:      app,
        webhooks: NewWebhookManager(),
    }
}

func (s *Server) Routes() http.Handler {
    mux := http.NewServeMux()

    // Auth middleware
    auth := s.authMiddleware

    // Email endpoints
    mux.HandleFunc("GET /api/emails", auth(s.listEmails))
    mux.HandleFunc("GET /api/emails/{id}", auth(s.getEmail))
    mux.HandleFunc("POST /api/emails", auth(s.sendEmail))
    mux.HandleFunc("DELETE /api/emails/{id}", auth(s.deleteEmail))
    mux.HandleFunc("POST /api/emails/{id}/archive", auth(s.archiveEmail))
    mux.HandleFunc("POST /api/emails/{id}/star", auth(s.starEmail))
    mux.HandleFunc("POST /api/emails/{id}/read", auth(s.markRead))

    // Search
    mux.HandleFunc("GET /api/search", auth(s.search))

    // Folders
    mux.HandleFunc("GET /api/folders", auth(s.listFolders))

    // Contacts
    mux.HandleFunc("GET /api/contacts", auth(s.listContacts))

    // Sync
    mux.HandleFunc("POST /api/sync", auth(s.triggerSync))

    // Webhooks
    mux.HandleFunc("GET /api/webhooks", auth(s.listWebhooks))
    mux.HandleFunc("POST /api/webhooks", auth(s.createWebhook))
    mux.HandleFunc("DELETE /api/webhooks/{id}", auth(s.deleteWebhook))

    return mux
}

// Email handlers
func (s *Server) listEmails(w http.ResponseWriter, r *http.Request) {
    folder := r.URL.Query().Get("folder")
    if folder == "" {
        folder = "INBOX"
    }
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 {
        limit = 50
    }

    emails, err := s.app.Email().GetEmails(r.Context(), accountID, folder, limit, 0)
    if err != nil {
        s.error(w, err, 500)
        return
    }

    s.json(w, emails)
}

func (s *Server) sendEmail(w http.ResponseWriter, r *http.Request) {
    var req SendEmailRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        s.error(w, err, 400)
        return
    }

    email := &Email{
        To:      req.To,
        Cc:      req.Cc,
        Subject: req.Subject,
        Body:    req.Body,
    }

    if err := s.app.Send().SendEmail(r.Context(), email); err != nil {
        s.error(w, err, 500)
        return
    }

    s.json(w, map[string]bool{"success": true})
}
```

### Authentication

```go
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if token == "" {
            token = r.URL.Query().Get("api_key")
        }

        if token == "" {
            s.error(w, errors.New("missing API key"), 401)
            return
        }

        // Validate token
        if !s.validateToken(token) {
            s.error(w, errors.New("invalid API key"), 401)
            return
        }

        next(w, r)
    }
}
```

### Webhooks

```go
type Webhook struct {
    ID        string
    URL       string
    Events    []string  // "email.received", "email.sent", etc.
    Secret    string
    CreatedAt time.Time
}

func (s *Server) notifyWebhooks(event string, data interface{}) {
    webhooks := s.webhooks.GetByEvent(event)

    for _, wh := range webhooks {
        go func(w Webhook) {
            payload := WebhookPayload{
                Event:     event,
                Timestamp: time.Now(),
                Data:      data,
            }

            body, _ := json.Marshal(payload)

            req, _ := http.NewRequest("POST", w.URL, bytes.NewReader(body))
            req.Header.Set("Content-Type", "application/json")
            req.Header.Set("X-Miau-Signature", s.sign(body, w.Secret))

            http.DefaultClient.Do(req)
        }(wh)
    }
}
```

## API Response Format

### Success Response

```json
{
  "data": {
    "id": "123",
    "from": "john@example.com",
    "subject": "Project Update",
    "body": "...",
    "date": "2024-12-15T10:30:00Z"
  }
}
```

### Error Response

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Email not found"
  }
}
```

### List Response

```json
{
  "data": [...],
  "meta": {
    "total": 150,
    "limit": 50,
    "offset": 0
  }
}
```

## CLI Command

```bash
miau api --port 9090          # Start API server
miau api --token              # Generate API token
miau api --list-tokens        # List active tokens
miau api --revoke <token>     # Revoke token
```

## Testing

1. Test all endpoints
2. Test authentication
3. Test error responses
4. Test webhooks
5. Test rate limiting
6. Test concurrent requests

## Acceptance Criteria

- [ ] REST API starts with `miau api`
- [ ] Authentication via API key
- [ ] All CRUD operations work
- [ ] Search endpoint works
- [ ] Webhooks deliver notifications
- [ ] Good error messages
- [ ] Rate limiting implemented
- [ ] API documentation available

## Configuration

```yaml
# config.yaml
api:
  enabled: true
  port: 9090
  host: "127.0.0.1"
  rate_limit: 100  # requests per minute
  cors_origins: ["http://localhost:3000"]
```

## Estimated Complexity

Medium-High - Full REST API with webhooks
