package encode

import (
	"strings"
	"testing"
	"time"
)

func TestParseProgress(t *testing.T) {
	tests := []struct {
		name    string
		segment string
		want    Progress
		wantOK  bool
	}{
		{
			name:    "colored segment with total",
			segment: "Encoding: \x1b[33m 157/240 Frames\x1b[0m @ \x1b[32m1130.77\x1b[0m fps | \x1b[35m14.22 kb/s\x1b[0m",
			want:    Progress{Frames: 157, Total: 240, FPS: 1130.77},
			wantOK:  true,
		},
		{
			name:    "total unknown while piping stdin",
			segment: "Encoding: \x1b[33m   2 Frames\x1b[0m @ \x1b[32m29.82\x1b[0m fps | Size: \x1b[31m0.00 MB\x1b[0m",
			want:    Progress{Frames: 2, Total: 0, FPS: 29.82},
			wantOK:  true,
		},
		{
			name:    "plain text without ANSI",
			segment: "Encoding:  240/240 Frames @ 1475.51 fps | Size: 0.02 MB",
			want:    Progress{Frames: 240, Total: 240, FPS: 1475.51},
			wantOK:  true,
		},
		{
			name:    "config banner is not progress",
			segment: "Svt[info]: SVT [config]: preset / tune / pred struct \t\t\t: 10 / PSNR (1) / RA",
			wantOK:  false,
		},
		{
			name:    "decode error line is not progress",
			segment: "Failed to read y4m frame delimeter. Read broken. EOF: 1",
			wantOK:  false,
		},
		{
			name:    "bare label without numbers is not progress",
			segment: "Encoding           Encoding:",
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseProgress(tt.segment)
			if ok != tt.wantOK {
				t.Fatalf("parseProgress(%q) ok = %v, want %v", tt.segment, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("parseProgress(%q) = %+v, want %+v", tt.segment, got, tt.want)
			}
		})
	}
}

func TestProgressWriterFeedsTailAndStreams(t *testing.T) {
	tail := &tailBuffer{max: 512}
	var got []Progress
	w := &progressWriter{
		tail:        tail,
		onProgress:  func(p Progress) { got = append(got, p) },
		minInterval: 0, // emit every advancing frame count; throttle tested separately
	}

	// Chunks split across segment boundaries, mixing \r and \n delimiters,
	// like the real encoder stream.
	writes := []string{
		"Svt[info]: banner\nEncoding: \x1b[33m   1 Fra",
		"mes\x1b[0m @ \x1b[32m18.49\x1b[0m fps | x\rEnc",
		"oding: \x1b[33m 2/2 Frames\x1b[0m @ \x1b[32m29.82\x1b[0m fps | y\n",
	}
	wantRaw := strings.Join(writes, "")
	for _, chunk := range writes {
		n, err := w.Write([]byte(chunk))
		if err != nil {
			t.Fatalf("write failed: %v", err)
		}
		if n != len(chunk) {
			t.Fatalf("Write returned n=%d, want %d", n, len(chunk))
		}
	}

	if len(got) != 2 {
		t.Fatalf("got %d progress updates (%+v), want 2", len(got), got)
	}
	if got[0] != (Progress{Frames: 1, Total: 0, FPS: 18.49}) {
		t.Errorf("first update = %+v, want {1 0 18.49}", got[0])
	}
	if got[1] != (Progress{Frames: 2, Total: 2, FPS: 29.82}) {
		t.Errorf("second update = %+v, want {2 2 29.82}", got[1])
	}
	if tail.String() != strings.TrimSpace(wantRaw) {
		t.Errorf("tail lost bytes: got %q, want %q", tail.String(), strings.TrimSpace(wantRaw))
	}
}

func TestProgressWriterSuppressesRepeatsAndStale(t *testing.T) {
	tail := &tailBuffer{max: 64}
	var got []Progress
	w := &progressWriter{
		tail:        tail,
		onProgress:  func(p Progress) { got = append(got, p) },
		minInterval: time.Hour, // after the first emit, the window never closes
	}

	segment := "Encoding: %s/240 Frames @ 100.0 fps\n"
	for _, frames := range []string{"10", "20", "20", "15", "240"} {
		if _, err := w.Write([]byte(strings.Replace(segment, "%s", frames, 1))); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}

	if len(got) != 1 || got[0].Frames != 10 {
		t.Fatalf("got %+v, want exactly the first update {10 240 100}: repeat suppressed by throttle, stale/backward counts ignored", got)
	}
}

func TestProgressWriterInjectsEstimatedTotal(t *testing.T) {
	tests := []struct {
		name           string
		estimatedTotal int
		segment        string
		want           Progress
	}{
		{
			name:           "no total yet: metadata estimate fills in",
			estimatedTotal: 33424,
			segment:        "Encoding:  100 Frames @ 22.50 fps | x",
			want:           Progress{Frames: 100, Total: 33424, Estimated: true, FPS: 22.5},
		},
		{
			name:           "encoder total wins over estimate",
			estimatedTotal: 999,
			segment:        "Encoding:  240/241 Frames @ 1475.51 fps | x",
			want:           Progress{Frames: 240, Total: 241, FPS: 1475.51},
		},
		{
			name:           "estimate already outgrown is not used",
			estimatedTotal: 50,
			segment:        "Encoding:  60 Frames @ 22.50 fps | x",
			want:           Progress{Frames: 60, Total: 0, FPS: 22.5},
		},
		{
			name:           "no estimate leaves total at zero",
			estimatedTotal: 0,
			segment:        "Encoding:  100 Frames @ 22.50 fps | x",
			want:           Progress{Frames: 100, Total: 0, FPS: 22.5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tail := &tailBuffer{max: 512}
			var got []Progress
			w := &progressWriter{
				tail:           tail,
				onProgress:     func(p Progress) { got = append(got, p) },
				estimatedTotal: tt.estimatedTotal,
				minInterval:    0,
			}
			if _, err := w.Write([]byte(tt.segment + "\n")); err != nil {
				t.Fatalf("write failed: %v", err)
			}
			if len(got) != 1 {
				t.Fatalf("got %d updates, want 1", len(got))
			}
			if got[0] != tt.want {
				t.Errorf("update = %+v, want %+v", got[0], tt.want)
			}
		})
	}
}

func TestProgressWriterNilCallbackKeepsTail(t *testing.T) {
	tail := &tailBuffer{max: 64}
	w := &progressWriter{tail: tail, minInterval: 0}
	if _, err := w.Write([]byte("Encoding:  2/4 Frames @ 10.0 fps")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if tail.String() != "Encoding:  2/4 Frames @ 10.0 fps" {
		t.Errorf("tail = %q, want raw stderr preserved", tail.String())
	}
}
