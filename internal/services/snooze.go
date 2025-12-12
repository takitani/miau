package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/storage"
)

// SnoozeService implements ports.SnoozeService
type SnoozeService struct {
	mu      sync.RWMutex
	storage ports.StoragePort
	events  ports.EventBus
	account *ports.AccountInfo
}

// NewSnoozeService creates a new SnoozeService
func NewSnoozeService(storagePort ports.StoragePort, events ports.EventBus) *SnoozeService {
	return &SnoozeService{
		storage: storagePort,
		events:  events,
	}
}

// SetAccount sets the current account
func (s *SnoozeService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// SnoozeEmail snoozes an email until specified time
func (s *SnoozeService) SnoozeEmail(ctx context.Context, emailID int64, until time.Time) error {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return fmt.Errorf("no account set")
	}

	if err := storage.SnoozeEmail(emailID, account.ID, until, string(ports.SnoozeCustom)); err != nil {
		return err
	}

	s.events.Publish(ports.BaseEvent{
		EventType: ports.EventTypeEmailSnoozed,
		Time:      time.Now(),
	})

	return nil
}

// SnoozeEmailPreset snoozes using a preset duration
func (s *SnoozeService) SnoozeEmailPreset(ctx context.Context, emailID int64, preset ports.SnoozePreset) error {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return fmt.Errorf("no account set")
	}

	var until = calculateSnoozeTime(preset)
	if err := storage.SnoozeEmail(emailID, account.ID, until, string(preset)); err != nil {
		return err
	}

	s.events.Publish(ports.BaseEvent{
		EventType: ports.EventTypeEmailSnoozed,
		Time:      time.Now(),
	})

	return nil
}

// UnsnoozeEmail removes snooze before it triggers
func (s *SnoozeService) UnsnoozeEmail(ctx context.Context, emailID int64) error {
	if err := storage.UnsnoozeEmail(emailID); err != nil {
		return err
	}

	s.events.Publish(ports.BaseEvent{
		EventType: ports.EventTypeEmailUnsnoozed,
		Time:      time.Now(),
	})

	return nil
}

// GetSnoozedEmails returns all currently snoozed emails
func (s *SnoozeService) GetSnoozedEmails(ctx context.Context) ([]ports.SnoozedEmail, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	var snoozes, err = storage.GetSnoozedEmails(account.ID)
	if err != nil {
		return nil, err
	}

	var result = make([]ports.SnoozedEmail, len(snoozes))
	for i, sn := range snoozes {
		result[i] = ports.SnoozedEmail{
			ID:          sn.ID,
			EmailID:     sn.EmailID,
			AccountID:   sn.AccountID,
			SnoozedAt:   sn.SnoozedAt,
			SnoozeUntil: sn.SnoozeUntil,
			Preset:      ports.SnoozePreset(sn.Preset),
			Processed:   sn.Processed,
		}
	}

	return result, nil
}

// GetSnoozedEmailsCount returns the count of snoozed emails
func (s *SnoozeService) GetSnoozedEmailsCount(ctx context.Context) (int, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return 0, fmt.Errorf("no account set")
	}

	return storage.GetSnoozedEmailsCount(account.ID)
}

// ProcessDueSnoozes processes snoozes that are due
func (s *SnoozeService) ProcessDueSnoozes(ctx context.Context) (int, error) {
	var dueSnoozes, err = storage.GetDueSnoozes()
	if err != nil {
		return 0, err
	}

	var processed = 0
	for _, snooze := range dueSnoozes {
		// Mark email as unread
		if err := storage.MarkEmailUnread(snooze.EmailID); err != nil {
			continue
		}

		// Bump email date to appear at top
		if err := storage.BumpEmailDate(snooze.EmailID); err != nil {
			continue
		}

		// Mark snooze as processed
		if err := storage.MarkSnoozeProcessed(snooze.ID); err != nil {
			continue
		}

		processed++

		// Emit event
		s.events.Publish(ports.BaseEvent{
			EventType: ports.EventTypeEmailUnsnoozed,
			Time:      time.Now(),
		})
	}

	return processed, nil
}

