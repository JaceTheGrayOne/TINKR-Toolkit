package retoc

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/JaceTheGrayOne/TINKR-Toolkit/modules/config"
	"github.com/JaceTheGrayOne/TINKR-Toolkit/modules/ui"
)

type setupStep int

const (
	stepModsDir setupStep = iota
	stepPakDir
	stepComplete
)

type PackSetupModel struct {
	step      setupStep
	textInput textinput.Model
	modsDir   string
	pakDir    string
	err       error
}

type transitionToPackBuilderMsg struct{}

func NewPackSetupModel() PackSetupModel {
	ti := textinput.New()
	ti.Placeholder = "Enter directory path..."
	ti.Focus()
	ti.Width = 60

	step := stepModsDir
	if config.Current.ModsDir != "" {
		step = stepPakDir
	}

	return PackSetupModel{
		step:      step,
		textInput: ti,
		modsDir:   config.Current.ModsDir,
		pakDir:    config.Current.PakDir,
	}
}

func (m PackSetupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m PackSetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit

		case tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyBackspace:
			if m.textInput.Value() == "" {
				return m, func() tea.Msg { return ui.BackMsg{} }
			}

		case tea.KeyEnter:
			return m.handleEnter()
		}

	case ui.BackMsg:
		return m, func() tea.Msg { return ui.BackMsg{} }

	case transitionToPackBuilderMsg:
		// Discover mods and quit to transition to pack builder
		return m, tea.Quit
	}

	if m.step == stepModsDir || m.step == stepPakDir {
		m.textInput, cmd = m.textInput.Update(msg)
	}

	return m, cmd
}

func (m PackSetupModel) handleEnter() (tea.Model, tea.Cmd) {
	value := m.textInput.Value()

	switch m.step {
	case stepModsDir:
		normalized, err := config.NormalizePath(value)
		if err != nil {
			m.err = fmt.Errorf("invalid path: %w", err)
			return m, nil
		}

		if _, err := os.Stat(normalized); err != nil {
			m.err = fmt.Errorf("directory not found: %s", normalized)
			return m, nil
		}

		m.modsDir = normalized
		config.Current.ModsDir = normalized

		if err := config.SaveConfig(); err != nil {
			m.err = fmt.Errorf("failed to save config: %w", err)
			return m, nil
		}

		m.step = stepPakDir
		m.textInput.SetValue("")
		m.err = nil
		return m, nil

	case stepPakDir:
		normalized, err := config.NormalizePath(value)
		if err != nil {
			m.err = fmt.Errorf("invalid path: %w", err)
			return m, nil
		}

		m.pakDir = normalized
		config.Current.PakDir = normalized

		if err := config.SaveConfig(); err != nil {
			m.err = fmt.Errorf("failed to save config: %w", err)
			return m, nil
		}

		// Set to complete state to show success
		m.step = stepComplete
		m.err = nil

		// Transition to pack builder
		return m, func() tea.Msg {
			return transitionToPackBuilderMsg{}
		}
	}

	return m, nil
}

func (m PackSetupModel) View() string {
	s := ui.TitleStyle.Render("Pack Setup") + "\n\n"

	switch m.step {
	case stepModsDir:
		s += ui.NormalStyle.Render("Modified UAsset/UEXP Directory:") + "\n"
		s += ui.InfoStyle.Render("  Where your modified game files are located") + "\n"
		s += ui.InfoStyle.Render("  Example: G:\\Grounded\\Modding\\Grounded2\\Mods") + "\n\n"
		s += m.textInput.View() + "\n\n"

	case stepPakDir:
		s += ui.SuccessStyle.Render("✓ Mods directory: "+m.modsDir) + "\n\n"
		s += ui.NormalStyle.Render("UE Game \"Paks\" Directory:") + "\n"
		s += ui.InfoStyle.Render("  Where the game's pak files are located") + "\n"
		s += ui.InfoStyle.Render("  Example: E:\\SteamLibrary\\steamapps\\common\\Grounded2\\Augusta\\Content\\Paks") + "\n\n"
		s += m.textInput.View() + "\n\n"

	case stepComplete:
		s += ui.SuccessStyle.Render("✓ Configuration saved!") + "\n\n"
		s += ui.InfoStyle.Render("Loading Pack Builder...") + "\n"
	}

	if m.err != nil {
		s += "\n" + ui.ErrorStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n"
	}

	s += "\n" + ui.InfoStyle.Render("Backspace: Go back • ESC: Quit")

	return s
}
