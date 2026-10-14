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

type clusterItem struct {
	id          string
	displayName string
}

func (i clusterItem) FilterValue() string { return i.displayName }

type clusterItemDelegate struct{}

func (d clusterItemDelegate) Height() int                             { return 1 }
func (d clusterItemDelegate) Spacing() int                            { return 0 }
func (d clusterItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d clusterItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(clusterItem)
	if !ok {
		return
	}

	idStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(i.id)
	str := fmt.Sprintf("  %s (%s)", i.displayName, idStyled)

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

type ClusterModel struct {
	list         list.Model
	selectedID   string
	selectedName string
	quit         bool
	showAbout    bool
}

func NewClusterModel(clusters []api.ClusterInfo) ClusterModel {
	items := make([]list.Item, len(clusters))
	for i, c := range clusters {
		items[i] = clusterItem{id: c.ID, displayName: c.DisplayName}
	}

	l := list.New(items, clusterItemDelegate{}, 70, 20)
	l.Title = "🐮 Selecione o Cluster"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = listTitleStyle

	return ClusterModel{list: l}
}

func (m ClusterModel) Init() tea.Cmd {
	return nil
}

func (m ClusterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 2)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.quit = true
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(clusterItem); ok {
				m.selectedID = item.id
				m.selectedName = item.displayName
			}
			return m, tea.Quit
		case "?":
			m.showAbout = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m ClusterModel) View() string {
	return "\n" + m.list.View()
}

func (m ClusterModel) SelectedID() string {
	return m.selectedID
}

func (m ClusterModel) SelectedName() string {
	return m.selectedName
}

func (m ClusterModel) Quit() bool {
	return m.quit
}

func (m ClusterModel) ShowAbout() bool {
	return m.showAbout
}
