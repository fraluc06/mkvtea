package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"mkvtea/internal/config"
	"mkvtea/internal/encode"
)

// encodeExtensions lists the containers encode mode accepts on top of mkv/mp4
// (mkvmerge reads them all; output is always .mkv).
var encodeExtensions = map[string]bool{
	".mov": true, ".avi": true, ".m2ts": true, ".ts": true, ".webm": true,
}

func isVideoFile(filename, mode string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".mkv" || ext == ".mp4" {
		return true
	}
	return mode == "encode" && encodeExtensions[ext]
}

// ScanFiles finds all video files in the configured directory, or a single
// file if cfg.Dir is one. In encode mode it accepts more containers and
// recursive walks skip encoder output subdirs (no re-encoding previous runs).
func ScanFiles(cfg config.Config) []string {
	var files []string
	path := cfg.Dir

	info, err := os.Stat(path)
	if err != nil {
		return nil
	}

	// If it's a single file
	if !info.IsDir() {
		if isVideoFile(path, cfg.Mode) {
			return []string{path}
		}
		return nil
	}

	// Recursive walks in encode mode must not descend into output subdirs.
	skipDir := ""
	if cfg.Mode == "encode" {
		skipDir = cfg.OutSubdir
		if skipDir == "" {
			skipDir = encode.DefaultOutSubdir
		}
	}

	if cfg.Recursive {
		err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if skipDir != "" && d.Name() == skipDir && p != path {
					return filepath.SkipDir
				}
				return nil
			}
			if isVideoFile(d.Name(), cfg.Mode) {
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return nil
		}
		return files
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return files
	}
	for _, e := range entries {
		if !e.IsDir() && isVideoFile(e.Name(), cfg.Mode) {
			files = append(files, filepath.Join(path, e.Name()))
		}
	}
	return files
}
