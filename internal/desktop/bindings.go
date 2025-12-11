package desktop

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/opik/miau/internal/app"
	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/services"
	"github.com/opik/miau/internal/storage"
)

// ============================================================================
// FOLDER OPERATIONS
// ============================================================================

// GetFolders returns all mail folders
func (a *App) GetFolders() (result []FolderDTO, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[GetFolders] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return nil, nil
	}

	var folders, ferr = a.application.Email().GetFolders(context.Background())
	if ferr != nil {
		return nil, ferr
	}

	for _, f := range folders {
		result = append(result, a.folderToDTO(&f))
	}
	return result, nil
}

// SelectFolder selects a folder as current
func (a *App) SelectFolder(name string) (result *FolderDTO, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[SelectFolder] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return nil, nil
	}

	var folder, ferr = a.application.Email().SelectFolder(context.Background(), name)
	if ferr != nil {
		return nil, ferr
	}

	a.mu.Lock()
	a.currentFolder = name
	a.mu.Unlock()

	var dto = a.folderToDTO(folder)
	return &dto, nil
}

// GetCurrentFolder returns the currently selected folder name
func (a *App) GetCurrentFolder() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentFolder
}

// ============================================================================
// EMAIL OPERATIONS
// ============================================================================

// GetEmails returns emails from a folder
func (a *App) GetEmails(folder string, limit int) (result []EmailDTO, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[GetEmails] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 50
	}

	var ctx = context.Background()

	var emails, ferr = a.application.Email().GetEmails(ctx, folder, limit)
	if ferr != nil {
		return nil, ferr
	}

	// Verify these emails still exist on server (purge deleted ones)
	if len(emails) > 0 {
		var uids = make([]uint32, len(emails))
		for i, e := range emails {
			uids[i] = e.UID
		}

		var deletedUIDs, purgeErr = a.application.Sync().PurgeSpecificUIDs(ctx, folder, uids)
		if purgeErr != nil {
			log.Printf("[GetEmails] purge check failed (non-fatal): %v", purgeErr)
		} else if len(deletedUIDs) > 0 {
			log.Printf("[GetEmails] purged %d deleted emails, reloading list", len(deletedUIDs))
			// Reload emails after purge
			emails, ferr = a.application.Email().GetEmails(ctx, folder, limit)
			if ferr != nil {
				return nil, ferr
			}
		}
	}

	for _, e := range emails {
		result = append(result, a.emailMetadataToDTO(&e))
	}
	return result, nil
}

// GetEmailsThreaded returns emails from a folder grouped by thread (only latest email per thread with thread count)
func (a *App) GetEmailsThreaded(folder string, limit int) (result []EmailDTO, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[GetEmailsThreaded] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return nil, nil
	}

	if a.account == nil {
		return nil, fmt.Errorf("no account set")
	}

	if limit <= 0 {
		limit = 50
	}

	var ctx = context.Background()

	// Get account and folder IDs
	var dbAccount, accountErr = storage.GetOrCreateAccount(a.account.Email, a.account.Name)
	if accountErr != nil {
		return nil, accountErr
	}

	var dbFolder, folderErr = storage.GetOrCreateFolder(dbAccount.ID, folder)
	if folderErr != nil {
		return nil, folderErr
	}

	// Get thread summaries (latest email per thread with thread count)
	var summaries, sErr = storage.GetThreadSummaries(dbAccount.ID, dbFolder.ID, limit, 0)
	if sErr != nil {
		return nil, sErr
	}

	// Verify these emails still exist on server (purge deleted ones)
	if len(summaries) > 0 {
		var uids = make([]uint32, len(summaries))
		for i, s := range summaries {
			uids[i] = s.UID
		}

		var deletedUIDs, purgeErr = a.application.Sync().PurgeSpecificUIDs(ctx, folder, uids)
		if purgeErr != nil {
			log.Printf("[GetEmailsThreaded] purge check failed (non-fatal): %v", purgeErr)
		} else if len(deletedUIDs) > 0 {
			log.Printf("[GetEmailsThreaded] purged %d deleted emails, reloading list", len(deletedUIDs))
			// Reload after purge
			summaries, sErr = storage.GetThreadSummaries(dbAccount.ID, dbFolder.ID, limit, 0)
			if sErr != nil {
				return nil, sErr
			}
		}
	}

	// Convert to EmailDTO
	for _, s := range summaries {
		result = append(result, EmailDTO{
			ID:             s.ID,
			UID:            s.UID,
			Subject:        s.Subject,
			FromName:       s.FromName,
			FromEmail:      s.FromEmail,
			Date:           s.Date.Time,
			IsRead:         s.IsRead,
			IsStarred:      s.IsStarred,
			HasAttachments: s.HasAttachments,
			Snippet:        s.Snippet,
			ThreadID:       s.ThreadID.String,
			ThreadCount:    s.ThreadCount,
		})
	}

	return result, nil
}

// GetEmail returns full email details by ID
func (a *App) GetEmail(id int64) (result *EmailDetailDTO, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[GetEmail] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return nil, nil
	}

	var email, ferr = a.application.Email().GetEmail(context.Background(), id)
	if ferr != nil {
		return nil, ferr
	}

	return a.emailContentToDTO(email), nil
}

// GetEmailByID returns email summary (EmailDTO) by ID for adding to email list
// This is used when selecting an email from search results that isn't in the current list
func (a *App) GetEmailByID(id int64) (result *EmailDTO, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[GetEmailByID] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return nil, nil
	}

	var email, ferr = a.application.Email().GetEmail(context.Background(), id)
	if ferr != nil {
		return nil, ferr
	}
	if email == nil {
		return nil, nil
	}

	// Convert to EmailDTO (summary format for list)
	var dto = EmailDTO{
		ID:             email.ID,
		UID:            email.UID,
		Subject:        email.Subject,
		FromName:       email.FromName,
		FromEmail:      email.FromEmail,
		Date:           email.Date,
		IsRead:         email.IsRead,
		IsStarred:      email.IsStarred,
		HasAttachments: email.HasAttachments,
		Snippet:        email.Snippet,
		ThreadID:       email.ThreadID,
	}
	return &dto, nil
}

// GetEmailByUID returns email by UID in current folder
func (a *App) GetEmailByUID(uid uint32) (*EmailDetailDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	a.mu.RLock()
	var folder = a.currentFolder
	a.mu.RUnlock()

	var email, err = a.application.Email().GetEmailByUID(context.Background(), folder, uid)
	if err != nil {
		return nil, err
	}

	return a.emailContentToDTO(email), nil
}

// ============================================================================
// EMAIL ACTIONS
// ============================================================================

// MarkAsRead marks an email as read or unread
func (a *App) MarkAsRead(id int64, read bool) error {
	if a.application == nil {
		return nil
	}
	return a.application.Email().MarkAsRead(context.Background(), id, read)
}

// MarkAsStarred marks an email as starred or unstarred
func (a *App) MarkAsStarred(id int64, starred bool) error {
	if a.application == nil {
		return nil
	}
	return a.application.Email().MarkAsStarred(context.Background(), id, starred)
}

// Archive archives an email
func (a *App) Archive(id int64) error {
	if a.application == nil {
		return nil
	}
	return a.application.Email().Archive(context.Background(), id)
}

// Delete moves an email to trash
func (a *App) Delete(id int64) error {
	if a.application == nil {
		return nil
	}
	return a.application.Email().Delete(context.Background(), id)
}

// MoveToFolder moves an email to a different folder
func (a *App) MoveToFolder(id int64, folder string) error {
	if a.application == nil {
		return nil
	}
	return a.application.Email().MoveToFolder(context.Background(), id, folder)
}

// ============================================================================
// BATCH OPERATIONS
// ============================================================================

// BatchArchive archives multiple emails
func (a *App) BatchArchive(ids []int64) error {
	if a.application == nil {
		return nil
	}
	var ctx = context.Background()
	for _, id := range ids {
		if err := a.application.Email().Archive(ctx, id); err != nil {
			log.Printf("[BatchArchive] failed to archive %d: %v", id, err)
			// Continue with other emails
		}
	}
	return nil
}

// BatchDelete deletes multiple emails
func (a *App) BatchDelete(ids []int64) error {
	if a.application == nil {
		return nil
	}
	var ctx = context.Background()
	for _, id := range ids {
		if err := a.application.Email().Delete(ctx, id); err != nil {
			log.Printf("[BatchDelete] failed to delete %d: %v", id, err)
		}
	}
	return nil
}

// BatchMarkRead marks multiple emails as read or unread
func (a *App) BatchMarkRead(ids []int64, read bool) error {
	if a.application == nil {
		return nil
	}
	var ctx = context.Background()
	for _, id := range ids {
		if err := a.application.Email().MarkAsRead(ctx, id, read); err != nil {
			log.Printf("[BatchMarkRead] failed to mark %d: %v", id, err)
		}
	}
	return nil
}

// BatchStar stars or unstars multiple emails
func (a *App) BatchStar(ids []int64, starred bool) error {
	if a.application == nil {
		return nil
	}
	var ctx = context.Background()
	for _, id := range ids {
		if err := a.application.Email().MarkAsStarred(ctx, id, starred); err != nil {
			log.Printf("[BatchStar] failed to star %d: %v", id, err)
		}
	}
	return nil
}

// ============================================================================
// SEARCH
// ============================================================================

