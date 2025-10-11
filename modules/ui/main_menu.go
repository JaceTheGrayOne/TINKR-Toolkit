package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// Tool
type Tool struct {
	Name        string
	Description string
	Model       tea.Model
}

// Tool selector
type MainMenuModel struct {
	tools  []Tool
	cursor int
}

// Back Navigation
type BackMsg struct{}

// Create main menu with available tools
func NewMainMenuModel(tools []Tool) MainMenuModel {
	return MainMenuModel{
		tools:  tools,
		cursor: 0,
	}
}

func (m MainMenuModel) Init() tea.Cmd {
	return nil
}

// Message handler
func (m MainMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down":
			if m.cursor < len(m.tools)-1 {
				m.cursor++
			}

		case "enter":
			selectedTool := m.tools[m.cursor]
			return selectedTool.Model, selectedTool.Model.Init()

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			// Hotkey selection
			idx := int(msg.String()[0] - '1')
			if idx < len(m.tools) {
				selectedTool := m.tools[idx]
				return selectedTool.Model, selectedTool.Model.Init()
			}
		}

	case BackMsg:
		return m, tea.Quit
	}

	return m, nil
}

// Render main menu
func (m MainMenuModel) View() string {
	s := TitleStyle.Render("TINK.R Toolkit") + "\n\n"
	s += NormalStyle.Render("Select a tool:") + "\n\n"

	for i, tool := range m.tools {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		hotkey := i + 1
		line := fmt.Sprintf("%s %d. %s", cursor, hotkey, tool.Name)

		if m.cursor == i {
			s += SelectedStyle.Render(line) + "\n"
			s += InfoStyle.Render(fmt.Sprintf("     %s", tool.Description)) + "\n"
		} else {
			s += NormalStyle.Render(line) + "\n"
			s += InfoStyle.Render(fmt.Sprintf("     %s", tool.Description)) + "\n"
		}
		s += "\n"
	}

	s += "\n" + InfoStyle.Render("↑/↓: Navigate • Enter: Select • Hotkeys: 1-9 • ESC: Quit")

	return s
}
