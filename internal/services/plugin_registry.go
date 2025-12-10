// Package services implements the plugin registry service.
// The registry manages plugin registration, lifecycle, and events.
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/opik/miau/internal/ports"
)

// pluginInstance holds a plugin and its runtime state
type pluginInstance struct {
	plugin    ports.Plugin
	config    ports.PluginConfig
	state     ports.PluginState
	accountID int64
}

// PluginRegistry manages plugin registration and lifecycle
type PluginRegistry struct {
	mu sync.RWMutex

	// Registered plugins (plugin ID -> plugin factory/instance)
	plugins map[ports.PluginID]ports.Plugin

	// Active instances per account (account ID -> plugin ID -> instance)
	instances map[int64]map[ports.PluginID]*pluginInstance

	// Storage for persistence
	storage ports.PluginStoragePort

	// Event handlers
	handlers []ports.PluginEventHandler
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry(storage ports.PluginStoragePort) *PluginRegistry {
	return &PluginRegistry{
		plugins:   make(map[ports.PluginID]ports.Plugin),
		instances: make(map[int64]map[ports.PluginID]*pluginInstance),
		storage:   storage,
		handlers:  make([]ports.PluginEventHandler, 0),
	}
}

// Register adds a plugin to the registry
func (r *PluginRegistry) Register(plugin ports.Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	info := plugin.Info()
	if info.ID == "" {
		return fmt.Errorf("plugin ID is required")
	}

	if _, exists := r.plugins[info.ID]; exists {
		return fmt.Errorf("plugin %s is already registered", info.ID)
	}

	r.plugins[info.ID] = plugin
	return nil
}

// Unregister removes a plugin from the registry
func (r *PluginRegistry) Unregister(pluginID ports.PluginID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[pluginID]; !exists {
		return fmt.Errorf("plugin %s is not registered", pluginID)
	}

	// Disconnect all instances before unregistering
	for accountID, accountInstances := range r.instances {
		if instance, exists := accountInstances[pluginID]; exists {
			instance.plugin.Disconnect(context.Background())
			delete(accountInstances, pluginID)
		}
		if len(accountInstances) == 0 {
			delete(r.instances, accountID)
		}
	}

	delete(r.plugins, pluginID)
	return nil
}

// Get returns a registered plugin by ID
func (r *PluginRegistry) Get(pluginID ports.PluginID) (ports.Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin %s is not registered", pluginID)
	}
	return plugin, nil
}

// List returns info about all registered plugins
func (r *PluginRegistry) List() []ports.PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]ports.PluginInfo, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		infos = append(infos, plugin.Info())
	}
	return infos
}

// Enable activates a plugin for an account
func (r *PluginRegistry) Enable(ctx context.Context, pluginID ports.PluginID, accountID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, exists := r.plugins[pluginID]
	if !exists {
		return fmt.Errorf("plugin %s is not registered", pluginID)
	}

	// Create account instance map if needed
	if r.instances[accountID] == nil {
		r.instances[accountID] = make(map[ports.PluginID]*pluginInstance)
	}

	// Check if already enabled
	if _, exists := r.instances[accountID][pluginID]; exists {
		return nil // Already enabled
	}

	// Load credentials from storage
	var config ports.PluginConfig
	config.AccountID = accountID
	if r.storage != nil {
		creds, err := r.storage.GetPluginCredentials(ctx, pluginID, accountID)
		if err == nil && creds != nil {
			config.Credentials = creds
		}
	}

	// Initialize plugin
	if err := plugin.Initialize(ctx, config); err != nil {
		return fmt.Errorf("failed to initialize plugin %s: %w", pluginID, err)
	}

	// Create instance
	instance := &pluginInstance{
		plugin:    plugin,
		config:    config,
		accountID: accountID,
		state: ports.PluginState{
			PluginID:  pluginID,
			AccountID: accountID,
			Status:    ports.PluginStatusEnabled,
		},
	}

	r.instances[accountID][pluginID] = instance

	// Save state
	if r.storage != nil {
		r.storage.SavePluginState(ctx, &instance.state)
	}

	// Emit event
	r.emitEvent(ports.PluginEvent{
		Type:      ports.PluginEventEnabled,
		PluginID:  pluginID,
		AccountID: accountID,
		Timestamp: time.Now(),
	})

	return nil
}

