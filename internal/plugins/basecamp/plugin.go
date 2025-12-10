// Package basecamp implements the Basecamp plugin for miau.
// It provides access to projects, todos, messages, schedules, and documents.
package basecamp

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/opik/miau/internal/ports"
)

const (
	PluginID      = "basecamp"
	PluginName    = "Basecamp"
	PluginVersion = "1.0.0"
)

// Plugin implements the Basecamp integration
type Plugin struct {
	mu sync.RWMutex

	client       *Client
	config       ports.PluginConfig
	status       ports.PluginStatus
	auth         *Authorization
	selectedAcct *Account

	// OAuth2 config
	clientID     string
	clientSecret string
	redirectURI  string

	// Cached data
	projects []Project
}

// New creates a new Basecamp plugin instance
func New() *Plugin {
	return &Plugin{
		status: ports.PluginStatusDisabled,
	}
}

// Info returns plugin metadata
func (p *Plugin) Info() ports.PluginInfo {
	return ports.PluginInfo{
		ID:          PluginID,
		Name:        PluginName,
		Description: "Connect to Basecamp 3/4 for projects, to-dos, and messages",
		Version:     PluginVersion,
		Author:      "miau",
		Website:     "https://basecamp.com",
		Icon:        "🏕️",
		Capabilities: []ports.PluginCapability{
			ports.CapabilityProjects,
			ports.CapabilityTasks,
			ports.CapabilityMessages,
			ports.CapabilityCalendar,
			ports.CapabilityPeople,
			ports.CapabilitySearch,
			ports.CapabilityWrite,
		},
		AuthType: ports.PluginAuthOAuth2,
	}
}

// Initialize sets up the plugin with configuration
func (p *Plugin) Initialize(ctx context.Context, config ports.PluginConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.config = config

	// Load OAuth2 settings
	if config.OAuth != nil {
		p.clientID = config.OAuth.ClientID
		p.clientSecret = config.OAuth.ClientSecret
		p.redirectURI = config.OAuth.RedirectURL
	}

	// Load saved credentials
	if config.Credentials != nil {
		accessToken := config.Credentials["access_token"]
		accountIDStr := config.Credentials["account_id"]

		if accessToken != "" && accountIDStr != "" {
			accountID, _ := strconv.ParseInt(accountIDStr, 10, 64)
			p.client = NewClient(accessToken, accountID)
			p.status = ports.PluginStatusEnabled
		}
	}

	return nil
}

// Connect establishes connection to Basecamp
func (p *Plugin) Connect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.client == nil {
		p.status = ports.PluginStatusAuthRequired
		return fmt.Errorf("not authenticated, OAuth2 required")
	}

	p.status = ports.PluginStatusConnecting

	// Verify credentials and get authorization info
	auth, err := p.client.GetAuthorization(ctx)
	if err != nil {
		p.status = ports.PluginStatusError
		return fmt.Errorf("failed to verify credentials: %w", err)
	}

	p.auth = auth

	// Find the selected account
	accountIDStr := p.config.Credentials["account_id"]
	if accountIDStr != "" {
		accountID, _ := strconv.ParseInt(accountIDStr, 10, 64)
		for _, acct := range auth.Accounts {
			if acct.ID == accountID && acct.Product == "bc3" {
				p.selectedAcct = &acct
				break
			}
		}
	}

	// If no account selected, use first bc3 account
	if p.selectedAcct == nil {
		for _, acct := range auth.Accounts {
			if acct.Product == "bc3" {
				p.selectedAcct = &acct
				p.client.SetAccountID(acct.ID)
				break
			}
		}
	}

	if p.selectedAcct == nil {
		p.status = ports.PluginStatusError
		return fmt.Errorf("no Basecamp 3 accounts found")
	}

	p.status = ports.PluginStatusConnected
	return nil
}

// Disconnect closes the connection
func (p *Plugin) Disconnect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.status = ports.PluginStatusEnabled
	return nil
}

// Status returns current connection status
func (p *Plugin) Status() ports.PluginStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status
}

// GetAuthURL returns the OAuth2 authorization URL
func (p *Plugin) GetAuthURL(state string) string {
	params := url.Values{}
	params.Set("type", "web_server")
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", p.redirectURI)
	params.Set("state", state)

	return AuthURL + "?" + params.Encode()
}

