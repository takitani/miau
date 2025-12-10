package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/storage"
)

// availableCommands returns all available quick commands
func availableCommands() []ports.QuickCommandInfo {
	return []ports.QuickCommandInfo{
		{
			Name:        "dr",
			Aliases:     []string{"draft", "reply"},
			Description: "Gerar resposta com IA",
			Usage:       "/dr [tom]",
			Args:        []string{"formal", "informal", "quick", "detalhado"},
			NeedsEmail:  true,
		},
		{
			Name:        "sum",
			Aliases:     []string{"resume", "resumo"},
			Description: "Resumir email",
			Usage:       "/sum",
			Args:        []string{},
			NeedsEmail:  true,
		},
		{
			Name:        "tldr",
			Aliases:     []string{},
			Description: "Resumo ultra-curto (1-2 frases)",
			Usage:       "/tldr",
			Args:        []string{},
			NeedsEmail:  true,
		},
		{
			Name:        "action",
			Aliases:     []string{"actions", "todo"},
			Description: "Extrair ações do email",
			Usage:       "/action",
			Args:        []string{},
			NeedsEmail:  true,
		},
		{
			Name:        "translate",
			Aliases:     []string{"trad", "tr"},
			Description: "Traduzir email",
			Usage:       "/translate [idioma]",
			Args:        []string{"en", "pt", "es", "fr", "de"},
			NeedsEmail:  true,
		},
		{
			Name:        "tone",
			Aliases:     []string{"tom"},
			Description: "Reescrever com tom diferente",
			Usage:       "/tone [estilo]",
			Args:        []string{"formal", "casual", "profissional", "amigável"},
			NeedsEmail:  true,
		},
		{
			Name:        "classify",
			Aliases:     []string{"class", "cat"},
			Description: "Classificar email (spam, importante, etc)",
			Usage:       "/classify",
			Args:        []string{},
			NeedsEmail:  true,
		},
		{
			Name:        "help",
			Aliases:     []string{"?", "h"},
			Description: "Mostrar comandos disponíveis",
			Usage:       "/help",
			Args:        []string{},
			NeedsEmail:  false,
		},
	}
}

// ParseQuickCommand parses user input and returns a QuickCommand if valid
func ParseQuickCommand(input string) (*ports.QuickCommand, bool) {
	var trimmed = strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "/") {
		return nil, false
	}

	var parts = strings.Fields(trimmed)
	if len(parts) == 0 {
		return nil, false
	}

	var cmdName = strings.TrimPrefix(parts[0], "/")
	if cmdName == "" {
		return nil, false
	}

	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}

	return &ports.QuickCommand{
		Name: strings.ToLower(cmdName),
		Args: args,
		Raw:  trimmed,
	}, true
}

// GetCommandSuggestions returns commands matching a prefix for autocomplete (exported function)
func GetCommandSuggestions(prefix string) []ports.QuickCommandInfo {
	var lower = strings.ToLower(strings.TrimPrefix(prefix, "/"))
	if lower == "" {
		return availableCommands()
	}

	var suggestions []ports.QuickCommandInfo
	for _, cmd := range availableCommands() {
		if strings.HasPrefix(cmd.Name, lower) {
			suggestions = append(suggestions, cmd)
			continue
		}
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(alias, lower) {
				suggestions = append(suggestions, cmd)
				break
			}
		}
	}
	return suggestions
}

// resolveCommandName resolves aliases to canonical command name
func resolveCommandName(name string) string {
	var lower = strings.ToLower(name)
	for _, cmd := range availableCommands() {
		if cmd.Name == lower {
			return cmd.Name
		}
		for _, alias := range cmd.Aliases {
			if alias == lower {
				return cmd.Name
			}
		}
	}
	return lower // Return as-is if not found
}

// GetAvailableCommands returns all available quick commands (interface method)
func (s *AIService) GetAvailableCommands() []ports.QuickCommandInfo {
	return availableCommands()
}

// GetCommandSuggestions returns commands matching a prefix for autocomplete (interface method)
func (s *AIService) GetCommandSuggestions(prefix string) []ports.QuickCommandInfo {
	var lower = strings.ToLower(strings.TrimPrefix(prefix, "/"))
	if lower == "" {
		return availableCommands()
	}

	var suggestions []ports.QuickCommandInfo
	for _, cmd := range availableCommands() {
		if strings.HasPrefix(cmd.Name, lower) {
			suggestions = append(suggestions, cmd)
			continue
		}
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(alias, lower) {
				suggestions = append(suggestions, cmd)
				break
			}
		}
	}
	return suggestions
}

