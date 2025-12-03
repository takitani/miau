package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
