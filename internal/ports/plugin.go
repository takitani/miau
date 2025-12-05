// Package ports defines the plugin system interfaces.
// Plugins are external data source integrations (Basecamp, Notion, Linear, etc.)
// that follow the same clean architecture principles as the core miau system.
package ports

import (
	"context"
	"time"
)

// PluginID is a unique identifier for a plugin
type PluginID string

// PluginStatus represents the current state of a plugin
type PluginStatus string

const (
	PluginStatusDisabled     PluginStatus = "disabled"
	PluginStatusEnabled      PluginStatus = "enabled"
	PluginStatusConnecting   PluginStatus = "connecting"
	PluginStatusConnected    PluginStatus = "connected"
	PluginStatusError        PluginStatus = "error"
	PluginStatusAuthRequired PluginStatus = "auth_required"
)

// PluginCapability represents what a plugin can do
type PluginCapability string

const (
	CapabilityTasks      PluginCapability = "tasks"       // To-dos, issues
	CapabilityMessages   PluginCapability = "messages"    // Discussions, comments
	CapabilityDocuments  PluginCapability = "documents"   // Files, notes
	CapabilityProjects   PluginCapability = "projects"    // Containers for items
	CapabilityCalendar   PluginCapability = "calendar"    // Events, schedules
	CapabilityPeople     PluginCapability = "people"      // Users, contacts
	CapabilitySearch     PluginCapability = "search"      // Full-text search
	CapabilityWebhooks   PluginCapability = "webhooks"    // Real-time updates
	CapabilityWrite      PluginCapability = "write"       // Create/update items
)

// PluginInfo contains metadata about a plugin
type PluginInfo struct {
	ID           PluginID           `json:"id"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	Version      string             `json:"version"`
	Author       string             `json:"author"`
	Website      string             `json:"website"`
	Icon         string             `json:"icon"`          // Path or emoji
	Capabilities []PluginCapability `json:"capabilities"`
	AuthType     PluginAuthType     `json:"auth_type"`
}

// PluginAuthType defines how the plugin authenticates
type PluginAuthType string

const (
	PluginAuthNone   PluginAuthType = "none"
	PluginAuthOAuth2 PluginAuthType = "oauth2"
	PluginAuthAPIKey PluginAuthType = "api_key"
	PluginAuthBasic  PluginAuthType = "basic"
)

// PluginOAuthConfig holds OAuth2 configuration for a plugin
type PluginOAuthConfig struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AuthURL      string   `json:"auth_url"`
	TokenURL     string   `json:"token_url"`
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes"`
}

// PluginState represents the runtime state of a plugin instance
type PluginState struct {
	PluginID     PluginID     `json:"plugin_id"`
	AccountID    int64        `json:"account_id"`
	Status       PluginStatus `json:"status"`
	Error        string       `json:"error,omitempty"`
	LastSyncAt   *time.Time   `json:"last_sync_at,omitempty"`
	ItemCount    int          `json:"item_count"`
	ExternalID   string       `json:"external_id,omitempty"`   // Account ID in external system
	ExternalName string       `json:"external_name,omitempty"` // Account name in external system
}

// Plugin is the main interface that all plugins must implement
type Plugin interface {
	// Metadata
	Info() PluginInfo

	// Lifecycle
	Initialize(ctx context.Context, config PluginConfig) error
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Status() PluginStatus

	// OAuth2 (if AuthType is oauth2)
	GetAuthURL(state string) string
	HandleAuthCallback(ctx context.Context, code string) error
	RefreshToken(ctx context.Context) error
}

