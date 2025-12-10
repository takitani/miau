// Package basecamp provides a client for the Basecamp 3/4 API
package basecamp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/opik/miau/internal/ports"
	"golang.org/x/oauth2"
)

const (
	// UserAgent is required by Basecamp API
	UserAgent = "miau (https://github.com/opik/miau)"
)

// Client is the Basecamp API client
type Client struct {
	httpClient *http.Client
	baseURL    string // e.g., "https://3.basecampapi.com/12345678"
	accountID  string
}

// NewClient creates a new Basecamp API client
func NewClient(token *oauth2.Token, accountID string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:   fmt.Sprintf("https://3.basecampapi.com/%s", accountID),
		accountID: accountID,
	}
}

// NewClientWithHTTP creates a client with a custom HTTP client (for OAuth2 auto-refresh)
func NewClientWithHTTP(httpClient *http.Client, accountID string) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    fmt.Sprintf("https://3.basecampapi.com/%s", accountID),
		accountID:  accountID,
	}
}

// request makes an HTTP request to the Basecamp API
func (c *Client) request(method, path string, body interface{}) (*http.Response, error) {
	var url = c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		var jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	var req, err = http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Content-Type", "application/json")

	var resp, respErr = c.httpClient.Do(req)
	if respErr != nil {
		return nil, respErr
	}

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		var retryAfter = resp.Header.Get("Retry-After")
		var seconds, _ = strconv.Atoi(retryAfter)
		if seconds == 0 {
			seconds = 10
		}
		resp.Body.Close()
		return nil, fmt.Errorf("rate limited, retry after %d seconds", seconds)
	}

	return resp, nil
}

