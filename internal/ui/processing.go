package ui

import (
	"fmt"
	"mkvtea/internal/mkv"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
		if m.cfg.CheckpointInterval > 0 {
			m.checkpointMgr.Create(m.cfg, m.totalFiles)
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

	// Add delay in dry-run mode for UI readability
	if m.cfg.DryRun {
		time.Sleep(500 * time.Millisecond)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	filename := filepath.Base(file)
	var logLine string

	if err != nil {
		if err.Error() == "skipped" {
			logLine = fmt.Sprintf("⏭️  SKIPPED: %s", filename)
			m.skippedCount++
			if m.cfg.CheckpointInterval > 0 {
				m.checkpointMgr.AddSkipped(file, "no subtitles")
			}
		} else {
			logLine = fmt.Sprintf("❌ FAILED: %s - %v", filename, err)
			m.errorCount++
			if m.cfg.CheckpointInterval > 0 {
				m.checkpointMgr.AddFailed(file, err.Error())
			}
		}
	} else {
		logLine = fmt.Sprintf("✅ SUCCESS: %s", filename)
		m.successCount++
		if m.cfg.CheckpointInterval > 0 {
			m.checkpointMgr.AddSuccess(file)
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
	if m.cfg.CheckpointInterval > 0 {
		m.checkpointCounter++
		if m.checkpointCounter >= m.cfg.CheckpointInterval {
			m.checkpointMgr.Save()
			m.checkpointCounter = 0
		}
	}

	// Update viewport - truncate will happen in renderLogs based on available width
	m.viewport.SetContent(m.renderLogs())
	m.viewport.GotoBottom()
}
