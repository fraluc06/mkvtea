package cmd

import (
	"os"
	"path/filepath"
	"strings"
)

// ScanFiles finds all .mkv files in the given directory, optionally recursive
func ScanFiles(dir string, recursive bool) []string {
	var files []string

	if recursive {
		filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.EqualFold(filepath.Ext(d.Name()), ".mkv") {
				files = append(files, path)
			}
			return nil
		})
	} else {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return files
		}
		for _, e := range entries {
			if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".mkv") {
				files = append(files, filepath.Join(dir, e.Name()))
			}
		}
	}

	return files
}