// HandleAuthCallback processes the OAuth2 callback
func (p *Plugin) HandleAuthCallback(ctx context.Context, code string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	token, err := ExchangeCode(ctx, p.clientID, p.clientSecret, code, p.redirectURI)
	if err != nil {
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	// Create client with new token
	p.client = NewClient(token.AccessToken, 0)

	// Get authorization to find account ID
	auth, err := p.client.GetAuthorization(ctx)
	if err != nil {
		return fmt.Errorf("failed to get authorization: %w", err)
	}

	p.auth = auth

	// Select first bc3 account
	for _, acct := range auth.Accounts {
		if acct.Product == "bc3" {
			p.selectedAcct = &acct
			p.client.SetAccountID(acct.ID)
			break
		}
	}

	if p.selectedAcct == nil {
		return fmt.Errorf("no Basecamp 3 accounts found")
	}

	// Update credentials in config
	p.config.Credentials = map[string]string{
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"account_id":    strconv.FormatInt(p.selectedAcct.ID, 10),
	}

	p.status = ports.PluginStatusConnected
	return nil
}

// RefreshToken refreshes the OAuth2 token
func (p *Plugin) RefreshToken(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	refreshToken := p.config.Credentials["refresh_token"]
	if refreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	token, err := RefreshAccessToken(ctx, p.clientID, p.clientSecret, refreshToken)
	if err != nil {
		p.status = ports.PluginStatusAuthRequired
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	p.client.SetAccessToken(token.AccessToken)
	p.config.Credentials["access_token"] = token.AccessToken
	if token.RefreshToken != "" {
		p.config.Credentials["refresh_token"] = token.RefreshToken
	}

	return nil
}

// GetCredentials returns current credentials for storage
func (p *Plugin) GetCredentials() map[string]string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config.Credentials
}

// ============================================================================
// ProjectProvider implementation
// ============================================================================

// ListProjects returns all Basecamp projects
func (p *Plugin) ListProjects(ctx context.Context) ([]ports.ExternalProject, error) {
	p.mu.RLock()
	if p.client == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := p.client
	p.mu.RUnlock()

	projects, err := client.ListProjects(ctx, "")
	if err != nil {
		return nil, err
	}

	// Cache projects
	p.mu.Lock()
	p.projects = projects
	p.mu.Unlock()

	return p.convertProjects(projects), nil
}

// GetProject returns a specific project
func (p *Plugin) GetProject(ctx context.Context, projectID string) (*ports.ExternalProject, error) {
	p.mu.RLock()
	if p.client == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := p.client
	p.mu.RUnlock()

	id, err := strconv.ParseInt(projectID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %s", projectID)
	}

	project, err := client.GetProject(ctx, id)
	if err != nil {
		return nil, err
	}

	converted := p.convertProject(*project)
	return &converted, nil
}

// ============================================================================
// TaskProvider implementation
// ============================================================================

// ListTasks returns todos from a project
func (p *Plugin) ListTasks(ctx context.Context, projectID string, opts ports.TaskListOptions) ([]ports.ExternalTask, error) {
	p.mu.RLock()
	if p.client == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := p.client
	p.mu.RUnlock()

	id, err := strconv.ParseInt(projectID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %s", projectID)
	}

	// Get todo set for project
	todoSet, err := client.GetTodoSet(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get all todo lists
	lists, err := client.ListTodoLists(ctx, id, todoSet.ID, "")
	if err != nil {
		return nil, err
	}

	// Get todos from each list
	var allTasks []ports.ExternalTask
	for _, list := range lists {
		status := ""
		if opts.Status == "completed" {
			status = "completed"
		}

		todos, err := client.ListTodos(ctx, id, list.ID, status)
		if err != nil {
			continue // Skip on error
		}

		for _, todo := range todos {
			task := p.convertTodo(todo, list.Title)
			allTasks = append(allTasks, task)
		}
	}

	return allTasks, nil
}

// GetTask returns a specific todo
func (p *Plugin) GetTask(ctx context.Context, taskID string) (*ports.ExternalTask, error) {
	p.mu.RLock()
	if p.client == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := p.client
	p.mu.RUnlock()

	// taskID format: "projectID:todoID"
	parts := splitID(taskID)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid task ID format: %s", taskID)
	}

	projectID, _ := strconv.ParseInt(parts[0], 10, 64)
	todoID, _ := strconv.ParseInt(parts[1], 10, 64)

	todo, err := client.GetTodo(ctx, projectID, todoID)
	if err != nil {
		return nil, err
	}

	listName := ""
	if todo.Parent != nil {
		listName = todo.Parent.Title
	}

	task := p.convertTodo(*todo, listName)
	return &task, nil
}

// CreateTask creates a new todo
func (p *Plugin) CreateTask(ctx context.Context, task ports.ExternalTaskCreate) (*ports.ExternalTask, error) {
	p.mu.RLock()
	if p.client == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := p.client
	p.mu.RUnlock()

	projectID, _ := strconv.ParseInt(task.ProjectID, 10, 64)
	listID, _ := strconv.ParseInt(task.ListID, 10, 64)

	// If no list ID, get default todo list
	if listID == 0 {
		todoSet, err := client.GetTodoSet(ctx, projectID)
		if err != nil {
			return nil, err
		}
		lists, err := client.ListTodoLists(ctx, projectID, todoSet.ID, "")
		if err != nil || len(lists) == 0 {
			return nil, fmt.Errorf("no todo lists found in project")
		}
		listID = lists[0].ID
	}

	req := CreateTodoRequest{
		Content:     task.Title,
		Description: task.Description,
		Notify:      task.Notify,
	}
	if task.DueOn != nil {
		req.DueOn = task.DueOn.Format("2006-01-02")
	}

	todo, err := client.CreateTodo(ctx, projectID, listID, req)
	if err != nil {
		return nil, err
	}

	result := p.convertTodo(*todo, "")
	return &result, nil
}

// UpdateTask updates an existing todo
func (p *Plugin) UpdateTask(ctx context.Context, taskID string, update ports.ExternalTaskUpdate) (*ports.ExternalTask, error) {
	p.mu.RLock()
	if p.client == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := p.client
	p.mu.RUnlock()

	parts := splitID(taskID)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid task ID format: %s", taskID)
	}

	projectID, _ := strconv.ParseInt(parts[0], 10, 64)
	todoID, _ := strconv.ParseInt(parts[1], 10, 64)

	req := UpdateTodoRequest{
		Content:     update.Title,
		Description: update.Description,
	}
	if update.DueOn != nil {
		dueStr := update.DueOn.Format("2006-01-02")
		req.DueOn = &dueStr
	}

	todo, err := client.UpdateTodo(ctx, projectID, todoID, req)
	if err != nil {
		return nil, err
	}

	// Handle completion separately
	if update.Completed != nil {
		if *update.Completed {
			client.CompleteTodo(ctx, projectID, todoID)
		} else {
			client.UncompleteTodo(ctx, projectID, todoID)
		}
	}

	result := p.convertTodo(*todo, "")
	return &result, nil
}

