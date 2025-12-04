package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// GetMessageByUID busca o messageID do Gmail dado um UID IMAP
// Usa o header X-GM-MSGID ou busca por Message-ID
func (c *Client) GetMessageIDByRFC822MsgID(rfc822MsgID string) (string, error) {
	if rfc822MsgID == "" {
		return "", fmt.Errorf("Message-ID vazio")
	}

	// Remove < e > se presentes
	var msgID = strings.Trim(rfc822MsgID, "<>")

	// Busca pelo rfc822msgid
	var url = fmt.Sprintf("https://gmail.googleapis.com/gmail/v1/users/me/messages?q=rfc822msgid:%s", msgID)

	var resp, err = c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("erro na requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body, _ = io.ReadAll(resp.Body)
		return "", fmt.Errorf("erro da API (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Messages []struct {
			ID       string `json:"id"`
			ThreadID string `json:"threadId"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	if len(result.Messages) == 0 {
		return "", fmt.Errorf("mensagem não encontrada")
	}

	return result.Messages[0].ID, nil
}
