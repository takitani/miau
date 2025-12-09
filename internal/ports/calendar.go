package ports

import (
	"context"
	"time"
)

// CalendarService defines the interface for calendar operations
type CalendarService interface {
	// CreateEvent creates a new calendar event
	CreateEvent(ctx context.Context, event *CalendarEventInput) (*CalendarEventInfo, error)

	// GetEvent returns an event by ID
	GetEvent(ctx context.Context, id int64) (*CalendarEventInfo, error)

	// GetEvents returns all events for an account
	GetEvents(ctx context.Context, accountID int64) ([]CalendarEventInfo, error)

	// GetEventsByDateRange returns events within a date range
	GetEventsByDateRange(ctx context.Context, accountID int64, start, end time.Time) ([]CalendarEventInfo, error)

	// GetEventsForWeek returns events for a specific week
	GetEventsForWeek(ctx context.Context, accountID int64, weekStart time.Time) ([]CalendarEventInfo, error)

	// GetUpcomingEvents returns upcoming events from now
	GetUpcomingEvents(ctx context.Context, accountID int64, limit int) ([]CalendarEventInfo, error)

	// GetEventByTask returns the event associated with a task
	GetEventByTask(ctx context.Context, taskID int64) (*CalendarEventInfo, error)

	// GetEventsByEmail returns events linked to an email
	GetEventsByEmail(ctx context.Context, emailID int64) ([]CalendarEventInfo, error)

	// UpdateEvent updates an existing event
	UpdateEvent(ctx context.Context, event *CalendarEventInput) (*CalendarEventInfo, error)

	// ToggleEventCompleted toggles the completed status
	ToggleEventCompleted(ctx context.Context, id int64) (bool, error)

	// DeleteEvent removes an event
	DeleteEvent(ctx context.Context, id int64) error

	// CountEvents returns event counts
	CountEvents(ctx context.Context, accountID int64) (*CalendarEventCounts, error)

	// === Task Sync ===

	// CreateEventFromTask creates a calendar event from a task with due_date
	CreateEventFromTask(ctx context.Context, taskID int64) (*CalendarEventInfo, error)

	// UpdateEventFromTask updates the calendar event when task is modified
	UpdateEventFromTask(ctx context.Context, taskID int64) (*CalendarEventInfo, error)

	// DeleteEventByTask removes the event when task is deleted
	DeleteEventByTask(ctx context.Context, taskID int64) error

	// SyncTasksToCalendar syncs all tasks with due_date to calendar
	SyncTasksToCalendar(ctx context.Context, accountID int64) error

	// === Email Follow-up ===

	// CreateFollowUpEvent creates a follow-up event for an email
	CreateFollowUpEvent(ctx context.Context, emailID int64, followUpTime time.Time, title string) (*CalendarEventInfo, error)
}

// CalendarEventType represents the type of calendar event
type CalendarEventType string

const (
	CalendarEventTypeCustom        CalendarEventType = "custom"
	CalendarEventTypeTaskDeadline  CalendarEventType = "task_deadline"
	CalendarEventTypeEmailFollowup CalendarEventType = "email_followup"
	CalendarEventTypeMeeting       CalendarEventType = "meeting"
)

// CalendarEventSource represents how the event was created
type CalendarEventSource string

const (
	CalendarEventSourceManual       CalendarEventSource = "manual"
	CalendarEventSourceTaskSync     CalendarEventSource = "task_sync"
	CalendarEventSourceAISuggestion CalendarEventSource = "ai_suggestion"
)

// CalendarSyncStatus represents sync status with external calendar
type CalendarSyncStatus string

const (
	CalendarSyncStatusLocal       CalendarSyncStatus = "local"
	CalendarSyncStatusSynced      CalendarSyncStatus = "synced"
	CalendarSyncStatusPendingSync CalendarSyncStatus = "pending_sync"
	CalendarSyncStatusConflict    CalendarSyncStatus = "conflict"
)

// CalendarEventInput represents input for creating/updating an event
type CalendarEventInput struct {
	ID               int64
	AccountID        int64
	Title            string
	Description      string
	EventType        CalendarEventType
	StartTime        time.Time
	EndTime          *time.Time
	AllDay           bool
	Color            string
	TaskID           *int64
	EmailID          *int64
	IsCompleted      bool
	Source           CalendarEventSource
	GoogleEventID    string
	GoogleCalendarID string
	SyncStatus       CalendarSyncStatus
}

// CalendarEventInfo represents event information returned by the service
type CalendarEventInfo struct {
	ID               int64
	AccountID        int64
	Title            string
	Description      string
	EventType        CalendarEventType
	StartTime        time.Time
	EndTime          *time.Time
	AllDay           bool
	Color            string
	TaskID           *int64
	EmailID          *int64
	IsCompleted      bool
	Source           CalendarEventSource
	GoogleEventID    string
	GoogleCalendarID string
	LastSyncedAt     *time.Time
	SyncStatus       CalendarSyncStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CalendarEventCounts represents event count statistics
type CalendarEventCounts struct {
	Upcoming  int
	Completed int
	Total     int
}

// CalendarSyncCallback is used by TaskService to sync with CalendarService
// This avoids circular dependency between the two services
type CalendarSyncCallback interface {
	// OnTaskCreated is called when a task with due_date is created
	OnTaskCreated(ctx context.Context, taskID int64) error

	// OnTaskUpdated is called when a task is updated (including due_date changes)
	OnTaskUpdated(ctx context.Context, taskID int64) error

	// OnTaskDeleted is called when a task is deleted
	OnTaskDeleted(ctx context.Context, taskID int64) error

	// OnTaskCompletedToggled is called when task completion status changes
	OnTaskCompletedToggled(ctx context.Context, taskID int64, isCompleted bool) error
}

// Default event colors by type
var DefaultEventColors = map[CalendarEventType]string{
	CalendarEventTypeCustom:        "#4ecdc4", // Teal
	CalendarEventTypeTaskDeadline:  "#f39c12", // Orange
	CalendarEventTypeEmailFollowup: "#9b59b6", // Purple
	CalendarEventTypeMeeting:       "#3498db", // Blue
}

// GetDefaultColor returns the default color for an event type
func GetDefaultColor(eventType CalendarEventType) string {
	if color, ok := DefaultEventColors[eventType]; ok {
		return color
	}
	return DefaultEventColors[CalendarEventTypeCustom]
}

// GoogleCalendarInfo represents a Google Calendar
type GoogleCalendarInfo struct {
	ID              string
	Summary         string
	Description     string
	Primary         bool
	BackgroundColor string
	AccessRole      string
}

// GoogleEventInfo represents a Google Calendar event
type GoogleEventInfo struct {
	ID          string
	CalendarID  string
	Summary     string
	Description string
	Location    string
	StartTime   time.Time
	EndTime     time.Time
	AllDay      bool
	Status      string
	HtmlLink    string
	ColorID     string
}
