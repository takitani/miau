package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/opik/miau/internal/ports"
)

// ScheduledSendService handles scheduled email sending with background processing
type ScheduledSendService struct {
	mu           sync.RWMutex
	storage      ports.StoragePort
	draftService *DraftService
	sendService  *SendService
	events       ports.EventBus
	account      *ports.AccountInfo

	// Background worker state
	running      bool
	stopChan     chan struct{}
	checkInterval time.Duration

	// Schedule presets
	defaultMorning   string // e.g. "09:00"
	defaultAfternoon string // e.g. "14:00"
	notifyOnSend     bool
	notifyOnFail     bool
}

// NewScheduledSendService creates a new ScheduledSendService
func NewScheduledSendService(
	storage ports.StoragePort,
	draftService *DraftService,
	sendService *SendService,
	events ports.EventBus,
) *ScheduledSendService {
	return &ScheduledSendService{
		storage:          storage,
		draftService:     draftService,
		sendService:      sendService,
		events:           events,
		stopChan:         make(chan struct{}),
		checkInterval:    1 * time.Minute,
		defaultMorning:   "09:00",
		defaultAfternoon: "14:00",
		notifyOnSend:     true,
		notifyOnFail:     true,
	}
}

// SetAccount sets the current account
func (s *ScheduledSendService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// SetCheckInterval sets how often to check for due drafts
func (s *ScheduledSendService) SetCheckInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checkInterval = interval
}

// SetPresets sets the default schedule presets
func (s *ScheduledSendService) SetPresets(morning, afternoon string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultMorning = morning
	s.defaultAfternoon = afternoon
}

// SetNotifications sets notification preferences
func (s *ScheduledSendService) SetNotifications(onSend, onFail bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifyOnSend = onSend
	s.notifyOnFail = onFail
}

// ScheduleEmail schedules a draft for later sending
func (s *ScheduledSendService) ScheduleEmail(ctx context.Context, draftID int64, sendAt time.Time) error {
	var sendAtUnix = sendAt.Unix()
	if err := s.draftService.ScheduleDraft(ctx, draftID, &sendAtUnix); err != nil {
		return err
	}

	s.events.Publish(ports.BaseEvent{
		EventType: ports.EventTypeDraftScheduled,
		Time:      time.Now(),
	})

	return nil
}

// GetScheduledEmails returns all scheduled emails for the current account
func (s *ScheduledSendService) GetScheduledEmails(ctx context.Context) ([]ports.Draft, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, nil
	}

	return s.storage.GetScheduledDrafts(ctx, account.ID)
}

// UpdateScheduledTime updates the send time for a scheduled email
func (s *ScheduledSendService) UpdateScheduledTime(ctx context.Context, draftID int64, newSendAt time.Time) error {
	var draft, err = s.storage.GetDraft(ctx, draftID)
	if err != nil {
		return err
	}

	if draft.Status != ports.DraftStatusScheduled {
		return nil // Not scheduled, nothing to update
	}

	draft.ScheduledSendAt = &newSendAt
	return s.storage.UpdateDraft(ctx, draft)
}

// CancelScheduledEmail cancels a scheduled email and converts back to draft
func (s *ScheduledSendService) CancelScheduledEmail(ctx context.Context, draftID int64) (*ports.Draft, error) {
	if err := s.draftService.CancelScheduledDraft(ctx, draftID); err != nil {
		return nil, err
	}

	return s.storage.GetDraft(ctx, draftID)
}

// SendNow sends a scheduled email immediately
func (s *ScheduledSendService) SendNow(ctx context.Context, draftID int64) (*ports.SendResult, error) {
	return s.sendService.SendDraft(ctx, draftID)
}

// Start starts the background worker that processes scheduled emails
func (s *ScheduledSendService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	go s.worker()
}

// Stop stops the background worker
func (s *ScheduledSendService) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()
}

// IsRunning returns whether the background worker is running
func (s *ScheduledSendService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// worker is the background goroutine that processes scheduled emails
func (s *ScheduledSendService) worker() {
	log.Printf("[ScheduledSendService] Background worker started")

	s.mu.RLock()
	var interval = s.checkInterval
	s.mu.RUnlock()

	var ticker = time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			log.Printf("[ScheduledSendService] Background worker stopped")
			return
		case <-ticker.C:
			if err := s.ProcessDueSchedules(context.Background()); err != nil {
				log.Printf("[ScheduledSendService] Error processing schedules: %v", err)
			}
		}
	}
}

