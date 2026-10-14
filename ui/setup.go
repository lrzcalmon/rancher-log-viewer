package ui

import (
	"fmt"
	"strings"

	"rancher-logs/config"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	focusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	titleStyle   = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginBottom(1)
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

type SetupModel struct {
	inputs     []textinput.Model
	focusIndex int
	done       bool
	Config     *config.Config
}

func NewSetupModel() SetupModel {
	inputs := make([]textinput.Model, 4)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "https://rancher.empresa.com.br"
	inputs[0].Focus()
	inputs[0].PromptStyle = focusedStyle
	inputs[0].TextStyle = focusedStyle
	inputs[0].CharLimit = 256

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "seu-usuario"
	inputs[1].CharLimit = 128

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "sua-senha"
	inputs[2].CharLimit = 256
	inputs[2].EchoMode = textinput.EchoPassword

	inputs[3] = textinput.New()
	inputs[3].Placeholder = "s/n (default: s)"
	inputs[3].CharLimit = 3

	return SetupModel{
		inputs: inputs,
	}
}

func (m SetupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			if s == "enter" && m.focusIndex == len(m.inputs) {
				m.done = true
				insecure := strings.ToLower(strings.TrimSpace(m.inputs[3].Value()))
				skipVerify := insecure != "n"

				m.Config = &config.Config{
					RancherURL:         strings.TrimSpace(m.inputs[0].Value()),
					Username:           strings.TrimSpace(m.inputs[1].Value()),
					Password:           strings.TrimSpace(m.inputs[2].Value()),
					InsecureSkipVerify: skipVerify,
				}
				return m, tea.Quit
			}

			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := range m.inputs {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = focusedStyle
					m.inputs[i].TextStyle = focusedStyle
				} else {
					m.inputs[i].Blur()
					m.inputs[i].PromptStyle = blurredStyle
					m.inputs[i].TextStyle = blurredStyle
				}
			}
			return m, tea.Batch(cmds...)
		}
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *SetupModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m SetupModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🐮 Rancher Log Viewer — Configuração Inicial"))
	b.WriteString("\n\n")

	labels := []string{
		"Rancher URL:",
		"Username:",
		"Password:",
		"Ignorar SSL (s/n):",
	}

	for i, input := range m.inputs {
		b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
		b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
	}

	button := blurredStyle.Render("[ Salvar e Continuar ]")
	if m.focusIndex == len(m.inputs) {
		button = focusedStyle.Render("[ Salvar e Continuar ]")
	}
	b.WriteString(fmt.Sprintf("  %s\n\n", button))
	b.WriteString(helpStyle.Render("  Tab/Enter: próximo • Esc: sair"))

	return b.String()
}

func (m SetupModel) Done() bool {
	return m.done
}
