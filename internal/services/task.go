package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/storage"
)

// TaskService implements ports.TaskService
type TaskService struct {
}

// NewTaskService creates a new TaskService
func NewTaskService() *TaskService {
	return &TaskService{}
}

// CreateTask creates a new task
func (s *TaskService) CreateTask(ctx context.Context, input *ports.TaskInput) (*ports.TaskInfo, error) {
	if input.Title == "" {
		return nil, fmt.Errorf("task title is required")
	}

	var task = &storage.Task{
		AccountID:   input.AccountID,
		Title:       input.Title,
		Description: toNullString(input.Description),
		IsCompleted: input.IsCompleted,
		Priority:    storage.TaskPriority(input.Priority),
		DueDate:     toNullTime(input.DueDate),
		EmailID:     toNullInt64(input.EmailID),
		Source:      storage.TaskSource(input.Source),
	}

	if task.Source == "" {
		task.Source = storage.TaskSourceManual
	}

	err := storage.CreateTask(task)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return storageTaskToInfo(task), nil
}

// GetTask returns a task by ID
func (s *TaskService) GetTask(ctx context.Context, id int64) (*ports.TaskInfo, error) {
	task, err := storage.GetTask(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return nil, nil
	}
	return storageTaskToInfo(task), nil
}

// GetTasks returns all tasks for an account
func (s *TaskService) GetTasks(ctx context.Context, accountID int64) ([]ports.TaskInfo, error) {
	tasks, err := storage.GetTasks(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}
	return storageTasksToInfos(tasks), nil
}

// GetPendingTasks returns only incomplete tasks
func (s *TaskService) GetPendingTasks(ctx context.Context, accountID int64) ([]ports.TaskInfo, error) {
	tasks, err := storage.GetPendingTasks(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending tasks: %w", err)
	}
	return storageTasksToInfos(tasks), nil
}

// GetCompletedTasks returns completed tasks
func (s *TaskService) GetCompletedTasks(ctx context.Context, accountID int64, limit int) ([]ports.TaskInfo, error) {
	if limit <= 0 {
		limit = 50
	}
	tasks, err := storage.GetCompletedTasks(accountID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get completed tasks: %w", err)
	}
	return storageTasksToInfos(tasks), nil
}

// GetTasksByEmail returns tasks linked to a specific email
func (s *TaskService) GetTasksByEmail(ctx context.Context, emailID int64) ([]ports.TaskInfo, error) {
	tasks, err := storage.GetTasksByEmail(emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks by email: %w", err)
	}
	return storageTasksToInfos(tasks), nil
}

// UpdateTask updates an existing task
func (s *TaskService) UpdateTask(ctx context.Context, input *ports.TaskInput) (*ports.TaskInfo, error) {
	if input.ID == 0 {
		return nil, fmt.Errorf("task ID is required")
	}
	if input.Title == "" {
		return nil, fmt.Errorf("task title is required")
	}

	var task = &storage.Task{
		ID:          input.ID,
		AccountID:   input.AccountID,
		Title:       input.Title,
		Description: toNullString(input.Description),
		IsCompleted: input.IsCompleted,
		Priority:    storage.TaskPriority(input.Priority),
		DueDate:     toNullTime(input.DueDate),
		EmailID:     toNullInt64(input.EmailID),
		Source:      storage.TaskSource(input.Source),
	}

	err := storage.UpdateTask(task)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	// Fetch updated task
	return s.GetTask(ctx, input.ID)
}

// ToggleTaskCompleted toggles the completed status
func (s *TaskService) ToggleTaskCompleted(ctx context.Context, id int64) (bool, error) {
	newStatus, err := storage.ToggleTaskCompleted(id)
	if err != nil {
		return false, fmt.Errorf("failed to toggle task: %w", err)
	}
	return newStatus, nil
}

// DeleteTask removes a task
func (s *TaskService) DeleteTask(ctx context.Context, id int64) error {
	err := storage.DeleteTask(id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	return nil
}

// DeleteCompletedTasks removes all completed tasks for an account
func (s *TaskService) DeleteCompletedTasks(ctx context.Context, accountID int64) (int64, error) {
	count, err := storage.DeleteCompletedTasks(accountID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete completed tasks: %w", err)
	}
	return count, nil
}

// CountTasks returns task counts by status
func (s *TaskService) CountTasks(ctx context.Context, accountID int64) (*ports.TaskCounts, error) {
	pending, completed, err := storage.CountTasks(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to count tasks: %w", err)
	}
	return &ports.TaskCounts{
		Pending:   pending,
		Completed: completed,
		Total:     pending + completed,
	}, nil
}

// Helper functions

func storageTaskToInfo(t *storage.Task) *ports.TaskInfo {
	var info = &ports.TaskInfo{
		ID:          t.ID,
		AccountID:   t.AccountID,
		Title:       t.Title,
		Description: nullStringToString(t.Description),
		IsCompleted: t.IsCompleted,
		Priority:    ports.TaskPriority(t.Priority),
		DueDate:     nullTimeToPtr(t.DueDate),
		EmailID:     nullInt64ToPtr(t.EmailID),
		Source:      ports.TaskSource(t.Source),
		CreatedAt:   t.CreatedAt.Time,
		UpdatedAt:   t.UpdatedAt.Time,
	}
	return info
}

func storageTasksToInfos(tasks []storage.Task) []ports.TaskInfo {
	var infos = make([]ports.TaskInfo, len(tasks))
	for i, t := range tasks {
		infos[i] = *storageTaskToInfo(&t)
	}
	return infos
}

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func toNullInt64(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

func nullStringToString(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}

func nullTimeToPtr(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	return &nt.Time
}

func nullInt64ToPtr(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}
	return &ni.Int64
}
