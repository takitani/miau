package storage

import (
	"database/sql"
	"time"
)

// === CALENDAR EVENTS ===

// CreateCalendarEvent creates a new calendar event
func CreateCalendarEvent(event *CalendarEvent) error {
	var result, err = db.Exec(`
		INSERT INTO calendar_events (
			account_id, title, description, event_type, start_time, end_time,
			all_day, color, task_id, email_id, is_completed, source,
			google_event_id, google_calendar_id, sync_status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.AccountID, event.Title, event.Description, event.EventType,
		event.StartTime, event.EndTime, event.AllDay, event.Color,
		event.TaskID, event.EmailID, event.IsCompleted, event.Source,
		event.GoogleEventID, event.GoogleCalendarID, event.SyncStatus)
	if err != nil {
		return err
	}

	var id, _ = result.LastInsertId()
	event.ID = id
	event.CreatedAt = SQLiteTime{time.Now()}
	event.UpdatedAt = SQLiteTime{time.Now()}
	return nil
}

// GetCalendarEvent returns an event by ID
func GetCalendarEvent(id int64) (*CalendarEvent, error) {
	var event CalendarEvent
	err := db.Get(&event, "SELECT * FROM calendar_events WHERE id = ?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}

// GetCalendarEvents returns all events for an account
func GetCalendarEvents(accountID int64) ([]CalendarEvent, error) {
	var events []CalendarEvent
	err := db.Select(&events, `
		SELECT * FROM calendar_events
		WHERE account_id = ?
		ORDER BY start_time ASC`,
		accountID)
	return events, err
}

// GetCalendarEventsByDateRange returns events within a date range
func GetCalendarEventsByDateRange(accountID int64, start, end time.Time) ([]CalendarEvent, error) {
	var events []CalendarEvent
	err := db.Select(&events, `
		SELECT * FROM calendar_events
		WHERE account_id = ?
		  AND start_time >= ?
		  AND start_time < ?
		ORDER BY start_time ASC`,
		accountID, start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))
	return events, err
}

// GetCalendarEventsForWeek returns events for a specific week (starting Monday)
func GetCalendarEventsForWeek(accountID int64, weekStart time.Time) ([]CalendarEvent, error) {
	var weekEnd = weekStart.AddDate(0, 0, 7)
	return GetCalendarEventsByDateRange(accountID, weekStart, weekEnd)
}

// GetCalendarEventByTask returns the event associated with a task
func GetCalendarEventByTask(taskID int64) (*CalendarEvent, error) {
	var event CalendarEvent
	err := db.Get(&event, `
		SELECT * FROM calendar_events
		WHERE task_id = ?`,
		taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &event, nil
}

// GetCalendarEventsByEmail returns events associated with an email
func GetCalendarEventsByEmail(emailID int64) ([]CalendarEvent, error) {
	var events []CalendarEvent
	err := db.Select(&events, `
		SELECT * FROM calendar_events
		WHERE email_id = ?
		ORDER BY start_time ASC`,
		emailID)
	return events, err
}

// GetUpcomingCalendarEvents returns upcoming events (from now)
func GetUpcomingCalendarEvents(accountID int64, limit int) ([]CalendarEvent, error) {
	var events []CalendarEvent
	var now = time.Now().Format("2006-01-02 15:04:05")
	err := db.Select(&events, `
		SELECT * FROM calendar_events
		WHERE account_id = ?
		  AND start_time >= ?
		  AND is_completed = 0
		ORDER BY start_time ASC
		LIMIT ?`,
		accountID, now, limit)
	return events, err
}

// GetCalendarEventsByType returns events of a specific type
func GetCalendarEventsByType(accountID int64, eventType CalendarEventType) ([]CalendarEvent, error) {
	var events []CalendarEvent
	err := db.Select(&events, `
		SELECT * FROM calendar_events
		WHERE account_id = ? AND event_type = ?
		ORDER BY start_time ASC`,
		accountID, eventType)
	return events, err
}

// UpdateCalendarEvent updates an existing event
func UpdateCalendarEvent(event *CalendarEvent) error {
	_, err := db.Exec(`
		UPDATE calendar_events SET
			title = ?,
			description = ?,
			event_type = ?,
			start_time = ?,
			end_time = ?,
			all_day = ?,
			color = ?,
			task_id = ?,
			email_id = ?,
			is_completed = ?,
			source = ?,
			google_event_id = ?,
			google_calendar_id = ?,
			last_synced_at = ?,
			sync_status = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		event.Title, event.Description, event.EventType,
		event.StartTime, event.EndTime, event.AllDay, event.Color,
		event.TaskID, event.EmailID, event.IsCompleted, event.Source,
		event.GoogleEventID, event.GoogleCalendarID, event.LastSyncedAt, event.SyncStatus,
		event.ID)
	return err
}

// ToggleCalendarEventCompleted toggles the completed status
func ToggleCalendarEventCompleted(id int64) (bool, error) {
	var event CalendarEvent
	err := db.Get(&event, "SELECT is_completed FROM calendar_events WHERE id = ?", id)
	if err != nil {
		return false, err
	}

	var newStatus = !event.IsCompleted
	_, err = db.Exec(`
		UPDATE calendar_events SET is_completed = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		newStatus, id)
	return newStatus, err
}

// DeleteCalendarEvent removes an event
func DeleteCalendarEvent(id int64) error {
	_, err := db.Exec("DELETE FROM calendar_events WHERE id = ?", id)
	return err
}

// DeleteCalendarEventByTask removes the event associated with a task
func DeleteCalendarEventByTask(taskID int64) error {
	_, err := db.Exec("DELETE FROM calendar_events WHERE task_id = ?", taskID)
	return err
}

// CountCalendarEvents returns event counts
func CountCalendarEvents(accountID int64) (upcoming, completed, total int, err error) {
	var now = time.Now().Format("2006-01-02 15:04:05")

	err = db.Get(&upcoming, `
		SELECT COUNT(*) FROM calendar_events
		WHERE account_id = ? AND start_time >= ? AND is_completed = 0`,
		accountID, now)
	if err != nil {
		return
	}

	err = db.Get(&completed, `
		SELECT COUNT(*) FROM calendar_events
		WHERE account_id = ? AND is_completed = 1`,
		accountID)
	if err != nil {
		return
	}

	err = db.Get(&total, `
		SELECT COUNT(*) FROM calendar_events
		WHERE account_id = ?`,
		accountID)
	return
}

// GetCalendarEventsPendingSync returns events that need to be synced to Google
func GetCalendarEventsPendingSync(accountID int64) ([]CalendarEvent, error) {
	var events []CalendarEvent
	err := db.Select(&events, `
		SELECT * FROM calendar_events
		WHERE account_id = ? AND sync_status = 'pending_sync'
		ORDER BY updated_at ASC`,
		accountID)
	return events, err
}

// UpdateCalendarEventSyncStatus updates the sync status of an event
func UpdateCalendarEventSyncStatus(id int64, status CalendarSyncStatus, googleEventID string) error {
	_, err := db.Exec(`
		UPDATE calendar_events SET
			sync_status = ?,
			google_event_id = ?,
			last_synced_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		status, googleEventID, id)
	return err
}

// GetCalendarEventByGoogleID returns an event by its Google Calendar event ID
func GetCalendarEventByGoogleID(googleEventID string) (*CalendarEvent, error) {
	var event CalendarEvent
	err := db.Get(&event, `
		SELECT * FROM calendar_events
		WHERE google_event_id = ?`,
		googleEventID)
	if err != nil {
		return nil, err
	}
	return &event, nil
}
