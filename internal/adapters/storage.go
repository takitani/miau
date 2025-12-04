package adapters

import (
	"context"
	"database/sql"
	"time"

	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/storage"
)

// StorageAdapter implements ports.StoragePort using the existing storage package
type StorageAdapter struct{}

// NewStorageAdapter creates a new StorageAdapter
func NewStorageAdapter() *StorageAdapter {
	return &StorageAdapter{}
}

// GetOrCreateAccount gets or creates an account
func (a *StorageAdapter) GetOrCreateAccount(ctx context.Context, email, name string) (*ports.AccountInfo, error) {
	var account, err = storage.GetOrCreateAccount(email, name)
	if err != nil {
		return nil, err
	}
	return &ports.AccountInfo{
		ID:        account.ID,
		Email:     account.Email,
		Name:      account.Name,
		CreatedAt: account.CreatedAt.Time,
	}, nil
}

// GetAccount gets an account by ID
func (a *StorageAdapter) GetAccount(ctx context.Context, id int64) (*ports.AccountInfo, error) {
	// The current storage package doesn't have GetAccountByID
	// For now, return not found
	return nil, ErrNotFound
}

// UpsertFolder creates or updates a folder
func (a *StorageAdapter) UpsertFolder(ctx context.Context, accountID int64, folder *ports.Folder) error {
	var _, err = storage.GetOrCreateFolder(accountID, folder.Name)
	if err != nil {
		return err
	}
	return nil
}

// GetFolders returns all folders for an account
func (a *StorageAdapter) GetFolders(ctx context.Context, accountID int64) ([]ports.Folder, error) {
	var folders, err = storage.GetFolders(accountID)
	if err != nil {
		return nil, err
	}

	var result = make([]ports.Folder, len(folders))
	for i, f := range folders {
		var lastSync *time.Time
		if f.LastSync.Valid {
			lastSync = &f.LastSync.Time
		}
		result[i] = ports.Folder{
			ID:             f.ID,
			Name:           f.Name,
			TotalMessages:  f.TotalMessages,
			UnreadMessages: f.UnreadMessages,
			LastSync:       lastSync,
		}
	}
	return result, nil
}

// GetFolderByName returns a folder by name
func (a *StorageAdapter) GetFolderByName(ctx context.Context, accountID int64, name string) (*ports.Folder, error) {
	var folder, err = storage.GetOrCreateFolder(accountID, name)
	if err != nil {
		return nil, err
	}

	var lastSync *time.Time
	if folder.LastSync.Valid {
		lastSync = &folder.LastSync.Time
	}

	return &ports.Folder{
		ID:             folder.ID,
		Name:           folder.Name,
		TotalMessages:  folder.TotalMessages,
		UnreadMessages: folder.UnreadMessages,
		LastSync:       lastSync,
	}, nil
}

// UpdateFolderStats updates folder statistics
func (a *StorageAdapter) UpdateFolderStats(ctx context.Context, folderID int64, total, unread int) error {
	return storage.UpdateFolderStats(folderID, total, unread)
}

// UpsertEmail creates or updates an email
func (a *StorageAdapter) UpsertEmail(ctx context.Context, accountID, folderID int64, email *ports.EmailContent) error {
	var e = &storage.Email{
		AccountID:      accountID,
		FolderID:       folderID,
		UID:            email.UID,
		MessageID:      sql.NullString{String: email.MessageID, Valid: email.MessageID != ""},
		Subject:        email.Subject,
		FromName:       email.FromName,
		FromEmail:      email.FromEmail,
		ToAddresses:    email.ToAddresses,
		CcAddresses:    email.CcAddresses,
		Date:           storage.SQLiteTime{Time: email.Date},
		IsRead:         email.IsRead,
		IsStarred:      email.IsStarred,
		HasAttachments: email.HasAttachments,
		Snippet:        email.Snippet,
		BodyText:       email.BodyText,
		BodyHTML:       email.BodyHTML,
		RawHeaders:     email.RawHeaders,
		Size:           email.Size,
	}
	return storage.UpsertEmail(e)
}

