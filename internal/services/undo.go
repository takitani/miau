package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/opik/miau/internal/ports"
)

// UndoServiceImpl implements ports.UndoService
type UndoServiceImpl struct {
	mu         sync.RWMutex
	storage    ports.StoragePort
	imap       ports.IMAPPort
	account    *ports.AccountInfo
	undoStack  []ports.Operation
	redoStack  []ports.Operation
	maxHistory int
}

// NewUndoService creates a new UndoService
func NewUndoService(storage ports.StoragePort, imap ports.IMAPPort) *UndoServiceImpl {
	return &UndoServiceImpl{
		storage:    storage,
		imap:       imap,
		undoStack:  make([]ports.Operation, 0),
		redoStack:  make([]ports.Operation, 0),
		maxHistory: 100, // default limit
	}
}

// SetAccount sets the current account
func (s *UndoServiceImpl) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
	// Load history from storage
	s.loadHistory()
}

// RecordOperation records an operation after it has been executed
func (s *UndoServiceImpl) RecordOperation(ctx context.Context, op ports.Operation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Add to undo stack
	s.undoStack = append(s.undoStack, op)

	// Clear redo stack (new operation invalidates redo history)
	s.redoStack = make([]ports.Operation, 0)

	// Enforce max history limit
	if len(s.undoStack) > s.maxHistory {
		s.undoStack = s.undoStack[1:]
	}

	// Persist to storage
	return s.saveOperation(ctx, op, "undo", len(s.undoStack)-1)
}

// Undo undoes the last operation
func (s *UndoServiceImpl) Undo(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.undoStack) == 0 {
		return fmt.Errorf("nothing to undo")
	}

	// Pop from undo stack
	op := s.undoStack[len(s.undoStack)-1]
	s.undoStack = s.undoStack[:len(s.undoStack)-1]

	// Execute undo
	if err := op.Undo(ctx); err != nil {
		// Re-add to undo stack on failure
		s.undoStack = append(s.undoStack, op)
		return fmt.Errorf("undo failed: %w", err)
	}

	// Add to redo stack
	s.redoStack = append(s.redoStack, op)

	// Persist to storage
	if err := s.saveOperation(ctx, op, "redo", len(s.redoStack)-1); err != nil {
		return err
	}

	// Remove from undo storage
	return s.removeOperation(ctx, op, "undo")
}

// Redo redoes the last undone operation
func (s *UndoServiceImpl) Redo(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.redoStack) == 0 {
		return fmt.Errorf("nothing to redo")
	}

	// Pop from redo stack
	op := s.redoStack[len(s.redoStack)-1]
	s.redoStack = s.redoStack[:len(s.redoStack)-1]

	// Execute redo
	if err := op.Execute(ctx); err != nil {
		// Re-add to redo stack on failure
		s.redoStack = append(s.redoStack, op)
		return fmt.Errorf("redo failed: %w", err)
	}

	// Add to undo stack
	s.undoStack = append(s.undoStack, op)

	// Persist to storage
	if err := s.saveOperation(ctx, op, "undo", len(s.undoStack)-1); err != nil {
		return err
	}

	// Remove from redo storage
	return s.removeOperation(ctx, op, "redo")
}

// CanUndo returns true if there are operations to undo
func (s *UndoServiceImpl) CanUndo(ctx context.Context) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.undoStack) > 0
}

// CanRedo returns true if there are operations to redo
func (s *UndoServiceImpl) CanRedo(ctx context.Context) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.redoStack) > 0
}

// GetUndoDescription returns the description of the next undo operation
func (s *UndoServiceImpl) GetUndoDescription(ctx context.Context) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.undoStack) == 0 {
		return ""
	}

	return s.undoStack[len(s.undoStack)-1].Description()
}

// GetRedoDescription returns the description of the next redo operation
func (s *UndoServiceImpl) GetRedoDescription(ctx context.Context) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.redoStack) == 0 {
		return ""
	}

	return s.redoStack[len(s.redoStack)-1].Description()
}

// ClearHistory clears all undo/redo history
func (s *UndoServiceImpl) ClearHistory(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.undoStack = make([]ports.Operation, 0)
	s.redoStack = make([]ports.Operation, 0)

	// Clear from storage
	if s.account == nil {
		return fmt.Errorf("no account set")
	}

	return s.storage.ClearOperationsHistory(ctx, s.account.ID)
}

// GetHistorySize returns the number of operations in undo and redo stacks
func (s *UndoServiceImpl) GetHistorySize(ctx context.Context) (undoCount, redoCount int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.undoStack), len(s.redoStack)
}

// saveOperation saves an operation to storage
func (s *UndoServiceImpl) saveOperation(ctx context.Context, op ports.Operation, stackType string, position int) error {
	if s.account == nil {
		return nil // skip if no account set
	}

	data, err := op.Data()
	if err != nil {
		return fmt.Errorf("failed to serialize operation data: %w", err)
	}

	return s.storage.SaveOperation(ctx, &ports.OperationRecord{
		AccountID:     s.account.ID,
		OperationType: string(op.Type()),
		OperationData: data,
		Description:   op.Description(),
		StackType:     stackType,
		StackPosition: position,
	})
}