// GetSnoozePresets returns all available presets with calculated times
func (s *SnoozeService) GetSnoozePresets() []ports.SnoozePresetInfo {
	return []ports.SnoozePresetInfo{
		{
			Preset:      ports.SnoozeLaterToday,
			Label:       "Later today",
			Description: formatTime(calculateSnoozeTime(ports.SnoozeLaterToday)),
			Time:        calculateSnoozeTime(ports.SnoozeLaterToday),
		},
		{
			Preset:      ports.SnoozeTomorrow,
			Label:       "Tomorrow",
			Description: formatSnoozeDate(calculateSnoozeTime(ports.SnoozeTomorrow)),
			Time:        calculateSnoozeTime(ports.SnoozeTomorrow),
		},
		{
			Preset:      ports.SnoozeThisWeekend,
			Label:       "This weekend",
			Description: formatSnoozeDate(calculateSnoozeTime(ports.SnoozeThisWeekend)),
			Time:        calculateSnoozeTime(ports.SnoozeThisWeekend),
		},
		{
			Preset:      ports.SnoozeNextWeek,
			Label:       "Next week",
			Description: formatSnoozeDate(calculateSnoozeTime(ports.SnoozeNextWeek)),
			Time:        calculateSnoozeTime(ports.SnoozeNextWeek),
		},
		{
			Preset:      ports.SnoozeNextMonth,
			Label:       "Next month",
			Description: formatSnoozeDateMonth(calculateSnoozeTime(ports.SnoozeNextMonth)),
			Time:        calculateSnoozeTime(ports.SnoozeNextMonth),
		},
	}
}

// IsEmailSnoozed checks if an email is currently snoozed
func (s *SnoozeService) IsEmailSnoozed(ctx context.Context, emailID int64) (bool, error) {
	return storage.IsEmailSnoozed(emailID)
}

// calculateSnoozeTime calculates the target time for a preset
func calculateSnoozeTime(preset ports.SnoozePreset) time.Time {
	var now = time.Now()
	var loc = now.Location()

	switch preset {
	case ports.SnoozeLaterToday:
		// +4 hours, but at least 4 PM today
		var later = now.Add(4 * time.Hour)
		var fourPM = time.Date(now.Year(), now.Month(), now.Day(), 16, 0, 0, 0, loc)
		if later.Before(fourPM) {
			return fourPM
		}
		return later

	case ports.SnoozeTomorrow:
		// Tomorrow 9 AM
		var tomorrow = now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, loc)

	case ports.SnoozeThisWeekend:
		// Next Saturday 9 AM
		var daysUntilSat = (6 - int(now.Weekday()) + 7) % 7
		if daysUntilSat == 0 {
			daysUntilSat = 7 // If today is Saturday, next Saturday
		}
		var saturday = now.AddDate(0, 0, daysUntilSat)
		return time.Date(saturday.Year(), saturday.Month(), saturday.Day(), 9, 0, 0, 0, loc)

	case ports.SnoozeNextWeek:
		// Next Monday 9 AM
		var daysUntilMon = (8 - int(now.Weekday())) % 7
		if daysUntilMon == 0 {
			daysUntilMon = 7 // If today is Monday, next Monday
		}
		var monday = now.AddDate(0, 0, daysUntilMon)
		return time.Date(monday.Year(), monday.Month(), monday.Day(), 9, 0, 0, 0, loc)

	case ports.SnoozeNextMonth:
		// 1st of next month 9 AM
		var nextMonth = now.AddDate(0, 1, 0)
		return time.Date(nextMonth.Year(), nextMonth.Month(), 1, 9, 0, 0, 0, loc)

	default:
		// Default to tomorrow 9 AM
		var tomorrow = now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, loc)
	}
}

// formatTime formats time as "3:04 PM"
func formatTime(t time.Time) string {
	return t.Format("3:04 PM")
}

// formatSnoozeDate formats date as "Mon, Jan 2 at 9:00 AM"
func formatSnoozeDate(t time.Time) string {
	return t.Format("Mon, Jan 2 at 3:04 PM")
}

// formatSnoozeDateMonth formats date as "Jan 2, 2006"
func formatSnoozeDateMonth(t time.Time) string {
	return t.Format("Jan 2, 2006 at 9:00 AM")
}