// GetEmails returns emails from a folder
func (a *StorageAdapter) GetEmails(ctx context.Context, folderID int64, limit int) ([]ports.EmailMetadata, error) {
	// We need accountID too, but the interface doesn't provide it
	// For now, get emails assuming folderID is sufficient
	var emails, err = storage.GetEmails(0, folderID, limit, 0)
	if err != nil {
		return nil, err
	}

	var result = make([]ports.EmailMetadata, len(emails))
	for i, e := range emails {
		result[i] = ports.EmailMetadata{
			ID:        e.ID,
			UID:       e.UID,
			MessageID: e.MessageID.String,
			Subject:   e.Subject,
			FromName:  e.FromName,
			FromEmail: e.FromEmail,
			Date:      e.Date.Time,
			IsRead:    e.IsRead,
			IsStarred: e.IsStarred,
			IsReplied: e.IsReplied,
			Snippet:   e.Snippet,
		}
	}
	return result, nil
}

// GetEmail returns a single email by ID (includes folder name for IMAP fetch)
func (a *StorageAdapter) GetEmail(ctx context.Context, id int64) (*ports.EmailContent, error) {
	var email, err = storage.GetEmailByIDWithFolder(id)
	if err != nil {
		return nil, err
	}

	var result = convertStorageEmail(&email.Email)
	result.FolderID = email.FolderID
	result.FolderName = email.FolderName
	return result, nil
}

// GetEmailByUID returns an email by UID
func (a *StorageAdapter) GetEmailByUID(ctx context.Context, folderID int64, uid uint32) (*ports.EmailContent, error) {
	var email, err = storage.GetEmailByUID(0, folderID, uid)
	if err != nil {
		return nil, err
	}

	return convertStorageEmail(email), nil
}

// GetLatestUID returns the latest UID for a folder
func (a *StorageAdapter) GetLatestUID(ctx context.Context, folderID int64) (uint32, error) {
	return storage.GetLatestUID(0, folderID)
}

// GetAllUIDs returns all UIDs for a folder
func (a *StorageAdapter) GetAllUIDs(ctx context.Context, folderID int64) ([]uint32, error) {
	// The current storage package doesn't have a direct GetAllUIDs
	// This would need to be implemented
	return nil, nil
}

// MarkAsRead marks an email as read
func (a *StorageAdapter) MarkAsRead(ctx context.Context, id int64, read bool) error {
	return storage.MarkAsRead(id, read)
}

// MarkAsStarred marks an email as starred
func (a *StorageAdapter) MarkAsStarred(ctx context.Context, id int64, starred bool) error {
	return storage.MarkAsStarred(id, starred)
}

// MarkAsArchived marks an email as archived
func (a *StorageAdapter) MarkAsArchived(ctx context.Context, id int64, archived bool) error {
	return storage.MarkAsArchived(id, archived)
}

// MarkAsDeleted marks an email as deleted
func (a *StorageAdapter) MarkAsDeleted(ctx context.Context, id int64, deleted bool) error {
	return storage.DeleteEmail(id)
}

// MarkAsReplied marks an email as replied
func (a *StorageAdapter) MarkAsReplied(ctx context.Context, id int64, replied bool) error {
	return storage.MarkAsReplied(id)
}

// MarkDeletedByUIDs marks emails as deleted by UIDs
func (a *StorageAdapter) MarkDeletedByUIDs(ctx context.Context, folderID int64, uids []uint32) error {
	// Use PurgeDeletedFromServer in reverse
	// This would need implementation
	return nil
}

// BulkMarkAsRead marks multiple emails as read
func (a *StorageAdapter) BulkMarkAsRead(ctx context.Context, ids []int64, read bool) error {
	for _, id := range ids {
		if err := storage.MarkAsRead(id, read); err != nil {
			return err
		}
	}
	return nil
}

// BulkMarkAsArchived marks multiple emails as archived
func (a *StorageAdapter) BulkMarkAsArchived(ctx context.Context, ids []int64) error {
	for _, id := range ids {
		if err := storage.MarkAsArchived(id, true); err != nil {
			return err
		}
	}
	return nil
}

// BulkMarkAsDeleted marks multiple emails as deleted
func (a *StorageAdapter) BulkMarkAsDeleted(ctx context.Context, ids []int64) error {
	for _, id := range ids {
		if err := storage.DeleteEmail(id); err != nil {
			return err
		}
	}
	return nil
}