// Disable deactivates a plugin for an account
func (r *PluginRegistry) Disable(ctx context.Context, pluginID ports.PluginID, accountID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	accountInstances, exists := r.instances[accountID]
	if !exists {
		return nil // Not enabled
	}

	instance, exists := accountInstances[pluginID]
	if !exists {
		return nil // Not enabled
	}

	// Disconnect if connected
	if instance.state.Status == ports.PluginStatusConnected {
		instance.plugin.Disconnect(ctx)
	}

	// Update state
	instance.state.Status = ports.PluginStatusDisabled
	if r.storage != nil {
		r.storage.SavePluginState(ctx, &instance.state)
	}

	// Remove instance
	delete(accountInstances, pluginID)
	if len(accountInstances) == 0 {
		delete(r.instances, accountID)
	}

	// Emit event
	r.emitEvent(ports.PluginEvent{
		Type:      ports.PluginEventDisabled,
		PluginID:  pluginID,
		AccountID: accountID,
		Timestamp: time.Now(),
	})

	return nil
}

// Connect establishes connection for a plugin
func (r *PluginRegistry) Connect(ctx context.Context, pluginID ports.PluginID, accountID int64) error {
	r.mu.Lock()
	instance, err := r.getInstance(pluginID, accountID)
	if err != nil {
		r.mu.Unlock()
		return err
	}

	// Update status
	instance.state.Status = ports.PluginStatusConnecting
	r.mu.Unlock()

	// Connect (outside lock to avoid deadlock)
	if err := instance.plugin.Connect(ctx); err != nil {
		r.mu.Lock()
		instance.state.Status = ports.PluginStatusError
		instance.state.Error = err.Error()
		if r.storage != nil {
			r.storage.SavePluginState(ctx, &instance.state)
		}
		r.mu.Unlock()

		// Emit error event
		r.emitEvent(ports.PluginEvent{
			Type:      ports.PluginEventError,
			PluginID:  pluginID,
			AccountID: accountID,
			Timestamp: time.Now(),
			Error:     err.Error(),
		})
		return err
	}

	r.mu.Lock()
	instance.state.Status = ports.PluginStatusConnected
	instance.state.Error = ""
	if r.storage != nil {
		r.storage.SavePluginState(ctx, &instance.state)
	}
	r.mu.Unlock()

	// Emit connected event
	r.emitEvent(ports.PluginEvent{
		Type:      ports.PluginEventConnected,
		PluginID:  pluginID,
		AccountID: accountID,
		Timestamp: time.Now(),
	})

	return nil
}

// Disconnect closes connection for a plugin
func (r *PluginRegistry) Disconnect(ctx context.Context, pluginID ports.PluginID, accountID int64) error {
	r.mu.Lock()
	instance, err := r.getInstance(pluginID, accountID)
	if err != nil {
		r.mu.Unlock()
		return err
	}
	r.mu.Unlock()

	// Disconnect
	if err := instance.plugin.Disconnect(ctx); err != nil {
		return err
	}

	r.mu.Lock()
	instance.state.Status = ports.PluginStatusEnabled
	if r.storage != nil {
		r.storage.SavePluginState(ctx, &instance.state)
	}
	r.mu.Unlock()

	// Emit disconnected event
	r.emitEvent(ports.PluginEvent{
		Type:      ports.PluginEventDisconnected,
		PluginID:  pluginID,
		AccountID: accountID,
		Timestamp: time.Now(),
	})

	return nil
}

// GetState returns the current state of a plugin for an account
func (r *PluginRegistry) GetState(pluginID ports.PluginID, accountID int64) (*ports.PluginState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instance, err := r.getInstance(pluginID, accountID)
	if err != nil {
		// Try loading from storage
		if r.storage != nil {
			state, err := r.storage.GetPluginState(context.Background(), pluginID, accountID)
			if err == nil {
				return state, nil
			}
		}
		return nil, err
	}

	stateCopy := instance.state
	return &stateCopy, nil
}

// GetAllStates returns states of all enabled plugins for an account
func (r *PluginRegistry) GetAllStates(accountID int64) ([]ports.PluginState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	accountInstances, exists := r.instances[accountID]
	if !exists {
		return []ports.PluginState{}, nil
	}

	states := make([]ports.PluginState, 0, len(accountInstances))
	for _, instance := range accountInstances {
		states = append(states, instance.state)
	}
	return states, nil
}

