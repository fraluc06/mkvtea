package encode

import (
	"encoding/json"
	"math"
	"os/exec"
	"strconv"
	"strings"
)

// probeEstimatedFrames estimates a video's frame count with ffprobe, for
// containers whose `mkvmerge -J` output hides duration/default_duration (MP4
// among them), leaving estimatedFrames with nothing to compute. nb_frames
// wins when present (exact for MP4); otherwise duration × r_frame_rate
// approximates. Best-effort by design: any failure reports 0 and the TUI
// falls back to the encoder's own total, which arrives at input EOF.
func probeEstimatedFrames(path string) int {
	out, err := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=nb_frames,duration,r_frame_rate:format=duration",
		"-of", "json", path,
	).Output()
	if err != nil {
		return 0
	}

	var payload struct {
		Streams []struct {
			NbFrames  string `json:"nb_frames"`
			Duration  string `json:"duration"`
			FrameRate string `json:"r_frame_rate"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return 0
	}
	if len(payload.Streams) == 0 {
		return 0
	}
	stream := payload.Streams[0]

	// Exact when the container stores the sample count (MP4 does).
	if n, err := strconv.Atoi(strings.TrimSpace(stream.NbFrames)); err == nil && isPlausibleFrameCount(n) {
		return n
	}

	fps, ok := parseFrameRate(stream.FrameRate)
	if !ok {
		return 0
	}
	seconds := firstFloat(stream.Duration, payload.Format.Duration)
	if seconds <= 0 {
		return 0
	}
	// Round, don't truncate: a 0.999... float would otherwise shave a frame
	// off the estimate.
	frames := int(math.Round(seconds * fps))
	if !isPlausibleFrameCount(frames) {
		return 0
	}
	return frames
}

// isPlausibleFrameCount mirrors estimatedFrames' bound: 1e9 covers a ~13-day
// 24 fps video; zero, negative and absurd values are rejected.
func isPlausibleFrameCount(frames int) bool {
	return frames > 0 && frames <= 1e9
}

// parseFrameRate reads ffprobe's "num/den" rational form ("30000/1001").
func parseFrameRate(s string) (float64, bool) {
	num, den, found := strings.Cut(strings.TrimSpace(s), "/")
	if !found {
		return 0, false
	}
	n, errN := strconv.ParseFloat(num, 64)
	d, errD := strconv.ParseFloat(den, 64)
	if errN != nil || errD != nil || n <= 0 || d <= 0 {
		return 0, false
	}
	return n / d, true
}

// firstFloat returns the first parseable positive float among its arguments,
// probing fallback fields in order.
func firstFloat(values ...string) float64 {
	for _, v := range values {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil && f > 0 {
			return f
		}
	}
	return 0
}