// SearchEmails searches emails
func (a *StorageAdapter) SearchEmails(ctx context.Context, accountID int64, query string, limit int) ([]ports.EmailMetadata, error) {
	var emails, err = storage.FuzzySearchEmails(accountID, query, limit)
	if err != nil {
		return nil, err
	}

	var result = make([]ports.EmailMetadata, len(emails))
	for i, e := range emails {
		result[i] = ports.EmailMetadata{
			ID:        e.ID,
			UID:       e.UID,
			MessageID: e.MessageID.String,
			Subject:   e.Subject,
			FromName:  e.FromName,
			FromEmail: e.FromEmail,
			Date:      e.Date.Time,
			IsRead:    e.IsRead,
			IsStarred: e.IsStarred,
			IsReplied: e.IsReplied,
			Snippet:   e.Snippet,
		}
	}
	return result, nil
}

// SearchEmailsInFolder searches emails in a specific folder
func (a *StorageAdapter) SearchEmailsInFolder(ctx context.Context, folderID int64, query string, limit int) ([]ports.EmailMetadata, error) {
	// The current storage package doesn't have folder-specific search
	// For now, use the general search
	return a.SearchEmails(ctx, 0, query, limit)
}

// CreateDraft creates a new draft
func (a *StorageAdapter) CreateDraft(ctx context.Context, accountID int64, draft *ports.Draft) (*ports.Draft, error) {
	var d = &storage.Draft{
		AccountID:       accountID,
		ToAddresses:     draft.ToAddresses,
		CcAddresses:     sql.NullString{String: draft.CcAddresses, Valid: draft.CcAddresses != ""},
		BccAddresses:    sql.NullString{String: draft.BccAddresses, Valid: draft.BccAddresses != ""},
		Subject:         draft.Subject,
		BodyHTML:        sql.NullString{String: draft.BodyHTML, Valid: draft.BodyHTML != ""},
		BodyText:        sql.NullString{String: draft.BodyText, Valid: draft.BodyText != ""},
		Classification:  sql.NullString{String: draft.Classification, Valid: draft.Classification != ""},
		InReplyTo:       sql.NullString{String: draft.InReplyTo, Valid: draft.InReplyTo != ""},
		ReferenceIDs:    sql.NullString{String: draft.ReferenceIDs, Valid: draft.ReferenceIDs != ""},
		Status:          storage.DraftStatus(draft.Status),
		GenerationSource: draft.Source,
		AIPrompt:        sql.NullString{String: draft.AIPrompt, Valid: draft.AIPrompt != ""},
	}

	if draft.ReplyToEmailID != nil {
		d.ReplyToEmailID = sql.NullInt64{Int64: *draft.ReplyToEmailID, Valid: true}
	}

	var id, err = storage.CreateDraft(d)
	if err != nil {
		return nil, err
	}

	draft.ID = id
	return draft, nil
}

// UpdateDraft updates a draft
func (a *StorageAdapter) UpdateDraft(ctx context.Context, draft *ports.Draft) error {
	var d = &storage.Draft{
		ID:              draft.ID,
		ToAddresses:     draft.ToAddresses,
		CcAddresses:     sql.NullString{String: draft.CcAddresses, Valid: draft.CcAddresses != ""},
		BccAddresses:    sql.NullString{String: draft.BccAddresses, Valid: draft.BccAddresses != ""},
		Subject:         draft.Subject,
		BodyHTML:        sql.NullString{String: draft.BodyHTML, Valid: draft.BodyHTML != ""},
		BodyText:        sql.NullString{String: draft.BodyText, Valid: draft.BodyText != ""},
		Classification:  sql.NullString{String: draft.Classification, Valid: draft.Classification != ""},
		InReplyTo:       sql.NullString{String: draft.InReplyTo, Valid: draft.InReplyTo != ""},
		ReferenceIDs:    sql.NullString{String: draft.ReferenceIDs, Valid: draft.ReferenceIDs != ""},
		Status:          storage.DraftStatus(draft.Status),
	}

	return storage.UpdateDraft(d)
}

// GetDraft gets a draft by ID
func (a *StorageAdapter) GetDraft(ctx context.Context, id int64) (*ports.Draft, error) {
	var draft, err = storage.GetDraftByID(id)
	if err != nil {
		return nil, err
	}

	return convertStorageDraft(draft), nil
}

