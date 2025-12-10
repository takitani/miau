package services

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/opik/miau/internal/ports"
)

// SearchService implements ports.SearchService
type SearchService struct {
	mu      sync.RWMutex
	storage ports.StoragePort
	imap    ports.IMAPPort
	events  ports.EventBus
	account *ports.AccountInfo
}

// NewSearchService creates a new SearchService
func NewSearchService(storage ports.StoragePort, events ports.EventBus) *SearchService {
	return &SearchService{
		storage: storage,
		events:  events,
	}
}

// SetIMAP sets the IMAP port for server-side search
func (s *SearchService) SetIMAP(imap ports.IMAPPort) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.imap = imap
}

// SetAccount sets the current account
func (s *SearchService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// Search performs a hybrid search: local DB + IMAP server-side
// This combines fast local results with full-text server search
func (s *SearchService) Search(ctx context.Context, query string, limit int) (*ports.SearchResult, error) {
	s.mu.RLock()
	var account = s.account
	var imapClient = s.imap
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	// 1. Local search (fast, but limited to indexed/downloaded content)
	var localEmails, err = s.storage.SearchEmails(ctx, account.ID, query, limit)
	if err != nil {
		return nil, err
	}

	// Track local IDs to avoid duplicates
	var localIDs = make(map[int64]bool)
	for _, e := range localEmails {
		localIDs[e.ID] = true
	}

	// 2. IMAP server-side search (slower, but searches full body)
	if imapClient != nil && imapClient.IsConnected() {
		var imapUIDs, imapErr = imapClient.SearchText(ctx, query, limit*2) // Get more to compensate for duplicates
		if imapErr != nil {
			log.Printf("[search] IMAP search error (continuing with local): %v", imapErr)
		} else if len(imapUIDs) > 0 {
			log.Printf("[search] IMAP found %d UIDs, local had %d results", len(imapUIDs), len(localEmails))

			// Find UIDs we don't have in local results
			var localUIDs = make(map[uint32]bool)
			for _, e := range localEmails {
				localUIDs[e.UID] = true
			}

			var newUIDs []uint32
			for _, uid := range imapUIDs {
				if !localUIDs[uid] {
					newUIDs = append(newUIDs, uid)
				}
			}

			if len(newUIDs) > 0 {
				log.Printf("[search] Fetching %d new emails from IMAP", len(newUIDs))

				// Separate UIDs into those we have locally vs need to fetch
				var uidsToFetch []uint32
				for _, uid := range newUIDs {
					var email, err = s.storage.GetEmailByUIDGlobal(ctx, account.ID, uid)
					if err == nil && email != nil {
						// We have it locally, just wasn't in search results
						if !localIDs[email.ID] {
							localEmails = append(localEmails, ports.EmailMetadata{
								ID:             email.ID,
								UID:            uint32(email.UID),
								MessageID:      email.MessageID,
								Subject:        email.Subject,
								FromName:       email.FromName,
								FromEmail:      email.FromEmail,
								Date:           email.Date,
								IsRead:         email.IsRead,
								IsStarred:      email.IsStarred,
								IsReplied:      email.IsReplied,
								HasAttachments: email.HasAttachments,
								Snippet:        email.Snippet,
								ThreadID:       email.ThreadID,
								ThreadCount:    email.ThreadCount,
							})
							localIDs[email.ID] = true
						}
					} else {
						// Email not in DB - need to fetch from IMAP
						uidsToFetch = append(uidsToFetch, uid)
					}
				}

				// Fetch missing emails from IMAP in batch
				if len(uidsToFetch) > 0 {
					log.Printf("[search] Fetching %d emails from IMAP (not in local DB)", len(uidsToFetch))
					var imapEmails, fetchErr = imapClient.FetchEmailsBatch(ctx, uidsToFetch)
					if fetchErr != nil {
						log.Printf("[search] Error fetching from IMAP: %v", fetchErr)
					} else {
						// Get INBOX folder ID for saving (search results are typically from INBOX)
						var inboxFolder, folderErr = s.storage.GetFolderByName(ctx, account.ID, "INBOX")
						var folderID int64 = 0
						if folderErr == nil && inboxFolder != nil {
							folderID = inboxFolder.ID
						}

						for _, ie := range imapEmails {
							var emailID int64 = -int64(ie.UID) // Default: negative = not saved

							// Save to DB for future searches (progressive sync)
							if folderID > 0 {
								var emailContent = &ports.EmailContent{
									EmailMetadata: ports.EmailMetadata{
										UID:            ie.UID,
										MessageID:      ie.MessageID,
										Subject:        ie.Subject,
										FromName:       ie.FromName,
										FromEmail:      ie.FromEmail,
										Date:           ie.Date,
										IsRead:         ie.Seen,
										IsStarred:      ie.Flagged,
										HasAttachments: ie.HasAttachments,
									},
								}
								var savedID, _, saveErr = s.storage.UpsertEmail(ctx, account.ID, folderID, emailContent)
								if saveErr == nil {
									emailID = savedID
									log.Printf("[search] Saved email UID %d to DB (ID: %d)", ie.UID, savedID)
								}
							}

							// Add to results
							localEmails = append(localEmails, ports.EmailMetadata{
								ID:             emailID,
								UID:            ie.UID,
								MessageID:      ie.MessageID,
								Subject:        ie.Subject,
								FromName:       ie.FromName,
								FromEmail:      ie.FromEmail,
								Date:           ie.Date,
								IsRead:         ie.Seen,
								IsStarred:      ie.Flagged,
								HasAttachments: ie.HasAttachments,
								Snippet:        "",
							})
						}
						log.Printf("[search] Added %d emails from IMAP to results", len(imapEmails))
					}
				}
			}
		}
	}

	// Sort by date (most recent first)
	sort.Slice(localEmails, func(i, j int) bool {
		return localEmails[i].Date.After(localEmails[j].Date)
	})

	// Limit results
	if len(localEmails) > limit {
		localEmails = localEmails[:limit]
	}

	return &ports.SearchResult{
		Emails:     localEmails,
		TotalCount: len(localEmails),
		Query:      query,
	}, nil
}

// SearchInFolder searches within a specific folder
func (s *SearchService) SearchInFolder(ctx context.Context, folder, query string, limit int) (*ports.SearchResult, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	var f, err = s.storage.GetFolderByName(ctx, account.ID, folder)
	if err != nil {
		return nil, err
	}

	var emails, err2 = s.storage.SearchEmailsInFolder(ctx, f.ID, query, limit)
	if err2 != nil {
		return nil, err2
	}

	return &ports.SearchResult{
		Emails:     emails,
		TotalCount: len(emails),
		Query:      query,
	}, nil
}

// GetIndexState returns the current indexing state
func (s *SearchService) GetIndexState(ctx context.Context) (*ports.IndexState, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	return s.storage.GetIndexState(ctx, account.ID)
}

// StartIndexing starts/resumes background indexing
func (s *SearchService) StartIndexing(ctx context.Context) error {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return fmt.Errorf("no account set")
	}

	var state, err = s.storage.GetIndexState(ctx, account.ID)
	if err != nil {
		state = &ports.IndexState{
			Status: ports.IndexStatusIdle,
		}
	}

	state.Status = ports.IndexStatusRunning
	if err := s.storage.UpdateIndexState(ctx, account.ID, state); err != nil {
		return err
	}

	s.events.Publish(ports.BaseEvent{
		EventType: ports.EventTypeIndexStarted,
		Time:      state.StartedAt.UTC(),
	})

	return nil
}

// PauseIndexing pauses background indexing
func (s *SearchService) PauseIndexing(ctx context.Context) error {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return fmt.Errorf("no account set")
	}

	var state, err = s.storage.GetIndexState(ctx, account.ID)
	if err != nil {
		return err
	}

	state.Status = ports.IndexStatusPaused

	return s.storage.UpdateIndexState(ctx, account.ID, state)
}

// IndexEmail indexes a single email's content
func (s *SearchService) IndexEmail(ctx context.Context, emailID int64, content string) error {
	// The FTS5 indexing is handled automatically by SQLite triggers
	// This method is for explicit indexing if needed
	return nil
}
