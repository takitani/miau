// Package basecamp implements the Basecamp 3/4 API plugin.
// Types based on https://github.com/basecamp/bc3-api
package basecamp

import "time"

// Authorization represents the response from /authorization.json
type Authorization struct {
	ExpiresAt time.Time `json:"expires_at"`
	Identity  Identity  `json:"identity"`
	Accounts  []Account `json:"accounts"`
}

// Identity represents the current user
type Identity struct {
	ID             int64  `json:"id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	EmailAddress   string `json:"email_address"`
	AvatarURL      string `json:"avatar_url"`
}

// Account represents a Basecamp account
type Account struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Product    string `json:"product"`   // bc3, bcx, campfire, etc.
	HREFUrl    string `json:"href"`      // API base URL
	AppHREFUrl string `json:"app_href"`  // Web URL
}

// Project represents a Basecamp project (basecamp)
type Project struct {
	ID              int64     `json:"id"`
	Status          string    `json:"status"` // active, trashed, archived
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Purpose         string    `json:"purpose"` // topic, team, etc.
	ClientCompany   *Company  `json:"client_company,omitempty"`
	BookmarkedURL   string    `json:"bookmark_url"`
	URL             string    `json:"url"`
	AppURL          string    `json:"app_url"`
	Dock            []Dock    `json:"dock"` // Available tools
}

// Company represents a client company
type Company struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Dock represents an available tool in a project
type Dock struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Name     string `json:"name"`     // todoset, message_board, vault, etc.
	Enabled  bool   `json:"enabled"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	AppURL   string `json:"app_url"`
}

// Person represents a Basecamp user
type Person struct {
	ID                  int64     `json:"id"`
	AttachableSGID      string    `json:"attachable_sgid"`
	Name                string    `json:"name"`
	EmailAddress        string    `json:"email_address"`
	PersonableType      string    `json:"personable_type"` // User, Client
	Title               string    `json:"title"`
	Bio                 string    `json:"bio"`
	Location            string    `json:"location"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
	Admin               bool      `json:"admin"`
	Owner               bool      `json:"owner"`
	Client              bool      `json:"client"`
	Employee            bool      `json:"employee"`
	TimeZone            string    `json:"time_zone"`
	AvatarURL           string    `json:"avatar_url"`
	CompanyObject       *Company  `json:"company,omitempty"`
	CanManageProjects   bool      `json:"can_manage_projects"`
	CanManageTemplates  bool      `json:"can_manage_templates"`
}

// TodoSet represents a todo list container
type TodoSet struct {
	ID               int64     `json:"id"`
	Status           string    `json:"status"`
	VisibleToClients bool      `json:"visible_to_clients"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Title            string    `json:"title"`
	InheritsStatus   bool      `json:"inherits_status"`
	Type             string    `json:"type"`
	URL              string    `json:"url"`
	AppURL           string    `json:"app_url"`
	Bucket           *Bucket   `json:"bucket,omitempty"`
	Creator          *Person   `json:"creator,omitempty"`
	Completed        bool      `json:"completed"`
	CompletedRatio   string    `json:"completed_ratio"`
	Name             string    `json:"name"`
	TodolistsCount   int       `json:"todolists_count"`
	TodolistsURL     string    `json:"todolists_url"`
}

