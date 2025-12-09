package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/oauth2"
)

// SendAs representa uma configuração de "enviar como" do Gmail
type SendAs struct {
	SendAsEmail     string `json:"sendAsEmail"`
	DisplayName     string `json:"displayName"`
	ReplyToAddress  string `json:"replyToAddress"`
	Signature       string `json:"signature"`
	IsDefault       bool   `json:"isDefault"`
	IsPrimary       bool   `json:"isPrimary"`
	TreatAsAlias    bool   `json:"treatAsAlias"`
	VerificationStatus string `json:"verificationStatus"`
}

// SendAsListResponse é a resposta da API sendAs.list
type SendAsListResponse struct {
	SendAs []SendAs `json:"sendAs"`
}

// Client é um cliente para a Gmail API
type Client struct {
	httpClient *http.Client
	email      string
}

// NewClient cria um novo cliente Gmail API com token OAuth2
func NewClient(token *oauth2.Token, cfg *oauth2.Config, email string) *Client {
	var ctx = context.Background()
	var httpClient = cfg.Client(ctx, token)
	return &Client{
		httpClient: httpClient,
		email:      email,
	}
}

// HTTPClient returns the underlying HTTP client for use with other Google APIs
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// GetSignature busca a assinatura do email primário do usuário
func (c *Client) GetSignature() (string, error) {
	var sendAsList, err = c.listSendAs()
	if err != nil {
		return "", err
	}

	// Procura a configuração primária ou padrão
	for _, sa := range sendAsList.SendAs {
		if sa.IsPrimary || sa.IsDefault || sa.SendAsEmail == c.email {
			return sa.Signature, nil
		}
	}

	// Se não encontrou, retorna a primeira
	if len(sendAsList.SendAs) > 0 {
		return sendAsList.SendAs[0].Signature, nil
	}

	return "", fmt.Errorf("nenhuma assinatura encontrada")
}

// GetSendAsConfig busca a configuração completa de envio
func (c *Client) GetSendAsConfig() (*SendAs, error) {
	var sendAsList, err = c.listSendAs()
	if err != nil {
		return nil, err
	}

	// Procura a configuração primária ou padrão
	for _, sa := range sendAsList.SendAs {
		if sa.IsPrimary || sa.IsDefault || sa.SendAsEmail == c.email {
			return &sa, nil
		}
	}

	if len(sendAsList.SendAs) > 0 {
		return &sendAsList.SendAs[0], nil
	}

	return nil, fmt.Errorf("nenhuma configuração de envio encontrada")
}

func (c *Client) listSendAs() (*SendAsListResponse, error) {
	var url = "https://gmail.googleapis.com/gmail/v1/users/me/settings/sendAs"

	var resp, err = c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(body))
	}

	var result SendAsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &result, nil
}

// === EMAIL SENDING ===

// SendRequest representa a requisição de envio de email
type SendRequest struct {
	To              []string
	Cc              []string
	Bcc             []string
	Subject         string
	Body            string
	InReplyTo       string
	References      string
	IsHTML          bool
	ClassificationID string // ID do label de classificação (ex: "Label_123")
}

// SendResponse representa a resposta do envio
type SendResponse struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
}

// ClassificationLabelValue representa um valor de classificação
type ClassificationLabelValue struct {
	LabelID string                        `json:"labelId"`
	Fields  []ClassificationLabelField    `json:"fields,omitempty"`
}

// ClassificationLabelField representa um campo de classificação
type ClassificationLabelField struct {
	FieldID   string `json:"fieldId"`
	Selection string `json:"selection,omitempty"`
}

// GmailMessage representa uma mensagem para a API
type GmailMessage struct {
	Raw                       string                     `json:"raw"`
	ThreadID                  string                     `json:"threadId,omitempty"`
	LabelIDs                  []string                   `json:"labelIds,omitempty"`
	ClassificationLabelValues []ClassificationLabelValue `json:"classificationLabelValues,omitempty"`
}

