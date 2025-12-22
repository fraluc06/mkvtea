package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// isUnderDirectory checks if a file is under a given directory
func isUnderDirectory(file, dir string) bool {
	absFile, err := filepath.Abs(file)
	if err != nil {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absDir, absFile)
	if err != nil {
		return false
	}
	return !filepath.IsAbs(rel) && rel != ".."
}

func TestScanFilesNonRecursive(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()

	// Create some test files
	testFiles := []string{
		"video1.mkv",
		"video2.mkv",
		"document.txt",
		"image.jpg",
		"video3.MKV", // Test case-insensitive extension
	}

	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Test non-recursive scan
	found := ScanFiles(tmpDir, false)

	// Should find exactly 3 .mkv files (case-insensitive)
	if len(found) != 3 {
		t.Errorf("Expected 3 MKV files, found %d", len(found))
	}

	// Verify all found files are in temp directory and end with .mkv
	for _, f := range found {
		if !isUnderDirectory(f, tmpDir) {
			t.Errorf("File %s is not in temp directory %s", f, tmpDir)
		}
	}
}

func TestScanFilesRecursive(t *testing.T) {
	// Create temporary nested directory structure
	tmpDir := t.TempDir()

	// Create subdirectories
	subdir1 := filepath.Join(tmpDir, "season1")
	subdir2 := filepath.Join(tmpDir, "season2")
	subdir3 := filepath.Join(tmpDir, "season2", "extras")

	for _, dir := range []string{subdir1, subdir2, subdir3} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}
	}

	// Create test files in various locations
	testFiles := map[string][]string{
		tmpDir:  {"episode0.mkv", "readme.txt"},
		subdir1: {"episode1.mkv", "episode2.mkv"},
		subdir2: {"episode3.mkv", "image.jpg"},
		subdir3: {"episode4.mkv"},
	}

	for dir, files := range testFiles {
		for _, f := range files {
			path := filepath.Join(dir, f)
			if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}
		}
	}

	// Test recursive scan
	found := ScanFiles(tmpDir, true)

	// Should find exactly 5 .mkv files
	if len(found) != 5 {
		t.Errorf("Expected 5 MKV files in recursive scan, found %d", len(found))
	}
}

func TestScanFilesEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Test empty directory
	found := ScanFiles(tmpDir, false)
	if len(found) != 0 {
		t.Errorf("Expected 0 files in empty directory, found %d", len(found))
	}

	// Test empty directory with recursive
	found = ScanFiles(tmpDir, true)
	if len(found) != 0 {
		t.Errorf("Expected 0 files in empty directory (recursive), found %d", len(found))
	}
}

func TestScanFilesNonExistentDirectory(t *testing.T) {
	// Test non-existent directory
	found := ScanFiles("/nonexistent/directory", false)
	if len(found) != 0 {
		t.Errorf("Expected 0 files for non-existent directory, found %d", len(found))
	}
}

func TestScanFilesCaseSensitivity(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with different case extensions but different names
	// (on case-insensitive filesystems like macOS/Windows, same name with different case is same file)
	testFiles := []string{
		"video1.mkv",
		"video2.MKV",
		"video3.Mkv",
		"video4.mKv",
	}

	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Should find all 4 files regardless of extension case
	found := ScanFiles(tmpDir, false)
	if len(found) != 4 {
		t.Errorf("Expected 4 MKV files (case-insensitive extensions), found %d", len(found))
	}
}
