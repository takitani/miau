package ports

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

	// Events
	Events() EventBus

	// Account
	GetCurrentAccount() *AccountInfo
	SetCurrentAccount(email string) error

	// Config
	GetConfig() AppConfig
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