// GetDrafts gets all drafts for an account
func (a *StorageAdapter) GetDrafts(ctx context.Context, accountID int64) ([]ports.Draft, error) {
	var drafts, err = storage.GetPendingDrafts(accountID)
	if err != nil {
		return nil, err
	}

	var result = make([]ports.Draft, len(drafts))
	for i, d := range drafts {
		result[i] = *convertStorageDraft(&d)
	}
	return result, nil
}

// GetPendingDrafts gets pending drafts
func (a *StorageAdapter) GetPendingDrafts(ctx context.Context, accountID int64) ([]ports.Draft, error) {
	return a.GetDrafts(ctx, accountID)
}

// DeleteDraft deletes a draft
func (a *StorageAdapter) DeleteDraft(ctx context.Context, id int64) error {
	return storage.DeleteDraft(id)
}

// UpdateDraftStatus updates draft status
func (a *StorageAdapter) UpdateDraftStatus(ctx context.Context, id int64, status ports.DraftStatus) error {
	switch status {
	case ports.DraftStatusSending:
		return storage.MarkDraftSending(id)
	case ports.DraftStatusSent:
		return storage.MarkDraftSent(id)
	case ports.DraftStatusCancelled:
		return storage.CancelDraft(id)
	default:
		return nil
	}
}

// CreateBatchOp creates a batch operation
func (a *StorageAdapter) CreateBatchOp(ctx context.Context, accountID int64, op *ports.BatchOperation) (*ports.BatchOperation, error) {
	// The current storage package has batch operations - we'd need to implement this
	return op, nil
}

// GetPendingBatchOp gets pending batch operation
func (a *StorageAdapter) GetPendingBatchOp(ctx context.Context, accountID int64) (*ports.BatchOperation, error) {
	// Need to implement using storage.GetPendingBatchOp
	return nil, nil
}

// UpdateBatchOpStatus updates batch operation status
func (a *StorageAdapter) UpdateBatchOpStatus(ctx context.Context, id int64, status ports.BatchOpStatus) error {
	return nil
}

// ExecuteBatchOp executes a batch operation
func (a *StorageAdapter) ExecuteBatchOp(ctx context.Context, id int64) error {
	return nil
}

// TrackSentEmail tracks a sent email
func (a *StorageAdapter) TrackSentEmail(ctx context.Context, accountID int64, messageID, to, subject string) error {
	_, err := storage.RecordSentEmail(accountID, messageID, to, "", "", subject, "", "", "", "", "smtp", sql.NullInt64{}, sql.NullInt64{})
	return err
}

// GetRecentSentEmails gets recent sent emails
func (a *StorageAdapter) GetRecentSentEmails(ctx context.Context, accountID int64, since time.Duration) ([]ports.SentEmailTrack, error) {
	// Need to implement filter by time
	var emails, err = storage.GetSentEmails(accountID, 20, 0)
	if err != nil {
		return nil, err
	}

	var result []ports.SentEmailTrack
	for _, e := range emails {
		if time.Since(e.SentAt.Time) < since {
			result = append(result, ports.SentEmailTrack{
				MessageID: e.MessageID.String,
				To:        e.ToAddresses,
				Subject:   e.Subject,
				SentAt:    e.SentAt.Time,
			})
		}
	}
	return result, nil
}

// GetIndexState gets index state
func (a *StorageAdapter) GetIndexState(ctx context.Context, accountID int64) (*ports.IndexState, error) {
	// Need to implement
	return nil, nil
}

// UpdateIndexState updates index state
func (a *StorageAdapter) UpdateIndexState(ctx context.Context, accountID int64, state *ports.IndexState) error {
	return nil
}

// GetSetting gets a setting
func (a *StorageAdapter) GetSetting(ctx context.Context, accountID int64, key string) (string, error) {
	return "", nil
}

// SetSetting sets a setting
func (a *StorageAdapter) SetSetting(ctx context.Context, accountID int64, key, value string) error {
	return nil
}

// === ANALYTICS ===