// CompleteTask marks a todo as complete
func (p *Plugin) CompleteTask(ctx context.Context, taskID string) error {
	p.mu.RLock()
	if p.client == nil {
		p.mu.RUnlock()
		return fmt.Errorf("not connected")
	}
	client := p.client
	p.mu.RUnlock()

	parts := splitID(taskID)
	if len(parts) != 2 {
		return fmt.Errorf("invalid task ID format: %s", taskID)
	}

	projectID, _ := strconv.ParseInt(parts[0], 10, 64)
	todoID, _ := strconv.ParseInt(parts[1], 10, 64)

	return client.CompleteTodo(ctx, projectID, todoID)
}

// ============================================================================
// MessageProvider implementation
// ============================================================================

// ListMessages returns messages from a project
func (p *Plugin) ListMessages(ctx context.Context, projectID string, opts ports.MessageListOptions) ([]ports.ExternalMessage, error) {
	p.mu.RLock()
	if p.client == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := p.client
	p.mu.RUnlock()

	id, err := strconv.ParseInt(projectID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %s", projectID)
	}

	board, err := client.GetMessageBoard(ctx, id)
	if err != nil {
		return nil, err
	}

	messages, err := client.ListMessages(ctx, id, board.ID)
	if err != nil {
		return nil, err
	}

	return p.convertMessages(messages), nil
}

