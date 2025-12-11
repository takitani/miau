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

// Summarize summarizes a single email using AI (with cache, defaults to brief style)
func (s *AIService) Summarize(ctx context.Context, emailID int64) (string, error) {
	// Check cache first
	var cached, cacheErr = storage.GetEmailSummary(emailID)
	if cacheErr == nil && cached != nil && storage.IsSummaryCacheFresh(cached.CreatedAt) {
		return cached.Content, nil
	}

	// Get email content from storage
	var email, err = storage.GetEmailByID(emailID)
	if err != nil {
		return "", fmt.Errorf("failed to get email: %w", err)
	}

	// Build prompt for summarization (brief style by default)
	var prompt = s.buildSummarizePromptWithStyle(email, ports.SummaryStyleBrief)

	// Call AI CLI
	var response, aiErr = s.callAI(ctx, "claude", prompt)
	if aiErr != nil {
		return "", aiErr
	}

	// Save to cache (ignore errors, cache is optional)
	storage.SaveEmailSummary(emailID, storage.SummaryStyleBrief, response, nil)

	return response, nil
}

// SummarizeWithStyle summarizes an email with a specific style
func (s *AIService) SummarizeWithStyle(ctx context.Context, emailID int64, style ports.SummaryStyle) (*ports.Summary, error) {
	// Check cache first
	var cached, cacheErr = storage.GetEmailSummary(emailID)
	if cacheErr == nil && cached != nil && storage.IsSummaryCacheFresh(cached.CreatedAt) {
		// If cached with same or more detailed style, use it
		if cached.Style == storage.SummaryStyle(style) || isMoreDetailed(cached.Style, storage.SummaryStyle(style)) {
			return &ports.Summary{
				EmailID:   emailID,
				Style:     style,
				Content:   cached.Content,
				KeyPoints: storage.GetKeyPointsFromSummary(cached),
				Cached:    true,
			}, nil
		}
	}

	// Get email content from storage
	var email, err = storage.GetEmailByID(emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	// Build prompt for summarization with specific style
	var prompt = s.buildSummarizePromptWithStyle(email, style)

	// Call AI CLI
	var response, aiErr = s.callAI(ctx, "claude", prompt)
	if aiErr != nil {
		return nil, fmt.Errorf("AI failed to generate summary: %w", aiErr)
	}

	// Parse key points from response if detailed
	var keyPoints []string
	if style == ports.SummaryStyleDetailed {
		keyPoints = s.parseKeyPoints(response)
	}

	// Save to cache
	storage.SaveEmailSummary(emailID, storage.SummaryStyle(style), response, keyPoints)

	return &ports.Summary{
		EmailID:   emailID,
		Style:     style,
		Content:   response,
		KeyPoints: keyPoints,
		Cached:    false,
	}, nil
}

// GetCachedSummary retrieves a cached summary if exists
func (s *AIService) GetCachedSummary(ctx context.Context, emailID int64) (*ports.Summary, error) {
	var cached, err = storage.GetEmailSummary(emailID)
	if err != nil {
		return nil, err
	}
	if cached == nil {
		return nil, nil
	}

	return &ports.Summary{
		EmailID:   emailID,
		Style:     ports.SummaryStyle(cached.Style),
		Content:   cached.Content,
		KeyPoints: storage.GetKeyPointsFromSummary(cached),
		Cached:    true,
	}, nil
}

// InvalidateSummary removes a cached summary
func (s *AIService) InvalidateSummary(ctx context.Context, emailID int64) error {
	return storage.DeleteEmailSummary(emailID)
}

// isMoreDetailed checks if style1 is more detailed than style2
func isMoreDetailed(style1, style2 storage.SummaryStyle) bool {
	var order = map[storage.SummaryStyle]int{
		storage.SummaryStyleTLDR:     1,
		storage.SummaryStyleBrief:    2,
		storage.SummaryStyleDetailed: 3,
	}
	return order[style1] >= order[style2]
}

// SummarizeThread summarizes an entire email thread using AI (with cache)
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

	// Get thread ID for caching
	var threadID string
	for _, e := range emails {
		if e.ThreadID != "" {
			threadID = e.ThreadID
			break
		}
	}

	// Check cache if we have a thread ID
	if threadID != "" {
		var cached, cacheErr = storage.GetThreadSummary(threadID)
		if cacheErr == nil && cached != nil && storage.IsSummaryCacheFresh(cached.CreatedAt) {
			return cached.Timeline, nil
		}
	}

	// Build thread prompt
	var prompt = s.buildThreadSummarizePrompt(emails)

	// Call AI CLI
	var response, aiErr = s.callAI(ctx, "claude", prompt)
	if aiErr != nil {
		return "", aiErr
	}

	// Save to cache if we have a thread ID
	if threadID != "" {
		var participants = s.extractParticipants(emails)
		storage.SaveThreadSummary(threadID, participants, response, nil, nil)
	}

	return response, nil
}

