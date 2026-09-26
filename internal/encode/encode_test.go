package encode

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"mkvtea/internal/config"
	"mkvtea/internal/mkv"
)

func TestOutputPath(t *testing.T) {
	tests := []struct {
		name string
		src  string
		cfg  config.Config
		want string
	}{
		{
			name: "default per-folder av1 subdir",
			src:  "/anime/season1/ep01.mkv",
			cfg:  config.Config{Dir: "/anime/season1"},
			want: "/anime/season1/av1/ep01.mkv",
		},
		{
			name: "extension always becomes mkv",
			src:  "/anime/movie.mp4",
			cfg:  config.Config{Dir: "/anime"},
			want: "/anime/av1/movie.mkv",
		},
		{
			name: "custom out subdir honored",
			src:  "/anime/ep.mkv",
			cfg:  config.Config{Dir: "/anime", OutSubdir: "encoded"},
			want: "/anime/encoded/ep.mkv",
		},
		{
			name: "explicit -o roots output",
			src:  "/anime/season1/ep01.mkv",
			cfg:  config.Config{Dir: "/anime", OutDir: "/out"},
			want: "/out/season1/ep01.mkv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := outputPath(tt.src, tt.cfg)
			if filepath.ToSlash(got) != tt.want {
				t.Fatalf("outputPath(%q) = %q, want %q", tt.src, got, tt.want)
			}
		})
	}
}

func TestTailBufferKeepsLastBytes(t *testing.T) {
	b := &tailBuffer{max: 4}
	for _, chunk := range []string{"aaaa", "bbbb", "cc"} {
		if _, err := b.Write([]byte(chunk)); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}
	if got := b.String(); !strings.HasSuffix(got, "bbcc") || len(got) != 4 {
		t.Fatalf("tailBuffer retained %q, want last 4 bytes", got)
	}
}

func TestEstimatedFrames(t *testing.T) {
	// Fixtures mirror real `mkvmerge -J` output; all durations are ns.
	// 1394060000000 / 41708333 = 33424 frames of 23.976 fps video.
	tests := []struct {
		name string
		json string
		want int
	}{
		{
			name: "real episode math",
			json: `{"container":{"properties":{"duration":1394060000000}},
			       "tracks":[{"id":0,"type":"video","properties":{"default_duration":41708333}}]}`,
			want: 33424,
		},
		{
			name: "missing container duration",
			json: `{"tracks":[{"id":0,"type":"video","properties":{"default_duration":40000000}}]}`,
			want: 0,
		},
		{
			name: "video without default_duration",
			json: `{"container":{"properties":{"duration":10000000000}},
			       "tracks":[{"id":0,"type":"video","properties":{}}]}`,
			want: 0,
		},
		{
			name: "cover art skipped, main video estimated",
			json: `{"container":{"properties":{"duration":10000000000}},
			       "tracks":[{"id":0,"type":"video","properties":{}},
			                 {"id":1,"type":"video","properties":{"default_duration":40000000}}]}`,
			want: 250,
		},
		{
			name: "no video track",
			json: `{"container":{"properties":{"duration":10000000000}},
			       "tracks":[{"id":0,"type":"audio","properties":{}}]}`,
			want: 0,
		},
		{
			name: "absurd frame count is rejected",
			json: `{"container":{"properties":{"duration":1000000000000000000}},
			       "tracks":[{"id":0,"type":"video","properties":{"default_duration":1}}]}`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var info mkv.Info
			if err := json.Unmarshal([]byte(strings.Join(strings.Fields(tt.json), " ")), &info); err != nil {
				t.Fatalf("unmarshal fixture: %v", err)
			}
			if got := estimatedFrames(&info); got != tt.want {
				t.Errorf("estimatedFrames() = %d, want %d", got, tt.want)
			}
		})
	}
}