// Search performs a full-text search
func (a *App) Search(query string, limit int) (*SearchResultDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 100 // Increased default limit
	}

	log.Printf("[Search] Starting search for '%s' with limit %d", query, limit)

	var result, err = a.application.Search().Search(context.Background(), query, limit)
	if err != nil {
		log.Printf("[Search] Error: %v", err)
		return nil, err
	}

	log.Printf("[Search] Got %d results (total: %d)", len(result.Emails), result.TotalCount)

	var emails []EmailDTO
	for _, e := range result.Emails {
		emails = append(emails, a.emailMetadataToDTO(&e))
	}

	return &SearchResultDTO{
		Emails:     emails,
		TotalCount: result.TotalCount,
		Query:      result.Query,
	}, nil
}

// SearchInFolder searches within a specific folder
func (a *App) SearchInFolder(folder, query string, limit int) (*SearchResultDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 50
	}

	var result, err = a.application.Search().SearchInFolder(context.Background(), folder, query, limit)
	if err != nil {
		return nil, err
	}

	var emails []EmailDTO
	for _, e := range result.Emails {
		emails = append(emails, a.emailMetadataToDTO(&e))
	}

	return &SearchResultDTO{
		Emails:     emails,
		TotalCount: result.TotalCount,
		Query:      result.Query,
	}, nil
}

// ============================================================================
// CONNECTION & SYNC
// ============================================================================

// Connect connects to the email server
func (a *App) Connect() (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Connect] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return nil
	}
	return a.application.Sync().Connect(context.Background())
}

// Disconnect disconnects from the email server
func (a *App) Disconnect() (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Disconnect] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return nil
	}
	return a.application.Sync().Disconnect(context.Background())
}

// IsConnected returns true if connected to email server
func (a *App) IsConnected() bool {
	if a.application == nil {
		return false
	}
	return a.application.Sync().IsConnected()
}

// SyncFolder syncs a specific folder and purges deleted emails
func (a *App) SyncFolder(folder string) (result *SyncResultDTO, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[SyncFolder] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	log.Printf("[SyncFolder] syncing folder: %s", folder)
	if a.application == nil {
		return nil, nil
	}

	var ctx = context.Background()

	// 1. Sync new emails from server
	var syncResult, syncErr = a.application.Sync().SyncFolder(ctx, folder)
	log.Printf("[SyncFolder] sync completed, err=%v", syncErr)
	if syncErr != nil {
		return nil, syncErr
	}

	// 2. Purge emails deleted from server (compare local UIDs with server UIDs)
	var purged, purgeErr = a.application.Sync().PurgeDeletedEmails(ctx, folder)
	if purgeErr != nil {
		log.Printf("[SyncFolder] purge error (non-fatal): %v", purgeErr)
	} else if purged > 0 {
		log.Printf("[SyncFolder] purged %d deleted emails", purged)
	}

	// 3. Sync thread IDs for NEW emails specifically (not random 50)
	if syncResult != nil && len(syncResult.NewEmailIDs) > 0 {
		log.Printf("[SyncFolder] syncing thread IDs for %d new emails", len(syncResult.NewEmailIDs))
		go a.syncThreadIDsForEmails(ctx, syncResult.NewEmailIDs)
	}

	if syncResult != nil {
		return &SyncResultDTO{
			NewEmails:     syncResult.NewEmails,
			DeletedEmails: purged,
		}, nil
	}
	return &SyncResultDTO{DeletedEmails: purged}, nil
}

// syncThreadIDsForEmails syncs thread IDs for specific email IDs via Gmail API
func (a *App) syncThreadIDsForEmails(ctx context.Context, emailIDs []int64) {
	if a.application == nil || len(emailIDs) == 0 {
		return
	}

	// Get Gmail adapter
	var coreApp, ok = a.application.(*app.Application)
	if !ok {
		return
	}
	var gmailAdapter = coreApp.GetGmailAdapter()
	if gmailAdapter == nil {
		log.Printf("[syncThreadIDsForEmails] Gmail adapter not available, skipping thread sync")
		return
	}

	// Get emails by IDs (excluding deleted ones)
	var emails, err = storage.GetEmailsByIDs(emailIDs)
	if err != nil || len(emails) == 0 {
		log.Printf("[syncThreadIDsForEmails] Failed to get emails: %v", err)
		return
	}

	// Filter out deleted emails
	var activeEmails []storage.EmailSummary
	for _, email := range emails {
		if !email.IsDeleted {
			activeEmails = append(activeEmails, email)
		}
	}

	if len(activeEmails) == 0 {
		log.Printf("[syncThreadIDsForEmails] No active emails to sync")
		return
	}

	log.Printf("[syncThreadIDsForEmails] Syncing thread IDs for %d emails (%d deleted skipped)", len(activeEmails), len(emails)-len(activeEmails))

	var updated = 0
	var notFound = 0
	var skipped = 0
	var notFoundIDs []int64

	for _, email := range activeEmails {
		// Skip if no message_id or already has thread_id
		if !email.MessageID.Valid || email.MessageID.String == "" {
			skipped++
			continue
		}
		if email.ThreadID.Valid && email.ThreadID.String != "" {
			skipped++
			continue
		}

		// Get thread ID from Gmail API
		var msgInfo, apiErr = gmailAdapter.GetMessageInfoByRFC822MsgID(email.MessageID.String)
		if apiErr != nil {
			// Track not found errors (likely deleted from Gmail)
			if strings.Contains(apiErr.Error(), "não encontrada") || strings.Contains(apiErr.Error(), "not found") {
				notFound++
				notFoundIDs = append(notFoundIDs, email.ID)
			}
			continue
		}

		if msgInfo != nil && msgInfo.ThreadID != "" {
			// Update thread_id in database
			if updateErr := storage.UpdateEmailThreadID(email.ID, msgInfo.ThreadID); updateErr == nil {
				updated++
			}
		}
	}

	// Mark "not found" emails as deleted (they were deleted from Gmail)
	if len(notFoundIDs) > 0 {
		storage.MarkDeletedByEmailIDs(notFoundIDs)
		log.Printf("[syncThreadIDsForEmails] Marked %d emails as deleted (not found in Gmail)", len(notFoundIDs))
	}

	log.Printf("[syncThreadIDsForEmails] Updated %d thread IDs, %d not found, %d skipped", updated, notFound, skipped)
}

// syncNewEmailThreadIDs syncs thread IDs for emails that don't have one yet (legacy/fallback)
func (a *App) syncNewEmailThreadIDs(ctx context.Context) {
	if a.application == nil {
		return
	}

	var account = a.application.GetCurrentAccount()
	if account == nil {
		return
	}

	// Get emails without thread_id (limit to recent ones for performance)
	var emails, err = storage.GetEmailsNeedingThreadSync(account.ID, 50)
	if err != nil || len(emails) == 0 {
		return
	}

	log.Printf("[syncNewEmailThreadIDs] Found %d emails needing thread_id sync", len(emails))

	// Get Gmail adapter through the app
	var coreApp, ok = a.application.(*app.Application)
	if !ok {
		return
	}
	var gmailAdapter = coreApp.GetGmailAdapter()
	if gmailAdapter == nil {
		return
	}

	var updated = 0
	for _, email := range emails {
		if email.MessageID == "" {
			continue
		}

		// Get thread ID from Gmail API
		var msgInfo, apiErr = gmailAdapter.GetMessageInfoByRFC822MsgID(email.MessageID)
		if apiErr != nil {
			continue
		}

		if msgInfo != nil && msgInfo.ThreadID != "" {
			// Update thread_id in database
			if updateErr := storage.UpdateEmailThreadID(email.ID, msgInfo.ThreadID); updateErr == nil {
				updated++
			}
		}
	}

	if updated > 0 {
		log.Printf("[syncNewEmailThreadIDs] Updated %d thread IDs", updated)
	}
}

// SyncCurrentFolder syncs the currently selected folder
func (a *App) SyncCurrentFolder() (*SyncResultDTO, error) {
	a.mu.RLock()
	var folder = a.currentFolder
	a.mu.RUnlock()

	return a.SyncFolder(folder)
}

// SyncEssentialFolders syncs essential folders (INBOX, Sent, Trash) and purges deleted
func (a *App) SyncEssentialFolders() (results []SyncResultDTO, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[SyncEssentialFolders] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	log.Printf("[SyncEssentialFolders] syncing essential folders")
	if a.application == nil {
		return nil, nil
	}

	var ctx = context.Background()

	// 1. Sync new emails from essential folders
	var syncResults, syncErr = a.application.Sync().SyncEssentialFolders(ctx)
	log.Printf("[SyncEssentialFolders] sync completed, err=%v", syncErr)
	if syncErr != nil {
		return nil, syncErr
	}

	// 2. Purge deleted emails from INBOX (most important folder)
	var purged, purgeErr = a.application.Sync().PurgeDeletedEmails(ctx, "INBOX")
	if purgeErr != nil {
		log.Printf("[SyncEssentialFolders] purge INBOX error (non-fatal): %v", purgeErr)
	} else if purged > 0 {
		log.Printf("[SyncEssentialFolders] purged %d deleted emails from INBOX", purged)
	}

	// 3. Collect all new email IDs for thread sync
	var allNewEmailIDs []int64
	for i, r := range syncResults {
		var deletedCount = 0
		if i == 0 { // INBOX is first
			deletedCount = purged
		}
		results = append(results, SyncResultDTO{
			NewEmails:     r.NewEmails,
			DeletedEmails: deletedCount,
		})
		// Collect IDs for thread sync
		log.Printf("[SyncEssentialFolders] folder %d: NewEmails=%d, NewEmailIDs=%d", i, r.NewEmails, len(r.NewEmailIDs))
		allNewEmailIDs = append(allNewEmailIDs, r.NewEmailIDs...)
	}

	// 4. Sync thread IDs for ALL new emails from all folders (SYNCHRONOUS - must complete before returning)
	if len(allNewEmailIDs) > 0 {
		log.Printf("[SyncEssentialFolders] syncing thread IDs for %d new emails", len(allNewEmailIDs))
		a.syncThreadIDsForEmails(ctx, allNewEmailIDs)
	}

	return results, nil
}

