package services

import (
	"context"
	"fmt"
	"time"

	"github.com/opik/miau/internal/gmail"
	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/storage"
)

// CalendarService implements ports.CalendarService
type CalendarService struct {
	taskService    ports.TaskService
	googleCalendar *gmail.CalendarClient
}

// NewCalendarService creates a new CalendarService
func NewCalendarService(taskService ports.TaskService) *CalendarService {
	return &CalendarService{
		taskService: taskService,
	}
}

// SetGoogleCalendarClient sets the Google Calendar client for syncing
func (s *CalendarService) SetGoogleCalendarClient(client *gmail.CalendarClient) {
	s.googleCalendar = client
}

// CreateEvent creates a new calendar event
func (s *CalendarService) CreateEvent(ctx context.Context, input *ports.CalendarEventInput) (*ports.CalendarEventInfo, error) {
	if input.Title == "" {
		return nil, fmt.Errorf("event title is required")
	}
	if input.StartTime.IsZero() {
		return nil, fmt.Errorf("event start time is required")
	}

	var event = &storage.CalendarEvent{
		AccountID:        input.AccountID,
		Title:            input.Title,
		Description:      toNullString(input.Description),
		EventType:        storage.CalendarEventType(input.EventType),
		StartTime:        storage.SQLiteTime{Time: input.StartTime},
		EndTime:          toNullTime(input.EndTime),
		AllDay:           input.AllDay,
		Color:            toNullString(input.Color),
		TaskID:           toNullInt64(input.TaskID),
		EmailID:          toNullInt64(input.EmailID),
		IsCompleted:      input.IsCompleted,
		Source:           storage.CalendarEventSource(input.Source),
		GoogleEventID:    toNullString(input.GoogleEventID),
		GoogleCalendarID: toNullString(input.GoogleCalendarID),
		SyncStatus:       storage.CalendarSyncStatus(input.SyncStatus),
	}

	// Set defaults
	if event.EventType == "" {
		event.EventType = storage.CalendarEventTypeCustom
	}
	if event.Source == "" {
		event.Source = storage.CalendarEventSourceManual
	}
	if event.SyncStatus == "" {
		event.SyncStatus = storage.CalendarSyncStatusLocal
	}
	// Set default color based on event type if not provided
	if !event.Color.Valid || event.Color.String == "" {
		event.Color = toNullString(ports.GetDefaultColor(input.EventType))
	}

	err := storage.CreateCalendarEvent(event)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return storageEventToInfo(event), nil
}

// GetEvent returns an event by ID
func (s *CalendarService) GetEvent(ctx context.Context, id int64) (*ports.CalendarEventInfo, error) {
	event, err := storage.GetCalendarEvent(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}
	if event == nil {
		return nil, nil
	}
	return storageEventToInfo(event), nil
}

// GetEvents returns all events for an account
func (s *CalendarService) GetEvents(ctx context.Context, accountID int64) ([]ports.CalendarEventInfo, error) {
	events, err := storage.GetCalendarEvents(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}
	return storageEventsToInfos(events), nil
}

// GetEventsByDateRange returns events within a date range
func (s *CalendarService) GetEventsByDateRange(ctx context.Context, accountID int64, start, end time.Time) ([]ports.CalendarEventInfo, error) {
	events, err := storage.GetCalendarEventsByDateRange(accountID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by date range: %w", err)
	}
	return storageEventsToInfos(events), nil
}

// GetEventsForWeek returns events for a specific week
func (s *CalendarService) GetEventsForWeek(ctx context.Context, accountID int64, weekStart time.Time) ([]ports.CalendarEventInfo, error) {
	events, err := storage.GetCalendarEventsForWeek(accountID, weekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to get events for week: %w", err)
	}
	return storageEventsToInfos(events), nil
}

// GetUpcomingEvents returns upcoming events from now
func (s *CalendarService) GetUpcomingEvents(ctx context.Context, accountID int64, limit int) ([]ports.CalendarEventInfo, error) {
	if limit <= 0 {
		limit = 10
	}
	events, err := storage.GetUpcomingCalendarEvents(accountID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming events: %w", err)
	}
	return storageEventsToInfos(events), nil
}

