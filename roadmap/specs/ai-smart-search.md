# AI-11: AI Smart Search (NLP)

## Overview

Enable natural language search queries that understand intent and context, going beyond keyword matching.

## User Stories

1. As a user, I want to search using natural language ("emails from John last week about budget")
2. As a user, I want the search to understand synonyms and related terms
3. As a user, I want to search by intent ("emails I need to respond to")
4. As a user, I want search suggestions as I type

## Technical Requirements

### Service Layer

Create `internal/services/smartsearch.go`:

```go
package services

type SmartSearchService interface {
    // NaturalSearch parses natural language query and searches
    NaturalSearch(ctx context.Context, accountID int64, query string) (*SearchResult, error)

    // ParseQuery converts natural language to structured query
    ParseQuery(ctx context.Context, query string) (*ParsedQuery, error)

    // GetSuggestions returns search suggestions
    GetSuggestions(ctx context.Context, accountID int64, partial string) ([]SearchSuggestion, error)

    // SemanticSearch finds similar emails by meaning
    SemanticSearch(ctx context.Context, emailID int64, limit int) ([]Email, error)
}

type ParsedQuery struct {
    Original    string
    Intent      SearchIntent
    Filters     QueryFilters
    Keywords    []string
    Confidence  float64
    Explanation string
}

type SearchIntent string

const (
    IntentFind        SearchIntent = "find"       // Find specific email
    IntentFilter      SearchIntent = "filter"     // Filter by criteria
    IntentAction      SearchIntent = "action"     // Find actionable emails
    IntentRecent      SearchIntent = "recent"     // Recent interactions
    IntentUnread      SearchIntent = "unread"     // Unread emails
    IntentAttachments SearchIntent = "attachment" // Emails with attachments
)

type QueryFilters struct {
    From        string
    To          string
    Subject     string
    DateFrom    *time.Time
    DateTo      *time.Time
    HasAttachment *bool
    IsUnread    *bool
    IsStarred   *bool
    Folder      string
    Category    string
}

type SearchSuggestion struct {
    Query       string
    Type        string  // "recent", "contact", "folder", "smart"
    Description string
}
```

### AI Prompt Template

```go
var searchParsePrompt = `Parse this natural language search query into structured filters.

Query: "{{.Query}}"

Extract:
- from: sender email/name
- to: recipient
- subject: subject keywords
- date_from: start date
- date_to: end date
- has_attachment: boolean
- is_unread: boolean
- folder: folder name
- keywords: search terms

Examples:
- "emails from john last week" → from:john, date_from:7 days ago
- "unread messages with attachments" → is_unread:true, has_attachment:true
- "invoices from december" → keywords:invoices, date_from:Dec 1, date_to:Dec 31

Output JSON:
{
  "intent": "filter",
  "filters": {
    "from": "john@example.com",
    "date_from": "2024-12-01T00:00:00Z"
  },
  "keywords": ["meeting", "budget"],
  "explanation": "Finding emails from John about meetings since December 1st"
}`
```

### Search Flow

```go
func (s *SmartSearchService) NaturalSearch(ctx context.Context, accountID int64, query string) (*SearchResult, error) {
    // 1. Parse natural language query
    parsed, err := s.ParseQuery(ctx, query)
    if err != nil {
        // Fallback to FTS search
        return s.fallbackSearch(ctx, accountID, query)
    }

    // 2. Build SQL query from parsed filters
    sqlQuery := s.buildQuery(parsed)

    // 3. Execute search
    emails, err := s.storage.Search(ctx, sqlQuery)
    if err != nil {
        return nil, err
    }

    // 4. Rank results by relevance
    ranked := s.rankResults(emails, parsed)

    return &SearchResult{
        Query:       query,
        Parsed:      parsed,
        Emails:      ranked,
        TotalCount:  len(ranked),
        Explanation: parsed.Explanation,
    }, nil
}
```

## UI/UX

### TUI
- Press `/` for search mode
- Natural language suggestions appear while typing
- Parse explanation shown below search bar

```
┌─ Search ──────────────────────────────────────────────────────────┐
│ > emails from john last week about project                        │
├───────────────────────────────────────────────────────────────────┤
│ 🔍 Finding: emails from john@example.com since Dec 4              │
│    Keywords: "project"                                            │
└───────────────────────────────────────────────────────────────────┘

Suggestions:
  • emails from john with attachments
  • emails to john
  • recent emails from john
```

### Desktop
- Search bar with autocomplete dropdown
- Advanced search modal
- Search history
- Saved searches

## Testing

1. Test various natural language queries
2. Test date parsing (relative and absolute)
3. Test contact name resolution
4. Test fallback to keyword search
5. Test performance with large mailboxes
6. Test multilingual queries

## Acceptance Criteria

- [ ] Natural language queries parsed correctly
- [ ] Date expressions understood (yesterday, last week, etc.)
- [ ] Contact names resolved to emails
- [ ] Shows explanation of parsed query
- [ ] Falls back gracefully on parse failure
- [ ] Search suggestions work
- [ ] Performance acceptable (<500ms)

## Configuration

```yaml
# config.yaml
search:
  smart_search: true
  show_explanation: true
  suggestion_count: 5
```

## Estimated Complexity

High - NLP parsing plus search optimization
