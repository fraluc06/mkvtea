package encode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mkvtea/internal/config"
)

func TestParseParams(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr string
	}{
		{
			name:  "empty string yields no args",
			input: "",
			want:  nil,
		},
		{
			name:  "single pair",
			input: "crf=30",
			want:  []string{"--crf", "30"},
		},
		{
			name:  "multiple pairs",
			input: "crf=30:tune=0:film-grain=8",
			want:  []string{"--crf", "30", "--tune", "0", "--film-grain", "8"},
		},
		{
			name:  "bare key becomes standalone flag",
			input: "enable-restoration",
			want:  []string{"--enable-restoration"},
		},
		{
			name:  "leading dashes are stripped",
			input: "--crf=30",
			want:  []string{"--crf", "30"},
		},
		{
			name:  "value may contain equals sign",
			input: "zones=0,10,28",
			want:  []string{"--zones", "0,10,28"},
		},
		{
			name:    "reserved input key rejected",
			input:   "crf=30:i=nope",
			wantErr: "reserved",
		},
		{
			name:    "reserved output key rejected",
			input:   "b=nope",
			wantErr: "reserved",
		},
		{
			name:    "empty value rejected",
			input:   "crf=",
			wantErr: "empty value",
		},
		{
			name:    "invalid characters rejected",
			input:   "crf 30",
			wantErr: "invalid parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseParams(tt.input)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ParseParams(%q) error = %v, want containing %q", tt.input, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseParams(%q) unexpected error: %v", tt.input, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseParams(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("ParseParams(%q) = %v, want %v", tt.input, got, tt.want)
				}
			}
		})
	}
}

func TestParseParamsFile(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
		wantErr string
	}{
		{
			name:    "comment and blank lines ignored",
			content: "# a comment\n\ncrf=30\n\n# another\ntune=0\n",
			want:    []string{"--crf", "30", "--tune", "0"},
		},
		{
			name:    "inline comment stripped",
			content: "crf=30 # quality\ntune=0\n",
			want:    []string{"--crf", "30", "--tune", "0"},
		},
		{
			name:    "flag-style one-liner pasted into file",
			content: "crf=30:tune=0\n",
			want:    []string{"--crf", "30", "--tune", "0"},
		},
		{
			name:    "error reports line number",
			content: "crf=30\ncrf 30\n",
			wantErr: "line 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseParamsFile(tt.content)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ParseParamsFile error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Join(got, " ") != strings.Join(tt.want, " ") {
				t.Fatalf("ParseParamsFile = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveParamsPrecedence(t *testing.T) {
	dir := t.TempDir()
	srcFile := filepath.Join(dir, "show.mkv")
	if err := os.WriteFile(srcFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	// Put a discovery file in the source folder (lowest precedence).
	if err := os.WriteFile(filepath.Join(dir, ParamsFileName), []byte("crf=40\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// And an explicit file (middle precedence).
	explicitFile := filepath.Join(dir, "explicit.txt")
	if err := os.WriteFile(explicitFile, []byte("tune=1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		cfg     config.Config
		wantArg string
		wantSrc string
	}{
		{
			name:    "flag wins over files",
			cfg:     config.Config{Dir: dir, SvtAv1Params: "crf=20", Av1ParamsFile: explicitFile},
			wantArg: "--crf",
			wantSrc: "--svtav1-params",
		},
		{
			name:    "explicit file wins over discovery",
			cfg:     config.Config{Dir: dir, Av1ParamsFile: explicitFile},
			wantArg: "--tune",
			wantSrc: explicitFile,
		},
		{
			name:    "discovery file used as fallback",
			cfg:     config.Config{Dir: dir},
			wantArg: "--crf",
			wantSrc: filepath.Join(dir, ParamsFileName),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, source, err := ResolveParams(tt.cfg, srcFile)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(args) == 0 || args[0] != tt.wantArg {
				t.Fatalf("ResolveParams args = %v, want first %q", args, tt.wantArg)
			}
			if source != tt.wantSrc {
				t.Fatalf("ResolveParams source = %q, want %q", source, tt.wantSrc)
			}
		})
	}
}

func TestResolveParamsDefaults(t *testing.T) {
	dir := t.TempDir()
	srcFile := filepath.Join(dir, "show.mkv")
	if err := os.WriteFile(srcFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	args, source, err := ResolveParams(config.Config{Dir: dir}, srcFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(args) != 0 {
		t.Fatalf("expected no encoder args, got %v", args)
	}
	if source != "" {
		t.Fatalf("expected empty source for defaults, got %q", source)
	}
}

func TestDiscoverParamsFileWalkUp(t *testing.T) {
	root := t.TempDir()
	series := filepath.Join(root, "show", "season1")
	if err := os.MkdirAll(series, 0755); err != nil {
		t.Fatal(err)
	}

	// Root-level file (library-wide default)
	rootFile := filepath.Join(root, ParamsFileName)
	if err := os.WriteFile(rootFile, []byte("crf=35\n"), 0644); err != nil {
		t.Fatal(err)
	}

	srcFile := filepath.Join(series, "ep01.mkv")
	if err := os.WriteFile(srcFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	// No series file yet: root file should be found by walking up.
	if got := discoverParamsFile(root, srcFile); got != rootFile {
		t.Fatalf("discoverParamsFile = %q, want %q", got, rootFile)
	}

	// Nearest folder wins: add a series-level file.
	seriesFile := filepath.Join(series, ParamsFileName)
	if err := os.WriteFile(seriesFile, []byte("crf=25\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := discoverParamsFile(root, srcFile); got != seriesFile {
		t.Fatalf("discoverParamsFile = %q, want nearest %q", got, seriesFile)
	}
}

func TestDiscoverParamsFileSingleFileMode(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ParamsFileName), []byte("crf=35\n"), 0644); err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(root, "movie.mkv")
	if err := os.WriteFile(srcFile, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	// cfg.Dir is the file itself: discovery must still find the sibling file.
	got := discoverParamsFile(srcFile, srcFile)
	if got != filepath.Join(root, ParamsFileName) {
		t.Fatalf("discoverParamsFile = %q, want the sibling params file", got)
	}
}

func TestValidateParams(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		wantErr bool
	}{
		{
			name: "empty sources are valid",
			cfg:  config.Config{},
		},
		{
			name:    "invalid flag params rejected",
			cfg:     config.Config{SvtAv1Params: "crf=30:i=x"},
			wantErr: true,
		},
		{
			name:    "missing file rejected",
			cfg:     config.Config{Av1ParamsFile: filepath.Join(t.TempDir(), "nope.txt")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParams(tt.cfg)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
