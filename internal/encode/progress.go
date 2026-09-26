package encode

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Progress is one SvtAv1EncApp status update for a running encode.
type Progress struct {
	Frames int
	Total  int // 0 while the encoder has not established the frame count yet
	FPS    float64
}

// ProgressFunc receives encode progress updates. The engine reports plain
// values so internal/encode stays unaware of the TUI (dependency law).
type ProgressFunc func(Progress)

// progressEmitInterval is the minimum gap between progress callbacks; it
// keeps a fast encoder from flooding the TUI with redraws.
const progressEmitInterval = time.Second

// pendingSegmentCap bounds the unterminated-segment buffer if the encoder
// ever streams without \r/\n delimiters; losing one segment is acceptable.
const pendingSegmentCap = 4096

// ansiPattern matches the SGR color codes SvtAv1EncApp embeds in its stderr
// progress lines even when the output is not a terminal.
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// progressPattern matches one color-stripped progress segment:
// "Encoding:  157/240 Frames @ 1130.77 fps | ..." — the total is absent while
// the encoder does not know the input length yet ("Encoding:   2 Frames @ ...").
var progressPattern = regexp.MustCompile(`Encoding:\s+(\d+)(?:/(\d+))?\s+Frames\s+@\s+([\d.]+)\s+fps`)

// parseProgress extracts the structured progress from one stderr segment.
// Non-progress lines (banner, config dump, errors) report ok == false.
func parseProgress(segment string) (Progress, bool) {
	m := progressPattern.FindStringSubmatch(ansiPattern.ReplaceAllString(segment, ""))
	if m == nil {
		return Progress{}, false
	}
	frames, err := strconv.Atoi(m[1])
	if err != nil {
		return Progress{}, false
	}
	total := 0
	if m[2] != "" {
		total, err = strconv.Atoi(m[2])
		if err != nil {
			return Progress{}, false
		}
	}
	fps, err := strconv.ParseFloat(m[3], 64)
	if err != nil {
		return Progress{}, false
	}
	return Progress{Frames: frames, Total: total, FPS: fps}, true
}

// progressWriter tees SvtAv1EncApp stderr: every byte still lands in the tail
// buffer for failure diagnostics, while \r/\n-delimited segments are parsed
// into progress updates for onProgress (nil disables reporting).
type progressWriter struct {
	tail        *tailBuffer
	onProgress  ProgressFunc
	minInterval time.Duration

	pending   []byte // trailing bytes of the unterminated segment
	lastFrame int    // last reported frame count, suppresses repeats
	lastEmit  time.Time
}

func (w *progressWriter) Write(p []byte) (int, error) {
	if _, err := w.tail.Write(p); err != nil {
		return 0, fmt.Errorf("failed to record encoder stderr: %w", err)
	}
	if w.onProgress == nil {
		return len(p), nil
	}

	w.pending = append(w.pending, p...)
	if len(w.pending) > pendingSegmentCap {
		w.pending = w.pending[len(w.pending)-pendingSegmentCap:]
	}

	for {
		i := bytes.IndexAny(w.pending, "\r\n")
		if i < 0 {
			break
		}
		w.emit(string(w.pending[:i]))
		w.pending = w.pending[i+1:]
	}
	return len(p), nil
}

// emit parses one segment and forwards it when the frame count advanced and
// the throttle window elapsed.
func (w *progressWriter) emit(segment string) {
	progress, ok := parseProgress(segment)
	if !ok || progress.Frames <= w.lastFrame {
		return
	}
	now := time.Now()
	if !w.lastEmit.IsZero() && now.Sub(w.lastEmit) < w.minInterval {
		return
	}
	w.lastEmit = now
	w.lastFrame = progress.Frames
	w.onProgress(progress)
}
