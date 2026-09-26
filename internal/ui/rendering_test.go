package ui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"charm.land/bubbles/v2/viewport"

	"mkvtea/internal/encode"
)

func TestRenderLogsTruncatesEveryPrefix(t *testing.T) {
	const width = 40
	long := strings.Repeat("x", 100)

	tests := []struct {
		name string
		line string
	}{
		{"success", "✅ SUCCESS: " + long},
		{"skipped", "⏭️  SKIPPED: " + long},
		{"failed", "❌ FAILED: " + long},
		{"encoding", "🔄 ENCODING: " + long},
		{"started", "🔄 STARTED: " + long},
	}

	m := &ProcessModel{viewport: viewport.New(viewport.WithWidth(width), viewport.WithHeight(3))}
	m.logs = make([]string, 0, len(tests))
	for _, tt := range tests {
		m.logs = append(m.logs, tt.line)
	}

	lines := strings.Split(m.renderLogs(), "\n")
	if len(lines) != len(tests) {
		t.Fatalf("rendered %d lines, want %d", len(lines), len(tests))
	}
	for i, tt := range tests {
		got := lines[i]
		if utf8.RuneCountInString(got) > width {
			t.Errorf("%s: rendered %d runes exceeds width %d: %q", tt.name, utf8.RuneCountInString(got), width, got)
		}
		wantPrefix, _, _ := strings.Cut(tt.line, " ")
		if !strings.HasPrefix(got, wantPrefix) {
			t.Errorf("%s: lost prefix %q in %q", tt.name, wantPrefix, got)
		}
		if !strings.HasSuffix(got, "...") {
			t.Errorf("%s: not truncated: %q", tt.name, got)
		}
	}
}

func TestRenderLogsKeepsEncodingStatusWhenTruncated(t *testing.T) {
	longName := strings.Repeat("Magilumiere", 20) + ".mkv"
	m := &ProcessModel{viewport: viewport.New(viewport.WithWidth(40), viewport.WithHeight(3))}
	m.logs = []string{encodeStatusLine(longName, encode.Progress{Frames: 1008, Total: 2400, FPS: 1234})}

	got := m.renderLogs()
	if !strings.Contains(got, "42.0%") {
		t.Errorf("percent lost to truncation: %q", got)
	}
}
