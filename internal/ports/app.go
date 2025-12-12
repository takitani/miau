package ports

import "context"

// App is the main application interface that UI layers use.
// It provides access to all services and manages the application lifecycle.
type App interface {
	// Lifecycle
	Start() error
	Stop() error

	// Services access
	Email() EmailService
	Send() SendService
	Draft() DraftService
	Search() SearchService
	Batch() BatchService
	Notification() NotificationService
	Sync() SyncService
	AI() AIService
	Analytics() AnalyticsService
	Attachment() AttachmentService
	Thread() ThreadService
	Undo() UndoService
	Contacts() ContactService
	Tasks() TaskService
	Calendar() CalendarService
	Basecamp() BasecampService
	Plugins() PluginService
	Snooze() SnoozeService
	Schedule() ScheduleService

	// Events
	Events() EventBus

	// Account
	GetCurrentAccount() *AccountInfo
	GetAllAccounts() []AccountInfo
	SetCurrentAccount(email string) error

	// Config
	GetConfig() AppConfig

	// SetIMAPClient sets an external IMAP client (for TUI to share connection)
	SetIMAPClient(client interface{})

	// SyncThreadIDsFromGmail syncs thread IDs from Gmail API for existing emails
	SyncThreadIDsFromGmail(ctx context.Context, progressCallback func(processed, total int)) (int, error)
}

// PluginService provides high-level plugin operations
type PluginService interface {
	// Plugin listing
	ListPlugins(ctx context.Context) ([]PluginWithState, error)

	// Lifecycle
	EnablePlugin(ctx context.Context, pluginID PluginID) error
	DisablePlugin(ctx context.Context, pluginID PluginID) error
	ConnectPlugin(ctx context.Context, pluginID PluginID) error
	DisconnectPlugin(ctx context.Context, pluginID PluginID) error

	// State
	GetPluginState(ctx context.Context, pluginID PluginID) (*PluginState, error)

	// OAuth2
	GetAuthURL(ctx context.Context, pluginID PluginID, state string) (string, error)
	HandleAuthCallback(ctx context.Context, pluginID PluginID, code string) error

	// Data access
	ListProjects(ctx context.Context, pluginID PluginID) ([]ExternalProject, error)
	ListTasks(ctx context.Context, pluginID PluginID, projectID string, opts TaskListOptions) ([]ExternalTask, error)
	ListMessages(ctx context.Context, pluginID PluginID, projectID string, opts MessageListOptions) ([]ExternalMessage, error)
	GetExternalItems(ctx context.Context, pluginID PluginID, query ExternalItemQuery) ([]ExternalItem, error)

	// Sync
	SyncPlugin(ctx context.Context, pluginID PluginID) (*PluginSyncResult, error)

	// Task operations
	CreateTask(ctx context.Context, pluginID PluginID, task ExternalTaskCreate) (*ExternalTask, error)
	CompleteTask(ctx context.Context, pluginID PluginID, taskID string) error
}

// PluginWithState combines plugin info with its current state
type PluginWithState struct {
	Info  PluginInfo   `json:"info"`
	State *PluginState `json:"state,omitempty"`
}

// AppConfig contains application configuration
type AppConfig struct {
	AccountEmail   string
	AccountName    string
	IMAPHost       string
	IMAPPort       int
	SMTPHost       string
	SMTPPort       int
	AuthType       AuthType
	SendMethod     SendMethod
	DebugMode      bool
	DataPath       string
	TokenPath      string
}

// AuthType defines the authentication method
type AuthType string

const (
	AuthTypePassword AuthType = "password"
	AuthTypeOAuth2   AuthType = "oauth2"
)

// SendMethod defines how emails are sent
type SendMethod string

const (
	SendMethodSMTP     SendMethod = "smtp"
	SendMethodGmailAPI SendMethod = "gmail_api"
)