// Subscribe adds an event handler
func (r *PluginRegistry) Subscribe(handler ports.PluginEventHandler) func() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers = append(r.handlers, handler)
	index := len(r.handlers) - 1

	// Return unsubscribe function
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		// Mark as nil instead of slice manipulation
		if index < len(r.handlers) {
			r.handlers[index] = nil
		}
	}
}

// GetPluginInstance returns the active plugin instance for an account
// Used by services to access plugin capabilities
func (r *PluginRegistry) GetPluginInstance(pluginID ports.PluginID, accountID int64) (ports.Plugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instance, err := r.getInstance(pluginID, accountID)
	if err != nil {
		return nil, err
	}
	return instance.plugin, nil
}

// GetProjectProvider returns the plugin as ProjectProvider if it supports it
func (r *PluginRegistry) GetProjectProvider(pluginID ports.PluginID, accountID int64) (ports.ProjectProvider, error) {
	plugin, err := r.GetPluginInstance(pluginID, accountID)
	if err != nil {
		return nil, err
	}

	provider, ok := plugin.(ports.ProjectProvider)
	if !ok {
		return nil, fmt.Errorf("plugin %s does not support projects", pluginID)
	}
	return provider, nil
}

// GetTaskProvider returns the plugin as TaskProvider if it supports it
func (r *PluginRegistry) GetTaskProvider(pluginID ports.PluginID, accountID int64) (ports.TaskProvider, error) {
	plugin, err := r.GetPluginInstance(pluginID, accountID)
	if err != nil {
		return nil, err
	}

	provider, ok := plugin.(ports.TaskProvider)
	if !ok {
		return nil, fmt.Errorf("plugin %s does not support tasks", pluginID)
	}
	return provider, nil
}

// GetMessageProvider returns the plugin as MessageProvider if it supports it
func (r *PluginRegistry) GetMessageProvider(pluginID ports.PluginID, accountID int64) (ports.MessageProvider, error) {
	plugin, err := r.GetPluginInstance(pluginID, accountID)
	if err != nil {
		return nil, err
	}

	provider, ok := plugin.(ports.MessageProvider)
	if !ok {
		return nil, fmt.Errorf("plugin %s does not support messages", pluginID)
	}
	return provider, nil
}

// GetSyncProvider returns the plugin as SyncProvider if it supports it
func (r *PluginRegistry) GetSyncProvider(pluginID ports.PluginID, accountID int64) (ports.SyncProvider, error) {
	plugin, err := r.GetPluginInstance(pluginID, accountID)
	if err != nil {
		return nil, err
	}

	provider, ok := plugin.(ports.SyncProvider)
	if !ok {
		return nil, fmt.Errorf("plugin %s does not support sync", pluginID)
	}
	return provider, nil
}

// UpdatePluginState updates the state of a plugin instance
func (r *PluginRegistry) UpdatePluginState(ctx context.Context, pluginID ports.PluginID, accountID int64, updater func(*ports.PluginState)) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance, err := r.getInstance(pluginID, accountID)
	if err != nil {
		return err
	}

	updater(&instance.state)

	if r.storage != nil {
		return r.storage.SavePluginState(ctx, &instance.state)
	}
	return nil
}

// SavePluginCredentials saves credentials for a plugin
func (r *PluginRegistry) SavePluginCredentials(ctx context.Context, pluginID ports.PluginID, accountID int64, creds map[string]string) error {
	if r.storage == nil {
		return fmt.Errorf("storage not configured")
	}
	return r.storage.SavePluginCredentials(ctx, pluginID, accountID, creds)
}

// getInstance returns an instance (must be called with lock held)
func (r *PluginRegistry) getInstance(pluginID ports.PluginID, accountID int64) (*pluginInstance, error) {
	accountInstances, exists := r.instances[accountID]
	if !exists {
		return nil, fmt.Errorf("no plugins enabled for account %d", accountID)
	}

	instance, exists := accountInstances[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin %s is not enabled for account %d", pluginID, accountID)
	}

	return instance, nil
}

// emitEvent sends event to all handlers
func (r *PluginRegistry) emitEvent(event ports.PluginEvent) {
	r.mu.RLock()
	handlers := make([]ports.PluginEventHandler, len(r.handlers))
	copy(handlers, r.handlers)
	r.mu.RUnlock()

	for _, handler := range handlers {
		if handler != nil {
			go handler(event)
		}
	}
}
