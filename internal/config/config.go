package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type AuthType string

const (
	AuthTypePassword AuthType = "password"
	AuthTypeOAuth2   AuthType = "oauth2"
)

type SendMethod string

const (
	SendMethodSMTP     SendMethod = "smtp"
	SendMethodGmailAPI SendMethod = "gmail_api"
)

type OAuth2Config struct {
	ClientID     string `yaml:"client_id" mapstructure:"client_id"`
	ClientSecret string `yaml:"client_secret" mapstructure:"client_secret"`
}

type ImapConfig struct {
	Host string `yaml:"host" mapstructure:"host"`
	Port int    `yaml:"port" mapstructure:"port"`
	TLS  bool   `yaml:"tls" mapstructure:"tls"`
}

type SMTPConfig struct {
	Host string `yaml:"host" mapstructure:"host"`
	Port int    `yaml:"port" mapstructure:"port"`
}

type SignatureConfig struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"`
	HTML    string `yaml:"html" mapstructure:"html"`
	Text    string `yaml:"text" mapstructure:"text"`
}

type Account struct {
	Name       string           `yaml:"name" mapstructure:"name"`
	Email      string           `yaml:"email" mapstructure:"email"`
	AuthType   AuthType         `yaml:"auth_type" mapstructure:"auth_type"`
	Password   string           `yaml:"password,omitempty" mapstructure:"password"`
	OAuth2     *OAuth2Config    `yaml:"oauth2,omitempty" mapstructure:"oauth2"`
	IMAP       ImapConfig       `yaml:"imap" mapstructure:"imap"`
	SMTP       SMTPConfig       `yaml:"smtp,omitempty" mapstructure:"smtp"`
	SendMethod SendMethod       `yaml:"send_method,omitempty" mapstructure:"send_method"`
	Signature  *SignatureConfig `yaml:"signature,omitempty" mapstructure:"signature"`
}

type StorageConfig struct {
	Path     string `yaml:"path" mapstructure:"path"`
	Database string `yaml:"database" mapstructure:"database"`
}

type SyncConfig struct {
	Interval    string `yaml:"interval" mapstructure:"interval"`
	InitialDays int    `yaml:"initial_days" mapstructure:"initial_days"`
}

type UIConfig struct {
	Theme       string `yaml:"theme" mapstructure:"theme"`
	ShowPreview bool   `yaml:"show_preview" mapstructure:"show_preview"`
	PageSize    int    `yaml:"page_size" mapstructure:"page_size"`
	Debug       bool   `yaml:"debug" mapstructure:"debug"`
}

type ComposeConfig struct {
	Format           string `yaml:"format" mapstructure:"format"`                       // "html" ou "plain"
	SendDelaySeconds int    `yaml:"send_delay_seconds" mapstructure:"send_delay_seconds"` // 0-60, default 30
}

// ScheduleConfig holds scheduled send settings
type ScheduleConfig struct {
	Enabled          bool   `yaml:"enabled" mapstructure:"enabled"`                     // Enable scheduled send feature
	CheckInterval    string `yaml:"check_interval" mapstructure:"check_interval"`       // How often to check for due emails (default: "1m")
	DefaultMorning   string `yaml:"default_morning" mapstructure:"default_morning"`     // Default morning time (default: "09:00")
	DefaultAfternoon string `yaml:"default_afternoon" mapstructure:"default_afternoon"` // Default afternoon time (default: "14:00")
	NotifyOnSend     bool   `yaml:"notify_on_send" mapstructure:"notify_on_send"`       // Notify when scheduled email is sent
	NotifyOnFail     bool   `yaml:"notify_on_fail" mapstructure:"notify_on_fail"`       // Notify when scheduled email fails
}

// BasecampConfig holds Basecamp API integration settings
type BasecampConfig struct {
	Enabled      bool   `yaml:"enabled" mapstructure:"enabled"`
	ClientID     string `yaml:"client_id" mapstructure:"client_id"`
	ClientSecret string `yaml:"client_secret" mapstructure:"client_secret"`
	AccountID    string `yaml:"account_id" mapstructure:"account_id"` // Basecamp account ID (number)
}

type Config struct {
	Accounts []Account        `yaml:"accounts" mapstructure:"accounts"`
	Storage  StorageConfig    `yaml:"storage" mapstructure:"storage"`
	Sync     SyncConfig       `yaml:"sync" mapstructure:"sync"`
	UI       UIConfig         `yaml:"ui" mapstructure:"ui"`
	Compose  ComposeConfig    `yaml:"compose" mapstructure:"compose"`
	Schedule ScheduleConfig   `yaml:"schedule" mapstructure:"schedule"`
	Basecamp *BasecampConfig  `yaml:"basecamp,omitempty" mapstructure:"basecamp"`
}

var cfg *Config

func GetConfigPath() string {
	var home, _ = os.UserHomeDir()
	return filepath.Join(home, ".config", "miau")
}

func GetConfigFile() string {
	return filepath.Join(GetConfigPath(), "config.yaml")
}

func ConfigExists() bool {
	var _, err = os.Stat(GetConfigFile())
	return err == nil
}

func Load() (*Config, error) {
	// Reset para forçar recarregar
	cfg = nil

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(GetConfigPath())
	viper.AddConfigPath(".")

	// Defaults
	viper.SetDefault("storage.path", filepath.Join(GetConfigPath(), "data"))
	viper.SetDefault("storage.database", filepath.Join(GetConfigPath(), "data", "miau.db"))
	viper.SetDefault("sync.interval", "5m")
	viper.SetDefault("sync.initial_days", 30)
	viper.SetDefault("ui.theme", "dark")
	viper.SetDefault("ui.show_preview", true)
	viper.SetDefault("ui.page_size", 50)
	viper.SetDefault("ui.debug", false)
	viper.SetDefault("compose.format", "html")
	viper.SetDefault("compose.send_delay_seconds", 30)
	viper.SetDefault("schedule.enabled", true)
	viper.SetDefault("schedule.check_interval", "1m")
	viper.SetDefault("schedule.default_morning", "09:00")
	viper.SetDefault("schedule.default_afternoon", "14:00")
	viper.SetDefault("schedule.notify_on_send", true)
	viper.SetDefault("schedule.notify_on_fail", true)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, nil // Config não existe ainda
		}
		return nil, err
	}

	cfg = &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func Save(c *Config) error {
	var configPath = GetConfigPath()
	if err := os.MkdirAll(configPath, 0700); err != nil {
		return err
	}

	var data, err = yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(GetConfigFile(), data, 0600)
}

func DefaultConfig() *Config {
	return &Config{
		Accounts: []Account{},
		Storage: StorageConfig{
			Path:     filepath.Join(GetConfigPath(), "data"),
			Database: filepath.Join(GetConfigPath(), "data", "miau.db"),
		},
		Sync: SyncConfig{
			Interval:    "5m",
			InitialDays: 30,
		},
		UI: UIConfig{
			Theme:       "dark",
			ShowPreview: true,
			PageSize:    50,
			Debug:       false,
		},
		Compose: ComposeConfig{
			Format:           "html",
			SendDelaySeconds: 30,
		},
		Schedule: ScheduleConfig{
			Enabled:          true,
			CheckInterval:    "1m",
			DefaultMorning:   "09:00",
			DefaultAfternoon: "14:00",
			NotifyOnSend:     true,
			NotifyOnFail:     true,
		},
		Basecamp: nil, // Disabled by default
	}
}
