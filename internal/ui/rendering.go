package ui

import (
	"fmt"
	"strings"
)

// renderLogs renders the log entries, truncating filenames to fit the viewport
func (m *ProcessModel) renderLogs() string {
	availableWidth := m.viewport.Width
	if availableWidth <= 20 {
		availableWidth = 80 // fallback
	}

	var truncatedLogs []string
	for _, logLine := range m.logs {
		// Log format examples:
		// ✅ SUCCESS: filename.mkv
		// ⏭️  SKIPPED: filename.mkv
		// ❌ FAILED: filename.mkv - error message

		// Extract prefix and content
		var prefix, content string
		if len(logLine) > 11 && logLine[:11] == "✅ SUCCESS:" {
			prefix = "✅ SUCCESS: "
			content = logLine[11:]
		} else if len(logLine) > 12 && logLine[:12] == "⏭️  SKIPPED:" {
			prefix = "⏭️  SKIPPED: "
			content = logLine[12:]
		} else if len(logLine) > 10 && logLine[:10] == "❌ FAILED:" {
			prefix = "❌ FAILED: "
			content = logLine[10:]
		} else {
			truncatedLogs = append(truncatedLogs, logLine)
			continue
		}

		// Calculate max length for content (reserve space for prefix and buffer)
		maxContentLen := availableWidth - len(prefix) - 2

		if maxContentLen < 10 {
			maxContentLen = 10
		}

		// Truncate if needed
		if len(content) > maxContentLen {
			content = content[:maxContentLen-3] + "..."
		}

		truncatedLogs = append(truncatedLogs, prefix+content)
	}

	return strings.Join(truncatedLogs, "\n")
}

// renderProgressBar renders a progress bar with percentage and counter
func (m *ProcessModel) renderProgressBar(maxWidth int) string {
	// Reserve space for percentage and counter text
	progressTextLen := len(fmt.Sprintf(" 100%% [%2d/%2d]", m.totalFiles, m.totalFiles))
	barWidth := maxWidth - progressTextLen

	if barWidth < 10 {
		barWidth = 10
	}

	percent := float64(m.processedIdx) / float64(m.totalFiles)
	filled := int(percent * float64(barWidth))

	// Progress bar with block characters: ████░░░░░░
	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}

	percentStr := fmt.Sprintf(" %3.0f%% [%2d/%2d]", percent*100, m.processedIdx, m.totalFiles)

	return progressStyle.Render(bar + percentStr)
}
