// Package services implements the plugin service.
// PluginService is the main entry point for plugin operations,
// following the REGRA DE OURO pattern.
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/opik/miau/internal/ports"
)

// PluginService provides high-level plugin operations
type PluginService struct {
	mu sync.RWMutex

	registry *PluginRegistry
	storage  ports.PluginStoragePort
	events   ports.EventBus
	account  *ports.AccountInfo
}

// NewPluginService creates a new plugin service
func NewPluginService(registry *PluginRegistry, storage ports.PluginStoragePort, events ports.EventBus) *PluginService {
	return &PluginService{
		registry: registry,
		storage:  storage,
		events:   events,
	}
}

// SetAccount sets the current account context
func (s *PluginService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// ListPlugins returns all registered plugins with their state
func (s *PluginService) ListPlugins(ctx context.Context) ([]ports.PluginWithState, error) {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	plugins := s.registry.List()
	result := make([]ports.PluginWithState, len(plugins))

	for i, info := range plugins {
		state, _ := s.registry.GetState(info.ID, account.ID)
		result[i] = ports.PluginWithState{
			Info:  info,
			State: state,
		}
	}

	return result, nil
}

// EnablePlugin enables a plugin for the current account
func (s *PluginService) EnablePlugin(ctx context.Context, pluginID ports.PluginID) error {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return fmt.Errorf("no account set")
	}

	return s.registry.Enable(ctx, pluginID, account.ID)
}

// DisablePlugin disables a plugin for the current account
func (s *PluginService) DisablePlugin(ctx context.Context, pluginID ports.PluginID) error {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return fmt.Errorf("no account set")
	}

	return s.registry.Disable(ctx, pluginID, account.ID)
}

// ConnectPlugin establishes connection for a plugin
func (s *PluginService) ConnectPlugin(ctx context.Context, pluginID ports.PluginID) error {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return fmt.Errorf("no account set")
	}

	return s.registry.Connect(ctx, pluginID, account.ID)
}

// DisconnectPlugin closes connection for a plugin
func (s *PluginService) DisconnectPlugin(ctx context.Context, pluginID ports.PluginID) error {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return fmt.Errorf("no account set")
	}

	return s.registry.Disconnect(ctx, pluginID, account.ID)
}

// GetPluginState returns the state of a plugin
func (s *PluginService) GetPluginState(ctx context.Context, pluginID ports.PluginID) (*ports.PluginState, error) {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	return s.registry.GetState(pluginID, account.ID)
}

// GetAuthURL returns the OAuth authorization URL for a plugin
func (s *PluginService) GetAuthURL(ctx context.Context, pluginID ports.PluginID, state string) (string, error) {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return "", fmt.Errorf("no account set")
	}

	plugin, err := s.registry.GetPluginInstance(pluginID, account.ID)
	if err != nil {
		return "", err
	}

	return plugin.GetAuthURL(state), nil
}

// HandleAuthCallback processes the OAuth callback
func (s *PluginService) HandleAuthCallback(ctx context.Context, pluginID ports.PluginID, code string) error {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return fmt.Errorf("no account set")
	}

	plugin, err := s.registry.GetPluginInstance(pluginID, account.ID)
	if err != nil {
		return err
	}

	return plugin.HandleAuthCallback(ctx, code)
}

// ListProjects returns projects from a plugin
func (s *PluginService) ListProjects(ctx context.Context, pluginID ports.PluginID) ([]ports.ExternalProject, error) {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	provider, err := s.registry.GetProjectProvider(pluginID, account.ID)
	if err != nil {
		return nil, err
	}

	projects, err := provider.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	// Save to storage
	if s.storage != nil {
		s.storage.SaveExternalProjects(ctx, pluginID, account.ID, projects)
	}

	return projects, nil
}

// GetProject returns a specific project
func (s *PluginService) GetProject(ctx context.Context, pluginID ports.PluginID, projectID string) (*ports.ExternalProject, error) {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	provider, err := s.registry.GetProjectProvider(pluginID, account.ID)
	if err != nil {
		return nil, err
	}

	return provider.GetProject(ctx, projectID)
}

// ListTasks returns tasks from a plugin
func (s *PluginService) ListTasks(ctx context.Context, pluginID ports.PluginID, projectID string, opts ports.TaskListOptions) ([]ports.ExternalTask, error) {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	provider, err := s.registry.GetTaskProvider(pluginID, account.ID)
	if err != nil {
		return nil, err
	}

	tasks, err := provider.ListTasks(ctx, projectID, opts)
	if err != nil {
		return nil, err
	}

	// Save to storage as ExternalItems
	if s.storage != nil {
		items := make([]ports.ExternalItem, len(tasks))
		for i, t := range tasks {
			items[i] = t.ToExternalItem()
		}
		s.storage.SaveExternalItems(ctx, pluginID, account.ID, items)
	}

	return tasks, nil
}

// GetTask returns a specific task
func (s *PluginService) GetTask(ctx context.Context, pluginID ports.PluginID, taskID string) (*ports.ExternalTask, error) {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	provider, err := s.registry.GetTaskProvider(pluginID, account.ID)
	if err != nil {
		return nil, err
	}

	return provider.GetTask(ctx, taskID)
}

// CreateTask creates a new task
func (s *PluginService) CreateTask(ctx context.Context, pluginID ports.PluginID, task ports.ExternalTaskCreate) (*ports.ExternalTask, error) {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	provider, err := s.registry.GetTaskProvider(pluginID, account.ID)
	if err != nil {
		return nil, err
	}

	created, err := provider.CreateTask(ctx, task)
	if err != nil {
		return nil, err
	}

	// Save to storage
	if s.storage != nil && created != nil {
		items := []ports.ExternalItem{created.ToExternalItem()}
		s.storage.SaveExternalItems(ctx, pluginID, account.ID, items)
	}

	return created, nil
}

