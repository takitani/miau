package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opik/miau/internal/ports"
)

// MarkReadOperation represents a mark as read/unread operation
type MarkReadOperation struct {
	emailID     int64
	newState    bool
	oldState    bool
	subject     string
	storage     ports.StoragePort
	imap        ports.IMAPPort
	emailUID    uint32
}

func NewMarkReadOperation(
	emailID int64,
	newState bool,
	oldState bool,
	subject string,
	emailUID uint32,
	storage ports.StoragePort,
	imap ports.IMAPPort,
) *MarkReadOperation {
	return &MarkReadOperation{
		emailID:  emailID,
		newState: newState,
		oldState: oldState,
		subject:  subject,
		emailUID: emailUID,
		storage:  storage,
		imap:     imap,
	}
}

func (o *MarkReadOperation) Execute(ctx context.Context) error {
	// Mark on IMAP server
	if o.newState {
		if err := o.imap.MarkAsRead(ctx, o.emailUID); err != nil {
			return err
		}
	} else {
		if err := o.imap.MarkAsUnread(ctx, o.emailUID); err != nil {
			return err
		}
	}

	// Mark in storage
	return o.storage.MarkAsRead(ctx, o.emailID, o.newState)
}

func (o *MarkReadOperation) Undo(ctx context.Context) error {
	// Revert on IMAP server
	if o.oldState {
		if err := o.imap.MarkAsRead(ctx, o.emailUID); err != nil {
			return err
		}
	} else {
		if err := o.imap.MarkAsUnread(ctx, o.emailUID); err != nil {
			return err
		}
	}

	// Revert in storage
	return o.storage.MarkAsRead(ctx, o.emailID, o.oldState)
}

func (o *MarkReadOperation) Description() string {
	if o.newState {
		return fmt.Sprintf("Marcar como lido: '%s'", truncate(o.subject, 50))
	}
	return fmt.Sprintf("Marcar como não lido: '%s'", truncate(o.subject, 50))
}

func (o *MarkReadOperation) Type() ports.OperationType {
	return ports.OperationTypeMarkRead
}

func (o *MarkReadOperation) Data() (string, error) {
	data := map[string]interface{}{
		"email_id":  o.emailID,
		"new_state": o.newState,
		"old_state": o.oldState,
		"subject":   o.subject,
		"email_uid": o.emailUID,
	}
	bytes, err := json.Marshal(data)
	return string(bytes), err
}

// MarkStarredOperation represents a mark as starred/unstarred operation
type MarkStarredOperation struct {
	emailID  int64
	newState bool
	oldState bool
	subject  string
	storage  ports.StoragePort
}

func NewMarkStarredOperation(
	emailID int64,
	newState bool,
	oldState bool,
	subject string,
	storage ports.StoragePort,
) *MarkStarredOperation {
	return &MarkStarredOperation{
		emailID:  emailID,
		newState: newState,
		oldState: oldState,
		subject:  subject,
		storage:  storage,
	}
}

func (o *MarkStarredOperation) Execute(ctx context.Context) error {
	return o.storage.MarkAsStarred(ctx, o.emailID, o.newState)
}

func (o *MarkStarredOperation) Undo(ctx context.Context) error {
	return o.storage.MarkAsStarred(ctx, o.emailID, o.oldState)
}

func (o *MarkStarredOperation) Description() string {
	if o.newState {
		return fmt.Sprintf("Marcar como favorito: '%s'", truncate(o.subject, 50))
	}
	return fmt.Sprintf("Remover favorito: '%s'", truncate(o.subject, 50))
}

func (o *MarkStarredOperation) Type() ports.OperationType {
	return ports.OperationTypeMarkStarred
}

func (o *MarkStarredOperation) Data() (string, error) {
	data := map[string]interface{}{
		"email_id":  o.emailID,
		"new_state": o.newState,
		"old_state": o.oldState,
		"subject":   o.subject,
	}
	bytes, err := json.Marshal(data)
	return string(bytes), err
}

// ArchiveOperation represents an archive operation
type ArchiveOperation struct {
	emailID     int64
	subject     string
	storage     ports.StoragePort
	imap        ports.IMAPPort
	emailUID    uint32
	wasArchived bool
}

func NewArchiveOperation(
	emailID int64,
	subject string,
	emailUID uint32,
	wasArchived bool,
	storage ports.StoragePort,
	imap ports.IMAPPort,
) *ArchiveOperation {
	return &ArchiveOperation{
		emailID:     emailID,
		subject:     subject,
		emailUID:    emailUID,
		wasArchived: wasArchived,
		storage:     storage,
		imap:        imap,
	}
}

func (o *ArchiveOperation) Execute(ctx context.Context) error {
	// Archive on IMAP
	if err := o.imap.Archive(ctx, o.emailUID); err != nil {
		return err
	}

	// Mark as archived in storage
	return o.storage.MarkAsArchived(ctx, o.emailID, true)
}

func (o *ArchiveOperation) Undo(ctx context.Context) error {
	// Unarchive on IMAP (move back to INBOX)
	if err := o.imap.MoveToFolder(ctx, o.emailUID, "INBOX"); err != nil {
		return err
	}

	// Unmark as archived in storage
	return o.storage.MarkAsArchived(ctx, o.emailID, false)
}

