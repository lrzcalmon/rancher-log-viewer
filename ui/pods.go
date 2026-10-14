package ui

import (
	"fmt"
	"io"
	"strings"

	"rancher-logs/api"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type podItem struct {
	name   string
	status string
}

func (i podItem) FilterValue() string { return i.name }

type podItemDelegate struct{}

func (d podItemDelegate) Height() int                             { return 1 }
func (d podItemDelegate) Spacing() int                            { return 0 }
func (d podItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d podItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(podItem)
	if !ok {
		return
	}

	statusColor := "82"
	switch i.status {
	case "Running":
		statusColor = "82"
	case "Pending":
		statusColor = "220"
	case "Failed", "CrashLoopBackOff":
		statusColor = "196"
	default:
		statusColor = "245"
	}

	statusStyled := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(i.status)
	str := fmt.Sprintf("  %s [%s]", i.name, statusStyled)

	fn := lipgloss.NewStyle().PaddingLeft(2).Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(lipgloss.Color("170")).
				Render("▸ " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type PodModel struct {
	list     list.Model
	selected string
	quit     bool
}

func NewPodModel(pods []api.PodInfo, namespace, clusterName string) PodModel {
	items := make([]list.Item, len(pods))
	for i, p := range pods {
		items[i] = podItem{name: p.Name, status: p.Status}
	}

	l := list.New(items, podItemDelegate{}, 80, 20)
	l.Title = fmt.Sprintf("🚀 %s > %s — Pods", clusterName, namespace)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = listTitleStyle

	return PodModel{list: l}
}

func (m PodModel) Init() tea.Cmd {
	return nil
}

func (m PodModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quit = true
			return m, tea.Quit
		case "esc":
			m.selected = ""
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(podItem); ok {
				m.selected = item.name
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m PodModel) View() string {
	return "\n" + m.list.View()
}

func (m PodModel) Selected() string {
	return m.selected
}

func (m PodModel) Quit() bool {
	return m.quit
}