// SendMessage envia um email usando a Gmail API
func (c *Client) SendMessage(req *SendRequest) (*SendResponse, error) {
	// Constrói a mensagem RFC 2822
	var rawMessage = c.buildRFC2822Message(req)

	// Codifica em base64url
	var encoded = base64.URLEncoding.EncodeToString([]byte(rawMessage))

	// Monta o payload
	var message = GmailMessage{
		Raw: encoded,
	}

	// Se tem classificação, adiciona
	if req.ClassificationID != "" {
		message.ClassificationLabelValues = []ClassificationLabelValue{
			{
				LabelID: req.ClassificationID,
			},
		}
	}

	// Serializa
	var payload, err = json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("erro ao serializar mensagem: %w", err)
	}

	// Envia via API
	var url = "https://gmail.googleapis.com/gmail/v1/users/me/messages/send"
	var httpReq, err2 = http.NewRequest("POST", url, bytes.NewReader(payload))
	if err2 != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err2)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	var resp, err3 = c.httpClient.Do(httpReq)
	if err3 != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err3)
	}
	defer resp.Body.Close()

	var respBody, _ = io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(respBody))
	}

	var result SendResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &result, nil
}

// buildRFC2822Message constrói uma mensagem no formato RFC 2822
func (c *Client) buildRFC2822Message(req *SendRequest) string {
	var msg strings.Builder

	// Headers obrigatórios
	msg.WriteString(fmt.Sprintf("From: %s\r\n", c.email))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(req.To, ", ")))

	if len(req.Cc) > 0 {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(req.Cc, ", ")))
	}

	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", req.Subject))
	msg.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))

	// Headers de threading
	if req.InReplyTo != "" {
		msg.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", req.InReplyTo))
	}
	if req.References != "" {
		msg.WriteString(fmt.Sprintf("References: %s\r\n", req.References))
	}

	// MIME
	msg.WriteString("MIME-Version: 1.0\r\n")
	if req.IsHTML {
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}

	// Corpo
	msg.WriteString("\r\n")
	msg.WriteString(req.Body)

	return msg.String()
}

