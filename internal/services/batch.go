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

// ArchiveSelected creates and executes a batch archive operation for selected emails
func (s *BatchService) ArchiveSelected(ctx context.Context, emailIDs []int64) error {
	var op = &ports.BatchOperation{
		Operation:   ports.BatchOpArchive,
		Description: fmt.Sprintf("Archive %d selected emails", len(emailIDs)),
		EmailIDs:    emailIDs,
		EmailCount:  len(emailIDs),
	}

	var created, err = s.CreateBatchOp(ctx, op)
	if err != nil {
		return err
	}

	return s.ConfirmBatchOp(ctx, created.ID)
}

// DeleteSelected creates and executes a batch delete operation for selected emails
func (s *BatchService) DeleteSelected(ctx context.Context, emailIDs []int64) error {
	var op = &ports.BatchOperation{
		Operation:   ports.BatchOpDelete,
		Description: fmt.Sprintf("Delete %d selected emails", len(emailIDs)),
		EmailIDs:    emailIDs,
		EmailCount:  len(emailIDs),
	}

	var created, err = s.CreateBatchOp(ctx, op)
	if err != nil {
		return err
	}

	return s.ConfirmBatchOp(ctx, created.ID)
}

// MarkReadSelected creates and executes a batch mark-as-read operation
func (s *BatchService) MarkReadSelected(ctx context.Context, emailIDs []int64, read bool) error {
	var opType = ports.BatchOpMarkRead
	var desc = fmt.Sprintf("Mark %d emails as read", len(emailIDs))
	if !read {
		opType = ports.BatchOpMarkUnread
		desc = fmt.Sprintf("Mark %d emails as unread", len(emailIDs))
	}

	var op = &ports.BatchOperation{
		Operation:   opType,
		Description: desc,
		EmailIDs:    emailIDs,
		EmailCount:  len(emailIDs),
	}

	var created, err = s.CreateBatchOp(ctx, op)
	if err != nil {
		return err
	}

	return s.ConfirmBatchOp(ctx, created.ID)
}

// StarSelected creates and executes a batch star/unstar operation
func (s *BatchService) StarSelected(ctx context.Context, emailIDs []int64, starred bool) error {
	var opType = ports.BatchOpStar
	var desc = fmt.Sprintf("Star %d emails", len(emailIDs))
	if !starred {
		opType = ports.BatchOpUnstar
		desc = fmt.Sprintf("Unstar %d emails", len(emailIDs))
	}

	var op = &ports.BatchOperation{
		Operation:   opType,
		Description: desc,
		EmailIDs:    emailIDs,
		EmailCount:  len(emailIDs),
	}

	var created, err = s.CreateBatchOp(ctx, op)
	if err != nil {
		return err
	}

	return s.ConfirmBatchOp(ctx, created.ID)
}

// ForwardSelected creates a batch forward operation (requires confirmation with recipient)
func (s *BatchService) ForwardSelected(ctx context.Context, emailIDs []int64, forwardTo string) error {
	var op = &ports.BatchOperation{
		Operation:   ports.BatchOpForward,
		Description: fmt.Sprintf("Forward %d emails to %s", len(emailIDs), forwardTo),
		EmailIDs:    emailIDs,
		EmailCount:  len(emailIDs),
		ForwardTo:   forwardTo,
	}

	var created, err = s.CreateBatchOp(ctx, op)
	if err != nil {
		return err
	}

	return s.ConfirmBatchOp(ctx, created.ID)
}
