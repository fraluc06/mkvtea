package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// View renders the TUI display
func (m *ProcessModel) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	// Default width if not set
	width := m.width
	if width == 0 {
		width = 80
	}

	// === HEADER ===
	mode := strings.ToUpper(m.cfg.Mode)
	headerText := fmt.Sprintf("🍵 MKVTEA - %s", mode)
	header := titleStyle.Render(headerText)

	// === STATS BOX ===
	statsContent := fmt.Sprintf(
		"📦  %2d Total  │  ✅  %2d Success  │  ⏭️  %2d Skipped  │  ❌  %2d Failed",
		m.totalFiles, m.successCount, m.skippedCount, m.errorCount)
	statsBox := statsBoxStyle.Render(statsContent)

	// === PROGRESS BOX ===
	progressBar := m.renderProgressBar(width - 4)
	progressBox := boxStyle.Render(progressBar)

	// === STATUS ===
	var statusIcon, statusText string
	if m.finished {
		statusIcon = "✨"
		statusText = successStyle.Render("PROCESSING COMPLETE")
	} else {
		statusIcon = m.spinner.View()
		statusText = fmt.Sprintf("%s %s", subtitleStyle.Render("Processing files"), warningStyle.Render("..."))
	}
	statusLine := fmt.Sprintf("%s %s", statusIcon, statusText)

	// === LOGS BOX ===
	logsHeader := subtitleStyle.Render("📋 Processing Log")
	logsView := m.viewport.View()
	logsBox := boxStyle.Render(fmt.Sprintf("%s\n%s", logsHeader, logsView))

	// === FOOTER ===
	var footerText string
	if m.finished {
		remaining := time.Until(m.autoCloseTime)
		if remaining < 0 {
			remaining = 0
		}
		secondsLeft := int(remaining.Seconds()) + 1
		footerText = warningStyle.Render(fmt.Sprintf("🔄 Window closes in %d second(s) | Press Q or Ctrl+C to exit now", secondsLeft))
	} else {
		footerText = warningStyle.Render("⚡ Processing (Ctrl+C to cancel)")
	}

	// === SEPARATOR ===
	separatorWidth := width - 2
	if separatorWidth < 10 {
		separatorWidth = 10
	}
	separator := strings.Repeat("─", separatorWidth) // Horizontal line separator

	// === COMBINE WITH LAYOUT ===
	content := fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s",
		header,
		statsBox,
		progressBox,
		statusLine,
		logsBox,
		separator,
		footerText,
	)

	v := tea.NewView(docStyle.Render(content))
	v.AltScreen = true
	return v
}
