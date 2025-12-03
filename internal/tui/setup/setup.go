package setup

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opik/miau/internal/auth"
	"github.com/opik/miau/internal/config"
)

type step int

const (
	stepWelcome step = iota
	stepEmail
	stepAuthType
	stepImapHost
	stepImapPort
	stepPassword      // Para auth_type = password
	stepOAuth2Client  // Para auth_type = oauth2
	stepOAuth2Secret  // Para auth_type = oauth2
	stepOAuth2Auth    // Executar autenticação OAuth2
	stepConfirm
	stepDone
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B6B")).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#73D216")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF6B6B")).
			Padding(1, 2)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true)

	unselectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ECDC4"))
)

type Model struct {
	width        int
	height       int
	step         step
	inputs       []textinput.Model
	err          error
	account      config.Account
	complete     bool
	authChoice   int  // 0 = OAuth2, 1 = Password
	authenticating bool
	authDone     bool
}

func New() Model {
	var inputs = make([]textinput.Model, 6)

	// 0: Email
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "seu@email.com"
	inputs[0].CharLimit = 100
	inputs[0].Width = 40

	// 1: IMAP Host
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "imap.gmail.com"
	inputs[1].CharLimit = 100
	inputs[1].Width = 40

	// 2: IMAP Port
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "993"
	inputs[2].CharLimit = 5
	inputs[2].Width = 10

	// 3: Password
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "senha ou app password"
	inputs[3].CharLimit = 100
	inputs[3].Width = 40
	inputs[3].EchoMode = textinput.EchoPassword
	inputs[3].EchoCharacter = '•'

	// 4: OAuth2 Client ID
	inputs[4] = textinput.New()
	inputs[4].Placeholder = "xxxxx.apps.googleusercontent.com"
	inputs[4].CharLimit = 200
	inputs[4].Width = 50

	// 5: OAuth2 Client Secret
	inputs[5] = textinput.New()
	inputs[5].Placeholder = "GOCSPX-xxxxx"
	inputs[5].CharLimit = 100
	inputs[5].Width = 40
	inputs[5].EchoMode = textinput.EchoPassword
	inputs[5].EchoCharacter = '•'

	return Model{
		step:       stepWelcome,
		inputs:     inputs,
		authChoice: 0, // OAuth2 como padrão
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "enter":
			return m.handleEnter()

		case "esc":
			if m.step > stepWelcome && m.step < stepDone {
				m.step = m.previousStep()
				m.updateFocus()
			}
			return m, nil

		case "up", "k":
			if m.step == stepAuthType {
				if m.authChoice > 0 {
					m.authChoice--
				}
			}
			return m, nil

		case "down", "j":
			if m.step == stepAuthType {
				if m.authChoice < 1 {
					m.authChoice++
				}
			}
			return m, nil

		case "tab":
			if m.step == stepAuthType {
				m.authChoice = (m.authChoice + 1) % 2
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case authResultMsg:
		m.authenticating = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.authDone = true
		m.step = stepConfirm
		return m, nil
	}

	// Atualiza o input focado
	var idx = m.currentInputIndex()
	if idx >= 0 && idx < len(m.inputs) {
		var cmd tea.Cmd
		m.inputs[idx], cmd = m.inputs[idx].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) previousStep() step {
	switch m.step {
	case stepEmail:
		return stepWelcome
	case stepAuthType:
		return stepEmail
	case stepImapHost:
		return stepAuthType
	case stepImapPort:
		return stepImapHost
	case stepPassword:
		return stepImapPort
	case stepOAuth2Client:
		return stepImapPort
	case stepOAuth2Secret:
		return stepOAuth2Client
	case stepOAuth2Auth:
		return stepOAuth2Secret
	case stepConfirm:
		if m.account.AuthType == config.AuthTypeOAuth2 {
			return stepOAuth2Auth
		}
		return stepPassword
	}
	return stepWelcome
}

func (m *Model) currentInputIndex() int {
	switch m.step {
	case stepEmail:
		return 0
	case stepImapHost:
		return 1
	case stepImapPort:
		return 2
	case stepPassword:
		return 3
	case stepOAuth2Client:
		return 4
	case stepOAuth2Secret:
		return 5
	}
	return -1
}

func (m *Model) updateFocus() {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	var idx = m.currentInputIndex()
	if idx >= 0 && idx < len(m.inputs) {
		m.inputs[idx].Focus()
	}
}

type authResultMsg struct {
	err error
}

func (m Model) doOAuth2Auth() tea.Cmd {
	return func() tea.Msg {
		var oauthCfg = auth.GetOAuth2Config(
			m.account.OAuth2.ClientID,
			m.account.OAuth2.ClientSecret,
		)

		var token, err = auth.AuthenticateWithBrowser(oauthCfg)
		if err != nil {
			return authResultMsg{err: err}
		}

		// Salva token
		var tokenPath = auth.GetTokenPath(config.GetConfigPath(), m.account.Name)
		if err := auth.SaveToken(tokenPath, token); err != nil {
			return authResultMsg{err: fmt.Errorf("erro ao salvar token: %w", err)}
		}

		return authResultMsg{err: nil}
	}
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case stepWelcome:
		m.step = stepEmail
		m.inputs[0].Focus()
		return m, textinput.Blink

	case stepEmail:
		var email = strings.TrimSpace(m.inputs[0].Value())
		if email == "" || !strings.Contains(email, "@") {
			m.err = fmt.Errorf("email inválido")
			return m, nil
		}
		m.err = nil
		m.account.Email = email
		m.account.Name = strings.Split(email, "@")[0]

		// Auto-detectar IMAP host
		var domain = strings.Split(email, "@")[1]
		m.inputs[1].SetValue(guessImapHost(domain))
		m.inputs[2].SetValue("993")

		// Se for Gmail/Google, sugerir OAuth2
		if isGoogleDomain(domain) {
			m.authChoice = 0 // OAuth2
		}

		m.step = stepAuthType
		m.inputs[0].Blur()
		return m, nil

	case stepAuthType:
		if m.authChoice == 0 {
			m.account.AuthType = config.AuthTypeOAuth2
			m.account.OAuth2 = &config.OAuth2Config{}
		} else {
			m.account.AuthType = config.AuthTypePassword
		}
		m.step = stepImapHost
		m.inputs[1].Focus()
		return m, textinput.Blink

	case stepImapHost:
		var host = strings.TrimSpace(m.inputs[1].Value())
		if host == "" {
			m.err = fmt.Errorf("host IMAP obrigatório")
			return m, nil
		}
		m.err = nil
		m.account.IMAP.Host = host
		m.step = stepImapPort
		m.inputs[1].Blur()
		m.inputs[2].Focus()
		return m, textinput.Blink

	case stepImapPort:
		var portStr = strings.TrimSpace(m.inputs[2].Value())
		if portStr == "" {
			portStr = "993"
		}
		var port, err = strconv.Atoi(portStr)
		if err != nil || port < 1 || port > 65535 {
			m.err = fmt.Errorf("porta inválida")
			return m, nil
		}
		m.err = nil
		m.account.IMAP.Port = port
		m.account.IMAP.TLS = (port == 993)
		m.inputs[2].Blur()

		if m.account.AuthType == config.AuthTypeOAuth2 {
			m.step = stepOAuth2Client
			m.inputs[4].Focus()
		} else {
			m.step = stepPassword
			m.inputs[3].Focus()
		}
		return m, textinput.Blink

	case stepPassword:
		var password = m.inputs[3].Value()
		if password == "" {
			m.err = fmt.Errorf("senha obrigatória")
			return m, nil
		}
		m.err = nil
		m.account.Password = password
		m.step = stepConfirm
		m.inputs[3].Blur()
		return m, nil

	case stepOAuth2Client:
		var clientID = strings.TrimSpace(m.inputs[4].Value())
		if clientID == "" {
			m.err = fmt.Errorf("Client ID obrigatório")
			return m, nil
		}
		m.err = nil
		m.account.OAuth2.ClientID = clientID
		m.step = stepOAuth2Secret
		m.inputs[4].Blur()
		m.inputs[5].Focus()
		return m, textinput.Blink

	case stepOAuth2Secret:
		var clientSecret = strings.TrimSpace(m.inputs[5].Value())
		if clientSecret == "" {
			m.err = fmt.Errorf("Client Secret obrigatório")
			return m, nil
		}
		m.err = nil
		m.account.OAuth2.ClientSecret = clientSecret
		m.step = stepOAuth2Auth
		m.inputs[5].Blur()
		return m, nil

	case stepOAuth2Auth:
		if m.authenticating {
			return m, nil
		}
		m.authenticating = true
		m.err = nil
		return m, m.doOAuth2Auth()

	case stepConfirm:
		// Salvar configuração
		var cfg = config.DefaultConfig()
		cfg.Accounts = append(cfg.Accounts, m.account)
		if err := config.Save(cfg); err != nil {
			m.err = fmt.Errorf("erro ao salvar: %v", err)
			return m, nil
		}
		m.complete = true
		m.step = stepDone
		return m, nil

	case stepDone:
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) View() string {
	var content string

	switch m.step {
	case stepWelcome:
		content = m.viewWelcome()
	case stepEmail:
		content = m.viewInput("Email", "Digite seu email:", 0, "1/6")
	case stepAuthType:
		content = m.viewAuthType()
	case stepImapHost:
		content = m.viewInput("Servidor IMAP", "Host do servidor IMAP:", 1, "3/6")
	case stepImapPort:
		content = m.viewInput("Porta IMAP", "Porta do servidor (993 para TLS):", 2, "4/6")
	case stepPassword:
		content = m.viewInput("Senha", "Senha ou App Password:", 3, "5/6")
	case stepOAuth2Client:
		content = m.viewOAuth2Client()
	case stepOAuth2Secret:
		content = m.viewInput("Client Secret", "OAuth2 Client Secret:", 5, "6/7")
	case stepOAuth2Auth:
		content = m.viewOAuth2Auth()
	case stepConfirm:
		content = m.viewConfirm()
	case stepDone:
		content = m.viewDone()
	}

	var box = boxStyle.Render(content)

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}

	return box
}

func (m Model) viewWelcome() string {
	var title = titleStyle.Render("miau 🐱")
	var subtitle = subtitleStyle.Render("Mail Intelligence Assistant Utility")
	var desc = "\n\nBem-vindo ao miau!\nVamos configurar sua primeira conta de email.\n"
	var hint = subtitleStyle.Render("\nPressione Enter para começar")

	return fmt.Sprintf("%s\n%s%s%s", title, subtitle, desc, hint)
}

func (m Model) viewAuthType() string {
	var header = titleStyle.Render("miau 🐱 Setup")
	var stepInfo = subtitleStyle.Render("Passo 2 de 6")
	var prompt = "\n\nTipo de autenticação:\n\n"

	var oauth2Label, passLabel string
	if m.authChoice == 0 {
		oauth2Label = selectedStyle.Render("● OAuth2 (Google)")
		passLabel = unselectedStyle.Render("○ Senha/App Password")
	} else {
		oauth2Label = unselectedStyle.Render("○ OAuth2 (Google)")
		passLabel = selectedStyle.Render("● Senha/App Password")
	}

	var options = fmt.Sprintf("  %s\n  %s\n", oauth2Label, passLabel)

	var info string
	if m.authChoice == 0 {
		info = infoStyle.Render("\n  OAuth2: Mais seguro, abre navegador para login.\n  Requer criar credenciais no Google Cloud Console.")
	} else {
		info = infoStyle.Render("\n  Senha: Use App Password se tiver 2FA ativado.\n  Vá em Google Account → Security → App Passwords")
	}

	var errText string
	if m.err != nil {
		errText = "\n" + errorStyle.Render(m.err.Error())
	}

	var hint = "\n\n" + subtitleStyle.Render("↑↓/Tab: selecionar • Enter: confirmar • Esc: voltar")

	return fmt.Sprintf("%s\n%s%s%s%s%s%s", header, stepInfo, prompt, options, info, errText, hint)
}

func (m Model) viewOAuth2Client() string {
	var header = titleStyle.Render("miau 🐱 Setup - OAuth2")
	var stepInfo = subtitleStyle.Render("Passo 5 de 7")

	var instructions = infoStyle.Render(`
Para obter as credenciais OAuth2:

1. Acesse: console.cloud.google.com
2. Crie um projeto (ou use existente)
3. APIs & Services → OAuth consent screen
   - Configure como "Internal" (para Workspace)
   - Ou "External" para conta pessoal
4. APIs & Services → Credentials
   - Create Credentials → OAuth client ID
   - Application type: Desktop app
5. Copie o Client ID abaixo:
`)

	var prompt = "\n\nClient ID:\n"
	var input = inputStyle.Render(m.inputs[4].View())

	var errText string
	if m.err != nil {
		errText = "\n" + errorStyle.Render(m.err.Error())
	}

	var hint = "\n\n" + subtitleStyle.Render("Enter: próximo • Esc: voltar")

	return fmt.Sprintf("%s\n%s%s%s%s%s%s", header, stepInfo, instructions, prompt, input, errText, hint)
}

func (m Model) viewOAuth2Auth() string {
	var header = titleStyle.Render("miau 🐱 Setup - Autenticação")

	var content string
	if m.authenticating {
		content = `
🔐 Autenticando...

O navegador deve abrir automaticamente.
Faça login na sua conta Google e autorize o acesso.

Aguardando...`
	} else if m.err != nil {
		content = fmt.Sprintf(`
❌ Erro na autenticação:

%s

Pressione Enter para tentar novamente.
Ou Esc para voltar e verificar as credenciais.`, m.err.Error())
	} else {
		content = `
Pronto para autenticar!

Pressione Enter para abrir o navegador
e fazer login na sua conta Google.`
	}

	var hint = "\n\n" + subtitleStyle.Render("Enter: autenticar • Esc: voltar")

	return fmt.Sprintf("%s%s%s", header, content, hint)
}

func (m Model) viewInput(title, prompt string, inputIdx int, stepStr string) string {
	var header = titleStyle.Render("miau 🐱 Setup")
	var stepInfo = subtitleStyle.Render("Passo " + stepStr)
	var promptText = "\n\n" + prompt + "\n\n"
	var input = inputStyle.Render(m.inputs[inputIdx].View())

	var errText string
	if m.err != nil {
		errText = "\n" + errorStyle.Render(m.err.Error())
	}

	var hint = "\n\n" + subtitleStyle.Render("Enter: próximo • Esc: voltar")

	return fmt.Sprintf("%s\n%s%s%s%s%s", header, stepInfo, promptText, input, errText, hint)
}

func (m Model) viewConfirm() string {
	var header = titleStyle.Render("miau 🐱 Confirmar")

	var authInfo string
	if m.account.AuthType == config.AuthTypeOAuth2 {
		authInfo = fmt.Sprintf("Auth:   OAuth2 ✓\nToken:  Salvo")
	} else {
		authInfo = fmt.Sprintf("Auth:   Senha")
	}

	var info = fmt.Sprintf(`
Email:  %s
Host:   %s
Porta:  %d
TLS:    %v
%s
`,
		m.account.Email,
		m.account.IMAP.Host,
		m.account.IMAP.Port,
		m.account.IMAP.TLS,
		authInfo,
	)

	var errText string
	if m.err != nil {
		errText = "\n" + errorStyle.Render(m.err.Error())
	}

	var hint = "\n" + subtitleStyle.Render("Enter: salvar • Esc: voltar")

	return fmt.Sprintf("%s%s%s%s", header, info, errText, hint)
}

func (m Model) viewDone() string {
	var header = successStyle.Render("✓ Configuração salva!")
	var info = fmt.Sprintf("\nArquivo: %s", config.GetConfigFile())
	if m.account.AuthType == config.AuthTypeOAuth2 {
		info += fmt.Sprintf("\nToken:   %s", auth.GetTokenPath(config.GetConfigPath(), m.account.Name))
	}
	var hint = "\n\n" + subtitleStyle.Render("Pressione Enter para continuar")

	return fmt.Sprintf("%s%s%s", header, info, hint)
}

func (m Model) IsComplete() bool {
	return m.complete
}

func guessImapHost(domain string) string {
	var knownHosts = map[string]string{
		"gmail.com":      "imap.gmail.com",
		"googlemail.com": "imap.gmail.com",
		"outlook.com":    "outlook.office365.com",
		"hotmail.com":    "outlook.office365.com",
		"live.com":       "outlook.office365.com",
		"yahoo.com":      "imap.mail.yahoo.com",
		"yahoo.com.br":   "imap.mail.yahoo.com",
		"icloud.com":     "imap.mail.me.com",
		"me.com":         "imap.mail.me.com",
		"uol.com.br":     "imap.uol.com.br",
		"bol.com.br":     "imap.bol.com.br",
		"terra.com.br":   "imap.terra.com.br",
	}

	if host, ok := knownHosts[domain]; ok {
		return host
	}

	// Por padrão, assume Google Workspace (muito comum para domínios corporativos)
	// O usuário pode mudar manualmente se não for Google
	return "imap.gmail.com"
}

func isGoogleDomain(domain string) bool {
	var googleDomains = map[string]bool{
		"gmail.com":      true,
		"googlemail.com": true,
	}

	if googleDomains[domain] {
		return true
	}

	// Assume que domínios customizados podem ser Google Workspace
	// Se o host IMAP for gmail, provavelmente é Google
	return false
}
