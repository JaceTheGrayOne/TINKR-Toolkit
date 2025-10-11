package config

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

// Application configuration paths
type Config struct {
	RetocDir  string `json:"retoc_dir"`
	PakDir    string `json:"pak_dir,omitempty"`
	ModsDir   string `json:"mods_dir,omitempty"`
	OutputDir string `json:"output_dir,omitempty"`
}

// Global Config
var Current Config

// Style for setup prompts
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("240"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// Load existing config or create new one
func LoadOrCreate() (Config, error) {
	exeDir, err := GetExecutableDir()
	if err != nil {
		return Config{}, err
	}

	configPath := filepath.Join(exeDir, "config.json")

	// Load existing config
	data, err := os.ReadFile(configPath)
	if err == nil {
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err == nil {
			if cfg.RetocDir != "" {
				Current = cfg
				return cfg, nil
			}
		}
	}

	// Run setup
	cfg, err := runSetup(exeDir)
	if err != nil {
		return Config{}, err
	}

	// Save config
	jsonData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return cfg, err
	}

	if err := os.WriteFile(configPath, jsonData, 0644); err != nil {
		return cfg, err
	}

	Current = cfg
	return cfg, nil
}

func runSetup(exeDir string) (Config, error) {
	retocDir := filepath.Join(exeDir, "retoc")

	cfg := Config{
		RetocDir: retocDir,
	}

	return cfg, nil
}

// Prompt for mods directory
func PromptForModsDir() (string, error) {
	fmt.Println(titleStyle.Render("Pack Setup"))
	fmt.Println()
	fmt.Println("Modified UAsset/UEXP Directory:")
	fmt.Println(infoStyle.Render("  Directory of your modified game files"))
	fmt.Println(infoStyle.Render("  Example: G:\\Grounded\\Modding\\Grounded2\\Mods"))
	fmt.Print("> ")

	reader := bufio.NewReader(os.Stdin)
	modsInput, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	modsDir, err := NormalizePath(modsInput)
	if err != nil {
		return "", fmt.Errorf("directory is invalid: %w", err)
	}

	// Validate directory
	if _, err := os.Stat(modsDir); err != nil {
		return "", fmt.Errorf("directory not found: %s", modsDir)
	}

	// Save to config
	Current.ModsDir = modsDir
	if err := saveConfig(); err != nil {
		return "", err
	}

	fmt.Println()
	fmt.Println(successStyle.Render("Mods directory saved to config."))
	fmt.Println()

	return modsDir, nil
}

// Prompt for pak directory
func PromptForPakDir() (string, error) {
	fmt.Println("UE Game \"Paks\" Directory:")
	fmt.Println(infoStyle.Render("  Example: E:\\SteamLibrary\\steamapps\\common\\Grounded2\\Augusta\\Content\\Paks"))
	fmt.Print("> ")

	reader := bufio.NewReader(os.Stdin)
	pakInput, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	pakDir, err := NormalizePath(pakInput)
	if err != nil {
		return "", fmt.Errorf("directory is invalid: %w", err)
	}

	// Save to config
	Current.PakDir = pakDir
	if err := saveConfig(); err != nil {
		return "", err
	}

	fmt.Println()
	fmt.Println(successStyle.Render("Pak directory saved to config."))
	fmt.Println()

	return pakDir, nil
}

// Prompt for output directory
func PromptForOutputDir() (string, error) {
	fmt.Println(titleStyle.Render("Unpack Setup"))
	fmt.Println()
	fmt.Println("Output Directory:")
	fmt.Println(infoStyle.Render("  Where extracted assets will be saved"))
	fmt.Println(infoStyle.Render("  Example: G:\\Grounded\\Modding\\Extracted"))
	fmt.Print("> ")

	reader := bufio.NewReader(os.Stdin)
	outputInput, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	outputDir, err := NormalizePath(outputInput)
	if err != nil {
		return "", fmt.Errorf("directory is invalid: %w", err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("couldn't create directory: %w", err)
	}

	// Save to config
	Current.OutputDir = outputDir
	if err := saveConfig(); err != nil {
		return "", err
	}

	fmt.Println()
	fmt.Println(successStyle.Render("Output directory saved to config."))
	fmt.Println()

	return outputDir, nil
}

// Save the current config to disk
func saveConfig() error {
	exeDir, err := GetExecutableDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(exeDir, "config.json")
	jsonData, err := json.MarshalIndent(Current, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, jsonData, 0644)
}

func SaveConfig() error {
	return saveConfig()
}