// GetMessage returns a specific message
func (p *Plugin) GetMessage(ctx context.Context, messageID string) (*ports.ExternalMessage, error) {
	p.mu.RLock()
	if p.client == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := p.client
	p.mu.RUnlock()

	parts := splitID(messageID)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid message ID format: %s", messageID)
	}

	projectID, _ := strconv.ParseInt(parts[0], 10, 64)
	msgID, _ := strconv.ParseInt(parts[1], 10, 64)

	msg, err := client.GetMessage(ctx, projectID, msgID)
	if err != nil {
		return nil, err
	}

	converted := p.convertMessage(*msg)
	return &converted, nil
}

// PostMessage creates a new message
func (p *Plugin) PostMessage(ctx context.Context, msg ports.ExternalMessageCreate) (*ports.ExternalMessage, error) {
	p.mu.RLock()
	if p.client == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := p.client
	p.mu.RUnlock()

	projectID, _ := strconv.ParseInt(msg.ProjectID, 10, 64)

	board, err := client.GetMessageBoard(ctx, projectID)
	if err != nil {
		return nil, err
	}

	req := CreateMessageRequest{
		Subject: msg.Subject,
		Content: msg.Content,
	}

	created, err := client.CreateMessage(ctx, projectID, board.ID, req)
	if err != nil {
		return nil, err
	}

	result := p.convertMessage(*created)
	return &result, nil
}

// ListComments returns comments on an item
func (p *Plugin) ListComments(ctx context.Context, parentID string) ([]ports.ExternalComment, error) {
	p.mu.RLock()
	if p.client == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := p.client
	p.mu.RUnlock()

	parts := splitID(parentID)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid parent ID format: %s", parentID)
	}

	projectID, _ := strconv.ParseInt(parts[0], 10, 64)
	recordingID, _ := strconv.ParseInt(parts[1], 10, 64)

	comments, err := client.ListComments(ctx, projectID, recordingID)
	if err != nil {
		return nil, err
	}

	return p.convertComments(comments, parentID), nil
}

// PostComment creates a new comment (not implemented - requires rich text API)
func (p *Plugin) PostComment(ctx context.Context, parentID string, content string) (*ports.ExternalComment, error) {
	return nil, fmt.Errorf("posting comments not implemented")
}

// ============================================================================
// CalendarProvider implementation
// ============================================================================

// ListEvents returns schedule entries from a project
func (p *Plugin) ListEvents(ctx context.Context, projectID string, opts ports.CalendarListOptions) ([]ports.ExternalEvent, error) {
	p.mu.RLock()
	if p.client == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := p.client
	p.mu.RUnlock()

	id, err := strconv.ParseInt(projectID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid project ID: %s", projectID)
	}

	schedule, err := client.GetSchedule(ctx, id)
	if err != nil {
		return nil, err
	}

	entries, err := client.ListScheduleEntries(ctx, id, schedule.ID)
	if err != nil {
		return nil, err
	}

	return p.convertScheduleEntries(entries), nil
}

// GetEvent returns a specific event (not implemented)
func (p *Plugin) GetEvent(ctx context.Context, eventID string) (*ports.ExternalEvent, error) {
	return nil, fmt.Errorf("get event not implemented")
}

// ============================================================================
// PeopleProvider implementation
// ============================================================================

// ListPeople returns people from a project
func (p *Plugin) ListPeople(ctx context.Context, projectID string) ([]ports.ExternalPerson, error) {
	p.mu.RLock()
	if p.client == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	client := p.client
	p.mu.RUnlock()

	var people []Person
	var err error

	if projectID == "" {
		people, err = client.ListPeople(ctx)
	} else {
		id, _ := strconv.ParseInt(projectID, 10, 64)
		people, err = client.ListProjectPeople(ctx, id)
	}

	if err != nil {
		return nil, err
	}

	return p.convertPeople(people), nil
}

// GetPerson returns a specific person (not implemented)
func (p *Plugin) GetPerson(ctx context.Context, personID string) (*ports.ExternalPerson, error) {
	return nil, fmt.Errorf("get person not implemented")
}

// ============================================================================
// SyncProvider implementation
// ============================================================================