// ListClassificationLabels lista os labels de classificação disponíveis
func (c *Client) ListClassificationLabels() ([]Label, error) {
	var url = "https://gmail.googleapis.com/gmail/v1/users/me/labels"

	var resp, err = c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Labels []Label `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return result.Labels, nil
}

// Label representa um label do Gmail
type Label struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	Type                  string `json:"type"`
	MessageListVisibility string `json:"messageListVisibility,omitempty"`
	LabelListVisibility   string `json:"labelListVisibility,omitempty"`
}

// === ARCHIVE/TRASH OPERATIONS ===

// ModifyLabelsRequest representa requisição para modificar labels
type ModifyLabelsRequest struct {
	AddLabelIDs    []string `json:"addLabelIds,omitempty"`
	RemoveLabelIDs []string `json:"removeLabelIds,omitempty"`
}

// ArchiveMessage arquiva uma mensagem (remove label INBOX)
// No Gmail, arquivar = remover do INBOX, o email continua em "All Mail"
func (c *Client) ArchiveMessage(messageID string) error {
	var url = fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages/%s/modify", messageID)

	var req = ModifyLabelsRequest{
		RemoveLabelIDs: []string{"INBOX"},
	}

	var payload, err = json.Marshal(req)
	if err != nil {
		return fmt.Errorf("erro ao serializar: %w", err)
	}

	var httpReq, err2 = http.NewRequest("POST", url, bytes.NewReader(payload))
	if err2 != nil {
		return fmt.Errorf("erro ao criar requisição: %w", err2)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	var resp, err3 = c.httpClient.Do(httpReq)
	if err3 != nil {
		return fmt.Errorf("erro na requisição: %w", err3)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// TrashMessage move uma mensagem para a lixeira
func (c *Client) TrashMessage(messageID string) error {
	var url = fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages/%s/trash", messageID)

	var httpReq, err = http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("erro ao criar requisição: %w", err)
	}

	var resp, err2 = c.httpClient.Do(httpReq)
	if err2 != nil {
		return fmt.Errorf("erro na requisição: %w", err2)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// UntrashMessage remove uma mensagem da lixeira
func (c *Client) UntrashMessage(messageID string) error {
	var url = fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages/%s/untrash", messageID)

	var httpReq, err = http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("erro ao criar requisição: %w", err)
	}

	var resp, err2 = c.httpClient.Do(httpReq)
	if err2 != nil {
		return fmt.Errorf("erro na requisição: %w", err2)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// MessageInfo contains Gmail message ID and thread ID
type MessageInfo struct {
	ID       string
	ThreadID string
}

// ListMessagesResponse represents the response from messages.list
type ListMessagesResponse struct {
	Messages           []MessageInfo `json:"messages"`
	NextPageToken      string        `json:"nextPageToken"`
	ResultSizeEstimate int           `json:"resultSizeEstimate"`
}

// ListAllMessages lists all messages and returns their IDs and thread IDs
// Uses pagination to handle large mailboxes
func (c *Client) ListAllMessages(maxResults int, pageToken string) (*ListMessagesResponse, error) {
	var url = fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages?maxResults=%d", maxResults)
	if pageToken != "" {
		url += "&pageToken=" + pageToken
	}

	var resp, err = c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(body))
	}

	var result ListMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &result, nil
}

// GetMessageMetadata gets message metadata including threadId and Message-ID header
func (c *Client) GetMessageMetadata(gmailMsgID string) (rfc822MsgID string, threadID string, err error) {
	var url = fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages/%s?format=metadata&metadataHeaders=Message-ID", gmailMsgID)

	var resp, reqErr = c.httpClient.Get(url)
	if reqErr != nil {
		return "", "", fmt.Errorf("erro na requisição: %w", reqErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID       string `json:"id"`
		ThreadID string `json:"threadId"`
		Payload  struct {
			Headers []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"headers"`
		} `json:"payload"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	// Find Message-ID header
	for _, h := range result.Payload.Headers {
		if h.Name == "Message-ID" || h.Name == "Message-Id" {
			rfc822MsgID = strings.Trim(h.Value, "<>")
			break
		}
	}

	return rfc822MsgID, result.ThreadID, nil
}

// GetMessageByUID busca o messageID do Gmail dado um UID IMAP
// Usa o header X-GM-MSGID ou busca por Message-ID
func (c *Client) GetMessageIDByRFC822MsgID(rfc822MsgID string) (string, error) {
	var info, err = c.GetMessageInfoByRFC822MsgID(rfc822MsgID)
	if err != nil {
		return "", err
	}
	return info.ID, nil
}

// GetMessageInfoByRFC822MsgID busca informações da mensagem pelo Message-ID RFC822
// Retorna tanto o Gmail message ID quanto o thread ID
func (c *Client) GetMessageInfoByRFC822MsgID(rfc822MsgID string) (*MessageInfo, error) {
	if rfc822MsgID == "" {
		return nil, fmt.Errorf("Message-ID vazio")
	}

	// Remove < e > se presentes
	var msgID = strings.Trim(rfc822MsgID, "<>")

	// Busca pelo rfc822msgid
	var url = fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages?q=rfc822msgid:%s", msgID)

	var resp, err = c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Messages []struct {
			ID       string `json:"id"`
			ThreadID string `json:"threadId"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	if len(result.Messages) == 0 {
		return nil, fmt.Errorf("mensagem não encontrada")
	}

	return &MessageInfo{
		ID:       result.Messages[0].ID,
		ThreadID: result.Messages[0].ThreadID,
	}, nil
}

// MessageThreadInfo maps RFC822 Message-ID to Gmail thread ID
type MessageThreadInfo struct {
	RFC822MsgID string
	ThreadID    string
}

