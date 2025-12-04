package imap

import (
	"encoding/base64"
	"fmt"
	"strings"
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
	UID       uint32
	MessageID string
	Subject   string
	From      string
	FromEmail string
	To        string
	Date      time.Time
	Seen      bool
	Flagged   bool
	Size      int64
	BodyText  string
}

// Connect estabelece conexão IMAP com a conta
func Connect(account *config.Account) (*Client, error) {
	var addr = fmt.Sprintf("%s:%d", account.IMAP.Host, account.IMAP.Port)

	var options = &imapclient.Options{}

	var client *imapclient.Client
	var err error

	if account.IMAP.TLS {
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
	var tokenPath = auth.GetTokenPath(config.GetConfigPath(), account.Name)
	var oauthCfg = auth.GetOAuth2Config(account.OAuth2.ClientID, account.OAuth2.ClientSecret)

	var token, err = auth.GetValidToken(oauthCfg, tokenPath)
	if err != nil {
		// Token não existe ou inválido - inicia fluxo de autenticação
		fmt.Println("\n🔐 Token OAuth2 não encontrado. Iniciando autenticação...")
		token, err = auth.AuthenticateWithBrowser(oauthCfg)
		if err != nil {
			return fmt.Errorf("erro na autenticação OAuth2: %w", err)
		}

		// Salva o token para uso futuro
		if err := auth.SaveToken(tokenPath, token); err != nil {
			return fmt.Errorf("erro ao salvar token: %w", err)
		}
		fmt.Println("✓ Token OAuth2 salvo com sucesso!")
	}

	var saslClient = newXOAuth2Client(account.Email, token.AccessToken)
	return client.Authenticate(saslClient)
}

// ListMailboxes lista todas as pastas (rápido, sem status)
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

		// Só pega status do INBOX pra acelerar boot
		if strings.EqualFold(mbox.Mailbox, "INBOX") {
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
		}

		mailboxes = append(mailboxes, mb)
	}

	return mailboxes, nil
}

// SelectMailbox seleciona uma mailbox e retorna info
func (c *Client) SelectMailbox(name string) (*imap.SelectData, error) {
	return c.client.Select(name, nil).Wait()
}

// FetchEmailsSeqNum busca emails por sequence number (mais confiável)
func (c *Client) FetchEmailsSeqNum(selectData *imap.SelectData, limit int) ([]Email, error) {
	var emails []Email

	if selectData.NumMessages == 0 {
		return emails, nil
	}

	// Calcula range de sequence numbers (mais recentes primeiro)
	var total = selectData.NumMessages
	var start uint32 = 1
	if total > uint32(limit) {
		start = total - uint32(limit) + 1
	}

	// Cria SeqSet do start até o final
	var seqSet = imap.SeqSet{}
	seqSet.AddRange(start, total)

	var fetchOptions = &imap.FetchOptions{
		Flags:       true,
		Envelope:    true,
		UID:         true,
		RFC822Size:  true,
		BodySection: []*imap.FetchItemBodySection{},
	}

	var fetchCmd = c.client.Fetch(seqSet, fetchOptions)
	var messages, err = fetchCmd.Collect()
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar emails: %w", err)
	}

	for _, msg := range messages {
		var email = Email{
			UID:  uint32(msg.UID),
			Size: msg.RFC822Size,
		}

		if msg.Envelope != nil {
			email.Subject = msg.Envelope.Subject
			email.Date = msg.Envelope.Date
			email.MessageID = msg.Envelope.MessageID

			if len(msg.Envelope.From) > 0 {
				var from = msg.Envelope.From[0]
				if from.Name != "" {
					email.From = from.Name
				} else {
					email.From = from.Mailbox
				}
				email.FromEmail = fmt.Sprintf("%s@%s", from.Mailbox, from.Host)
			}

			if len(msg.Envelope.To) > 0 {
				var to = msg.Envelope.To[0]
				email.To = fmt.Sprintf("%s@%s", to.Mailbox, to.Host)
			}
		}

		for _, flag := range msg.Flags {
			if flag == imap.FlagSeen {
				email.Seen = true
			}
			if flag == imap.FlagFlagged {
				email.Flagged = true
			}
		}

		emails = append(emails, email)
	}

	// Inverte para mais recentes primeiro
	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	return emails, nil
}

