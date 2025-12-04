package services

import (
	"context"
	"errors"
	"testing"

	"github.com/opik/miau/internal/testutil"
	"github.com/opik/miau/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEmailService_GetFolders_Success(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)
	svc.SetAccount(testutil.TestAccount())

	var folders = testutil.TestFolders()
	mockStorage.On("GetFolders", mock.Anything, int64(1)).Return(folders, nil)

	// Act
	var result, err = svc.GetFolders(context.Background())

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 4)
	assert.Equal(t, "INBOX", result[0].Name)

	mockStorage.AssertExpectations(t)
}

func TestEmailService_GetFolders_NoAccount(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)
	// Don't set account

	// Act
	var result, err = svc.GetFolders(context.Background())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no account set")

	mockStorage.AssertNotCalled(t, "GetFolders", mock.Anything, mock.Anything)
}

func TestEmailService_SelectFolder_Success(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)
	svc.SetAccount(testutil.TestAccount())

	var folder = testutil.TestFolder()
	var mailboxStatus = testutil.TestMailboxStatus()

	mockStorage.On("GetFolderByName", mock.Anything, int64(1), "INBOX").Return(folder, nil)
	mockIMAP.On("SelectMailbox", mock.Anything, "INBOX").Return(mailboxStatus, nil)

	// Act
	var result, err = svc.SelectFolder(context.Background(), "INBOX")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "INBOX", result.Name)

	mockStorage.AssertExpectations(t)
	mockIMAP.AssertExpectations(t)
}

func TestEmailService_SelectFolder_FolderNotFound(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)
	svc.SetAccount(testutil.TestAccount())

	mockStorage.On("GetFolderByName", mock.Anything, int64(1), "NonExistent").Return(nil, errors.New("folder not found"))

	// Act
	var result, err = svc.SelectFolder(context.Background(), "NonExistent")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)

	mockIMAP.AssertNotCalled(t, "SelectMailbox", mock.Anything, mock.Anything)
}

func TestEmailService_SelectFolder_IMAPError(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)
	svc.SetAccount(testutil.TestAccount())

	var folder = testutil.TestFolder()

	mockStorage.On("GetFolderByName", mock.Anything, int64(1), "INBOX").Return(folder, nil)
	mockIMAP.On("SelectMailbox", mock.Anything, "INBOX").Return(nil, errors.New("IMAP connection failed"))

	// Act
	var result, err = svc.SelectFolder(context.Background(), "INBOX")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to select mailbox")
}

func TestEmailService_GetEmails_Success(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)
	svc.SetAccount(testutil.TestAccount())

	var folder = testutil.TestFolder()
	var emails = testutil.TestEmailList()

	mockStorage.On("GetFolderByName", mock.Anything, int64(1), "INBOX").Return(folder, nil)
	mockStorage.On("GetEmails", mock.Anything, folder.ID, 50).Return(emails, nil)

	// Act
	var result, err = svc.GetEmails(context.Background(), "INBOX", 50)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 3)

	mockStorage.AssertExpectations(t)
}

func TestEmailService_GetEmails_NoAccount(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)
	// Don't set account

	// Act
	var result, err = svc.GetEmails(context.Background(), "INBOX", 50)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no account set")
}

func TestEmailService_GetEmail_FromStorage(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)

	var email = testutil.TestEmailContent()
	email.BodyText = "Already has body"
	email.BodyHTML = "<p>Already has HTML</p>"

	mockStorage.On("GetEmail", mock.Anything, int64(1)).Return(email, nil)

	// Act
	var result, err = svc.GetEmail(context.Background(), 1)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Already has body", result.BodyText)
	assert.Equal(t, "<p>Already has HTML</p>", result.BodyHTML)

	// Should NOT fetch from IMAP since body is already there
	mockIMAP.AssertNotCalled(t, "FetchEmailRaw", mock.Anything, mock.Anything)
}

func TestEmailService_GetEmail_FetchFromIMAP(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)

	var email = testutil.TestEmailContent()
	email.BodyText = "" // Empty body - should trigger IMAP fetch
	email.BodyHTML = ""
	email.FolderName = "INBOX"

	// Simple raw email data for parsing
	var rawEmail = []byte("From: sender@example.com\r\nTo: test@example.com\r\nSubject: Test\r\n\r\nPlain text body")

	mockStorage.On("GetEmail", mock.Anything, int64(1)).Return(email, nil)
	mockIMAP.On("SelectMailbox", mock.Anything, "INBOX").Return(testutil.TestMailboxStatus(), nil)
	mockIMAP.On("FetchEmailRaw", mock.Anything, email.UID).Return(rawEmail, nil)

	// Act
	var result, err = svc.GetEmail(context.Background(), 1)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)

	mockIMAP.AssertCalled(t, "SelectMailbox", mock.Anything, "INBOX")
	mockIMAP.AssertCalled(t, "FetchEmailRaw", mock.Anything, email.UID)
}

func TestEmailService_GetEmail_NotFound(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)

	mockStorage.On("GetEmail", mock.Anything, int64(999)).Return(nil, errors.New("email not found"))

	// Act
	var result, err = svc.GetEmail(context.Background(), 999)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestEmailService_MarkAsRead_Success(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)

	var email = testutil.TestEmailContent()

	mockStorage.On("GetEmail", mock.Anything, int64(1)).Return(email, nil)
	mockIMAP.On("MarkAsRead", mock.Anything, email.UID).Return(nil)
	mockStorage.On("MarkAsRead", mock.Anything, int64(1), true).Return(nil)
	mockEvents.On("Publish", mock.Anything).Return()

	// Act
	var err = svc.MarkAsRead(context.Background(), 1, true)

	// Assert
	assert.NoError(t, err)

	mockIMAP.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockEvents.AssertExpectations(t)
}