// SummarizeThreadDetailed returns detailed thread summary with structured data
func (s *AIService) SummarizeThreadDetailed(ctx context.Context, emailID int64) (*ports.ThreadSummaryResult, error) {
	// Get thread for this email
	var emails, err = storage.GetThreadForEmail(emailID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	if len(emails) == 0 {
		return nil, fmt.Errorf("email not found")
	}

	// Get thread ID for caching
	var threadID string
	for _, e := range emails {
		if e.ThreadID != "" {
			threadID = e.ThreadID
			break
		}
	}

	// Check cache if we have a thread ID
	if threadID != "" {
		var cached, cacheErr = storage.GetThreadSummary(threadID)
		if cacheErr == nil && cached != nil && storage.IsSummaryCacheFresh(cached.CreatedAt) {
			return &ports.ThreadSummaryResult{
				ThreadID:     threadID,
				Participants: storage.ParseThreadSummaryParticipants(cached),
				Timeline:     cached.Timeline,
				KeyDecisions: storage.ParseThreadSummaryKeyDecisions(cached),
				ActionItems:  storage.ParseThreadSummaryActionItems(cached),
				Cached:       true,
			}, nil
		}
	}

	// If only one email, convert to thread summary format
	if len(emails) == 1 {
		var summary, sumErr = s.Summarize(ctx, emailID)
		if sumErr != nil {
			return nil, sumErr
		}
		return &ports.ThreadSummaryResult{
			ThreadID:     threadID,
			Participants: []string{emails[0].FromEmail},
			Timeline:     summary,
			KeyDecisions: nil,
			ActionItems:  nil,
			Cached:       false,
		}, nil
	}

	// Build detailed thread prompt
	var prompt = s.buildDetailedThreadSummarizePrompt(emails)

	// Call AI CLI
	var response, aiErr = s.callAI(ctx, "claude", prompt)
	if aiErr != nil {
		return nil, fmt.Errorf("AI failed to summarize thread: %w", aiErr)
	}

	// Extract participants
	var participants = s.extractParticipants(emails)

	// Parse structured response
	var keyDecisions = s.parseSection(response, "Decisões:")
	var actionItems = s.parseSection(response, "Ações:")

	// Save to cache if we have a thread ID
	if threadID != "" {
		storage.SaveThreadSummary(threadID, participants, response, keyDecisions, actionItems)
	}

	return &ports.ThreadSummaryResult{
		ThreadID:     threadID,
		Participants: participants,
		Timeline:     response,
		KeyDecisions: keyDecisions,
		ActionItems:  actionItems,
		Cached:       false,
	}, nil
}

// extractParticipants extracts unique participants from emails
func (s *AIService) extractParticipants(emails []storage.Email) []string {
	var seen = make(map[string]bool)
	var participants []string
	for _, e := range emails {
		if e.FromEmail != "" && !seen[e.FromEmail] {
			seen[e.FromEmail] = true
			participants = append(participants, e.FromEmail)
		}
	}
	return participants
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

// buildSummarizePromptWithStyle builds the prompt with specific style
func (s *AIService) buildSummarizePromptWithStyle(email *storage.Email, style ports.SummaryStyle) string {
	var body = email.BodyText
	if body == "" {
		body = email.Snippet
	}

	var sb strings.Builder

	switch style {
	case ports.SummaryStyleTLDR:
		sb.WriteString("Resuma este email em 1-2 frases curtas em português brasileiro.\n")
		sb.WriteString("Seja extremamente conciso - capture apenas a essência do email.\n")
	case ports.SummaryStyleBrief:
		sb.WriteString("Resuma este email em 3-5 frases em português brasileiro.\n")
		sb.WriteString("Destaque:\n")
		sb.WriteString("- O ponto principal\n")
		sb.WriteString("- Ações necessárias (se houver)\n")
		sb.WriteString("- Prazo ou urgência (se mencionado)\n")
	case ports.SummaryStyleDetailed:
		sb.WriteString("Resuma este email de forma detalhada em português brasileiro.\n")
		sb.WriteString("Inclua:\n")
		sb.WriteString("- Contexto e propósito do email\n")
		sb.WriteString("- Todos os pontos importantes discutidos\n")
		sb.WriteString("- Ações solicitadas ou sugeridas\n")
		sb.WriteString("- Prazos, datas ou compromissos mencionados\n")
		sb.WriteString("- Informações relevantes de anexos (se mencionados)\n")
	default:
		sb.WriteString("Resuma este email em português brasileiro.\n")
	}

	sb.WriteString("\nNÃO use markdown, asteriscos ou formatação especial.\n")
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

// buildDetailedThreadSummarizePrompt builds detailed thread prompt
func (s *AIService) buildDetailedThreadSummarizePrompt(emails []storage.Email) string {
	var sb strings.Builder
	sb.WriteString("Resuma esta conversa de emails de forma detalhada em português brasileiro.\n")
	sb.WriteString("Organize o resumo em seções:\n\n")
	sb.WriteString("Resumo: [visão geral da conversa]\n\n")
	sb.WriteString("Cronologia: [sequência dos principais eventos/mensagens]\n\n")
	sb.WriteString("Decisões: [lista de decisões tomadas, uma por linha]\n\n")
	sb.WriteString("Ações: [lista de ações pendentes ou solicitadas, uma por linha]\n\n")
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

// parseKeyPoints extracts key points from detailed summary
func (s *AIService) parseKeyPoints(response string) []string {
	var lines = strings.Split(response, "\n")
	var points []string
	var inSection = false

	for _, line := range lines {
		var trimmed = strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Look for bullet points or numbered items
		if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "•") {
			inSection = true
			var point = strings.TrimPrefix(trimmed, "-")
			point = strings.TrimPrefix(point, "•")
			point = strings.TrimSpace(point)
			if point != "" {
				points = append(points, point)
			}
		} else if inSection && len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' {
			// Numbered items
			if trimmed[1] == '.' || trimmed[1] == ')' {
				var point = strings.TrimSpace(trimmed[2:])
				if point != "" {
					points = append(points, point)
				}
			}
		}
	}
	return points
}

// parseSection extracts items from a specific section
func (s *AIService) parseSection(response, sectionHeader string) []string {
	var lines = strings.Split(response, "\n")
	var items []string
	var inSection = false

	for _, line := range lines {
		var trimmed = strings.TrimSpace(line)

		// Check if we're entering the target section
		if strings.Contains(strings.ToLower(trimmed), strings.ToLower(sectionHeader)) {
			inSection = true
			continue
		}

		// Check if we're leaving the section (new section header)
		if inSection && (strings.HasSuffix(trimmed, ":") || trimmed == "") {
			if strings.HasSuffix(trimmed, ":") && !strings.Contains(strings.ToLower(trimmed), strings.ToLower(sectionHeader)) {
				inSection = false
			}
			continue
		}

		// Collect items in section
		if inSection && trimmed != "" {
			var item = strings.TrimPrefix(trimmed, "-")
			item = strings.TrimPrefix(item, "•")
			item = strings.TrimSpace(item)
			if item != "" {
				items = append(items, item)
			}
		}
	}
	return items
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