// FetchEmailsSince busca emails desde uma data específica (0 = todos)
func (c *Client) FetchEmailsSince(sinceDays int) ([]Email, error) {
	var emails []Email

	var criteria = &imap.SearchCriteria{}

	// Se sinceDays > 0, filtra por data
	if sinceDays > 0 {
		var sinceDate = time.Now().AddDate(0, 0, -sinceDays)
		criteria.Since = sinceDate
	}

	var searchCmd = c.client.Search(criteria, nil)
	var searchData, err = searchCmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar UIDs: %w", err)
	}

	var seqNums = searchData.AllSeqNums()
	if len(seqNums) == 0 {
		return emails, nil
	}

	// Cria SeqSet com todos os resultados
	var seqSet = imap.SeqSet{}
	for _, seq := range seqNums {
		seqSet.AddNum(seq)
	}

	var fetchOptions = &imap.FetchOptions{
		Flags:       true,
		Envelope:    true,
		UID:         true,
		RFC822Size:  true,
		BodySection: []*imap.FetchItemBodySection{},
	}

	var fetchCmd = c.client.Fetch(seqSet, fetchOptions)
	var messages, err2 = fetchCmd.Collect()
	if err2 != nil {
		return nil, fmt.Errorf("erro ao buscar emails: %w", err2)
	}

	for _, msg := range messages {
		var email = Email{
			UID:  uint32(msg.UID),
			Size: msg.RFC822Size,
		}

		if msg.Envelope != nil {
			email.Subject = msg.Envelope.Subject
			email.Date = msg.Envelope.Date
			email.MessageID = msg.Envelope.MessageID

			if len(msg.Envelope.From) > 0 {
				var from = msg.Envelope.From[0]
				if from.Name != "" {
					email.From = from.Name
				} else {
					email.From = from.Mailbox
				}
				email.FromEmail = fmt.Sprintf("%s@%s", from.Mailbox, from.Host)
			}

			if len(msg.Envelope.To) > 0 {
				var to = msg.Envelope.To[0]
				email.To = fmt.Sprintf("%s@%s", to.Mailbox, to.Host)
			}
		}

		for _, flag := range msg.Flags {
			if flag == imap.FlagSeen {
				email.Seen = true
			}
			if flag == imap.FlagFlagged {
				email.Flagged = true
			}
		}

		emails = append(emails, email)
	}

	// Inverte para mais recentes primeiro
	for i, j := 0, len(emails)-1; i < j; i, j = i+1, j-1 {
		emails[i], emails[j] = emails[j], emails[i]
	}

	return emails, nil
}

// FetchEmails busca emails (wrapper para compatibilidade)
func (c *Client) FetchEmails(limit int) ([]Email, error) {
	// Primeiro seleciona INBOX se não selecionou
	var selectData, err = c.client.Select("INBOX", nil).Wait()
	if err != nil {
		return nil, err
	}
	return c.FetchEmailsSeqNum(selectData, limit)
}

// FetchNewEmails busca emails com UID maior que o especificado
func (c *Client) FetchNewEmails(sinceUID uint32, limit int) ([]Email, error) {
	var emails []Email

	// Busca UIDs maiores que sinceUID
	var criteria = &imap.SearchCriteria{
		UID: []imap.UIDSet{{imap.UIDRange{Start: imap.UID(sinceUID + 1), Stop: 0}}},
	}

	var searchCmd = c.client.UIDSearch(criteria, nil)
	var searchData, err = searchCmd.Wait()
	if err != nil {
		return nil, err
	}

	var uids = searchData.AllUIDs()
	if len(uids) == 0 {
		return emails, nil
	}

	// Limita
	if len(uids) > limit {
		uids = uids[len(uids)-limit:]
	}

	var uidSet = imap.UIDSet{}
	for _, uid := range uids {
		uidSet.AddNum(uid)
	}

	var fetchOptions = &imap.FetchOptions{
		Flags:      true,
		Envelope:   true,
		UID:        true,
		RFC822Size: true,
	}

	var fetchCmd = c.client.Fetch(uidSet, fetchOptions)
	var messages, err2 = fetchCmd.Collect()
	if err2 != nil {
		return nil, err2
	}

	for _, msg := range messages {
		var email = Email{
			UID:  uint32(msg.UID),
			Size: msg.RFC822Size,
		}

		if msg.Envelope != nil {
			email.Subject = msg.Envelope.Subject
			email.Date = msg.Envelope.Date
			email.MessageID = msg.Envelope.MessageID

			if len(msg.Envelope.From) > 0 {
				var from = msg.Envelope.From[0]
				if from.Name != "" {
					email.From = from.Name
				} else {
					email.From = from.Mailbox
				}
				email.FromEmail = fmt.Sprintf("%s@%s", from.Mailbox, from.Host)
			}
		}

		for _, flag := range msg.Flags {
			if flag == imap.FlagSeen {
				email.Seen = true
			}
			if flag == imap.FlagFlagged {
				email.Flagged = true
			}
		}

		emails = append(emails, email)
	}

	return emails, nil
}

// FetchEmailBody busca o corpo completo de um email (TEXT part)
func (c *Client) FetchEmailBody(uid uint32) (string, error) {
	var uidSet = imap.UIDSet{}
	uidSet.AddNum(imap.UID(uid))

	var bodySection = &imap.FetchItemBodySection{
		Specifier: imap.PartSpecifierText,
	}

	var fetchOptions = &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}

	var fetchCmd = c.client.Fetch(uidSet, fetchOptions)
	var messages, err = fetchCmd.Collect()
	if err != nil {
		return "", err
	}

	if len(messages) == 0 {
		return "", fmt.Errorf("email não encontrado")
	}

	var msg = messages[0]
	var body = msg.FindBodySection(bodySection)
	if body != nil {
		return string(body), nil
	}

	return "", nil
}

