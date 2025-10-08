package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Config holds all application configuration
type Config struct {
	RetocDir string `json:"retoc_dir"`
	PakDir   string `json:"pak_dir"`
	ModsDir  string `json:"mods_dir"`
}

// Global config
var cfg Config

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("240")) // Gray

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")). // Light blue (PowerShell blue)
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	buildingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	checkboxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")) // Light blue checkbox
)

type Mod struct {
	Name        string
	DisplayName string
	Path        string
}

type buildCompleteMsg struct {
	log        string
	err        error
	builtMods  []string
	failedMods []string
}

type model struct {
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

func getExecutableDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exe), nil
}

func normalizePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	path = strings.Trim(path, "\"")
	path = strings.Trim(path, "'")
	path = os.ExpandEnv(path)

	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("couldn't expand ~: %w", err)
		}
		path = filepath.Join(homeDir, path[1:])
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("couldn't convert to absolute path: %w", err)
	}

	cleanPath := filepath.Clean(absPath)
	cleanPath = strings.TrimSuffix(cleanPath, string(filepath.Separator))

	return cleanPath, nil
}

func loadOrCreateConfig() (Config, error) {
	exeDir, err := getExecutableDir()
	if err != nil {
		return Config{}, err
	}

	configPath := filepath.Join(exeDir, "config.json")

	data, err := os.ReadFile(configPath)
	if err == nil {
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err == nil {
			if cfg.RetocDir != "" && cfg.PakDir != "" && cfg.ModsDir != "" {
				return cfg, nil
			}
		}
	}

	retocDir := filepath.Join(exeDir, "retoc")

	fmt.Println(titleStyle.Render("First Time Setup"))
	fmt.Println()
	fmt.Println("Let's configure PakBuilder. Press Enter after each path.")
	fmt.Println(infoStyle.Render("(Paths with spaces don't need quotes)"))
	fmt.Println()

	fmt.Println("Modified UAsset/UEXP Directory:")
	fmt.Println(infoStyle.Render("  Directory of your modified game files"))
	fmt.Println(infoStyle.Render("  Example: G:\\Grounded\\Modding\\Grounded2\\Mods"))
	fmt.Println(infoStyle.Render("  Supports: quotes, ~, ./, ../, environment variables"))
	fmt.Print("> ")

	reader := bufio.NewReader(os.Stdin)
	modsInput, err := reader.ReadString('\n')
	if err != nil {
		return Config{}, err
	}

	modsDir, err := normalizePath(modsInput)
	if err != nil {
		return Config{}, fmt.Errorf("invalid mods directory: %w", err)
	}

	fmt.Println()

	fmt.Println("UE Game \"Paks\" Directory:")
	fmt.Println(infoStyle.Render("  Directory of the \"Paks\" folder for the game"))
	fmt.Println(infoStyle.Render("  Example: E:\\SteamLibrary\\steamapps\\common\\Grounded2\\Augusta\\Content\\Paks"))
	fmt.Println(infoStyle.Render("  Supports: quotes, ~, ./, ../, environment variables"))
	fmt.Print("> ")

	pakInput, err := reader.ReadString('\n')
	if err != nil {
		return Config{}, err
	}

	pakDir, err := normalizePath(pakInput)
	if err != nil {
		return Config{}, fmt.Errorf("invalid pak directory: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render("✓ Paths normalized:"))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Mods: %s", modsDir)))
	fmt.Println(infoStyle.Render(fmt.Sprintf("  Paks: %s", pakDir)))
	fmt.Println()

	cfg := Config{
		RetocDir: retocDir,
		ModsDir:  modsDir,
		PakDir:   pakDir,
	}

	jsonData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return cfg, err
	}

	if err := os.WriteFile(configPath, jsonData, 0644); err != nil {
		return cfg, err
	}

	fmt.Println()
	fmt.Println(successStyle.Render("✓ Configuration saved to config.json"))
	fmt.Println()

	if _, err := os.Stat(cfg.ModsDir); err != nil {
		return cfg, fmt.Errorf("mods directory not found: %s", cfg.ModsDir)
	}

	fmt.Println(successStyle.Render("✓ Configuration validated"))
	fmt.Println()
	time.Sleep(1 * time.Second)

	return cfg, nil
}

