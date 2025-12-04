package adapters

import (
	"context"
	"fmt"

	"github.com/opik/miau/internal/auth"
	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/gmail"
	"github.com/opik/miau/internal/ports"
)

// GmailAPIAdapter wraps gmail.Client to implement ports.GmailAPIPort
type GmailAPIAdapter struct {
	client  *gmail.Client
	account *config.Account
}

// NewGmailAPIAdapter creates a new GmailAPIAdapter
// Returns nil if OAuth2 is not configured or token is not available
func NewGmailAPIAdapter(account *config.Account, configDir string) *GmailAPIAdapter {
	if account.AuthType != config.AuthTypeOAuth2 {
		return nil
	}

	if account.OAuth2 == nil {
		return nil
	}

	var oauth2Cfg = auth.GetOAuth2Config(account.OAuth2.ClientID, account.OAuth2.ClientSecret)
	var tokenPath = auth.GetTokenPath(configDir, account.Email)

	var token, err = auth.GetValidToken(oauth2Cfg, tokenPath)
	if err != nil {
		return nil
	}

	var client = gmail.NewClient(token, oauth2Cfg, account.Email)

	return &GmailAPIAdapter{
		client:  client,
		account: account,
	}
}

// Send sends an email via Gmail API
func (a *GmailAPIAdapter) Send(ctx context.Context, req *ports.SendRequest) (*ports.SendResult, error) {
	if a.client == nil {
		return nil, fmt.Errorf("Gmail API client not initialized")
	}

	// Determine body to send (prefer HTML if available)
	var body = req.BodyText
	var isHTML = false
	if req.BodyHTML != "" {
		body = req.BodyHTML
		isHTML = true
	}

	// Convert ports.SendRequest to gmail.SendRequest
	var gmailReq = &gmail.SendRequest{
		To:         req.To,
		Cc:         req.Cc,
		Bcc:        req.Bcc,
		Subject:    req.Subject,
		Body:       body,
		InReplyTo:  req.InReplyTo,
		References: req.ReferenceIDs,
		IsHTML:     isHTML,
	}

	var result, err = a.client.SendMessage(gmailReq)
	if err != nil {
		return nil, err
	}

	return &ports.SendResult{
		Success:   true,
		MessageID: result.ID,
	}, nil
}

// GetSignature retrieves the user's signature from Gmail
func (a *GmailAPIAdapter) GetSignature(ctx context.Context) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("Gmail API client not initialized")
	}

	return a.client.GetSignature()
}

// Archive archives an email by message ID
func (a *GmailAPIAdapter) Archive(ctx context.Context, messageID string) error {
	if a.client == nil {
		return fmt.Errorf("Gmail API client not initialized")
	}

	// Need to convert RFC822 message ID to Gmail message ID
	var gmailID, err = a.client.GetMessageIDByRFC822MsgID(messageID)
	if err != nil {
		return err
	}

	return a.client.ArchiveMessage(gmailID)
}
