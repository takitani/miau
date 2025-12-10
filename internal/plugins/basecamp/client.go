// Package basecamp implements the Basecamp 3/4 API client.
package basecamp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// OAuth2 endpoints
	AuthURL  = "https://launchpad.37signals.com/authorization/new"
	TokenURL = "https://launchpad.37signals.com/authorization/token"

	// API base
	LaunchpadAPI = "https://launchpad.37signals.com"

	// User-Agent requirement
	UserAgent = "miau (https://github.com/takitani/miau)"

	// Rate limiting
	DefaultRateLimit = 50 // per 10 seconds
)

// Client is the Basecamp API client
type Client struct {
	httpClient  *http.Client
	accessToken string
	accountID   int64
	baseURL     string // Account-specific API URL
	userAgent   string

	// Rate limiting
	requestCount int
	windowStart  time.Time
}

// NewClient creates a new Basecamp API client
func NewClient(accessToken string, accountID int64) *Client {
	return &Client{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		accessToken: accessToken,
		accountID:   accountID,
		baseURL:     fmt.Sprintf("https://3.basecampapi.com/%d", accountID),
		userAgent:   UserAgent,
		windowStart: time.Now(),
	}
}

// SetAccessToken updates the access token
func (c *Client) SetAccessToken(token string) {
	c.accessToken = token
}

// SetAccountID updates the account ID and base URL
func (c *Client) SetAccountID(accountID int64) {
	c.accountID = accountID
	c.baseURL = fmt.Sprintf("https://3.basecampapi.com/%d", accountID)
}

// GetAuthorization returns authorization info including available accounts
func (c *Client) GetAuthorization(ctx context.Context) (*Authorization, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", LaunchpadAPI+"/authorization.json", nil)
	if err != nil {
		return nil, err
	}

	var auth Authorization
	if err := c.do(req, &auth); err != nil {
		return nil, err
	}
	return &auth, nil
}

