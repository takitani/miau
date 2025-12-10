package ports

import (
	"context"
	"time"
)

// BasecampService provides business logic for Basecamp integration
// Following REGRA DE OURO: all Basecamp operations go through this service
type BasecampService interface {
	// Connection status
	IsConnected() bool
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error

	// Projects
	GetProjects(ctx context.Context) ([]BasecampProject, error)
	GetProject(ctx context.Context, projectID int64) (*BasecampProject, error)

	// To-dos
	GetTodoLists(ctx context.Context, projectID int64) ([]BasecampTodoList, error)
	GetTodos(ctx context.Context, todoListID int64) ([]BasecampTodo, error)
	GetTodo(ctx context.Context, todoID int64) (*BasecampTodo, error)
	CreateTodo(ctx context.Context, input *BasecampTodoInput) (*BasecampTodo, error)
	UpdateTodo(ctx context.Context, input *BasecampTodoInput) (*BasecampTodo, error)
	CompleteTodo(ctx context.Context, todoID int64) error
	UncompleteTodo(ctx context.Context, todoID int64) error

	// Messages (Campfire / Message Board)
	GetMessages(ctx context.Context, projectID int64, limit int) ([]BasecampMessage, error)
	PostMessage(ctx context.Context, input *BasecampMessageInput) (*BasecampMessage, error)

	// Comments
	GetComments(ctx context.Context, recordingID int64) ([]BasecampComment, error)
	PostComment(ctx context.Context, recordingID int64, content string) (*BasecampComment, error)

	// People
	GetPeople(ctx context.Context) ([]BasecampPerson, error)
	GetPerson(ctx context.Context, personID int64) (*BasecampPerson, error)

	// Sync
	SyncProjects(ctx context.Context) error
}

// BasecampAPIPort defines the low-level API client interface
type BasecampAPIPort interface {
	// Projects
	ListProjects() ([]BasecampProject, error)
	GetProject(projectID int64) (*BasecampProject, error)

	// To-do lists
	GetTodoSet(projectID int64) (*BasecampTodoSet, error)
	ListTodoLists(todoSetID int64) ([]BasecampTodoList, error)
	GetTodoList(todoListID int64) (*BasecampTodoList, error)

	// To-dos
	ListTodos(todoListID int64) ([]BasecampTodo, error)
	GetTodo(todoID int64) (*BasecampTodo, error)
	CreateTodo(todoListID int64, content string, assigneeIDs []int64, dueDate *time.Time) (*BasecampTodo, error)
	UpdateTodo(todoID int64, content string, assigneeIDs []int64, dueDate *time.Time) (*BasecampTodo, error)
	CompleteTodo(todoID int64) error
	UncompleteTodo(todoID int64) error

	// Message Board
	GetMessageBoard(projectID int64) (*BasecampMessageBoard, error)
	ListMessages(messageBoardID int64, page int) ([]BasecampMessage, error)
	CreateMessage(messageBoardID int64, subject, content string) (*BasecampMessage, error)

	// Comments
	ListComments(recordingID int64) ([]BasecampComment, error)
	CreateComment(recordingID int64, content string) (*BasecampComment, error)

	// People
	ListPeople() ([]BasecampPerson, error)
	GetPerson(personID int64) (*BasecampPerson, error)
}

// BasecampStoragePort defines storage operations for Basecamp data (optional caching)
type BasecampStoragePort interface {
	// Projects
	SaveProject(project *BasecampProject) error
	GetProjects(accountID int64) ([]BasecampProject, error)
	GetProject(projectID int64) (*BasecampProject, error)

	// Todo lists
	SaveTodoList(list *BasecampTodoList) error
	GetTodoLists(projectID int64) ([]BasecampTodoList, error)

	// Todos
	SaveTodo(todo *BasecampTodo) error
	GetTodos(todoListID int64) ([]BasecampTodo, error)
	GetTodo(todoID int64) (*BasecampTodo, error)
}

// =============================================================================
// DATA TYPES
// =============================================================================