// PluginConfig is passed to plugins during initialization
type PluginConfig struct {
	AccountID   int64                  `json:"account_id"`
	OAuth       *PluginOAuthConfig     `json:"oauth,omitempty"`
	APIKey      string                 `json:"api_key,omitempty"`
	Credentials map[string]string      `json:"credentials,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// ProjectProvider is implemented by plugins that support projects/workspaces
type ProjectProvider interface {
	Plugin
	ListProjects(ctx context.Context) ([]ExternalProject, error)
	GetProject(ctx context.Context, projectID string) (*ExternalProject, error)
}

// TaskProvider is implemented by plugins that support tasks/to-dos
type TaskProvider interface {
	Plugin
	ListTasks(ctx context.Context, projectID string, opts TaskListOptions) ([]ExternalTask, error)
	GetTask(ctx context.Context, taskID string) (*ExternalTask, error)
	CreateTask(ctx context.Context, task ExternalTaskCreate) (*ExternalTask, error)
	UpdateTask(ctx context.Context, taskID string, update ExternalTaskUpdate) (*ExternalTask, error)
	CompleteTask(ctx context.Context, taskID string) error
}

// TaskListOptions configures task listing
type TaskListOptions struct {
	Status      string // all, pending, completed
	AssignedTo  string // user ID filter
	DueAfter    *time.Time
	DueBefore   *time.Time
	Limit       int
	Cursor      string // for pagination
}

// MessageProvider is implemented by plugins that support messages/comments
type MessageProvider interface {
	Plugin
	ListMessages(ctx context.Context, projectID string, opts MessageListOptions) ([]ExternalMessage, error)
	GetMessage(ctx context.Context, messageID string) (*ExternalMessage, error)
	PostMessage(ctx context.Context, msg ExternalMessageCreate) (*ExternalMessage, error)
	ListComments(ctx context.Context, parentID string) ([]ExternalComment, error)
	PostComment(ctx context.Context, parentID string, content string) (*ExternalComment, error)
}

// MessageListOptions configures message listing
type MessageListOptions struct {
	Since  *time.Time
	Limit  int
	Cursor string
}

// DocumentProvider is implemented by plugins that support documents/files
type DocumentProvider interface {
	Plugin
	ListDocuments(ctx context.Context, projectID string) ([]ExternalDocument, error)
	GetDocument(ctx context.Context, docID string) (*ExternalDocument, error)
	GetDocumentContent(ctx context.Context, docID string) ([]byte, error)
}

// CalendarProvider is implemented by plugins that support calendar/schedules
type CalendarProvider interface {
	Plugin
	ListEvents(ctx context.Context, projectID string, opts CalendarListOptions) ([]ExternalEvent, error)
	GetEvent(ctx context.Context, eventID string) (*ExternalEvent, error)
}

// CalendarListOptions configures event listing
type CalendarListOptions struct {
	From  time.Time
	To    time.Time
	Limit int
}

// PeopleProvider is implemented by plugins that support user/people data
type PeopleProvider interface {
	Plugin
	ListPeople(ctx context.Context, projectID string) ([]ExternalPerson, error)
	GetPerson(ctx context.Context, personID string) (*ExternalPerson, error)
}

// SearchProvider is implemented by plugins that support search
type SearchProvider interface {
	Plugin
	Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error)
}

// SearchOptions configures search
type SearchOptions struct {
	ProjectID string
	Types     []string // task, message, document, etc.
	Limit     int
}

// SearchResult contains search results
type SearchResult struct {
	Query      string           `json:"query"`
	TotalCount int              `json:"total_count"`
	Items      []ExternalItem   `json:"items"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

// SyncProvider is implemented by plugins that support sync
type SyncProvider interface {
	Plugin
	// Sync fetches all changes since lastSync (or all items if nil)
	Sync(ctx context.Context, lastSync *time.Time) (*SyncResult, error)
}

// SyncResult contains the result of a sync operation
type SyncResult struct {
	NewItems     []ExternalItem `json:"new_items"`
	UpdatedItems []ExternalItem `json:"updated_items"`
	DeletedIDs   []string       `json:"deleted_ids"`
	SyncedAt     time.Time      `json:"synced_at"`
	HasMore      bool           `json:"has_more"`
	Cursor       string         `json:"cursor,omitempty"`
}

// PluginRegistry manages plugin registration and lifecycle
type PluginRegistry interface {
	// Registration
	Register(plugin Plugin) error
	Unregister(pluginID PluginID) error
	Get(pluginID PluginID) (Plugin, error)
	List() []PluginInfo

	// Lifecycle
	Enable(ctx context.Context, pluginID PluginID, accountID int64) error
	Disable(ctx context.Context, pluginID PluginID, accountID int64) error
	Connect(ctx context.Context, pluginID PluginID, accountID int64) error
	Disconnect(ctx context.Context, pluginID PluginID, accountID int64) error

	// State
	GetState(pluginID PluginID, accountID int64) (*PluginState, error)
	GetAllStates(accountID int64) ([]PluginState, error)

	// Events
	Subscribe(handler PluginEventHandler) func()
}

// PluginEventHandler handles plugin events
type PluginEventHandler func(event PluginEvent)

// PluginEvent represents a plugin-related event
type PluginEvent struct {
	Type      PluginEventType `json:"type"`
	PluginID  PluginID        `json:"plugin_id"`
	AccountID int64           `json:"account_id"`
	Timestamp time.Time       `json:"timestamp"`
	Data      interface{}     `json:"data,omitempty"`
	Error     string          `json:"error,omitempty"`
}

// PluginEventType identifies the type of plugin event
type PluginEventType string

const (
	PluginEventEnabled      PluginEventType = "plugin_enabled"
	PluginEventDisabled     PluginEventType = "plugin_disabled"
	PluginEventConnected    PluginEventType = "plugin_connected"
	PluginEventDisconnected PluginEventType = "plugin_disconnected"
	PluginEventAuthRequired PluginEventType = "plugin_auth_required"
	PluginEventAuthComplete PluginEventType = "plugin_auth_complete"
	PluginEventSyncStarted  PluginEventType = "plugin_sync_started"
	PluginEventSyncComplete PluginEventType = "plugin_sync_complete"
	PluginEventSyncError    PluginEventType = "plugin_sync_error"
	PluginEventItemsAdded   PluginEventType = "plugin_items_added"
	PluginEventItemsUpdated PluginEventType = "plugin_items_updated"
	PluginEventItemsDeleted PluginEventType = "plugin_items_deleted"
	PluginEventError        PluginEventType = "plugin_error"
)

// PluginStoragePort handles plugin data persistence
type PluginStoragePort interface {
	// Plugin state
	SavePluginState(ctx context.Context, state *PluginState) error
	GetPluginState(ctx context.Context, pluginID PluginID, accountID int64) (*PluginState, error)
	GetAllPluginStates(ctx context.Context, accountID int64) ([]PluginState, error)
	DeletePluginState(ctx context.Context, pluginID PluginID, accountID int64) error

	// Plugin credentials (encrypted)
	SavePluginCredentials(ctx context.Context, pluginID PluginID, accountID int64, creds map[string]string) error
	GetPluginCredentials(ctx context.Context, pluginID PluginID, accountID int64) (map[string]string, error)
	DeletePluginCredentials(ctx context.Context, pluginID PluginID, accountID int64) error

	// External items (normalized data from plugins)
	SaveExternalItems(ctx context.Context, pluginID PluginID, accountID int64, items []ExternalItem) error
	GetExternalItems(ctx context.Context, pluginID PluginID, accountID int64, opts ExternalItemQuery) ([]ExternalItem, error)
	GetExternalItem(ctx context.Context, pluginID PluginID, itemID string) (*ExternalItem, error)
	DeleteExternalItems(ctx context.Context, pluginID PluginID, accountID int64, itemIDs []string) error
	DeleteAllPluginItems(ctx context.Context, pluginID PluginID, accountID int64) error

	// Projects
	SaveExternalProjects(ctx context.Context, pluginID PluginID, accountID int64, projects []ExternalProject) error
	GetExternalProjects(ctx context.Context, pluginID PluginID, accountID int64) ([]ExternalProject, error)
}

// ExternalItemQuery options for querying external items
type ExternalItemQuery struct {
	ProjectID  string
	Types      []ExternalItemType
	Status     string // pending, completed, all
	AssignedTo string
	Since      *time.Time
	Limit      int
	Offset     int
}
