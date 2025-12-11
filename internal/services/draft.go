package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/opik/miau/internal/ports"
)

// DraftService implements ports.DraftService
type DraftService struct {
	mu      sync.RWMutex
	storage ports.StoragePort
	events  ports.EventBus
	account *ports.AccountInfo
}

// NewDraftService creates a new DraftService
func NewDraftService(storage ports.StoragePort, events ports.EventBus) *DraftService {
	return &DraftService{
		storage: storage,
		events:  events,
	}
}

// SetAccount sets the current account
func (s *DraftService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// CreateDraft creates a new draft
func (s *DraftService) CreateDraft(ctx context.Context, draft *ports.Draft) (*ports.Draft, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	draft.Status = ports.DraftStatusDraft
	draft.CreatedAt = time.Now()
	draft.UpdatedAt = time.Now()

	var created, err = s.storage.CreateDraft(ctx, account.ID, draft)
	if err != nil {
		return nil, err
	}

	s.events.Publish(ports.BaseEvent{
		EventType: ports.EventTypeDraftCreated,
		Time:      time.Now(),
	})

	return created, nil
}

// UpdateDraft updates an existing draft
func (s *DraftService) UpdateDraft(ctx context.Context, draft *ports.Draft) error {
	draft.UpdatedAt = time.Now()
	return s.storage.UpdateDraft(ctx, draft)
}

// GetDraft gets a draft by ID
func (s *DraftService) GetDraft(ctx context.Context, id int64) (*ports.Draft, error) {
	return s.storage.GetDraft(ctx, id)
}

// ListDrafts lists all drafts for the current account
func (s *DraftService) ListDrafts(ctx context.Context) ([]ports.Draft, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	return s.storage.GetDrafts(ctx, account.ID)
}

// DeleteDraft deletes a draft
func (s *DraftService) DeleteDraft(ctx context.Context, id int64) error {
	return s.storage.DeleteDraft(ctx, id)
}

// ScheduleDraft schedules a draft for sending
func (s *DraftService) ScheduleDraft(ctx context.Context, id int64, sendAt *int64) error {
	var draft, err = s.storage.GetDraft(ctx, id)
	if err != nil {
		return err
	}

	draft.Status = ports.DraftStatusScheduled
	if sendAt != nil {
		var t = time.Unix(*sendAt, 0)
		draft.ScheduledSendAt = &t
	}
	draft.UpdatedAt = time.Now()

	if err := s.storage.UpdateDraft(ctx, draft); err != nil {
		return err
	}

	s.events.Publish(ports.BaseEvent{
		EventType: ports.EventTypeDraftScheduled,
		Time:      time.Now(),
	})

	return nil
}

// CancelScheduledDraft cancels a scheduled draft
func (s *DraftService) CancelScheduledDraft(ctx context.Context, id int64) error {
	var draft, err = s.storage.GetDraft(ctx, id)
	if err != nil {
		return err
	}

	draft.Status = ports.DraftStatusCancelled
	draft.ScheduledSendAt = nil
	draft.UpdatedAt = time.Now()

	if err := s.storage.UpdateDraft(ctx, draft); err != nil {
		return err
	}

	s.events.Publish(ports.BaseEvent{
		EventType: ports.EventTypeDraftCancelled,
		Time:      time.Now(),
	})

	return nil
}

// GetPendingDrafts returns drafts that are scheduled and ready to send
func (s *DraftService) GetPendingDrafts(ctx context.Context) ([]ports.Draft, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	return s.storage.GetPendingDrafts(ctx, account.ID)
}

// GetScheduledDrafts returns all scheduled drafts for the current account
func (s *DraftService) GetScheduledDrafts(ctx context.Context) ([]ports.Draft, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	return s.storage.GetScheduledDrafts(ctx, account.ID)
}
