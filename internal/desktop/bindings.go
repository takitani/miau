package desktop

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/opik/miau/internal/ports"
)

// ============================================================================
// FOLDER OPERATIONS
// ============================================================================

// GetFolders returns all mail folders
func (a *App) GetFolders() ([]FolderDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	var folders, err = a.application.Email().GetFolders(context.Background())
	if err != nil {
		return nil, err
	}

	var result []FolderDTO
	for _, f := range folders {
		result = append(result, a.folderToDTO(&f))
	}
	return result, nil
}

// SelectFolder selects a folder as current
func (a *App) SelectFolder(name string) (*FolderDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	var folder, err = a.application.Email().SelectFolder(context.Background(), name)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.currentFolder = name
	a.mu.Unlock()

	var dto = a.folderToDTO(folder)
	return &dto, nil
}

// GetCurrentFolder returns the currently selected folder name
func (a *App) GetCurrentFolder() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentFolder
}

// ============================================================================
// EMAIL OPERATIONS
// ============================================================================

// GetEmails returns emails from a folder
func (a *App) GetEmails(folder string, limit int) ([]EmailDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 50
	}

	var emails, err = a.application.Email().GetEmails(context.Background(), folder, limit)
	if err != nil {
		return nil, err
	}

	var result []EmailDTO
	for _, e := range emails {
		result = append(result, a.emailMetadataToDTO(&e))
	}
	return result, nil
}

// GetEmail returns full email details by ID
func (a *App) GetEmail(id int64) (*EmailDetailDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	var email, err = a.application.Email().GetEmail(context.Background(), id)
	if err != nil {
		return nil, err
	}

	return a.emailContentToDTO(email), nil
}

// GetEmailByUID returns email by UID in current folder
func (a *App) GetEmailByUID(uid uint32) (*EmailDetailDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	a.mu.RLock()
	var folder = a.currentFolder
	a.mu.RUnlock()

	var email, err = a.application.Email().GetEmailByUID(context.Background(), folder, uid)
	if err != nil {
		return nil, err
	}

	return a.emailContentToDTO(email), nil
}

// ============================================================================
// EMAIL ACTIONS
// ============================================================================

// MarkAsRead marks an email as read or unread
func (a *App) MarkAsRead(id int64, read bool) error {
	if a.application == nil {
		return nil
	}
	return a.application.Email().MarkAsRead(context.Background(), id, read)
}

// MarkAsStarred marks an email as starred or unstarred
func (a *App) MarkAsStarred(id int64, starred bool) error {
	if a.application == nil {
		return nil
	}
	return a.application.Email().MarkAsStarred(context.Background(), id, starred)
}

// Archive archives an email
func (a *App) Archive(id int64) error {
	if a.application == nil {
		return nil
	}
	return a.application.Email().Archive(context.Background(), id)
}

// Delete moves an email to trash
func (a *App) Delete(id int64) error {
	if a.application == nil {
		return nil
	}
	return a.application.Email().Delete(context.Background(), id)
}

// MoveToFolder moves an email to a different folder
func (a *App) MoveToFolder(id int64, folder string) error {
	if a.application == nil {
		return nil
	}
	return a.application.Email().MoveToFolder(context.Background(), id, folder)
}

// ============================================================================
// SEARCH
// ============================================================================

// Search performs a full-text search
func (a *App) Search(query string, limit int) (*SearchResultDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 50
	}

	var result, err = a.application.Search().Search(context.Background(), query, limit)
	if err != nil {
		return nil, err
	}

	var emails []EmailDTO
	for _, e := range result.Emails {
		emails = append(emails, a.emailMetadataToDTO(&e))
	}

	return &SearchResultDTO{
		Emails:     emails,
		TotalCount: result.TotalCount,
		Query:      result.Query,
	}, nil
}

// SearchInFolder searches within a specific folder
func (a *App) SearchInFolder(folder, query string, limit int) (*SearchResultDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	if limit <= 0 {
		limit = 50
	}

	var result, err = a.application.Search().SearchInFolder(context.Background(), folder, query, limit)
	if err != nil {
		return nil, err
	}

	var emails []EmailDTO
	for _, e := range result.Emails {
		emails = append(emails, a.emailMetadataToDTO(&e))
	}

	return &SearchResultDTO{
		Emails:     emails,
		TotalCount: result.TotalCount,
		Query:      result.Query,
	}, nil
}

// ============================================================================
// CONNECTION & SYNC
// ============================================================================

// Connect connects to the email server
func (a *App) Connect() error {
	if a.application == nil {
		return nil
	}
	return a.application.Sync().Connect(context.Background())
}

