// Package services - BasecampService provides Basecamp integration
// Following REGRA DE OURO: all Basecamp operations go through this service
package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/opik/miau/internal/basecamp"
	"github.com/opik/miau/internal/ports"
)

// BasecampService provides business logic for Basecamp integration
type BasecampService struct {
	mu sync.RWMutex

	client    *basecamp.Client
	connected bool
	projects  map[int64]*ports.BasecampProject // cached projects
	eventBus  ports.EventBus
}

// NewBasecampService creates a new Basecamp service
func NewBasecampService(eventBus ports.EventBus) *BasecampService {
	return &BasecampService{
		projects: make(map[int64]*ports.BasecampProject),
		eventBus: eventBus,
	}
}

// SetClient sets the Basecamp API client
func (s *BasecampService) SetClient(client *basecamp.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.client = client
	s.connected = client != nil
}

// IsConnected returns true if connected to Basecamp
func (s *BasecampService) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected && s.client != nil
}

// Connect connects to Basecamp (client must be set first via SetClient)
func (s *BasecampService) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return fmt.Errorf("basecamp client not configured")
	}

	// Test connection by getting user info
	var _, err = s.client.GetMyInfo()
	if err != nil {
		s.connected = false
		return fmt.Errorf("failed to connect to Basecamp: %w", err)
	}

	s.connected = true
	return nil
}

// Disconnect disconnects from Basecamp
func (s *BasecampService) Disconnect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connected = false
	s.projects = make(map[int64]*ports.BasecampProject)
	return nil
}

// =============================================================================
// PROJECTS
// =============================================================================

// GetProjects returns all projects
func (s *BasecampService) GetProjects(ctx context.Context) ([]ports.BasecampProject, error) {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("basecamp not connected")
	}

	var projects, err = client.ListProjects()
	if err != nil {
		return nil, err
	}

	// Cache projects
	s.mu.Lock()
	for i := range projects {
		s.projects[projects[i].ID] = &projects[i]
	}
	s.mu.Unlock()

	return projects, nil
}

// GetProject returns a single project by ID
func (s *BasecampService) GetProject(ctx context.Context, projectID int64) (*ports.BasecampProject, error) {
	s.mu.RLock()
	var client = s.client
	var cached = s.projects[projectID]
	s.mu.RUnlock()

	if cached != nil {
		return cached, nil
	}

	if client == nil {
		return nil, fmt.Errorf("basecamp not connected")
	}

	var project, err = client.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	// Cache
	s.mu.Lock()
	s.projects[projectID] = project
	s.mu.Unlock()

	return project, nil
}

// =============================================================================
// TO-DO LISTS
// =============================================================================

// GetTodoLists returns all todo lists for a project
func (s *BasecampService) GetTodoLists(ctx context.Context, projectID int64) ([]ports.BasecampTodoList, error) {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("basecamp not connected")
	}

	// First get the todoset
	var todoSet, err = client.GetTodoSet(projectID)
	if err != nil {
		return nil, err
	}

	// Then get all todo lists in that todoset
	var lists []ports.BasecampTodoList
	// The todoset URL contains the lists
	// We need to parse the URL or use a different approach
	// For now, use the project ID as both bucket and todoset
	lists, err = client.ListTodoLists(todoSet.ID)
	if err != nil {
		// Try alternative path
		var path = fmt.Sprintf("/buckets/%d/todosets/%d/todolists.json", projectID, todoSet.ID)
		_ = path // Would need to expose get method or add to client
		return nil, fmt.Errorf("failed to get todo lists: %w", err)
	}

	return lists, nil
}

// =============================================================================
// TO-DOS
// =============================================================================

// GetTodos returns all todos in a todo list
func (s *BasecampService) GetTodos(ctx context.Context, todoListID int64) ([]ports.BasecampTodo, error) {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("basecamp not connected")
	}

	// We need project context - this is a limitation of the Basecamp API
	// For now, return error asking for project context
	return nil, fmt.Errorf("GetTodos requires project context - use GetTodosInProject")
}

// GetTodosInProject returns all todos in a project's todo list
func (s *BasecampService) GetTodosInProject(ctx context.Context, projectID, todoListID int64) ([]ports.BasecampTodo, error) {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("basecamp not connected")
	}

	return client.ListTodosInProject(projectID, todoListID)
}

// GetTodo returns a single todo by ID
func (s *BasecampService) GetTodo(ctx context.Context, todoID int64) (*ports.BasecampTodo, error) {
	return nil, fmt.Errorf("GetTodo requires project context - use GetTodoInProject")
}

// GetTodoInProject returns a todo within a project
func (s *BasecampService) GetTodoInProject(ctx context.Context, projectID, todoID int64) (*ports.BasecampTodo, error) {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("basecamp not connected")
	}

	return client.GetTodoInProject(projectID, todoID)
}

// CreateTodo creates a new todo
func (s *BasecampService) CreateTodo(ctx context.Context, input *ports.BasecampTodoInput) (*ports.BasecampTodo, error) {
	return nil, fmt.Errorf("CreateTodo requires project context - use CreateTodoInProject")
}