func formatDisplayName(folderName string) string {
	name := strings.TrimPrefix(folderName, "z_")
	name = strings.TrimPrefix(name, "Z_")

	if idx := strings.LastIndex(name, "_"); idx != -1 {
		if strings.HasSuffix(name, "_P") {
			parts := strings.Split(name, "_")
			if len(parts) >= 2 {
				secondLast := parts[len(parts)-2]
				if len(secondLast) == 4 {
					allDigits := true
					for _, c := range secondLast {
						if c < '0' || c > '9' {
							allDigits = false
							break
						}
					}
					if allDigits {
						name = strings.Join(parts[:len(parts)-2], "_")
					}
				}
			}
		}
	}

	name = strings.ReplaceAll(name, "_", " ")
	return name
}

func discoverMods() ([]Mod, error) {
	entries, err := os.ReadDir(cfg.ModsDir)
	if err != nil {
		return nil, err
	}

	var mods []Mod
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			folderName := entry.Name()
			mods = append(mods, Mod{
				Name:        folderName,
				DisplayName: formatDisplayName(folderName),
				Path:        filepath.Join(cfg.ModsDir, folderName),
			})
		}
	}

	if len(mods) == 0 {
		return nil, errors.New("no mods found")
	}

	return mods, nil
}

func initialModel(mods []Mod) model {
	return model{
		mods:     mods,
		cursor:   0,
		selected: make(map[int]bool),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.building {
			switch msg.String() {
			case "ctrl+c", "esc":
				if m.cancel != nil {
					m.cancel()
				}
				return m, tea.Quit
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
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
			return m, m.buildAllAsync()

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
				return m, m.buildOneAsync(selectedMod)
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
				return m, m.buildAllAsync()
			} else if len(m.selected) > 1 {
				m.currentTask = fmt.Sprintf("%d Selected Mods", len(m.selected))
				m.parallelMode = true
				return m, m.buildSelectedParallelAsync()
			} else if len(m.selected) == 1 {
				m.parallelMode = false
				for i := range m.mods {
					if m.selected[i] {
						m.currentTask = m.mods[i].DisplayName
						return m, m.buildOneAsync(m.mods[i])
					}
				}
			} else {
				m.parallelMode = false
				selectedMod := m.mods[m.cursor-1]
				m.currentTask = selectedMod.DisplayName
				return m, m.buildOneAsync(selectedMod)
			}
		}

	case buildCompleteMsg:
		m.building = false
		m.log = msg.log
		m.err = msg.err
		m.parallelMode = false
		if msg.err == nil {
			m.selected = make(map[int]bool)
		}
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	if m.building {
		elapsed := time.Since(m.buildStart)
		if elapsed > 500*time.Millisecond {
			return m.buildingView()
		}
		return m.menuView()
	}
	return m.menuView()
}

func (m model) menuView() string {
	s := titleStyle.Render("TINK.R Toolkit - Pak Builder") + "\n\n"

	cursor := " "
	if m.cursor == 0 {
		cursor = ">"
		s += selectedStyle.Render(fmt.Sprintf("%s [ - ] - 0. Build ALL", cursor)) + "\n"
	} else {
		s += normalStyle.Render(fmt.Sprintf("%s [ - ] - 0. Build ALL", cursor)) + "\n"
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
			// Selected line - everything is light blue
			if m.selected[i] {
				s += selectedStyle.Render(fmt.Sprintf("%s [%s] - %d. %s", cursor, checkbox, hotkey, modName)) + "\n"
			} else {
				s += selectedStyle.Render(fmt.Sprintf("%s [%s] - %d. %s", cursor, checkbox, hotkey, modName)) + "\n"
			}
		} else {
			// Normal line
			if m.selected[i] {
				// Checkbox is checked - make it light blue, rest is normal
				s += normalStyle.Render(fmt.Sprintf("%s [", cursor)) + checkboxStyle.Render(checkbox) + normalStyle.Render(fmt.Sprintf("] - %d. %s", hotkey, modName)) + "\n"
			} else {
				// Everything normal
				s += normalStyle.Render(fmt.Sprintf("%s [%s] - %d. %s", cursor, checkbox, hotkey, modName)) + "\n"
			}
		}
	}

	s += "\n"

	if len(m.selected) > 0 {
		s += infoStyle.Render(fmt.Sprintf("%d mod(s) selected", len(m.selected))) + "\n"
	} else {
		s += "\n"
	}

	s += "\n"

	if m.err != nil {
		s += errorStyle.Render("Error: ") + m.err.Error() + "\n"
	} else if m.log != "" {
		lines := strings.Split(strings.TrimSpace(m.log), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "✓") {
				s += successStyle.Render("✓") + " " + normalStyle.Render(strings.TrimPrefix(line, "✓ ")) + "\n"
			} else if strings.HasPrefix(line, "✗") {
				s += errorStyle.Render("✗") + " " + normalStyle.Render(strings.TrimPrefix(line, "✗ ")) + "\n"
			}
		}
	} else {
		s += "\n"
	}

	s += "\nSpace to select • Enter to build • 0-9 hotkeys"

	return s
}