// ExecuteQuickCommand executes a quick command and returns the result (interface method)
func (s *AIService) ExecuteQuickCommand(ctx context.Context, cmd *ports.QuickCommand, emailID int64) (string, error) {
	var resolved = resolveCommandName(cmd.Name)

	switch resolved {
	case "dr":
		return s.executeDraftReply(ctx, cmd, emailID)
	case "sum":
		return s.executeSummarize(ctx, cmd, emailID)
	case "tldr":
		return s.executeTLDR(ctx, cmd, emailID)
	case "action":
		return s.executeExtractActions(ctx, cmd, emailID)
	case "translate":
		return s.executeTranslate(ctx, cmd, emailID)
	case "tone":
		return s.executeTone(ctx, cmd, emailID)
	case "classify":
		return s.executeClassify(ctx, cmd, emailID)
	case "help":
		return s.executeHelp(ctx, cmd)
	default:
		return "", fmt.Errorf("comando desconhecido: /%s\nDigite /help para ver comandos disponíveis", cmd.Name)
	}
}

// executeDraftReply generates a reply draft
func (s *AIService) executeDraftReply(ctx context.Context, cmd *ports.QuickCommand, emailID int64) (string, error) {
	if emailID == 0 {
		return "", fmt.Errorf("selecione um email primeiro (tecla 'a' no inbox)")
	}

	var tone = "informal"
	if len(cmd.Args) > 0 {
		tone = cmd.Args[0]
	}

	var prompt = fmt.Sprintf("tom: %s", tone)
	var draft, err = s.GenerateReply(ctx, emailID, prompt)
	if err != nil {
		return "", err
	}

	// Save draft
	var savedDraft, saveErr = storage.SaveDraft(&storage.Draft{
		Subject:     draft.Subject,
		ToAddresses: draft.ToAddresses,
		BodyText:    storage.NullString(draft.BodyText),
		InReplyTo:   storage.NullString(draft.InReplyTo),
		Source:      "ai_quickcmd",
	})
	if saveErr != nil {
		return draft.BodyText, nil // Return body even if save fails
	}

	return fmt.Sprintf("Rascunho criado (ID: %d)\n\nPara: %s\nAssunto: %s\n\n%s\n\n[Pressione 'e' para editar ou 'd' para ver drafts]",
		savedDraft.ID, draft.ToAddresses, draft.Subject, draft.BodyText), nil
}

// executeSummarize summarizes an email
func (s *AIService) executeSummarize(ctx context.Context, cmd *ports.QuickCommand, emailID int64) (string, error) {
	if emailID == 0 {
		return "", fmt.Errorf("selecione um email primeiro (tecla 'a' no inbox)")
	}

	return s.Summarize(ctx, emailID)
}

// executeTLDR creates an ultra-short summary
func (s *AIService) executeTLDR(ctx context.Context, cmd *ports.QuickCommand, emailID int64) (string, error) {
	if emailID == 0 {
		return "", fmt.Errorf("selecione um email primeiro (tecla 'a' no inbox)")
	}

	var email, err = storage.GetEmailByID(emailID)
	if err != nil {
		return "", fmt.Errorf("erro ao carregar email: %w", err)
	}

	var body = email.BodyText
	if body == "" {
		body = email.Snippet
	}

	var prompt = fmt.Sprintf(`Resuma este email em APENAS 1-2 frases curtas.
Seja extremamente conciso. Máximo 50 palavras.
NÃO use markdown ou formatação.

De: %s <%s>
Assunto: %s
---
%s`, email.FromName, email.FromEmail, email.Subject, body)

	return s.callAI(ctx, "claude", prompt)
}

// executeExtractActions extracts action items
func (s *AIService) executeExtractActions(ctx context.Context, cmd *ports.QuickCommand, emailID int64) (string, error) {
	if emailID == 0 {
		return "", fmt.Errorf("selecione um email primeiro (tecla 'a' no inbox)")
	}

	var actions, err = s.ExtractActions(ctx, emailID)
	if err != nil {
		return "", err
	}

	if len(actions) == 0 {
		return "Nenhuma ação encontrada neste email.", nil
	}

	var sb strings.Builder
	sb.WriteString("Ações encontradas:\n\n")
	for i, action := range actions {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, action))
	}
	return sb.String(), nil
}

