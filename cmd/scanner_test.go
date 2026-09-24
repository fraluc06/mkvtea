package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"mkvtea/internal/config"
)

func scanCfg(dir string, recursive bool, mode string) config.Config {
	return config.Config{Dir: dir, Recursive: recursive, Mode: mode}
}

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
	found := ScanFiles(scanCfg(tmpDir, false, "extract"))

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
	found := ScanFiles(scanCfg(tmpDir, true, "extract"))

	// Should find exactly 5 .mkv files
	if len(found) != 5 {
		t.Errorf("Expected 5 MKV files in recursive scan, found %d", len(found))
	}
}

func TestScanFilesEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Test empty directory
	found := ScanFiles(scanCfg(tmpDir, false, "extract"))
	if len(found) != 0 {
		t.Errorf("Expected 0 files in empty directory, found %d", len(found))
	}

	// Test empty directory with recursive
	found = ScanFiles(scanCfg(tmpDir, true, "extract"))
	if len(found) != 0 {
		t.Errorf("Expected 0 files in empty directory (recursive), found %d", len(found))
	}
}

func TestScanFilesNonExistentDirectory(t *testing.T) {
	// Test non-existent directory
	found := ScanFiles(scanCfg("/nonexistent/directory", false, "extract"))
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
	found := ScanFiles(scanCfg(tmpDir, false, "extract"))
	if len(found) != 4 {
		t.Errorf("Expected 4 MKV files (case-insensitive extensions), found %d", len(found))
	}
}

// TestScanFilesEncodeModeExtraExtensions verifies encode mode accepts the extra
// containers (mov/avi/m2ts/ts/webm) while extract mode still rejects them.
func TestScanFilesEncodeModeExtraExtensions(t *testing.T) {
	tmpDir := t.TempDir()

	files := []string{"a.mkv", "b.mp4", "c.mov", "d.avi", "e.m2ts", "f.ts", "g.webm", "h.txt"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, f), []byte("x"), 0644); err != nil {
			t.Fatalf("failed to create %s: %v", f, err)
		}
	}

	encodeFound := ScanFiles(scanCfg(tmpDir, false, "encode"))
	if len(encodeFound) != 7 {
		t.Errorf("encode mode: expected 7 video files, found %d", len(encodeFound))
	}

	extractFound := ScanFiles(scanCfg(tmpDir, false, "extract"))
	if len(extractFound) != 2 {
		t.Errorf("extract mode: expected 2 files (mkv/mp4 only), found %d", len(extractFound))
	}
}

// TestScanFilesEncodeSkipsOutSubdir verifies recursive encode walks skip the
// output subdir so previously encoded files are never re-encoded.
func TestScanFilesEncodeSkipsOutSubdir(t *testing.T) {
	tmpDir := t.TempDir()

	seriesDir := filepath.Join(tmpDir, "season1")
	outDir := filepath.Join(seriesDir, "av1")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	for _, f := range []string{
		filepath.Join(seriesDir, "ep01.mkv"), // source
		filepath.Join(outDir, "ep01.mkv"),    // prior output — must be skipped
	} {
		if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
			t.Fatalf("failed to create %s: %v", f, err)
		}
	}

	// encode mode: only the source, the av1/ copy is skipped
	encodeFound := ScanFiles(scanCfg(tmpDir, true, "encode"))
	if len(encodeFound) != 1 {
		t.Errorf("encode: expected 1 file (av1/ skipped), found %d: %v", len(encodeFound), encodeFound)
	}

	// extract mode: both are seen (no skip in non-encode modes)
	extractFound := ScanFiles(scanCfg(tmpDir, true, "extract"))
	if len(extractFound) != 2 {
		t.Errorf("extract: expected 2 files (no skip), found %d", len(extractFound))
	}
}