// ProcessDueSchedules sends emails that are due (called by background job)
func (s *ScheduledSendService) ProcessDueSchedules(ctx context.Context) error {
	var dueEmails, err = s.storage.GetDueScheduledDrafts(ctx, time.Now())
	if err != nil {
		return err
	}

	if len(dueEmails) == 0 {
		return nil
	}

	log.Printf("[ScheduledSendService] Found %d scheduled drafts ready to send", len(dueEmails))

	for _, draft := range dueEmails {
		s.processSingleSchedule(ctx, &draft)
	}

	return nil
}

// processSingleSchedule processes a single scheduled email
func (s *ScheduledSendService) processSingleSchedule(ctx context.Context, draft *ports.Draft) {
	log.Printf("[ScheduledSendService] Processing scheduled draft #%d to %s", draft.ID, draft.ToAddresses)

	// Mark as sending
	if err := s.storage.UpdateDraftStatus(ctx, draft.ID, ports.DraftStatusSending); err != nil {
		log.Printf("[ScheduledSendService] Failed to mark draft #%d as sending: %v", draft.ID, err)
		return
	}

	// Send the email
	var result, sendErr = s.sendService.SendDraft(ctx, draft.ID)

	s.mu.RLock()
	var notifyOnSend = s.notifyOnSend
	var notifyOnFail = s.notifyOnFail
	s.mu.RUnlock()

	if sendErr != nil {
		log.Printf("[ScheduledSendService] Failed to send scheduled draft #%d: %v", draft.ID, sendErr)

		// Mark as failed
		s.storage.UpdateDraftStatus(ctx, draft.ID, ports.DraftStatusFailed)

		// Notify user of failure
		if notifyOnFail {
			s.events.Publish(ports.BaseEvent{
				EventType: ports.EventTypeSendError,
				Time:      time.Now(),
			})
		}
		return
	}

	log.Printf("[ScheduledSendService] Successfully sent scheduled draft #%d, messageID=%s", draft.ID, result.MessageID)

	// Notify user of success
	if notifyOnSend {
		s.events.Publish(ports.SendCompletedEvent{
			BaseEvent: ports.NewBaseEvent(ports.EventTypeSendCompleted),
			Result:    result,
		})
	}
}

// GetPresetTimes returns the preset schedule times
func (s *ScheduledSendService) GetPresetTimes() SchedulePresets {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var now = time.Now()
	var loc = now.Location()

	// Parse morning time
	var morningHour, morningMin = parseTimeString(s.defaultMorning)
	// Parse afternoon time
	var afternoonHour, afternoonMin = parseTimeString(s.defaultAfternoon)

	// Calculate tomorrow morning
	var tomorrowMorning = time.Date(
		now.Year(), now.Month(), now.Day()+1,
		morningHour, morningMin, 0, 0, loc,
	)

	// Calculate tomorrow afternoon
	var tomorrowAfternoon = time.Date(
		now.Year(), now.Month(), now.Day()+1,
		afternoonHour, afternoonMin, 0, 0, loc,
	)

	// Calculate next Monday morning
	var daysUntilMonday = (8 - int(now.Weekday())) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7 // If today is Monday, get next Monday
	}
	var mondayMorning = time.Date(
		now.Year(), now.Month(), now.Day()+daysUntilMonday,
		morningHour, morningMin, 0, 0, loc,
	)

	return SchedulePresets{
		TomorrowMorning:   tomorrowMorning,
		TomorrowAfternoon: tomorrowAfternoon,
		MondayMorning:     mondayMorning,
		MorningTime:       s.defaultMorning,
		AfternoonTime:     s.defaultAfternoon,
	}
}

// SchedulePresets contains pre-calculated schedule times
type SchedulePresets struct {
	TomorrowMorning   time.Time
	TomorrowAfternoon time.Time
	MondayMorning     time.Time
	MorningTime       string // e.g. "09:00"
	AfternoonTime     string // e.g. "14:00"
}

// parseTimeString parses a time string like "09:00" into hours and minutes
func parseTimeString(t string) (int, int) {
	var hour, min int
	if len(t) >= 5 {
		// Parse HH:MM format
		hour = int(t[0]-'0')*10 + int(t[1]-'0')
		min = int(t[3]-'0')*10 + int(t[4]-'0')
	}
	// Default to 9:00 if parsing fails
	if hour < 0 || hour > 23 {
		hour = 9
	}
	if min < 0 || min > 59 {
		min = 0
	}
	return hour, min
}
