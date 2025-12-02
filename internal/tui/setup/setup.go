package setup

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opik/miau/internal/config"
)

type step int

const (
	stepWelcome step = iota
	stepEmail
	stepImapHost
	stepImapPort
	stepPassword
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

	focusedButton = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#FF6B6B")).
			Padding(0, 2)

	blurredButton = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Padding(0, 2)
)

type Model struct {
	width    int
	height   int
	step     step
	inputs   []textinput.Model
	focused  int
	err      error
	account  config.Account
	complete bool
}

func New() Model {
	var inputs = make([]textinput.Model, 4)

	// Email
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "seu@email.com"
	inputs[0].CharLimit = 100
	inputs[0].Width = 40

	// IMAP Host
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "imap.gmail.com"
	inputs[1].CharLimit = 100
	inputs[1].Width = 40

	// IMAP Port
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "993"
	inputs[2].CharLimit = 5
	inputs[2].Width = 10

	// Password
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "senha ou app password"
	inputs[3].CharLimit = 100
	inputs[3].Width = 40
	inputs[3].EchoMode = textinput.EchoPassword
	inputs[3].EchoCharacter = '•'

	return Model{
		step:   stepWelcome,
		inputs: inputs,
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
				m.step--
				m.updateFocus()
			}
			return m, nil

		case "tab", "down":
			if m.step >= stepEmail && m.step <= stepPassword {
				m.focused++
				if m.focused > len(m.inputs)-1 {
					m.focused = 0
				}
				m.updateFocus()
			}
			return m, nil

		case "shift+tab", "up":
			if m.step >= stepEmail && m.step <= stepPassword {
				m.focused--
				if m.focused < 0 {
					m.focused = len(m.inputs) - 1
				}
				m.updateFocus()
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Atualiza o input focado
	if m.step >= stepEmail && m.step <= stepPassword {
		var cmd tea.Cmd
		var idx = m.stepToInputIndex()
		if idx >= 0 && idx < len(m.inputs) {
			m.inputs[idx], cmd = m.inputs[idx].Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m *Model) stepToInputIndex() int {
	switch m.step {
	case stepEmail:
		return 0
	case stepImapHost:
		return 1
	case stepImapPort:
		return 2
	case stepPassword:
		return 3
	}
	return -1
}

func (m *Model) updateFocus() {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	var idx = m.stepToInputIndex()
	if idx >= 0 && idx < len(m.inputs) {
		m.inputs[idx].Focus()
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

		m.step = stepImapHost
		m.inputs[0].Blur()
		m.inputs[1].Focus()
		return m, textinput.Blink

	case stepImapHost:
		var host = strings.TrimSpace(m.inputs[1].Value())
		if host == "" {
			m.err = fmt.Errorf("host IMAP obrigatório")
			return m, nil
		}
		m.err = nil
		m.account.Imap.Host = host
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
		m.account.Imap.Port = port
		m.account.Imap.TLS = (port == 993)
		m.step = stepPassword
		m.inputs[2].Blur()
		m.inputs[3].Focus()
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
		content = m.viewInput("Email", "Digite seu email:", 0)
	case stepImapHost:
		content = m.viewInput("Servidor IMAP", "Host do servidor IMAP:", 1)
	case stepImapPort:
		content = m.viewInput("Porta IMAP", "Porta do servidor (993 para TLS):", 2)
	case stepPassword:
		content = m.viewInput("Senha", "Senha ou App Password:", 3)
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

func (m Model) viewInput(title, prompt string, inputIdx int) string {
	var header = titleStyle.Render("miau 🐱 Setup")
	var stepInfo = subtitleStyle.Render(fmt.Sprintf("Passo %d de 4", inputIdx+1))
	var promptText = "\n" + prompt + "\n\n"
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
	var info = fmt.Sprintf(`
Email:  %s
Host:   %s
Porta:  %d
TLS:    %v
`,
		m.account.Email,
		m.account.Imap.Host,
		m.account.Imap.Port,
		m.account.Imap.TLS,
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

	return "imap." + domain
}
