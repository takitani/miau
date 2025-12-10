// Package ports defines the normalized types for external plugin data.
// All plugins convert their native data formats to these common types.
package ports

import "time"

// ExternalItemType identifies the type of external item
type ExternalItemType string

const (
	ExternalItemTask     ExternalItemType = "task"
	ExternalItemMessage  ExternalItemType = "message"
	ExternalItemComment  ExternalItemType = "comment"
	ExternalItemDocument ExternalItemType = "document"
	ExternalItemEvent    ExternalItemType = "event"
	ExternalItemFile     ExternalItemType = "file"
)

// ExternalItem is a generic container for any external item.
// Used for storage and search results.
type ExternalItem struct {
	ID           string           `json:"id"`
	PluginID     PluginID         `json:"plugin_id"`
	ProjectID    string           `json:"project_id"`
	ProjectName  string           `json:"project_name"`
	Type         ExternalItemType `json:"type"`
	Title        string           `json:"title"`
	Content      string           `json:"content"`       // Body text
	ContentHTML  string           `json:"content_html"`  // HTML if available
	URL          string           `json:"url"`           // Link to original
	Status       string           `json:"status"`        // pending, completed, etc.
	Priority     string           `json:"priority"`      // low, normal, high
	DueAt        *time.Time       `json:"due_at,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`
	Creator      *ExternalPerson  `json:"creator,omitempty"`
	Assignees    []ExternalPerson `json:"assignees,omitempty"`
	Tags         []string         `json:"tags,omitempty"`
	Attachments  []ExternalFile   `json:"attachments,omitempty"`
	ParentID     string           `json:"parent_id,omitempty"` // For nested items
	CommentCount int              `json:"comment_count"`
	Metadata     map[string]any   `json:"metadata,omitempty"` // Plugin-specific data
}

// ExternalProject represents a project/workspace from an external system
type ExternalProject struct {
	ID          string          `json:"id"`
	PluginID    PluginID        `json:"plugin_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	URL         string          `json:"url"`
	Status      string          `json:"status"` // active, archived
	Color       string          `json:"color"`  // Hex color if available
	Icon        string          `json:"icon"`   // Emoji or path
	Creator     *ExternalPerson `json:"creator,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ItemCount   int             `json:"item_count"`
	Metadata    map[string]any  `json:"metadata,omitempty"`
}

// ExternalTask represents a to-do/task from an external system
type ExternalTask struct {
	ID            string           `json:"id"`
	PluginID      PluginID         `json:"plugin_id"`
	ProjectID     string           `json:"project_id"`
	ProjectName   string           `json:"project_name"`
	ListID        string           `json:"list_id,omitempty"`   // To-do list ID
	ListName      string           `json:"list_name,omitempty"` // To-do list name
	Title         string           `json:"title"`
	Description   string           `json:"description"`
	DescriptionHTML string         `json:"description_html"`
	URL           string           `json:"url"`
	Status        string           `json:"status"` // pending, completed
	Priority      string           `json:"priority"`
	Position      int              `json:"position"` // Sort order
	DueOn         *time.Time       `json:"due_on,omitempty"`
	DueAt         *time.Time       `json:"due_at,omitempty"` // With time
	StartsOn      *time.Time       `json:"starts_on,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	CompletedAt   *time.Time       `json:"completed_at,omitempty"`
	CompletedBy   *ExternalPerson  `json:"completed_by,omitempty"`
	Creator       *ExternalPerson  `json:"creator,omitempty"`
	Assignees     []ExternalPerson `json:"assignees,omitempty"`
	Tags          []string         `json:"tags,omitempty"`
	CommentCount  int              `json:"comment_count"`
	Attachments   []ExternalFile   `json:"attachments,omitempty"`
	Metadata      map[string]any   `json:"metadata,omitempty"`
}

// ExternalTaskCreate is used to create a new task
type ExternalTaskCreate struct {
	ProjectID   string    `json:"project_id"`
	ListID      string    `json:"list_id,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	DueOn       *time.Time `json:"due_on,omitempty"`
	AssigneeIDs []string  `json:"assignee_ids,omitempty"`
	Notify      bool      `json:"notify"`
}

// ExternalTaskUpdate is used to update an existing task
type ExternalTaskUpdate struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	DueOn       *time.Time `json:"due_on,omitempty"`
	AssigneeIDs []string   `json:"assignee_ids,omitempty"`
	Completed   *bool      `json:"completed,omitempty"`
}

// ExternalMessage represents a message/post from an external system
type ExternalMessage struct {
	ID           string           `json:"id"`
	PluginID     PluginID         `json:"plugin_id"`
	ProjectID    string           `json:"project_id"`
	ProjectName  string           `json:"project_name"`
	Category     string           `json:"category,omitempty"` // Announcement, FYI, etc.
	Subject      string           `json:"subject"`
	Content      string           `json:"content"`
	ContentHTML  string           `json:"content_html"`
	URL          string           `json:"url"`
	Author       *ExternalPerson  `json:"author,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
	CommentCount int              `json:"comment_count"`
	Attachments  []ExternalFile   `json:"attachments,omitempty"`
	Subscribers  []ExternalPerson `json:"subscribers,omitempty"`
	Metadata     map[string]any   `json:"metadata,omitempty"`
}

// ExternalMessageCreate is used to create a new message
type ExternalMessageCreate struct {
	ProjectID string `json:"project_id"`
	Subject   string `json:"subject"`
	Content   string `json:"content"`
	Category  string `json:"category,omitempty"`
}

// ExternalComment represents a comment on any item
type ExternalComment struct {
	ID          string          `json:"id"`
	PluginID    PluginID        `json:"plugin_id"`
	ParentID    string          `json:"parent_id"` // The item being commented on
	ParentType  string          `json:"parent_type"`
	Content     string          `json:"content"`
	ContentHTML string          `json:"content_html"`
	URL         string          `json:"url"`
	Author      *ExternalPerson `json:"author,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Attachments []ExternalFile  `json:"attachments,omitempty"`
	Metadata    map[string]any  `json:"metadata,omitempty"`
}

