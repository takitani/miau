package testutil

import (
	"time"

	"github.com/opik/miau/internal/ports"
)

// TestAccount returns a sample account for testing
func TestAccount() *ports.AccountInfo {
	return &ports.AccountInfo{
		ID:        1,
		Email:     "test@example.com",
		Name:      "Test User",
		CreatedAt: time.Now(),
	}
}

// TestAccount2 returns another sample account for testing
func TestAccount2() *ports.AccountInfo {
	return &ports.AccountInfo{
		ID:        2,
		Email:     "other@example.com",
		Name:      "Other User",
		CreatedAt: time.Now(),
	}
}

// TestFolder returns a sample folder for testing
func TestFolder() *ports.Folder {
	return &ports.Folder{
		ID:             1,
		Name:           "INBOX",
		TotalMessages:  100,
		UnreadMessages: 10,
	}
}

// TestFolderSent returns a sent folder for testing
func TestFolderSent() *ports.Folder {
	return &ports.Folder{
		ID:             2,
		Name:           "[Gmail]/Sent Mail",
		TotalMessages:  50,
		UnreadMessages: 0,
	}
}

// TestFolders returns a list of sample folders
func TestFolders() []ports.Folder {
	return []ports.Folder{
		*TestFolder(),
		*TestFolderSent(),
		{ID: 3, Name: "[Gmail]/Trash", TotalMessages: 5, UnreadMessages: 0},
		{ID: 4, Name: "[Gmail]/All Mail", TotalMessages: 200, UnreadMessages: 10},
	}
}

// TestEmailMetadata returns a sample email metadata for testing
func TestEmailMetadata() ports.EmailMetadata {
	return ports.EmailMetadata{
		ID:        1,
		UID:       1001,
		MessageID: "<msg001@example.com>",
		Subject:   "Test Email Subject",
		FromName:  "Sender Name",
		FromEmail: "sender@example.com",
		ToAddress: "test@example.com",
		Date:      time.Now().Add(-1 * time.Hour),
		IsRead:    false,
		IsStarred: false,
		IsReplied: false,
		Snippet:   "This is a test email preview...",
		Size:      1024,
	}
}

// TestEmailMetadataRead returns a read email for testing
func TestEmailMetadataRead() ports.EmailMetadata {
	var email = TestEmailMetadata()
	email.ID = 2
	email.UID = 1002
	email.MessageID = "<msg002@example.com>"
	email.Subject = "Already Read Email"
	email.IsRead = true
	return email
}

// TestEmailMetadataStarred returns a starred email for testing
func TestEmailMetadataStarred() ports.EmailMetadata {
	var email = TestEmailMetadata()
	email.ID = 3
	email.UID = 1003
	email.MessageID = "<msg003@example.com>"
	email.Subject = "Important Starred Email"
	email.IsStarred = true
	return email
}

// TestEmailList returns a list of sample emails
func TestEmailList() []ports.EmailMetadata {
	return []ports.EmailMetadata{
		TestEmailMetadata(),
		TestEmailMetadataRead(),
		TestEmailMetadataStarred(),
	}
}

// TestEmailContent returns a sample full email for testing
func TestEmailContent() *ports.EmailContent {
	return &ports.EmailContent{
		EmailMetadata: TestEmailMetadata(),
		FolderID:      1,
		FolderName:    "INBOX",
		ToAddresses:   "test@example.com",
		CcAddresses:   "cc@example.com",
		BodyText:      "This is the plain text body of the email.",
		BodyHTML:      "<html><body><p>This is the <b>HTML</b> body of the email.</p></body></html>",
		RawHeaders:    "From: sender@example.com\r\nTo: test@example.com\r\n",
		HasAttachments: false,
		Attachments:   nil,
	}
}

// TestEmailContentWithAttachment returns an email with attachments
func TestEmailContentWithAttachment() *ports.EmailContent {
	var email = TestEmailContent()
	email.ID = 4
	email.UID = 1004
	email.HasAttachments = true
	email.Attachments = []ports.Attachment{
		{
			Filename:    "document.pdf",
			ContentType: "application/pdf",
			Size:        2048,
			IsInline:    false,
		},
		{
			Filename:    "image.png",
			ContentType: "image/png",
			Size:        4096,
			ContentID:   "cid:image001",
			IsInline:    true,
		},
	}
	return email
}

// TestSendRequest returns a sample send request
func TestSendRequest() *ports.SendRequest {
	return &ports.SendRequest{
		To:       []string{"recipient@example.com"},
		Cc:       nil,
		Bcc:      nil,
		Subject:  "Test Subject",
		BodyText: "Hello, this is a test email.",
		BodyHTML: "<p>Hello, this is a test email.</p>",
	}
}