// BasecampProject represents a Basecamp project
type BasecampProject struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Purpose     string    `json:"purpose"` // "topic" or "team"
	Status      string    `json:"status"`  // "active", "archived", "trashed"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	BookmarksCount int    `json:"bookmarks_count"`

	// Dock contains links to project tools (message_board, todoset, etc.)
	Dock []BasecampDockItem `json:"dock"`
}

// BasecampDockItem represents a tool in a project's dock
type BasecampDockItem struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Name     string `json:"name"` // "message_board", "todoset", "vault", etc.
	Enabled  bool   `json:"enabled"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

// BasecampTodoSet represents the To-do set container in a project
type BasecampTodoSet struct {
	ID               int64  `json:"id"`
	Title            string `json:"title"`
	TodoListsCount   int    `json:"todolists_count"`
	TodoListsURL     string `json:"todolists_url"`
	CompletedCount   int    `json:"completed_count"`
	CompletedRatio   string `json:"completed_ratio"`
}

// BasecampTodoList represents a to-do list
type BasecampTodoList struct {
	ID              int64             `json:"id"`
	ProjectID       int64             `json:"project_id"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Name            string            `json:"name"` // for named to-do lists
	Completed       bool              `json:"completed"`
	CompletedRatio  string            `json:"completed_ratio"`
	TodosURL        string            `json:"todos_url"`
	TodosCount      int               `json:"todos_count"`
	CompletedCount  int               `json:"completed_count"`
	Creator         *BasecampPerson   `json:"creator"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// BasecampTodo represents a single to-do item
type BasecampTodo struct {
	ID          int64             `json:"id"`
	TodoListID  int64             `json:"parent_id"`
	ProjectID   int64             `json:"bucket_id"`
	Content     string            `json:"content"`
	Description string            `json:"description"`
	StartsOn    *string           `json:"starts_on"`  // date string
	DueOn       *string           `json:"due_on"`     // date string
	Completed   bool              `json:"completed"`
	CompletedAt *time.Time        `json:"completed_at"`
	Creator     *BasecampPerson   `json:"creator"`
	Assignees   []BasecampPerson  `json:"assignees"`
	Position    int               `json:"position"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	CommentsCount int             `json:"comments_count"`
	CommentsURL   string          `json:"comments_url"`
}

// BasecampTodoInput is used to create/update todos
type BasecampTodoInput struct {
	ID          int64     `json:"id,omitempty"`
	TodoListID  int64     `json:"todo_list_id"`
	Content     string    `json:"content"`
	Description string    `json:"description,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	AssigneeIDs []int64   `json:"assignee_ids,omitempty"`
}

// BasecampMessageBoard represents a message board in a project
type BasecampMessageBoard struct {
	ID           int64  `json:"id"`
	Title        string `json:"title"`
	MessagesCount int   `json:"messages_count"`
	MessagesURL  string `json:"messages_url"`
}

// BasecampMessage represents a message post
type BasecampMessage struct {
	ID          int64           `json:"id"`
	ProjectID   int64           `json:"bucket_id"`
	Subject     string          `json:"subject"`
	Content     string          `json:"content"`
	Creator     *BasecampPerson `json:"creator"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	CommentsCount int           `json:"comments_count"`
	CommentsURL   string        `json:"comments_url"`
}

// BasecampMessageInput is used to create messages
type BasecampMessageInput struct {
	ProjectID int64  `json:"project_id"`
	Subject   string `json:"subject"`
	Content   string `json:"content"`
}

// BasecampComment represents a comment on any recording
type BasecampComment struct {
	ID        int64           `json:"id"`
	Content   string          `json:"content"`
	Creator   *BasecampPerson `json:"creator"`
	CreatedAt time.Time       `json:"created_at"`
}

// BasecampPerson represents a person/user in Basecamp
type BasecampPerson struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	EmailAddress   string `json:"email_address"`
	PersonableType string `json:"personable_type"` // "User" or "Client"
	Title          string `json:"title"`
	AvatarURL      string `json:"avatar_url"`
	Company        *struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"company"`
	Admin     bool      `json:"admin"`
	Owner     bool      `json:"owner"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
