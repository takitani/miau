package smtp

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"github.com/opik/miau/internal/config"
)

// Classificações de email disponíveis (Google Workspace labels)
var Classifications = []string{"Público", "Interno", "Externo", "Confidencial"}

// Email representa um email a ser enviado
type Email struct {
	To             []string
	Cc             []string
	Bcc            []string
	Subject        string
	Body           string // Corpo do email (HTML ou plain text)
	ReplyTo        string
	InReplyTo      string // Message-ID do email original (para threading)
	References     string // Chain de Message-IDs
	Classification string // Classificação do email (Public, Interno, etc)
	IsHTML         bool   // Se true, envia como text/html; senão text/plain
}

// Client é o cliente SMTP
type Client struct {
	account *config.Account
}

// NewClient cria um novo cliente SMTP
func NewClient(account *config.Account) *Client {
	return &Client{account: account}
}

// SendResult contém detalhes do envio
type SendResult struct {
	Host      string
	Port      int
	MessageID string
}

// Send envia um email
func (c *Client) Send(email *Email) (*SendResult, error) {
	var host = c.getHost()
	var port = c.getPort()
	var addr = fmt.Sprintf("%s:%d", host, port)

	// Gera Message-ID único
	var domain = c.account.Email[strings.Index(c.account.Email, "@")+1:]
	var messageID = fmt.Sprintf("<%d.%d@%s>", time.Now().UnixNano(), time.Now().Unix(), domain)

	// Headers do email (ordem importa para alguns servidores)
	var message strings.Builder
	message.WriteString(fmt.Sprintf("From: %s\r\n", c.formatFrom()))
	message.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(email.To, ", ")))
	if len(email.Cc) > 0 {
		message.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(email.Cc, ", ")))
	}
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", email.Subject))
	message.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	message.WriteString(fmt.Sprintf("Message-ID: %s\r\n", messageID))

	// Headers de threading para replies
	if email.InReplyTo != "" {
		message.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", email.InReplyTo))
	}
	if email.References != "" {
		message.WriteString(fmt.Sprintf("References: %s\r\n", email.References))
	}

	if email.ReplyTo != "" {
		message.WriteString(fmt.Sprintf("Reply-To: %s\r\n", email.ReplyTo))
	}

	// Headers de classificação (vários formatos para compatibilidade)
	var classification = email.Classification
	if classification == "" {
		classification = "Público"
	}
	// Google Workspace / genérico
	message.WriteString(fmt.Sprintf("X-Classification: %s\r\n", classification))
	message.WriteString(fmt.Sprintf("X-Google-Classification: %s\r\n", classification))
	// Microsoft / Exchange
	message.WriteString(fmt.Sprintf("Sensitivity: %s\r\n", classification))
	message.WriteString(fmt.Sprintf("X-MS-Exchange-Organization-Classification: %s\r\n", classification))
	// Outros sistemas DLP comuns
	message.WriteString(fmt.Sprintf("X-Message-Classification: %s\r\n", classification))
	message.WriteString(fmt.Sprintf("X-Data-Classification: %s\r\n", classification))
	message.WriteString("X-Priority: 3\r\n")

	message.WriteString("MIME-Version: 1.0\r\n")
	if email.IsHTML {
		message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	} else {
		message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	}
	message.WriteString("\r\n")
	message.WriteString(email.Body)

	// Destinatários (To + Cc + Bcc)
	var recipients []string
	recipients = append(recipients, email.To...)
	recipients = append(recipients, email.Cc...)
	recipients = append(recipients, email.Bcc...)

	// Autenticação
	var auth = smtp.PlainAuth("", c.account.Email, c.account.Password, host)

	// Conexão TLS
	var tlsConfig = &tls.Config{
		ServerName: host,
	}

	var conn, err = tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar a %s: %w", addr, err)
	}
	defer conn.Close()

	var client, err2 = smtp.NewClient(conn, host)
	if err2 != nil {
		return nil, fmt.Errorf("erro ao criar cliente SMTP: %w", err2)
	}
	defer client.Close()

	// Autentica
	if err := client.Auth(auth); err != nil {
		return nil, fmt.Errorf("erro de autenticação SMTP (%s): %w", host, err)
	}

	// Define remetente
	if err := client.Mail(c.account.Email); err != nil {
		return nil, fmt.Errorf("erro ao definir remetente: %w", err)
	}

	// Define destinatários
	for _, rcpt := range recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return nil, fmt.Errorf("servidor rejeitou destinatário %s: %w", rcpt, err)
		}
	}

	// Envia corpo
	var w, err3 = client.Data()
	if err3 != nil {
		return nil, fmt.Errorf("erro ao iniciar envio: %w", err3)
	}

	if _, err := w.Write([]byte(message.String())); err != nil {
		return nil, fmt.Errorf("erro ao enviar mensagem: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("servidor rejeitou mensagem: %w", err)
	}

	if err := client.Quit(); err != nil {
		// Quit error não é crítico, mensagem já foi aceita
	}

	return &SendResult{
		Host:      host,
		Port:      port,
		MessageID: messageID,
	}, nil
}

func (c *Client) getHost() string {
	if c.account.SMTP.Host != "" {
		return c.account.SMTP.Host
	}
	// Auto-detect para Gmail e Google Workspace
	if strings.Contains(c.account.Email, "@gmail.com") ||
		strings.Contains(c.account.Email, "@googlemail.com") ||
		c.account.IMAP.Host == "imap.gmail.com" {
		return "smtp.gmail.com"
	}
	// Tenta derivar do IMAP
	return strings.Replace(c.account.IMAP.Host, "imap.", "smtp.", 1)
}

func (c *Client) getPort() int {
	if c.account.SMTP.Port != 0 {
		return c.account.SMTP.Port
	}
	return 465 // SMTPS padrão
}

func (c *Client) formatFrom() string {
	if c.account.Name != "" {
		return fmt.Sprintf("%s <%s>", c.account.Name, c.account.Email)
	}
	return c.account.Email
}
