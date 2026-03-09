package ui

import (
	"fmt"
	"mkvtea/internal/mkv"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
)

// startAutoClose returns a command that closes the TUI after 5 seconds
func (m *ProcessModel) startAutoClose() tea.Cmd {
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
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

	var err error
	if m.cfg.Mode == "extract" {
		err = mkv.RunExtract(file, m.cfg)
	} else {
		err = mkv.RunMerge(file, m.cfg)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	filename := filepath.Base(file)
	var logLine string

	if err != nil {
		if err.Error() == "skipped" {
			logLine = fmt.Sprintf("⏭️  SKIPPED: %s", filename)
			m.skippedCount++
			if m.cfg.CheckpointInterval > 0 && m.checkpointMgr != nil {
				if addErr := m.checkpointMgr.AddSkipped(file, "no subtitles"); addErr != nil {
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
		if m.cfg.Mode == "extract" {
			lang := m.cfg.Lang
			if len(m.cfg.Languages) > 0 {
				lang = m.cfg.Languages[0]
			}
			subsDir := filepath.Join(filepath.Dir(file), "subs", lang)
			if !contains(m.extractedPaths, subsDir) {
				m.extractedPaths = append(m.extractedPaths, subsDir)
			}
		} else if m.cfg.Mode == "merge" {
			outRoot := m.cfg.OutDir
			if outRoot == "" {
				lang := m.cfg.Lang
				if len(m.cfg.Languages) > 0 {
					lang = m.cfg.Languages[0]
				}
				outRoot = filepath.Join(filepath.Dir(m.cfg.Dir), filepath.Base(m.cfg.Dir)+"_"+lang)
			}
			m.outputDir = outRoot
		}
	}

	m.logs = append(m.logs, logLine)
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

	// Update viewport - truncate will happen in renderLogs based on available width
	m.viewport.SetContent(m.renderLogs())
	m.viewport.GotoBottom()
}
