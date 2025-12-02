package imap

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-sasl"
	"github.com/opik/miau/internal/auth"
	"github.com/opik/miau/internal/config"
)

// xoauth2Client implementa sasl.Client para XOAUTH2
type xoauth2Client struct {
	username    string
	accessToken string
	done        bool
}

func newXOAuth2Client(username, accessToken string) sasl.Client {
	return &xoauth2Client{username: username, accessToken: accessToken}
}

func (c *xoauth2Client) Start() (string, []byte, error) {
	var response = fmt.Sprintf("user=%s\x01auth=Bearer %s\x01\x01", c.username, c.accessToken)
	c.done = true
	return "XOAUTH2", []byte(response), nil
}

func (c *xoauth2Client) Next(challenge []byte) ([]byte, error) {
	// Se recebemos um challenge, provavelmente é um erro
	// Decodifica para ver a mensagem
	if len(challenge) > 0 {
		var decoded, _ = base64.StdEncoding.DecodeString(string(challenge))
		return nil, fmt.Errorf("XOAUTH2 error: %s", string(decoded))
	}
	return nil, nil
}

type Client struct {
	client  *imapclient.Client
	account *config.Account
}

type Mailbox struct {
	Name     string
	Messages uint32
	Unseen   uint32
}

type Email struct {
	UID     uint32
	Subject string
	From    string
	Date    time.Time
	Seen    bool
}

// Connect estabelece conexão IMAP com a conta
func Connect(account *config.Account) (*Client, error) {
	var addr = fmt.Sprintf("%s:%d", account.Imap.Host, account.Imap.Port)

	var options = &imapclient.Options{}

	var client *imapclient.Client
	var err error

	if account.Imap.TLS {
		client, err = imapclient.DialTLS(addr, options)
	} else {
		client, err = imapclient.DialInsecure(addr, options)
	}

	if err != nil {
		return nil, fmt.Errorf("erro ao conectar: %w", err)
	}

	// Autenticar
	if account.AuthType == config.AuthTypeOAuth2 {
		err = authenticateOAuth2(client, account)
	} else {
		err = authenticatePassword(client, account)
	}

	if err != nil {
		client.Close()
		return nil, fmt.Errorf("erro na autenticação: %w", err)
	}

	return &Client{
		client:  client,
		account: account,
	}, nil
}

func authenticatePassword(client *imapclient.Client, account *config.Account) error {
	return client.Login(account.Email, account.Password).Wait()
}

func authenticateOAuth2(client *imapclient.Client, account *config.Account) error {
	// Carrega token
	var tokenPath = auth.GetTokenPath(config.GetConfigPath(), account.Name)
	var oauthCfg = auth.GetOAuth2Config(account.OAuth2.ClientID, account.OAuth2.ClientSecret)

	var token, err = auth.GetValidToken(oauthCfg, tokenPath)
	if err != nil {
		return fmt.Errorf("erro ao obter token: %w", err)
	}

	// XOAUTH2
	var saslClient = newXOAuth2Client(account.Email, token.AccessToken)
	return client.Authenticate(saslClient)
}

// ListMailboxes lista todas as pastas
func (c *Client) ListMailboxes() ([]Mailbox, error) {
	var mailboxes []Mailbox

	var listCmd = c.client.List("", "*", nil)
	var list, err = listCmd.Collect()
	if err != nil {
		return nil, err
	}

	for _, mbox := range list {
		var mb = Mailbox{
			Name: mbox.Mailbox,
		}

		// Tenta obter status da mailbox
		var statusCmd = c.client.Status(mbox.Mailbox, &imap.StatusOptions{
			NumMessages: true,
			NumUnseen:   true,
		})
		var status, err = statusCmd.Wait()
		if err == nil {
			if status.NumMessages != nil {
				mb.Messages = *status.NumMessages
			}
			if status.NumUnseen != nil {
				mb.Unseen = *status.NumUnseen
			}
		}

		mailboxes = append(mailboxes, mb)
	}

	return mailboxes, nil
}

// SelectMailbox seleciona uma mailbox
func (c *Client) SelectMailbox(name string) (*imap.SelectData, error) {
	return c.client.Select(name, nil).Wait()
}

// FetchEmails busca emails da mailbox selecionada
func (c *Client) FetchEmails(limit int) ([]Email, error) {
	var emails []Email

	// Busca os UIDs primeiro
	var searchCmd = c.client.Search(&imap.SearchCriteria{}, nil)
	var searchData, err = searchCmd.Wait()
	if err != nil {
		return nil, err
	}

	var uids = searchData.AllUIDs()
	if len(uids) == 0 {
		return emails, nil
	}

	// Limita quantidade
	var start = 0
	if len(uids) > limit {
		start = len(uids) - limit
	}
	var recentUIDs = uids[start:]

	// Inverte para mostrar mais recentes primeiro
	for i, j := 0, len(recentUIDs)-1; i < j; i, j = i+1, j-1 {
		recentUIDs[i], recentUIDs[j] = recentUIDs[j], recentUIDs[i]
	}

	// Busca dados dos emails
	var uidSet = imap.UIDSet{}
	for _, uid := range recentUIDs {
		uidSet.AddNum(uid)
	}

	var fetchOptions = &imap.FetchOptions{
		Flags:    true,
		Envelope: true,
		UID:      true,
	}

	var fetchCmd = c.client.Fetch(uidSet, fetchOptions)
	var messages, err2 = fetchCmd.Collect()
	if err2 != nil {
		return nil, err2
	}

	for _, msg := range messages {
		var email = Email{
			UID: uint32(msg.UID),
		}

		if msg.Envelope != nil {
			email.Subject = msg.Envelope.Subject
			email.Date = msg.Envelope.Date
			if len(msg.Envelope.From) > 0 {
				var from = msg.Envelope.From[0]
				if from.Name != "" {
					email.From = from.Name
				} else {
					email.From = fmt.Sprintf("%s@%s", from.Mailbox, from.Host)
				}
			}
		}

		// Verifica flag Seen
		for _, flag := range msg.Flags {
			if flag == imap.FlagSeen {
				email.Seen = true
				break
			}
		}

		emails = append(emails, email)
	}

	return emails, nil
}

// Close fecha a conexão
func (c *Client) Close() error {
	return c.client.Close()
}