// removeOperation removes an operation from storage
func (s *UndoServiceImpl) removeOperation(ctx context.Context, op ports.Operation, stackType string) error {
	if s.account == nil {
		return nil
	}

	// Get operation data for matching
	data, err := op.Data()
	if err != nil {
		return err
	}

	return s.storage.RemoveOperation(ctx, s.account.ID, stackType, data)
}

// loadHistory loads operation history from storage
func (s *UndoServiceImpl) loadHistory() {
	if s.account == nil {
		return
	}

	ctx := context.Background()

	// Load undo stack
	undoRecords, err := s.storage.GetOperations(ctx, s.account.ID, "undo")
	if err == nil {
		s.undoStack = s.reconstructOperations(undoRecords)
	}

	// Load redo stack
	redoRecords, err := s.storage.GetOperations(ctx, s.account.ID, "redo")
	if err == nil {
		s.redoStack = s.reconstructOperations(redoRecords)
	}
}

// reconstructOperations reconstructs operations from storage records
func (s *UndoServiceImpl) reconstructOperations(records []ports.OperationRecord) []ports.Operation {
	operations := make([]ports.Operation, 0, len(records))

	for _, record := range records {
		op := s.reconstructOperation(record)
		if op != nil {
			operations = append(operations, op)
		}
	}

	return operations
}

// reconstructOperation reconstructs a single operation from a record
func (s *UndoServiceImpl) reconstructOperation(record ports.OperationRecord) ports.Operation {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(record.OperationData), &data); err != nil {
		return nil
	}

	switch ports.OperationType(record.OperationType) {
	case ports.OperationTypeMarkRead:
		return s.reconstructMarkReadOp(data)
	case ports.OperationTypeMarkStarred:
		return s.reconstructMarkStarredOp(data)
	case ports.OperationTypeArchive:
		return s.reconstructArchiveOp(data)
	case ports.OperationTypeDelete:
		return s.reconstructDeleteOp(data)
	case ports.OperationTypeMove:
		return s.reconstructMoveOp(data)
	default:
		return nil
	}
}

func (s *UndoServiceImpl) reconstructMarkReadOp(data map[string]interface{}) ports.Operation {
	return NewMarkReadOperation(
		int64(data["email_id"].(float64)),
		data["new_state"].(bool),
		data["old_state"].(bool),
		data["subject"].(string),
		uint32(data["email_uid"].(float64)),
		s.storage,
		s.imap,
	)
}

func (s *UndoServiceImpl) reconstructMarkStarredOp(data map[string]interface{}) ports.Operation {
	return NewMarkStarredOperation(
		int64(data["email_id"].(float64)),
		data["new_state"].(bool),
		data["old_state"].(bool),
		data["subject"].(string),
		s.storage,
	)
}

func (s *UndoServiceImpl) reconstructArchiveOp(data map[string]interface{}) ports.Operation {
	wasArchived := false
	if val, ok := data["was_archived"]; ok {
		wasArchived = val.(bool)
	}

	return NewArchiveOperation(
		int64(data["email_id"].(float64)),
		data["subject"].(string),
		uint32(data["email_uid"].(float64)),
		wasArchived,
		s.storage,
		s.imap,
	)
}

func (s *UndoServiceImpl) reconstructDeleteOp(data map[string]interface{}) ports.Operation {
	wasDeleted := false
	if val, ok := data["was_deleted"]; ok {
		wasDeleted = val.(bool)
	}

	return NewDeleteOperation(
		int64(data["email_id"].(float64)),
		data["subject"].(string),
		uint32(data["email_uid"].(float64)),
		wasDeleted,
		s.storage,
		s.imap,
	)
}

func (s *UndoServiceImpl) reconstructMoveOp(data map[string]interface{}) ports.Operation {
	return NewMoveOperation(
		int64(data["email_id"].(float64)),
		data["subject"].(string),
		data["from_folder"].(string),
		data["to_folder"].(string),
		uint32(data["email_uid"].(float64)),
		s.storage,
		s.imap,
	)
}

// Ensure UndoServiceImpl implements ports.UndoService
var _ ports.UndoService = (*UndoServiceImpl)(nil)

// OperationRecord is stored in the database
// (this should be defined in ports/types.go but we define it here for now)
func init() {
	// Verify interface compliance at compile time
	var _ ports.UndoService = (*UndoServiceImpl)(nil)
}

// Add storage interface methods (these need to be implemented in storage package)
type undoStorage interface {
	SaveOperation(ctx context.Context, op *ports.OperationRecord) error
	RemoveOperation(ctx context.Context, accountID int64, stackType, data string) error
	GetOperations(ctx context.Context, accountID int64, stackType string) ([]ports.OperationRecord, error)
	ClearOperationsHistory(ctx context.Context, accountID int64) error
}

// Verify storage implements undo storage methods
var _ interface {
	SaveOperation(ctx context.Context, op *ports.OperationRecord) error
	RemoveOperation(ctx context.Context, accountID int64, stackType, data string) error
	GetOperations(ctx context.Context, accountID int64, stackType string) ([]ports.OperationRecord, error)
	ClearOperationsHistory(ctx context.Context, accountID int64) error
} = (ports.StoragePort)(nil)
