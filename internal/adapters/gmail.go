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
	client         *gmail.Client
	calendarClient *gmail.CalendarClient
	account        *config.Account
}

// NewGmailAPIAdapter creates a new GmailAPIAdapter
// Returns nil if OAuth2 is not configured or token is not available
func NewGmailAPIAdapter(account *config.Account, configDir string) *GmailAPIAdapter {
	fmt.Printf("[NewGmailAPIAdapter] Starting for account %s\n", account.Email)

	if account.AuthType != config.AuthTypeOAuth2 {
		fmt.Printf("[NewGmailAPIAdapter] Not OAuth2, skipping\n")
		return nil
	}

	if account.OAuth2 == nil {
		fmt.Printf("[NewGmailAPIAdapter] OAuth2 config is nil\n")
		return nil
	}

	var oauth2Cfg = auth.GetOAuth2Config(account.OAuth2.ClientID, account.OAuth2.ClientSecret)
	var tokenPath = auth.GetTokenPath(configDir, account.Email)
	fmt.Printf("[NewGmailAPIAdapter] Token path: %s\n", tokenPath)

	var token, err = auth.GetValidToken(oauth2Cfg, tokenPath)
	if err != nil {
		fmt.Printf("[NewGmailAPIAdapter] Failed to get valid token: %v\n", err)
		return nil
	}
	fmt.Printf("[NewGmailAPIAdapter] Token obtained successfully\n")

	var client = gmail.NewClient(token, oauth2Cfg, account.Email)

	// Create Calendar client (optional, may fail if scope not granted)
	var calendarClient *gmail.CalendarClient
	calendarClient, calErr := gmail.NewCalendarClient(context.Background(), client.HTTPClient())
	if calErr != nil {
		fmt.Printf("[GmailAPIAdapter] Failed to create Calendar client: %v\n", calErr)
	} else if calendarClient == nil {
		fmt.Printf("[GmailAPIAdapter] Calendar client is nil\n")
	} else {
		fmt.Printf("[GmailAPIAdapter] Calendar client created successfully\n")
	}

	return &GmailAPIAdapter{
		client:         client,
		calendarClient: calendarClient,
		account:        account,
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

// GetMessageInfoByRFC822MsgID returns Gmail message info (ID and ThreadID) by RFC822 Message-ID
func (a *GmailAPIAdapter) GetMessageInfoByRFC822MsgID(rfc822MsgID string) (*gmail.MessageInfo, error) {
	if a.client == nil {
		return nil, fmt.Errorf("Gmail API client not initialized")
	}

	return a.client.GetMessageInfoByRFC822MsgID(rfc822MsgID)
}

// SyncAllThreadIDs fetches all messages from Gmail and returns thread mappings
// Returns map of RFC822 Message-ID -> Gmail Thread ID
// Supports cancellation via context
func (a *GmailAPIAdapter) SyncAllThreadIDs(ctx context.Context, progressCallback func(processed, total int)) (map[string]string, error) {
	if a.client == nil {
		return nil, fmt.Errorf("Gmail API client not initialized")
	}

	return a.client.SyncAllThreadIDs(ctx, progressCallback)
}

// Client returns the underlying Gmail client
func (a *GmailAPIAdapter) Client() *gmail.Client {
	return a.client
}

// ContactsAdapter returns a GmailContactsPort adapter for the Gmail client
func (a *GmailAPIAdapter) ContactsAdapter() ports.GmailContactsPort {
	if a.client == nil {
		return nil
	}
	return gmail.NewContactsAdapter(a.client)
}

// CalendarClient returns the Google Calendar API client
func (a *GmailAPIAdapter) CalendarClient() *gmail.CalendarClient {
	return a.calendarClient
}
