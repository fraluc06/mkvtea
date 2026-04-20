package cmd

import (
	"os"
	"path/filepath"
	"strings"
)

func isVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".mkv" || ext == ".mp4"
}

// ScanFiles finds all .mkv or .mp4 files in the given directory or a single file if specified
func ScanFiles(path string, recursive bool) []string {
	var files []string

	info, err := os.Stat(path)
	if err != nil {
		return nil
	}

	// If it's a single file
	if !info.IsDir() {
		if isVideoFile(path) {
			return []string{path}
		}
		return nil
	}

	// If it's a directory
	if recursive {
		err := filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && isVideoFile(d.Name()) {
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return nil
		}
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return files
		}
		for _, e := range entries {
			if !e.IsDir() && isVideoFile(e.Name()) {
				files = append(files, filepath.Join(path, e.Name()))
			}
		}
	}

	return files
}
