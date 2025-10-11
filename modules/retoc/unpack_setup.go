package retoc

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/JaceTheGrayOne/TINKR-Toolkit/modules/ui"
)

type UnpackSetupModel struct {
	// TODO: Add fields for unpack setup
}

func NewUnpackSetupModel() UnpackSetupModel {
	return UnpackSetupModel{}
}

func (m UnpackSetupModel) Init() tea.Cmd {
	return nil
}

func (m UnpackSetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			return m, tea.Quit

		case "backspace":
			return m, func() tea.Msg { return ui.BackMsg{} }
		}
	}

	return m, nil
}

func (m UnpackSetupModel) View() string {
	s := ui.TitleStyle.Render("Unpack Setup") + "\n\n"
	s += ui.InfoStyle.Render("Unpack functionality coming soon...") + "\n\n"
	s += ui.InfoStyle.Render("Backspace: Go back • ESC: Quit")
	return s
}
