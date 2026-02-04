package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pickMode int

const (
	pickModeList pickMode = iota
	pickModeAction
)

type pickResult struct {
	Item   QueueItem
	Action string
}

type pickModel struct {
	allItems []QueueItem
	list     list.Model
	search   textinput.Model
	mode     pickMode
	query    string
	result   pickResult
	width    int
	height   int
}

type listItem struct {
	item QueueItem
}

func (i listItem) Title() string {
	return fmt.Sprintf("%s#%d %s", i.item.Repo, i.item.Number, i.item.Title)
}

func (i listItem) Description() string {
	return fmt.Sprintf("Author: %s  Age: %dd  Updated: %dd  Draft: %v  Checks: %s", i.item.Author, i.item.AgeDays, i.item.UpdatedDays, i.item.IsDraft, i.item.Checks)
}

func (i listItem) FilterValue() string {
	return strings.ToLower(fmt.Sprintf("%s %s", i.item.Repo, i.item.Title))
}

func newPickModel(items []QueueItem) pickModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	listModel := list.New([]list.Item{}, delegate, 0, 0)
	listModel.Title = "PRQ Picker"
	listModel.SetShowStatusBar(false)
	listModel.SetShowHelp(false)
	listModel.SetFilteringEnabled(false)

	search := textinput.New()
	search.Placeholder = "type to search"
	search.Prompt = "Search: "
	search.Focus()

	m := pickModel{
		allItems: items,
		list:     listModel,
		search:   search,
		mode:     pickModeList,
	}
	m.applyFilter()
	return m
}

func (m *pickModel) applyFilter() {
	query := strings.ToLower(strings.TrimSpace(m.search.Value()))
	filtered := make([]list.Item, 0, len(m.allItems))
	for _, item := range m.allItems {
		value := strings.ToLower(fmt.Sprintf("%s %s", item.Repo, item.Title))
		if query == "" || strings.Contains(value, query) {
			filtered = append(filtered, listItem{item: item})
		}
	}
	m.list.SetItems(filtered)
	if len(filtered) > 0 {
		m.list.Select(0)
	}
	m.query = m.search.Value()
}

func (m pickModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m pickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		listHeight := msg.Height - headerHeight - footerHeight - 2
		if listHeight < 4 {
			listHeight = 4
		}
		m.list.SetSize(msg.Width, listHeight)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
		if m.mode == pickModeAction {
			switch msg.String() {
			case "esc", "backspace":
				m.mode = pickModeList
				return m, nil
			case "enter":
				return m.chooseAction("review"), tea.Quit
			case "r":
				return m.chooseAction("review"), tea.Quit
			case "d":
				return m.chooseAction("draft"), tea.Quit
			case "s":
				return m.chooseAction("submit"), tea.Quit
			case "f":
				return m.chooseAction("followup"), tea.Quit
			case "o":
				return m.chooseAction("open"), tea.Quit
			}
		}
	}

	if m.mode == pickModeList {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		if m.search.Value() != m.query {
			m.applyFilter()
		}
		var listCmd tea.Cmd
		m.list, listCmd = m.list.Update(msg)
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "enter" {
			if len(m.list.Items()) == 0 {
				return m, nil
			}
			m.mode = pickModeAction
		}
		return m, tea.Batch(cmd, listCmd)
	}

	return m, nil
}

func (m pickModel) chooseAction(action string) pickModel {
	selected, ok := m.list.SelectedItem().(listItem)
	if !ok {
		return m
	}
	m.result = pickResult{Item: selected.item, Action: action}
	return m
}

func (m pickModel) View() string {
	header := m.headerView()
	footer := m.footerView()
	content := m.list.View()
	if len(m.list.Items()) == 0 {
		content = "No PRs match your search."
	}
	search := m.search.View()

	if m.mode == pickModeAction {
		return lipgloss.JoinVertical(lipgloss.Left, header, search, content, m.actionView(), footer)
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, search, content, footer)
}

func (m pickModel) headerView() string {
	return lipgloss.NewStyle().Bold(true).Render("PRQ Picker")
}

func (m pickModel) footerView() string {
	if m.mode == pickModeAction {
		return "Press ESC to go back, or choose an action."
	}
	return "Type to search • ↑/↓ to move • Enter for actions • q to quit"
}

func (m pickModel) actionView() string {
	style := lipgloss.NewStyle().Bold(true)
	return style.Render("Actions: [r]eview [d]raft [s]ubmit [f]ollowup [o]pen [esc] back")
}

func runPickTUI(items []QueueItem) (pickResult, error) {
	model := newPickModel(items)
	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return pickResult{}, err
	}
	finalPick, ok := finalModel.(pickModel)
	if !ok {
		return pickResult{}, fmt.Errorf("unexpected TUI model")
	}
	return finalPick.result, nil
}