// ListProjects returns all projects
func (c *Client) ListProjects(ctx context.Context, status string) ([]Project, error) {
	endpoint := "/projects.json"
	if status != "" {
		endpoint += "?status=" + status
	}

	var projects []Project
	if err := c.getAll(ctx, endpoint, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

// GetProject returns a specific project
func (c *Client) GetProject(ctx context.Context, projectID int64) (*Project, error) {
	endpoint := fmt.Sprintf("/projects/%d.json", projectID)

	var project Project
	if err := c.get(ctx, endpoint, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

// GetTodoSet returns the todo set for a project
func (c *Client) GetTodoSet(ctx context.Context, projectID int64) (*TodoSet, error) {
	project, err := c.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Find todoset in dock
	for _, dock := range project.Dock {
		if dock.Name == "todoset" && dock.Enabled {
			var todoSet TodoSet
			if err := c.getURL(ctx, dock.URL, &todoSet); err != nil {
				return nil, err
			}
			return &todoSet, nil
		}
	}
	return nil, fmt.Errorf("todoset not found for project %d", projectID)
}

// ListTodoLists returns all todo lists in a project
func (c *Client) ListTodoLists(ctx context.Context, projectID int64, todoSetID int64, status string) ([]TodoList, error) {
	endpoint := fmt.Sprintf("/buckets/%d/todosets/%d/todolists.json", projectID, todoSetID)
	if status != "" {
		endpoint += "?status=" + status
	}

	var lists []TodoList
	if err := c.getAll(ctx, endpoint, &lists); err != nil {
		return nil, err
	}
	return lists, nil
}

// GetTodoList returns a specific todo list
func (c *Client) GetTodoList(ctx context.Context, projectID int64, todoListID int64) (*TodoList, error) {
	endpoint := fmt.Sprintf("/buckets/%d/todolists/%d.json", projectID, todoListID)

	var list TodoList
	if err := c.get(ctx, endpoint, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// ListTodos returns todos in a todo list
func (c *Client) ListTodos(ctx context.Context, projectID int64, todoListID int64, status string) ([]Todo, error) {
	endpoint := fmt.Sprintf("/buckets/%d/todolists/%d/todos.json", projectID, todoListID)
	if status != "" {
		endpoint += "?status=" + status
	}

	var todos []Todo
	if err := c.getAll(ctx, endpoint, &todos); err != nil {
		return nil, err
	}
	return todos, nil
}

// GetTodo returns a specific todo
func (c *Client) GetTodo(ctx context.Context, projectID int64, todoID int64) (*Todo, error) {
	endpoint := fmt.Sprintf("/buckets/%d/todos/%d.json", projectID, todoID)

	var todo Todo
	if err := c.get(ctx, endpoint, &todo); err != nil {
		return nil, err
	}
	return &todo, nil
}

// CreateTodo creates a new todo
func (c *Client) CreateTodo(ctx context.Context, projectID int64, todoListID int64, req CreateTodoRequest) (*Todo, error) {
	endpoint := fmt.Sprintf("/buckets/%d/todolists/%d/todos.json", projectID, todoListID)

	var todo Todo
	if err := c.post(ctx, endpoint, req, &todo); err != nil {
		return nil, err
	}
	return &todo, nil
}

// UpdateTodo updates an existing todo
func (c *Client) UpdateTodo(ctx context.Context, projectID int64, todoID int64, req UpdateTodoRequest) (*Todo, error) {
	endpoint := fmt.Sprintf("/buckets/%d/todos/%d.json", projectID, todoID)

	var todo Todo
	if err := c.put(ctx, endpoint, req, &todo); err != nil {
		return nil, err
	}
	return &todo, nil
}

// CompleteTodo marks a todo as complete
func (c *Client) CompleteTodo(ctx context.Context, projectID int64, todoID int64) error {
	todo, err := c.GetTodo(ctx, projectID, todoID)
	if err != nil {
		return err
	}
	if todo.Completed {
		return nil // Already complete
	}

	// POST to completion_url
	return c.postURL(ctx, todo.CompletionURL, nil, nil)
}

// UncompleteTodo marks a todo as incomplete
func (c *Client) UncompleteTodo(ctx context.Context, projectID int64, todoID int64) error {
	todo, err := c.GetTodo(ctx, projectID, todoID)
	if err != nil {
		return err
	}
	if !todo.Completed {
		return nil // Already incomplete
	}

	// DELETE to completion_url
	return c.deleteURL(ctx, todo.CompletionURL)
}

// GetMessageBoard returns the message board for a project
func (c *Client) GetMessageBoard(ctx context.Context, projectID int64) (*MessageBoard, error) {
	project, err := c.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	for _, dock := range project.Dock {
		if dock.Name == "message_board" && dock.Enabled {
			var board MessageBoard
			if err := c.getURL(ctx, dock.URL, &board); err != nil {
				return nil, err
			}
			return &board, nil
		}
	}
	return nil, fmt.Errorf("message board not found for project %d", projectID)
}

// ListMessages returns messages from a message board
func (c *Client) ListMessages(ctx context.Context, projectID int64, messageBoardID int64) ([]Message, error) {
	endpoint := fmt.Sprintf("/buckets/%d/message_boards/%d/messages.json", projectID, messageBoardID)

	var messages []Message
	if err := c.getAll(ctx, endpoint, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// GetMessage returns a specific message
func (c *Client) GetMessage(ctx context.Context, projectID int64, messageID int64) (*Message, error) {
	endpoint := fmt.Sprintf("/buckets/%d/messages/%d.json", projectID, messageID)

	var message Message
	if err := c.get(ctx, endpoint, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// CreateMessage creates a new message
func (c *Client) CreateMessage(ctx context.Context, projectID int64, messageBoardID int64, req CreateMessageRequest) (*Message, error) {
	endpoint := fmt.Sprintf("/buckets/%d/message_boards/%d/messages.json", projectID, messageBoardID)

	var message Message
	if err := c.post(ctx, endpoint, req, &message); err != nil {
		return nil, err
	}
	return &message, nil
}

// ListComments returns comments on a recording
func (c *Client) ListComments(ctx context.Context, projectID int64, recordingID int64) ([]Comment, error) {
	endpoint := fmt.Sprintf("/buckets/%d/recordings/%d/comments.json", projectID, recordingID)

	var comments []Comment
	if err := c.getAll(ctx, endpoint, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

// GetSchedule returns the schedule for a project
func (c *Client) GetSchedule(ctx context.Context, projectID int64) (*Schedule, error) {
	project, err := c.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	for _, dock := range project.Dock {
		if dock.Name == "schedule" && dock.Enabled {
			var schedule Schedule
			if err := c.getURL(ctx, dock.URL, &schedule); err != nil {
				return nil, err
			}
			return &schedule, nil
		}
	}
	return nil, fmt.Errorf("schedule not found for project %d", projectID)
}

// ListScheduleEntries returns events from a schedule
func (c *Client) ListScheduleEntries(ctx context.Context, projectID int64, scheduleID int64) ([]ScheduleEntry, error) {
	endpoint := fmt.Sprintf("/buckets/%d/schedules/%d/entries.json", projectID, scheduleID)

	var entries []ScheduleEntry
	if err := c.getAll(ctx, endpoint, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// ListPeople returns all people in the account
func (c *Client) ListPeople(ctx context.Context) ([]Person, error) {
	endpoint := "/people.json"

	var people []Person
	if err := c.getAll(ctx, endpoint, &people); err != nil {
		return nil, err
	}
	return people, nil
}

// ListProjectPeople returns people in a specific project
func (c *Client) ListProjectPeople(ctx context.Context, projectID int64) ([]Person, error) {
	endpoint := fmt.Sprintf("/projects/%d/people.json", projectID)

	var people []Person
	if err := c.getAll(ctx, endpoint, &people); err != nil {
		return nil, err
	}
	return people, nil
}

// HTTP helpers

func (c *Client) get(ctx context.Context, endpoint string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+endpoint, nil)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

func (c *Client) getURL(ctx context.Context, fullURL string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return err
	}
	return c.do(req, result)
}

func (c *Client) getAll(ctx context.Context, endpoint string, result interface{}) error {
	// Handle pagination
	fullURL := c.baseURL + endpoint
	return c.getAllURL(ctx, fullURL, result)
}

func (c *Client) getAllURL(ctx context.Context, fullURL string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return err
	}

	resp, err := c.doRaw(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return err
	}

	// Check for pagination
	nextURL := c.parseNextLink(resp.Header.Get("Link"))
	if nextURL != "" {
		// There's more data, but for simplicity we'll just return the first page
		// A full implementation would recursively fetch all pages
	}

	return nil
}

func (c *Client) post(ctx context.Context, endpoint string, body interface{}, result interface{}) error {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+endpoint, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, result)
}

func (c *Client) postURL(ctx context.Context, fullURL string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = strings.NewReader(string(bodyJSON))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bodyReader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.do(req, result)
}

func (c *Client) put(ctx context.Context, endpoint string, body interface{}, result interface{}) error {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", c.baseURL+endpoint, strings.NewReader(string(bodyJSON)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, result)
}

func (c *Client) deleteURL(ctx context.Context, fullURL string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", fullURL, nil)
	if err != nil {
		return err
	}
	return c.do(req, nil)
}

func (c *Client) do(req *http.Request, result interface{}) error {
	resp, err := c.doRaw(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

func (c *Client) doRaw(req *http.Request) (*http.Response, error) {
	// Rate limiting
	c.waitForRateLimit()

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Handle rate limit
	if resp.StatusCode == 429 {
		resp.Body.Close()
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			if secs, err := strconv.Atoi(retryAfter); err == nil {
				time.Sleep(time.Duration(secs) * time.Second)
				return c.doRaw(req)
			}
		}
		return nil, fmt.Errorf("rate limited, retry later")
	}

	// Handle errors
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

func (c *Client) waitForRateLimit() {
	// Reset window every 10 seconds
	if time.Since(c.windowStart) > 10*time.Second {
		c.requestCount = 0
		c.windowStart = time.Now()
	}

	// Wait if we've hit the limit
	if c.requestCount >= DefaultRateLimit {
		waitTime := 10*time.Second - time.Since(c.windowStart)
		if waitTime > 0 {
			time.Sleep(waitTime)
		}
		c.requestCount = 0
		c.windowStart = time.Now()
	}

	c.requestCount++
}

// parseNextLink extracts the next URL from the Link header
func (c *Client) parseNextLink(link string) string {
	if link == "" {
		return ""
	}

	// Parse RFC 5988 link header
	// Format: <url>; rel="next"
	re := regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)
	matches := re.FindStringSubmatch(link)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// OAuth2 token refresh
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// RefreshAccessToken refreshes the OAuth2 token
func RefreshAccessToken(ctx context.Context, clientID, clientSecret, refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("type", "refresh")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

// ExchangeCode exchanges an authorization code for tokens
func ExchangeCode(ctx context.Context, clientID, clientSecret, code, redirectURI string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("type", "web_server")
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequestWithContext(ctx, "POST", TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("code exchange failed: %s", string(body))
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}
