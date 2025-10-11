package retoc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/JaceTheGrayOne/TINKR-Toolkit/modules/config"
	"github.com/JaceTheGrayOne/TINKR-Toolkit/modules/utils"
)

// DiscoverMods scans the configured mods directory and returns all valid mod folders
func DiscoverMods() ([]Mod, error) {
	entries, err := os.ReadDir(config.Current.ModsDir)
	if err != nil {
		return nil, err
	}

	var mods []Mod
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			folderName := entry.Name()
			mods = append(mods, Mod{
				Name:        folderName,
				DisplayName: utils.FormatDisplayName(folderName),
				Path:        filepath.Join(config.Current.ModsDir, folderName),
			})
		}
	}

	if len(mods) == 0 {
		return nil, errors.New("no mods found")
	}

	return mods, nil
}
