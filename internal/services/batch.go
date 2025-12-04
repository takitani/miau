package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/opik/miau/internal/ports"
)

// BatchService implements ports.BatchService
type BatchService struct {
	mu      sync.RWMutex
	storage ports.StoragePort
	events  ports.EventBus
	account *ports.AccountInfo
}

// NewBatchService creates a new BatchService
func NewBatchService(storage ports.StoragePort, events ports.EventBus) *BatchService {
	return &BatchService{
		storage: storage,
		events:  events,
	}
}

// SetAccount sets the current account
func (s *BatchService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// CreateBatchOp creates a pending batch operation
func (s *BatchService) CreateBatchOp(ctx context.Context, op *ports.BatchOperation) (*ports.BatchOperation, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	op.Status = ports.BatchOpStatusPending

	var created, err = s.storage.CreateBatchOp(ctx, account.ID, op)
	if err != nil {
		return nil, err
	}

	s.events.Publish(ports.BatchCreatedEvent{
		BaseEvent: ports.NewBaseEvent(ports.EventTypeBatchCreated),
		Operation: created,
	})

	return created, nil
}

// GetPendingBatchOp returns the current pending batch operation if any
func (s *BatchService) GetPendingBatchOp(ctx context.Context) (*ports.BatchOperation, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	return s.storage.GetPendingBatchOp(ctx, account.ID)
}

// ConfirmBatchOp confirms and executes a batch operation
func (s *BatchService) ConfirmBatchOp(ctx context.Context, id int64) error {
	// Update status to confirmed
	if err := s.storage.UpdateBatchOpStatus(ctx, id, ports.BatchOpStatusConfirmed); err != nil {
		return err
	}

	// Execute the operation
	if err := s.storage.ExecuteBatchOp(ctx, id); err != nil {
		return err
	}

	// Update status to executed
	if err := s.storage.UpdateBatchOpStatus(ctx, id, ports.BatchOpStatusExecuted); err != nil {
		return err
	}

	s.events.Publish(ports.BaseEvent{
		EventType: ports.EventTypeBatchExecuted,
	})

	return nil
}

// CancelBatchOp cancels a batch operation
func (s *BatchService) CancelBatchOp(ctx context.Context, id int64) error {
	if err := s.storage.UpdateBatchOpStatus(ctx, id, ports.BatchOpStatusCancelled); err != nil {
		return err
	}

	s.events.Publish(ports.BaseEvent{
		EventType: ports.EventTypeBatchCancelled,
	})

	return nil
}

// GetBatchOpEmails returns the emails affected by a batch operation
func (s *BatchService) GetBatchOpEmails(ctx context.Context, id int64) ([]ports.EmailMetadata, error) {
	// This would be implemented by joining batch_ops with emails table
	// For now, return empty as placeholder
	return nil, nil
}
