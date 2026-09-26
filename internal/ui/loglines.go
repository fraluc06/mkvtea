package ui

import (
	"fmt"
	"path/filepath"

	"mkvtea/internal/encode"
)

// The Processing Log keeps one line per file: 🔄 STARTED appears when a
// worker slot is acquired, 🔄 ENCODING rewrites it live in encode mode, and
// the final ✅/⏭️/❌ status replaces it in place — results never grow a
// second line. Indexes live in m.activeLogs; every method here requires m.mu
// to be held unless stated otherwise.

// progressCallback adapts live encoder updates into in-place rewrites of the
// file's log line; updates arriving after completion are dropped.
func (m *ProcessModel) progressCallback(file string) encode.ProgressFunc {
	return func(p encode.Progress) {
		m.mu.Lock()
		defer m.mu.Unlock()
		idx, ok := m.activeLogs[file]
		if !ok {
			return
		}
		m.replaceLogLocked(idx, encodeStatusLine(filepath.Base(file), p))
	}
}

// encodeStatusLine builds the live 🔄 ENCODING line. The status leads and the
// filename trails so renderLogs truncation keeps the interesting part. A
// metadata-estimated total is shown as an approximate percent.
func encodeStatusLine(filename string, p encode.Progress) string {
	if p.Total > 0 {
		approx := ""
		if p.Estimated {
			approx = "≈"
		}
		percent := 100 * float64(p.Frames) / float64(p.Total)
		return fmt.Sprintf("🔄 ENCODING: %s%.1f%% @ %.0f fps — %s", approx, percent, p.FPS, filename)
	}
	return fmt.Sprintf("🔄 ENCODING: %d frames @ %.0f fps — %s", p.Frames, p.FPS, filename)
}

// appendLogLocked appends a log line and refreshes the viewport; it returns
// the line index.
func (m *ProcessModel) appendLogLocked(line string) int {
	m.logs = append(m.logs, line)
	m.refreshLogsLocked()
	return len(m.logs) - 1
}

// replaceLogLocked rewrites an existing log line (bounds-checked) and
// refreshes the viewport.
func (m *ProcessModel) replaceLogLocked(idx int, line string) {
	if idx < 0 || idx >= len(m.logs) {
		return
	}
	m.logs[idx] = line
	m.refreshLogsLocked()
}

// refreshLogsLocked re-renders the log viewport; truncation happens in
// renderLogs based on the available width.
func (m *ProcessModel) refreshLogsLocked() {
	m.viewport.SetContent(m.renderLogs())
	m.viewport.GotoBottom()
}
