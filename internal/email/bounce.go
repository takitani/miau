package email

import (
	"strings"
)

// BounceInfo contains information about a detected bounce
type BounceInfo struct {
	IsBounce   bool
	Reason     string
	Snippet    string
	FromEmail  string
	Subject    string
}

// DetectBounce checks if an email is a bounce/NDR message and extracts info
func DetectBounce(fromEmail, fromName, subject, snippet string) *BounceInfo {
	if !IsBounceEmail(fromEmail, fromName, subject) {
		return nil
	}

	return &BounceInfo{
		IsBounce:  true,
		Reason:    ExtractBounceReason(snippet, subject),
		Snippet:   snippet,
		FromEmail: fromEmail,
		Subject:   subject,
	}
}

// IsBounceEmail detects if an email is a bounce/NDR message
func IsBounceEmail(fromEmail, fromName, subject string) bool {
	var from = strings.ToLower(fromEmail + " " + fromName)
	var subj = strings.ToLower(subject)

	// Typical bounce senders
	var bounceSenders = []string{
		"mailer-daemon",
		"postmaster",
		"mail delivery subsystem",
		"mail delivery",
		"mailerdaemon",
		"noreply",
		"no-reply",
		"mail-daemon",
		"delivery",
		"daemon",
		"bounce",
		"mailmaster",
	}

	for _, sender := range bounceSenders {
		if strings.Contains(from, sender) {
			return true
		}
	}

	// Typical bounce subjects
	var bounceSubjects = []string{
		"delivery status notification",
		"delivery status",
		"delivery failed",
		"delivery failure",
		"undeliverable",
		"undelivered",
		"returned mail",
		"mail delivery failed",
		"failure notice",
		"não foi possível entregar",
		"falha na entrega",
		"mensagem devolvida",
		"não entregue",
		"rejected",
		"mail returned",
		"returned to sender",
		"could not be delivered",
		"notification (failure)",
		"(failure)",
	}

	for _, bs := range bounceSubjects {
		if strings.Contains(subj, bs) {
			return true
		}
	}

	return false
}

// ExtractBounceReason extracts the bounce reason from content
func ExtractBounceReason(snippet, subject string) string {
	var content = strings.ToLower(snippet + " " + subject)

	// Common reasons
	var reasons = map[string]string{
		"classificação":            "Requer classificação de email",
		"classification":           "Requires email classification",
		"spam":                     "Marcado como spam",
		"rejected":                 "Rejeitado pelo servidor",
		"user unknown":             "Usuário desconhecido",
		"mailbox full":             "Caixa de correio cheia",
		"quota exceeded":           "Cota excedida",
		"does not exist":           "Endereço não existe",
		"address rejected":         "Endereço rejeitado",
		"policy":                   "Violação de política",
		"blocked":                  "Bloqueado",
		"blacklist":                "Na lista negra",
		"administrador":            "Bloqueado pelo administrador",
		"administrator":            "Blocked by administrator",
		"enterprise administrator": "Bloqueado pela política corporativa",
	}

	for key, reason := range reasons {
		if strings.Contains(content, key) {
			return reason
		}
	}

	// If no specific reason found, return generic
	if len(snippet) > 100 {
		return snippet[:100] + "..."
	}
	return snippet
}
