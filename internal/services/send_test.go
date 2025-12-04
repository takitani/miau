package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/testutil"
	"github.com/opik/miau/internal/testutil/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSendService_Send_Success_SMTP(t *testing.T) {
	// Arrange
	var mockSMTP = new(mocks.SMTPPort)
	var mockGmail = new(mocks.GmailAPIPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewSendService(mockSMTP, mockGmail, mockStorage, mockEvents)
	svc.SetAccount(testutil.TestAccount())
	svc.SetSendMethod(ports.SendMethodSMTP)

	var req = testutil.TestSendRequest()
	var result = testutil.TestSendResult()

	mockEvents.On("Publish", mock.Anything).Return()
	mockSMTP.On("Send", mock.Anything, req).Return(result, nil)
	mockStorage.On("TrackSentEmail", mock.Anything, int64(1), result.MessageID, req.To[0], req.Subject).Return(nil)

	// Act
	var gotResult, err = svc.Send(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, gotResult)
	assert.True(t, gotResult.Success)
	assert.Equal(t, result.MessageID, gotResult.MessageID)

	mockSMTP.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
	mockEvents.AssertExpectations(t)
}

func TestSendService_Send_Success_GmailAPI(t *testing.T) {
	// Arrange
	var mockSMTP = new(mocks.SMTPPort)
	var mockGmail = new(mocks.GmailAPIPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewSendService(mockSMTP, mockGmail, mockStorage, mockEvents)
	svc.SetAccount(testutil.TestAccount())
	svc.SetSendMethod(ports.SendMethodGmailAPI)

	var req = testutil.TestSendRequest()
	var result = testutil.TestSendResult()

	mockEvents.On("Publish", mock.Anything).Return()
	mockGmail.On("Send", mock.Anything, req).Return(result, nil)
	mockStorage.On("TrackSentEmail", mock.Anything, int64(1), result.MessageID, req.To[0], req.Subject).Return(nil)

	// Act
	var gotResult, err = svc.Send(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, gotResult)
	assert.True(t, gotResult.Success)

	mockGmail.AssertExpectations(t)
	mockSMTP.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
}

func TestSendService_Send_NoAccountSet(t *testing.T) {
	// Arrange
	var mockSMTP = new(mocks.SMTPPort)
	var mockGmail = new(mocks.GmailAPIPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewSendService(mockSMTP, mockGmail, mockStorage, mockEvents)
	// Don't set account

	var req = testutil.TestSendRequest()

	// Act
	var result, err = svc.Send(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "no account set")

	mockSMTP.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
	mockGmail.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
}

func TestSendService_Send_SMTPError(t *testing.T) {
	// Arrange
	var mockSMTP = new(mocks.SMTPPort)
	var mockGmail = new(mocks.GmailAPIPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewSendService(mockSMTP, mockGmail, mockStorage, mockEvents)
	svc.SetAccount(testutil.TestAccount())
	svc.SetSendMethod(ports.SendMethodSMTP)

	var req = testutil.TestSendRequest()
	var smtpErr = errors.New("SMTP connection failed")

	mockEvents.On("Publish", mock.Anything).Return()
	mockSMTP.On("Send", mock.Anything, req).Return(nil, smtpErr)

	// Act
	var result, err = svc.Send(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, smtpErr, err)

	// TrackSentEmail should NOT be called on error
	mockStorage.AssertNotCalled(t, "TrackSentEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSendService_Send_GmailAPIError(t *testing.T) {
	// Arrange
	var mockSMTP = new(mocks.SMTPPort)
	var mockGmail = new(mocks.GmailAPIPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewSendService(mockSMTP, mockGmail, mockStorage, mockEvents)
	svc.SetAccount(testutil.TestAccount())
	svc.SetSendMethod(ports.SendMethodGmailAPI)

	var req = testutil.TestSendRequest()
	var apiErr = errors.New("Gmail API rate limit exceeded")

	mockEvents.On("Publish", mock.Anything).Return()
	mockGmail.On("Send", mock.Anything, req).Return(nil, apiErr)

	// Act
	var result, err = svc.Send(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, apiErr, err)
}

func TestSendService_SendDraft_Success(t *testing.T) {
	// Arrange
	var mockSMTP = new(mocks.SMTPPort)
	var mockGmail = new(mocks.GmailAPIPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewSendService(mockSMTP, mockGmail, mockStorage, mockEvents)
	svc.SetAccount(testutil.TestAccount())
	svc.SetSendMethod(ports.SendMethodSMTP)

	var draft = testutil.TestDraft()
	var sendResult = testutil.TestSendResult()

	mockStorage.On("GetDraft", mock.Anything, draft.ID).Return(draft, nil)
	mockStorage.On("UpdateDraftStatus", mock.Anything, draft.ID, ports.DraftStatusSending).Return(nil)
	mockEvents.On("Publish", mock.Anything).Return()
	mockSMTP.On("Send", mock.Anything, mock.Anything).Return(sendResult, nil)
	mockStorage.On("TrackSentEmail", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockStorage.On("UpdateDraftStatus", mock.Anything, draft.ID, ports.DraftStatusSent).Return(nil)

	// Act
	var result, err = svc.SendDraft(context.Background(), draft.ID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	mockStorage.AssertExpectations(t)
}

func TestSendService_SendDraft_NotFound(t *testing.T) {
	// Arrange
	var mockSMTP = new(mocks.SMTPPort)
	var mockGmail = new(mocks.GmailAPIPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewSendService(mockSMTP, mockGmail, mockStorage, mockEvents)
	svc.SetAccount(testutil.TestAccount())

	var draftErr = errors.New("draft not found")
	mockStorage.On("GetDraft", mock.Anything, int64(999)).Return(nil, draftErr)

	// Act
	var result, err = svc.SendDraft(context.Background(), 999)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "draft not found")

	mockSMTP.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
}

func TestSendService_SendDraft_SendFailed(t *testing.T) {
	// Arrange
	var mockSMTP = new(mocks.SMTPPort)
	var mockGmail = new(mocks.GmailAPIPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewSendService(mockSMTP, mockGmail, mockStorage, mockEvents)
	svc.SetAccount(testutil.TestAccount())
	svc.SetSendMethod(ports.SendMethodSMTP)

	var draft = testutil.TestDraft()
	var sendErr = errors.New("send failed")

	mockStorage.On("GetDraft", mock.Anything, draft.ID).Return(draft, nil)
	mockStorage.On("UpdateDraftStatus", mock.Anything, draft.ID, ports.DraftStatusSending).Return(nil)
	mockEvents.On("Publish", mock.Anything).Return()
	mockSMTP.On("Send", mock.Anything, mock.Anything).Return(nil, sendErr)
	mockStorage.On("UpdateDraftStatus", mock.Anything, draft.ID, ports.DraftStatusFailed).Return(nil)

	// Act
	var result, err = svc.SendDraft(context.Background(), draft.ID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)

	// Should mark as failed
	mockStorage.AssertCalled(t, "UpdateDraftStatus", mock.Anything, draft.ID, ports.DraftStatusFailed)
}

func TestSendService_GetSignature_Cached(t *testing.T) {
	// Arrange
	var mockSMTP = new(mocks.SMTPPort)
	var mockGmail = new(mocks.GmailAPIPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewSendService(mockSMTP, mockGmail, mockStorage, mockEvents)
	svc.SetSendMethod(ports.SendMethodGmailAPI)

	// Pre-cache signature
	svc.signatureCache = "-- \nMy Signature"
	svc.signatureCached = true

	// Act
	var sig, err = svc.GetSignature(context.Background())

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "-- \nMy Signature", sig)

	// Should NOT call Gmail API since it's cached
	mockGmail.AssertNotCalled(t, "GetSignature", mock.Anything)
}

func TestSendService_LoadSignature_GmailAPI(t *testing.T) {
	// Arrange
	var mockSMTP = new(mocks.SMTPPort)
	var mockGmail = new(mocks.GmailAPIPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewSendService(mockSMTP, mockGmail, mockStorage, mockEvents)
	svc.SetSendMethod(ports.SendMethodGmailAPI)

	var expectedSig = "-- \nTest Signature from Gmail"
	mockGmail.On("GetSignature", mock.Anything).Return(expectedSig, nil)

	// Act
	var err = svc.LoadSignature(context.Background())

	// Assert
	assert.NoError(t, err)
	assert.True(t, svc.signatureCached)
	assert.Equal(t, expectedSig, svc.signatureCache)

	mockGmail.AssertExpectations(t)
}

func TestSendService_LoadSignature_SMTP(t *testing.T) {
	// Arrange
	var mockSMTP = new(mocks.SMTPPort)
	var mockGmail = new(mocks.GmailAPIPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewSendService(mockSMTP, mockGmail, mockStorage, mockEvents)
	svc.SetSendMethod(ports.SendMethodSMTP)

	// Act
	var err = svc.LoadSignature(context.Background())

	// Assert
	assert.NoError(t, err)
	assert.True(t, svc.signatureCached)
	assert.Equal(t, "", svc.signatureCache) // SMTP doesn't support signatures

	// Should NOT call Gmail API
	mockGmail.AssertNotCalled(t, "GetSignature", mock.Anything)
}

func TestSendService_Send_TracksEmailForBounceDetection(t *testing.T) {
	// Arrange
	var mockSMTP = new(mocks.SMTPPort)
	var mockGmail = new(mocks.GmailAPIPort)
	var mockStorage = new(mocks.StoragePort)
	var mockEvents = new(mocks.EventBus)

	var svc = NewSendService(mockSMTP, mockGmail, mockStorage, mockEvents)
	svc.SetAccount(testutil.TestAccount())
	svc.SetSendMethod(ports.SendMethodSMTP)

	var req = testutil.TestSendRequest()
	var result = &ports.SendResult{
		Success:   true,
		MessageID: "<unique-msg-id@test.com>",
		SentAt:    time.Now(),
	}

	mockEvents.On("Publish", mock.Anything).Return()
	mockSMTP.On("Send", mock.Anything, req).Return(result, nil)
	mockStorage.On("TrackSentEmail", mock.Anything, int64(1), "<unique-msg-id@test.com>", "recipient@example.com", "Test Subject").Return(nil)

	// Act
	var _, err = svc.Send(context.Background(), req)

	// Assert
	assert.NoError(t, err)

	// Verify TrackSentEmail was called with correct parameters
	mockStorage.AssertCalled(t, "TrackSentEmail", mock.Anything, int64(1), "<unique-msg-id@test.com>", "recipient@example.com", "Test Subject")
}

func TestParseAddresses(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single address",
			input:    "test@example.com",
			expected: []string{"test@example.com"},
		},
		{
			name:     "multiple addresses",
			input:    "a@test.com, b@test.com, c@test.com",
			expected: []string{"a@test.com", "b@test.com", "c@test.com"},
		},
		{
			name:     "addresses with extra spaces",
			input:    "  a@test.com ,  b@test.com  ",
			expected: []string{"a@test.com", "b@test.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result = parseAddresses(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