func (m model) buildingView() string {
	elapsed := time.Since(m.startTime).Round(time.Second)

	s := buildingStyle.Render(fmt.Sprintf("⚙️  Building: %s", m.currentTask)) + "\n"
	s += fmt.Sprintf("Elapsed: %s\n\n", elapsed)

	if m.parallelMode {
		s += infoStyle.Render("Building mods in parallel...") + "\n"
		s += infoStyle.Render("This may take a moment depending on mod count.") + "\n"
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

func (m model) buildSelectedParallelAsync() tea.Cmd {
	return func() tea.Msg {
		var selectedMods []Mod
		for i := range m.mods {
			if m.selected[i] {
				selectedMods = append(selectedMods, m.mods[i])
			}
		}

		var wg sync.WaitGroup
		var mu sync.Mutex
		var finalLog strings.Builder
		var buildErrors []error
		var builtMods []string
		var failedMods []string

		for _, mod := range selectedMods {
			wg.Add(1)
			go func(mod Mod) {
				defer wg.Done()

				var log strings.Builder
				err := buildMod(m.ctx, &log, mod)

				mu.Lock()
				finalLog.WriteString(fmt.Sprintf("==== %s ====\n", mod.DisplayName))
				finalLog.WriteString(log.String())
				finalLog.WriteString("\n")

				if err != nil {
					buildErrors = append(buildErrors, fmt.Errorf("%s: %w", mod.DisplayName, err))
					failedMods = append(failedMods, mod.DisplayName)
				} else {
					builtMods = append(builtMods, mod.DisplayName)
				}
				mu.Unlock()
			}(mod)
		}

		wg.Wait()

		var displayLog strings.Builder
		for _, modName := range builtMods {
			displayLog.WriteString(fmt.Sprintf("✓ %s\n", modName))
		}
		for _, modName := range failedMods {
			displayLog.WriteString(fmt.Sprintf("✗ %s\n", modName))
		}

		if len(buildErrors) > 0 {
			return buildCompleteMsg{
				log:        displayLog.String(),
				err:        fmt.Errorf("%d mod(s) failed to build", len(buildErrors)),
				builtMods:  builtMods,
				failedMods: failedMods,
			}
		}

		return buildCompleteMsg{
			log:        displayLog.String(),
			err:        nil,
			builtMods:  builtMods,
			failedMods: failedMods,
		}
	}
}

func (m model) buildAllAsync() tea.Cmd {
	return func() tea.Msg {
		var log strings.Builder
		var builtMods []string
		var failedMods []string

		for i, mod := range m.mods {
			fmt.Fprintf(&log, "==== [%d/%d] Building %s ====\n", i+1, len(m.mods), mod.DisplayName)
			if err := buildMod(m.ctx, &log, mod); err != nil {
				failedMods = append(failedMods, mod.DisplayName)
			} else {
				builtMods = append(builtMods, mod.DisplayName)
			}
			log.WriteString("\n")
		}

		var displayLog strings.Builder
		for _, modName := range builtMods {
			displayLog.WriteString(fmt.Sprintf("✓ %s\n", modName))
		}
		for _, modName := range failedMods {
			displayLog.WriteString(fmt.Sprintf("✗ %s\n", modName))
		}

		var finalErr error
		if len(failedMods) > 0 {
			finalErr = fmt.Errorf("%d mod(s) failed to build", len(failedMods))
		}

		return buildCompleteMsg{
			log:        displayLog.String(),
			err:        finalErr,
			builtMods:  builtMods,
			failedMods: failedMods,
		}
	}
}

func (m model) buildOneAsync(mod Mod) tea.Cmd {
	return func() tea.Msg {
		var log strings.Builder

		fmt.Fprintf(&log, "==== Building %s ====\n", mod.DisplayName)
		err := buildMod(m.ctx, &log, mod)

		var displayLog strings.Builder
		if err != nil {
			displayLog.WriteString(fmt.Sprintf("✗ %s\n", mod.DisplayName))
			return buildCompleteMsg{
				log:        displayLog.String(),
				err:        err,
				builtMods:  []string{},
				failedMods: []string{mod.DisplayName},
			}
		}

		displayLog.WriteString(fmt.Sprintf("✓ %s\n", mod.DisplayName))
		return buildCompleteMsg{
			log:        displayLog.String(),
			err:        nil,
			builtMods:  []string{mod.DisplayName},
			failedMods: []string{},
		}
	}
}

func buildMod(ctx context.Context, log *strings.Builder, mod Mod) error {
	outUtoc := filepath.Join(filepath.Dir(mod.Path), mod.Name+".utoc")

	retocExe := filepath.Join(cfg.RetocDir, "retoc.exe")
	if runtime.GOOS != "windows" {
		retocExe = filepath.Join(cfg.RetocDir, "retoc")
	}

	fmt.Fprintf(log, "  Folder: %s\n", mod.Name)
	fmt.Fprintf(log, "  Output: %s\n", filepath.Base(outUtoc))

	cmd := exec.CommandContext(ctx, retocExe, "to-zen", "--version", "UE5_4", "--", mod.Path, outUtoc)
	cmd.Dir = cfg.RetocDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.Canceled {
			return errors.New("build cancelled")
		}
		fmt.Fprintf(log, "  retoc error: %s\n", strings.TrimSpace(string(output)))
		return fmt.Errorf("retoc failed: %w", err)
	}

	if len(output) > 0 {
		fmt.Fprintf(log, "  retoc: %s\n", strings.TrimSpace(string(output)))
	}

	pattern := filepath.Join(filepath.Dir(mod.Path), mod.Name+".*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		return fmt.Errorf("no output files found")
	}

	fmt.Fprintf(log, "  Found %d file(s) to copy\n", len(matches))

	for _, srcPath := range matches {
		fileName := filepath.Base(srcPath)
		dstPath := filepath.Join(cfg.PakDir, fileName)

		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("copy %s: %w", fileName, err)
		}

		if err := os.Remove(srcPath); err != nil {
			return fmt.Errorf("remove %s: %w", fileName, err)
		}

		fmt.Fprintf(log, "  ✓ Copied %s → Paks/\n", fileName)
	}

	return nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	tmpPath := dst + ".tmp"
	dstFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(dstFile, srcFile)
	closeErr := dstFile.Close()

	if copyErr != nil {
		os.Remove(tmpPath)
		return copyErr
	}

	if closeErr != nil {
		os.Remove(tmpPath)
		return closeErr
	}

	return os.Rename(tmpPath, dst)
}

