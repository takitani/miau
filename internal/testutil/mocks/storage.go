package mocks

import (
	"context"
	"time"

	"github.com/opik/miau/internal/ports"
	"github.com/stretchr/testify/mock"
)

// StoragePort is a mock implementation of ports.StoragePort
type StoragePort struct {
	mock.Mock
}

// Account operations
func (m *StoragePort) GetOrCreateAccount(ctx context.Context, email, name string) (*ports.AccountInfo, error) {
	var args = m.Called(ctx, email, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.AccountInfo), args.Error(1)
}

func (m *StoragePort) GetAccount(ctx context.Context, id int64) (*ports.AccountInfo, error) {
	var args = m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.AccountInfo), args.Error(1)
}

// Folder operations
func (m *StoragePort) UpsertFolder(ctx context.Context, accountID int64, folder *ports.Folder) error {
	var args = m.Called(ctx, accountID, folder)
	return args.Error(0)
}

func (m *StoragePort) GetFolders(ctx context.Context, accountID int64) ([]ports.Folder, error) {
	var args = m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.Folder), args.Error(1)
}

func (m *StoragePort) GetFolderByName(ctx context.Context, accountID int64, name string) (*ports.Folder, error) {
	var args = m.Called(ctx, accountID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.Folder), args.Error(1)
}

func (m *StoragePort) UpdateFolderStats(ctx context.Context, folderID int64, total, unread int) error {
	var args = m.Called(ctx, folderID, total, unread)
	return args.Error(0)
}

// Email operations
func (m *StoragePort) UpsertEmail(ctx context.Context, accountID, folderID int64, email *ports.EmailContent) error {
	var args = m.Called(ctx, accountID, folderID, email)
	return args.Error(0)
}

func (m *StoragePort) GetEmails(ctx context.Context, folderID int64, limit int) ([]ports.EmailMetadata, error) {
	var args = m.Called(ctx, folderID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.EmailMetadata), args.Error(1)
}

func (m *StoragePort) GetEmail(ctx context.Context, id int64) (*ports.EmailContent, error) {
	var args = m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.EmailContent), args.Error(1)
}

func (m *StoragePort) GetEmailByUID(ctx context.Context, folderID int64, uid uint32) (*ports.EmailContent, error) {
	var args = m.Called(ctx, folderID, uid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.EmailContent), args.Error(1)
}

func (m *StoragePort) GetLatestUID(ctx context.Context, folderID int64) (uint32, error) {
	var args = m.Called(ctx, folderID)
	return args.Get(0).(uint32), args.Error(1)
}

func (m *StoragePort) GetAllUIDs(ctx context.Context, folderID int64) ([]uint32, error) {
	var args = m.Called(ctx, folderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint32), args.Error(1)
}

// Email status updates
func (m *StoragePort) MarkAsRead(ctx context.Context, id int64, read bool) error {
	var args = m.Called(ctx, id, read)
	return args.Error(0)
}

func (m *StoragePort) MarkAsStarred(ctx context.Context, id int64, starred bool) error {
	var args = m.Called(ctx, id, starred)
	return args.Error(0)
}

func (m *StoragePort) MarkAsArchived(ctx context.Context, id int64, archived bool) error {
	var args = m.Called(ctx, id, archived)
	return args.Error(0)
}

func (m *StoragePort) MarkAsDeleted(ctx context.Context, id int64, deleted bool) error {
	var args = m.Called(ctx, id, deleted)
	return args.Error(0)
}

func (m *StoragePort) MarkAsReplied(ctx context.Context, id int64, replied bool) error {
	var args = m.Called(ctx, id, replied)
	return args.Error(0)
}

// Bulk operations
func (m *StoragePort) MarkDeletedByUIDs(ctx context.Context, folderID int64, uids []uint32) error {
	var args = m.Called(ctx, folderID, uids)
	return args.Error(0)
}

func (m *StoragePort) BulkMarkAsRead(ctx context.Context, ids []int64, read bool) error {
	var args = m.Called(ctx, ids, read)
	return args.Error(0)
}

func (m *StoragePort) BulkMarkAsArchived(ctx context.Context, ids []int64) error {
	var args = m.Called(ctx, ids)
	return args.Error(0)
}

