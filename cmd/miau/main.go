package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/opik/miau/internal/config"
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
}

func initialModel() model {
	var m = model{}

	// Verifica se já existe configuração
	if config.ConfigExists() {
		var cfg, err = config.Load()
		if err == nil && cfg != nil && len(cfg.Accounts) > 0 {
			m.state = stateInbox
			m.cfg = cfg
			m.inboxModel = inbox.New(&cfg.Accounts[0])
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
			m.inboxModel = inbox.New(&cfg.Accounts[0])
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
	var p = tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Erro ao iniciar miau: %v\n", err)
		os.Exit(1)
	}
}