// Sync fetches all data since lastSync
func (p *Plugin) Sync(ctx context.Context, lastSync *time.Time) (*ports.PluginSyncResult, error) {
	p.mu.RLock()
	if p.client == nil {
		p.mu.RUnlock()
		return nil, fmt.Errorf("not connected")
	}
	p.mu.RUnlock()

	// Fetch all projects
	projects, err := p.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	var items []ports.ExternalItem

	// Fetch tasks from each project
	for _, proj := range projects {
		tasks, err := p.ListTasks(ctx, proj.ID, ports.TaskListOptions{})
		if err != nil {
			continue
		}
		for _, task := range tasks {
			items = append(items, task.ToExternalItem())
		}

		// Fetch messages from each project
		messages, err := p.ListMessages(ctx, proj.ID, ports.MessageListOptions{})
		if err != nil {
			continue
		}
		for _, msg := range messages {
			items = append(items, msg.ToExternalItem())
		}
	}

	// Filter by lastSync if provided
	var newItems, updatedItems []ports.ExternalItem
	for _, item := range items {
		if lastSync == nil || item.CreatedAt.After(*lastSync) {
			newItems = append(newItems, item)
		} else if item.UpdatedAt.After(*lastSync) {
			updatedItems = append(updatedItems, item)
		}
	}

	return &ports.PluginSyncResult{
		NewItems:     newItems,
		UpdatedItems: updatedItems,
		SyncedAt:     time.Now(),
	}, nil
}

// ============================================================================
// Type conversion helpers
// ============================================================================

func (p *Plugin) convertProjects(projects []Project) []ports.ExternalProject {
	result := make([]ports.ExternalProject, len(projects))
	for i, proj := range projects {
		result[i] = p.convertProject(proj)
	}
	return result
}

func (p *Plugin) convertProject(proj Project) ports.ExternalProject {
	return ports.ExternalProject{
		ID:          strconv.FormatInt(proj.ID, 10),
		PluginID:    PluginID,
		Name:        proj.Name,
		Description: proj.Description,
		URL:         proj.AppURL,
		Status:      proj.Status,
		CreatedAt:   proj.CreatedAt,
		UpdatedAt:   proj.UpdatedAt,
	}
}

func (p *Plugin) convertTodo(todo Todo, listName string) ports.ExternalTask {
	task := ports.ExternalTask{
		ID:              makeID(todo.Bucket.ID, todo.ID),
		PluginID:        PluginID,
		ProjectID:       strconv.FormatInt(todo.Bucket.ID, 10),
		ProjectName:     todo.Bucket.Name,
		ListName:        listName,
		Title:           todo.Content,
		Description:     todo.Description,
		DescriptionHTML: todo.Description,
		URL:             todo.AppURL,
		Position:        todo.Position,
		CreatedAt:       todo.CreatedAt,
		UpdatedAt:       todo.UpdatedAt,
		CommentCount:    todo.CommentCount,
	}

	if todo.Completed {
		task.Status = "completed"
		task.CompletedAt = todo.CompletedAt
		if todo.Completer != nil {
			task.CompletedBy = &ports.ExternalPerson{
				ID:    strconv.FormatInt(todo.Completer.ID, 10),
				Name:  todo.Completer.Name,
				Email: todo.Completer.EmailAddress,
			}
		}
	} else {
		task.Status = "pending"
	}

	if todo.DueOn != nil {
		task.DueOn = todo.DueOn.ToTime()
	}
	if todo.StartsOn != nil {
		task.StartsOn = todo.StartsOn.ToTime()
	}

	if todo.Creator != nil {
		task.Creator = &ports.ExternalPerson{
			ID:        strconv.FormatInt(todo.Creator.ID, 10),
			Name:      todo.Creator.Name,
			Email:     todo.Creator.EmailAddress,
			AvatarURL: todo.Creator.AvatarURL,
		}
	}

	task.Assignees = make([]ports.ExternalPerson, len(todo.Assignees))
	for i, a := range todo.Assignees {
		task.Assignees[i] = ports.ExternalPerson{
			ID:        strconv.FormatInt(a.ID, 10),
			Name:      a.Name,
			Email:     a.EmailAddress,
			AvatarURL: a.AvatarURL,
		}
	}

	return task
}

func (p *Plugin) convertMessages(messages []Message) []ports.ExternalMessage {
	result := make([]ports.ExternalMessage, len(messages))
	for i, msg := range messages {
		result[i] = p.convertMessage(msg)
	}
	return result
}

