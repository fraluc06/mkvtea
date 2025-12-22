package ui

import (
	"bufio"
	"fmt"
	"mkvtea/internal/checkpoint"
	"mkvtea/internal/config"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// RunProcessTUI starts the TUI processing
func RunProcessTUI(cfg config.Config, files []string) error {
	if len(files) == 0 {
		fmt.Printf("❌ No MKV files found in %s\n", cfg.Dir)
		return nil
	}

	// Check for checkpoint and offer resume if enabled
	if cfg.CheckpointInterval > 0 {
		canResume, _ := checkpoint.CanResume(cfg)
		if canResume {
			processed, remaining, total, _ := checkpoint.GetResumeStats(cfg)
			fmt.Printf("\n📋 Checkpoint found: %d/%d files already processed\n", processed, total)
			fmt.Printf("   ✅ %d complete | ⏳ %d remaining\n", processed, remaining)
			fmt.Print("   Resume processing? (y/n): ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(response)) == "y" {
				// Load checkpoint and filter out already-processed files
				manager, _ := checkpoint.NewManager(cfg)
				manager.Load()
				files = checkpoint.FilterProcessedFiles(manager, files)

				if len(files) == 0 {
					fmt.Println("✅ All files have been processed!")
					return nil
				}

				fmt.Printf("📥 Resuming with %d remaining files...\n\n", len(files))
			} else {
				// Clear checkpoint and start fresh
				manager, _ := checkpoint.NewManager(cfg)
				manager.Clear()
				fmt.Println("🔄 Starting fresh (checkpoint cleared)...")
			}
		}
	}

	model := NewProcessModel(cfg, files)
	p := tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	// Get final stats
	pm := finalModel.(*ProcessModel)

	// Show final summary
	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("📊 FINAL SUMMARY:")
	fmt.Printf("   ✅ Success: %d\n", pm.successCount)
	fmt.Printf("   ⏭️  Skipped: %d\n", pm.skippedCount)
	fmt.Printf("   ❌ Errors:  %d\n", pm.errorCount)

	// Show checkpoint info
	if cfg.CheckpointInterval > 0 {
		fmt.Printf("   💾 Checkpoint: .mkvtea_checkpoint.json (in %s)\n", cfg.Dir)
	}

	fmt.Println("==================================================")

	// Show DRY-RUN details
	if cfg.DryRun {
		fmt.Println()
		fmt.Println("📋 DRY-RUN DETAILS (no files were modified):")

		if cfg.Mode == "extract" && len(pm.extractedPaths) > 0 {
			fmt.Println("   📁 Subtitles would be saved to:")
			for _, path := range pm.extractedPaths {
				fmt.Printf("      • %s/\n", path)
			}
			fmt.Printf("   🗂️  File naming: [episode]_[lang]_[track].[srt/ass]\n")
		} else if cfg.Mode == "extract" {
			fmt.Printf("   📁 Subtitles would be saved to: ./subs/%s/\n", cfg.Lang)
		}

		if cfg.Mode == "merge" {
			if pm.outputDir != "" {
				fmt.Printf("   📁 Output directory: %s/ (%d file(s))\n", pm.outputDir, pm.successCount)
			}
			if cfg.KeepAudio != "" {
				fmt.Printf("   🔊 Audio: keeping only %s (removing others)\n", cfg.KeepAudio)
			}
			if cfg.SubsDir != "" {
				fmt.Printf("   📝 Subtitles from: %s\n", cfg.SubsDir)
			}
		}

		fmt.Println("==================================================")
	}

	return nil
}