// Bucket is a reference to the parent project
type Bucket struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// TodoList represents a todo list
type TodoList struct {
	ID               int64     `json:"id"`
	Status           string    `json:"status"`
	VisibleToClients bool      `json:"visible_to_clients"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Title            string    `json:"title"`
	InheritsStatus   bool      `json:"inherits_status"`
	Type             string    `json:"type"`
	URL              string    `json:"url"`
	AppURL           string    `json:"app_url"`
	Bucket           *Bucket   `json:"bucket,omitempty"`
	Creator          *Person   `json:"creator,omitempty"`
	Description      string    `json:"description"`
	Completed        bool      `json:"completed"`
	CompletedRatio   string    `json:"completed_ratio"`
	Name             string    `json:"name"`
	TodosURL         string    `json:"todos_url"`
	GroupsURL        string    `json:"groups_url"`
	AppTodosURL      string    `json:"app_todos_url"`
	Parent           *Parent   `json:"parent,omitempty"`
}

// Parent is a reference to the parent todolist or todoset
type Parent struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	AppURL string `json:"app_url"`
}

// Todo represents a to-do item
type Todo struct {
	ID               int64      `json:"id"`
	Status           string     `json:"status"`
	VisibleToClients bool       `json:"visible_to_clients"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Title            string     `json:"title"`
	InheritsStatus   bool       `json:"inherits_status"`
	Type             string     `json:"type"`
	URL              string     `json:"url"`
	AppURL           string     `json:"app_url"`
	Bucket           *Bucket    `json:"bucket,omitempty"`
	Creator          *Person    `json:"creator,omitempty"`
	Description      string     `json:"description"`
	Completed        bool       `json:"completed"`
	Content          string     `json:"content"`
	StartsOn         *Date      `json:"starts_on,omitempty"`
	DueOn            *Date      `json:"due_on,omitempty"`
	Assignees        []Person   `json:"assignees"`
	CompletionURL    string     `json:"completion_url"`
	CommentCount     int        `json:"comments_count"`
	Parent           *Parent    `json:"parent,omitempty"`
	Position         int        `json:"position"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	Completer        *Person    `json:"completer,omitempty"`
}

// Date is a date-only type (no time)
type Date string

// ToTime converts Date to time.Time
func (d *Date) ToTime() *time.Time {
	if d == nil || *d == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", string(*d))
	if err != nil {
		return nil
	}
	return &t
}

// Message represents a message board message
type Message struct {
	ID               int64     `json:"id"`
	Status           string    `json:"status"`
	VisibleToClients bool      `json:"visible_to_clients"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Title            string    `json:"title"`
	InheritsStatus   bool      `json:"inherits_status"`
	Type             string    `json:"type"`
	URL              string    `json:"url"`
	AppURL           string    `json:"app_url"`
	Bucket           *Bucket   `json:"bucket,omitempty"`
	Creator          *Person   `json:"creator,omitempty"`
	Content          string    `json:"content"`
	Subject          string    `json:"subject"`
	Category         *Category `json:"category,omitempty"`
	CommentCount     int       `json:"comments_count"`
	Subscribed       bool      `json:"subscribed"`
}

// Category represents a message category
type Category struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color,omitempty"`
}

// MessageBoard represents a message board
type MessageBoard struct {
	ID               int64     `json:"id"`
	Status           string    `json:"status"`
	VisibleToClients bool      `json:"visible_to_clients"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Title            string    `json:"title"`
	InheritsStatus   bool      `json:"inherits_status"`
	Type             string    `json:"type"`
	URL              string    `json:"url"`
	AppURL           string    `json:"app_url"`
	Bucket           *Bucket   `json:"bucket,omitempty"`
	Creator          *Person   `json:"creator,omitempty"`
	Position         int       `json:"position"`
	MessagesCount    int       `json:"messages_count"`
	MessagesURL      string    `json:"messages_url"`
}

// Comment represents a comment on any recording
type Comment struct {
	ID               int64     `json:"id"`
	Status           string    `json:"status"`
	VisibleToClients bool      `json:"visible_to_clients"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Title            string    `json:"title"`
	InheritsStatus   bool      `json:"inherits_status"`
	Type             string    `json:"type"`
	URL              string    `json:"url"`
	AppURL           string    `json:"app_url"`
	Bucket           *Bucket   `json:"bucket,omitempty"`
	Creator          *Person   `json:"creator,omitempty"`
	Content          string    `json:"content"`
	Parent           *Parent   `json:"parent,omitempty"`
}

// Schedule represents a schedule (calendar)
type Schedule struct {
	ID               int64     `json:"id"`
	Status           string    `json:"status"`
	VisibleToClients bool      `json:"visible_to_clients"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Title            string    `json:"title"`
	InheritsStatus   bool      `json:"inherits_status"`
	Type             string    `json:"type"`
	URL              string    `json:"url"`
	AppURL           string    `json:"app_url"`
	Bucket           *Bucket   `json:"bucket,omitempty"`
	Creator          *Person   `json:"creator,omitempty"`
	EntriesCount     int       `json:"entries_count"`
	EntriesURL       string    `json:"entries_url"`
}

// ScheduleEntry represents a calendar event
type ScheduleEntry struct {
	ID               int64      `json:"id"`
	Status           string     `json:"status"`
	VisibleToClients bool       `json:"visible_to_clients"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	Title            string     `json:"title"`
	InheritsStatus   bool       `json:"inherits_status"`
	Type             string     `json:"type"`
	URL              string     `json:"url"`
	AppURL           string     `json:"app_url"`
	Bucket           *Bucket    `json:"bucket,omitempty"`
	Creator          *Person    `json:"creator,omitempty"`
	Description      string     `json:"description"`
	Summary          string     `json:"summary"`
	AllDay           bool       `json:"all_day"`
	StartsAt         time.Time  `json:"starts_at"`
	EndsAt           time.Time  `json:"ends_at"`
	Participants     []Person   `json:"participants"`
	CommentCount     int        `json:"comments_count"`
	Subscribed       bool       `json:"subscribed"`
	RecurrenceSchedule string   `json:"recurrence_schedule,omitempty"`
}

