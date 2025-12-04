package adapters

import (
	"context"

	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/ports"
	"github.com/opik/miau/internal/smtp"
)

// SMTPAdapter wraps smtp.Client to implement ports.SMTPPort
type SMTPAdapter struct {
	client *smtp.Client
}

// NewSMTPAdapter creates a new SMTPAdapter
func NewSMTPAdapter(account *config.Account) *SMTPAdapter {
	return &SMTPAdapter{
		client: smtp.NewClient(account),
	}
}

// Send sends an email via SMTP
func (a *SMTPAdapter) Send(ctx context.Context, req *ports.SendRequest) (*ports.SendResult, error) {
	// Determine body to send (prefer HTML if available)
	var body = req.BodyText
	var isHTML = false
	if req.BodyHTML != "" {
		body = req.BodyHTML
		isHTML = true
	}

	// Convert ports.SendRequest to smtp.Email
	var email = &smtp.Email{
		To:             req.To,
		Cc:             req.Cc,
		Bcc:            req.Bcc,
		Subject:        req.Subject,
		Body:           body,
		InReplyTo:      req.InReplyTo,
		References:     req.ReferenceIDs,
		Classification: req.Classification,
		IsHTML:         isHTML,
	}

	var result, err = a.client.Send(email)
	if err != nil {
		return nil, err
	}

	return &ports.SendResult{
		Success:   true,
		MessageID: result.MessageID,
	}, nil
}
