# AI-15: AI Meeting Notes Extraction

## Overview

Extract meeting details, notes, and action items from meeting-related emails (invites, recaps, notes).

## User Stories

1. As a user, I want meeting details extracted from calendar invites
2. As a user, I want meeting notes summarized and action items extracted
3. As a user, I want meeting prep info consolidated before meetings
4. As a user, I want post-meeting follow-ups tracked

## Technical Requirements

### Service Layer

Create `internal/services/meetingnotes.go`:

```go
package services

type MeetingNotesService interface {
    // ExtractMeetingInfo extracts info from meeting-related email
    ExtractMeetingInfo(ctx context.Context, emailID int64) (*MeetingInfo, error)

    // GetUpcomingMeetings returns meetings from emails
    GetUpcomingMeetings(ctx context.Context, accountID int64) ([]MeetingInfo, error)

    // GetMeetingPrep consolidates info for upcoming meeting
    GetMeetingPrep(ctx context.Context, meetingID int64) (*MeetingPrep, error)

    // TrackFollowUp tracks post-meeting action items
    TrackFollowUp(ctx context.Context, meetingID int64, items []ActionItem) error
}

type MeetingInfo struct {
    ID           int64
    EmailID      int64
    Title        string
    DateTime     time.Time
    Duration     time.Duration
    Location     string
    MeetingLink  string
    Organizer    string
    Attendees    []string
    Agenda       []string
    Notes        string
    ActionItems  []ActionItem
    Type         MeetingType
}

type MeetingType string

const (
    MeetingInvite  MeetingType = "invite"
    MeetingUpdate  MeetingType = "update"
    MeetingNotes   MeetingType = "notes"
    MeetingRecap   MeetingType = "recap"
    MeetingCancel  MeetingType = "cancel"
)

type MeetingPrep struct {
    Meeting       MeetingInfo
    RelatedEmails []Email
    PreviousNotes []MeetingInfo
    ActionItems   []ActionItem
    Agenda        []string
    Context       string  // AI-generated context summary
}
```

### Database Schema

```sql
CREATE TABLE meeting_info (
    id INTEGER PRIMARY KEY,
    email_id INTEGER REFERENCES emails(id),
    title TEXT NOT NULL,
    meeting_datetime DATETIME,
    duration_minutes INTEGER,
    location TEXT,
    meeting_link TEXT,
    organizer TEXT,
    attendees TEXT,  -- JSON array
    agenda TEXT,  -- JSON array
    notes TEXT,
    meeting_type TEXT,
    extracted_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE meeting_action_items (
    id INTEGER PRIMARY KEY,
    meeting_id INTEGER REFERENCES meeting_info(id),
    description TEXT NOT NULL,
    assignee TEXT,
    due_date DATETIME,
    status TEXT DEFAULT 'pending'
);
```

### AI Prompt Template

```go
var meetingPrompt = `Extract meeting information from this email.

From: {{.From}}
Subject: {{.Subject}}
Body: {{.Body}}

Identify:
1. Meeting type (invite, notes, recap, update, cancel)
2. Date, time, duration
3. Location or meeting link
4. Attendees
5. Agenda items
6. Action items with assignees
7. Key decisions (if notes/recap)

Output JSON:
{
  "type": "notes",
  "title": "Weekly Standup",
  "datetime": "2024-12-15T10:00:00Z",
  "duration_minutes": 30,
  "location": "Zoom",
  "meeting_link": "https://zoom.us/j/123",
  "organizer": "john@example.com",
  "attendees": ["john@example.com", "jane@example.com"],
  "agenda": ["Sprint review", "Blockers"],
  "notes": "Discussed Q4 priorities...",
  "action_items": [
    {
      "description": "Send updated timeline",
      "assignee": "Jane",
      "due_date": "2024-12-17"
    }
  ]
}`
```

## UI/UX

### TUI
- Meeting icon in email list for detected meetings
- Press `M` for meeting prep view

```
┌─ Meeting Prep: Weekly Standup ────────────────────────────────────┐
│ 📅 Tomorrow, Dec 15 at 10:00 AM (30 min)                          │
│ 📍 Zoom: https://zoom.us/j/123                                    │
│                                                                   │
│ Attendees:                                                        │
│ • John Smith (organizer)                                          │
│ • Jane Doe                                                        │
│ • You                                                             │
│                                                                   │
│ Agenda:                                                           │
│ 1. Sprint review                                                  │
│ 2. Blockers discussion                                            │
│                                                                   │
│ Related Context:                                                  │
│ • Last meeting: Discussed Q4 priorities                           │
│ • Your pending action: Send updated timeline (due Dec 17)         │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Meeting card view
- Calendar integration
- Meeting history
- Action item tracking

## Acceptance Criteria

- [ ] Detects meeting-related emails
- [ ] Extracts meeting details from invites
- [ ] Summarizes meeting notes
- [ ] Extracts action items from recaps
- [ ] Links meetings across emails
- [ ] Shows meeting prep before meetings
- [ ] Integrates with calendar service

## Estimated Complexity

Medium - Specialized extraction plus calendar integration
