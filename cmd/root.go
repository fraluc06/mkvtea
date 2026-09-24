package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"mkvtea/internal/config"
	"mkvtea/internal/encode"
	"mkvtea/internal/mkv"
	"mkvtea/internal/ui"
)

var cfg config.Config

// --- ROOT COMMAND ---
var rootCmd = &cobra.Command{
	Use:   "mkvtea",
	Short: "🍵 Advanced MKV Tool with TUI (Extract/Merge)",
	Long:  `MKVTea is a blazing fast batch processing tool for managing your Anime/TV Series library.`,
	// Errors are printed once by Execute; usage spam on runtime errors is noise.
	Version:       config.Version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	// --- GLOBAL FLAGS ---
	rootCmd.PersistentFlags().StringVarP(&cfg.Lang, "lang", "l", "ita", "Target subtitle language code (ita, eng, jpn, etc.)")
	rootCmd.PersistentFlags().StringVarP(&cfg.OutDir, "output", "o", "", "Custom output directory (optional)")
	rootCmd.PersistentFlags().StringVarP(&cfg.SubsDir, "subs-dir", "s", "", "Custom directory for external subtitles (merge mode only)")
	rootCmd.PersistentFlags().StringVar(&cfg.AudioDir, "audio-dir", "", "Custom directory for external audio (merge mode only)")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Recursive, "recursive", "r", false, "Recursively process all subdirectories")
	rootCmd.PersistentFlags().BoolVarP(&cfg.Audio, "audio", "a", false, "Extract or merge audio tracks of the target language")
	rootCmd.PersistentFlags().StringVar(&cfg.KeepOnlyAudio, "keep-only-audio", "", "Keep only this audio language (removes all others)")
	rootCmd.PersistentFlags().IntVarP(&cfg.CheckpointInterval, "checkpoint-interval", "", 10, "Save checkpoint every N files (0 to disable)")

	// --- SUBCOMMANDS ---

	// Extract (Alias: e)
	rootCmd.AddCommand(createCmd("extract", "e",
		"(e) Extract subtitles, audio, and fonts from MKV files",
		"Extracts internal subtitles (SRT/ASS), audio tracks, and attached fonts from MKV files.\nOrganizes extracted files into a local 'subs' directory for each video.",
		"  mkvtea e . -r -l ita -a\n  mkvtea e /path/to/anime -r -l eng"))

	// Merge (Alias: m)
	rootCmd.AddCommand(createCmd("merge", "m",
		"(m) Merge subtitles, audio, and fonts back into MKV files",
		"Merges external subtitles and audio tracks back into MKV files with proper language and default track settings.\nSupports audio track filtering and font embedding.",
		"  mkvtea m . -r -l ita -a\n  mkvtea m /path/to/anime -r -l eng"))

	// Encode (Alias: en) — batch AV1 encoding via SVT-AV1-Essential.
	encodeCmd := createCmd("encode", "en",
		"(en) Encode videos to AV1 with SVT-AV1-Essential",
		"Encodes every video file to AV1 using SvtAv1EncApp, then remuxes all audio, subtitles,\n"+
			"fonts and chapters from the original file. Results are written to an 'av1' subdirectory\n"+
			"next to each source (or under -o). Encoder parameters come from --svtav1-params, a\n"+
			"params file, or a per-folder .mkvtea-av1-params (otherwise encoder defaults are used).\n"+
			"The -a, -s and --audio-dir flags do not apply to this mode.",
		"  mkvtea en /path/to/anime -r\n  mkvtea en . -r --svtav1-params \"crf=30:speed=slow:film-grain=8\"\n  mkvtea en . -l ita -r")
	encodeCmd.Flags().StringVar(&cfg.SvtAv1Params, "svtav1-params", "",
		"SVT-AV1 params, ffmpeg-style: \"crf=30:tune=0:film-grain=8\" (overrides param files)")
	encodeCmd.Flags().StringVar(&cfg.Av1ParamsFile, "av1-params-file", "",
		"File with one SVT-AV1 \"key=value\" per line (# comments allowed)")
	encodeCmd.Flags().StringVar(&cfg.OutSubdir, "out-subdir", encode.DefaultOutSubdir,
		"Per-folder output subdirectory name (encode mode)")
	rootCmd.AddCommand(encodeCmd)
}

// createCmd generates an extract/merge/encode command with proper descriptions
func createCmd(mode, alias, short, long, example string) *cobra.Command {
	return &cobra.Command{
		Use:     mode + " [dir]",
		Aliases: []string{alias},
		Short:   short,
		Long:    long,
		Args:    cobra.MaximumNArgs(1),
		Example: example,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Mode = mode
			// encode only filters subtitles when -l was actually typed; the
			// persistent flag defaults to "ita" which is NOT an explicit choice.
			cfg.LangExplicit = cmd.Flags().Changed("lang")
			if len(args) > 0 {
				cfg.Dir = args[0]
			} else {
				dir, err := os.Getwd()
				if err != nil {
					return fmt.Errorf("failed to get current directory: %w", err)
				}
				cfg.Dir = dir
			}

			// Ensure Dir is an absolute path to avoid issues with "." or relative paths
			// when calculating output directory names.
			absDir, err := filepath.Abs(cfg.Dir)
			if err != nil {
				return fmt.Errorf("failed to resolve path %q: %w", cfg.Dir, err)
			}
			cfg.Dir = absDir

			return processFiles(cfg)
		},
	}
}

// calculateOptimalWorkers calculates optimal number of parallel workers based on CPU count
func calculateOptimalWorkers() int {
	// Use 50% of available CPUs, with min 2 and max 8 for balance
	return min(max(runtime.NumCPU()/2, 2), 8)
}

// processFiles processes files based on the configuration
func processFiles(cfg config.Config) error {
	// Validate dependencies for the selected mode
	switch cfg.Mode {
	case "encode":
		if err := encode.ValidateDependencies(); err != nil {
			return err
		}
		// Fail fast on a malformed global param source before scanning.
		if err := encode.ValidateParams(cfg); err != nil {
			return err
		}
	default:
		if err := mkv.ValidateDependencies(); err != nil {
			return err
		}
	}

	// Auto-detect optimal worker count if not explicitly set
	if cfg.MaxProcs == 0 {
		if cfg.Mode == "encode" {
			// SvtAv1EncApp already saturates cores on its own; parallel encodes
			// would only thrash, so run one at a time by default.
			cfg.MaxProcs = 1
		} else {
			cfg.MaxProcs = calculateOptimalWorkers()
		}
	}

	// Parse multiple languages from Lang flag (e.g., "ita,eng,jpn")
	if cfg.Lang != "" {
		cfg.Languages = strings.Split(cfg.Lang, ",")
		// Trim whitespace from each language
		for i, lang := range cfg.Languages {
			cfg.Languages[i] = strings.TrimSpace(lang)
		}
	}

	// Scan for video files
	files := ScanFiles(cfg)

	if len(files) == 0 {
		if cfg.Mode == "encode" {
			fmt.Printf("❌ No video files found in: %s\n", cfg.Dir)
		} else {
			fmt.Printf("❌ No MKV files found in: %s\n", cfg.Dir)
		}
		return nil
	}

	// Launch TUI processor
	if err := ui.RunProcessTUI(cfg, files); err != nil {
		return fmt.Errorf("processing: %w", err)
	}
	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
