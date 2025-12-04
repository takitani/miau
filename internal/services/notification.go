package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	emailparser "github.com/opik/miau/internal/email"
	"github.com/opik/miau/internal/ports"
)

// NotificationService implements ports.NotificationService
type NotificationService struct {
	mu         sync.RWMutex
	storage    ports.StoragePort
	events     ports.EventBus
	account    *ports.AccountInfo
	sentEmails []trackedEmail
	alerts     []ports.Alert
}

type trackedEmail struct {
	MessageID string
	Recipient string
	SentAt    time.Time
}

// NewNotificationService creates a new NotificationService
func NewNotificationService(storage ports.StoragePort, events ports.EventBus) *NotificationService {
	return &NotificationService{
		storage: storage,
		events:  events,
	}
}

// SetAccount sets the current account
func (s *NotificationService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// CheckBounces checks for bounce notifications in recent emails
func (s *NotificationService) CheckBounces(ctx context.Context) ([]ports.BounceInfo, error) {
	s.mu.RLock()
	var account = s.account
	var tracked = s.sentEmails
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	// Get recent sent emails from storage
	var recentSent, err = s.storage.GetRecentSentEmails(ctx, account.ID, 5*time.Minute)
	if err != nil {
		return nil, err
	}

	// Combine with in-memory tracked emails
	var allTracked = make(map[string]string) // messageID -> recipient
	for _, t := range tracked {
		if time.Since(t.SentAt) < 5*time.Minute {
			allTracked[t.MessageID] = t.Recipient
		}
	}
	for _, t := range recentSent {
		allTracked[t.MessageID] = t.To
	}

	if len(allTracked) == 0 {
		return nil, nil
	}

	// Get recent emails that might be bounces
	// We search INBOX for bounce emails
	var emails, err2 = s.storage.GetEmails(ctx, 0, 20) // Get from default folder
	if err2 != nil {
		return nil, err2
	}

	var bounces []ports.BounceInfo
	for _, email := range emails {
		// Check if it's a bounce
		if !emailparser.IsBounceEmail(email.FromEmail, email.FromName, email.Subject) {
			continue
		}

		// Check if it's for a tracked email
		for msgID, recipient := range allTracked {
			// The bounce might reference the original message ID
			// or mention the recipient
			if containsAny(email.Snippet, []string{msgID, recipient}) {
				var bounce = ports.BounceInfo{
					OriginalMessageID: msgID,
					Recipient:         recipient,
					Reason:            emailparser.ExtractBounceReason(email.Snippet, email.Subject),
					BouncedAt:         email.Date,
				}
				bounces = append(bounces, bounce)

				// Publish event
				s.events.Publish(ports.BounceEvent{
					BaseEvent: ports.NewBaseEvent(ports.EventTypeBounce),
					Bounce:    bounce,
				})
			}
		}
	}

	return bounces, nil
}

// containsAny checks if s contains any of the substrings
func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if sub != "" && contains(s, sub) {
			return true
		}
	}
	return false
}

// contains is a simple string contains check
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetAlerts returns pending alerts
func (s *NotificationService) GetAlerts(ctx context.Context) ([]ports.Alert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.alerts, nil
}

// DismissAlert dismisses an alert
func (s *NotificationService) DismissAlert(ctx context.Context, alertID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove alert from list
	var newAlerts []ports.Alert
	for _, a := range s.alerts {
		if a.Title != alertID { // Using title as ID for simplicity
			newAlerts = append(newAlerts, a)
		}
	}
	s.alerts = newAlerts

	return nil
}

// TrackSentEmail tracks a sent email for bounce detection
func (s *NotificationService) TrackSentEmail(ctx context.Context, messageID, recipient string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sentEmails = append(s.sentEmails, trackedEmail{
		MessageID: messageID,
		Recipient: recipient,
		SentAt:    time.Now(),
	})

	// Cleanup old entries (older than 10 minutes)
	var newTracked []trackedEmail
	for _, t := range s.sentEmails {
		if time.Since(t.SentAt) < 10*time.Minute {
			newTracked = append(newTracked, t)
		}
	}
	s.sentEmails = newTracked

	return nil
}

// AddAlert adds a new alert
func (s *NotificationService) AddAlert(alert ports.Alert) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = append(s.alerts, alert)
}

// ClearAlerts clears all alerts
func (s *NotificationService) ClearAlerts() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = nil
}