// GetConnectionStatus returns current connection status
func (a *App) GetConnectionStatus() ConnectionStatus {
	if a.application == nil {
		return ConnectionStatus{Connected: false}
	}

	a.mu.RLock()
	var connected = a.connected
	a.mu.RUnlock()

	return ConnectionStatus{
		Connected: connected,
	}
}

// SyncThreadsFromGmail syncs thread IDs from Gmail API
// This is more accurate than local thread detection
func (a *App) SyncThreadsFromGmail() (updated int, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[SyncThreadsFromGmail] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	log.Printf("[SyncThreadsFromGmail] starting Gmail thread sync")
	if a.application == nil {
		return 0, fmt.Errorf("application not initialized")
	}

	// Create cancellable context
	var ctx, cancel = context.WithCancel(context.Background())
	a.mu.Lock()
	a.threadSyncCancel = cancel
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.threadSyncCancel = nil
		a.mu.Unlock()
	}()

	var count, syncErr = a.application.SyncThreadIDsFromGmail(ctx, func(processed, total int) {
		// Negative processed = listing phase (page number), total = accumulated count
		if processed < 0 {
			log.Printf("[SyncThreadsFromGmail] listing messages: page %d (%d found)", -processed, total)
			if a.wailsApp != nil {
				a.wailsApp.Event.Emit("thread-sync-progress", map[string]interface{}{
					"phase":     "listing",
					"page":      -processed,
					"found":     total, // accumulated count
					"processed": 0,
					"total":     0,
				})
			}
		} else {
			log.Printf("[SyncThreadsFromGmail] fetching metadata: %d/%d", processed, total)
			if a.wailsApp != nil {
				a.wailsApp.Event.Emit("thread-sync-progress", map[string]interface{}{
					"phase":     "fetching",
					"processed": processed,
					"total":     total,
				})
			}
		}
	})

	if syncErr != nil && ctx.Err() != nil {
		log.Printf("[SyncThreadsFromGmail] cancelled by user")
		return count, fmt.Errorf("cancelled")
	}

	log.Printf("[SyncThreadsFromGmail] completed, updated=%d, err=%v", count, syncErr)
	return count, syncErr
}

// CancelThreadSync cancels an ongoing thread sync operation
func (a *App) CancelThreadSync() {
	a.mu.RLock()
	var cancel = a.threadSyncCancel
	a.mu.RUnlock()

	if cancel != nil {
		log.Printf("[CancelThreadSync] cancelling thread sync")
		cancel()
	}
}

// ============================================================================
// SEND EMAIL
// ============================================================================

// SendEmail sends an email
func (a *App) SendEmail(req SendRequest) (*SendResult, error) {
	if a.application == nil {
		return &SendResult{Success: false, Error: "Application not initialized"}, nil
	}

	var portsReq = &ports.SendRequest{
		To:       req.To,
		Cc:       req.Cc,
		Bcc:      req.Bcc,
		Subject:  req.Subject,
		BodyText: req.Body,
	}

	if req.IsHTML {
		portsReq.BodyHTML = req.Body
		portsReq.BodyText = "" // TODO: generate text version
	}

	if req.ReplyTo > 0 {
		portsReq.ReplyToEmailID = &req.ReplyTo
	}

	var result, err = a.application.Send().Send(context.Background(), portsReq)
	if err != nil {
		return &SendResult{Success: false, Error: err.Error()}, nil
	}

	return &SendResult{
		Success:   result.Success,
		MessageID: result.MessageID,
		Error:     a.getError(result.Error),
	}, nil
}

// GetSignature returns the configured email signature
func (a *App) GetSignature() (sig string, err error) {
	// Recover from any panic to prevent crash
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[GetSignature] PANIC recovered: %v", r)
			sig = ""
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	log.Printf("[GetSignature] called, application=%v", a.application != nil)

	if a.application == nil {
		log.Printf("[GetSignature] application is nil, returning empty")
		return "", nil
	}

	sendService := a.application.Send()
	log.Printf("[GetSignature] sendService=%v", sendService != nil)

	if sendService == nil {
		log.Printf("[GetSignature] sendService is nil, returning empty")
		return "", nil
	}

	log.Printf("[GetSignature] calling GetSignature on sendService...")
	sig, err = sendService.GetSignature(context.Background())
	log.Printf("[GetSignature] result: sig=%d bytes, err=%v", len(sig), err)

	return sig, err
}

// ============================================================================
// DRAFTS
// ============================================================================

// SaveDraft saves a draft email
func (a *App) SaveDraft(draft DraftDTO) (int64, error) {
	if a.application == nil {
		return 0, nil
	}

	var portsDraft = &ports.Draft{
		ID:           draft.ID,
		ToAddresses:  strings.Join(draft.To, ", "),
		CcAddresses:  strings.Join(draft.Cc, ", "),
		BccAddresses: strings.Join(draft.Bcc, ", "),
		Subject:      draft.Subject,
		BodyHTML:     draft.BodyHTML,
		BodyText:     draft.BodyText,
		Status:       ports.DraftStatusDraft,
	}

	if draft.ReplyToID > 0 {
		portsDraft.ReplyToEmailID = &draft.ReplyToID
	}

	var result *ports.Draft
	var err error

	if draft.ID > 0 {
		err = a.application.Draft().UpdateDraft(context.Background(), portsDraft)
		if err != nil {
			return 0, err
		}
		result = portsDraft
	} else {
		result, err = a.application.Draft().CreateDraft(context.Background(), portsDraft)
		if err != nil {
			return 0, err
		}
	}

	return result.ID, nil
}

// GetDraft returns a draft by ID
func (a *App) GetDraft(id int64) (*DraftDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	var draft, err = a.application.Draft().GetDraft(context.Background(), id)
	if err != nil {
		return nil, err
	}

	return a.draftToDTO(draft), nil
}

// ListDrafts returns all drafts
func (a *App) ListDrafts() ([]DraftDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	var drafts, err = a.application.Draft().ListDrafts(context.Background())
	if err != nil {
		return nil, err
	}

	var result []DraftDTO
	for _, d := range drafts {
		result = append(result, *a.draftToDTO(&d))
	}
	return result, nil
}

// DeleteDraft deletes a draft
func (a *App) DeleteDraft(id int64) error {
	if a.application == nil {
		return nil
	}
	return a.application.Draft().DeleteDraft(context.Background(), id)
}

// SendDraft sends a draft
func (a *App) SendDraft(id int64) (*SendResult, error) {
	if a.application == nil {
		return &SendResult{Success: false, Error: "Application not initialized"}, nil
	}

	var result, err = a.application.Send().SendDraft(context.Background(), id)
	if err != nil {
		return &SendResult{Success: false, Error: err.Error()}, nil
	}

	return &SendResult{
		Success:   result.Success,
		MessageID: result.MessageID,
		Error:     a.getError(result.Error),
	}, nil
}

// ============================================================================
// AI INTEGRATION
// ============================================================================

