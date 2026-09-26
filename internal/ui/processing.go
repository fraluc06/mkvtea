package ui

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"

	"mkvtea/internal/encode"
	"mkvtea/internal/mkv"
)

// autoCloseDelay is how long the TUI stays open after processing completes.
// Shared by the countdown display and the close timer so they cannot drift apart.
const autoCloseDelay = 10 * time.Second

// startAutoClose returns a command that closes the TUI after autoCloseDelay
func (m *ProcessModel) startAutoClose() tea.Cmd {
	return tea.Tick(autoCloseDelay, func(time.Time) tea.Msg {
		return AutoCloseMsg{}
	})
}

// startProcessing returns a command that begins file processing
func (m *ProcessModel) startProcessing() tea.Cmd {
	return func() tea.Msg {
		// Initialize checkpoint if enabled
		if m.cfg.CheckpointInterval > 0 && m.checkpointMgr != nil {
			if err := m.checkpointMgr.Create(m.cfg, m.totalFiles); err != nil {
				m.logCheckpointWarning("failed to create checkpoint: %v", err)
			}
		}

		// Start processing all files
		for _, file := range m.files {
			m.wg.Add(1)
			go m.processFile(file)
		}

		// Wait for all to complete
		m.wg.Wait()
		return ProcessingDoneMsg{}
	}
}

// processFile processes a single MKV file and updates progress
func (m *ProcessModel) processFile(file string) {
	defer m.wg.Done()

	m.sem <- struct{}{}        // Acquire token
	defer func() { <-m.sem }() // Release token

	filename := filepath.Base(file)

	// Announce the file the moment it gets a worker slot; the line is
	// rewritten in place as the file progresses and finishes.
	m.mu.Lock()
	m.activeLogs[file] = m.appendLogLocked("🔄 STARTED: " + filename)
	m.mu.Unlock()

	var err error
	switch m.cfg.Mode {
	case "extract":
		err = mkv.RunExtract(file, m.cfg)
	case "encode":
		err = encode.RunEncode(file, m.cfg, m.progressCallback(file))
	default:
		err = mkv.RunMerge(file, m.cfg)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var logLine string

	if err != nil {
		if errors.Is(err, mkv.ErrSkipped) {
			logLine = fmt.Sprintf("⏭️  SKIPPED: %s", filename)
			m.skippedCount++
			reason := "no assets found"
			if m.cfg.Mode == "encode" {
				reason = "output already exists"
			}
			if m.cfg.CheckpointInterval > 0 && m.checkpointMgr != nil {
				if addErr := m.checkpointMgr.AddSkipped(file, reason); addErr != nil {
					m.logCheckpointWarningLocked("failed to record skipped file %s: %v", filename, addErr)
				}
			}
		} else {
			logLine = fmt.Sprintf("❌ FAILED: %s - %v", filename, err)
			m.errorCount++
			if m.cfg.CheckpointInterval > 0 && m.checkpointMgr != nil {
				if addErr := m.checkpointMgr.AddFailed(file, err.Error()); addErr != nil {
					m.logCheckpointWarningLocked("failed to record failed file %s: %v", filename, addErr)
				}
			}
		}
	} else {
		logLine = fmt.Sprintf("✅ SUCCESS: %s", filename)
		m.successCount++
		if m.cfg.CheckpointInterval > 0 && m.checkpointMgr != nil {
			if addErr := m.checkpointMgr.AddSuccess(file); addErr != nil {
				m.logCheckpointWarningLocked("failed to record successful file %s: %v", filename, addErr)
			}
		}

		// Track output paths for DRY-RUN summary
		switch m.cfg.Mode {
		case "extract":
			lang := m.cfg.Lang
			if len(m.cfg.Languages) > 0 {
				lang = m.cfg.Languages[0]
			}
			subsDir := filepath.Join(filepath.Dir(file), "subs", lang)
			if !slices.Contains(m.extractedPaths, subsDir) {
				m.extractedPaths = append(m.extractedPaths, subsDir)
			}
		case "merge":
			outRoot := m.cfg.OutDir
			if outRoot == "" {
				lang := m.cfg.Lang
				if len(m.cfg.Languages) > 0 {
					lang = m.cfg.Languages[0]
				}
				outRoot = filepath.Join(filepath.Dir(m.cfg.Dir), filepath.Base(m.cfg.Dir)+"_"+lang)
			}
			m.outputDir = outRoot
		case "encode":
			subdir := m.cfg.OutSubdir
			if subdir == "" {
				subdir = encode.DefaultOutSubdir
			}
			outRoot := m.cfg.OutDir
			if outRoot == "" {
				outRoot = filepath.Join(filepath.Dir(file), subdir)
			}
			if !slices.Contains(m.extractedPaths, outRoot) {
				m.extractedPaths = append(m.extractedPaths, outRoot)
			}
		}
	}

	// Final status replaces this file's 🔄 STARTED/ENCODING line in place.
	if idx, ok := m.activeLogs[file]; ok {
		m.replaceLogLocked(idx, logLine)
		delete(m.activeLogs, file)
	} else {
		// Defensive: the line should exist; never drop a result.
		m.appendLogLocked(logLine)
	}
	m.processedIdx++

	// Save checkpoint at intervals
	if m.cfg.CheckpointInterval > 0 && m.checkpointMgr != nil {
		m.checkpointCounter++
		if m.checkpointCounter >= m.cfg.CheckpointInterval {
			if saveErr := m.checkpointMgr.Save(); saveErr != nil {
				m.logCheckpointWarningLocked("failed to save checkpoint: %v", saveErr)
			} else {
				m.checkpointCounter = 0
			}
		}
	}

	// Refresh once more so checkpoint warnings appended above are visible.
	m.refreshLogsLocked()
}