// Disconnect disconnects from the email server
func (a *App) Disconnect() error {
	if a.application == nil {
		return nil
	}
	return a.application.Sync().Disconnect(context.Background())
}

// IsConnected returns true if connected to email server
func (a *App) IsConnected() bool {
	if a.application == nil {
		return false
	}
	return a.application.Sync().IsConnected()
}

// SyncFolder syncs a specific folder
func (a *App) SyncFolder(folder string) error {
	if a.application == nil {
		return nil
	}
	_, err := a.application.Sync().SyncFolder(context.Background(), folder)
	return err
}

// SyncCurrentFolder syncs the currently selected folder
func (a *App) SyncCurrentFolder() error {
	a.mu.RLock()
	var folder = a.currentFolder
	a.mu.RUnlock()

	return a.SyncFolder(folder)
}

// GetConnectionStatus returns current connection status
func (a *App) GetConnectionStatus() ConnectionStatus {
	if a.application == nil {
		return ConnectionStatus{Connected: false}
	}

	a.mu.RLock()
	var connected = a.connected
	a.mu.RUnlock()

	return ConnectionStatus{
		Connected: connected,
	}
}

// ============================================================================
// SEND EMAIL
// ============================================================================

// SendEmail sends an email
func (a *App) SendEmail(req SendRequest) (*SendResult, error) {
	if a.application == nil {
		return &SendResult{Success: false, Error: "Application not initialized"}, nil
	}

	var portsReq = &ports.SendRequest{
		To:       req.To,
		Cc:       req.Cc,
		Bcc:      req.Bcc,
		Subject:  req.Subject,
		BodyText: req.Body,
	}

	if req.IsHTML {
		portsReq.BodyHTML = req.Body
		portsReq.BodyText = "" // TODO: generate text version
	}

	if req.ReplyTo > 0 {
		portsReq.ReplyToEmailID = &req.ReplyTo
	}

	var result, err = a.application.Send().Send(context.Background(), portsReq)
	if err != nil {
		return &SendResult{Success: false, Error: err.Error()}, nil
	}

	return &SendResult{
		Success:   result.Success,
		MessageID: result.MessageID,
		Error:     a.getError(result.Error),
	}, nil
}

// GetSignature returns the configured email signature
func (a *App) GetSignature() (string, error) {
	if a.application == nil {
		return "", nil
	}
	return a.application.Send().GetSignature(context.Background())
}

// ============================================================================
// DRAFTS
// ============================================================================

// SaveDraft saves a draft email
func (a *App) SaveDraft(draft DraftDTO) (int64, error) {
	if a.application == nil {
		return 0, nil
	}

	var portsDraft = &ports.Draft{
		ID:           draft.ID,
		ToAddresses:  strings.Join(draft.To, ", "),
		CcAddresses:  strings.Join(draft.Cc, ", "),
		BccAddresses: strings.Join(draft.Bcc, ", "),
		Subject:      draft.Subject,
		BodyHTML:     draft.BodyHTML,
		BodyText:     draft.BodyText,
		Status:       ports.DraftStatusDraft,
	}

	if draft.ReplyToID > 0 {
		portsDraft.ReplyToEmailID = &draft.ReplyToID
	}

	var result *ports.Draft
	var err error

	if draft.ID > 0 {
		err = a.application.Draft().UpdateDraft(context.Background(), portsDraft)
		if err != nil {
			return 0, err
		}
		result = portsDraft
	} else {
		result, err = a.application.Draft().CreateDraft(context.Background(), portsDraft)
		if err != nil {
			return 0, err
		}
	}

	return result.ID, nil
}

// GetDraft returns a draft by ID
func (a *App) GetDraft(id int64) (*DraftDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	var draft, err = a.application.Draft().GetDraft(context.Background(), id)
	if err != nil {
		return nil, err
	}

	return a.draftToDTO(draft), nil
}

// ListDrafts returns all drafts
func (a *App) ListDrafts() ([]DraftDTO, error) {
	if a.application == nil {
		return nil, nil
	}

	var drafts, err = a.application.Draft().ListDrafts(context.Background())
	if err != nil {
		return nil, err
	}

	var result []DraftDTO
	for _, d := range drafts {
		result = append(result, *a.draftToDTO(&d))
	}
	return result, nil
}

// DeleteDraft deletes a draft
func (a *App) DeleteDraft(id int64) error {
	if a.application == nil {
		return nil
	}
	return a.application.Draft().DeleteDraft(context.Background(), id)
}

