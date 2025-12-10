package desktop

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/opik/miau/internal/app"
	"github.com/opik/miau/internal/auth"
	"github.com/opik/miau/internal/basecamp"
	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/services"
)

// ============================================================================
// BASECAMP BINDINGS
// ============================================================================

// IsBasecampConnected returns true if Basecamp is connected
func (a *App) IsBasecampConnected() bool {
	if a.application == nil {
		return false
	}
	var basecampService = a.application.Basecamp()
	if basecampService == nil {
		return false
	}
	return basecampService.IsConnected()
}

// GetBasecampConfig returns the current Basecamp configuration
func (a *App) GetBasecampConfig() (*BasecampConfigDTO, error) {
	if a.cfg == nil || a.cfg.Basecamp == nil {
		return &BasecampConfigDTO{
			Enabled:   false,
			Connected: false,
		}, nil
	}

	var bc = a.cfg.Basecamp
	var connected = a.IsBasecampConnected()

	// Mask client secret for display
	var maskedSecret = ""
	if bc.ClientSecret != "" {
		maskedSecret = "••••••••"
	}

	return &BasecampConfigDTO{
		Enabled:      bc.Enabled,
		ClientID:     bc.ClientID,
		ClientSecret: maskedSecret,
		AccountID:    bc.AccountID,
		Connected:    connected,
	}, nil
}