// executeTranslate translates an email
func (s *AIService) executeTranslate(ctx context.Context, cmd *ports.QuickCommand, emailID int64) (string, error) {
	if emailID == 0 {
		return "", fmt.Errorf("selecione um email primeiro (tecla 'a' no inbox)")
	}

	var targetLang = "en"
	if len(cmd.Args) > 0 {
		targetLang = cmd.Args[0]
	}

	var langNames = map[string]string{
		"en": "inglês",
		"pt": "português brasileiro",
		"es": "espanhol",
		"fr": "francês",
		"de": "alemão",
		"it": "italiano",
		"ja": "japonês",
		"zh": "chinês",
	}

	var langName = langNames[targetLang]
	if langName == "" {
		langName = targetLang
	}

	var email, err = storage.GetEmailByID(emailID)
	if err != nil {
		return "", fmt.Errorf("erro ao carregar email: %w", err)
	}

	var body = email.BodyText
	if body == "" {
		body = email.Snippet
	}

	var prompt = fmt.Sprintf(`Traduza este email para %s.
Mantenha o tom e estilo original.
NÃO use markdown ou formatação.
Retorne apenas a tradução.

---
%s`, langName, body)

	var translation, aiErr = s.callAI(ctx, "claude", prompt)
	if aiErr != nil {
		return "", aiErr
	}

	return fmt.Sprintf("Tradução para %s:\n\n%s", langName, translation), nil
}

// executeTone rewrites with different tone
func (s *AIService) executeTone(ctx context.Context, cmd *ports.QuickCommand, emailID int64) (string, error) {
	if emailID == 0 {
		return "", fmt.Errorf("selecione um email primeiro (tecla 'a' no inbox)")
	}

	var tone = "formal"
	if len(cmd.Args) > 0 {
		tone = cmd.Args[0]
	}

	var email, err = storage.GetEmailByID(emailID)
	if err != nil {
		return "", fmt.Errorf("erro ao carregar email: %w", err)
	}

	var body = email.BodyText
	if body == "" {
		body = email.Snippet
	}

	var prompt = fmt.Sprintf(`Reescreva este email com um tom mais %s.
Mantenha o conteúdo e informações principais.
NÃO use markdown ou formatação.

---
%s`, tone, body)

	var rewritten, aiErr = s.callAI(ctx, "claude", prompt)
	if aiErr != nil {
		return "", aiErr
	}

	return fmt.Sprintf("Email reescrito (tom: %s):\n\n%s", tone, rewritten), nil
}

// executeClassify classifies an email
func (s *AIService) executeClassify(ctx context.Context, cmd *ports.QuickCommand, emailID int64) (string, error) {
	if emailID == 0 {
		return "", fmt.Errorf("selecione um email primeiro (tecla 'a' no inbox)")
	}

	var category, err = s.ClassifyEmail(ctx, emailID)
	if err != nil {
		return "", err
	}

	var descriptions = map[string]string{
		"importante":  "Requer atenção/resposta urgente",
		"pessoal":     "Comunicação pessoal",
		"trabalho":    "Comunicação profissional regular",
		"newsletter":  "Boletins informativos",
		"promocional": "Ofertas e marketing",
		"notificacao": "Alertas automáticos de sistemas",
		"spam":        "Não solicitado/suspeito",
	}

	var desc = descriptions[category]
	if desc == "" {
		desc = "Categoria identificada"
	}

	return fmt.Sprintf("Classificação: %s\n%s", strings.ToUpper(category), desc), nil
}

// executeHelp shows available commands
func (s *AIService) executeHelp(ctx context.Context, cmd *ports.QuickCommand) (string, error) {
	var sb strings.Builder
	sb.WriteString("Comandos rápidos disponíveis:\n\n")

	for _, c := range availableCommands() {
		var aliases = ""
		if len(c.Aliases) > 0 {
			aliases = fmt.Sprintf(" (ou /%s)", strings.Join(c.Aliases, ", /"))
		}
		var emailHint = ""
		if c.NeedsEmail {
			emailHint = " [requer email]"
		}
		sb.WriteString(fmt.Sprintf("  %s%s\n", c.Usage, aliases))
		sb.WriteString(fmt.Sprintf("    %s%s\n\n", c.Description, emailHint))
	}

	sb.WriteString("Dica: Tecla 'a' abre a AI com o email selecionado. 'A' abre sem contexto.")

	return sb.String(), nil
}
