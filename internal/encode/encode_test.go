package encode

import (
	"path/filepath"
	"strings"
	"testing"

	"mkvtea/internal/config"
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
