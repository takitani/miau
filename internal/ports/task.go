package ports

import (
	"context"
	"time"
)

// TaskService defines the interface for task operations
type TaskService interface {
	// CreateTask creates a new task
	CreateTask(ctx context.Context, task *TaskInput) (*TaskInfo, error)

	// GetTask returns a task by ID
	GetTask(ctx context.Context, id int64) (*TaskInfo, error)

	// GetTasks returns all tasks for an account
	GetTasks(ctx context.Context, accountID int64) ([]TaskInfo, error)

	// GetPendingTasks returns only incomplete tasks
	GetPendingTasks(ctx context.Context, accountID int64) ([]TaskInfo, error)

	// GetCompletedTasks returns completed tasks
	GetCompletedTasks(ctx context.Context, accountID int64, limit int) ([]TaskInfo, error)

	// GetTasksByEmail returns tasks linked to a specific email
	GetTasksByEmail(ctx context.Context, emailID int64) ([]TaskInfo, error)

	// UpdateTask updates an existing task
	UpdateTask(ctx context.Context, task *TaskInput) (*TaskInfo, error)

	// ToggleTaskCompleted toggles the completed status
	ToggleTaskCompleted(ctx context.Context, id int64) (bool, error)

	// DeleteTask removes a task
	DeleteTask(ctx context.Context, id int64) error

	// DeleteCompletedTasks removes all completed tasks for an account
	DeleteCompletedTasks(ctx context.Context, accountID int64) (int64, error)

	// CountTasks returns task counts by status
	CountTasks(ctx context.Context, accountID int64) (*TaskCounts, error)
}

// TaskPriority represents task priority levels
type TaskPriority int

const (
	TaskPriorityNormal TaskPriority = 0
	TaskPriorityHigh   TaskPriority = 1
	TaskPriorityUrgent TaskPriority = 2
)

// TaskSource represents how the task was created
type TaskSource string

const (
	TaskSourceManual       TaskSource = "manual"
	TaskSourceAISuggestion TaskSource = "ai_suggestion"
)

// TaskInput represents input for creating/updating a task
type TaskInput struct {
	ID          int64
	AccountID   int64
	Title       string
	Description string
	IsCompleted bool
	Priority    TaskPriority
	DueDate     *time.Time
	EmailID     *int64
	Source      TaskSource
}

// TaskInfo represents task information returned by the service
type TaskInfo struct {
	ID          int64
	AccountID   int64
	Title       string
	Description string
	IsCompleted bool
	Priority    TaskPriority
	DueDate     *time.Time
	EmailID     *int64
	Source      TaskSource
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TaskCounts represents task count statistics
type TaskCounts struct {
	Pending   int
	Completed int
	Total     int
}

// TaskStoragePort defines the storage interface for tasks
type TaskStoragePort interface {
	CreateTask(task *TaskInput) (*TaskInfo, error)
	GetTask(id int64) (*TaskInfo, error)
	GetTasks(accountID int64) ([]TaskInfo, error)
	GetPendingTasks(accountID int64) ([]TaskInfo, error)
	GetCompletedTasks(accountID int64, limit int) ([]TaskInfo, error)
	GetTasksByEmail(emailID int64) ([]TaskInfo, error)
	UpdateTask(task *TaskInput) error
	ToggleTaskCompleted(id int64) (bool, error)
	DeleteTask(id int64) error
	DeleteCompletedTasks(accountID int64) (int64, error)
	CountTasks(accountID int64) (pending, completed int, err error)
}