// ExternalDocument represents a document/note from an external system
type ExternalDocument struct {
	ID          string          `json:"id"`
	PluginID    PluginID        `json:"plugin_id"`
	ProjectID   string          `json:"project_id"`
	ProjectName string          `json:"project_name"`
	Title       string          `json:"title"`
	Content     string          `json:"content"`
	ContentHTML string          `json:"content_html"`
	URL         string          `json:"url"`
	Status      string          `json:"status"` // active, archived
	Author      *ExternalPerson `json:"author,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Metadata    map[string]any  `json:"metadata,omitempty"`
}

// ExternalEvent represents a calendar event from an external system
type ExternalEvent struct {
	ID          string           `json:"id"`
	PluginID    PluginID         `json:"plugin_id"`
	ProjectID   string           `json:"project_id"`
	ProjectName string           `json:"project_name"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	URL         string           `json:"url"`
	Location    string           `json:"location,omitempty"`
	StartsAt    time.Time        `json:"starts_at"`
	EndsAt      time.Time        `json:"ends_at"`
	AllDay      bool             `json:"all_day"`
	Recurring   bool             `json:"recurring"`
	Creator     *ExternalPerson  `json:"creator,omitempty"`
	Attendees   []ExternalPerson `json:"attendees,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Metadata    map[string]any   `json:"metadata,omitempty"`
}

// ExternalPerson represents a user from an external system
type ExternalPerson struct {
	ID         string         `json:"id"`
	PluginID   PluginID       `json:"plugin_id"`
	Name       string         `json:"name"`
	Email      string         `json:"email"`
	AvatarURL  string         `json:"avatar_url,omitempty"`
	Title      string         `json:"title,omitempty"` // Job title
	Company    string         `json:"company,omitempty"`
	Admin      bool           `json:"admin"`
	Owner      bool           `json:"owner"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// ExternalFile represents a file/attachment from an external system
type ExternalFile struct {
	ID          string         `json:"id"`
	PluginID    PluginID       `json:"plugin_id"`
	Name        string         `json:"name"`
	ContentType string         `json:"content_type"`
	Size        int64          `json:"size"`
	URL         string         `json:"url"` // Download URL
	Uploader    *ExternalPerson `json:"uploader,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// ToExternalItem converts an ExternalTask to ExternalItem
func (t *ExternalTask) ToExternalItem() ExternalItem {
	return ExternalItem{
		ID:           t.ID,
		PluginID:     t.PluginID,
		ProjectID:    t.ProjectID,
		ProjectName:  t.ProjectName,
		Type:         ExternalItemTask,
		Title:        t.Title,
		Content:      t.Description,
		ContentHTML:  t.DescriptionHTML,
		URL:          t.URL,
		Status:       t.Status,
		Priority:     t.Priority,
		DueAt:        t.DueAt,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
		CompletedAt:  t.CompletedAt,
		Creator:      t.Creator,
		Assignees:    t.Assignees,
		Tags:         t.Tags,
		Attachments:  t.Attachments,
		CommentCount: t.CommentCount,
		Metadata:     t.Metadata,
	}
}

// ToExternalItem converts an ExternalMessage to ExternalItem
func (m *ExternalMessage) ToExternalItem() ExternalItem {
	return ExternalItem{
		ID:           m.ID,
		PluginID:     m.PluginID,
		ProjectID:    m.ProjectID,
		ProjectName:  m.ProjectName,
		Type:         ExternalItemMessage,
		Title:        m.Subject,
		Content:      m.Content,
		ContentHTML:  m.ContentHTML,
		URL:          m.URL,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		Creator:      m.Author,
		Attachments:  m.Attachments,
		CommentCount: m.CommentCount,
		Metadata:     m.Metadata,
	}
}

// ToExternalItem converts an ExternalDocument to ExternalItem
func (d *ExternalDocument) ToExternalItem() ExternalItem {
	return ExternalItem{
		ID:          d.ID,
		PluginID:    d.PluginID,
		ProjectID:   d.ProjectID,
		ProjectName: d.ProjectName,
		Type:        ExternalItemDocument,
		Title:       d.Title,
		Content:     d.Content,
		ContentHTML: d.ContentHTML,
		URL:         d.URL,
		Status:      d.Status,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
		Creator:     d.Author,
		Metadata:    d.Metadata,
	}
}

// ToExternalItem converts an ExternalEvent to ExternalItem
func (e *ExternalEvent) ToExternalItem() ExternalItem {
	return ExternalItem{
		ID:          e.ID,
		PluginID:    e.PluginID,
		ProjectID:   e.ProjectID,
		ProjectName: e.ProjectName,
		Type:        ExternalItemEvent,
		Title:       e.Title,
		Content:     e.Description,
		URL:         e.URL,
		DueAt:       &e.StartsAt,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		Creator:     e.Creator,
		Assignees:   e.Attendees,
		Metadata:    e.Metadata,
	}
}

// ToExternalItem converts an ExternalComment to ExternalItem
func (c *ExternalComment) ToExternalItem() ExternalItem {
	return ExternalItem{
		ID:          c.ID,
		PluginID:    c.PluginID,
		Type:        ExternalItemComment,
		Title:       "", // Comments don't have titles
		Content:     c.Content,
		ContentHTML: c.ContentHTML,
		URL:         c.URL,
		ParentID:    c.ParentID,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		Creator:     c.Author,
		Attachments: c.Attachments,
		Metadata:    c.Metadata,
	}
}
