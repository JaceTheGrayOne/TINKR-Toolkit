package retoc

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/JaceTheGrayOne/TINKR-Toolkit/modules/ui"
)

type PackBuilderModel struct {
	mods         []Mod
	cursor       int
	selected     map[int]bool
	building     bool
	buildStart   time.Time
	log          string
	err          error
	ctx          context.Context
	cancel       context.CancelFunc
	startTime    time.Time
	currentTask  string
	parallelMode bool
}

type BackMsg struct{}

func NewPackBuilderModel(mods []Mod) PackBuilderModel {
	return PackBuilderModel{
		mods:     mods,
		cursor:   0,
		selected: make(map[int]bool),
	}
}

func (m PackBuilderModel) Init() tea.Cmd {
	fmt.Fprintf(os.Stderr, "DEBUG: PackBuilderModel.Init() called with %d mods\n", len(m.mods))
	return nil
}

func (m PackBuilderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.building {
			switch msg.String() {
			case "ctrl+c", "esc":
				if m.cancel != nil {
					m.cancel()
				}
				return m, tea.Quit

			case "backspace":
				if m.cancel != nil {
					m.cancel()
				}
				return m, func() tea.Msg { return BackMsg{} }
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "backspace":
			return m, func() tea.Msg { return BackMsg{} }

		case "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down":
			if m.cursor < len(m.mods) {
				m.cursor++
			}

		case " ":
			if m.cursor > 0 {
				modIndex := m.cursor - 1
				m.selected[modIndex] = !m.selected[modIndex]
			}

		case "0":
			m.cursor = 0
			m.building = true
			m.buildStart = time.Now()
			m.log = ""
			m.err = nil
			m.startTime = time.Now()
			m.ctx, m.cancel = context.WithCancel(context.Background())
			m.currentTask = "Build ALL"
			m.parallelMode = false
			return m, BuildAllAsync(m.ctx, m.mods)

		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			modIndex := int(msg.String()[0] - '1')
			if modIndex < len(m.mods) {
				m.cursor = modIndex + 1
				m.building = true
				m.buildStart = time.Now()
				m.log = ""
				m.err = nil
				m.startTime = time.Now()
				m.ctx, m.cancel = context.WithCancel(context.Background())
				m.parallelMode = false
				selectedMod := m.mods[modIndex]
				m.currentTask = selectedMod.DisplayName
				return m, BuildOneAsync(m.ctx, selectedMod)
			}

		case "enter":
			m.building = true
			m.buildStart = time.Now()
			m.log = ""
			m.err = nil
			m.startTime = time.Now()
			m.ctx, m.cancel = context.WithCancel(context.Background())

			if m.cursor == 0 {
				m.currentTask = "Build ALL"
				m.parallelMode = false
				return m, BuildAllAsync(m.ctx, m.mods)
			} else if len(m.selected) > 1 {
				m.currentTask = fmt.Sprintf("%d Mods Selected", len(m.selected))
				m.parallelMode = true
				var selectedMods []Mod
				for i := range m.mods {
					if m.selected[i] {
						selectedMods = append(selectedMods, m.mods[i])
					}
				}
				return m, BuildSelectedParallelAsync(m.ctx, selectedMods)
			} else if len(m.selected) == 1 {
				m.parallelMode = false
				for i := range m.mods {
					if m.selected[i] {
						m.currentTask = m.mods[i].DisplayName
						return m, BuildOneAsync(m.ctx, m.mods[i])
					}
				}
			} else {
				m.parallelMode = false
				selectedMod := m.mods[m.cursor-1]
				m.currentTask = selectedMod.DisplayName
				return m, BuildOneAsync(m.ctx, selectedMod)
			}
		}

	case BuildCompleteMsg:
		m.building = false
		m.log = msg.Log
		m.err = msg.Err
		m.parallelMode = false
		if msg.Err == nil {
			m.selected = make(map[int]bool)
		}
		return m, nil
	}

	return m, nil
}

// Renders the UI
func (m PackBuilderModel) View() string {
	fmt.Fprintf(os.Stderr, "DEBUG: PackBuilderModel.View() called\n")

	if m.building {
		elapsed := time.Since(m.buildStart)
		if elapsed > 500*time.Millisecond {
			return m.buildingView()
		}
		return m.menuView()
	}
	return m.menuView()
}

func (m PackBuilderModel) menuView() string {
	s := ui.TitleStyle.Render("TINK.R Toolkit - Pak Builder") + "\n\n"

	cursor := " "
	if m.cursor == 0 {
		cursor = ">"
		s += ui.SelectedStyle.Render(fmt.Sprintf("%s [ - ] - 0. Build ALL", cursor)) + "\n"
	} else {
		s += ui.NormalStyle.Render(fmt.Sprintf("%s [ - ] - 0. Build ALL", cursor)) + "\n"
	}

	for i, mod := range m.mods {
		cursor = " "
		var checkbox string

		if m.selected[i] {
			checkbox = " X "
		} else {
			checkbox = "   "
		}

		hotkey := i + 1
		modName := mod.DisplayName

		if m.cursor == i+1 {
			cursor = ">"
			if m.selected[i] {
				s += ui.SelectedStyle.Render(fmt.Sprintf("%s [%s] - %d. %s", cursor, checkbox, hotkey, modName)) + "\n"
			} else {
				s += ui.SelectedStyle.Render(fmt.Sprintf("%s [%s] - %d. %s", cursor, checkbox, hotkey, modName)) + "\n"
			}
		} else {
			if m.selected[i] {
				s += ui.NormalStyle.Render(fmt.Sprintf("%s [", cursor)) + ui.CheckboxStyle.Render(checkbox) + ui.NormalStyle.Render(fmt.Sprintf("] - %d. %s", hotkey, modName)) + "\n"
			} else {
				s += ui.NormalStyle.Render(fmt.Sprintf("%s [%s] - %d. %s", cursor, checkbox, hotkey, modName)) + "\n"
			}
		}
	}

	s += "\n"

	if len(m.selected) > 0 {
		s += ui.InfoStyle.Render(fmt.Sprintf("%d mod(s) selected", len(m.selected))) + "\n"
	} else {
		s += "\n"
	}

	s += "\n"

	if m.err != nil {
		s += ui.ErrorStyle.Render("Error: ") + m.err.Error() + "\n"
	} else if m.log != "" {
		lines := strings.Split(strings.TrimSpace(m.log), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "✓") {
				s += ui.SuccessStyle.Render("✓") + " " + ui.NormalStyle.Render(strings.TrimPrefix(line, "✓ ")) + "\n"
			} else if strings.HasPrefix(line, "✗") {
				s += ui.ErrorStyle.Render("✗") + " " + ui.NormalStyle.Render(strings.TrimPrefix(line, "✗ ")) + "\n"
			}
		}
	} else {
		s += "\n"
	}

	s += "\nSpace to select • Enter to build • Hotkeys: 0-9 • ESC: Quit"

	return s
}

func (m PackBuilderModel) buildingView() string {
	elapsed := time.Since(m.startTime).Round(time.Second)

	s := ui.BuildingStyle.Render(fmt.Sprintf("⚙️  Building: %s", m.currentTask)) + "\n"
	s += fmt.Sprintf("Elapsed: %s\n\n", elapsed)

	if m.parallelMode {
		s += ui.InfoStyle.Render("Building mods...") + "\n"
	} else {
		if m.log != "" {
			lines := strings.Split(m.log, "\n")
			start := 0
			if len(lines) > 20 {
				start = len(lines) - 20
			}
			s += strings.Join(lines[start:], "\n")
		}
	}

	s += "\n\nPress Esc to cancel"

	return s
}