func (m *StoragePort) BulkMarkAsDeleted(ctx context.Context, ids []int64) error {
	var args = m.Called(ctx, ids)
	return args.Error(0)
}

// Search
func (m *StoragePort) SearchEmails(ctx context.Context, accountID int64, query string, limit int) ([]ports.EmailMetadata, error) {
	var args = m.Called(ctx, accountID, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.EmailMetadata), args.Error(1)
}

func (m *StoragePort) SearchEmailsInFolder(ctx context.Context, folderID int64, query string, limit int) ([]ports.EmailMetadata, error) {
	var args = m.Called(ctx, folderID, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.EmailMetadata), args.Error(1)
}

// Draft operations
func (m *StoragePort) CreateDraft(ctx context.Context, accountID int64, draft *ports.Draft) (*ports.Draft, error) {
	var args = m.Called(ctx, accountID, draft)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.Draft), args.Error(1)
}

func (m *StoragePort) UpdateDraft(ctx context.Context, draft *ports.Draft) error {
	var args = m.Called(ctx, draft)
	return args.Error(0)
}

func (m *StoragePort) GetDraft(ctx context.Context, id int64) (*ports.Draft, error) {
	var args = m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.Draft), args.Error(1)
}

func (m *StoragePort) GetDrafts(ctx context.Context, accountID int64) ([]ports.Draft, error) {
	var args = m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.Draft), args.Error(1)
}

func (m *StoragePort) GetPendingDrafts(ctx context.Context, accountID int64) ([]ports.Draft, error) {
	var args = m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.Draft), args.Error(1)
}

func (m *StoragePort) DeleteDraft(ctx context.Context, id int64) error {
	var args = m.Called(ctx, id)
	return args.Error(0)
}

func (m *StoragePort) UpdateDraftStatus(ctx context.Context, id int64, status ports.DraftStatus) error {
	var args = m.Called(ctx, id, status)
	return args.Error(0)
}

// Batch operations
func (m *StoragePort) CreateBatchOp(ctx context.Context, accountID int64, op *ports.BatchOperation) (*ports.BatchOperation, error) {
	var args = m.Called(ctx, accountID, op)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.BatchOperation), args.Error(1)
}

func (m *StoragePort) GetPendingBatchOp(ctx context.Context, accountID int64) (*ports.BatchOperation, error) {
	var args = m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.BatchOperation), args.Error(1)
}

func (m *StoragePort) UpdateBatchOpStatus(ctx context.Context, id int64, status ports.BatchOpStatus) error {
	var args = m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *StoragePort) ExecuteBatchOp(ctx context.Context, id int64) error {
	var args = m.Called(ctx, id)
	return args.Error(0)
}

// Sent email tracking
func (m *StoragePort) TrackSentEmail(ctx context.Context, accountID int64, messageID, to, subject string) error {
	var args = m.Called(ctx, accountID, messageID, to, subject)
	return args.Error(0)
}

func (m *StoragePort) GetRecentSentEmails(ctx context.Context, accountID int64, since time.Duration) ([]ports.SentEmailTrack, error) {
	var args = m.Called(ctx, accountID, since)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.SentEmailTrack), args.Error(1)
}

// Index state
func (m *StoragePort) GetIndexState(ctx context.Context, accountID int64) (*ports.IndexState, error) {
	var args = m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.IndexState), args.Error(1)
}

func (m *StoragePort) UpdateIndexState(ctx context.Context, accountID int64, state *ports.IndexState) error {
	var args = m.Called(ctx, accountID, state)
	return args.Error(0)
}

// Settings
func (m *StoragePort) GetSetting(ctx context.Context, accountID int64, key string) (string, error) {
	var args = m.Called(ctx, accountID, key)
	return args.String(0), args.Error(1)
}

func (m *StoragePort) SetSetting(ctx context.Context, accountID int64, key, value string) error {
	var args = m.Called(ctx, accountID, key, value)
	return args.Error(0)
}

// Ensure StoragePort implements ports.StoragePort
var _ ports.StoragePort = (*StoragePort)(nil)