// SendDraft sends a draft
func (a *App) SendDraft(id int64) (*SendResult, error) {
	if a.application == nil {
		return &SendResult{Success: false, Error: "Application not initialized"}, nil
	}

	var result, err = a.application.Send().SendDraft(context.Background(), id)
	if err != nil {
		return &SendResult{Success: false, Error: err.Error()}, nil
	}

	return &SendResult{
		Success:   result.Success,
		MessageID: result.MessageID,
		Error:     a.getError(result.Error),
	}, nil
}

// ============================================================================
// AI INTEGRATION
// ============================================================================

// AskAI sends a question to a CLI-based AI provider
func (a *App) AskAI(provider, question, emailContextJSON string) (string, error) {
	var cmd *exec.Cmd
	var prompt = question

	// Add email context if provided
	if emailContextJSON != "" {
		prompt = fmt.Sprintf("Contexto do email:\n%s\n\nPergunta: %s", emailContextJSON, question)
	}

	switch provider {
	case "claude":
		// Claude Code CLI - uses stdin for prompt
		// --dangerously-skip-permissions allows sqlite3 without asking
		cmd = exec.Command("claude", "-p", "--dangerously-skip-permissions", prompt)
	case "gemini":
		// Gemini CLI
		cmd = exec.Command("gemini", prompt)
	case "ollama":
		// Ollama with llama3
		cmd = exec.Command("ollama", "run", "llama3", prompt)
	case "openai":
		// OpenAI CLI (if installed)
		cmd = exec.Command("openai", "api", "chat.completions.create", "-m", "gpt-4", "-g", "user", prompt)
	default:
		return "", fmt.Errorf("provider não suportado: %s", provider)
	}

	var output, err = cmd.CombinedOutput()
	if err != nil {
		// Try to provide helpful error message
		if strings.Contains(err.Error(), "executable file not found") {
			return "", fmt.Errorf("%s CLI não encontrado. Instale com: %s", provider, getInstallHint(provider))
		}
		return "", fmt.Errorf("erro ao executar %s: %v\nOutput: %s", provider, err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// getInstallHint returns installation instructions for AI CLIs
func getInstallHint(provider string) string {
	switch provider {
	case "claude":
		return "npm install -g @anthropic-ai/claude-code"
	case "gemini":
		return "pip install google-generativeai"
	case "ollama":
		return "curl https://ollama.ai/install.sh | sh"
	case "openai":
		return "pip install openai"
	default:
		return "consulte a documentação do provider"
	}
}

// GetAIProviders returns available AI providers and their status
func (a *App) GetAIProviders() []map[string]interface{} {
	providers := []struct {
		id   string
		name string
		cmd  string
	}{
		{"claude", "Claude", "claude"},
		{"gemini", "Gemini", "gemini"},
		{"ollama", "Ollama", "ollama"},
		{"openai", "OpenAI", "openai"},
	}

	var result []map[string]interface{}
	for _, p := range providers {
		_, err := exec.LookPath(p.cmd)
		result = append(result, map[string]interface{}{
			"id":        p.id,
			"name":      p.name,
			"available": err == nil,
		})
	}
	return result
}

// ============================================================================
// ACCOUNTS
// ============================================================================

// GetAccounts returns all configured accounts
func (a *App) GetAccounts() []AccountDTO {
	if a.cfg == nil {
		return nil
	}

	var result []AccountDTO
	for _, acc := range a.cfg.Accounts {
		result = append(result, AccountDTO{
			Email: acc.Email,
			Name:  acc.Name,
		})
	}
	return result
}

// GetCurrentAccount returns the current account
func (a *App) GetCurrentAccount() *AccountDTO {
	if a.account == nil {
		return nil
	}
	return &AccountDTO{
		Email: a.account.Email,
		Name:  a.account.Name,
	}
}

// ============================================================================
// HELPERS
// ============================================================================

// draftToDTO converts ports.Draft to DraftDTO
func (a *App) draftToDTO(draft *ports.Draft) *DraftDTO {
	if draft == nil {
		return nil
	}

	// Parse addresses
	var to, cc, bcc []string
	if draft.ToAddresses != "" {
		to = strings.Split(draft.ToAddresses, ", ")
	}
	if draft.CcAddresses != "" {
		cc = strings.Split(draft.CcAddresses, ", ")
	}
	if draft.BccAddresses != "" {
		bcc = strings.Split(draft.BccAddresses, ", ")
	}

	var replyToID int64
	if draft.ReplyToEmailID != nil {
		replyToID = *draft.ReplyToEmailID
	}

	return &DraftDTO{
		ID:        draft.ID,
		To:        to,
		Cc:        cc,
		Bcc:       bcc,
		Subject:   draft.Subject,
		BodyHTML:  draft.BodyHTML,
		BodyText:  draft.BodyText,
		ReplyToID: replyToID,
	}
}
