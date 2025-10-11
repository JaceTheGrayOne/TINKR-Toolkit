package retoc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/JaceTheGrayOne/TINKR-Toolkit/modules/config"
	"github.com/JaceTheGrayOne/TINKR-Toolkit/modules/utils"
)

// Executes the retoc packing process for a single mod
func BuildMod(ctx context.Context, log *strings.Builder, mod Mod) error {
	outUtoc := filepath.Join(filepath.Dir(mod.Path), mod.Name+".utoc")

	retocExe := filepath.Join(config.Current.RetocDir, "retoc.exe")
	if runtime.GOOS != "windows" {
		retocExe = filepath.Join(config.Current.RetocDir, "retoc")
	}

	fmt.Fprintf(log, "  Folder: %s\n", mod.Name)
	fmt.Fprintf(log, "  Output: %s\n", filepath.Base(outUtoc))

	cmd := exec.CommandContext(ctx, retocExe, "to-zen", "--version", "UE5_4", "--", mod.Path, outUtoc)
	cmd.Dir = config.Current.RetocDir

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
		dstPath := filepath.Join(config.Current.PakDir, fileName)

		if err := utils.CopyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("copy %s: %w", fileName, err)
		}

		if err := os.Remove(srcPath); err != nil {
			return fmt.Errorf("remove %s: %w", fileName, err)
		}

		fmt.Fprintf(log, "  ✓ Copied %s → Paks/\n", fileName)
	}

	return nil
}

// Builds all mods sequentially
func BuildAllAsync(ctx context.Context, mods []Mod) tea.Cmd {
	return func() tea.Msg {
		var log strings.Builder
		var builtMods []string
		var failedMods []string

		for i, mod := range mods {
			fmt.Fprintf(&log, "==== [%d/%d] Building %s ====\n", i+1, len(mods), mod.DisplayName)
			if err := BuildMod(ctx, &log, mod); err != nil {
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

		return BuildCompleteMsg{
			Log:        displayLog.String(),
			Err:        finalErr,
			BuiltMods:  builtMods,
			FailedMods: failedMods,
		}
	}
}

// Builds a single mod
func BuildOneAsync(ctx context.Context, mod Mod) tea.Cmd {
	return func() tea.Msg {
		var log strings.Builder

		fmt.Fprintf(&log, "==== Building %s ====\n", mod.DisplayName)
		err := BuildMod(ctx, &log, mod)

		var displayLog strings.Builder
		if err != nil {
			displayLog.WriteString(fmt.Sprintf("✗ %s\n", mod.DisplayName))
			return BuildCompleteMsg{
				Log:        displayLog.String(),
				Err:        err,
				BuiltMods:  []string{},
				FailedMods: []string{mod.DisplayName},
			}
		}

		displayLog.WriteString(fmt.Sprintf("✓ %s\n", mod.DisplayName))
		return BuildCompleteMsg{
			Log:        displayLog.String(),
			Err:        nil,
			BuiltMods:  []string{mod.DisplayName},
			FailedMods: []string{},
		}
	}
}

// Builds multiple mods in parallel
func BuildSelectedParallelAsync(ctx context.Context, selectedMods []Mod) tea.Cmd {
	return func() tea.Msg {
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
				err := BuildMod(ctx, &log, mod)

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
			return BuildCompleteMsg{
				Log:        displayLog.String(),
				Err:        fmt.Errorf("%d mod(s) failed to build", len(buildErrors)),
				BuiltMods:  builtMods,
				FailedMods: failedMods,
			}
		}

		return BuildCompleteMsg{
			Log:        displayLog.String(),
			Err:        nil,
			BuiltMods:  builtMods,
			FailedMods: failedMods,
		}
	}
}