// GetEventByTask returns the event associated with a task
func (s *CalendarService) GetEventByTask(ctx context.Context, taskID int64) (*ports.CalendarEventInfo, error) {
	event, err := storage.GetCalendarEventByTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event by task: %w", err)
	}
	if event == nil {
		return nil, nil
	}
	return storageEventToInfo(event), nil
}

// GetEventsByEmail returns events linked to an email
func (s *CalendarService) GetEventsByEmail(ctx context.Context, emailID int64) ([]ports.CalendarEventInfo, error) {
	events, err := storage.GetCalendarEventsByEmail(emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by email: %w", err)
	}
	return storageEventsToInfos(events), nil
}

// UpdateEvent updates an existing event
func (s *CalendarService) UpdateEvent(ctx context.Context, input *ports.CalendarEventInput) (*ports.CalendarEventInfo, error) {
	if input.ID == 0 {
		return nil, fmt.Errorf("event ID is required")
	}
	if input.Title == "" {
		return nil, fmt.Errorf("event title is required")
	}

	var event = &storage.CalendarEvent{
		ID:               input.ID,
		AccountID:        input.AccountID,
		Title:            input.Title,
		Description:      toNullString(input.Description),
		EventType:        storage.CalendarEventType(input.EventType),
		StartTime:        storage.SQLiteTime{Time: input.StartTime},
		EndTime:          toNullTime(input.EndTime),
		AllDay:           input.AllDay,
		Color:            toNullString(input.Color),
		TaskID:           toNullInt64(input.TaskID),
		EmailID:          toNullInt64(input.EmailID),
		IsCompleted:      input.IsCompleted,
		Source:           storage.CalendarEventSource(input.Source),
		GoogleEventID:    toNullString(input.GoogleEventID),
		GoogleCalendarID: toNullString(input.GoogleCalendarID),
		SyncStatus:       storage.CalendarSyncStatus(input.SyncStatus),
	}

	err := storage.UpdateCalendarEvent(event)
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	return s.GetEvent(ctx, input.ID)
}

// ToggleEventCompleted toggles the completed status
func (s *CalendarService) ToggleEventCompleted(ctx context.Context, id int64) (bool, error) {
	newStatus, err := storage.ToggleCalendarEventCompleted(id)
	if err != nil {
		return false, fmt.Errorf("failed to toggle event: %w", err)
	}
	return newStatus, nil
}

// DeleteEvent removes an event
func (s *CalendarService) DeleteEvent(ctx context.Context, id int64) error {
	err := storage.DeleteCalendarEvent(id)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	return nil
}

// CountEvents returns event counts
func (s *CalendarService) CountEvents(ctx context.Context, accountID int64) (*ports.CalendarEventCounts, error) {
	upcoming, completed, total, err := storage.CountCalendarEvents(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to count events: %w", err)
	}
	return &ports.CalendarEventCounts{
		Upcoming:  upcoming,
		Completed: completed,
		Total:     total,
	}, nil
}

// === Task Sync ===

// CreateEventFromTask creates a calendar event from a task with due_date
func (s *CalendarService) CreateEventFromTask(ctx context.Context, taskID int64) (*ports.CalendarEventInfo, error) {
	task, err := s.taskService.GetTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %d", taskID)
	}
	if task.DueDate == nil {
		return nil, fmt.Errorf("task has no due date")
	}

	// Check if event already exists for this task
	existingEvent, _ := s.GetEventByTask(ctx, taskID)
	if existingEvent != nil {
		return existingEvent, nil
	}

	// Create event from task
	var input = &ports.CalendarEventInput{
		AccountID:   task.AccountID,
		Title:       task.Title,
		Description: task.Description,
		EventType:   ports.CalendarEventTypeTaskDeadline,
		StartTime:   *task.DueDate,
		AllDay:      true, // Task deadlines are typically all-day events
		TaskID:      &taskID,
		IsCompleted: task.IsCompleted,
		Source:      ports.CalendarEventSourceTaskSync,
		Color:       ports.GetDefaultColor(ports.CalendarEventTypeTaskDeadline),
	}

	return s.CreateEvent(ctx, input)
}

