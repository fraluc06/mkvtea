package ui

import (
	"strings"
	"testing"
	"unicode/utf8"

	"charm.land/bubbles/v2/viewport"

	"mkvtea/internal/encode"
)

func TestEncodeStatusLine(t *testing.T) {
	tests := []struct {
		name string
		p    encode.Progress
		want string
	}{
		{
			name: "with total shows percent and fps first",
			p:    encode.Progress{Frames: 1008, Total: 2400, FPS: 1234.56},
			want: "🔄 ENCODING: 42.0% @ 1235 fps — ep.mkv",
		},
		{
			name: "unknown total falls back to frame count",
			p:    encode.Progress{Frames: 156, Total: 0, FPS: 1123.85},
			want: "🔄 ENCODING: 156 frames @ 1124 fps — ep.mkv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encodeStatusLine("ep.mkv", tt.p); got != tt.want {
				t.Errorf("encodeStatusLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

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

func TestReplaceLogLockedInPlace(t *testing.T) {
	m := &ProcessModel{viewport: viewport.New(viewport.WithWidth(80), viewport.WithHeight(3))}
	idx := m.appendLogLocked("🔄 STARTED: a.mkv")
	m.appendLogLocked("🔄 STARTED: b.mkv")
	m.replaceLogLocked(idx, "✅ SUCCESS: a.mkv")

	if len(m.logs) != 2 {
		t.Fatalf("logs grew to %d entries, want the line replaced in place", len(m.logs))
	}
	if m.logs[0] != "✅ SUCCESS: a.mkv" || m.logs[1] != "🔄 STARTED: b.mkv" {
		t.Errorf("logs = %q, want final status at idx 0 and untouched b", m.logs)
	}

	// Out-of-range indices are ignored, not panics.
	m.replaceLogLocked(len(m.logs), "✅ SUCCESS: c.mkv")
	if len(m.logs) != 2 {
		t.Errorf("out-of-range replace mutated logs: %q", m.logs)
	}
}
