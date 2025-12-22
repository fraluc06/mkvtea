package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"mkvtea/internal/config"
	"mkvtea/internal/mkv"
	"mkvtea/internal/ui"
)

// WatchConfig extends config with watch-specific options
type WatchConfig struct {
	Config config.Config
	OnAdd  func(path string) // Callback when file is added
}

// FileWatcher monitors a directory for new MKV files
type FileWatcher struct {
	path      string
	cfg       config.Config
	watcher   *fsnotify.Watcher
	done      chan struct{}
	processed map[string]time.Time // Track recently processed files to avoid duplicates
	mu        sync.Mutex
}

// NewFileWatcher creates a new file watcher
func NewFileWatcher(path string, cfg config.Config) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %v", err)
	}

	return &FileWatcher{
		path:      path,
		cfg:       cfg,
		watcher:   watcher,
		done:      make(chan struct{}),
		processed: make(map[string]time.Time),
	}, nil
}

// Start begins watching the directory
func (fw *FileWatcher) Start() error {
	// Add main directory
	if err := fw.watcher.Add(fw.path); err != nil {
		return fmt.Errorf("failed to watch directory %s: %v", fw.path, err)
	}

	// If recursive, add all subdirectories
	if fw.cfg.Recursive {
		if err := fw.addSubdirectories(fw.path); err != nil {
			return fmt.Errorf("failed to add subdirectories to watcher: %v", err)
		}
	}

	fmt.Printf("🔍 Watching directory: %s\n", fw.path)
	if fw.cfg.Recursive {
		fmt.Println("📁 Recursive mode enabled")
	}
	fmt.Printf("🎬 Mode: %s | 🗣️  Languages: %v\n", fw.cfg.Mode, fw.cfg.Languages)
	fmt.Println("⏰ Waiting for new MKV files... (Press Ctrl+C to exit)")

	go fw.handleEvents()
	return nil
}

// Stop stops watching the directory
func (fw *FileWatcher) Stop() error {
	close(fw.done)
	return fw.watcher.Close()
}

// addSubdirectories adds all subdirectories to the watcher
func (fw *FileWatcher) addSubdirectories(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && path != root {
			if err := fw.watcher.Add(path); err != nil {
				fmt.Printf("⚠️  Could not watch directory %s: %v\n", path, err)
			}
		}
		return nil
	})
}

// handleEvents processes file system events
func (fw *FileWatcher) handleEvents() {
	ticker := time.NewTicker(500 * time.Millisecond) // Debounce file writes
	defer ticker.Stop()

	fileBuffer := make(map[string]time.Time)
	fileBufferMu := sync.Mutex{}

	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			// Only process Create and Write events
			if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
				fileBufferMu.Lock()
				fileBuffer[event.Name] = time.Now()
				fileBufferMu.Unlock()
			}

			// Handle newly created directories in recursive mode
			if event.Op&fsnotify.Create == fsnotify.Create && fw.cfg.Recursive {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if err := fw.watcher.Add(event.Name); err != nil {
						fmt.Printf("⚠️  Could not watch new directory %s: %v\n", event.Name, err)
					}
				}
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("❌ Watcher error: %v\n", err)

		case <-ticker.C:
			// Process buffered files
			fileBufferMu.Lock()
			if len(fileBuffer) > 0 {
				filesToProcess := make([]string, 0)
				now := time.Now()

				for file, createdAt := range fileBuffer {
					// Wait 1 second to ensure file write is complete
					if now.Sub(createdAt) > 1*time.Second {
						if strings.EqualFold(filepath.Ext(file), ".mkv") {
							filesToProcess = append(filesToProcess, file)
						}
						delete(fileBuffer, file)
					}
				}

				fileBufferMu.Unlock()

				// Process collected files
				for _, file := range filesToProcess {
					fw.processFile(file)
				}
			} else {
				fileBufferMu.Unlock()
			}

		case <-fw.done:
			return
		}
	}
}

// processFile processes a single MKV file
func (fw *FileWatcher) processFile(path string) {
	// Check if already processed recently
	fw.mu.Lock()
	if lastProcessed, exists := fw.processed[path]; exists && time.Since(lastProcessed) < 30*time.Second {
		fw.mu.Unlock()
		return
	}
	fw.processed[path] = time.Now()
	fw.mu.Unlock()

	fmt.Printf("\n📥 New file detected: %s\n", filepath.Base(path))

	// Create config for this file
	fileCfg := fw.cfg
	fileCfg.Dir = filepath.Dir(path)

	// Scan just this one file
	files := []string{path}

	// Process using the UI processor
	fmt.Printf("🔄 Starting %s...\n", fileCfg.Mode)
	if err := ui.RunProcessTUI(fileCfg, files); err != nil {
		fmt.Printf("❌ Processing error: %v\n", err)
	} else {
		fmt.Printf("✅ Completed: %s\n", filepath.Base(path))
	}

	fmt.Println("⏰ Waiting for new MKV files... (Press Ctrl+C to exit)")
}

// WatchAndProcess starts watching a directory and processes files automatically
func WatchAndProcess(path string, cfg config.Config) error {
	// Validate dependencies first
	if err := mkv.ValidateDependencies(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Validate path exists
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("directory not found: %s", path)
	}

	watcher, err := NewFileWatcher(path, cfg)
	if err != nil {
		return err
	}

	if err := watcher.Start(); err != nil {
		return err
	}

	// Keep watching until interrupted
	<-watcher.done
	return nil
}
