package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"mkvtea/internal/config"
	"mkvtea/internal/mkv"
	"mkvtea/internal/ui"
)

var cfg config.Config

// --- ROOT COMMAND ---
var rootCmd = &cobra.Command{
	Use:     "mkvtea",
	Short:   "🍵 Advanced MKV Tool with TUI (Extract/Merge)",
	Long:    `MKVTea is a blazing fast batch processing tool for managing your Anime/TV Series library.`,
	Version: config.Version,
}

func init() {
	// --- GLOBAL FLAGS ---
	rootCmd.PersistentFlags().StringVarP(&cfg.Lang, "lang", "l", "ita", "Target subtitle language code (ita, eng, jpn, etc.)")
	rootCmd.PersistentFlags().StringVarP(&cfg.OutDir, "output", "o", "", "Custom output directory (optional)")
	rootCmd.PersistentFlags().StringVarP(&cfg.SubsDir, "subs-dir", "s", "", "Custom directory for external subtitles (merge mode only)")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Recursive, "recursive", "r", false, "Recursively process all subdirectories")
	rootCmd.PersistentFlags().StringVarP(&cfg.KeepAudio, "audio", "a", "", "Keep only this audio language (removes all others)")
	rootCmd.PersistentFlags().IntVarP(&cfg.CheckpointInterval, "checkpoint-interval", "", 10, "Save checkpoint every N files (0 to disable)")

	// --- SUBCOMMANDS ---

	// Extract (Alias: e)
	rootCmd.AddCommand(createCmd("extract", "e",
		"(e) Extract subtitles and fonts from MKV files",
		"Extracts internal subtitles (SRT/ASS) and attached fonts from MKV files.\nOrganizes extracted files into a local 'subs' directory for each video."))

	// Merge (Alias: m)
	rootCmd.AddCommand(createCmd("merge", "m",
		"(m) Merge subtitles and fonts back into MKV files",
		"Merges external subtitles back into MKV files with proper language and default track settings.\nSupports audio track filtering and font embedding."))
}

// createCmd generates extract/merge commands with proper descriptions
func createCmd(mode, alias, short, long string) *cobra.Command {
	return &cobra.Command{
		Use:     mode + " [dir]",
		Aliases: []string{alias},
		Short:   short,
		Long:    long,
		Args:    cobra.MaximumNArgs(1),
		Example: fmt.Sprintf("  mkvtea %s . -r -l eng\n  mkvtea %s /path/to/anime -r -a jpn", alias, alias),
		Run: func(cmd *cobra.Command, args []string) {
			cfg.Mode = mode
			if len(args) > 0 {
				cfg.Dir = args[0]
			} else {
				dir, err := os.Getwd()
				if err != nil {
					fmt.Printf("❌ Failed to get current directory: %v\n", err)
					os.Exit(1)
				}
				cfg.Dir = dir
			}
			processFiles(cfg)
		},
	}
}

// calculateOptimalWorkers calculates optimal number of parallel workers based on CPU count
func calculateOptimalWorkers() int {
	// Use 50% of available CPUs, with min 2 and max 8 for balance
	maxProcs := runtime.NumCPU() / 2
	if maxProcs < 2 {
		maxProcs = 2
	}
	if maxProcs > 8 {
		maxProcs = 8
	}
	return maxProcs
}

// processFiles processes MKV files based on the configuration
func processFiles(cfg config.Config) {
	// Validate dependencies first
	if err := mkv.ValidateDependencies(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Auto-detect optimal worker count if not explicitly set
	if cfg.MaxProcs == 0 {
		cfg.MaxProcs = calculateOptimalWorkers()
	}

	// Parse multiple languages from Lang flag (e.g., "ita,eng,jpn")
	if cfg.Lang != "" {
		cfg.Languages = strings.Split(cfg.Lang, ",")
		// Trim whitespace from each language
		for i, lang := range cfg.Languages {
			cfg.Languages[i] = strings.TrimSpace(lang)
		}
	}

	// Scan for MKV files
	files := ScanFiles(cfg.Dir, cfg.Recursive)

	if len(files) == 0 {
		fmt.Printf("❌ No MKV files found in: %s\n", cfg.Dir)
		return
	}

	// Launch TUI processor
	if err := ui.RunProcessTUI(cfg, files); err != nil {
		fmt.Printf("❌ Processing error: %v\n", err)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