// UpdateEventFromTask updates the calendar event when task is modified
func (s *CalendarService) UpdateEventFromTask(ctx context.Context, taskID int64) (*ports.CalendarEventInfo, error) {
	task, err := s.taskService.GetTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task not found: %d", taskID)
	}

	// Get existing event
	event, err := s.GetEventByTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	// If task no longer has due_date, delete the event
	if task.DueDate == nil {
		if event != nil {
			s.DeleteEvent(ctx, event.ID)
		}
		return nil, nil
	}

	// If no event exists, create one
	if event == nil {
		return s.CreateEventFromTask(ctx, taskID)
	}

	// Update event with task data
	var input = &ports.CalendarEventInput{
		ID:          event.ID,
		AccountID:   task.AccountID,
		Title:       task.Title,
		Description: task.Description,
		EventType:   ports.CalendarEventTypeTaskDeadline,
		StartTime:   *task.DueDate,
		AllDay:      true,
		TaskID:      &taskID,
		IsCompleted: task.IsCompleted,
		Source:      ports.CalendarEventSourceTaskSync,
		Color:       event.Color,
		SyncStatus:  event.SyncStatus,
	}

	return s.UpdateEvent(ctx, input)
}

// DeleteEventByTask removes the event when task is deleted
func (s *CalendarService) DeleteEventByTask(ctx context.Context, taskID int64) error {
	err := storage.DeleteCalendarEventByTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to delete event by task: %w", err)
	}
	return nil
}

// SyncTasksToCalendar syncs all tasks with due_date to calendar
func (s *CalendarService) SyncTasksToCalendar(ctx context.Context, accountID int64) error {
	// Get all pending tasks
	tasks, err := s.taskService.GetPendingTasks(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get tasks: %w", err)
	}

	for _, task := range tasks {
		if task.DueDate != nil {
			_, err := s.CreateEventFromTask(ctx, task.ID)
			if err != nil {
				// Log error but continue
				continue
			}
		}
	}

	return nil
}

// === Email Follow-up ===

// CreateFollowUpEvent creates a follow-up event for an email
func (s *CalendarService) CreateFollowUpEvent(ctx context.Context, emailID int64, followUpTime time.Time, title string) (*ports.CalendarEventInfo, error) {
	// Get email to get account_id
	email, err := storage.GetEmailByID(emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}
	if email == nil {
		return nil, fmt.Errorf("email not found: %d", emailID)
	}

	if title == "" {
		title = fmt.Sprintf("Follow up: %s", email.Subject)
	}

	var input = &ports.CalendarEventInput{
		AccountID:   email.AccountID,
		Title:       title,
		Description: fmt.Sprintf("Follow up on email from %s", email.FromEmail),
		EventType:   ports.CalendarEventTypeEmailFollowup,
		StartTime:   followUpTime,
		AllDay:      false,
		EmailID:     &emailID,
		Source:      ports.CalendarEventSourceManual,
		Color:       ports.GetDefaultColor(ports.CalendarEventTypeEmailFollowup),
	}

	return s.CreateEvent(ctx, input)
}

// Helper functions

func storageEventToInfo(e *storage.CalendarEvent) *ports.CalendarEventInfo {
	return &ports.CalendarEventInfo{
		ID:               e.ID,
		AccountID:        e.AccountID,
		Title:            e.Title,
		Description:      nullStringToString(e.Description),
		EventType:        ports.CalendarEventType(e.EventType),
		StartTime:        e.StartTime.Time,
		EndTime:          nullTimeToPtr(e.EndTime),
		AllDay:           e.AllDay,
		Color:            nullStringToString(e.Color),
		TaskID:           nullInt64ToPtr(e.TaskID),
		EmailID:          nullInt64ToPtr(e.EmailID),
		IsCompleted:      e.IsCompleted,
		Source:           ports.CalendarEventSource(e.Source),
		GoogleEventID:    nullStringToString(e.GoogleEventID),
		GoogleCalendarID: nullStringToString(e.GoogleCalendarID),
		LastSyncedAt:     nullTimeToPtr(e.LastSyncedAt),
		SyncStatus:       ports.CalendarSyncStatus(e.SyncStatus),
		CreatedAt:        e.CreatedAt.Time,
		UpdatedAt:        e.UpdatedAt.Time,
	}
}