// GetAnalyticsOverview returns overall email statistics
func (a *StorageAdapter) GetAnalyticsOverview(ctx context.Context, accountID int64) (*ports.AnalyticsOverview, error) {
	var overview, err = storage.GetAnalyticsOverview(accountID)
	if err != nil {
		return nil, err
	}
	return &ports.AnalyticsOverview{
		TotalEmails:    overview.TotalEmails,
		UnreadEmails:   overview.UnreadEmails,
		StarredEmails:  overview.StarredEmails,
		ArchivedEmails: overview.ArchivedEmails,
		SentEmails:     overview.SentEmails,
		DraftCount:     overview.DraftCount,
		StorageUsedMB:  overview.StorageUsedMB,
	}, nil
}

// GetTopSenders returns top email senders
func (a *StorageAdapter) GetTopSenders(ctx context.Context, accountID int64, limit int, sinceDays int) ([]ports.SenderStats, error) {
	var senders, err = storage.GetTopSenders(accountID, limit, sinceDays)
	if err != nil {
		return nil, err
	}

	var result = make([]ports.SenderStats, len(senders))
	for i, s := range senders {
		result[i] = ports.SenderStats{
			Email:       s.Email,
			Name:        s.Name,
			Count:       s.Count,
			UnreadCount: s.UnreadCount,
		}
	}
	return result, nil
}

// GetEmailCountByHour returns email count by hour of day
func (a *StorageAdapter) GetEmailCountByHour(ctx context.Context, accountID int64, sinceDays int) ([]ports.HourlyStats, error) {
	var stats, err = storage.GetEmailCountByHour(accountID, sinceDays)
	if err != nil {
		return nil, err
	}

	var result = make([]ports.HourlyStats, len(stats))
	for i, s := range stats {
		result[i] = ports.HourlyStats{
			Hour:  s.Hour,
			Count: s.Count,
		}
	}
	return result, nil
}

// GetEmailCountByDay returns email count by day
func (a *StorageAdapter) GetEmailCountByDay(ctx context.Context, accountID int64, sinceDays int) ([]ports.DailyStats, error) {
	var stats, err = storage.GetEmailCountByDay(accountID, sinceDays)
	if err != nil {
		return nil, err
	}

	var result = make([]ports.DailyStats, len(stats))
	for i, s := range stats {
		result[i] = ports.DailyStats{
			Date:  s.Date,
			Count: s.Count,
		}
	}
	return result, nil
}

// GetEmailCountByWeekday returns email count by day of week
func (a *StorageAdapter) GetEmailCountByWeekday(ctx context.Context, accountID int64, sinceDays int) ([]ports.WeekdayStats, error) {
	var stats, err = storage.GetEmailCountByWeekday(accountID, sinceDays)
	if err != nil {
		return nil, err
	}

	var weekdayNames = []string{"Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"}
	var result = make([]ports.WeekdayStats, len(stats))
	for i, s := range stats {
		result[i] = ports.WeekdayStats{
			Weekday: s.Weekday,
			Name:    weekdayNames[s.Weekday],
			Count:   s.Count,
		}
	}
	return result, nil
}

// GetResponseStats returns response time statistics
func (a *StorageAdapter) GetResponseStats(ctx context.Context, accountID int64) (*ports.ResponseTimeStats, error) {
	var stats, err = storage.GetResponseStats(accountID)
	if err != nil {
		return nil, err
	}
	return &ports.ResponseTimeStats{
		AvgResponseMinutes: stats.AvgResponseMinutes,
		ResponseRate:       stats.ResponseRate,
	}, nil
}

// convertStorageEmail converts storage.Email to ports.EmailContent
func convertStorageEmail(e *storage.Email) *ports.EmailContent {
	return &ports.EmailContent{
		EmailMetadata: ports.EmailMetadata{
			ID:        e.ID,
			UID:       e.UID,
			MessageID: e.MessageID.String,
			Subject:   e.Subject,
			FromName:  e.FromName,
			FromEmail: e.FromEmail,
			Date:      e.Date.Time,
			IsRead:    e.IsRead,
			IsStarred: e.IsStarred,
			IsReplied: e.IsReplied,
			Snippet:   e.Snippet,
			Size:      e.Size,
		},
		ToAddresses:    e.ToAddresses,
		CcAddresses:    e.CcAddresses,
		BodyText:       e.BodyText,
		BodyHTML:       e.BodyHTML,
		RawHeaders:     e.RawHeaders,
		HasAttachments: e.HasAttachments,
	}
}

// === ATTACHMENTS ===

