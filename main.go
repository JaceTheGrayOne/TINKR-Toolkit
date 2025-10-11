package main

import (
	"bufio"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/JaceTheGrayOne/TINKR-Toolkit/modules/config"
	"github.com/JaceTheGrayOne/TINKR-Toolkit/modules/retoc"
	"github.com/JaceTheGrayOne/TINKR-Toolkit/modules/ui"
)

func main() {
	// Load or create config
	var err error
	config.Current, err = config.LoadOrCreate()
	if err != nil {
		fmt.Printf("❌ Failed to load config: %v\n", err)
		fmt.Println("\nPress Enter to exit...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		os.Exit(1)
	}

	// Validate retoc directory
	if _, err := os.Stat(config.Current.RetocDir); err != nil {
		fmt.Printf("❌ Retoc directory not found: %s\n", config.Current.RetocDir)
		fmt.Println("   Make sure retoc.exe is in the 'retoc' subfolder")
		fmt.Println("\nPress Enter to exit...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		os.Exit(1)
	}

	// Create main menu with available tools
	tools := []ui.Tool{
		{
			Name:        "Retoc",
			Description: "Zen asset packer/unpacker for Unreal Engine",
			Model:       retoc.NewRetocMenuModel(),
		},
		// Future tools go here:
		// {
		//     Name:        "Tool2",
		//     Description: "Description of tool 2",
		//     Model:       tool2.NewTool2Menu(),
		// },
	}

	// Launch main menu
	mainMenu := ui.NewMainMenuModel(tools)
	currentModel := tea.Model(mainMenu)

	for {
		p := tea.NewProgram(currentModel, tea.WithAltScreen())
		finalModel, err := p.Run()

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Println("\nPress Enter to exit...")
			bufio.NewReader(os.Stdin).ReadString('\n')
			os.Exit(1)
		}

		// Handle navigation between models
		switch finalModel.(type) {
		case retoc.RetocMenuModel:
			// Return from Retoc menu to main menu
			currentModel = mainMenu
			continue

		case retoc.PackBuilderModel:
			// Return from Pack Builder to Retoc menu
			currentModel = retoc.NewRetocMenuModel()
			continue

		case retoc.PackSetupModel:
			// Check if setup complete
			if config.Current.ModsDir != "" && config.Current.PakDir != "" {
				// Discover mods and transition to pack builder
				mods, err := retoc.DiscoverMods()
				if err == nil && len(mods) > 0 {
					currentModel = retoc.NewPackBuilderModel(mods)
					continue
				}
			}
			// Setup cancelled or failed - return to Retoc menu
			currentModel = retoc.NewRetocMenuModel()
			continue

		case retoc.UnpackSetupModel:
			// Return from Unpack Setup to Retoc menu
			currentModel = retoc.NewRetocMenuModel()
			continue

		case ui.MainMenuModel:
			// If BackMsg to main menu, quit
			return

		default:
			// Any other case, quit
			return
		}
	}
}
