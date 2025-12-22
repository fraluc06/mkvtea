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
		} else {
			logLine = fmt.Sprintf("❌ FAILED: %s - %v", filename, err)
			m.errorCount++
		}
	} else {
		logLine = fmt.Sprintf("✅ SUCCESS: %s", filename)
		m.successCount++

		// Track output paths for DRY-RUN summary
		if m.cfg.Mode == "extract" {
			subsDir := filepath.Join(filepath.Dir(file), "subs", m.cfg.Lang)
			if !contains(m.extractedPaths, subsDir) {
				m.extractedPaths = append(m.extractedPaths, subsDir)
			}
		} else if m.cfg.Mode == "merge" {
			outRoot := m.cfg.OutDir
			if outRoot == "" {
				outRoot = filepath.Join(filepath.Dir(m.cfg.Dir), filepath.Base(m.cfg.Dir)+"_"+m.cfg.Lang)
			}
			m.outputDir = outRoot
		}
	}

	m.logs = append(m.logs, logLine)
	m.processedIdx++

	// Update viewport - truncate will happen in renderLogs based on available width
	m.viewport.SetContent(m.renderLogs())
	m.viewport.GotoBottom()
}
