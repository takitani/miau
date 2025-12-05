package ports

import "context"

// Operation represents a reversible operation (Command Pattern)
type Operation interface {
	// Execute performs the operation
	Execute(ctx context.Context) error

	// Undo reverses the operation
	Undo(ctx context.Context) error

	// Description returns a human-readable description
	Description() string

	// Type returns the operation type
	Type() OperationType

	// Data returns the operation data as JSON
	Data() (string, error)
}

// OperationType defines the type of operation
type OperationType string

const (
	OperationTypeMarkRead    OperationType = "mark_read"
	OperationTypeMarkStarred OperationType = "mark_starred"
	OperationTypeArchive     OperationType = "archive"
	OperationTypeDelete      OperationType = "delete"
	OperationTypeMove        OperationType = "move"
	OperationTypeBatch       OperationType = "batch"
)

// UndoService manages undo/redo operations
type UndoService interface {
	// RecordOperation records an operation after it has been executed
	RecordOperation(ctx context.Context, op Operation) error

	// Undo undoes the last operation
	Undo(ctx context.Context) error

	// Redo redoes the last undone operation
	Redo(ctx context.Context) error

	// CanUndo returns true if there are operations to undo
	CanUndo(ctx context.Context) bool

	// CanRedo returns true if there are operations to redo
	CanRedo(ctx context.Context) bool

	// GetUndoDescription returns the description of the next undo operation
	GetUndoDescription(ctx context.Context) string

	// GetRedoDescription returns the description of the next redo operation
	GetRedoDescription(ctx context.Context) string

	// ClearHistory clears all undo/redo history
	ClearHistory(ctx context.Context) error

	// GetHistorySize returns the number of operations in undo and redo stacks
	GetHistorySize(ctx context.Context) (undoCount, redoCount int)
}