func TestEmailService_MarkAsRead_Unread(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)

	var email = testutil.TestEmailContent()

	mockStorage.On("GetEmail", mock.Anything, int64(1)).Return(email, nil)
	mockIMAP.On("MarkAsUnread", mock.Anything, email.UID).Return(nil)
	mockStorage.On("MarkAsRead", mock.Anything, int64(1), false).Return(nil)
	mockEvents.On("Publish", mock.Anything).Return()

	// Act
	var err = svc.MarkAsRead(context.Background(), 1, false)

	// Assert
	assert.NoError(t, err)

	mockIMAP.AssertCalled(t, "MarkAsUnread", mock.Anything, email.UID)
}

func TestEmailService_MarkAsStarred_Success(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)

	mockStorage.On("MarkAsStarred", mock.Anything, int64(1), true).Return(nil)

	// Act
	var err = svc.MarkAsStarred(context.Background(), 1, true)

	// Assert
	assert.NoError(t, err)

	mockStorage.AssertExpectations(t)
}

func TestEmailService_Archive_Success(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)

	var email = testutil.TestEmailContent()

	mockStorage.On("GetEmail", mock.Anything, int64(1)).Return(email, nil)
	mockIMAP.On("Archive", mock.Anything, email.UID).Return(nil)
	mockStorage.On("MarkAsArchived", mock.Anything, int64(1), true).Return(nil)

	// Act
	var err = svc.Archive(context.Background(), 1)

	// Assert
	assert.NoError(t, err)

	mockIMAP.AssertCalled(t, "Archive", mock.Anything, email.UID)
	mockStorage.AssertCalled(t, "MarkAsArchived", mock.Anything, int64(1), true)
}

func TestEmailService_Archive_IMAPError(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)

	var email = testutil.TestEmailContent()

	mockStorage.On("GetEmail", mock.Anything, int64(1)).Return(email, nil)
	mockIMAP.On("Archive", mock.Anything, email.UID).Return(errors.New("IMAP archive failed"))

	// Act
	var err = svc.Archive(context.Background(), 1)

	// Assert
	assert.Error(t, err)

	// Should NOT update storage on IMAP error
	mockStorage.AssertNotCalled(t, "MarkAsArchived", mock.Anything, mock.Anything, mock.Anything)
}

func TestEmailService_Delete_Success(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)

	var email = testutil.TestEmailContent()

	mockStorage.On("GetEmail", mock.Anything, int64(1)).Return(email, nil)
	mockIMAP.On("Delete", mock.Anything, email.UID).Return(nil)
	mockStorage.On("MarkAsDeleted", mock.Anything, int64(1), true).Return(nil)

	// Act
	var err = svc.Delete(context.Background(), 1)

	// Assert
	assert.NoError(t, err)

	mockIMAP.AssertCalled(t, "Delete", mock.Anything, email.UID)
	mockStorage.AssertCalled(t, "MarkAsDeleted", mock.Anything, int64(1), true)
}

func TestEmailService_Delete_IMAPError(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)

	var email = testutil.TestEmailContent()

	mockStorage.On("GetEmail", mock.Anything, int64(1)).Return(email, nil)
	mockIMAP.On("Delete", mock.Anything, email.UID).Return(errors.New("IMAP delete failed"))

	// Act
	var err = svc.Delete(context.Background(), 1)

	// Assert
	assert.Error(t, err)

	// Should NOT update storage on IMAP error
	mockStorage.AssertNotCalled(t, "MarkAsDeleted", mock.Anything, mock.Anything, mock.Anything)
}

func TestEmailService_MoveToFolder_Success(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)

	var email = testutil.TestEmailContent()

	mockStorage.On("GetEmail", mock.Anything, int64(1)).Return(email, nil)
	mockIMAP.On("MoveToFolder", mock.Anything, email.UID, "[Gmail]/Trash").Return(nil)

	// Act
	var err = svc.MoveToFolder(context.Background(), 1, "[Gmail]/Trash")

	// Assert
	assert.NoError(t, err)

	mockIMAP.AssertCalled(t, "MoveToFolder", mock.Anything, email.UID, "[Gmail]/Trash")
}

func TestEmailService_GetLatestUID_Success(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)
	svc.SetAccount(testutil.TestAccount())

	var folder = testutil.TestFolder()

	mockStorage.On("GetFolderByName", mock.Anything, int64(1), "INBOX").Return(folder, nil)
	mockStorage.On("GetLatestUID", mock.Anything, folder.ID).Return(uint32(1050), nil)

	// Act
	var uid, err = svc.GetLatestUID(context.Background(), "INBOX")

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, uint32(1050), uid)
}

func TestEmailService_GetLatestUID_NoAccount(t *testing.T) {
	// Arrange
	var mockIMAP = new(mocks.IMAPPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewEmailService(mockIMAP, mockStorage, mockEvents)
	// Don't set account

	// Act
	var uid, err = svc.GetLatestUID(context.Background(), "INBOX")

	// Assert
	assert.Error(t, err)
	assert.Equal(t, uint32(0), uid)
	assert.Contains(t, err.Error(), "no account set")
}