// AskAI sends a question to a CLI-based AI provider
func (a *App) AskAI(provider, question, emailContextJSON string) (string, error) {
	var cmd *exec.Cmd
	var prompt = question

	// Add email context if provided
	if emailContextJSON != "" {
		prompt = fmt.Sprintf("Contexto do email:\n%s\n\nPergunta: %s", emailContextJSON, question)
	}

	switch provider {
	case "claude":
		// Claude Code CLI - uses stdin for prompt
		// --dangerously-skip-permissions allows sqlite3 without asking
		cmd = exec.Command("claude", "-p", "--dangerously-skip-permissions", prompt)
	case "gemini":
		// Gemini CLI
		cmd = exec.Command("gemini", prompt)
	case "ollama":
		// Ollama with llama3
		cmd = exec.Command("ollama", "run", "llama3", prompt)
	case "openai":
		// OpenAI CLI (if installed)
		cmd = exec.Command("openai", "api", "chat.completions.create", "-m", "gpt-4", "-g", "user", prompt)
	default:
		return "", fmt.Errorf("provider não suportado: %s", provider)
	}

	var output, err = cmd.CombinedOutput()
	if err != nil {
		// Try to provide helpful error message
		if strings.Contains(err.Error(), "executable file not found") {
			return "", fmt.Errorf("%s CLI não encontrado. Instale com: %s", provider, getInstallHint(provider))
		}
		return "", fmt.Errorf("erro ao executar %s: %v\nOutput: %s", provider, err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// getInstallHint returns installation instructions for AI CLIs
func getInstallHint(provider string) string {
	switch provider {
	case "claude":
		return "npm install -g @anthropic-ai/claude-code"
	case "gemini":
		return "pip install google-generativeai"
	case "ollama":
		return "curl https://ollama.ai/install.sh | sh"
	case "openai":
		return "pip install openai"
	default:
		return "consulte a documentação do provider"
	}
}

// SummarizeEmail summarizes a single email using AI
func (a *App) SummarizeEmail(emailID int64) (string, error) {
	if a.application == nil {
		return "", fmt.Errorf("application not initialized")
	}

	var aiService = a.application.AI()
	if aiService == nil {
		return "", fmt.Errorf("AI service not available")
	}

	return aiService.Summarize(context.Background(), emailID)
}

// SummarizeThread summarizes an entire email thread using AI
func (a *App) SummarizeThread(emailID int64) (string, error) {
	if a.application == nil {
		return "", fmt.Errorf("application not initialized")
	}

	var aiService = a.application.AI()
	if aiService == nil {
		return "", fmt.Errorf("AI service not available")
	}

	return aiService.SummarizeThread(context.Background(), emailID)
}

// SummaryResult represents an AI summary with metadata
type SummaryResult struct {
	EmailID   int64    `json:"emailId"`
	Style     string   `json:"style"`
	Content   string   `json:"content"`
	KeyPoints []string `json:"keyPoints"`
	Cached    bool     `json:"cached"`
}

// SummarizeEmailWithStyle summarizes an email with a specific style (tldr, brief, detailed)
func (a *App) SummarizeEmailWithStyle(emailID int64, style string) (*SummaryResult, error) {
	if a.application == nil {
		return nil, fmt.Errorf("application not initialized")
	}

	var aiService = a.application.AI()
	if aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	var summaryStyle = ports.SummaryStyleBrief
	switch style {
	case "tldr":
		summaryStyle = ports.SummaryStyleTLDR
	case "detailed":
		summaryStyle = ports.SummaryStyleDetailed
	}

	var summary, err = aiService.SummarizeWithStyle(context.Background(), emailID, summaryStyle)
	if err != nil {
		return nil, err
	}

	return &SummaryResult{
		EmailID:   summary.EmailID,
		Style:     string(summary.Style),
		Content:   summary.Content,
		KeyPoints: summary.KeyPoints,
		Cached:    summary.Cached,
	}, nil
}

// GetCachedSummary retrieves a cached summary if exists
func (a *App) GetCachedSummary(emailID int64) (*SummaryResult, error) {
	if a.application == nil {
		return nil, fmt.Errorf("application not initialized")
	}

	var aiService = a.application.AI()
	if aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	var summary, err = aiService.GetCachedSummary(context.Background(), emailID)
	if err != nil {
		return nil, err
	}
	if summary == nil {
		return nil, nil
	}

	return &SummaryResult{
		EmailID:   summary.EmailID,
		Style:     string(summary.Style),
		Content:   summary.Content,
		KeyPoints: summary.KeyPoints,
		Cached:    true,
	}, nil
}

// InvalidateSummary removes a cached summary
func (a *App) InvalidateSummary(emailID int64) error {
	if a.application == nil {
		return fmt.Errorf("application not initialized")
	}

	var aiService = a.application.AI()
	if aiService == nil {
		return fmt.Errorf("AI service not available")
	}

	return aiService.InvalidateSummary(context.Background(), emailID)
}

// ThreadSummaryResult represents a detailed thread summary
type ThreadSummaryResult struct {
	ThreadID     string   `json:"threadId"`
	Participants []string `json:"participants"`
	Timeline     string   `json:"timeline"`
	KeyDecisions []string `json:"keyDecisions"`
	ActionItems  []string `json:"actionItems"`
	Cached       bool     `json:"cached"`
}

// SummarizeThreadDetailed returns detailed thread summary with structured data
func (a *App) SummarizeThreadDetailed(emailID int64) (*ThreadSummaryResult, error) {
	if a.application == nil {
		return nil, fmt.Errorf("application not initialized")
	}

	var aiService = a.application.AI()
	if aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	var summary, err = aiService.SummarizeThreadDetailed(context.Background(), emailID)
	if err != nil {
		return nil, err
	}

	return &ThreadSummaryResult{
		ThreadID:     summary.ThreadID,
		Participants: summary.Participants,
		Timeline:     summary.Timeline,
		KeyDecisions: summary.KeyDecisions,
		ActionItems:  summary.ActionItems,
		Cached:       summary.Cached,
	}, nil
}

// ExtractActions extracts action items from an email using AI
func (a *App) ExtractActions(emailID int64) ([]string, error) {
	if a.application == nil {
		return nil, fmt.Errorf("application not initialized")
	}

	var aiService = a.application.AI()
	if aiService == nil {
		return nil, fmt.Errorf("AI service not available")
	}

	return aiService.ExtractActions(context.Background(), emailID)
}

// GetAIProviders returns available AI providers and their status
func (a *App) GetAIProviders() []map[string]interface{} {
	providers := []struct {
		id   string
		name string
		cmd  string
	}{
		{"claude", "Claude", "claude"},
		{"gemini", "Gemini", "gemini"},
		{"ollama", "Ollama", "ollama"},
		{"openai", "OpenAI", "openai"},
	}

	var result []map[string]interface{}
	for _, p := range providers {
		_, err := exec.LookPath(p.cmd)
		result = append(result, map[string]interface{}{
			"id":        p.id,
			"name":      p.name,
			"available": err == nil,
		})
	}
	return result
}

// ============================================================================
// ATTACHMENTS
// ============================================================================

// GetAttachments returns all attachments for an email
func (a *App) GetAttachments(emailID int64) (result []AttachmentDTO, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[GetAttachments] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return nil, nil
	}

	var attService = a.application.Attachment()
	if attService == nil {
		return nil, nil
	}

	var attachments, ferr = attService.GetAttachments(context.Background(), emailID)
	if ferr != nil {
		return nil, ferr
	}

	for _, att := range attachments {
		result = append(result, AttachmentDTO{
			ID:          att.ID,
			Filename:    att.Filename,
			ContentType: att.ContentType,
			ContentID:   att.ContentID,
			Size:        att.Size,
			IsInline:    att.IsInline,
		})
	}
	return result, nil
}

// DownloadAttachment downloads an attachment and returns base64-encoded content
func (a *App) DownloadAttachment(attachmentID int64) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[DownloadAttachment] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return "", fmt.Errorf("application not initialized")
	}

	var attService = a.application.Attachment()
	if attService == nil {
		return "", fmt.Errorf("attachment service not available")
	}

	var data, ferr = attService.Download(context.Background(), attachmentID)
	if ferr != nil {
		return "", ferr
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

// SaveAttachment saves an attachment to a file
func (a *App) SaveAttachment(attachmentID int64, path string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[SaveAttachment] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return fmt.Errorf("application not initialized")
	}

	var attService = a.application.Attachment()
	if attService == nil {
		return fmt.Errorf("attachment service not available")
	}

	return attService.SaveToFile(context.Background(), attachmentID, path)
}

// SaveAttachmentDialog opens a file dialog and saves attachment to selected location
func (a *App) SaveAttachmentDialog(attachmentID int64, filename string) (path string, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[SaveAttachmentDialog] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	// Show save file dialog
	if a.wailsApp == nil {
		return "", fmt.Errorf("wails app not initialized")
	}
	savePath, dialogErr := a.wailsApp.Dialog.SaveFile().
		SetFilename(filename).
		PromptForSingleSelection()
	if dialogErr != nil {
		return "", dialogErr
	}
	if savePath == "" {
		return "", nil // User cancelled
	}

	// Save the attachment
	if err := a.SaveAttachment(attachmentID, savePath); err != nil {
		return "", err
	}

	return savePath, nil
}

// OpenAttachment downloads attachment to temp and opens with default app
func (a *App) OpenAttachment(attachmentID int64, filename string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[OpenAttachment] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	if a.application == nil {
		return fmt.Errorf("application not initialized")
	}

	var attService = a.application.Attachment()
	if attService == nil {
		return fmt.Errorf("attachment service not available")
	}

	// Download to temp file
	var tempDir = os.TempDir()
	var tempPath = fmt.Sprintf("%s/miau-%d-%s", tempDir, attachmentID, filename)

	if err := attService.SaveToFile(context.Background(), attachmentID, tempPath); err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}

	// Open with default application
	var cmd *exec.Cmd
	switch goos := strings.ToLower(os.Getenv("GOOS")); {
	case goos == "darwin" || (goos == "" && fileExists("/usr/bin/open")):
		cmd = exec.Command("open", tempPath)
	case goos == "windows" || (goos == "" && fileExists("C:\\Windows\\System32\\cmd.exe")):
		cmd = exec.Command("cmd", "/c", "start", "", tempPath)
	default:
		cmd = exec.Command("xdg-open", tempPath)
	}

	return cmd.Start()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SaveAttachmentByPart downloads attachment by email ID + part number and saves to user-selected location
func (a *App) SaveAttachmentByPart(emailID int64, partNumber string, filename string) (path string, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[SaveAttachmentByPart] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	if a.application == nil {
		return "", fmt.Errorf("application not initialized")
	}

	var attService = a.application.Attachment()
	if attService == nil {
		return "", fmt.Errorf("attachment service not available")
	}

	// Show save file dialog
	if a.wailsApp == nil {
		return "", fmt.Errorf("wails app not initialized")
	}
	savePath, dialogErr := a.wailsApp.Dialog.SaveFile().
		SetFilename(filename).
		PromptForSingleSelection()
	if dialogErr != nil {
		return "", dialogErr
	}
	if savePath == "" {
		return "", nil // User cancelled
	}

	// Download and save
	if err := attService.SaveToFileByPart(context.Background(), emailID, partNumber, savePath); err != nil {
		return "", err
	}

	return savePath, nil
}

// OpenAttachmentByPart downloads attachment by email ID + part number and opens with default app
func (a *App) OpenAttachmentByPart(emailID int64, partNumber string, filename string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[OpenAttachmentByPart] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	if a.application == nil {
		return fmt.Errorf("application not initialized")
	}

	var attService = a.application.Attachment()
	if attService == nil {
		return fmt.Errorf("attachment service not available")
	}

	// Download content
	var data, downloadErr = attService.DownloadByPart(context.Background(), emailID, partNumber)
	if downloadErr != nil {
		return fmt.Errorf("failed to download: %w", downloadErr)
	}

	// Save to temp file
	var tempDir = os.TempDir()
	var tempPath = fmt.Sprintf("%s/miau-%d-%s-%s", tempDir, emailID, partNumber, filename)

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Open with default application
	var cmd *exec.Cmd
	switch goos := strings.ToLower(os.Getenv("GOOS")); {
	case goos == "darwin" || (goos == "" && fileExists("/usr/bin/open")):
		cmd = exec.Command("open", tempPath)
	case goos == "windows" || (goos == "" && fileExists("C:\\Windows\\System32\\cmd.exe")):
		cmd = exec.Command("cmd", "/c", "start", "", tempPath)
	default:
		cmd = exec.Command("xdg-open", tempPath)
	}

	return cmd.Start()
}

// ============================================================================
// ACCOUNTS
// ============================================================================

// GetAccounts returns all configured accounts
func (a *App) GetAccounts() []AccountDTO {
	if a.cfg == nil {
		return nil
	}

	var result []AccountDTO
	for _, acc := range a.cfg.Accounts {
		result = append(result, AccountDTO{
			Email: acc.Email,
			Name:  acc.Name,
		})
	}
	return result
}

// ============================================================================
// ANALYTICS
// ============================================================================

// GetAnalytics returns comprehensive analytics for a time period
func (a *App) GetAnalytics(period string) (*AnalyticsResultDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	if period == "" {
		period = "30d"
	}

	var result, err = a.application.Analytics().GetAnalytics(context.Background(), period)
	if err != nil {
		return nil, err
	}

	return a.analyticsResultToDTO(result), nil
}

// GetAnalyticsOverview returns basic email statistics
func (a *App) GetAnalyticsOverview() (*AnalyticsOverviewDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	var overview, err = a.application.Analytics().GetOverview(context.Background())
	if err != nil {
		return nil, err
	}

	return &AnalyticsOverviewDTO{
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
func (a *App) GetTopSenders(limit int, period string) ([]SenderStatsDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 10
	}
	if period == "" {
		period = "30d"
	}

	var senders, err = a.application.Analytics().GetTopSenders(context.Background(), limit, period)
	if err != nil {
		return nil, err
	}

	var result []SenderStatsDTO
	for _, s := range senders {
		result = append(result, SenderStatsDTO{
			Email:       s.Email,
			Name:        s.Name,
			Count:       s.Count,
			UnreadCount: s.UnreadCount,
			Percentage:  s.Percentage,
		})
	}
	return result, nil
}

// analyticsResultToDTO converts ports.AnalyticsResult to AnalyticsResultDTO
func (a *App) analyticsResultToDTO(result *ports.AnalyticsResult) *AnalyticsResultDTO {
	if result == nil {
		return nil
	}

	var topSenders []SenderStatsDTO
	for _, s := range result.TopSenders {
		topSenders = append(topSenders, SenderStatsDTO{
			Email:       s.Email,
			Name:        s.Name,
			Count:       s.Count,
			UnreadCount: s.UnreadCount,
			Percentage:  s.Percentage,
		})
	}

	var daily []DailyStatsDTO
	for _, d := range result.Trends.Daily {
		daily = append(daily, DailyStatsDTO{
			Date:  d.Date,
			Count: d.Count,
		})
	}

	var hourly []HourlyStatsDTO
	for _, h := range result.Trends.Hourly {
		hourly = append(hourly, HourlyStatsDTO{
			Hour:  h.Hour,
			Count: h.Count,
		})
	}

	var weekday []WeekdayStatsDTO
	for _, w := range result.Trends.Weekday {
		weekday = append(weekday, WeekdayStatsDTO{
			Weekday: w.Weekday,
			Name:    w.Name,
			Count:   w.Count,
		})
	}

	return &AnalyticsResultDTO{
		Overview: AnalyticsOverviewDTO{
			TotalEmails:    result.Overview.TotalEmails,
			UnreadEmails:   result.Overview.UnreadEmails,
			StarredEmails:  result.Overview.StarredEmails,
			ArchivedEmails: result.Overview.ArchivedEmails,
			SentEmails:     result.Overview.SentEmails,
			DraftCount:     result.Overview.DraftCount,
			StorageUsedMB:  result.Overview.StorageUsedMB,
		},
		TopSenders: topSenders,
		Trends: EmailTrendsDTO{
			Daily:   daily,
			Hourly:  hourly,
			Weekday: weekday,
		},
		ResponseTime: ResponseTimeStatsDTO{
			AvgResponseMinutes: result.ResponseTime.AvgResponseMinutes,
			ResponseRate:       result.ResponseTime.ResponseRate,
		},
		Period:      result.Period,
		GeneratedAt: result.GeneratedAt,
	}
}

// ============================================================================
// SETTINGS
// ============================================================================

// GetSettings returns all application settings
func (a *App) GetSettings() (*SettingsDTO, error) {
	if a.account == nil {
		return nil, fmt.Errorf("no account set")
	}

	// Get account ID from database
	var dbAccount, accountErr = storage.GetOrCreateAccount(a.account.Email, a.account.Name)
	if accountErr != nil {
		return nil, accountErr
	}

	var settings, err = storage.GetAllSettings(dbAccount.ID)
	if err != nil {
		return nil, err
	}

	// Parse sync folders from JSON
	var syncFolders []string
	if foldersJSON, ok := settings["sync_folders"]; ok && foldersJSON != "" {
		json.Unmarshal([]byte(foldersJSON), &syncFolders)
	}
	if len(syncFolders) == 0 {
		// Default to essential folders
		syncFolders = []string{"INBOX", "[Gmail]/Sent Mail", "[Gmail]/Trash"}
	}

	// Parse other settings with defaults
	var uiTheme = settings["ui_theme"]
	if uiTheme == "" {
		uiTheme = "dark"
	}

	var uiShowPreview = true
	if val, ok := settings["ui_show_preview"]; ok {
		uiShowPreview = val == "true"
	}

	var uiPageSize = 50
	if val, ok := settings["ui_page_size"]; ok {
		fmt.Sscanf(val, "%d", &uiPageSize)
	}

	var composeFormat = settings["compose_format"]
	if composeFormat == "" {
		composeFormat = "html"
	}

	var composeSendDelay = 30
	if val, ok := settings["compose_send_delay"]; ok {
		fmt.Sscanf(val, "%d", &composeSendDelay)
	}

	var syncInterval = settings["sync_interval"]
	if syncInterval == "" {
		syncInterval = "5m"
	}

	return &SettingsDTO{
		SyncFolders:      syncFolders,
		UITheme:          uiTheme,
		UIShowPreview:    uiShowPreview,
		UIPageSize:       uiPageSize,
		ComposeFormat:    composeFormat,
		ComposeSendDelay: composeSendDelay,
		SyncInterval:     syncInterval,
	}, nil
}

// SaveSettings saves all application settings
func (a *App) SaveSettings(settings SettingsDTO) error {
	if a.account == nil {
		return fmt.Errorf("no account set")
	}

	// Get account ID from database
	var dbAccount, accountErr = storage.GetOrCreateAccount(a.account.Email, a.account.Name)
	if accountErr != nil {
		return accountErr
	}

	// Save sync folders as JSON
	var foldersJSON, _ = json.Marshal(settings.SyncFolders)
	storage.SetSetting(dbAccount.ID, "sync_folders", string(foldersJSON))

	// Save other settings
	storage.SetSetting(dbAccount.ID, "ui_theme", settings.UITheme)
	storage.SetSetting(dbAccount.ID, "ui_show_preview", fmt.Sprintf("%v", settings.UIShowPreview))
	storage.SetSetting(dbAccount.ID, "ui_page_size", fmt.Sprintf("%d", settings.UIPageSize))
	storage.SetSetting(dbAccount.ID, "compose_format", settings.ComposeFormat)
	storage.SetSetting(dbAccount.ID, "compose_send_delay", fmt.Sprintf("%d", settings.ComposeSendDelay))
	storage.SetSetting(dbAccount.ID, "sync_interval", settings.SyncInterval)

	return nil
}

// GetAvailableFolders returns all folders with their sync selection status
func (a *App) GetAvailableFolders() ([]AvailableFolderDTO, error) {
	if a.account == nil {
		return nil, fmt.Errorf("no account set")
	}

	// Get account ID from database
	var dbAccount, accountErr = storage.GetOrCreateAccount(a.account.Email, a.account.Name)
	if accountErr != nil {
		return nil, accountErr
	}

	// Get all folders from database
	var folders, err = storage.GetFolders(dbAccount.ID)
	if err != nil {
		return nil, err
	}

	// Get currently selected folders
	var settings, _ = a.GetSettings()
	var selectedSet = make(map[string]bool)
	if settings != nil {
		for _, f := range settings.SyncFolders {
			selectedSet[f] = true
		}
	}

	var result []AvailableFolderDTO
	for _, folder := range folders {
		result = append(result, AvailableFolderDTO{
			Name:       folder.Name,
			IsSelected: selectedSet[folder.Name],
		})
	}

	return result, nil
}

// ============================================================================
// THREADS
// ============================================================================

// GetThread returns a complete thread with all messages for a given email ID
func (a *App) GetThread(emailID int64) (result *ThreadDTO, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[GetThread] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	log.Printf("[GetThread] called with emailID=%d", emailID)
	if a.application == nil {
		log.Printf("[GetThread] application is nil")
		return nil, nil
	}

	var thread, ferr = a.application.Thread().GetThread(context.Background(), emailID)
	if ferr != nil {
		log.Printf("[GetThread] error: %v", ferr)
		return nil, ferr
	}

	log.Printf("[GetThread] got thread with %d messages", len(thread.Messages))
	return a.threadToDTO(thread), nil
}

// GetThreadByID returns a thread by its thread_id
func (a *App) GetThreadByID(threadID string) (result *ThreadDTO, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[GetThreadByID] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return nil, nil
	}

	var thread, ferr = a.application.Thread().GetThreadByID(context.Background(), threadID)
	if ferr != nil {
		return nil, ferr
	}

	return a.threadToDTO(thread), nil
}

// GetThreadSummary returns thread metadata for inbox display
func (a *App) GetThreadSummary(threadID string) (result *ThreadSummaryDTO, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[GetThreadSummary] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return nil, nil
	}

	var summary, ferr = a.application.Thread().GetThreadSummary(context.Background(), threadID)
	if ferr != nil {
		return nil, ferr
	}

	return &ThreadSummaryDTO{
		ThreadID:        summary.ThreadID,
		Subject:         summary.Subject,
		LastSender:      summary.LastSender,
		LastSenderEmail: summary.LastSenderEmail,
		LastDate:        summary.LastDate,
		MessageCount:    summary.MessageCount,
		UnreadCount:     summary.UnreadCount,
		HasAttachments:  summary.HasAttachments,
		Participants:    summary.Participants,
	}, nil
}