// GetAttachmentsByEmail returns all attachments for an email
func (a *StorageAdapter) GetAttachmentsByEmail(ctx context.Context, emailID int64) ([]ports.Attachment, error) {
	var attachments, err = storage.GetAttachmentsByEmail(emailID)
	if err != nil {
		return nil, err
	}

	var result = make([]ports.Attachment, len(attachments))
	for i, att := range attachments {
		result[i] = ports.Attachment{
			ID:          att.ID,
			EmailID:     att.EmailID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			ContentID:   att.ContentID.String,
			IsInline:    att.IsInline,
			PartNumber:  att.PartNumber.String,
			Encoding:    att.Encoding.String,
			IsCached:    att.IsCached,
		}
	}
	return result, nil
}

// GetAttachment returns a single attachment by ID
func (a *StorageAdapter) GetAttachment(ctx context.Context, id int64) (*ports.Attachment, error) {
	var att, err = storage.GetAttachmentByID(id)
	if err != nil {
		return nil, err
	}

	return &ports.Attachment{
		ID:          att.ID,
		EmailID:     att.EmailID,
		Filename:    att.Filename,
		ContentType: att.ContentType,
		Size:        att.Size,
		ContentID:   att.ContentID.String,
		IsInline:    att.IsInline,
		PartNumber:  att.PartNumber.String,
		Encoding:    att.Encoding.String,
		IsCached:    att.IsCached,
	}, nil
}

// GetAttachmentContent returns cached attachment content
func (a *StorageAdapter) GetAttachmentContent(ctx context.Context, id int64) ([]byte, error) {
	var data, _, err = storage.GetCachedAttachmentContent(id)
	return data, err
}

// CacheAttachmentContent stores attachment content in cache
func (a *StorageAdapter) CacheAttachmentContent(ctx context.Context, id int64, content []byte) error {
	return storage.CacheAttachmentContent(id, content, false)
}

// UpsertAttachment creates or updates an attachment
func (a *StorageAdapter) UpsertAttachment(ctx context.Context, attachment *ports.Attachment) (int64, error) {
	var att = &storage.Attachment{
		EmailID:     attachment.EmailID,
		AccountID:   0, // Will be filled from the email
		Filename:    attachment.Filename,
		ContentType: attachment.ContentType,
		Size:        attachment.Size,
		IsInline:    attachment.IsInline,
	}

	if attachment.ContentID != "" {
		att.ContentID = sql.NullString{String: attachment.ContentID, Valid: true}
	}
	if attachment.PartNumber != "" {
		att.PartNumber = sql.NullString{String: attachment.PartNumber, Valid: true}
	}
	if attachment.Encoding != "" {
		att.Encoding = sql.NullString{String: attachment.Encoding, Valid: true}
	}
	if attachment.IsInline {
		att.ContentDisposition = sql.NullString{String: "inline", Valid: true}
	} else {
		att.ContentDisposition = sql.NullString{String: "attachment", Valid: true}
	}

	return storage.UpsertAttachment(att)
}

// convertStorageDraft converts storage.Draft to ports.Draft
func convertStorageDraft(d *storage.Draft) *ports.Draft {
	var draft = &ports.Draft{
		ID:             d.ID,
		ToAddresses:    d.ToAddresses,
		CcAddresses:    d.CcAddresses.String,
		BccAddresses:   d.BccAddresses.String,
		Subject:        d.Subject,
		BodyHTML:       d.BodyHTML.String,
		BodyText:       d.BodyText.String,
		Classification: d.Classification.String,
		InReplyTo:      d.InReplyTo.String,
		ReferenceIDs:   d.ReferenceIDs.String,
		Status:         ports.DraftStatus(d.Status),
		Source:         d.GenerationSource,
		AIPrompt:       d.AIPrompt.String,
		ErrorMessage:   d.ErrorMessage.String,
		CreatedAt:      d.CreatedAt.Time,
		UpdatedAt:      d.UpdatedAt.Time,
	}

	if d.ReplyToEmailID.Valid {
		var id = d.ReplyToEmailID.Int64
		draft.ReplyToEmailID = &id
	}

	if d.ScheduledSendAt.Valid {
		draft.ScheduledSendAt = &d.ScheduledSendAt.Time
	}

	if d.SentAt.Valid {
		draft.SentAt = &d.SentAt.Time
	}

	return draft
}
