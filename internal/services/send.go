package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/opik/miau/internal/ports"
)

// SendService implements ports.SendService
type SendService struct {
	mu         sync.RWMutex
	smtp       ports.SMTPPort
	gmailAPI   ports.GmailAPIPort
	storage    ports.StoragePort
	events     ports.EventBus
	account    *ports.AccountInfo
	sendMethod ports.SendMethod
}

// NewSendService creates a new SendService
func NewSendService(smtp ports.SMTPPort, gmailAPI ports.GmailAPIPort, storage ports.StoragePort, events ports.EventBus) *SendService {
	return &SendService{
		smtp:       smtp,
		gmailAPI:   gmailAPI,
		storage:    storage,
		events:     events,
		sendMethod: ports.SendMethodSMTP,
	}
}

// SetAccount sets the current account
func (s *SendService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// SetSendMethod sets the send method (SMTP or Gmail API)
func (s *SendService) SetSendMethod(method ports.SendMethod) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sendMethod = method
}

// Send sends an email immediately
func (s *SendService) Send(ctx context.Context, req *ports.SendRequest) (*ports.SendResult, error) {
	s.mu.RLock()
	var account = s.account
	var method = s.sendMethod
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	s.events.Publish(ports.BaseEvent{
		EventType: ports.EventTypeSendStarted,
		Time:      time.Now(),
	})

	var result *ports.SendResult
	var err error

	switch method {
	case ports.SendMethodGmailAPI:
		result, err = s.gmailAPI.Send(ctx, req)
	default:
		result, err = s.smtp.Send(ctx, req)
	}

	if err != nil {
		s.events.Publish(ports.BaseEvent{
			EventType: ports.EventTypeSendError,
			Time:      time.Now(),
		})
		return nil, err
	}

	// Track sent email for bounce detection
	var recipient string
	if len(req.To) > 0 {
		recipient = req.To[0]
	}
	s.storage.TrackSentEmail(ctx, account.ID, result.MessageID, recipient, req.Subject)

	s.events.Publish(ports.SendCompletedEvent{
		BaseEvent: ports.NewBaseEvent(ports.EventTypeSendCompleted),
		Result:    result,
	})

	return result, nil
}

// SendDraft sends a draft
func (s *SendService) SendDraft(ctx context.Context, draftID int64) (*ports.SendResult, error) {
	var draft, err = s.storage.GetDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("draft not found: %w", err)
	}

	// Update draft status to sending
	s.storage.UpdateDraftStatus(ctx, draftID, ports.DraftStatusSending)

	// Convert draft to send request
	var req = &ports.SendRequest{
		To:             parseAddresses(draft.ToAddresses),
		Cc:             parseAddresses(draft.CcAddresses),
		Bcc:            parseAddresses(draft.BccAddresses),
		Subject:        draft.Subject,
		BodyText:       draft.BodyText,
		BodyHTML:       draft.BodyHTML,
		InReplyTo:      draft.InReplyTo,
		ReferenceIDs:   draft.ReferenceIDs,
		ReplyToEmailID: draft.ReplyToEmailID,
		Classification: draft.Classification,
	}

	var result, sendErr = s.Send(ctx, req)
	if sendErr != nil {
		s.storage.UpdateDraftStatus(ctx, draftID, ports.DraftStatusFailed)
		return nil, sendErr
	}

	// Update draft status to sent
	s.storage.UpdateDraftStatus(ctx, draftID, ports.DraftStatusSent)

	return result, nil
}

// GetSignature returns the configured email signature
func (s *SendService) GetSignature(ctx context.Context) (string, error) {
	s.mu.RLock()
	var method = s.sendMethod
	s.mu.RUnlock()

	if method == ports.SendMethodGmailAPI {
		return s.gmailAPI.GetSignature(ctx)
	}

	// For SMTP, we don't have signature support
	return "", nil
}

// parseAddresses splits comma-separated addresses
func parseAddresses(addresses string) []string {
	if addresses == "" {
		return nil
	}

	var result []string
	var parts = splitAndTrim(addresses, ",")
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// splitAndTrim splits a string and trims whitespace
func splitAndTrim(s string, sep string) []string {
	var parts = make([]string, 0)
	var current = ""
	for _, c := range s {
		if string(c) == sep {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	// Trim whitespace
	for i, p := range parts {
		var trimmed = ""
		var start, end = 0, len(p) - 1
		for start <= end && (p[start] == ' ' || p[start] == '\t') {
			start++
		}
		for end >= start && (p[end] == ' ' || p[end] == '\t') {
			end--
		}
		if start <= end {
			trimmed = p[start : end+1]
		}
		parts[i] = trimmed
	}

	return parts
}