// GetThreadMessageCount returns the number of messages in a thread for an email
func (a *App) GetThreadMessageCount(emailID int64) (count int, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[GetThreadMessageCount] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return 0, nil
	}

	// First get the email to find its thread_id
	var email, emailErr = a.application.Email().GetEmail(context.Background(), emailID)
	if emailErr != nil {
		return 0, emailErr
	}

	if email.ThreadID == "" {
		return 1, nil // Single email, no thread
	}

	return a.application.Thread().CountThreadMessages(context.Background(), email.ThreadID)
}

// MarkThreadAsRead marks all messages in a thread as read
func (a *App) MarkThreadAsRead(threadID string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MarkThreadAsRead] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return nil
	}

	return a.application.Thread().MarkThreadAsRead(context.Background(), threadID)
}

// MarkThreadAsUnread marks the most recent message in a thread as unread
func (a *App) MarkThreadAsUnread(threadID string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MarkThreadAsUnread] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	if a.application == nil {
		return nil
	}

	return a.application.Thread().MarkThreadAsUnread(context.Background(), threadID)
}

// threadToDTO converts ports.Thread to ThreadDTO
func (a *App) threadToDTO(thread *ports.Thread) *ThreadDTO {
	if thread == nil {
		return nil
	}

	var messages []ThreadEmailDTO
	for _, msg := range thread.Messages {
		// Generate snippet from body if not available
		var snippet = msg.Snippet
		if snippet == "" && msg.BodyText != "" {
			snippet = generateSnippet(msg.BodyText, 150)
		} else if snippet == "" && msg.BodyHTML != "" {
			snippet = generateSnippetFromHTML(msg.BodyHTML, 150)
		}

		messages = append(messages, ThreadEmailDTO{
			ID:             msg.ID,
			UID:            msg.UID,
			MessageID:      msg.MessageID,
			Subject:        msg.Subject,
			FromName:       msg.FromName,
			FromEmail:      msg.FromEmail,
			ToAddresses:    msg.ToAddresses,
			Date:           msg.Date,
			IsRead:         msg.IsRead,
			IsStarred:      msg.IsStarred,
			IsReplied:      msg.IsReplied,
			HasAttachments: msg.HasAttachments,
			Snippet:        snippet,
			BodyText:       msg.BodyText,
			BodyHTML:       msg.BodyHTML,
		})
	}

	return &ThreadDTO{
		ThreadID:     thread.ThreadID,
		Subject:      thread.Subject,
		Participants: thread.Participants,
		MessageCount: thread.MessageCount,
		Messages:     messages,
		IsRead:       thread.IsRead,
	}
}

