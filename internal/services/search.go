package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/opik/miau/internal/ports"
)

// SearchService implements ports.SearchService
type SearchService struct {
	mu      sync.RWMutex
	storage ports.StoragePort
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

// SetAccount sets the current account
func (s *SearchService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// Search performs a full-text search on emails
func (s *SearchService) Search(ctx context.Context, query string, limit int) (*ports.SearchResult, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	var emails, err = s.storage.SearchEmails(ctx, account.ID, query, limit)
	if err != nil {
		return nil, err
	}

	return &ports.SearchResult{
		Emails:     emails,
		TotalCount: len(emails),
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