func storageEventsToInfos(events []storage.CalendarEvent) []ports.CalendarEventInfo {
	var infos = make([]ports.CalendarEventInfo, len(events))
	for i, e := range events {
		infos[i] = *storageEventToInfo(&e)
	}
	return infos
}

// Re-use existing helper functions from task.go
// These are already defined there, so we don't need to redeclare them

// === CalendarSyncCallback implementation ===
// CalendarService implements ports.CalendarSyncCallback for bidirectional Task ↔ Calendar sync

// OnTaskCreated is called when a task with due_date is created
func (s *CalendarService) OnTaskCreated(ctx context.Context, taskID int64) error {
	_, err := s.CreateEventFromTask(ctx, taskID)
	if err != nil {
		// Don't fail the task creation if calendar sync fails
		fmt.Printf("[CalendarSync] Failed to create event from task %d: %v\n", taskID, err)
	}
	return nil
}

// OnTaskUpdated is called when a task is updated (including due_date changes)
func (s *CalendarService) OnTaskUpdated(ctx context.Context, taskID int64) error {
	_, err := s.UpdateEventFromTask(ctx, taskID)
	if err != nil {
		fmt.Printf("[CalendarSync] Failed to update event from task %d: %v\n", taskID, err)
	}
	return nil
}

// OnTaskDeleted is called when a task is deleted
func (s *CalendarService) OnTaskDeleted(ctx context.Context, taskID int64) error {
	err := s.DeleteEventByTask(ctx, taskID)
	if err != nil {
		fmt.Printf("[CalendarSync] Failed to delete event for task %d: %v\n", taskID, err)
	}
	return nil
}

// OnTaskCompletedToggled is called when task completion status changes
func (s *CalendarService) OnTaskCompletedToggled(ctx context.Context, taskID int64, isCompleted bool) error {
	event, err := s.GetEventByTask(ctx, taskID)
	if err != nil || event == nil {
		return nil // No event to update
	}

	// Only update if completion status differs
	if event.IsCompleted != isCompleted {
		_, err := s.ToggleEventCompleted(ctx, event.ID)
		if err != nil {
			fmt.Printf("[CalendarSync] Failed to toggle event completion for task %d: %v\n", taskID, err)
		}
	}
	return nil
}

// === Google Calendar Sync ===

// SyncFromGoogleCalendar syncs events from Google Calendar to local storage
func (s *CalendarService) SyncFromGoogleCalendar(ctx context.Context, accountID int64, calendarID string) (int, error) {
	fmt.Printf("[CalendarService.SyncFromGoogleCalendar] Starting sync for account %d, calendar %s\n", accountID, calendarID)

	if s.googleCalendar == nil {
		fmt.Printf("[CalendarService.SyncFromGoogleCalendar] Google Calendar client is nil\n")
		return 0, fmt.Errorf("Google Calendar client not configured")
	}

	if calendarID == "" {
		calendarID = "primary"
	}

	// Get events for the next 4 weeks
	now := time.Now()
	start := now.AddDate(0, 0, -7)  // 1 week ago
	end := now.AddDate(0, 0, 28)    // 4 weeks ahead

	fmt.Printf("[CalendarService.SyncFromGoogleCalendar] Fetching events from %v to %v\n", start, end)
	googleEvents, err := s.googleCalendar.ListEvents(ctx, calendarID, start, end, 250)
	if err != nil {
		fmt.Printf("[CalendarService.SyncFromGoogleCalendar] ListEvents error: %v\n", err)
		return 0, fmt.Errorf("failed to fetch Google Calendar events: %w", err)
	}
	fmt.Printf("[CalendarService.SyncFromGoogleCalendar] Got %d events from Google\n", len(googleEvents))

	var synced int
	for _, ge := range googleEvents {
		// Skip cancelled events
		if ge.Status == "cancelled" {
			continue
		}

		// Check if event already exists locally
		existing, _ := storage.GetCalendarEventByGoogleID(ge.ID)
		if existing != nil {
			// Update existing event
			existing.Title = ge.Summary
			existing.Description = toNullString(ge.Description)
			existing.StartTime = storage.SQLiteTime{Time: ge.StartTime}
			existing.EndTime = toNullTime(&ge.EndTime)
			existing.AllDay = ge.AllDay
			existing.LastSyncedAt = toNullTime(&now)
			existing.SyncStatus = storage.CalendarSyncStatusSynced

			err := storage.UpdateCalendarEvent(existing)
			if err != nil {
				fmt.Printf("[GoogleSync] Failed to update event %s: %v\n", ge.ID, err)
				continue
			}
		} else {
			// Create new event
			eventType := storage.CalendarEventTypeMeeting
			if ge.AllDay {
				eventType = storage.CalendarEventTypeCustom
			}

			event := &storage.CalendarEvent{
				AccountID:        accountID,
				Title:            ge.Summary,
				Description:      toNullString(ge.Description),
				EventType:        eventType,
				StartTime:        storage.SQLiteTime{Time: ge.StartTime},
				EndTime:          toNullTime(&ge.EndTime),
				AllDay:           ge.AllDay,
				Color:            toNullString(getColorFromGoogleColorID(ge.ColorID)),
				Source:           storage.CalendarEventSourceManual,
				GoogleEventID:    toNullString(ge.ID),
				GoogleCalendarID: toNullString(calendarID),
				LastSyncedAt:     toNullTime(&now),
				SyncStatus:       storage.CalendarSyncStatusSynced,
			}

			err := storage.CreateCalendarEvent(event)
			if err != nil {
				fmt.Printf("[GoogleSync] Failed to create event %s: %v\n", ge.ID, err)
				continue
			}
		}
		synced++
	}

	return synced, nil
}

