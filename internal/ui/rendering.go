package ui

import (
	"fmt"
	"strings"
)

// logPrefixes are the status prefixes used by log lines, in match order.
// Note: prefixes contain multi-byte emoji, so byte-based slicing must not be
// used to detect or strip them.
var logPrefixes = []string{"✅ SUCCESS: ", "⏭️  SKIPPED: ", "❌ FAILED: "}

// renderLogs renders the log entries, truncating filenames to fit the viewport
func (m *ProcessModel) renderLogs() string {
	availableWidth := m.viewport.Width()
	if availableWidth <= 20 {
		availableWidth = 80 // fallback
	}

	truncatedLogs := make([]string, 0, len(m.logs))
	for _, logLine := range m.logs {
		// Log format examples:
		// ✅ SUCCESS: filename.mkv
		// ⏭️  SKIPPED: filename.mkv
		// ❌ FAILED: filename.mkv - error message
		line := logLine
		for _, prefix := range logPrefixes {
			content, found := strings.CutPrefix(logLine, prefix)
			if !found {
				continue
			}

			// Calculate max length for content (reserve space for prefix and buffer)
			maxContentLen := availableWidth - len([]rune(prefix)) - 2
			if maxContentLen < 10 {
				maxContentLen = 10
			}

			// Truncate on rune boundaries to avoid splitting multi-byte characters
			if runes := []rune(content); len(runes) > maxContentLen {
				content = string(runes[:maxContentLen-3]) + "..."
			}

			line = prefix + content
			break
		}
		truncatedLogs = append(truncatedLogs, line)
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
	filled := min(int(percent*float64(barWidth)), barWidth)

	// Progress bar with block characters: ████░░░░░░
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	percentStr := fmt.Sprintf(" %3.0f%% [%2d/%2d]", percent*100, m.processedIdx, m.totalFiles)

	return progressStyle.Render(bar + percentStr)
}