// TestSendRequestWithCc returns a send request with CC
func TestSendRequestWithCc() *ports.SendRequest {
	var req = TestSendRequest()
	req.Cc = []string{"cc1@example.com", "cc2@example.com"}
	return req
}

// TestSendRequestReply returns a send request for a reply
func TestSendRequestReply() *ports.SendRequest {
	var originalID int64 = 1
	return &ports.SendRequest{
		To:             []string{"sender@example.com"},
		Subject:        "Re: Test Email Subject",
		BodyText:       "Thanks for your email.",
		BodyHTML:       "<p>Thanks for your email.</p>",
		InReplyTo:      "<msg001@example.com>",
		ReferenceIDs:   "<msg001@example.com>",
		ReplyToEmailID: &originalID,
	}
}

// TestSendResult returns a successful send result
func TestSendResult() *ports.SendResult {
	return &ports.SendResult{
		Success:   true,
		MessageID: "<sent001@example.com>",
		Error:     nil,
		SentAt:    time.Now(),
	}
}

// TestSendResultFailed returns a failed send result
func TestSendResultFailed(err error) *ports.SendResult {
	return &ports.SendResult{
		Success: false,
		Error:   err,
		SentAt:  time.Now(),
	}
}

// TestDraft returns a sample draft
func TestDraft() *ports.Draft {
	return &ports.Draft{
		ID:          1,
		ToAddresses: "recipient@example.com",
		Subject:     "Draft Subject",
		BodyText:    "Draft body text",
		BodyHTML:    "<p>Draft body</p>",
		Status:      ports.DraftStatusDraft,
		Source:      "manual",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// TestDraftScheduled returns a scheduled draft
func TestDraftScheduled() *ports.Draft {
	var draft = TestDraft()
	draft.ID = 2
	draft.Status = ports.DraftStatusScheduled
	var scheduledTime = time.Now().Add(1 * time.Hour)
	draft.ScheduledSendAt = &scheduledTime
	return draft
}

// TestDraftAI returns an AI-generated draft
func TestDraftAI() *ports.Draft {
	var draft = TestDraft()
	draft.ID = 3
	draft.Source = "ai"
	draft.AIPrompt = "Reply saying thank you"
	return draft
}

// TestMailboxInfo returns sample mailbox info
func TestMailboxInfo() ports.MailboxInfo {
	return ports.MailboxInfo{
		Name:     "INBOX",
		Messages: 100,
		Unseen:   10,
	}
}

// TestMailboxStatus returns sample mailbox status
func TestMailboxStatus() *ports.MailboxStatus {
	return &ports.MailboxStatus{
		Name:        "INBOX",
		NumMessages: 100,
		NumUnseen:   10,
		UIDNext:     1001,
		UIDValidity: 12345,
	}
}

// TestIMAPEmail returns a sample IMAP email
func TestIMAPEmail() ports.IMAPEmail {
	return ports.IMAPEmail{
		UID:       1001,
		MessageID: "<msg001@example.com>",
		Subject:   "Test Email Subject",
		FromName:  "Sender Name",
		FromEmail: "sender@example.com",
		To:        "test@example.com",
		Date:      time.Now().Add(-1 * time.Hour),
		Seen:      false,
		Flagged:   false,
		Size:      1024,
		BodyText:  "This is the plain text body.",
	}
}

// TestIMAPEmails returns a list of IMAP emails
func TestIMAPEmails() []ports.IMAPEmail {
	var email1 = TestIMAPEmail()
	var email2 = TestIMAPEmail()
	email2.UID = 1002
	email2.MessageID = "<msg002@example.com>"
	email2.Subject = "Second Test Email"
	email2.Seen = true

	return []ports.IMAPEmail{email1, email2}
}

// TestSyncResult returns a sample sync result
func TestSyncResult() *ports.SyncResult {
	return &ports.SyncResult{
		NewEmails:     5,
		DeletedEmails: 2,
		LatestUID:     1010,
		Errors:        nil,
	}
}

// TestBatchOperation returns a sample batch operation
func TestBatchOperation() *ports.BatchOperation {
	return &ports.BatchOperation{
		ID:          1,
		Operation:   ports.BatchOpArchive,
		Description: "Archive 10 emails from newsletter",
		FilterQuery: "from_email LIKE '%newsletter%'",
		EmailIDs:    []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		EmailCount:  10,
		Status:      ports.BatchOpStatusPending,
		CreatedAt:   time.Now(),
	}
}
