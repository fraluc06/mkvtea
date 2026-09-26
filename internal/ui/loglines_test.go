package ui

import (
	"testing"

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
			name: "with encoder total shows percent and fps first",
			p:    encode.Progress{Frames: 1008, Total: 2400, FPS: 1234.56},
			want: "🔄 ENCODING: 42.0% @ 1235 fps — ep.mkv",
		},
		{
			name: "metadata estimate is marked approximate",
			p:    encode.Progress{Frames: 100, Total: 33424, Estimated: true, FPS: 22.51},
			want: "🔄 ENCODING: ≈0.3% @ 23 fps — ep.mkv",
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

func TestProgressCallbackRewritesActiveLine(t *testing.T) {
	m := &ProcessModel{
		viewport:   viewport.New(viewport.WithWidth(80), viewport.WithHeight(3)),
		activeLogs: map[string]int{"/series/ep.mkv": 0},
	}
	m.logs = []string{"🔄 STARTED: ep.mkv"}

	m.progressCallback("/series/ep.mkv")(encode.Progress{Frames: 10, Total: 100, FPS: 50})

	if m.logs[0] != "🔄 ENCODING: 10.0% @ 50 fps — ep.mkv" {
		t.Errorf("line not rewritten in place: %q", m.logs[0])
	}

	// Updates for finished (or unknown) files are dropped, not applied.
	delete(m.activeLogs, "/series/ep.mkv")
	m.progressCallback("/series/ep.mkv")(encode.Progress{Frames: 99, Total: 100, FPS: 50})
	if m.logs[0] != "🔄 ENCODING: 10.0% @ 50 fps — ep.mkv" {
		t.Errorf("stale update mutated finished file's line: %q", m.logs[0])
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
