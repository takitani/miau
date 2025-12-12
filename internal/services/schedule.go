package services

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/storage"
)

// ScheduleService implements ports.ScheduleService
type ScheduleService struct {
	mu          sync.RWMutex
	storage     ports.StoragePort
	sendService ports.SendService
	events      ports.EventBus
	account     *ports.AccountInfo
}

// NewScheduleService creates a new ScheduleService
func NewScheduleService(storagePort ports.StoragePort, sendService ports.SendService, events ports.EventBus) *ScheduleService {
	return &ScheduleService{
		storage:     storagePort,
		sendService: sendService,
		events:      events,
	}
}

// SetAccount sets the current account
func (s *ScheduleService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// GetSchedulePresets returns all available schedule presets with calculated times
func (s *ScheduleService) GetSchedulePresets() []ports.SchedulePresetInfo {
	return []ports.SchedulePresetInfo{
		{
			Preset:      ports.ScheduleTomorrowMorning,
			Label:       "Tomorrow morning",
			Description: formatScheduleDate(calculateScheduleTime(ports.ScheduleTomorrowMorning)),
			Time:        calculateScheduleTime(ports.ScheduleTomorrowMorning),
		},
		{
			Preset:      ports.ScheduleTomorrowAfternoon,
			Label:       "Tomorrow afternoon",
			Description: formatScheduleDate(calculateScheduleTime(ports.ScheduleTomorrowAfternoon)),
			Time:        calculateScheduleTime(ports.ScheduleTomorrowAfternoon),
		},
		{
			Preset:      ports.ScheduleMondayMorning,
			Label:       "Monday morning",
			Description: formatScheduleDate(calculateScheduleTime(ports.ScheduleMondayMorning)),
			Time:        calculateScheduleTime(ports.ScheduleMondayMorning),
		},
	}
}

// GetScheduledDrafts returns all scheduled drafts
func (s *ScheduleService) GetScheduledDrafts(ctx context.Context) ([]ports.Draft, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	var drafts, err = storage.GetScheduledDrafts(account.ID)
	if err != nil {
		return nil, err
	}

	var result = make([]ports.Draft, len(drafts))
	for i, d := range drafts {
		result[i] = draftToPort(d)
	}
	return result, nil
}

// GetScheduledDraftsCount returns the count of scheduled drafts
func (s *ScheduleService) GetScheduledDraftsCount(ctx context.Context) (int, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return 0, fmt.Errorf("no account set")
	}

	return storage.CountScheduledDrafts(account.ID)
}

// ProcessDueSchedules sends emails that are due
func (s *ScheduleService) ProcessDueSchedules(ctx context.Context) (int, error) {
	var readyDrafts, err = storage.GetScheduledDraftsReady()
	if err != nil {
		return 0, err
	}

	var sent = 0
	for _, draft := range readyDrafts {
		// Mark as sending
		storage.MarkDraftSending(draft.ID)

		// Get full draft for sending
		var draftData, getErr = storage.GetDraftByID(draft.ID)
		if getErr != nil {
			storage.MarkDraftFailed(draft.ID, getErr.Error())
			continue
		}

		// Send the email
		var _, sendErr = s.sendService.SendDraft(ctx, draftData.ID)
		if sendErr != nil {
			storage.MarkDraftFailed(draft.ID, sendErr.Error())
			continue
		}

		sent++
	}

	return sent, nil
}

// calculateScheduleTime calculates the target time for a schedule preset
func calculateScheduleTime(preset ports.SchedulePreset) time.Time {
	var now = time.Now()
	var loc = now.Location()

	switch preset {
	case ports.ScheduleTomorrowMorning:
		// Tomorrow 9 AM
		var tomorrow = now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, loc)

	case ports.ScheduleTomorrowAfternoon:
		// Tomorrow 2 PM
		var tomorrow = now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 14, 0, 0, 0, loc)

	case ports.ScheduleMondayMorning:
		// Next Monday 9 AM
		var daysUntilMon = (8 - int(now.Weekday())) % 7
		if daysUntilMon == 0 {
			daysUntilMon = 7 // If today is Monday, next Monday
		}
		var monday = now.AddDate(0, 0, daysUntilMon)
		return time.Date(monday.Year(), monday.Month(), monday.Day(), 9, 0, 0, 0, loc)

	default:
		// Default to tomorrow 9 AM
		var tomorrow = now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, loc)
	}
}

// formatScheduleDate formats date for schedule preset display
func formatScheduleDate(t time.Time) string {
	return t.Format("Mon, Jan 2 at 3:04 PM")
}

// draftToPort converts storage draft to ports draft
func draftToPort(d storage.Draft) ports.Draft {
	var result = ports.Draft{
		ID:             d.ID,
		ToAddresses:    d.ToAddresses,
		CcAddresses:    nullStringValue(d.CcAddresses),
		BccAddresses:   nullStringValue(d.BccAddresses),
		Subject:        d.Subject,
		BodyHTML:       nullStringValue(d.BodyHTML),
		BodyText:       nullStringValue(d.BodyText),
		Classification: nullStringValue(d.Classification),
		InReplyTo:      nullStringValue(d.InReplyTo),
		ReferenceIDs:   nullStringValue(d.ReferenceIDs),
		Status:         ports.DraftStatus(d.Status),
		Source:         d.GenerationSource,
		AIPrompt:       nullStringValue(d.AIPrompt),
		ErrorMessage:   nullStringValue(d.ErrorMessage),
		CreatedAt:      d.CreatedAt.Time,
		UpdatedAt:      d.UpdatedAt.Time,
	}
	if d.ReplyToEmailID.Valid {
		var id = d.ReplyToEmailID.Int64
		result.ReplyToEmailID = &id
	}
	if d.ScheduledSendAt.Valid {
		var t = d.ScheduledSendAt.Time
		result.ScheduledSendAt = &t
	}
	if d.SentAt.Valid {
		var t = d.SentAt.Time
		result.SentAt = &t
	}
	return result
}

// nullStringValue returns the string value or empty string
func nullStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
