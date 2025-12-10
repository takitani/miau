package services

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/storage"
)

// AIService handles AI-assisted email operations
// CRITICAL: This is the SINGLE SOURCE OF TRUTH for AI logic
// TUI and Desktop MUST use this service, NEVER call AI CLIs directly
type AIService struct {
	mu      sync.RWMutex
	storage ports.StoragePort
	events  ports.EventBus
	account *ports.AccountInfo
}

// NewAIService creates a new AIService
func NewAIService(storage ports.StoragePort, events ports.EventBus) *AIService {
	return &AIService{
		storage: storage,
		events:  events,
	}
}

// SetAccount sets the current account
func (s *AIService) SetAccount(account *ports.AccountInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.account = account
}

// Summarize summarizes a single email using AI
func (s *AIService) Summarize(ctx context.Context, emailID int64) (string, error) {
	// Get email content from storage
	var email, err = storage.GetEmailByID(emailID)
	if err != nil {
		return "", fmt.Errorf("failed to get email: %w", err)
	}

	// Build prompt for summarization
	var prompt = s.buildSummarizePrompt(email, false)

	// Call AI CLI
	return s.callAI(ctx, "claude", prompt)
}

// SummarizeThread summarizes an entire email thread using AI
func (s *AIService) SummarizeThread(ctx context.Context, emailID int64) (string, error) {
	// Get thread for this email
	var emails, err = storage.GetThreadForEmail(emailID)
	if err != nil {
		return "", fmt.Errorf("failed to get thread: %w", err)
	}

	if len(emails) == 0 {
		return "", fmt.Errorf("email not found")
	}

	// If only one email, use simple summarize
	if len(emails) == 1 {
		return s.Summarize(ctx, emailID)
	}

	// Build thread prompt
	var prompt = s.buildThreadSummarizePrompt(emails)

	// Call AI CLI
	return s.callAI(ctx, "claude", prompt)
}