// FetchEmailRaw busca o email completo (RFC822) para parsing
func (c *Client) FetchEmailRaw(uid uint32) ([]byte, error) {
	var uidSet = imap.UIDSet{}
	uidSet.AddNum(imap.UID(uid))

	var bodySection = &imap.FetchItemBodySection{}

	var fetchOptions = &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}

	var fetchCmd = c.client.Fetch(uidSet, fetchOptions)
	var messages, err = fetchCmd.Collect()
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("email não encontrado")
	}

	var msg = messages[0]
	var body = msg.FindBodySection(bodySection)
	if body != nil {
		return body, nil
	}

	return nil, nil
}

// MarkAsRead marca um email como lido no servidor
func (c *Client) MarkAsRead(uid uint32) error {
	var uidSet = imap.UIDSet{}
	uidSet.AddNum(imap.UID(uid))

	var storeFlags = imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: true,
		Flags:  []imap.Flag{imap.FlagSeen},
	}

	var _, err = c.client.Store(uidSet, &storeFlags, nil).Collect()
	return err
}

// GetAllUIDs retorna todos os UIDs da mailbox selecionada
func (c *Client) GetAllUIDs() ([]uint32, error) {
	var criteria = &imap.SearchCriteria{}
	var searchCmd = c.client.UIDSearch(criteria, nil)
	var searchData, err = searchCmd.Wait()
	if err != nil {
		return nil, err
	}

	var uids = searchData.AllUIDs()
	var result = make([]uint32, len(uids))
	for i, uid := range uids {
		result[i] = uint32(uid)
	}
	return result, nil
}

// MoveToFolder move um email para outra pasta usando MOVE ou COPY+DELETE
func (c *Client) MoveToFolder(uid uint32, targetFolder string) error {
	var uidSet = imap.UIDSet{}
	uidSet.AddNum(imap.UID(uid))

	// Tenta usar MOVE primeiro (mais eficiente, suportado pela maioria dos servidores modernos)
	var moveCmd = c.client.Move(uidSet, targetFolder)
	var _, err = moveCmd.Wait()
	if err == nil {
		return nil
	}

	// Fallback: COPY + DELETE
	var copyCmd = c.client.Copy(uidSet, targetFolder)
	if _, err = copyCmd.Wait(); err != nil {
		return fmt.Errorf("erro ao copiar email: %w", err)
	}

	// Marca como deleted
	var storeFlags = imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: true,
		Flags:  []imap.Flag{imap.FlagDeleted},
	}
	if _, err = c.client.Store(uidSet, &storeFlags, nil).Collect(); err != nil {
		return fmt.Errorf("erro ao marcar como deletado: %w", err)
	}

	// Expunge
	var _, err2 = c.client.Expunge().Collect()
	return err2
}

// ArchiveEmail arquiva um email (remove do INBOX, mantém em All Mail)
// Para Gmail: remove da pasta atual (fica automaticamente em All Mail)
// Para outros: move para Archive ou All Mail
func (c *Client) ArchiveEmail(uid uint32) error {
	var uidSet = imap.UIDSet{}
	uidSet.AddNum(imap.UID(uid))

	// Para Gmail, basta deletar do INBOX que o email permanece em All Mail
	// Marca como deleted e expunge
	var storeFlags = imap.StoreFlags{
		Op:     imap.StoreFlagsAdd,
		Silent: true,
		Flags:  []imap.Flag{imap.FlagDeleted},
	}

	if _, err := c.client.Store(uidSet, &storeFlags, nil).Collect(); err != nil {
		return fmt.Errorf("erro ao marcar como deletado: %w", err)
	}

	var _, err = c.client.Expunge().Collect()
	return err
}

// TrashEmail move um email para a lixeira
// Para Gmail: move para [Gmail]/Trash
// Para outros: move para Trash ou Deleted Items
func (c *Client) TrashEmail(uid uint32, trashFolder string) error {
	if trashFolder == "" {
		trashFolder = "[Gmail]/Trash"
	}
	return c.MoveToFolder(uid, trashFolder)
}

// GetTrashFolder tenta detectar a pasta de lixeira
func (c *Client) GetTrashFolder() string {
	var listCmd = c.client.List("", "*", nil)
	var list, err = listCmd.Collect()
	if err != nil {
		return "[Gmail]/Trash"
	}

	var trashNames = []string{
		"[Gmail]/Trash",
		"Trash",
		"Deleted Items",
		"Deleted",
		"[Gmail]/Lixeira",
		"Lixeira",
	}

	for _, name := range trashNames {
		for _, mbox := range list {
			if strings.EqualFold(mbox.Mailbox, name) {
				return mbox.Mailbox
			}
		}
	}

	return "[Gmail]/Trash"
}

// Close fecha a conexão
func (c *Client) Close() error {
	return c.client.Close()
}
