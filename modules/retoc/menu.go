package retoc

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/JaceTheGrayOne/TINKR-Toolkit/modules/config"
	"github.com/JaceTheGrayOne/TINKR-Toolkit/modules/ui"
)

// Workflow choice
type WorkflowOption struct {
	Name        string
	Description string
	Handler     func() tea.Model
}

type RetocMenuModel struct {
	workflows []WorkflowOption
	cursor    int
}

func NewRetocMenuModel() RetocMenuModel {
	workflows := []WorkflowOption{
		{
			Name:        "Pack Legacy Assets (Legacy → Zen)",
			Description: "Build mods from modified UAsset/UEXP files into Zen pak format",
			Handler: func() tea.Model {
				// If paths are already configured, go directly to pack builder
				if config.Current.ModsDir != "" && config.Current.PakDir != "" {
					mods, err := DiscoverMods()
					if err == nil && len(mods) > 0 {
						return NewPackBuilderModel(mods)
					}
					// If error, fall through to setup to reconfigure
				}
				return NewPackSetupModel()
			},
		},
		{
			Name:        "Unpack Game Files (Zen → Legacy)",
			Description: "Extract game assets from Zen pak format to Legacy format",
			Handler: func() tea.Model {
				return NewUnpackSetupModel()
			},
		},
	}

	return RetocMenuModel{
		workflows: workflows,
		cursor:    0,
	}
}

func (m RetocMenuModel) Init() tea.Cmd {
	return nil
}

// Message handler
func (m RetocMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			return m, tea.Quit

		case "backspace":
			return m, func() tea.Msg { return ui.BackMsg{} }

		case "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down":
			if m.cursor < len(m.workflows)-1 {
				m.cursor++
			}

		case "enter":
			// Switch to selected workflow
			selectedWorkflow := m.workflows[m.cursor]
			return selectedWorkflow.Handler(), nil

		case "1", "2", "3", "4", "5":
			// Hotkey selection
			idx := int(msg.String()[0] - '1')
			if idx < len(m.workflows) {
				selectedWorkflow := m.workflows[idx]
				return selectedWorkflow.Handler(), nil
			}
		}

	case ui.BackMsg:
		return m, func() tea.Msg { return ui.BackMsg{} }
	}

	return m, nil
}

// Render workflow selector
func (m RetocMenuModel) View() string {
	s := ui.TitleStyle.Render("Retoc - Zen Asset Packer/Unpacker") + "\n\n"
	s += ui.NormalStyle.Render("Select a workflow:") + "\n\n"

	for i, workflow := range m.workflows {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		hotkey := i + 1
		line := fmt.Sprintf("%s %d. %s", cursor, hotkey, workflow.Name)

		if m.cursor == i {
			s += ui.SelectedStyle.Render(line) + "\n"
			s += ui.InfoStyle.Render(fmt.Sprintf("     %s", workflow.Description)) + "\n"
		} else {
			s += ui.NormalStyle.Render(line) + "\n"
			s += ui.InfoStyle.Render(fmt.Sprintf("     %s", workflow.Description)) + "\n"
		}
		s += "\n"
	}

	s += "\n" + ui.InfoStyle.Render("↑/↓: Navigate • Enter: Select • Hotkeys: 1-9 • Backspace: Back • ESC: Quit")

	return s
}