// GenerateReply generates a reply draft using AI
func (s *AIService) GenerateReply(ctx context.Context, emailID int64, userPrompt string) (*ports.Draft, error) {
	s.mu.RLock()
	var account = s.account
	s.mu.RUnlock()

	if account == nil {
		return nil, fmt.Errorf("no account set")
	}

	// Get email content
	var email, err = storage.GetEmailByID(emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	// Build prompt for reply generation
	var prompt = s.buildReplyPrompt(email, userPrompt)

	// Call AI CLI
	var reply, aiErr = s.callAI(ctx, "claude", prompt)
	if aiErr != nil {
		return nil, fmt.Errorf("AI failed to generate reply: %w", aiErr)
	}

	// Create draft with the generated reply
	var inReplyTo string
	if email.MessageID.Valid {
		inReplyTo = email.MessageID.String
	}
	var draft = &ports.Draft{
		Subject:     s.buildReplySubject(email.Subject),
		ToAddresses: email.FromEmail,
		BodyText:    reply,
		InReplyTo:   inReplyTo,
		Source:      "ai",
	}

	return draft, nil
}

// ExtractActions extracts action items from an email
func (s *AIService) ExtractActions(ctx context.Context, emailID int64) ([]string, error) {
	// Get email content
	var email, err = storage.GetEmailByID(emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	// Build prompt for action extraction
	var prompt = s.buildExtractActionsPrompt(email)

	// Call AI CLI
	var response, aiErr = s.callAI(ctx, "claude", prompt)
	if aiErr != nil {
		return nil, fmt.Errorf("AI failed to extract actions: %w", aiErr)
	}

	// Parse response into action items
	var actions = s.parseActionItems(response)

	return actions, nil
}

// ClassifyEmail classifies an email (spam, important, newsletter, etc.)
func (s *AIService) ClassifyEmail(ctx context.Context, emailID int64) (string, error) {
	// Get email content
	var email, err = storage.GetEmailByID(emailID)
	if err != nil {
		return "", fmt.Errorf("failed to get email: %w", err)
	}

	// Build prompt for classification
	var prompt = s.buildClassifyPrompt(email)

	// Call AI CLI
	return s.callAI(ctx, "claude", prompt)
}

// buildSummarizePrompt builds the prompt for email summarization
func (s *AIService) buildSummarizePrompt(email *storage.Email, isThread bool) string {
	var body = email.BodyText
	if body == "" {
		body = email.Snippet
	}

	var sb strings.Builder
	sb.WriteString("Resuma este email de forma concisa em português brasileiro.\n")
	sb.WriteString("O resumo deve ter no máximo 3-4 linhas e destacar:\n")
	sb.WriteString("- O ponto principal do email\n")
	sb.WriteString("- Ações necessárias (se houver)\n")
	sb.WriteString("- Prazo ou urgência (se mencionado)\n\n")
	sb.WriteString("NÃO use markdown, asteriscos ou formatação especial.\n")
	sb.WriteString("Retorne apenas texto simples.\n\n")
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("De: %s <%s>\n", email.FromName, email.FromEmail))
	sb.WriteString(fmt.Sprintf("Assunto: %s\n", email.Subject))
	sb.WriteString(fmt.Sprintf("Data: %s\n", email.Date.Time.Format("02/01/2006 15:04")))
	sb.WriteString("---\n")
	sb.WriteString(body)
	sb.WriteString("\n---\n")

	return sb.String()
}

// buildThreadSummarizePrompt builds the prompt for thread summarization
func (s *AIService) buildThreadSummarizePrompt(emails []storage.Email) string {
	var sb strings.Builder
	sb.WriteString("Resuma esta conversa de emails de forma concisa em português brasileiro.\n")
	sb.WriteString("O resumo deve:\n")
	sb.WriteString("- Apresentar o contexto inicial da conversa\n")
	sb.WriteString("- Destacar os principais pontos discutidos\n")
	sb.WriteString("- Mencionar decisões tomadas ou pendências\n")
	sb.WriteString("- Indicar o estado atual da discussão\n\n")
	sb.WriteString("NÃO use markdown, asteriscos ou formatação especial.\n")
	sb.WriteString("Retorne apenas texto simples.\n\n")
	sb.WriteString(fmt.Sprintf("Conversa com %d mensagens:\n\n", len(emails)))

	// Emails are DESC order, reverse for chronological reading
	for i := len(emails) - 1; i >= 0; i-- {
		var email = emails[i]
		var body = email.BodyText
		if body == "" {
			body = email.Snippet
		}
		// Truncate very long emails
		if len(body) > 500 {
			body = body[:500] + "..."
		}

		sb.WriteString(fmt.Sprintf("--- Mensagem %d ---\n", len(emails)-i))
		sb.WriteString(fmt.Sprintf("De: %s <%s>\n", email.FromName, email.FromEmail))
		sb.WriteString(fmt.Sprintf("Data: %s\n", email.Date.Time.Format("02/01/2006 15:04")))
		sb.WriteString(body)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// buildReplyPrompt builds the prompt for generating a reply
func (s *AIService) buildReplyPrompt(email *storage.Email, userPrompt string) string {
	var body = email.BodyText
	if body == "" {
		body = email.Snippet
	}

	var sb strings.Builder
	sb.WriteString("Escreva uma resposta para este email em português brasileiro.\n")
	if userPrompt != "" {
		sb.WriteString(fmt.Sprintf("Instruções específicas: %s\n", userPrompt))
	}
	sb.WriteString("\nIMPORTANTE:\n")
	sb.WriteString("- Retorne APENAS o corpo do email, pronto para enviar\n")
	sb.WriteString("- NÃO use markdown (**, --, ##)\n")
	sb.WriteString("- NÃO inclua 'Assunto:', 'Para:', 'De:'\n")
	sb.WriteString("- NÃO pergunte se deve enviar\n")
	sb.WriteString("- Mantenha tom profissional mas cordial\n\n")
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("Email original de %s:\n", email.FromName))
	sb.WriteString(fmt.Sprintf("Assunto: %s\n", email.Subject))
	sb.WriteString("---\n")
	sb.WriteString(body)
	sb.WriteString("\n---\n")

	return sb.String()
}

// buildExtractActionsPrompt builds the prompt for action extraction
func (s *AIService) buildExtractActionsPrompt(email *storage.Email) string {
	var body = email.BodyText
	if body == "" {
		body = email.Snippet
	}

	var sb strings.Builder
	sb.WriteString("Extraia as ações necessárias deste email.\n")
	sb.WriteString("Retorne uma lista simples, uma ação por linha.\n")
	sb.WriteString("Se não houver ações claras, retorne 'Nenhuma ação necessária'.\n")
	sb.WriteString("NÃO use markdown ou formatação especial.\n\n")
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("De: %s\n", email.FromName))
	sb.WriteString(fmt.Sprintf("Assunto: %s\n", email.Subject))
	sb.WriteString("---\n")
	sb.WriteString(body)
	sb.WriteString("\n---\n")

	return sb.String()
}

// buildClassifyPrompt builds the prompt for email classification
func (s *AIService) buildClassifyPrompt(email *storage.Email) string {
	var body = email.BodyText
	if body == "" {
		body = email.Snippet
	}
	// Truncate for classification
	if len(body) > 300 {
		body = body[:300] + "..."
	}

	var sb strings.Builder
	sb.WriteString("Classifique este email em UMA das categorias:\n")
	sb.WriteString("- importante: requer atenção/resposta urgente\n")
	sb.WriteString("- pessoal: comunicação pessoal\n")
	sb.WriteString("- trabalho: comunicação profissional regular\n")
	sb.WriteString("- newsletter: boletins informativos\n")
	sb.WriteString("- promocional: ofertas e marketing\n")
	sb.WriteString("- notificacao: alertas automáticos de sistemas\n")
	sb.WriteString("- spam: não solicitado/suspeito\n\n")
	sb.WriteString("Retorne APENAS a categoria, sem explicação.\n\n")
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("De: %s <%s>\n", email.FromName, email.FromEmail))
	sb.WriteString(fmt.Sprintf("Assunto: %s\n", email.Subject))
	sb.WriteString("---\n")
	sb.WriteString(body)
	sb.WriteString("\n---\n")

	return sb.String()
}