// ============================================================================
// HELPERS
// ============================================================================

// draftToDTO converts ports.Draft to DraftDTO
func (a *App) draftToDTO(draft *ports.Draft) *DraftDTO {
	if draft == nil {
		return nil
	}

	// Parse addresses
	var to, cc, bcc []string
	if draft.ToAddresses != "" {
		to = strings.Split(draft.ToAddresses, ", ")
	}
	if draft.CcAddresses != "" {
		cc = strings.Split(draft.CcAddresses, ", ")
	}
	if draft.BccAddresses != "" {
		bcc = strings.Split(draft.BccAddresses, ", ")
	}

	var replyToID int64
	if draft.ReplyToEmailID != nil {
		replyToID = *draft.ReplyToEmailID
	}

	return &DraftDTO{
		ID:        draft.ID,
		To:        to,
		Cc:        cc,
		Bcc:       bcc,
		Subject:   draft.Subject,
		BodyHTML:  draft.BodyHTML,
		BodyText:  draft.BodyText,
		ReplyToID: replyToID,
	}
}

// generateSnippet creates a snippet from plain text
func generateSnippet(text string, maxLen int) string {
	// Clean whitespace
	var cleaned = strings.Join(strings.Fields(text), " ")
	cleaned = strings.TrimSpace(cleaned)

	if len(cleaned) <= maxLen {
		return cleaned
	}
	return cleaned[:maxLen] + "..."
}

// generateSnippetFromHTML creates a snippet from HTML content
func generateSnippetFromHTML(html string, maxLen int) string {
	// Simple HTML tag removal - could use a proper parser but this is faster
	var text = html

	// Remove style and script tags with content
	for _, tag := range []string{"style", "script"} {
		for {
			var start = strings.Index(strings.ToLower(text), "<"+tag)
			if start == -1 {
				break
			}
			var end = strings.Index(strings.ToLower(text[start:]), "</"+tag+">")
			if end == -1 {
				end = len(text) - start
			} else {
				end += len("</"+tag+">") + start
			}
			text = text[:start] + text[end:]
		}
	}

	// Remove remaining HTML tags
	var result strings.Builder
	var inTag = false
	for _, r := range text {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}

	return generateSnippet(result.String(), maxLen)
}

// UndoResult represents the result of an undo/redo operation
type UndoResult struct {
	Success     bool   `json:"success"`
	Description string `json:"description"`
	CanUndo     bool   `json:"canUndo"`
	CanRedo     bool   `json:"canRedo"`
}

// Undo undoes the last operation
func (a *App) Undo() (result UndoResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Undo] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	if a.application == nil {
		return UndoResult{Success: false, Description: "App not initialized"}, nil
	}

	var ctx = context.Background()
	var undo = a.application.Undo()

	if !undo.CanUndo(ctx) {
		return UndoResult{
			Success:     false,
			Description: "Nada para desfazer",
			CanUndo:     false,
			CanRedo:     undo.CanRedo(ctx),
		}, nil
	}

	var description = undo.GetUndoDescription(ctx)
	if undoErr := undo.Undo(ctx); undoErr != nil {
		return UndoResult{
			Success:     false,
			Description: fmt.Sprintf("Erro ao desfazer: %v", undoErr),
			CanUndo:     undo.CanUndo(ctx),
			CanRedo:     undo.CanRedo(ctx),
		}, nil
	}

	return UndoResult{
		Success:     true,
		Description: fmt.Sprintf("Desfeito: %s", description),
		CanUndo:     undo.CanUndo(ctx),
		CanRedo:     undo.CanRedo(ctx),
	}, nil
}

// Redo redoes the last undone operation
func (a *App) Redo() (result UndoResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Redo] PANIC recovered: %v", r)
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	if a.application == nil {
		return UndoResult{Success: false, Description: "App not initialized"}, nil
	}

	var ctx = context.Background()
	var undo = a.application.Undo()

	if !undo.CanRedo(ctx) {
		return UndoResult{
			Success:     false,
			Description: "Nada para refazer",
			CanUndo:     undo.CanUndo(ctx),
			CanRedo:     false,
		}, nil
	}

	var description = undo.GetRedoDescription(ctx)
	if redoErr := undo.Redo(ctx); redoErr != nil {
		return UndoResult{
			Success:     false,
			Description: fmt.Sprintf("Erro ao refazer: %v", redoErr),
			CanUndo:     undo.CanUndo(ctx),
			CanRedo:     undo.CanRedo(ctx),
		}, nil
	}

	return UndoResult{
		Success:     true,
		Description: fmt.Sprintf("Refeito: %s", description),
		CanUndo:     undo.CanUndo(ctx),
		CanRedo:     undo.CanRedo(ctx),
	}, nil
}

// CanUndo returns whether undo is available
func (a *App) CanUndo() bool {
	if a.application == nil {
		return false
	}
	return a.application.Undo().CanUndo(context.Background())
}

// CanRedo returns whether redo is available
func (a *App) CanRedo() bool {
	if a.application == nil {
		return false
	}
	return a.application.Undo().CanRedo(context.Background())
}

// ============================================================================
// CONTACTS
// ============================================================================

// SearchContacts searches for contacts by name or email
func (a *App) SearchContacts(query string, limit int) ([]ContactDTO, error) {
	if a.application == nil || a.application.Contacts() == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 20
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return nil, nil
	}

	var contacts, err = a.application.Contacts().SearchContacts(ctx, account.ID, query, limit)
	if err != nil {
		log.Printf("[SearchContacts] error: %v", err)
		return nil, err
	}

	var result []ContactDTO
	for _, c := range contacts {
		result = append(result, a.contactToDTO(&c))
	}

	return result, nil
}

