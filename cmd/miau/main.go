package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opik/miau/internal/config"
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
	stateMain
)

type model struct {
	width      int
	height     int
	state      appState
	setupModel setup.Model
	cfg        *config.Config
}

func initialModel() model {
	var m = model{}

	// Verifica se já existe configuração
	if config.ConfigExists() {
		var cfg, err = config.Load()
		if err == nil && cfg != nil && len(cfg.Accounts) > 0 {
			m.state = stateMain
			m.cfg = cfg
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
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == stateMain {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			}
		}
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
			var cfg, _ = config.Load()
			m.cfg = cfg
			m.state = stateMain
			return m, nil
		}

		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	if m.state == stateSetup {
		return m.setupModel.View()
	}

	return m.viewMain()
}

func (m model) viewMain() string {
	var title = titleStyle.Render("miau 🐱")
	var subtitle = subtitleStyle.Render("Mail Intelligence Assistant Utility")

	var accountInfo string
	if m.cfg != nil && len(m.cfg.Accounts) > 0 {
		accountInfo = fmt.Sprintf("\n\nConta: %s", m.cfg.Accounts[0].Email)
	}

	var hint = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("\n\nPressione 'q' para sair")

	var content = fmt.Sprintf("%s\n%s%s%s",
		title,
		subtitle,
		accountInfo,
		hint,
	)

	var box = boxStyle.Render(content)

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}

	return box
}

func main() {
	var p = tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Erro ao iniciar miau: %v\n", err)
		os.Exit(1)
	}
}