func main() {
	var err error
	cfg, err = loadOrCreateConfig()
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		fmt.Println("\nPress Enter to exit...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		os.Exit(1)
	}

	if _, err := os.Stat(cfg.RetocDir); err != nil {
		fmt.Printf("❌ Retoc directory not found: %s\n", cfg.RetocDir)
		fmt.Println("   Make sure retoc.exe is in the 'retoc' subfolder")
		fmt.Println("\nPress Enter to exit...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		os.Exit(1)
	}

	if _, err := os.Stat(cfg.ModsDir); err != nil {
		fmt.Printf("❌ Mods directory not found: %s\n", cfg.ModsDir)
		fmt.Println("   Edit config.json to fix the path")
		fmt.Println("\nPress Enter to exit...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		os.Exit(1)
	}

	if _, err := os.Stat(cfg.PakDir); err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Printf("❌ Pak directory error: %s\n", cfg.PakDir)
		fmt.Println("   Edit config.json to fix the path")
		fmt.Println("\nPress Enter to exit...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		os.Exit(1)
	}

	mods, err := discoverMods()
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		fmt.Println("\nPress Enter to exit...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		os.Exit(1)
	}

	fmt.Printf("✓ Found %d mod(s)\n", len(mods))
	time.Sleep(500 * time.Millisecond)

	p := tea.NewProgram(initialModel(mods), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nPress Enter to exit...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		os.Exit(1)
	}
}