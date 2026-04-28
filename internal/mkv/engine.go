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
		return fmt.Errorf(`❌ Missing required MKV tools: %s

These tools are part of MKVToolNix suite. Install with:

  macOS:  brew install mkvtoolnix
  Ubuntu: sudo apt install mkvtoolnix
  Fedora: sudo dnf install mkvtoolnix
  Arch:   sudo pacman -S mkvtoolnix-cli

Ensure they are in your PATH and try again.`, strings.Join(missingTools, ", "))
	}

	return nil
}

// execute runs a command
func execute(command string, args ...string) error {
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
		if err := os.MkdirAll(subsDir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create subtitle directory: %v", err)
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
				if err := execute("mkvextract", path, "tracks", fmt.Sprintf("%d:%s", t.ID, outputPath)); err != nil {
					return fmt.Errorf("subtitle extraction failed: %v", err)
				}
			}

			// Extract audio if requested and matches language
			if cfg.Audio && t.Type == "audio" && t.Props.Lang == lang {
				overallFound = true
				ext := getAudioExtension(t.Codec)
				suffix := ""
				if i > 0 {
					suffix = fmt.Sprintf("_%d", i)
				}
				outName := fmt.Sprintf("%s_%s%s%s", epNum, lang, suffix, ext)
				outputPath := filepath.Join(subsDir, outName)
				if err := execute("mkvextract", path, "tracks", fmt.Sprintf("%d:%s", t.ID, outputPath)); err != nil {
					return fmt.Errorf("audio extraction failed: %v", err)
				}
			}
		}
	}

	if !overallFound {
		return fmt.Errorf("skipped")
	}
	return nil
}

// RunMerge merges subtitles and audio back into an MKV file
func RunMerge(path string, cfg config.Config) error {
	epNum := GetEpisodeNumber(filepath.Base(path))

	subsSource := cfg.SubsDir
	if subsSource == "" {
		subsSource = filepath.Join(filepath.Dir(path), "subs", cfg.Lang)
	}

	audioSource := cfg.AudioDir
	if audioSource == "" {
		audioSource = subsSource
	}

	var subFile, audioFile string
	hasSub, hasAudio := false, false

	// Search for subtitles
	if entries, err := os.ReadDir(subsSource); err == nil {
		for _, f := range entries {
			if strings.HasPrefix(f.Name(), epNum) && strings.Contains(f.Name(), cfg.Lang) && !strings.HasSuffix(f.Name(), ".xml") {
				ext := strings.ToLower(filepath.Ext(f.Name()))
				if ext == ".srt" || ext == ".ass" {
					subFile = filepath.Join(subsSource, f.Name())
					hasSub = true
					break
				}
			}
		}
	}

	// Search for audio
	if cfg.Audio {
		if entries, err := os.ReadDir(audioSource); err == nil {
			for _, f := range entries {
				if strings.HasPrefix(f.Name(), epNum) && strings.Contains(f.Name(), cfg.Lang) && !strings.HasSuffix(f.Name(), ".xml") {
					ext := strings.ToLower(filepath.Ext(f.Name()))
					if isAudioExt(ext) {
						audioFile = filepath.Join(audioSource, f.Name())
						hasAudio = true
						break
					}
				}
			}
		}
	}

	// Skip if nothing found
	if !hasSub && !hasAudio {
		return fmt.Errorf("skipped")
	}

	return runMkvMergeStandard(path, subFile, audioFile, subsSource, cfg)
}

func runMkvMergeStandard(path, subFile, audioFile, subsSource string, cfg config.Config) error {
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

	if err := os.MkdirAll(finalOutDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Ensure output filename ends in .mkv
	outName := filepath.Base(path)
	if ext := filepath.Ext(outName); ext != ".mkv" {
		outName = strings.TrimSuffix(outName, ext) + ".mkv"
	}
	args := []string{"-o", filepath.Join(finalOutDir, outName)}

	// Filter audio tracks if requested
	if cfg.KeepOnlyAudio != "" {
		var audioIDs []string
		for _, t := range info.Tracks {
			if t.Type == "audio" && t.Props.Lang == cfg.KeepOnlyAudio {
				audioIDs = append(audioIDs, fmt.Sprintf("%d", t.ID))
			}
		}
		if len(audioIDs) > 0 {
			args = append(args, "--audio-tracks", strings.Join(audioIDs, ","))
		}
	}

	// If we are merging a new audio file, we might want to set other audio tracks as NOT default
	if audioFile != "" {
		for _, t := range info.Tracks {
			if t.Type == "audio" {
				args = append(args, "--default-track", fmt.Sprintf("%d:no", t.ID))
			}
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

	// Add audio if found
	if audioFile != "" {
		args = append(args,
			"--language", "0:"+cfg.Lang,
			"--track-name", "0:"+strings.ToUpper(cfg.Lang),
			"--default-track", "0:yes",
			audioFile,
		)
	}

	// Add subtitle if found
	if subFile != "" {
		// Determine forced flag based on filename
		forcedFlag := "0:no"
		if strings.Contains(strings.ToLower(subFile), "forced") || strings.Contains(strings.ToLower(subFile), "sign") {
			forcedFlag = "0:yes"
		}

		args = append(args,
			"--language", "0:"+cfg.Lang,
			"--track-name", "0:"+strings.ToUpper(cfg.Lang),
			"--default-track", "0:yes",
			"--forced-display-flag", forcedFlag,
			subFile,
		)
	}

	return execute("mkvmerge", args...)
}

func getAudioExtension(codec string) string {
	codec = strings.ToLower(codec)
	switch {
	case strings.Contains(codec, "eac3") || strings.Contains(codec, "e-ac-3"):
		return ".eac3"
	case strings.Contains(codec, "ac3") || strings.Contains(codec, "ac-3"):
		return ".ac3"
	case strings.Contains(codec, "dts"):
		return ".dts"
	case strings.Contains(codec, "flac"):
		return ".flac"
	case strings.Contains(codec, "aac"):
		return ".aac"
	case strings.Contains(codec, "mp3") || strings.Contains(codec, "mpeg-1 layer 3"):
		return ".mp3"
	case strings.Contains(codec, "opus"):
		return ".opus"
	case strings.Contains(codec, "vorbis"):
		return ".ogg"
	case strings.Contains(codec, "pcm"):
		return ".wav"
	default:
		return ".mka"
	}
}

func isAudioExt(ext string) bool {
	ext = strings.ToLower(ext)
	audioExts := map[string]bool{
		".ac3": true, ".eac3": true, ".dts": true, ".flac": true, ".aac": true,
		".m4a": true, ".mp3": true, ".opus": true, ".ogg": true, ".wav": true,
		".mka": true,
	}
	return audioExts[ext]
}