// Document represents a document in a vault
type Document struct {
	ID               int64     `json:"id"`
	Status           string    `json:"status"`
	VisibleToClients bool      `json:"visible_to_clients"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Title            string    `json:"title"`
	InheritsStatus   bool      `json:"inherits_status"`
	Type             string    `json:"type"`
	URL              string    `json:"url"`
	AppURL           string    `json:"app_url"`
	Bucket           *Bucket   `json:"bucket,omitempty"`
	Creator          *Person   `json:"creator,omitempty"`
	Content          string    `json:"content"`
	Position         int       `json:"position"`
}

// Vault represents a files/docs vault
type Vault struct {
	ID               int64     `json:"id"`
	Status           string    `json:"status"`
	VisibleToClients bool      `json:"visible_to_clients"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Title            string    `json:"title"`
	InheritsStatus   bool      `json:"inherits_status"`
	Type             string    `json:"type"`
	URL              string    `json:"url"`
	AppURL           string    `json:"app_url"`
	Bucket           *Bucket   `json:"bucket,omitempty"`
	Creator          *Person   `json:"creator,omitempty"`
	DocumentsCount   int       `json:"documents_count"`
	DocumentsURL     string    `json:"documents_url"`
	UploadsCount     int       `json:"uploads_count"`
	UploadsURL       string    `json:"uploads_url"`
	VaultsCount      int       `json:"vaults_count"`
	VaultsURL        string    `json:"vaults_url"`
}

// Upload represents an uploaded file
type Upload struct {
	ID               int64     `json:"id"`
	Status           string    `json:"status"`
	VisibleToClients bool      `json:"visible_to_clients"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	Title            string    `json:"title"`
	InheritsStatus   bool      `json:"inherits_status"`
	Type             string    `json:"type"`
	URL              string    `json:"url"`
	AppURL           string    `json:"app_url"`
	Bucket           *Bucket   `json:"bucket,omitempty"`
	Creator          *Person   `json:"creator,omitempty"`
	Description      string    `json:"description"`
	Filename         string    `json:"filename"`
	ContentType      string    `json:"content_type"`
	ByteSize         int64     `json:"byte_size"`
	Width            int       `json:"width,omitempty"`
	Height           int       `json:"height,omitempty"`
	DownloadURL      string    `json:"download_url"`
	AppDownloadURL   string    `json:"app_download_url"`
}

// CreateTodoRequest is the request body for creating a todo
type CreateTodoRequest struct {
	Content       string   `json:"content"`
	Description   string   `json:"description,omitempty"`
	AssigneeIDs   []int64  `json:"assignee_ids,omitempty"`
	CompletionSubscriberIDs []int64 `json:"completion_subscriber_ids,omitempty"`
	Notify        bool     `json:"notify,omitempty"`
	DueOn         string   `json:"due_on,omitempty"`  // YYYY-MM-DD
	StartsOn      string   `json:"starts_on,omitempty"` // YYYY-MM-DD
}

// UpdateTodoRequest is the request body for updating a todo
type UpdateTodoRequest struct {
	Content       *string  `json:"content,omitempty"`
	Description   *string  `json:"description,omitempty"`
	AssigneeIDs   []int64  `json:"assignee_ids,omitempty"`
	DueOn         *string  `json:"due_on,omitempty"`
	StartsOn      *string  `json:"starts_on,omitempty"`
}

// CreateMessageRequest is the request body for creating a message
type CreateMessageRequest struct {
	Subject    string `json:"subject"`
	Content    string `json:"content"`
	CategoryID int64  `json:"category_id,omitempty"`
}

// Pagination info from Link header
type Pagination struct {
	NextURL string
	TotalCount int
}