// CreateTodoInProject creates a new todo in a project
func (s *BasecampService) CreateTodoInProject(ctx context.Context, projectID int64, input *ports.BasecampTodoInput) (*ports.BasecampTodo, error) {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("basecamp not connected")
	}

	return client.CreateTodoInProject(projectID, input.TodoListID, input.Content, input.AssigneeIDs, input.DueDate)
}

// UpdateTodo updates an existing todo
func (s *BasecampService) UpdateTodo(ctx context.Context, input *ports.BasecampTodoInput) (*ports.BasecampTodo, error) {
	return nil, fmt.Errorf("UpdateTodo requires project context - use UpdateTodoInProject")
}

// UpdateTodoInProject updates a todo within a project
func (s *BasecampService) UpdateTodoInProject(ctx context.Context, projectID int64, input *ports.BasecampTodoInput) (*ports.BasecampTodo, error) {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("basecamp not connected")
	}

	return client.UpdateTodoInProject(projectID, input.ID, input.Content, input.AssigneeIDs, input.DueDate)
}

// CompleteTodo marks a todo as completed
func (s *BasecampService) CompleteTodo(ctx context.Context, todoID int64) error {
	return fmt.Errorf("CompleteTodo requires project context - use CompleteTodoInProject")
}

// CompleteTodoInProject marks a todo as completed
func (s *BasecampService) CompleteTodoInProject(ctx context.Context, projectID, todoID int64) error {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("basecamp not connected")
	}

	return client.CompleteTodoInProject(projectID, todoID)
}

// UncompleteTodo marks a todo as not completed
func (s *BasecampService) UncompleteTodo(ctx context.Context, todoID int64) error {
	return fmt.Errorf("UncompleteTodo requires project context - use UncompleteTodoInProject")
}

// UncompleteTodoInProject marks a todo as not completed
func (s *BasecampService) UncompleteTodoInProject(ctx context.Context, projectID, todoID int64) error {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("basecamp not connected")
	}

	return client.UncompleteTodoInProject(projectID, todoID)
}

// =============================================================================
// MESSAGES
// =============================================================================

// GetMessages returns messages from a project
func (s *BasecampService) GetMessages(ctx context.Context, projectID int64, limit int) ([]ports.BasecampMessage, error) {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("basecamp not connected")
	}

	// Get the message board first
	var board, err = client.GetMessageBoard(projectID)
	if err != nil {
		return nil, err
	}

	// Get messages (page 1 for now)
	return client.ListMessagesInProject(projectID, board.ID, 1)
}

// PostMessage posts a new message to a project
func (s *BasecampService) PostMessage(ctx context.Context, input *ports.BasecampMessageInput) (*ports.BasecampMessage, error) {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("basecamp not connected")
	}

	// Get the message board first
	var board, err = client.GetMessageBoard(input.ProjectID)
	if err != nil {
		return nil, err
	}

	return client.CreateMessageInProject(input.ProjectID, board.ID, input.Subject, input.Content)
}

// =============================================================================
// COMMENTS
// =============================================================================

// GetComments returns comments on a recording
func (s *BasecampService) GetComments(ctx context.Context, recordingID int64) ([]ports.BasecampComment, error) {
	return nil, fmt.Errorf("GetComments requires project context - use GetCommentsInProject")
}

// GetCommentsInProject returns comments on a recording
func (s *BasecampService) GetCommentsInProject(ctx context.Context, projectID, recordingID int64) ([]ports.BasecampComment, error) {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("basecamp not connected")
	}

	return client.ListCommentsInProject(projectID, recordingID)
}

// PostComment posts a comment on a recording
func (s *BasecampService) PostComment(ctx context.Context, recordingID int64, content string) (*ports.BasecampComment, error) {
	return nil, fmt.Errorf("PostComment requires project context - use PostCommentInProject")
}

// PostCommentInProject posts a comment on a recording
func (s *BasecampService) PostCommentInProject(ctx context.Context, projectID, recordingID int64, content string) (*ports.BasecampComment, error) {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("basecamp not connected")
	}

	return client.CreateCommentInProject(projectID, recordingID, content)
}

// =============================================================================
// PEOPLE
// =============================================================================

// GetPeople returns all people in the Basecamp account
func (s *BasecampService) GetPeople(ctx context.Context) ([]ports.BasecampPerson, error) {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("basecamp not connected")
	}

	return client.ListPeople()
}

// GetPerson returns a single person by ID
func (s *BasecampService) GetPerson(ctx context.Context, personID int64) (*ports.BasecampPerson, error) {
	s.mu.RLock()
	var client = s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("basecamp not connected")
	}

	return client.GetPerson(personID)
}

// =============================================================================
// SYNC
// =============================================================================

// SyncProjects syncs all projects from Basecamp
func (s *BasecampService) SyncProjects(ctx context.Context) error {
	var _, err = s.GetProjects(ctx)
	return err
}
