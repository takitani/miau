package mocks

import (
	"context"
	"time"

	"github.com/opik/miau/internal/ports"
	"github.com/stretchr/testify/mock"
)

// IMAPPort is a mock implementation of ports.IMAPPort
type IMAPPort struct {
	mock.Mock
}

// Connection
func (m *IMAPPort) Connect(ctx context.Context) error {
	var args = m.Called(ctx)
	return args.Error(0)
}

func (m *IMAPPort) Close() error {
	var args = m.Called()
	return args.Error(0)
}

func (m *IMAPPort) IsConnected() bool {
	var args = m.Called()
	return args.Bool(0)
}

// Mailbox operations
func (m *IMAPPort) ListMailboxes(ctx context.Context) ([]ports.MailboxInfo, error) {
	var args = m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.MailboxInfo), args.Error(1)
}

func (m *IMAPPort) SelectMailbox(ctx context.Context, name string) (*ports.MailboxStatus, error) {
	var args = m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.MailboxStatus), args.Error(1)
}

// Email fetching
func (m *IMAPPort) FetchEmails(ctx context.Context, limit int) ([]ports.IMAPEmail, error) {
	var args = m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.IMAPEmail), args.Error(1)
}

func (m *IMAPPort) FetchNewEmails(ctx context.Context, sinceUID uint32, limit int) ([]ports.IMAPEmail, error) {
	var args = m.Called(ctx, sinceUID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.IMAPEmail), args.Error(1)
}

func (m *IMAPPort) FetchEmailRaw(ctx context.Context, uid uint32) ([]byte, error) {
	var args = m.Called(ctx, uid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *IMAPPort) FetchEmailBody(ctx context.Context, uid uint32) (string, error) {
	var args = m.Called(ctx, uid)
	return args.String(0), args.Error(1)
}

func (m *IMAPPort) GetAllUIDs(ctx context.Context) ([]uint32, error) {
	var args = m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint32), args.Error(1)
}

// Batch email fetching (optimized methods)
func (m *IMAPPort) SearchSince(ctx context.Context, sinceDate time.Time) ([]uint32, error) {
	var args = m.Called(ctx, sinceDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uint32), args.Error(1)
}

func (m *IMAPPort) FetchEmailsBatch(ctx context.Context, uids []uint32) ([]ports.IMAPEmail, error) {
	var args = m.Called(ctx, uids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.IMAPEmail), args.Error(1)
}

func (m *IMAPPort) FetchNewEmailsBatch(ctx context.Context, sinceUID uint32, limit int) ([]ports.IMAPEmail, error) {
	var args = m.Called(ctx, sinceUID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.IMAPEmail), args.Error(1)
}

func (m *IMAPPort) FetchEmailsSinceDateBatch(ctx context.Context, sinceDays int, limit int) ([]ports.IMAPEmail, error) {
	var args = m.Called(ctx, sinceDays, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]ports.IMAPEmail), args.Error(1)
}

// Email actions
func (m *IMAPPort) MarkAsRead(ctx context.Context, uid uint32) error {
	var args = m.Called(ctx, uid)
	return args.Error(0)
}

func (m *IMAPPort) MarkAsUnread(ctx context.Context, uid uint32) error {
	var args = m.Called(ctx, uid)
	return args.Error(0)
}

func (m *IMAPPort) Archive(ctx context.Context, uid uint32) error {
	var args = m.Called(ctx, uid)
	return args.Error(0)
}

func (m *IMAPPort) MoveToFolder(ctx context.Context, uid uint32, folder string) error {
	var args = m.Called(ctx, uid, folder)
	return args.Error(0)
}

func (m *IMAPPort) Delete(ctx context.Context, uid uint32) error {
	var args = m.Called(ctx, uid)
	return args.Error(0)
}

// Utility
func (m *IMAPPort) GetTrashFolder() string {
	var args = m.Called()
	return args.String(0)
}

// Attachments
func (m *IMAPPort) FetchAttachmentMetadata(ctx context.Context, uid uint32) ([]ports.AttachmentInfo, bool, error) {
	var args = m.Called(ctx, uid)
	if args.Get(0) == nil {
		return nil, args.Bool(1), args.Error(2)
	}
	return args.Get(0).([]ports.AttachmentInfo), args.Bool(1), args.Error(2)
}

func (m *IMAPPort) FetchAttachmentPart(ctx context.Context, uid uint32, partNumber string) ([]byte, error) {
	var args = m.Called(ctx, uid, partNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

// Undelete removes the \Deleted flag from an email
func (m *IMAPPort) Undelete(ctx context.Context, uid uint32) error {
	args := m.Called(ctx, uid)
	return args.Error(0)
}

// Ensure IMAPPort implements ports.IMAPPort
var _ ports.IMAPPort = (*IMAPPort)(nil)