// ListGoogleCalendars lists all Google Calendars for the user
func (s *CalendarService) ListGoogleCalendars(ctx context.Context) ([]ports.GoogleCalendarInfo, error) {
	if s.googleCalendar == nil {
		return nil, fmt.Errorf("Google Calendar client not configured")
	}

	calendars, err := s.googleCalendar.ListCalendars(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	var result []ports.GoogleCalendarInfo
	for _, cal := range calendars {
		result = append(result, ports.GoogleCalendarInfo{
			ID:              cal.ID,
			Summary:         cal.Summary,
			Description:     cal.Description,
			Primary:         cal.Primary,
			BackgroundColor: cal.BackgroundColor,
			AccessRole:      cal.AccessRole,
		})
	}

	return result, nil
}

// GetGoogleCalendarEvents returns events from Google Calendar for a week
func (s *CalendarService) GetGoogleCalendarEvents(ctx context.Context, calendarID string, weekStart time.Time) ([]ports.GoogleEventInfo, error) {
	if s.googleCalendar == nil {
		return nil, fmt.Errorf("Google Calendar client not configured")
	}

	if calendarID == "" {
		calendarID = "primary"
	}

	events, err := s.googleCalendar.ListEventsForWeek(ctx, calendarID, weekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	var result []ports.GoogleEventInfo
	for _, e := range events {
		result = append(result, ports.GoogleEventInfo{
			ID:          e.ID,
			CalendarID:  e.CalendarID,
			Summary:     e.Summary,
			Description: e.Description,
			Location:    e.Location,
			StartTime:   e.StartTime,
			EndTime:     e.EndTime,
			AllDay:      e.AllDay,
			Status:      e.Status,
			HtmlLink:    e.HtmlLink,
			ColorID:     e.ColorID,
		})
	}

	return result, nil
}

// IsGoogleCalendarConnected returns true if Google Calendar is connected
func (s *CalendarService) IsGoogleCalendarConnected() bool {
	return s.googleCalendar != nil
}

// getColorFromGoogleColorID maps Google Calendar color IDs to hex colors
func getColorFromGoogleColorID(colorID string) string {
	// Google Calendar color IDs
	colors := map[string]string{
		"1":  "#7986cb", // Lavender
		"2":  "#33b679", // Sage
		"3":  "#8e24aa", // Grape
		"4":  "#e67c73", // Flamingo
		"5":  "#f6c026", // Banana
		"6":  "#f5511d", // Tangerine
		"7":  "#039be5", // Peacock
		"8":  "#616161", // Graphite
		"9":  "#3f51b5", // Blueberry
		"10": "#0b8043", // Basil
		"11": "#d60000", // Tomato
	}

	if c, ok := colors[colorID]; ok {
		return c
	}
	return "#3498db" // Default blue
}
