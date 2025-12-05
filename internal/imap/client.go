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
	UID        uint32
	MessageID  string
	Subject    string
	From       string
	FromEmail  string
	To         string
	Date       time.Time
	Seen       bool
	Flagged    bool
	Size       int64
	BodyText   string
	InReplyTo  string
	References string
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
			email.InReplyTo = msg.Envelope.InReplyTo

			// References é uma lista, concatenamos com espaços
			if len(msg.Envelope.References) > 0 {
				email.References = strings.Join(msg.Envelope.References, " ")
			}

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
			email.InReplyTo = msg.Envelope.InReplyTo

			// References é uma lista, concatenamos com espaços
			if len(msg.Envelope.References) > 0 {
				email.References = strings.Join(msg.Envelope.References, " ")
			}

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

// AttachmentInfo represents metadata about an email attachment
type AttachmentInfo struct {
	PartNumber  string // e.g., "1.2", "2"
	Filename    string
	ContentType string
	ContentID   string // for inline images (cid:xxx)
	Encoding    string // base64, quoted-printable, etc.
	Size        int64
	IsInline    bool
	Charset     string
}

// FetchAttachmentMetadata retrieves attachment metadata from BODYSTRUCTURE
func (c *Client) FetchAttachmentMetadata(uid uint32) ([]AttachmentInfo, bool, error) {
	var uidSet = imap.UIDSet{}
	uidSet.AddNum(imap.UID(uid))

	var fetchOptions = &imap.FetchOptions{
		UID:           true,
		BodyStructure: &imap.FetchItemBodyStructure{Extended: true},
	}

	var fetchCmd = c.client.Fetch(uidSet, fetchOptions)
	var messages, err = fetchCmd.Collect()
	if err != nil {
		return nil, false, fmt.Errorf("erro ao buscar BODYSTRUCTURE: %w", err)
	}

	if len(messages) == 0 {
		return nil, false, fmt.Errorf("email não encontrado")
	}

	var msg = messages[0]
	if msg.BodyStructure == nil {
		return nil, false, nil
	}

	var attachments []AttachmentInfo
	var hasAttachments = false

	// Parse the body structure recursively
	parseBodyStructure(msg.BodyStructure, "", &attachments, &hasAttachments)

	return attachments, hasAttachments, nil
}

// parseBodyStructure recursively parses BODYSTRUCTURE to find attachments
func parseBodyStructure(bs imap.BodyStructure, partNum string, attachments *[]AttachmentInfo, hasAttachments *bool) {
	switch body := bs.(type) {
	case *imap.BodyStructureSinglePart:
		var info = AttachmentInfo{
			PartNumber:  partNum,
			ContentType: strings.ToLower(body.Type) + "/" + strings.ToLower(body.Subtype),
			Encoding:    strings.ToLower(body.Encoding),
			Size:        int64(body.Size),
		}

		// Content-ID is in the ID field (for inline images cid:xxx)
		if body.ID != "" {
			info.ContentID = body.ID
		}

		// Check for charset in params
		if body.Params != nil {
			if charset, ok := body.Params["charset"]; ok {
				info.Charset = charset
			}
		}

		// Use helper method for filename
		if body.Filename() != "" {
			info.Filename = body.Filename()
		}

		// Check disposition for inline vs attachment
		var disposition = body.Disposition()
		if disposition != nil {
			info.IsInline = strings.EqualFold(disposition.Value, "inline")
			// Also check disposition params for filename if not found yet
			if info.Filename == "" && disposition.Params != nil {
				if filename, ok := disposition.Params["filename"]; ok {
					info.Filename = filename
				}
			}
		}

		// Fallback: check params for name
		if info.Filename == "" && body.Params != nil {
			if name, ok := body.Params["name"]; ok {
				info.Filename = name
			}
		}

		// Determine if this is an attachment
		var isTextPlain = strings.HasPrefix(info.ContentType, "text/plain")
		var isTextHTML = strings.HasPrefix(info.ContentType, "text/html")
		var isMainBody = (isTextPlain || isTextHTML) && info.Filename == ""

		if !isMainBody {
			// It's an attachment or embedded content
			if info.Filename != "" || info.ContentID != "" || (!isTextPlain && !isTextHTML) {
				*hasAttachments = true
				// Only add if it has a filename or is an image/media type
				if info.Filename != "" || strings.HasPrefix(info.ContentType, "image/") ||
					strings.HasPrefix(info.ContentType, "audio/") ||
					strings.HasPrefix(info.ContentType, "video/") ||
					strings.HasPrefix(info.ContentType, "application/") {
					*attachments = append(*attachments, info)
				}
			}
		}

	case *imap.BodyStructureMultiPart:
		// Recursively process each part
		for i, part := range body.Children {
			var childPartNum string
			if partNum == "" {
				childPartNum = fmt.Sprintf("%d", i+1)
			} else {
				childPartNum = fmt.Sprintf("%s.%d", partNum, i+1)
			}
			parseBodyStructure(part, childPartNum, attachments, hasAttachments)
		}
	}
}

// FetchAttachmentPart fetches a specific body part (attachment content)
func (c *Client) FetchAttachmentPart(uid uint32, partNumber string) ([]byte, error) {
	var uidSet = imap.UIDSet{}
	uidSet.AddNum(imap.UID(uid))

	// Parse part number into section path
	var part []int
	if partNumber != "" {
		for _, s := range strings.Split(partNumber, ".") {
			var n int
			fmt.Sscanf(s, "%d", &n)
			part = append(part, n)
		}
	}

	var bodySection = &imap.FetchItemBodySection{
		Part: part,
	}

	var fetchOptions = &imap.FetchOptions{
		BodySection: []*imap.FetchItemBodySection{bodySection},
	}

	var fetchCmd = c.client.Fetch(uidSet, fetchOptions)
	var messages, err = fetchCmd.Collect()
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar parte do email: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("email não encontrado")
	}

	var msg = messages[0]
	var body = msg.FindBodySection(bodySection)
	if body == nil {
		return nil, fmt.Errorf("parte não encontrada: %s", partNumber)
	}

	return body, nil
}

// DecodeAttachmentContent decodes attachment content based on encoding
func DecodeAttachmentContent(data []byte, encoding string) ([]byte, error) {
	switch strings.ToLower(encoding) {
	case "base64":
		var decoded = make([]byte, base64.StdEncoding.DecodedLen(len(data)))
		var n, err = base64.StdEncoding.Decode(decoded, data)
		if err != nil {
			return nil, fmt.Errorf("erro ao decodificar base64: %w", err)
		}
		return decoded[:n], nil

	case "quoted-printable":
		// Simple quoted-printable decode
		var result []byte
		for i := 0; i < len(data); i++ {
			if data[i] == '=' && i+2 < len(data) {
				if data[i+1] == '\r' || data[i+1] == '\n' {
					// Soft line break
					i++
					if i+1 < len(data) && data[i] == '\r' && data[i+1] == '\n' {
						i++
					}
					continue
				}
				// Hex encoded byte
				var hex = string(data[i+1 : i+3])
				var b byte
				fmt.Sscanf(hex, "%02X", &b)
				result = append(result, b)
				i += 2
			} else {
				result = append(result, data[i])
			}
		}
		return result, nil

	case "7bit", "8bit", "binary", "":
		return data, nil

	default:
		// Unknown encoding, return as-is
		return data, nil
	}
}