// get makes a GET request
func (c *Client) get(path string, result interface{}) error {
	var resp, err = c.request("GET", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

// post makes a POST request
func (c *Client) post(path string, body, result interface{}) error {
	var resp, err = c.request("POST", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var respBody, _ = io.ReadAll(resp.Body)
		return fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// put makes a PUT request
func (c *Client) put(path string, body, result interface{}) error {
	var resp, err = c.request("PUT", path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var respBody, _ = io.ReadAll(resp.Body)
		return fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

// =============================================================================
// PROJECTS
// =============================================================================

// ListProjects returns all projects
func (c *Client) ListProjects() ([]ports.BasecampProject, error) {
	var projects []ports.BasecampProject
	if err := c.get("/projects.json", &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

// GetProject returns a single project by ID
func (c *Client) GetProject(projectID int64) (*ports.BasecampProject, error) {
	var project ports.BasecampProject
	if err := c.get(fmt.Sprintf("/projects/%d.json", projectID), &project); err != nil {
		return nil, err
	}
	return &project, nil
}

// =============================================================================
// TO-DO SETS & LISTS
// =============================================================================

// GetTodoSet returns the todo set for a project
func (c *Client) GetTodoSet(projectID int64) (*ports.BasecampTodoSet, error) {
	// First, get the project to find the todoset URL
	var project, err = c.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	// Find todoset in dock
	for _, dock := range project.Dock {
		if dock.Name == "todoset" && dock.Enabled {
			var todoSet ports.BasecampTodoSet
			// Extract the todoset path from the URL
			var path = fmt.Sprintf("/buckets/%d/todosets/%d.json", projectID, dock.ID)
			if err := c.get(path, &todoSet); err != nil {
				return nil, err
			}
			return &todoSet, nil
		}
	}

	return nil, fmt.Errorf("todoset not found for project %d", projectID)
}

// ListTodoLists returns all todo lists in a todoset
func (c *Client) ListTodoLists(todoSetID int64) ([]ports.BasecampTodoList, error) {
	// Note: todoSetID is actually the bucket/project ID in this context
	var lists []ports.BasecampTodoList
	var path = fmt.Sprintf("/buckets/%d/todosets/%d/todolists.json", todoSetID, todoSetID)
	if err := c.get(path, &lists); err != nil {
		return nil, err
	}
	return lists, nil
}

// GetTodoList returns a single todo list
func (c *Client) GetTodoList(todoListID int64) (*ports.BasecampTodoList, error) {
	// We need the project ID to build the correct path
	// This is a limitation - caller needs to know the project
	return nil, fmt.Errorf("GetTodoList requires project context - use GetTodoListInProject")
}

// GetTodoListInProject returns a todo list within a specific project
func (c *Client) GetTodoListInProject(projectID, todoListID int64) (*ports.BasecampTodoList, error) {
	var list ports.BasecampTodoList
	var path = fmt.Sprintf("/buckets/%d/todolists/%d.json", projectID, todoListID)
	if err := c.get(path, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// =============================================================================
// TO-DOS
// =============================================================================

// ListTodos returns all todos in a todolist
func (c *Client) ListTodos(todoListID int64) ([]ports.BasecampTodo, error) {
	// This requires project context - simplified version
	return nil, fmt.Errorf("ListTodos requires project context - use ListTodosInProject")
}

// ListTodosInProject returns todos in a todolist within a project
func (c *Client) ListTodosInProject(projectID, todoListID int64) ([]ports.BasecampTodo, error) {
	var todos []ports.BasecampTodo
	var path = fmt.Sprintf("/buckets/%d/todolists/%d/todos.json", projectID, todoListID)
	if err := c.get(path, &todos); err != nil {
		return nil, err
	}
	return todos, nil
}

// GetTodo returns a single todo
func (c *Client) GetTodo(todoID int64) (*ports.BasecampTodo, error) {
	return nil, fmt.Errorf("GetTodo requires project context - use GetTodoInProject")
}

// GetTodoInProject returns a todo within a project
func (c *Client) GetTodoInProject(projectID, todoID int64) (*ports.BasecampTodo, error) {
	var todo ports.BasecampTodo
	var path = fmt.Sprintf("/buckets/%d/todos/%d.json", projectID, todoID)
	if err := c.get(path, &todo); err != nil {
		return nil, err
	}
	return &todo, nil
}

// CreateTodo creates a new todo in a todolist
func (c *Client) CreateTodo(todoListID int64, content string, assigneeIDs []int64, dueDate *time.Time) (*ports.BasecampTodo, error) {
	return nil, fmt.Errorf("CreateTodo requires project context - use CreateTodoInProject")
}

// CreateTodoInProject creates a new todo in a project's todolist
func (c *Client) CreateTodoInProject(projectID, todoListID int64, content string, assigneeIDs []int64, dueDate *time.Time) (*ports.BasecampTodo, error) {
	var body = map[string]interface{}{
		"content": content,
	}

	if len(assigneeIDs) > 0 {
		body["assignee_ids"] = assigneeIDs
	}

	if dueDate != nil {
		body["due_on"] = dueDate.Format("2006-01-02")
	}

	var todo ports.BasecampTodo
	var path = fmt.Sprintf("/buckets/%d/todolists/%d/todos.json", projectID, todoListID)
	if err := c.post(path, body, &todo); err != nil {
		return nil, err
	}
	return &todo, nil
}

// UpdateTodo updates an existing todo
func (c *Client) UpdateTodo(todoID int64, content string, assigneeIDs []int64, dueDate *time.Time) (*ports.BasecampTodo, error) {
	return nil, fmt.Errorf("UpdateTodo requires project context - use UpdateTodoInProject")
}

// UpdateTodoInProject updates a todo within a project
func (c *Client) UpdateTodoInProject(projectID, todoID int64, content string, assigneeIDs []int64, dueDate *time.Time) (*ports.BasecampTodo, error) {
	var body = map[string]interface{}{
		"content": content,
	}

	if len(assigneeIDs) > 0 {
		body["assignee_ids"] = assigneeIDs
	}

	if dueDate != nil {
		body["due_on"] = dueDate.Format("2006-01-02")
	} else {
		body["due_on"] = nil
	}

	var todo ports.BasecampTodo
	var path = fmt.Sprintf("/buckets/%d/todos/%d.json", projectID, todoID)
	if err := c.put(path, body, &todo); err != nil {
		return nil, err
	}
	return &todo, nil
}

// CompleteTodo marks a todo as completed
func (c *Client) CompleteTodo(todoID int64) error {
	return fmt.Errorf("CompleteTodo requires project context - use CompleteTodoInProject")
}

// CompleteTodoInProject marks a todo as completed
func (c *Client) CompleteTodoInProject(projectID, todoID int64) error {
	var path = fmt.Sprintf("/buckets/%d/todos/%d/completion.json", projectID, todoID)
	var resp, err = c.request("POST", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return fmt.Errorf("failed to complete todo: status %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}

// UncompleteTodo marks a todo as not completed
func (c *Client) UncompleteTodo(todoID int64) error {
	return fmt.Errorf("UncompleteTodo requires project context - use UncompleteTodoInProject")
}

// UncompleteTodoInProject marks a todo as not completed
func (c *Client) UncompleteTodoInProject(projectID, todoID int64) error {
	var path = fmt.Sprintf("/buckets/%d/todos/%d/completion.json", projectID, todoID)
	var req, err = http.NewRequest("DELETE", c.baseURL+path, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", UserAgent)

	var resp, respErr = c.httpClient.Do(req)
	if respErr != nil {
		return respErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return fmt.Errorf("failed to uncomplete todo: status %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}

// =============================================================================
// MESSAGE BOARD
// =============================================================================

// GetMessageBoard returns the message board for a project
func (c *Client) GetMessageBoard(projectID int64) (*ports.BasecampMessageBoard, error) {
	var project, err = c.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	for _, dock := range project.Dock {
		if dock.Name == "message_board" && dock.Enabled {
			var board ports.BasecampMessageBoard
			var path = fmt.Sprintf("/buckets/%d/message_boards/%d.json", projectID, dock.ID)
			if err := c.get(path, &board); err != nil {
				return nil, err
			}
			return &board, nil
		}
	}

	return nil, fmt.Errorf("message board not found for project %d", projectID)
}

// ListMessages returns messages from a message board
func (c *Client) ListMessages(messageBoardID int64, page int) ([]ports.BasecampMessage, error) {
	return nil, fmt.Errorf("ListMessages requires project context - use ListMessagesInProject")
}

// ListMessagesInProject returns messages from a project's message board
func (c *Client) ListMessagesInProject(projectID, messageBoardID int64, page int) ([]ports.BasecampMessage, error) {
	var messages []ports.BasecampMessage
	var path = fmt.Sprintf("/buckets/%d/message_boards/%d/messages.json?page=%d", projectID, messageBoardID, page)
	if err := c.get(path, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// CreateMessage creates a new message on a message board
func (c *Client) CreateMessage(messageBoardID int64, subject, content string) (*ports.BasecampMessage, error) {
	return nil, fmt.Errorf("CreateMessage requires project context - use CreateMessageInProject")
}

// CreateMessageInProject creates a message on a project's message board
func (c *Client) CreateMessageInProject(projectID, messageBoardID int64, subject, content string) (*ports.BasecampMessage, error) {
	var body = map[string]interface{}{
		"subject": subject,
		"content": content,
		"status":  "active",
	}

	var message ports.BasecampMessage
	var path = fmt.Sprintf("/buckets/%d/message_boards/%d/messages.json", projectID, messageBoardID)
	if err := c.post(path, body, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// =============================================================================
// COMMENTS
// =============================================================================

// ListComments returns comments on a recording (todo, message, etc.)
func (c *Client) ListComments(recordingID int64) ([]ports.BasecampComment, error) {
	return nil, fmt.Errorf("ListComments requires project context - use ListCommentsInProject")
}

// ListCommentsInProject returns comments on a recording within a project
func (c *Client) ListCommentsInProject(projectID, recordingID int64) ([]ports.BasecampComment, error) {
	var comments []ports.BasecampComment
	var path = fmt.Sprintf("/buckets/%d/recordings/%d/comments.json", projectID, recordingID)
	if err := c.get(path, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// CreateComment creates a comment on a recording
func (c *Client) CreateComment(recordingID int64, content string) (*ports.BasecampComment, error) {
	return nil, fmt.Errorf("CreateComment requires project context - use CreateCommentInProject")
}

// CreateCommentInProject creates a comment on a recording
func (c *Client) CreateCommentInProject(projectID, recordingID int64, content string) (*ports.BasecampComment, error) {
	var body = map[string]interface{}{
		"content": content,
	}

	var comment ports.BasecampComment
	var path = fmt.Sprintf("/buckets/%d/recordings/%d/comments.json", projectID, recordingID)
	if err := c.post(path, body, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

// =============================================================================
// PEOPLE
// =============================================================================

// ListPeople returns all people in the Basecamp account
func (c *Client) ListPeople() ([]ports.BasecampPerson, error) {
	var people []ports.BasecampPerson
	if err := c.get("/people.json", &people); err != nil {
		return nil, err
	}
	return people, nil
}

// GetPerson returns a single person by ID
func (c *Client) GetPerson(personID int64) (*ports.BasecampPerson, error) {
	var person ports.BasecampPerson
	if err := c.get(fmt.Sprintf("/people/%d.json", personID), &person); err != nil {
		return nil, err
	}
	return &person, nil
}

// GetMyInfo returns the authenticated user's info
func (c *Client) GetMyInfo() (*ports.BasecampPerson, error) {
	var person ports.BasecampPerson
	if err := c.get("/my/profile.json", &person); err != nil {
		return nil, err
	}
	return &person, nil
}
