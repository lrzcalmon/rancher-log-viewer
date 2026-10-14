// Rancher Log Viewer
// Autor: Luiz Ricardo Z. Calmon

package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	AppName    = "Rancher Log Viewer"
	AppVersion = "1.0.0"
	AppAuthor  = "Luiz Ricardo Z. Calmon"
	AppLang    = "Go (Golang)"
	AppDesc    = "Visualização interativa de logs de pods Kubernetes via Rancher API"
)

var (
	aboutTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginBottom(1).
			PaddingLeft(2)
	aboutLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			PaddingLeft(4)
	aboutValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
	aboutFooterStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				PaddingLeft(4).
				MarginTop(1)
)

type AboutModel struct {
	quit bool
}

func NewAboutModel() AboutModel {
	return AboutModel{}
}

func (m AboutModel) Init() tea.Cmd {
	return nil
}

func (m AboutModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q", "enter":
			m.quit = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m AboutModel) View() string {
	var b strings.Builder

	b.WriteString("\n\n")
	b.WriteString(aboutTitleStyle.Render("🐮 " + AppName))
	b.WriteString("\n\n")

	lines := []struct {
		label string
		value string
	}{
		{"Nome", AppName},
		{"Versão", AppVersion},
		{"Autor", AppAuthor},
		{"Linguagem", AppLang},
		{"Plataforma", "Windows 64 bits"},
		{"Finalidade", AppDesc},
		{"Interface", "Terminal UI (Bubble Tea + Lip Gloss)"},
		{"Autenticação", "Basic Auth (Rancher API Key)"},
		{"Portabilidade", "Executável único, sem dependências externas"},
	}

	for _, l := range lines {
		b.WriteString(fmt.Sprintf("%s  %s\n",
			aboutLabelStyle.Render(l.label+":"),
			aboutValueStyle.Render(l.value),
		))
	}

	b.WriteString("\n")
	b.WriteString(aboutFooterStyle.Render("Pressione Enter, Esc ou Q para voltar"))

	return b.String()
}

func (m AboutModel) Quit() bool {
	return m.quit
}