// BatchGetMessageMetadata fetches metadata for up to 100 messages in a single HTTP request
// Uses Gmail Batch API: POST https://www.googleapis.com/batch/gmail/v1
// Returns a map of RFC822 Message-ID -> ThreadID
func (c *Client) BatchGetMessageMetadata(messages []MessageInfo) (map[string]string, error) {
	if len(messages) == 0 {
		return make(map[string]string), nil
	}
	if len(messages) > 100 {
		return nil, fmt.Errorf("batch size exceeds 100 messages")
	}

	// Build multipart request body
	var boundary = fmt.Sprintf("batch_%d", time.Now().UnixNano())
	var body bytes.Buffer

	for i, msg := range messages {
		body.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		body.WriteString("Content-Type: application/http\r\n")
		body.WriteString(fmt.Sprintf("Content-ID: <%d>\r\n\r\n", i))
		body.WriteString(fmt.Sprintf("GET /gmail/v1/users/me/messages/%s?format=metadata&metadataHeaders=Message-ID\r\n\r\n", msg.ID))
	}
	body.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	// Create batch request
	var req, err = http.NewRequest("POST", "https://www.googleapis.com/batch/gmail/v1", &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create batch request: %w", err)
	}
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary))

	// Execute request
	var resp, respErr = c.httpClient.Do(req)
	if respErr != nil {
		return nil, fmt.Errorf("batch request failed: %w", respErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var respBody, _ = io.ReadAll(resp.Body)
		return nil, fmt.Errorf("batch request returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse multipart response
	var contentType = resp.Header.Get("Content-Type")
	var _, params, parseErr = mime.ParseMediaType(contentType)
	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse response content-type: %w", parseErr)
	}

	var respBoundary = params["boundary"]
	if respBoundary == "" {
		return nil, fmt.Errorf("no boundary in response content-type")
	}

	var result = make(map[string]string)
	var reader = multipart.NewReader(resp.Body, respBoundary)

	for i := 0; ; i++ {
		var part, partErr = reader.NextPart()
		if partErr == io.EOF {
			break
		}
		if partErr != nil {
			continue // Skip malformed parts
		}

		// Read part body (contains HTTP response)
		var partBody, _ = io.ReadAll(part)
		part.Close()

		// Parse the embedded HTTP response to extract JSON
		var jsonStart = bytes.Index(partBody, []byte("{"))
		if jsonStart == -1 {
			continue
		}
		var jsonData = partBody[jsonStart:]

		// Parse JSON response
		var msgResp struct {
			ID       string `json:"id"`
			ThreadID string `json:"threadId"`
			Payload  struct {
				Headers []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"headers"`
			} `json:"payload"`
		}

		if err := json.Unmarshal(jsonData, &msgResp); err != nil {
			continue
		}

		// Extract Message-ID header
		var rfc822MsgID string
		for _, h := range msgResp.Payload.Headers {
			if h.Name == "Message-ID" || h.Name == "Message-Id" {
				rfc822MsgID = strings.Trim(h.Value, "<>")
				break
			}
		}

		// Map RFC822 Message-ID to ThreadID
		if rfc822MsgID != "" && msgResp.ThreadID != "" {
			// Find the original message to get its threadID (from messages.list)
			if i < len(messages) {
				result[rfc822MsgID] = messages[i].ThreadID
			}
		}
	}

	return result, nil
}

// SyncAllThreadIDs fetches all messages from Gmail and returns their thread mappings
// Uses Gmail Batch API for efficient bulk retrieval (100 messages per request)
// Runs parallel batch requests for speed (Gmail allows ~100 QPS)
// Supports cancellation via context
func (c *Client) SyncAllThreadIDs(ctx context.Context, progressCallback func(processed, total int)) (map[string]string, error) {
	var result = make(map[string]string) // RFC822 Message-ID -> Thread ID
	var resultMu sync.Mutex

	// Phase 1: List all messages (this gives us Gmail IDs + ThreadIDs)
	var allMessages []MessageInfo
	var pageToken = ""
	var listingPage = 0

	for {
		// Check for cancellation
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		var listResp, err = c.ListAllMessages(500, pageToken) // Max 500 per page
		if err != nil {
			return result, fmt.Errorf("failed to list messages: %w", err)
		}

		allMessages = append(allMessages, listResp.Messages...)
		listingPage++

		if listResp.NextPageToken == "" {
			break
		}
		pageToken = listResp.NextPageToken

		// Progress update during listing phase (negative = listing phase indicator)
		// Second param shows current accumulated count
		if progressCallback != nil {
			progressCallback(-listingPage, len(allMessages))
		}
	}

	var total = len(allMessages)
	if progressCallback != nil {
		progressCallback(0, total)
	}

	// Check for cancellation before starting phase 2
	if ctx.Err() != nil {
		return result, ctx.Err()
	}

	// Phase 2: Parallel batch fetch Message-ID headers
	var batchSize = 100
	var parallelWorkers = 10 // Run 10 batches in parallel (~1000 msgs at once)
	var processed int64 = 0
	var cancelled int32 = 0

	// Create work channel
	type batchWork struct {
		start int
		end   int
		msgs  []MessageInfo
	}
	var workChan = make(chan batchWork, parallelWorkers*2)
	var wg sync.WaitGroup

	// Start workers
	for w := 0; w < parallelWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workChan {
				// Check for cancellation
				if atomic.LoadInt32(&cancelled) == 1 {
					continue // Drain channel without processing
				}

				// Retry logic
				var batchResult map[string]string
				var batchErr error
				for retry := 0; retry < 3; retry++ {
					if atomic.LoadInt32(&cancelled) == 1 {
						break
					}
					batchResult, batchErr = c.BatchGetMessageMetadata(work.msgs)
					if batchErr == nil {
						break
					}
					time.Sleep(time.Duration(1<<retry) * time.Second)
				}

				if batchErr == nil && atomic.LoadInt32(&cancelled) == 0 {
					resultMu.Lock()
					for msgID, threadID := range batchResult {
						result[msgID] = threadID
					}
					resultMu.Unlock()
				}

				// Update progress
				var newProcessed = atomic.AddInt64(&processed, int64(len(work.msgs)))
				if progressCallback != nil && atomic.LoadInt32(&cancelled) == 0 {
					progressCallback(int(newProcessed), total)
				}
			}
		}()
	}

	// Monitor for cancellation in a goroutine
	go func() {
		<-ctx.Done()
		atomic.StoreInt32(&cancelled, 1)
	}()

	// Send work to workers
	for i := 0; i < total; i += batchSize {
		// Check for cancellation before sending more work
		if ctx.Err() != nil {
			break
		}
		var end = i + batchSize
		if end > total {
			end = total
		}
		workChan <- batchWork{
			start: i,
			end:   end,
			msgs:  allMessages[i:end],
		}
	}
	close(workChan)

	// Wait for all workers to finish
	wg.Wait()

	if ctx.Err() != nil {
		return result, ctx.Err()
	}

	return result, nil
}

// === PEOPLE API (CONTACTS) ===

// Person representa um contato da People API
type Person struct {
	ResourceName  string         `json:"resourceName"`
	Etag          string         `json:"etag"`
	Names         []Name         `json:"names"`
	EmailAddresses []EmailAddress `json:"emailAddresses"`
	PhoneNumbers  []PhoneNumber  `json:"phoneNumbers"`
	Photos        []Photo        `json:"photos"`
	Metadata      PersonMetadata `json:"metadata"`
}

// Name representa um nome no People API
type Name struct {
	DisplayName       string `json:"displayName"`
	FamilyName        string `json:"familyName"`
	GivenName         string `json:"givenName"`
	DisplayNameLastFirst string `json:"displayNameLastFirst"`
	UnstructuredName  string `json:"unstructuredName"`
	Metadata          FieldMetadata `json:"metadata"`
}

// EmailAddress representa um email no People API
type EmailAddress struct {
	Value    string        `json:"value"`
	Type     string        `json:"type"`
	Metadata FieldMetadata `json:"metadata"`
}

// PhoneNumber representa um telefone no People API
type PhoneNumber struct {
	Value           string        `json:"value"`
	CanonicalForm   string        `json:"canonicalForm"`
	Type            string        `json:"type"`
	Metadata        FieldMetadata `json:"metadata"`
}

// Photo representa uma foto no People API
type Photo struct {
	URL      string        `json:"url"`
	Metadata FieldMetadata `json:"metadata"`
	Default  bool          `json:"default"`
}

// PersonMetadata metadados de um Person
type PersonMetadata struct {
	Sources       []Source `json:"sources"`
	ObjectType    string   `json:"objectType"`
	LinkedPeopleResourceNames []string `json:"linkedPeopleResourceNames"`
}

// FieldMetadata metadados de um campo
type FieldMetadata struct {
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
	Source   Source `json:"source"`
}

// Source representa a fonte de um campo
type Source struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// ListConnectionsResponse resposta da API connections.list
type ListConnectionsResponse struct {
	Connections       []Person `json:"connections"`
	NextPageToken     string   `json:"nextPageToken"`
	NextSyncToken     string   `json:"nextSyncToken"`
	TotalPeople       int      `json:"totalPeople"`
	TotalItems        int      `json:"totalItems"`
}

// ListContactsRequest requisição para listar contatos
type ListContactsRequest struct {
	PageSize      int    // 1-1000, default 100
	PageToken     string // para paginação
	SyncToken     string // para sync incremental
	PersonFields  string // campos a retornar (ex: "names,emailAddresses,phoneNumbers,photos")
}

// ListContacts lista os contatos do usuário via People API
func (c *Client) ListContacts(req *ListContactsRequest) (*ListConnectionsResponse, error) {
	var url = "https://people.googleapis.com/v1/people/me/connections"

	// Parametros default
	if req.PageSize == 0 {
		req.PageSize = 100
	}
	if req.PersonFields == "" {
		req.PersonFields = "names,emailAddresses,phoneNumbers,photos"
	}

	// Monta query params
	var params = fmt.Sprintf("?pageSize=%d&personFields=%s", req.PageSize, req.PersonFields)

	if req.PageToken != "" {
		params += fmt.Sprintf("&pageToken=%s", req.PageToken)
	}

	if req.SyncToken != "" {
		params += fmt.Sprintf("&syncToken=%s", req.SyncToken)
	}

	url += params

	var resp, err = c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(body))
	}

	var result ListConnectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &result, nil
}

// ListOtherContacts lista os "Other Contacts" (contatos sugeridos automaticamente)
// Estes são contatos extraídos automaticamente de emails enviados/recebidos
func (c *Client) ListOtherContacts(pageSize int, pageToken string) (*ListConnectionsResponse, error) {
	var url = "https://people.googleapis.com/v1/otherContacts"

	if pageSize == 0 {
		pageSize = 100
	}

	var params = fmt.Sprintf("?pageSize=%d&readMask=names,emailAddresses,phoneNumbers,photos", pageSize)

	if pageToken != "" {
		params += fmt.Sprintf("&pageToken=%s", pageToken)
	}

	url += params

	var resp, err = c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(body))
	}

	// OtherContacts usa "otherContacts" em vez de "connections"
	var rawResult struct {
		OtherContacts []Person `json:"otherContacts"`
		NextPageToken string   `json:"nextPageToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rawResult); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &ListConnectionsResponse{
		Connections:   rawResult.OtherContacts,
		NextPageToken: rawResult.NextPageToken,
	}, nil
}

// GetContact busca um contato específico por resourceName
func (c *Client) GetContact(resourceName string, personFields string) (*Person, error) {
	if personFields == "" {
		personFields = "names,emailAddresses,phoneNumbers,photos"
	}

	var url = fmt.Sprintf("https://people.googleapis.com/v1/%s?personFields=%s",
		resourceName, personFields)

	var resp, err = c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(body))
	}

	var result Person
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &result, nil
}

// DownloadPhoto faz download da foto de perfil de um contato
func (c *Client) DownloadPhoto(photoURL string) ([]byte, error) {
	var resp, err = c.httpClient.Get(photoURL)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro da API: %d", resp.StatusCode)
	}

	var data, err2 = io.ReadAll(resp.Body)
	if err2 != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err2)
	}

	return data, nil
}
