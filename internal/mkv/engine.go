package mkv

import (
	"fmt"
	"mkvtea/internal/config"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ValidateDependencies checks if required MKV tools are installed
func ValidateDependencies() error {
	tools := []string{"mkvmerge", "mkvextract", "mkvpropedit"}
	var missingTools []string

	for _, tool := range tools {
		_, err := exec.LookPath(tool)
		if err != nil {
			missingTools = append(missingTools, tool)
		}
	}

	if len(missingTools) > 0 {
		return fmt.Errorf(`
❌ Missing required MKV tools: %s

These tools are part of MKVToolNix suite. Install with:

  macOS:  brew install mkvtoolnix
  Ubuntu: sudo apt install mkvtoolnix
  Fedora: sudo dnf install mkvtoolnix
  Arch:   sudo pacman -S mkvtoolnix-cli

Ensure they are in your PATH and try again.
`, strings.Join(missingTools, ", "))
	}

	return nil
}

// execute runs a command with optional dry-run mode
// If DryRun is enabled, it skips execution and returns nil
func execute(cfg config.Config, command string, args ...string) error {
	if cfg.DryRun {
		return nil
	}

	cmd := exec.Command(command, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s command failed: %v", command, err)
	}
	return nil
}

// RunExtract extracts subtitles from an MKV file based on the configured language(s)
func RunExtract(path string, cfg config.Config) error {
	info, err := GetInfo(path)
	if err != nil {
		return err
	}

	epNum := GetEpisodeNumber(filepath.Base(path))
	overallFound := false

	// If no languages specified, use the main Lang field
	languages := cfg.Languages
	if len(languages) == 0 && cfg.Lang != "" {
		languages = []string{cfg.Lang}
	}

	// Extract subtitles for each requested language
	for _, lang := range languages {
		subsDir := filepath.Join(filepath.Dir(path), "subs", lang)
		if !cfg.DryRun {
			if err := os.MkdirAll(subsDir, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create subtitle directory: %v", err)
			}
		}

		for i, t := range info.Tracks {
			if t.Type == "subtitles" && (t.Props.Lang == lang || t.Props.Lang == "und") {
				overallFound = true

				ext := ".srt"
				// Detect ASS format
				if strings.Contains(strings.ToLower(t.Codec), "ass") || strings.Contains(strings.ToLower(t.Codec), "substationalpha") {
					ext = ".ass"
				}
				// Handle forced/sign subtitle tracks
				suffix := ""
				if strings.Contains(strings.ToLower(t.Props.TrackName), "sign") || t.Props.Forced {
					suffix = "_forced"
				} else if i > 0 {
					suffix = fmt.Sprintf("_%d", i)
				}

				outName := fmt.Sprintf("%s_%s%s%s", epNum, lang, suffix, ext)
				outputPath := filepath.Join(subsDir, outName)
				if err := execute(cfg, "mkvextract", path, "tracks", fmt.Sprintf("%d:%s", t.ID, outputPath)); err != nil {
					return fmt.Errorf("subtitle extraction failed: %v", err)
				}
			}
		}
	}

	if !overallFound {
		return fmt.Errorf("skipped")
	}
	return nil
}

// RunMerge merges subtitles back into an MKV file
func RunMerge(path string, cfg config.Config) error {
	epNum := GetEpisodeNumber(filepath.Base(path))

	subsSource := cfg.SubsDir
	if subsSource == "" {
		// Default: subtitle folder relative to the MKV file
		subsSource = filepath.Join(filepath.Dir(path), "subs", cfg.Lang)
	}

	// Search for external subtitles
	hasExternalSub := false
	var subFile string
	entries, err := os.ReadDir(subsSource)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cannot read subtitle directory: %v", err)
	}

	for _, f := range entries {
		if strings.HasPrefix(f.Name(), epNum) && strings.Contains(f.Name(), cfg.Lang) && !strings.HasSuffix(f.Name(), ".xml") {
			subFile = filepath.Join(subsSource, f.Name())
			hasExternalSub = true
			break
		}
	}

	// Skip if no external subtitle found
	if !hasExternalSub {
		return fmt.Errorf("skipped")
	}

	return runMkvMergeStandard(path, subFile, subsSource, cfg)
}

func runMkvMergeStandard(path, subFile, subsSource string, cfg config.Config) error {
	info, err := GetInfo(path)
	if err != nil {
		return fmt.Errorf("failed to read MKV metadata: %v", err)
	}

	outRoot := cfg.OutDir
	if outRoot == "" {
		outRoot = filepath.Join(filepath.Dir(cfg.Dir), filepath.Base(cfg.Dir)+"_"+cfg.Lang)
	}

	// Maintain directory structure mirroring
	relPath, _ := filepath.Rel(cfg.Dir, path)
	finalOutDir := filepath.Join(outRoot, filepath.Dir(relPath))

	if !cfg.DryRun {
		if err := os.MkdirAll(finalOutDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create output directory: %v", err)
		}
	}

	args := []string{"-o", filepath.Join(finalOutDir, filepath.Base(path))}

	// Filter audio tracks if requested
	if cfg.KeepAudio != "" {
		var audioIDs []string
		for _, t := range info.Tracks {
			if t.Type == "audio" && t.Props.Lang == cfg.KeepAudio {
				audioIDs = append(audioIDs, fmt.Sprintf("%d", t.ID))
			}
		}
		if len(audioIDs) > 0 {
			args = append(args, "--audio-tracks", strings.Join(audioIDs, ","))
		}
	}

	// Remove original subtitles
	args = append(args, "--no-subtitles", path)

	// Attach fonts if found
	fonts, err := filepath.Glob(filepath.Join(subsSource, "*.[ot]t[f]"))
	if err != nil {
		return fmt.Errorf("failed to search for fonts: %v", err)
	}
	for _, f := range fonts {
		args = append(args, "--attach-file", f)
	}

	// Determine forced flag based on filename
	forcedFlag := "0:no"
	if strings.Contains(strings.ToLower(subFile), "forced") || strings.Contains(strings.ToLower(subFile), "sign") {
		forcedFlag = "0:yes"
	}

	// Add merged subtitle as default
	args = append(args,
		"--language", "0:"+cfg.Lang,
		"--track-name", "0:"+strings.ToUpper(cfg.Lang),
		"--default-track", "0:yes",
		"--forced-display-flag", forcedFlag,
		subFile,
	)

	return execute(cfg, "mkvmerge", args...)
}