func (o *ArchiveOperation) Description() string {
	return fmt.Sprintf("Arquivar email: '%s'", truncate(o.subject, 50))
}

func (o *ArchiveOperation) Type() ports.OperationType {
	return ports.OperationTypeArchive
}

func (o *ArchiveOperation) Data() (string, error) {
	data := map[string]interface{}{
		"email_id":     o.emailID,
		"subject":      o.subject,
		"email_uid":    o.emailUID,
		"was_archived": o.wasArchived,
	}
	bytes, err := json.Marshal(data)
	return string(bytes), err
}

// DeleteOperation represents a delete operation
type DeleteOperation struct {
	emailID    int64
	subject    string
	storage    ports.StoragePort
	imap       ports.IMAPPort
	emailUID   uint32
	wasDeleted bool
}

func NewDeleteOperation(
	emailID int64,
	subject string,
	emailUID uint32,
	wasDeleted bool,
	storage ports.StoragePort,
	imap ports.IMAPPort,
) *DeleteOperation {
	return &DeleteOperation{
		emailID:    emailID,
		subject:    subject,
		emailUID:   emailUID,
		wasDeleted: wasDeleted,
		storage:    storage,
		imap:       imap,
	}
}

func (o *DeleteOperation) Execute(ctx context.Context) error {
	// Delete on IMAP
	if err := o.imap.Delete(ctx, o.emailUID); err != nil {
		return err
	}

	// Mark as deleted in storage
	return o.storage.MarkAsDeleted(ctx, o.emailID, true)
}

func (o *DeleteOperation) Undo(ctx context.Context) error {
	// Undelete on IMAP (remove deleted flag)
	if err := o.imap.Undelete(ctx, o.emailUID); err != nil {
		return err
	}

	// Unmark as deleted in storage
	return o.storage.MarkAsDeleted(ctx, o.emailID, false)
}

func (o *DeleteOperation) Description() string {
	return fmt.Sprintf("Deletar email: '%s'", truncate(o.subject, 50))
}

func (o *DeleteOperation) Type() ports.OperationType {
	return ports.OperationTypeDelete
}

func (o *DeleteOperation) Data() (string, error) {
	data := map[string]interface{}{
		"email_id":    o.emailID,
		"subject":     o.subject,
		"email_uid":   o.emailUID,
		"was_deleted": o.wasDeleted,
	}
	bytes, err := json.Marshal(data)
	return string(bytes), err
}

// MoveOperation represents a move to folder operation
type MoveOperation struct {
	emailID    int64
	subject    string
	fromFolder string
	toFolder   string
	storage    ports.StoragePort
	imap       ports.IMAPPort
	emailUID   uint32
}

func NewMoveOperation(
	emailID int64,
	subject string,
	fromFolder string,
	toFolder string,
	emailUID uint32,
	storage ports.StoragePort,
	imap ports.IMAPPort,
) *MoveOperation {
	return &MoveOperation{
		emailID:    emailID,
		subject:    subject,
		fromFolder: fromFolder,
		toFolder:   toFolder,
		emailUID:   emailUID,
		storage:    storage,
		imap:       imap,
	}
}

func (o *MoveOperation) Execute(ctx context.Context) error {
	return o.imap.MoveToFolder(ctx, o.emailUID, o.toFolder)
}

func (o *MoveOperation) Undo(ctx context.Context) error {
	return o.imap.MoveToFolder(ctx, o.emailUID, o.fromFolder)
}

func (o *MoveOperation) Description() string {
	return fmt.Sprintf("Mover '%s' de %s para %s", truncate(o.subject, 30), o.fromFolder, o.toFolder)
}

func (o *MoveOperation) Type() ports.OperationType {
	return ports.OperationTypeMove
}

func (o *MoveOperation) Data() (string, error) {
	data := map[string]interface{}{
		"email_id":    o.emailID,
		"subject":     o.subject,
		"from_folder": o.fromFolder,
		"to_folder":   o.toFolder,
		"email_uid":   o.emailUID,
	}
	bytes, err := json.Marshal(data)
	return string(bytes), err
}

// BatchOperation represents a batch operation (composite)
type BatchOperation struct {
	operations  []ports.Operation
	description string
}

func NewBatchOperation(operations []ports.Operation, description string) *BatchOperation {
	return &BatchOperation{
		operations:  operations,
		description: description,
	}
}

func (o *BatchOperation) Execute(ctx context.Context) error {
	for _, op := range o.operations {
		if err := op.Execute(ctx); err != nil {
			return fmt.Errorf("batch operation failed: %w", err)
		}
	}
	return nil
}

func (o *BatchOperation) Undo(ctx context.Context) error {
	// Undo in reverse order
	for i := len(o.operations) - 1; i >= 0; i-- {
		if err := o.operations[i].Undo(ctx); err != nil {
			return fmt.Errorf("batch undo failed: %w", err)
		}
	}
	return nil
}

func (o *BatchOperation) Description() string {
	return o.description
}

func (o *BatchOperation) Type() ports.OperationType {
	return ports.OperationTypeBatch
}

func (o *BatchOperation) Data() (string, error) {
	data := map[string]interface{}{
		"description":     o.description,
		"operation_count": len(o.operations),
	}
	bytes, err := json.Marshal(data)
	return string(bytes), err
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