// SaveBasecampConfig saves Basecamp configuration
func (a *App) SaveBasecampConfig(cfg BasecampConfigDTO) error {
	if a.cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	// Create or update Basecamp config
	if a.cfg.Basecamp == nil {
		a.cfg.Basecamp = &config.BasecampConfig{}
	}

	a.cfg.Basecamp.Enabled = cfg.Enabled
	a.cfg.Basecamp.ClientID = cfg.ClientID

	// Only update client secret if it's not masked
	if cfg.ClientSecret != "" && cfg.ClientSecret != "••••••••" {
		a.cfg.Basecamp.ClientSecret = cfg.ClientSecret
	}

	a.cfg.Basecamp.AccountID = cfg.AccountID

	// Save config to file
	if err := config.Save(a.cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// AuthenticateBasecamp starts the OAuth2 flow for Basecamp
func (a *App) AuthenticateBasecamp() ([]BasecampAccountDTO, error) {
	if a.cfg == nil || a.cfg.Basecamp == nil {
		return nil, fmt.Errorf("basecamp not configured")
	}

	var bc = a.cfg.Basecamp
	if bc.ClientID == "" || bc.ClientSecret == "" {
		return nil, fmt.Errorf("basecamp client_id and client_secret are required")
	}

	// Create OAuth2 config
	var oauthConfig = auth.GetBasecampOAuth2Config(bc.ClientID, bc.ClientSecret)

	// Start browser-based authentication
	var token, err = auth.AuthenticateBasecampWithBrowser(oauthConfig)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Save token
	var tokenPath = auth.GetBasecampTokenPath(config.GetConfigPath())
	if err := auth.SaveBasecampToken(tokenPath, token); err != nil {
		return nil, fmt.Errorf("failed to save token: %w", err)
	}

	// Get available accounts
	var authResp, authErr = auth.GetBasecampAccounts(token)
	if authErr != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", authErr)
	}

	// Convert to DTOs - only return Basecamp 3/4 accounts
	var accounts []BasecampAccountDTO
	for _, acc := range authResp.Accounts {
		if acc.Product == "bc3" {
			accounts = append(accounts, BasecampAccountDTO{
				ID:   acc.ID,
				Name: acc.Name,
				Href: acc.Href,
			})
		}
	}

	return accounts, nil
}

// SelectBasecampAccount selects a Basecamp account and connects
func (a *App) SelectBasecampAccount(accountID int64) error {
	if a.cfg == nil || a.cfg.Basecamp == nil {
		return fmt.Errorf("basecamp not configured")
	}

	// Save the account ID to config
	a.cfg.Basecamp.AccountID = strconv.FormatInt(accountID, 10)
	a.cfg.Basecamp.Enabled = true

	if err := config.Save(a.cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Connect to Basecamp
	return a.ConnectBasecamp()
}

// ConnectBasecamp connects to Basecamp using saved credentials
func (a *App) ConnectBasecamp() error {
	if a.cfg == nil || a.cfg.Basecamp == nil || !a.cfg.Basecamp.Enabled {
		return fmt.Errorf("basecamp not enabled")
	}

	var bc = a.cfg.Basecamp
	if bc.ClientID == "" || bc.ClientSecret == "" || bc.AccountID == "" {
		return fmt.Errorf("basecamp not fully configured")
	}

	// Load token
	var tokenPath = auth.GetBasecampTokenPath(config.GetConfigPath())
	var oauthConfig = auth.GetBasecampOAuth2Config(bc.ClientID, bc.ClientSecret)

	var token, err = auth.GetValidBasecampToken(oauthConfig, tokenPath)
	if err != nil {
		return fmt.Errorf("failed to load token: %w", err)
	}

	// Create OAuth2 HTTP client (auto-refreshes token)
	var ctx = context.Background()
	var httpClient = oauthConfig.Client(ctx, token)

	// Create Basecamp client
	var client = basecamp.NewClientWithHTTP(httpClient, bc.AccountID)

	// Get the Basecamp service and set the client
	if a.application == nil {
		return fmt.Errorf("application not initialized")
	}

	var coreApp, ok = a.application.(*app.Application)
	if !ok {
		return fmt.Errorf("invalid application type")
	}

	var basecampService = coreApp.Basecamp().(*services.BasecampService)
	basecampService.SetClient(client)

	// Test connection
	if err := basecampService.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	log.Printf("[ConnectBasecamp] Connected to Basecamp account %s", bc.AccountID)
	return nil
}

// DisconnectBasecamp disconnects from Basecamp
func (a *App) DisconnectBasecamp() error {
	if a.application == nil {
		return nil
	}

	var basecampService = a.application.Basecamp()
	if basecampService == nil {
		return nil
	}

	return basecampService.Disconnect(context.Background())
}

// ============================================================================
// BASECAMP PROJECTS
// ============================================================================

// GetBasecampProjects returns all Basecamp projects
func (a *App) GetBasecampProjects() ([]BasecampProjectDTO, error) {
	if a.application == nil {
		return nil, fmt.Errorf("application not initialized")
	}

	var basecampService = a.application.Basecamp()
	if basecampService == nil || !basecampService.IsConnected() {
		return nil, fmt.Errorf("basecamp not connected")
	}

	var ctx = context.Background()
	var projects, err = basecampService.GetProjects(ctx)
	if err != nil {
		return nil, err
	}

	var result []BasecampProjectDTO
	for _, p := range projects {
		result = append(result, BasecampProjectDTO{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Status:      p.Status,
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		})
	}

	return result, nil
}

// ============================================================================
// BASECAMP TODOS
// ============================================================================

// GetBasecampTodoLists returns todo lists for a project
func (a *App) GetBasecampTodoLists(projectID int64) ([]BasecampTodoListDTO, error) {
	if a.application == nil {
		return nil, fmt.Errorf("application not initialized")
	}

	var basecampService = a.application.Basecamp()
	if basecampService == nil || !basecampService.IsConnected() {
		return nil, fmt.Errorf("basecamp not connected")
	}

	var ctx = context.Background()
	var lists, err = basecampService.GetTodoLists(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var result []BasecampTodoListDTO
	for _, l := range lists {
		result = append(result, BasecampTodoListDTO{
			ID:             l.ID,
			ProjectID:      l.ProjectID,
			Title:          l.Title,
			Description:    l.Description,
			Completed:      l.Completed,
			CompletedRatio: l.CompletedRatio,
			TodosCount:     l.TodosCount,
			CompletedCount: l.CompletedCount,
			CreatedAt:      l.CreatedAt,
			UpdatedAt:      l.UpdatedAt,
		})
	}

	return result, nil
}

// GetBasecampTodos returns todos for a todo list
func (a *App) GetBasecampTodos(projectID, todoListID int64) ([]BasecampTodoDTO, error) {
	if a.application == nil {
		return nil, fmt.Errorf("application not initialized")
	}

	var basecampService = a.application.Basecamp().(*services.BasecampService)
	if basecampService == nil || !basecampService.IsConnected() {
		return nil, fmt.Errorf("basecamp not connected")
	}

	var ctx = context.Background()
	var todos, err = basecampService.GetTodosInProject(ctx, projectID, todoListID)
	if err != nil {
		return nil, err
	}

	var result []BasecampTodoDTO
	for _, t := range todos {
		result = append(result, a.basecampTodoToDTO(&t))
	}

	return result, nil
}

// CreateBasecampTodo creates a new todo
func (a *App) CreateBasecampTodo(input BasecampTodoInputDTO) (*BasecampTodoDTO, error) {
	if a.application == nil {
		return nil, fmt.Errorf("application not initialized")
	}

	var basecampService = a.application.Basecamp().(*services.BasecampService)
	if basecampService == nil || !basecampService.IsConnected() {
		return nil, fmt.Errorf("basecamp not connected")
	}

	var ctx = context.Background()
	var todoInput = &ports.BasecampTodoInput{
		TodoListID:  input.TodoListID,
		Content:     input.Content,
		Description: input.Description,
		DueDate:     input.DueDate,
		AssigneeIDs: input.AssigneeIDs,
	}

	var todo, err = basecampService.CreateTodoInProject(ctx, input.ProjectID, todoInput)
	if err != nil {
		return nil, err
	}

	var dto = a.basecampTodoToDTO(todo)
	return &dto, nil
}

// UpdateBasecampTodo updates an existing todo
func (a *App) UpdateBasecampTodo(input BasecampTodoInputDTO) (*BasecampTodoDTO, error) {
	if a.application == nil {
		return nil, fmt.Errorf("application not initialized")
	}

	var basecampService = a.application.Basecamp().(*services.BasecampService)
	if basecampService == nil || !basecampService.IsConnected() {
		return nil, fmt.Errorf("basecamp not connected")
	}

	var ctx = context.Background()
	var todoInput = &ports.BasecampTodoInput{
		ID:          input.ID,
		TodoListID:  input.TodoListID,
		Content:     input.Content,
		Description: input.Description,
		DueDate:     input.DueDate,
		AssigneeIDs: input.AssigneeIDs,
	}

	var todo, err = basecampService.UpdateTodoInProject(ctx, input.ProjectID, todoInput)
	if err != nil {
		return nil, err
	}

	var dto = a.basecampTodoToDTO(todo)
	return &dto, nil
}

// CompleteBasecampTodo marks a todo as completed
func (a *App) CompleteBasecampTodo(projectID, todoID int64) error {
	if a.application == nil {
		return fmt.Errorf("application not initialized")
	}

	var basecampService = a.application.Basecamp().(*services.BasecampService)
	if basecampService == nil || !basecampService.IsConnected() {
		return fmt.Errorf("basecamp not connected")
	}

	var ctx = context.Background()
	return basecampService.CompleteTodoInProject(ctx, projectID, todoID)
}

// UncompleteBasecampTodo marks a todo as not completed
func (a *App) UncompleteBasecampTodo(projectID, todoID int64) error {
	if a.application == nil {
		return fmt.Errorf("application not initialized")
	}

	var basecampService = a.application.Basecamp().(*services.BasecampService)
	if basecampService == nil || !basecampService.IsConnected() {
		return fmt.Errorf("basecamp not connected")
	}

	var ctx = context.Background()
	return basecampService.UncompleteTodoInProject(ctx, projectID, todoID)
}

// ============================================================================
// BASECAMP MESSAGES
// ============================================================================

// GetBasecampMessages returns messages for a project
func (a *App) GetBasecampMessages(projectID int64, limit int) ([]BasecampMessageDTO, error) {
	if a.application == nil {
		return nil, fmt.Errorf("application not initialized")
	}

	var basecampService = a.application.Basecamp()
	if basecampService == nil || !basecampService.IsConnected() {
		return nil, fmt.Errorf("basecamp not connected")
	}

	var ctx = context.Background()
	var messages, err = basecampService.GetMessages(ctx, projectID, limit)
	if err != nil {
		return nil, err
	}

	var result []BasecampMessageDTO
	for _, m := range messages {
		result = append(result, a.basecampMessageToDTO(&m))
	}

	return result, nil
}

// PostBasecampMessage posts a new message to a project
func (a *App) PostBasecampMessage(projectID int64, subject, content string) (*BasecampMessageDTO, error) {
	if a.application == nil {
		return nil, fmt.Errorf("application not initialized")
	}

	var basecampService = a.application.Basecamp()
	if basecampService == nil || !basecampService.IsConnected() {
		return nil, fmt.Errorf("basecamp not connected")
	}

	var ctx = context.Background()
	var input = &ports.BasecampMessageInput{
		ProjectID: projectID,
		Subject:   subject,
		Content:   content,
	}

	var message, err = basecampService.PostMessage(ctx, input)
	if err != nil {
		return nil, err
	}

	var dto = a.basecampMessageToDTO(message)
	return &dto, nil
}

// ============================================================================
// BASECAMP PEOPLE
// ============================================================================

// GetBasecampPeople returns all people in the Basecamp account
func (a *App) GetBasecampPeople() ([]BasecampPersonDTO, error) {
	if a.application == nil {
		return nil, fmt.Errorf("application not initialized")
	}

	var basecampService = a.application.Basecamp()
	if basecampService == nil || !basecampService.IsConnected() {
		return nil, fmt.Errorf("basecamp not connected")
	}

	var ctx = context.Background()
	var people, err = basecampService.GetPeople(ctx)
	if err != nil {
		return nil, err
	}

	var result []BasecampPersonDTO
	for _, p := range people {
		result = append(result, a.basecampPersonToDTO(&p))
	}

	return result, nil
}

// ============================================================================
// HELPERS
// ============================================================================

func (a *App) basecampTodoToDTO(t *ports.BasecampTodo) BasecampTodoDTO {
	var dto = BasecampTodoDTO{
		ID:            t.ID,
		TodoListID:    t.TodoListID,
		ProjectID:     t.ProjectID,
		Content:       t.Content,
		Description:   t.Description,
		DueOn:         t.DueOn,
		Completed:     t.Completed,
		CompletedAt:   t.CompletedAt,
		CommentsCount: t.CommentsCount,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
	}

	if t.Creator != nil {
		var creator = a.basecampPersonToDTO(t.Creator)
		dto.Creator = &creator
	}

	for _, assignee := range t.Assignees {
		dto.Assignees = append(dto.Assignees, a.basecampPersonToDTO(&assignee))
	}

	return dto
}

func (a *App) basecampMessageToDTO(m *ports.BasecampMessage) BasecampMessageDTO {
	var dto = BasecampMessageDTO{
		ID:            m.ID,
		ProjectID:     m.ProjectID,
		Subject:       m.Subject,
		Content:       m.Content,
		CommentsCount: m.CommentsCount,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}

	if m.Creator != nil {
		var creator = a.basecampPersonToDTO(m.Creator)
		dto.Creator = &creator
	}

	return dto
}

func (a *App) basecampPersonToDTO(p *ports.BasecampPerson) BasecampPersonDTO {
	return BasecampPersonDTO{
		ID:           p.ID,
		Name:         p.Name,
		EmailAddress: p.EmailAddress,
		Title:        p.Title,
		AvatarURL:    p.AvatarURL,
		Admin:        p.Admin,
	}
}
