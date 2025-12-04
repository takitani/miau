package main

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opik/miau/internal/auth"
	"github.com/opik/miau/internal/config"
	"github.com/opik/miau/internal/gmail"
	"github.com/opik/miau/internal/imap"
	"github.com/opik/miau/internal/tui/inbox"
	"github.com/opik/miau/internal/tui/setup"
)

// Estilos básicos
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B6B")).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF6B6B")).
			Padding(1, 2)
)

type appState int

const (
	stateSetup appState = iota
	stateInbox
)

type model struct {
	width      int
	height     int
	state      appState
	setupModel setup.Model
	inboxModel inbox.Model
	cfg        *config.Config
	debugMode  bool
}

func initialModel(debugMode bool) model {
	var m = model{debugMode: debugMode}

	// Verifica se já existe configuração
	if config.ConfigExists() {
		var cfg, err = config.Load()
		if err == nil && cfg != nil && len(cfg.Accounts) > 0 {
			m.state = stateInbox
			m.cfg = cfg
			m.inboxModel = inbox.New(&cfg.Accounts[0], debugMode)
			return m
		}
	}

	// Não existe config, iniciar setup
	m.state = stateSetup
	m.setupModel = setup.New()
	return m
}

func (m model) Init() tea.Cmd {
	if m.state == stateSetup {
		return m.setupModel.Init()
	}
	if m.state == stateInbox {
		return m.inboxModel.Init()
	}
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Delega para o modelo atual
	if m.state == stateSetup {
		var updatedSetup, cmd = m.setupModel.Update(msg)
		m.setupModel = updatedSetup.(setup.Model)

		// Verifica se setup terminou
		if m.setupModel.IsComplete() {
			// Recarrega config e inicia inbox
			config.Load() // força recarregar
			var cfg, _ = config.Load()
			m.cfg = cfg
			m.inboxModel = inbox.New(&cfg.Accounts[0], m.debugMode)
			m.state = stateInbox
			return m, m.inboxModel.Init()
		}

		return m, cmd
	}

	if m.state == stateInbox {
		var updatedInbox, cmd = m.inboxModel.Update(msg)
		m.inboxModel = updatedInbox.(inbox.Model)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	if m.state == stateSetup {
		return m.setupModel.View()
	}

	if m.state == stateInbox {
		return m.inboxModel.View()
	}

	return ""
}

func main() {
	// Comando para testar assinatura
	if len(os.Args) > 1 && os.Args[1] == "signature" {
		showSignature()
		return
	}

	// Comando para autenticação OAuth2
	if len(os.Args) > 1 && os.Args[1] == "auth" {
		runOAuth2Auth()
		return
	}

	// Verifica flag --debug
	var debugMode = false
	for _, arg := range os.Args[1:] {
		if arg == "--debug" || arg == "-d" {
			debugMode = true
			break
		}
	}

	var p = tea.NewProgram(initialModel(debugMode), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Erro ao iniciar miau: %v\n", err)
		os.Exit(1)
	}
}

func showSignature() {
	var cfg, err = config.Load()
	if err != nil || cfg == nil || len(cfg.Accounts) == 0 {
		fmt.Println("❌ Nenhuma conta configurada")
		os.Exit(1)
	}

	var account = &cfg.Accounts[0]

	// Tenta OAuth2 primeiro (Gmail API)
	if account.AuthType == config.AuthTypeOAuth2 {
		var tokenPath = auth.GetTokenPath(config.GetConfigPath(), account.Name)
		var oauthCfg = auth.GetOAuth2Config(account.OAuth2.ClientID, account.OAuth2.ClientSecret)
		var token, err2 = auth.GetValidToken(oauthCfg, tokenPath)
		if err2 == nil {
			var client = gmail.NewClient(token, oauthCfg, account.Email)
			var sendAs, err3 = client.GetSendAsConfig()
			if err3 == nil {
				fmt.Println("🐱 miau - Assinatura (via Gmail API)")
				fmt.Println("====================================")
				fmt.Printf("📧 Email: %s\n", sendAs.SendAsEmail)
				fmt.Printf("👤 Nome: %s\n", sendAs.DisplayName)
				fmt.Println("\n📝 Assinatura HTML:")
				fmt.Println("-------------------")
				if sendAs.Signature != "" {
					fmt.Println(sendAs.Signature)
				} else {
					fmt.Println("(sem assinatura configurada)")
				}
				return
			}
		}
	}

	// Fallback: extrai de email enviado via IMAP
	fmt.Println("🐱 miau - Extraindo assinatura dos enviados...")
	extractSignatureFromSent(account)
}

func extractSignatureFromSent(account *config.Account) {
	var client, err = imap.Connect(account)
	if err != nil {
		fmt.Printf("❌ Erro ao conectar: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Tenta encontrar a pasta de enviados
	var sentFolders = []string{"[Gmail]/Sent Mail", "[Gmail]/E-mails enviados", "Sent", "Sent Messages", "INBOX.Sent"}
	var sentFolder string

	var mailboxes, _ = client.ListMailboxes()
	for _, mb := range mailboxes {
		for _, sf := range sentFolders {
			if mb.Name == sf {
				sentFolder = mb.Name
				break
			}
		}
		if sentFolder != "" {
			break
		}
	}

	if sentFolder == "" {
		fmt.Println("❌ Pasta de enviados não encontrada")
		fmt.Println("Pastas disponíveis:")
		for _, mb := range mailboxes {
			fmt.Printf("  - %s\n", mb.Name)
		}
		os.Exit(1)
	}

	fmt.Printf("📁 Usando pasta: %s\n", sentFolder)

	// Seleciona e busca último email
	var selectData, err2 = client.SelectMailbox(sentFolder)
	if err2 != nil {
		fmt.Printf("❌ Erro ao selecionar pasta: %v\n", err2)
		os.Exit(1)
	}

	if selectData.NumMessages == 0 {
		fmt.Println("❌ Nenhum email enviado encontrado")
		os.Exit(1)
	}

	// Busca os últimos 5 emails para tentar encontrar assinatura
	var emails, err3 = client.FetchEmailsSeqNum(selectData, 5)
	if err3 != nil || len(emails) == 0 {
		fmt.Printf("❌ Erro ao buscar emails: %v\n", err3)
		os.Exit(1)
	}

	// Pega o email mais recente
	var email = emails[0]
	var rawData, err4 = client.FetchEmailRaw(email.UID)
	if err4 != nil {
		fmt.Printf("❌ Erro ao buscar conteúdo: %v\n", err4)
		os.Exit(1)
	}

	// Extrai HTML do email
	var htmlContent = extractHTMLFromRaw(rawData)
	if htmlContent == "" {
		fmt.Println("❌ Email não contém HTML")
		os.Exit(1)
	}

	// Tenta extrair assinatura (procura por padrões comuns)
	var signature = extractSignatureFromHTML(htmlContent)

	fmt.Println("\n📝 Assinatura encontrada:")
	fmt.Println("-------------------------")
	if signature != "" {
		fmt.Println(signature)
	} else {
		fmt.Println("⚠️  Não foi possível detectar assinatura automaticamente")
		fmt.Println("\nConteúdo HTML completo do último email:")
		fmt.Println(htmlContent)
	}
}

// extractHTMLFromRaw extrai conteúdo HTML de email raw
func extractHTMLFromRaw(rawData []byte) string {
	var msg, err = mail.ReadMessage(bytes.NewReader(rawData))
	if err != nil {
		return ""
	}

	var contentType = msg.Header.Get("Content-Type")
	var mediaType, params, _ = mime.ParseMediaType(contentType)

	if strings.HasPrefix(mediaType, "text/html") {
		var body, _ = io.ReadAll(msg.Body)
		return decodeBodyMain(body, msg.Header.Get("Content-Transfer-Encoding"))
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		var boundary = params["boundary"]
		if boundary != "" {
			return findHTMLPartMain(msg.Body, boundary)
		}
	}

	return ""
}

func findHTMLPartMain(r io.Reader, boundary string) string {
	var mr = multipart.NewReader(r, boundary)
	for {
		var part, err = mr.NextPart()
		if err != nil {
			break
		}

		var contentType = part.Header.Get("Content-Type")
		var mediaType, params, _ = mime.ParseMediaType(contentType)

		if strings.HasPrefix(mediaType, "text/html") {
			var body, _ = io.ReadAll(part)
			return decodeBodyMain(body, part.Header.Get("Content-Transfer-Encoding"))
		}

		if strings.HasPrefix(mediaType, "multipart/") {
			var boundary = params["boundary"]
			if boundary != "" {
				if html := findHTMLPartMain(part, boundary); html != "" {
					return html
				}
			}
		}
	}
	return ""
}

func decodeBodyMain(body []byte, encoding string) string {
	switch strings.ToLower(encoding) {
	case "quoted-printable":
		var decoded, err = io.ReadAll(quotedprintable.NewReader(bytes.NewReader(body)))
		if err != nil {
			return string(body)
		}
		return string(decoded)
	default:
		return string(body)
	}
}

// extractSignatureFromHTML tenta extrair assinatura de HTML
func extractSignatureFromHTML(html string) string {
	// Padrões comuns de assinatura:
	// 1. Div com class contendo "signature"
	// 2. Após "--" ou "-- " em linha separada
	// 3. Div com id contendo "signature"
	// 4. Gmail usa div com class="gmail_signature"

	// Tenta encontrar gmail_signature
	var gmailSigRegex = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*gmail_signature[^"]*"[^>]*>(.*?)</div>`)
	if matches := gmailSigRegex.FindStringSubmatch(html); len(matches) > 1 {
		return strings.TrimSpace(matches[0])
	}

	// Tenta encontrar qualquer div com "signature" no class ou id
	var sigRegex = regexp.MustCompile(`(?is)<div[^>]*(class|id)="[^"]*signature[^"]*"[^>]*>(.*?)</div>`)
	if matches := sigRegex.FindStringSubmatch(html); len(matches) > 0 {
		return strings.TrimSpace(matches[0])
	}

	// Procura por "-- " seguido de conteúdo (padrão de assinatura em texto)
	var dashRegex = regexp.MustCompile(`(?is)(<br\s*/?>|<p>)\s*--\s*(<br\s*/?>|</p>)(.*?)$`)
	if matches := dashRegex.FindStringSubmatch(html); len(matches) > 3 {
		return strings.TrimSpace(matches[3])
	}

	return ""
}

// runOAuth2Auth executa o fluxo de autenticação OAuth2 fora da TUI
func runOAuth2Auth() {
	var cfg, err = config.Load()
	if err != nil || cfg == nil || len(cfg.Accounts) == 0 {
		fmt.Println("❌ Nenhuma conta configurada")
		os.Exit(1)
	}

	var account = &cfg.Accounts[0]

	if account.AuthType != config.AuthTypeOAuth2 {
		fmt.Println("❌ Conta não está configurada para OAuth2")
		fmt.Printf("   auth_type atual: %s\n", account.AuthType)
		os.Exit(1)
	}

	if account.OAuth2 == nil {
		fmt.Println("❌ Configuração OAuth2 não encontrada")
		os.Exit(1)
	}

	fmt.Println("🐱 miau - Autenticação OAuth2")
	fmt.Println("=============================")
	fmt.Printf("📧 Conta: %s\n", account.Email)
	fmt.Printf("🔑 Client ID: %s...\n", account.OAuth2.ClientID[:20])

	var tokenPath = auth.GetTokenPath(config.GetConfigPath(), account.Name)
	var oauthCfg = auth.GetOAuth2Config(account.OAuth2.ClientID, account.OAuth2.ClientSecret)

	// Verifica se já tem token válido
	var existingToken, err2 = auth.GetValidToken(oauthCfg, tokenPath)
	if err2 == nil {
		fmt.Println("\n✓ Token OAuth2 já existe e é válido!")
		fmt.Printf("   Expira em: %s\n", existingToken.Expiry.Format("2006-01-02 15:04:05"))
		fmt.Println("\nDeseja renovar o token? (y/N)")

		var input string
		fmt.Scanln(&input)
		if input != "y" && input != "Y" {
			fmt.Println("Mantendo token existente.")
			return
		}
	}

	// Inicia fluxo de autenticação
	var token, err3 = auth.AuthenticateWithBrowser(oauthCfg)
	if err3 != nil {
		fmt.Printf("❌ Erro na autenticação: %v\n", err3)
		os.Exit(1)
	}

	// Salva token
	if err := auth.SaveToken(tokenPath, token); err != nil {
		fmt.Printf("❌ Erro ao salvar token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ Token OAuth2 salvo com sucesso!")
	fmt.Printf("   Local: %s\n", tokenPath)
	fmt.Printf("   Expira em: %s\n", token.Expiry.Format("2006-01-02 15:04:05"))
	fmt.Println("\nAgora você pode executar 'miau' normalmente.")
}
