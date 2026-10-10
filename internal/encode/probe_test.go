package encode

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// requireProbeTools skips ffprobe-based tests when the toolchain is absent;
// probing is best-effort and the encoders themselves don't need it.
func requireProbeTools(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not installed", tool)
		}
	}
}

// genSample runs ffmpeg with lavfi to produce a fixture of n frames.
func genSample(t *testing.T, name string, rate int, extraArgs ...string) string {
	t.Helper()
	requireProbeTools(t)
	path := filepath.Join(t.TempDir(), name)
	args := []string{"-v", "error", "-y",
		"-f", "lavfi", "-i", "testsrc=duration=1:size=128x72:rate=10"}
	args = append(args, extraArgs...)
	args = append(args, path)
	if out, err := exec.Command("ffmpeg", args...).CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg fixture failed: %v: %s", err, out)
	}
	return path
}

func TestProbeEstimatedFramesMP4UsesExactFrameCount(t *testing.T) {
	requireProbeTools(t)
	mp4 := genSample(t, "sample.mp4", 10)
	if got := probeEstimatedFrames(mp4); got != 10 {
		t.Errorf("probeEstimatedFrames(%q) = %d, want exact 10 frames", mp4, got)
	}
}

func TestProbeEstimatedFramesMKVFallsBackToDurationTimesRate(t *testing.T) {
	requireProbeTools(t)
	// MKV carries no nb_frames; duration × r_frame_rate must approximate.
	mkv := genSample(t, "sample.mkv", 10)
	if got := probeEstimatedFrames(mkv); got < 9 || got > 11 {
		t.Errorf("probeEstimatedFrames(%q) = %d, want ≈10 frames", mkv, got)
	}
}

func TestProbeEstimatedFramesRejectsNonMediaFiles(t *testing.T) {
	requireProbeTools(t)
	path := filepath.Join(t.TempDir(), "garbage.mp4")
	if err := exec.Command("sh", "-c", "echo not a video > "+path).Run(); err != nil {
		t.Fatalf("fixture write failed: %v", err)
	}
	if got := probeEstimatedFrames(path); got != 0 {
		t.Errorf("probeEstimatedFrames(garbage) = %d, want 0", got)
	}
}

func TestParseFrameRate(t *testing.T) {
	tests := []struct {
		name string
		rate string
		want float64
		ok   bool
	}{
		{name: "integer rate", rate: "30/1", want: 30, ok: true},
		{name: "ntsc rate", rate: "30000/1001", want: 30000.0 / 1001.0, ok: true},
		{name: "zero numerator", rate: "0/1", ok: false},
		{name: "zero denominator", rate: "30/0", ok: false},
		{name: "not a rational", rate: "abc", ok: false},
		{name: "missing denominator", rate: "30", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseFrameRate(tt.rate)
			if ok != tt.ok {
				t.Fatalf("parseFrameRate(%q) ok = %v, want %v", tt.rate, ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Errorf("parseFrameRate(%q) = %v, want %v", tt.rate, got, tt.want)
			}
		})
	}
}

func TestIsPlausibleFrameCount(t *testing.T) {
	tests := []struct {
		name   string
		frames int
		want   bool
	}{
		{name: "zero", frames: 0, want: false},
		{name: "negative", frames: -5, want: false},
		{name: "normal episode", frames: 33424, want: true},
		{name: "absurd value rejected", frames: 2_000_000_000, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPlausibleFrameCount(tt.frames); got != tt.want {
				t.Errorf("isPlausibleFrameCount(%d) = %v, want %v", tt.frames, got, tt.want)
			}
		})
	}
}