// CompleteTask marks a task as complete
func (s *PluginService) CompleteTask(ctx context.Context, pluginID ports.PluginID, taskID string) error {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return fmt.Errorf("no account set")
	}

	provider, err := s.registry.GetTaskProvider(pluginID, account.ID)
	if err != nil {
		return err
	}

	return provider.CompleteTask(ctx, taskID)
}

// ListMessages returns messages from a plugin
func (s *PluginService) ListMessages(ctx context.Context, pluginID ports.PluginID, projectID string, opts ports.MessageListOptions) ([]ports.ExternalMessage, error) {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	provider, err := s.registry.GetMessageProvider(pluginID, account.ID)
	if err != nil {
		return nil, err
	}

	messages, err := provider.ListMessages(ctx, projectID, opts)
	if err != nil {
		return nil, err
	}

	// Save to storage as ExternalItems
	if s.storage != nil {
		items := make([]ports.ExternalItem, len(messages))
		for i, m := range messages {
			items[i] = m.ToExternalItem()
		}
		s.storage.SaveExternalItems(ctx, pluginID, account.ID, items)
	}

	return messages, nil
}

// SyncPlugin performs a sync operation for a plugin
func (s *PluginService) SyncPlugin(ctx context.Context, pluginID ports.PluginID) (*ports.PluginSyncResult, error) {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	provider, err := s.registry.GetSyncProvider(pluginID, account.ID)
	if err != nil {
		return nil, err
	}

	// Get last sync time from state
	state, _ := s.registry.GetState(pluginID, account.ID)
	var lastSync *time.Time
	if state != nil && state.LastSyncAt != nil {
		lastSync = state.LastSyncAt
	}

	// Emit sync started event
	if s.events != nil {
		s.events.Publish(&pluginSyncStartedEvent{
			pluginID:  pluginID,
			accountID: account.ID,
			ts:        time.Now(),
		})
	}

	result, err := provider.Sync(ctx, lastSync)
	if err != nil {
		// Emit sync error event
		if s.events != nil {
			s.events.Publish(&pluginSyncErrorEvent{
				pluginID:  pluginID,
				accountID: account.ID,
				err:       err,
				ts:        time.Now(),
			})
		}
		return nil, err
	}

	// Save items to storage
	if s.storage != nil {
		if len(result.NewItems) > 0 || len(result.UpdatedItems) > 0 {
			allItems := append(result.NewItems, result.UpdatedItems...)
			s.storage.SaveExternalItems(ctx, pluginID, account.ID, allItems)
		}
		if len(result.DeletedIDs) > 0 {
			s.storage.DeleteExternalItems(ctx, pluginID, account.ID, result.DeletedIDs)
		}
	}

	// Update state with sync time
	s.registry.UpdatePluginState(ctx, pluginID, account.ID, func(state *ports.PluginState) {
		state.LastSyncAt = &result.SyncedAt
		state.ItemCount += len(result.NewItems)
	})

	// Emit sync complete event
	if s.events != nil {
		s.events.Publish(&pluginSyncCompleteEvent{
			pluginID:  pluginID,
			accountID: account.ID,
			result:    result,
			ts:        time.Now(),
		})
	}

	return result, nil
}

// GetExternalItems returns stored external items
func (s *PluginService) GetExternalItems(ctx context.Context, pluginID ports.PluginID, query ports.ExternalItemQuery) ([]ports.ExternalItem, error) {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	if s.storage == nil {
		return nil, fmt.Errorf("storage not configured")
	}

	return s.storage.GetExternalItems(ctx, pluginID, account.ID, query)
}

// SearchItems searches across all plugin items
func (s *PluginService) SearchItems(ctx context.Context, query string, opts ports.SearchOptions) ([]ports.ExternalItem, error) {
	s.mu.RLock()
	account := s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	// Get all enabled plugin states
	states, err := s.registry.GetAllStates(account.ID)
	if err != nil {
		return nil, err
	}

	var allResults []ports.ExternalItem

	// Search in each connected plugin
	for _, state := range states {
		if state.Status != ports.PluginStatusConnected {
			continue
		}

		// Try SearchProvider first
		plugin, err := s.registry.GetPluginInstance(state.PluginID, account.ID)
		if err != nil {
			continue
		}

		if searchProvider, ok := plugin.(ports.SearchProvider); ok {
			result, err := searchProvider.Search(ctx, query, opts)
			if err == nil && result != nil {
				allResults = append(allResults, result.Items...)
			}
		}
	}

	return allResults, nil
}

// Plugin event types for the event bus
type pluginSyncStartedEvent struct {
	pluginID  ports.PluginID
	accountID int64
	ts        time.Time
}

func (e *pluginSyncStartedEvent) Type() ports.EventType {
	return "plugin_sync_started"
}

func (e *pluginSyncStartedEvent) Timestamp() time.Time {
	return e.ts
}

type pluginSyncCompleteEvent struct {
	pluginID  ports.PluginID
	accountID int64
	result    *ports.PluginSyncResult
	ts        time.Time
}

func (e *pluginSyncCompleteEvent) Type() ports.EventType {
	return "plugin_sync_complete"
}

func (e *pluginSyncCompleteEvent) Timestamp() time.Time {
	return e.ts
}

type pluginSyncErrorEvent struct {
	pluginID  ports.PluginID
	accountID int64
	err       error
	ts        time.Time
}

func (e *pluginSyncErrorEvent) Type() ports.EventType {
	return "plugin_sync_error"
}

func (e *pluginSyncErrorEvent) Timestamp() time.Time {
	return e.ts
}
