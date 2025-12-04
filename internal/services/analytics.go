// Package services provides business logic implementations.
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/opik/miau/internal/ports"
)

// AnalyticsService implements ports.AnalyticsService
type AnalyticsService struct {
	mu      sync.RWMutex
	storage ports.StoragePort
	events  ports.EventBus
	account *ports.AccountInfo
}

// NewAnalyticsService creates a new AnalyticsService
func NewAnalyticsService(storage ports.StoragePort, events ports.EventBus) *AnalyticsService {
	return &AnalyticsService{
		storage: storage,
		events:  events,
	}
}

// SetAccount sets the current account
func (s *AnalyticsService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// GetAnalytics returns comprehensive analytics for a time period
func (s *AnalyticsService) GetAnalytics(ctx context.Context, period string) (*ports.AnalyticsResult, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	var sinceDays = periodToDays(period)

	// Get all analytics data in parallel
	var overview, err = s.storage.GetAnalyticsOverview(ctx, account.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get overview: %w", err)
	}

	var topSenders, _ = s.storage.GetTopSenders(ctx, account.ID, 10, sinceDays)
	var hourly, _ = s.storage.GetEmailCountByHour(ctx, account.ID, sinceDays)
	var daily, _ = s.storage.GetEmailCountByDay(ctx, account.ID, sinceDays)
	var weekday, _ = s.storage.GetEmailCountByWeekday(ctx, account.ID, sinceDays)
	var responseStats, _ = s.storage.GetResponseStats(ctx, account.ID)

	// Calculate percentages for top senders
	var totalFromSenders = 0
	for _, sender := range topSenders {
		totalFromSenders += sender.Count
	}
	for i := range topSenders {
		if totalFromSenders > 0 {
			topSenders[i].Percentage = float64(topSenders[i].Count) / float64(totalFromSenders) * 100
		}
	}

	return &ports.AnalyticsResult{
		Overview:   *overview,
		TopSenders: topSenders,
		Trends: ports.EmailTrends{
			Daily:   daily,
			Hourly:  hourly,
			Weekday: weekday,
		},
		ResponseTime: *responseStats,
		Period:       period,
		GeneratedAt:  time.Now(),
	}, nil
}

// GetOverview returns basic email statistics
func (s *AnalyticsService) GetOverview(ctx context.Context) (*ports.AnalyticsOverview, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	return s.storage.GetAnalyticsOverview(ctx, account.ID)
}

// GetTopSenders returns top email senders
func (s *AnalyticsService) GetTopSenders(ctx context.Context, limit int, period string) ([]ports.SenderStats, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	if limit <= 0 {
		limit = 10
	}

	var sinceDays = periodToDays(period)
	var senders, err = s.storage.GetTopSenders(ctx, account.ID, limit, sinceDays)
	if err != nil {
		return nil, err
	}

	// Calculate percentages
	var total = 0
	for _, sender := range senders {
		total += sender.Count
	}
	for i := range senders {
		if total > 0 {
			senders[i].Percentage = float64(senders[i].Count) / float64(total) * 100
		}
	}

	return senders, nil
}

// GetEmailTrends returns email volume trends
func (s *AnalyticsService) GetEmailTrends(ctx context.Context, period string) (*ports.EmailTrends, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	var sinceDays = periodToDays(period)

	var hourly, _ = s.storage.GetEmailCountByHour(ctx, account.ID, sinceDays)
	var daily, _ = s.storage.GetEmailCountByDay(ctx, account.ID, sinceDays)
	var weekday, _ = s.storage.GetEmailCountByWeekday(ctx, account.ID, sinceDays)

	return &ports.EmailTrends{
		Daily:   daily,
		Hourly:  hourly,
		Weekday: weekday,
	}, nil
}

// GetResponseStats returns response time statistics
func (s *AnalyticsService) GetResponseStats(ctx context.Context) (*ports.ResponseTimeStats, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	return s.storage.GetResponseStats(ctx, account.ID)
}

// periodToDays converts a period string to number of days
func periodToDays(period string) int {
	switch period {
	case "7d":
		return 7
	case "30d":
		return 30
	case "90d":
		return 90
	case "all":
		return 0 // 0 means no limit
	default:
		return 30 // default to 30 days
	}
}
