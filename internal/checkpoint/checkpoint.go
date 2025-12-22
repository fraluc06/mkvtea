package checkpoint

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mkvtea/internal/config"
)

// ProcessedFile tracks a file that has been processed
type ProcessedFile struct {
	Name string `json:"name"`
	Hash string `json:"hash"` // MD5 hash of filename for reliable matching
}

// FailedFile tracks a file that failed processing
type FailedFile struct {
	Name  string `json:"name"`
	Error string `json:"error"`
	Hash  string `json:"hash"`
}

// SkippedFile tracks a file that was skipped
type SkippedFile struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
	Hash   string `json:"hash"`
}

// ProcessedFiles groups processed, failed, and skipped files
type ProcessedFiles struct {
	Successful []ProcessedFile `json:"successful"`
	Failed     []FailedFile    `json:"failed"`
	Skipped    []SkippedFile   `json:"skipped"`
}

// Checkpoint represents a saved checkpoint state
type Checkpoint struct {
	Mode              string         `json:"mode"`
	Languages         []string       `json:"languages"`
	Directory         string         `json:"directory"`
	Recursive         bool           `json:"recursive"`
	StartedAt         time.Time      `json:"started_at"`
	LastCheckpoint    time.Time      `json:"last_checkpoint"`
	TotalFiles        int            `json:"total_files"`
	Processed         ProcessedFiles `json:"processed"`
	CheckpointVersion string         `json:"checkpoint_version"` // For future compatibility
}

// Manager handles checkpoint operations
type Manager struct {
	checkpointFile string
	checkpoint     *Checkpoint
}

// NewManager creates a new checkpoint manager
func NewManager(cfg config.Config) (*Manager, error) {
	// Determine checkpoint file location
	checkpointFile := filepath.Join(cfg.Dir, ".mkvtea_checkpoint.json")

	return &Manager{
		checkpointFile: checkpointFile,
		checkpoint:     nil,
	}, nil
}

// Load loads an existing checkpoint
func (m *Manager) Load() (*Checkpoint, error) {
	data, err := os.ReadFile(m.checkpointFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No checkpoint exists
		}
		return nil, fmt.Errorf("failed to read checkpoint: %w", err)
	}

	var cp Checkpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, fmt.Errorf("failed to parse checkpoint: %w", err)
	}

	m.checkpoint = &cp
	return &cp, nil
}

// Create creates a new checkpoint
func (m *Manager) Create(cfg config.Config, totalFiles int) error {
	languages := cfg.Languages
	if len(languages) == 0 && cfg.Lang != "" {
		languages = []string{cfg.Lang}
	}

	m.checkpoint = &Checkpoint{
		Mode:              cfg.Mode,
		Languages:         languages,
		Directory:         cfg.Dir,
		Recursive:         cfg.Recursive,
		StartedAt:         time.Now(),
		LastCheckpoint:    time.Now(),
		TotalFiles:        totalFiles,
		Processed:         ProcessedFiles{},
		CheckpointVersion: "1.0",
	}

	return m.Save()
}

// AddSuccess marks a file as successfully processed
func (m *Manager) AddSuccess(filename string) error {
	if m.checkpoint == nil {
		return fmt.Errorf("no active checkpoint")
	}

	pf := ProcessedFile{
		Name: filepath.Base(filename),
		Hash: hashFilename(filename),
	}

	m.checkpoint.Processed.Successful = append(m.checkpoint.Processed.Successful, pf)
	return nil
}

// AddFailed marks a file as failed
func (m *Manager) AddFailed(filename, errMsg string) error {
	if m.checkpoint == nil {
		return fmt.Errorf("no active checkpoint")
	}

	ff := FailedFile{
		Name:  filepath.Base(filename),
		Error: errMsg,
		Hash:  hashFilename(filename),
	}

	m.checkpoint.Processed.Failed = append(m.checkpoint.Processed.Failed, ff)
	return nil
}

// AddSkipped marks a file as skipped
func (m *Manager) AddSkipped(filename, reason string) error {
	if m.checkpoint == nil {
		return fmt.Errorf("no active checkpoint")
	}

	sf := SkippedFile{
		Name:   filepath.Base(filename),
		Reason: reason,
		Hash:   hashFilename(filename),
	}

	m.checkpoint.Processed.Skipped = append(m.checkpoint.Processed.Skipped, sf)
	return nil
}

