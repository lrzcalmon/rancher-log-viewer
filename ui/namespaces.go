package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	listTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			PaddingLeft(2)
	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			PaddingLeft(2)
)

type nsItem string

func (i nsItem) FilterValue() string { return string(i) }

type nsItemDelegate struct{}

func (d nsItemDelegate) Height() int                             { return 1 }
func (d nsItemDelegate) Spacing() int                            { return 0 }
func (d nsItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d nsItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(nsItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("  %s", string(i))

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

type NamespaceModel struct {
	list     list.Model
	selected string
	quit     bool
}

func NewNamespaceModel(namespaces []string) NamespaceModel {
	items := make([]list.Item, len(namespaces))
	for i, ns := range namespaces {
		items[i] = nsItem(ns)
	}

	l := list.New(items, nsItemDelegate{}, 60, 20)
	l.Title = "📦 Selecione o Namespace"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = listTitleStyle

	return NamespaceModel{list: l}
}

func (m NamespaceModel) Init() tea.Cmd {
	return nil
}

func (m NamespaceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if item, ok := m.list.SelectedItem().(nsItem); ok {
				m.selected = string(item)
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m NamespaceModel) View() string {
	return "\n" + m.list.View()
}

func (m NamespaceModel) Selected() string {
	return m.selected
}

func (m NamespaceModel) Quit() bool {
	return m.quit
}
