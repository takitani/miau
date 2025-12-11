# EM-13: Focus Mode

## Overview

A distraction-free mode that hides non-essential emails and notifications to help users concentrate.

## User Stories

1. As a user, I want to hide newsletters and promotions while working
2. As a user, I want to only see emails from specific contacts in focus mode
3. As a user, I want to schedule focus mode times
4. As a user, I want focus mode stats (emails blocked, time saved)

## Technical Requirements

### Service Layer

Create `internal/services/focusmode.go`:

```go
package services

type FocusModeService interface {
    // EnableFocusMode activates focus mode
    EnableFocusMode(ctx context.Context, accountID int64, opts FocusOptions) error

    // DisableFocusMode deactivates focus mode
    DisableFocusMode(ctx context.Context, accountID int64) error

    // GetFocusModeStatus returns current status
    GetFocusModeStatus(ctx context.Context, accountID int64) (*FocusStatus, error)

    // GetFilteredInbox returns emails visible in focus mode
    GetFilteredInbox(ctx context.Context, accountID int64) ([]Email, error)

    // ScheduleFocusMode sets up recurring focus times
    ScheduleFocusMode(ctx context.Context, accountID int64, schedule FocusSchedule) error

    // GetFocusStats returns focus mode statistics
    GetFocusStats(ctx context.Context, accountID int64) (*FocusStats, error)
}

type FocusOptions struct {
    AllowVIP          bool
    AllowCategories   []string  // Empty = allow all
    BlockCategories   []string  // Categories to hide
    AllowContacts     []string  // Specific emails to allow
    BlockNotifications bool
    Duration          *time.Duration  // Auto-disable after duration
}

type FocusStatus struct {
    Active        bool
    StartedAt     *time.Time
    EndsAt        *time.Time
    Options       FocusOptions
    EmailsHidden  int
    EmailsVisible int
}

type FocusSchedule struct {
    Enabled   bool
    Days      []time.Weekday  // e.g., [Monday, Tuesday, ...]
    StartTime string          // "09:00"
    EndTime   string          // "12:00"
    Options   FocusOptions
}

type FocusStats struct {
    TotalFocusTime    time.Duration
    EmailsBlocked     int
    SessionCount      int
    AvgSessionLength  time.Duration
    MostBlockedSenders []SenderStat
}

type SenderStat struct {
    Email      string
    Name       string
    BlockCount int
}
```

### Database Schema

```sql
CREATE TABLE focus_mode (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id) UNIQUE,
    is_active BOOLEAN DEFAULT 0,
    started_at DATETIME,
    ends_at DATETIME,
    options TEXT  -- JSON FocusOptions
);

CREATE TABLE focus_schedule (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    enabled BOOLEAN DEFAULT 1,
    days TEXT,  -- JSON array of weekdays
    start_time TEXT,
    end_time TEXT,
    options TEXT  -- JSON FocusOptions
);

CREATE TABLE focus_stats (
    id INTEGER PRIMARY KEY,
    account_id INTEGER REFERENCES accounts(id),
    session_start DATETIME,
    session_end DATETIME,
    emails_blocked INTEGER DEFAULT 0
);
```

### Filter Logic

```go
func (s *FocusModeService) GetFilteredInbox(ctx context.Context, accountID int64) ([]Email, error) {
    status, err := s.GetFocusModeStatus(ctx, accountID)
    if err != nil {
        return nil, err
    }

    if !status.Active {
        // Return normal inbox
        return s.emailService.GetInbox(ctx, accountID)
    }

    // Build filters based on focus options
    opts := status.Options

    filters := EmailFilters{}

    // Allow VIP emails
    if opts.AllowVIP {
        vips, _ := s.vipService.GetVIPs(ctx, accountID)
        for _, vip := range vips {
            filters.AllowFromEmails = append(filters.AllowFromEmails, vip.Email)
        }
    }

    // Allow specific contacts
    filters.AllowFromEmails = append(filters.AllowFromEmails, opts.AllowContacts...)

    // Block specific categories
    if len(opts.BlockCategories) > 0 {
        filters.ExcludeCategories = opts.BlockCategories
    }

    return s.emailService.GetFilteredInbox(ctx, accountID, filters)
}
```

## UI/UX

### TUI
- Press `Ctrl+F` to toggle focus mode
- Indicator in status bar

```
┌─ FOCUS MODE ──────────────────────────────────────────────────────┐
│ 🎯 Focus Mode Active                                              │
│ Showing: VIP + Work emails only                                   │
│ Hidden: 45 emails (Newsletters, Promotions, Social)               │
│ Time remaining: 1h 30m                                            │
├───────────────────────────────────────────────────────────────────┤
│ ★ Boss           │ URGENT: Board meeting     │ 10:30 AM          │
│   John (Work)    │ Project update            │ 09:45 AM          │
│ ★ Client         │ Contract ready            │ 09:00 AM          │
│   Team           │ Sprint review             │ Yesterday         │
└───────────────────────────────────────────────────────────────────┘
  [Ctrl+F] Exit focus  [s] Settings  [v] View all (temp)
```

### Focus Mode Settings

```
┌─ Focus Mode Settings ─────────────────────────────────────────────┐
│                                                                   │
│ Quick Presets:                                                    │
│ [1] Work Only (VIP + Work category)                               │
│ [2] VIP Only (Just important contacts)                            │
│ [3] Custom...                                                     │
│                                                                   │
│ Duration:                                                         │
│ [1h] [2h] [4h] [Until I disable] [Custom...]                      │
│                                                                   │
│ Schedule:                                                         │
│ [ ] Mon-Fri 9:00 AM - 12:00 PM                                    │
│ [ ] Mon-Fri 2:00 PM - 5:00 PM                                     │
│                                                                   │
│ Options:                                                          │
│ [✓] Allow VIP emails                                              │
│ [✓] Block notifications for hidden emails                         │
│ [ ] Auto-enable on schedule                                       │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘
```

### Desktop
- Focus mode toggle in toolbar
- Timer display
- Settings modal
- Statistics view

## Testing

1. Test filter logic for various configurations
2. Test VIP passthrough
3. Test scheduled activation
4. Test auto-disable after duration
5. Test notification blocking
6. Test statistics tracking

## Acceptance Criteria

- [ ] Can enable/disable focus mode
- [ ] VIP emails pass through when enabled
- [ ] Can block specific categories
- [ ] Can allow specific contacts
- [ ] Duration-based auto-disable works
- [ ] Scheduled focus mode works
- [ ] Statistics tracked
- [ ] Works in TUI and Desktop

## Configuration

```yaml
# config.yaml
focus_mode:
  enabled: true
  default_preset: "work_only"
  presets:
    work_only:
      allow_vip: true
      allow_categories: ["Work"]
      block_categories: ["Newsletters", "Promotions", "Social"]
    vip_only:
      allow_vip: true
      block_categories: ["*"]
```

## Estimated Complexity

Medium - Filtering logic plus scheduling