// buildReplySubject builds the reply subject line
func (s *AIService) buildReplySubject(originalSubject string) string {
	// Remove existing Re: prefixes
	var subject = strings.TrimSpace(originalSubject)
	for strings.HasPrefix(strings.ToLower(subject), "re:") {
		subject = strings.TrimSpace(subject[3:])
	}
	return "Re: " + subject
}

// parseActionItems parses the AI response into action items
func (s *AIService) parseActionItems(response string) []string {
	var lines = strings.Split(response, "\n")
	var actions []string

	for _, line := range lines {
		var trimmed = strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Remove common list prefixes
		trimmed = strings.TrimPrefix(trimmed, "- ")
		trimmed = strings.TrimPrefix(trimmed, "* ")
		trimmed = strings.TrimPrefix(trimmed, "• ")
		// Remove numbered prefixes like "1. " or "1) "
		if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' {
			if trimmed[1] == '.' || trimmed[1] == ')' {
				trimmed = strings.TrimSpace(trimmed[2:])
			}
		}
		if trimmed != "" && !strings.Contains(strings.ToLower(trimmed), "nenhuma ação") {
			actions = append(actions, trimmed)
		}
	}

	return actions
}

// callAI calls the AI CLI with the given prompt
func (s *AIService) callAI(ctx context.Context, provider, prompt string) (string, error) {
	var cmd *exec.Cmd

	switch provider {
	case "claude":
		cmd = exec.CommandContext(ctx, "claude", "-p", "--permission-mode", "bypassPermissions", prompt)
	case "gemini":
		cmd = exec.CommandContext(ctx, "gemini", prompt)
	case "ollama":
		cmd = exec.CommandContext(ctx, "ollama", "run", "llama3", prompt)
	case "openai":
		cmd = exec.CommandContext(ctx, "openai", "api", "chat.completions", "create",
			"-m", "gpt-4o-mini",
			"-g", "user", prompt)
	default:
		// Default to claude
		cmd = exec.CommandContext(ctx, "claude", "-p", "--permission-mode", "bypassPermissions", prompt)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	var err = cmd.Run()
	if err != nil {
		// Check if it's a context cancellation
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("AI command failed: %w - stderr: %s", err, stderr.String())
	}

	var response = strings.TrimSpace(stdout.String())
	if response == "" {
		return "", fmt.Errorf("AI returned empty response")
	}

	return response, nil
}