func (p *Plugin) convertMessage(msg Message) ports.ExternalMessage {
	result := ports.ExternalMessage{
		ID:           makeID(msg.Bucket.ID, msg.ID),
		PluginID:     PluginID,
		ProjectID:    strconv.FormatInt(msg.Bucket.ID, 10),
		ProjectName:  msg.Bucket.Name,
		Subject:      msg.Subject,
		Content:      msg.Content,
		ContentHTML:  msg.Content,
		URL:          msg.AppURL,
		CreatedAt:    msg.CreatedAt,
		UpdatedAt:    msg.UpdatedAt,
		CommentCount: msg.CommentCount,
	}

	if msg.Category != nil {
		result.Category = msg.Category.Name
	}

	if msg.Creator != nil {
		result.Author = &ports.ExternalPerson{
			ID:        strconv.FormatInt(msg.Creator.ID, 10),
			Name:      msg.Creator.Name,
			Email:     msg.Creator.EmailAddress,
			AvatarURL: msg.Creator.AvatarURL,
		}
	}

	return result
}

func (p *Plugin) convertComments(comments []Comment, parentID string) []ports.ExternalComment {
	result := make([]ports.ExternalComment, len(comments))
	for i, c := range comments {
		result[i] = ports.ExternalComment{
			ID:          makeID(c.Bucket.ID, c.ID),
			PluginID:    PluginID,
			ParentID:    parentID,
			Content:     c.Content,
			ContentHTML: c.Content,
			URL:         c.AppURL,
			CreatedAt:   c.CreatedAt,
			UpdatedAt:   c.UpdatedAt,
		}
		if c.Creator != nil {
			result[i].Author = &ports.ExternalPerson{
				ID:        strconv.FormatInt(c.Creator.ID, 10),
				Name:      c.Creator.Name,
				Email:     c.Creator.EmailAddress,
				AvatarURL: c.Creator.AvatarURL,
			}
		}
	}
	return result
}

func (p *Plugin) convertScheduleEntries(entries []ScheduleEntry) []ports.ExternalEvent {
	result := make([]ports.ExternalEvent, len(entries))
	for i, e := range entries {
		result[i] = ports.ExternalEvent{
			ID:          makeID(e.Bucket.ID, e.ID),
			PluginID:    PluginID,
			ProjectID:   strconv.FormatInt(e.Bucket.ID, 10),
			ProjectName: e.Bucket.Name,
			Title:       e.Summary,
			Description: e.Description,
			URL:         e.AppURL,
			StartsAt:    e.StartsAt,
			EndsAt:      e.EndsAt,
			AllDay:      e.AllDay,
			Recurring:   e.RecurrenceSchedule != "",
			CreatedAt:   e.CreatedAt,
			UpdatedAt:   e.UpdatedAt,
		}
		if e.Creator != nil {
			result[i].Creator = &ports.ExternalPerson{
				ID:        strconv.FormatInt(e.Creator.ID, 10),
				Name:      e.Creator.Name,
				Email:     e.Creator.EmailAddress,
				AvatarURL: e.Creator.AvatarURL,
			}
		}
		result[i].Attendees = make([]ports.ExternalPerson, len(e.Participants))
		for j, p := range e.Participants {
			result[i].Attendees[j] = ports.ExternalPerson{
				ID:        strconv.FormatInt(p.ID, 10),
				Name:      p.Name,
				Email:     p.EmailAddress,
				AvatarURL: p.AvatarURL,
			}
		}
	}
	return result
}

func (p *Plugin) convertPeople(people []Person) []ports.ExternalPerson {
	result := make([]ports.ExternalPerson, len(people))
	for i, person := range people {
		result[i] = ports.ExternalPerson{
			ID:        strconv.FormatInt(person.ID, 10),
			PluginID:  PluginID,
			Name:      person.Name,
			Email:     person.EmailAddress,
			AvatarURL: person.AvatarURL,
			Title:     person.Title,
			Admin:     person.Admin,
			Owner:     person.Owner,
			CreatedAt: person.CreatedAt,
			UpdatedAt: person.UpdatedAt,
		}
	}
	return result
}

// Helper functions

func makeID(projectID, itemID int64) string {
	return fmt.Sprintf("%d:%d", projectID, itemID)
}

func splitID(id string) []string {
	return strings.Split(id, ":")
}

// strings package is imported at top