// Save persists the checkpoint to disk
func (m *Manager) Save() error {
	if m.checkpoint == nil {
		return fmt.Errorf("no active checkpoint to save")
	}

	m.checkpoint.LastCheckpoint = time.Now()

	data, err := json.MarshalIndent(m.checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	// Write to temp file first, then atomic move
	tmpFile := m.checkpointFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write checkpoint: %w", err)
	}

	if err := os.Rename(tmpFile, m.checkpointFile); err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
}

// IsProcessed checks if a file has been processed (by hash or filename)
func (m *Manager) IsProcessed(filename string) bool {
	if m.checkpoint == nil {
		return false
	}

	hash := hashFilename(filename)
	base := filepath.Base(filename)

	// Check successful
	for _, f := range m.checkpoint.Processed.Successful {
		if f.Name == base || f.Hash == hash {
			return true
		}
	}

	// Check failed
	for _, f := range m.checkpoint.Processed.Failed {
		if f.Name == base || f.Hash == hash {
			return true
		}
	}

	// Check skipped
	for _, f := range m.checkpoint.Processed.Skipped {
		if f.Name == base || f.Hash == hash {
			return true
		}
	}

	return false
}

// GetProcessedCount returns count of all processed files
func (m *Manager) GetProcessedCount() int {
	if m.checkpoint == nil {
		return 0
	}

	return len(m.checkpoint.Processed.Successful) +
		len(m.checkpoint.Processed.Failed) +
		len(m.checkpoint.Processed.Skipped)
}

// GetStats returns processing statistics
func (m *Manager) GetStats() (successful, failed, skipped int) {
	if m.checkpoint == nil {
		return 0, 0, 0
	}

	return len(m.checkpoint.Processed.Successful),
		len(m.checkpoint.Processed.Failed),
		len(m.checkpoint.Processed.Skipped)
}

// Clear deletes the checkpoint file
func (m *Manager) Clear() error {
	if err := os.Remove(m.checkpointFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}
	m.checkpoint = nil
	return nil
}

// hashFilename generates a simple hash of the filename for reliable matching
func hashFilename(filename string) string {
	h := md5.New()
	io.WriteString(h, strings.ToLower(filepath.Base(filename)))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// CanResume checks if there's a resumable checkpoint
func CanResume(cfg config.Config) (bool, error) {
	manager, err := NewManager(cfg)
	if err != nil {
		return false, err
	}

	cp, err := manager.Load()
	if err != nil {
		return false, err
	}

	if cp == nil {
		return false, nil
	}

	// Check if checkpoint matches current config
	languages := cfg.Languages
	if len(languages) == 0 && cfg.Lang != "" {
		languages = []string{cfg.Lang}
	}

	if cp.Mode != cfg.Mode || cp.Recursive != cfg.Recursive {
		return false, nil
	}

	// Languages must match (order doesn't matter)
	if !slicesEqual(cp.Languages, languages) {
		return false, nil
	}

	return true, nil
}

// GetResumeStats returns stats for resume prompt
func GetResumeStats(cfg config.Config) (processed, remaining, total int, err error) {
	manager, err := NewManager(cfg)
	if err != nil {
		return 0, 0, 0, err
	}

	cp, loadErr := manager.Load()
	if loadErr != nil {
		return 0, 0, 0, loadErr
	}

	if cp == nil {
		return 0, 0, 0, nil
	}

	successCount := len(cp.Processed.Successful)
	failCount := len(cp.Processed.Failed)
	skipCount := len(cp.Processed.Skipped)
	processedCount := successCount + failCount + skipCount

	remainingCount := cp.TotalFiles - processedCount

	return processedCount, remainingCount, cp.TotalFiles, nil
}

// FilterProcessedFiles removes already-processed files from the list
func FilterProcessedFiles(manager *Manager, files []string) []string {
	if manager.checkpoint == nil {
		return files
	}

	filtered := make([]string, 0, len(files))
	for _, file := range files {
		if !manager.IsProcessed(file) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

// slicesEqual checks if two string slices contain the same elements (order-independent)
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	seen := make(map[string]bool)
	for _, v := range a {
		seen[v] = true
	}

	for _, v := range b {
		if !seen[v] {
			return false
		}
	}

	return true
}
