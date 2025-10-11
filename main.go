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
	// Load or create configuration
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

	// Launch the main menu
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

		switch finalModel.(type) {
		case retoc.RetocMenuModel:
			currentModel = mainMenu
			continue

		case retoc.PackBuilderModel:
			currentModel = retoc.NewRetocMenuModel()
			continue

		default:
			return
		}
	}
}