// GetTopContacts returns the most frequently contacted contacts
func (a *App) GetTopContacts(limit int) ([]ContactDTO, error) {
	if a.application == nil || a.application.Contacts() == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 10
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return nil, nil
	}

	var contacts, err = a.application.Contacts().GetTopContacts(ctx, account.ID, limit)
	if err != nil {
		log.Printf("[GetTopContacts] error: %v", err)
		return nil, err
	}

	var result []ContactDTO
	for _, c := range contacts {
		result = append(result, a.contactToDTO(&c))
	}

	return result, nil
}

// SyncContacts syncs contacts from Gmail
func (a *App) SyncContacts(fullSync bool) error {
	if a.application == nil || a.application.Contacts() == nil {
		return fmt.Errorf("contacts service not available")
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return fmt.Errorf("no account selected")
	}

	log.Printf("[SyncContacts] starting sync (full=%v) for account %d", fullSync, account.ID)
	var err = a.application.Contacts().SyncContacts(ctx, account.ID, fullSync)
	if err != nil {
		log.Printf("[SyncContacts] sync failed: %v", err)
		return err
	}
	log.Printf("[SyncContacts] sync completed successfully")
	return nil
}

// GetContactSyncStatus returns the current contact sync status
func (a *App) GetContactSyncStatus() (*ContactSyncStatusDTO, error) {
	if a.application == nil || a.application.Contacts() == nil {
		return nil, nil
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return nil, nil
	}

	var status, err = a.application.Contacts().GetSyncStatus(ctx, account.ID)
	if err != nil {
		return nil, err
	}

	if status == nil {
		return &ContactSyncStatusDTO{Status: "never_synced"}, nil
	}

	var result = &ContactSyncStatusDTO{
		TotalContacts: status.TotalContacts,
		Status:        status.Status,
		Error:         status.ErrorMessage,
	}

	if status.LastFullSync != nil {
		result.LastSync = *status.LastFullSync
	} else if status.LastIncrementalSync != nil {
		result.LastSync = *status.LastIncrementalSync
	}

	return result, nil
}

// contactToDTO converts a contact to DTO
func (a *App) contactToDTO(c *ports.ContactInfo) ContactDTO {
	var dto = ContactDTO{
		ID:               c.ID,
		DisplayName:      c.DisplayName,
		GivenName:        c.GivenName,
		FamilyName:       c.FamilyName,
		PhotoURL:         c.PhotoURL,
		PhotoPath:        c.PhotoPath,
		IsStarred:        c.IsStarred,
		InteractionCount: c.InteractionCount,
	}

	for _, e := range c.Emails {
		dto.Emails = append(dto.Emails, ContactEmailDTO{
			Email:     e.Email,
			Type:      e.Type,
			IsPrimary: e.IsPrimary,
		})
	}

	for _, p := range c.Phones {
		dto.Phones = append(dto.Phones, ContactPhoneDTO{
			Phone:     p.PhoneNumber,
			Type:      p.Type,
			IsPrimary: p.IsPrimary,
		})
	}

	return dto
}

// ============================================================================
// TASKS
// ============================================================================

// GetTasks returns all tasks for the current account
func (a *App) GetTasks() ([]TaskDTO, error) {
	if a.application == nil || a.application.Tasks() == nil {
		return nil, nil
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return nil, nil
	}

	var tasks, err = a.application.Tasks().GetTasks(ctx, account.ID)
	if err != nil {
		log.Printf("[GetTasks] error: %v", err)
		return nil, err
	}

	var result []TaskDTO
	for _, t := range tasks {
		result = append(result, a.taskToDTO(&t))
	}
	return result, nil
}

// GetPendingTasks returns only incomplete tasks
func (a *App) GetPendingTasks() ([]TaskDTO, error) {
	if a.application == nil || a.application.Tasks() == nil {
		return nil, nil
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return nil, nil
	}

	var tasks, err = a.application.Tasks().GetPendingTasks(ctx, account.ID)
	if err != nil {
		log.Printf("[GetPendingTasks] error: %v", err)
		return nil, err
	}

	var result []TaskDTO
	for _, t := range tasks {
		result = append(result, a.taskToDTO(&t))
	}
	return result, nil
}

// CreateTask creates a new task
func (a *App) CreateTask(input TaskInputDTO) (*TaskDTO, error) {
	if a.application == nil || a.application.Tasks() == nil {
		return nil, fmt.Errorf("task service not available")
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return nil, fmt.Errorf("no account selected")
	}

	var source = ports.TaskSource(input.Source)
	if source == "" {
		source = ports.TaskSourceManual
	}

	var taskInput = &ports.TaskInput{
		AccountID:   account.ID,
		Title:       input.Title,
		Description: input.Description,
		IsCompleted: input.IsCompleted,
		Priority:    ports.TaskPriority(input.Priority),
		DueDate:     input.DueDate,
		EmailID:     input.EmailID,
		Source:      source,
	}

	var task, err = a.application.Tasks().CreateTask(ctx, taskInput)
	if err != nil {
		log.Printf("[CreateTask] error: %v", err)
		return nil, err
	}

	var dto = a.taskToDTO(task)
	return &dto, nil
}

// UpdateTask updates an existing task
func (a *App) UpdateTask(input TaskInputDTO) (*TaskDTO, error) {
	if a.application == nil || a.application.Tasks() == nil {
		return nil, fmt.Errorf("task service not available")
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return nil, fmt.Errorf("no account selected")
	}

	var source = ports.TaskSource(input.Source)
	if source == "" {
		source = ports.TaskSourceManual
	}

	var taskInput = &ports.TaskInput{
		ID:          input.ID,
		AccountID:   account.ID,
		Title:       input.Title,
		Description: input.Description,
		IsCompleted: input.IsCompleted,
		Priority:    ports.TaskPriority(input.Priority),
		DueDate:     input.DueDate,
		EmailID:     input.EmailID,
		Source:      source,
	}

	var task, err = a.application.Tasks().UpdateTask(ctx, taskInput)
	if err != nil {
		log.Printf("[UpdateTask] error: %v", err)
		return nil, err
	}

	var dto = a.taskToDTO(task)
	return &dto, nil
}

// ToggleTaskComplete toggles the completed status of a task
func (a *App) ToggleTaskComplete(id int64) (bool, error) {
	if a.application == nil || a.application.Tasks() == nil {
		return false, fmt.Errorf("task service not available")
	}

	var ctx = context.Background()
	return a.application.Tasks().ToggleTaskCompleted(ctx, id)
}

// DeleteTask removes a task
func (a *App) DeleteTask(id int64) error {
	if a.application == nil || a.application.Tasks() == nil {
		return fmt.Errorf("task service not available")
	}

	var ctx = context.Background()
	return a.application.Tasks().DeleteTask(ctx, id)
}

// GetTaskCounts returns task count statistics
func (a *App) GetTaskCounts() (*TaskCountsDTO, error) {
	if a.application == nil || a.application.Tasks() == nil {
		return nil, nil
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return nil, nil
	}

	var counts, err = a.application.Tasks().CountTasks(ctx, account.ID)
	if err != nil {
		log.Printf("[GetTaskCounts] error: %v", err)
		return nil, err
	}

	return &TaskCountsDTO{
		Pending:   counts.Pending,
		Completed: counts.Completed,
		Total:     counts.Total,
	}, nil
}

