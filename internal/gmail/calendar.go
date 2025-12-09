package gmail

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// CalendarClient wraps Google Calendar API operations
type CalendarClient struct {
	service *calendar.Service
}

// NewCalendarClient creates a new Calendar API client
func NewCalendarClient(ctx context.Context, httpClient *http.Client) (*CalendarClient, error) {
	service, err := calendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}

	return &CalendarClient{service: service}, nil
}

// CalendarEventInfo represents a Google Calendar event
type CalendarEventInfo struct {
	ID              string
	CalendarID      string
	Summary         string
	Description     string
	Location        string
	StartTime       time.Time
	EndTime         time.Time
	AllDay          bool
	Status          string // confirmed, tentative, cancelled
	HtmlLink        string
	Organizer       string
	OrganizerEmail  string
	Attendees       []AttendeeInfo
	ColorID         string
	Recurring       bool
	RecurrenceRule  string
	Created         time.Time
	Updated         time.Time
}

// AttendeeInfo represents a calendar event attendee
type AttendeeInfo struct {
	Email          string
	DisplayName    string
	ResponseStatus string // needsAction, declined, tentative, accepted
	Organizer      bool
	Self           bool
}

// ListCalendars returns all calendars for the user
func (c *CalendarClient) ListCalendars(ctx context.Context) ([]CalendarInfo, error) {
	resp, err := c.service.CalendarList.List().Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list calendars: %w", err)
	}

	var calendars []CalendarInfo
	for _, cal := range resp.Items {
		calendars = append(calendars, CalendarInfo{
			ID:              cal.Id,
			Summary:         cal.Summary,
			Description:     cal.Description,
			Primary:         cal.Primary,
			BackgroundColor: cal.BackgroundColor,
			ForegroundColor: cal.ForegroundColor,
			AccessRole:      cal.AccessRole,
		})
	}

	return calendars, nil
}

// CalendarInfo represents a Google Calendar
type CalendarInfo struct {
	ID              string
	Summary         string
	Description     string
	Primary         bool
	BackgroundColor string
	ForegroundColor string
	AccessRole      string // owner, writer, reader, freeBusyReader
}

// ListEvents returns events from a calendar within a time range
func (c *CalendarClient) ListEvents(ctx context.Context, calendarID string, timeMin, timeMax time.Time, maxResults int64) ([]CalendarEventInfo, error) {
	if calendarID == "" {
		calendarID = "primary"
	}
	if maxResults <= 0 {
		maxResults = 100
	}

	call := c.service.Events.List(calendarID).
		Context(ctx).
		TimeMin(timeMin.Format(time.RFC3339)).
		TimeMax(timeMax.Format(time.RFC3339)).
		MaxResults(maxResults).
		SingleEvents(true).
		OrderBy("startTime")

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	var events []CalendarEventInfo
	for _, item := range resp.Items {
		event := parseCalendarEvent(item, calendarID)
		events = append(events, event)
	}

	return events, nil
}

// ListUpcomingEvents returns upcoming events from now
func (c *CalendarClient) ListUpcomingEvents(ctx context.Context, calendarID string, maxResults int64) ([]CalendarEventInfo, error) {
	if calendarID == "" {
		calendarID = "primary"
	}
	if maxResults <= 0 {
		maxResults = 50
	}

	call := c.service.Events.List(calendarID).
		Context(ctx).
		TimeMin(time.Now().Format(time.RFC3339)).
		MaxResults(maxResults).
		SingleEvents(true).
		OrderBy("startTime")

	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list upcoming events: %w", err)
	}

	var events []CalendarEventInfo
	for _, item := range resp.Items {
		event := parseCalendarEvent(item, calendarID)
		events = append(events, event)
	}

	return events, nil
}

// GetEvent returns a specific event by ID
func (c *CalendarClient) GetEvent(ctx context.Context, calendarID, eventID string) (*CalendarEventInfo, error) {
	if calendarID == "" {
		calendarID = "primary"
	}

	item, err := c.service.Events.Get(calendarID, eventID).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	event := parseCalendarEvent(item, calendarID)
	return &event, nil
}

// ListEventsForWeek returns events for a specific week
func (c *CalendarClient) ListEventsForWeek(ctx context.Context, calendarID string, weekStart time.Time) ([]CalendarEventInfo, error) {
	// Ensure weekStart is at start of day
	weekStart = time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
	weekEnd := weekStart.AddDate(0, 0, 7)

	return c.ListEvents(ctx, calendarID, weekStart, weekEnd, 100)
}

// parseCalendarEvent converts a Google Calendar event to our format
func parseCalendarEvent(item *calendar.Event, calendarID string) CalendarEventInfo {
	var event = CalendarEventInfo{
		ID:          item.Id,
		CalendarID:  calendarID,
		Summary:     item.Summary,
		Description: item.Description,
		Location:    item.Location,
		Status:      item.Status,
		HtmlLink:    item.HtmlLink,
		ColorID:     item.ColorId,
	}

	// Parse organizer
	if item.Organizer != nil {
		event.Organizer = item.Organizer.DisplayName
		event.OrganizerEmail = item.Organizer.Email
	}

	// Parse start time
	if item.Start != nil {
		if item.Start.DateTime != "" {
			t, _ := time.Parse(time.RFC3339, item.Start.DateTime)
			event.StartTime = t
			event.AllDay = false
		} else if item.Start.Date != "" {
			t, _ := time.Parse("2006-01-02", item.Start.Date)
			event.StartTime = t
			event.AllDay = true
		}
	}

	// Parse end time
	if item.End != nil {
		if item.End.DateTime != "" {
			t, _ := time.Parse(time.RFC3339, item.End.DateTime)
			event.EndTime = t
		} else if item.End.Date != "" {
			t, _ := time.Parse("2006-01-02", item.End.Date)
			event.EndTime = t
		}
	}

	// Parse attendees
	for _, att := range item.Attendees {
		event.Attendees = append(event.Attendees, AttendeeInfo{
			Email:          att.Email,
			DisplayName:    att.DisplayName,
			ResponseStatus: att.ResponseStatus,
			Organizer:      att.Organizer,
			Self:           att.Self,
		})
	}

	// Check if recurring
	if len(item.Recurrence) > 0 {
		event.Recurring = true
		if len(item.Recurrence) > 0 {
			event.RecurrenceRule = item.Recurrence[0]
		}
	}

	// Parse timestamps
	if item.Created != "" {
		t, _ := time.Parse(time.RFC3339, item.Created)
		event.Created = t
	}
	if item.Updated != "" {
		t, _ := time.Parse(time.RFC3339, item.Updated)
		event.Updated = t
	}

	return event
}