// taskToDTO converts ports.TaskInfo to TaskDTO
func (a *App) taskToDTO(t *ports.TaskInfo) TaskDTO {
	return TaskDTO{
		ID:          t.ID,
		AccountID:   t.AccountID,
		Title:       t.Title,
		Description: t.Description,
		IsCompleted: t.IsCompleted,
		Priority:    int(t.Priority),
		DueDate:     t.DueDate,
		EmailID:     t.EmailID,
		Source:      string(t.Source),
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

// ============================================================================
// CALENDAR
// ============================================================================

// GetCalendarEvents returns all calendar events for the current account
func (a *App) GetCalendarEvents() ([]CalendarEventDTO, error) {
	if a.application == nil || a.application.Calendar() == nil {
		return nil, nil
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return nil, nil
	}

	var events, err = a.application.Calendar().GetEvents(ctx, account.ID)
	if err != nil {
		log.Printf("[GetCalendarEvents] error: %v", err)
		return nil, err
	}

	var dtos = make([]CalendarEventDTO, len(events))
	for i, e := range events {
		dtos[i] = a.calendarEventToDTO(&e)
	}
	return dtos, nil
}

// GetCalendarEventsForWeek returns events for a specific week
func (a *App) GetCalendarEventsForWeek(weekStartDate string) ([]CalendarEventDTO, error) {
	if a.application == nil || a.application.Calendar() == nil {
		return nil, nil
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return nil, nil
	}

	// Parse week start date (format: YYYY-MM-DD)
	weekStart, err := time.Parse("2006-01-02", weekStartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	var events, fetchErr = a.application.Calendar().GetEventsForWeek(ctx, account.ID, weekStart)
	if fetchErr != nil {
		log.Printf("[GetCalendarEventsForWeek] error: %v", fetchErr)
		return nil, fetchErr
	}

	var dtos = make([]CalendarEventDTO, len(events))
	for i, e := range events {
		dtos[i] = a.calendarEventToDTO(&e)
	}
	return dtos, nil
}

// GetUpcomingCalendarEvents returns upcoming events
func (a *App) GetUpcomingCalendarEvents(limit int) ([]CalendarEventDTO, error) {
	if a.application == nil || a.application.Calendar() == nil {
		return nil, nil
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return nil, nil
	}

	var events, err = a.application.Calendar().GetUpcomingEvents(ctx, account.ID, limit)
	if err != nil {
		log.Printf("[GetUpcomingCalendarEvents] error: %v", err)
		return nil, err
	}

	var dtos = make([]CalendarEventDTO, len(events))
	for i, e := range events {
		dtos[i] = a.calendarEventToDTO(&e)
	}
	return dtos, nil
}

// CreateCalendarEvent creates a new calendar event
func (a *App) CreateCalendarEvent(input CalendarEventInputDTO) (*CalendarEventDTO, error) {
	if a.application == nil || a.application.Calendar() == nil {
		return nil, fmt.Errorf("calendar service not available")
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return nil, fmt.Errorf("no account selected")
	}

	var eventInput = &ports.CalendarEventInput{
		AccountID:   account.ID,
		Title:       input.Title,
		Description: input.Description,
		EventType:   ports.CalendarEventType(input.EventType),
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
		AllDay:      input.AllDay,
		Color:       input.Color,
		TaskID:      input.TaskID,
		EmailID:     input.EmailID,
		IsCompleted: input.IsCompleted,
		Source:      ports.CalendarEventSource(input.Source),
	}

	var event, err = a.application.Calendar().CreateEvent(ctx, eventInput)
	if err != nil {
		log.Printf("[CreateCalendarEvent] error: %v", err)
		return nil, err
	}

	var dto = a.calendarEventToDTO(event)
	return &dto, nil
}

// UpdateCalendarEvent updates an existing calendar event
func (a *App) UpdateCalendarEvent(input CalendarEventInputDTO) (*CalendarEventDTO, error) {
	if a.application == nil || a.application.Calendar() == nil {
		return nil, fmt.Errorf("calendar service not available")
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return nil, fmt.Errorf("no account selected")
	}

	var eventInput = &ports.CalendarEventInput{
		ID:          input.ID,
		AccountID:   account.ID,
		Title:       input.Title,
		Description: input.Description,
		EventType:   ports.CalendarEventType(input.EventType),
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
		AllDay:      input.AllDay,
		Color:       input.Color,
		TaskID:      input.TaskID,
		EmailID:     input.EmailID,
		IsCompleted: input.IsCompleted,
		Source:      ports.CalendarEventSource(input.Source),
	}

	var event, err = a.application.Calendar().UpdateEvent(ctx, eventInput)
	if err != nil {
		log.Printf("[UpdateCalendarEvent] error: %v", err)
		return nil, err
	}

	var dto = a.calendarEventToDTO(event)
	return &dto, nil
}

// DeleteCalendarEvent deletes a calendar event
func (a *App) DeleteCalendarEvent(id int64) error {
	if a.application == nil || a.application.Calendar() == nil {
		return fmt.Errorf("calendar service not available")
	}

	var ctx = context.Background()
	return a.application.Calendar().DeleteEvent(ctx, id)
}

// ToggleCalendarEventComplete toggles the completed status of an event
func (a *App) ToggleCalendarEventComplete(id int64) (bool, error) {
	if a.application == nil || a.application.Calendar() == nil {
		return false, fmt.Errorf("calendar service not available")
	}

	var ctx = context.Background()
	return a.application.Calendar().ToggleEventCompleted(ctx, id)
}

// GetCalendarEventCounts returns calendar event statistics
func (a *App) GetCalendarEventCounts() (*CalendarEventCountsDTO, error) {
	if a.application == nil || a.application.Calendar() == nil {
		return nil, nil
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return nil, nil
	}

	var counts, err = a.application.Calendar().CountEvents(ctx, account.ID)
	if err != nil {
		log.Printf("[GetCalendarEventCounts] error: %v", err)
		return nil, err
	}

	return &CalendarEventCountsDTO{
		Upcoming:  counts.Upcoming,
		Completed: counts.Completed,
		Total:     counts.Total,
	}, nil
}

// CreateFollowUpEvent creates a follow-up event for an email
func (a *App) CreateFollowUpEvent(emailID int64, followUpDate string, title string) (*CalendarEventDTO, error) {
	if a.application == nil || a.application.Calendar() == nil {
		return nil, fmt.Errorf("calendar service not available")
	}

	var ctx = context.Background()

	// Parse follow-up date
	followUpTime, err := time.Parse("2006-01-02T15:04:05", followUpDate)
	if err != nil {
		// Try date-only format
		followUpTime, err = time.Parse("2006-01-02", followUpDate)
		if err != nil {
			return nil, fmt.Errorf("invalid date format: %w", err)
		}
		// Set default time to 9:00 AM
		followUpTime = followUpTime.Add(9 * time.Hour)
	}

	var event, createErr = a.application.Calendar().CreateFollowUpEvent(ctx, emailID, followUpTime, title)
	if createErr != nil {
		log.Printf("[CreateFollowUpEvent] error: %v", createErr)
		return nil, createErr
	}

	var dto = a.calendarEventToDTO(event)
	return &dto, nil
}

// SyncTasksToCalendar syncs all tasks with due dates to calendar
func (a *App) SyncTasksToCalendar() error {
	if a.application == nil || a.application.Calendar() == nil {
		return fmt.Errorf("calendar service not available")
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		return fmt.Errorf("no account selected")
	}

	return a.application.Calendar().SyncTasksToCalendar(ctx, account.ID)
}

// calendarEventToDTO converts ports.CalendarEventInfo to CalendarEventDTO
func (a *App) calendarEventToDTO(e *ports.CalendarEventInfo) CalendarEventDTO {
	return CalendarEventDTO{
		ID:               e.ID,
		AccountID:        e.AccountID,
		Title:            e.Title,
		Description:      e.Description,
		EventType:        string(e.EventType),
		StartTime:        e.StartTime,
		EndTime:          e.EndTime,
		AllDay:           e.AllDay,
		Color:            e.Color,
		TaskID:           e.TaskID,
		EmailID:          e.EmailID,
		IsCompleted:      e.IsCompleted,
		Source:           string(e.Source),
		GoogleEventID:    e.GoogleEventID,
		GoogleCalendarID: e.GoogleCalendarID,
		LastSyncedAt:     e.LastSyncedAt,
		SyncStatus:       string(e.SyncStatus),
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}
}

// ============================================================================
// GOOGLE CALENDAR SYNC
// ============================================================================

// IsGoogleCalendarConnected returns true if Google Calendar is connected
func (a *App) IsGoogleCalendarConnected() bool {
	if a.application == nil {
		log.Printf("[IsGoogleCalendarConnected] application is nil")
		return false
	}
	if a.application.Calendar() == nil {
		log.Printf("[IsGoogleCalendarConnected] calendar service is nil")
		return false
	}
	var calService = a.application.Calendar().(*services.CalendarService)
	var connected = calService.IsGoogleCalendarConnected()
	log.Printf("[IsGoogleCalendarConnected] connected=%v", connected)
	return connected
}

// ListGoogleCalendars returns all Google Calendars for the user
func (a *App) ListGoogleCalendars() ([]GoogleCalendarDTO, error) {
	if a.application == nil || a.application.Calendar() == nil {
		return nil, fmt.Errorf("calendar service not available")
	}

	var ctx = context.Background()
	var calService = a.application.Calendar().(*services.CalendarService)

	calendars, err := calService.ListGoogleCalendars(ctx)
	if err != nil {
		return nil, err
	}

	var result []GoogleCalendarDTO
	for _, cal := range calendars {
		result = append(result, GoogleCalendarDTO{
			ID:              cal.ID,
			Summary:         cal.Summary,
			Description:     cal.Description,
			Primary:         cal.Primary,
			BackgroundColor: cal.BackgroundColor,
			AccessRole:      cal.AccessRole,
		})
	}

	return result, nil
}

// SyncFromGoogleCalendar syncs events from Google Calendar to local storage
func (a *App) SyncFromGoogleCalendar(calendarID string) (int, error) {
	log.Printf("[SyncFromGoogleCalendar] Starting sync for calendar: %s", calendarID)

	if a.application == nil || a.application.Calendar() == nil {
		log.Printf("[SyncFromGoogleCalendar] calendar service not available")
		return 0, fmt.Errorf("calendar service not available")
	}

	var ctx = context.Background()
	var account = a.application.GetCurrentAccount()
	if account == nil {
		log.Printf("[SyncFromGoogleCalendar] no account selected")
		return 0, fmt.Errorf("no account selected")
	}

	var calService = a.application.Calendar().(*services.CalendarService)
	count, err := calService.SyncFromGoogleCalendar(ctx, account.ID, calendarID)
	if err != nil {
		log.Printf("[SyncFromGoogleCalendar] Error: %v", err)
		return 0, err
	}
	log.Printf("[SyncFromGoogleCalendar] Synced %d events", count)
	return count, nil
}

// GetGoogleCalendarEvents returns events from Google Calendar for a week
func (a *App) GetGoogleCalendarEvents(calendarID, weekStartDate string) ([]GoogleEventDTO, error) {
	if a.application == nil || a.application.Calendar() == nil {
		return nil, fmt.Errorf("calendar service not available")
	}

	var ctx = context.Background()
	var calService = a.application.Calendar().(*services.CalendarService)

	weekStart, err := time.Parse("2006-01-02", weekStartDate)
	if err != nil {
		weekStart = time.Now()
	}

	events, err := calService.GetGoogleCalendarEvents(ctx, calendarID, weekStart)
	if err != nil {
		return nil, err
	}

	var result []GoogleEventDTO
	for _, e := range events {
		result = append(result, GoogleEventDTO{
			ID:          e.ID,
			CalendarID:  e.CalendarID,
			Summary:     e.Summary,
			Description: e.Description,
			Location:    e.Location,
			StartTime:   e.StartTime,
			EndTime:     e.EndTime,
			AllDay:      e.AllDay,
			Status:      e.Status,
			HtmlLink:    e.HtmlLink,
			ColorID:     e.ColorID,
		})
	}

	return result, nil
}
