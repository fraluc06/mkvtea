package config

// Version is injected at build time via -ldflags "-X mkvtea/internal/config.Version=...".
// The git tag is the single source of truth; local builds fall back to "dev".
var Version = "dev"

type Config struct {
	Dir                string
	Lang               string   // Single language (backward compatibility)
	Languages          []string // Multiple languages for extraction
	OutDir             string
	SubsDir            string // Custom directory for external subtitles
	AudioDir           string // Custom directory for external audio
	Mode               string // "extract", "merge", "encode"
	Recursive          bool
	KeepOnlyAudio      string
	Audio              bool
	MaxProcs           int // Concurrency workers (auto-detected based on CPU count, 50% with min 2 and max 8)
	CheckpointInterval int // Save checkpoint every N files (0 = disabled)

	// Encode mode (AV1 via SVT-AV1-Essential)
	SvtAv1Params  string // ffmpeg-style "key=value:key=value" pairs for SvtAv1EncApp
	Av1ParamsFile string // File with encoder params (lower precedence than SvtAv1Params)
	OutSubdir     string // Per-folder output subdirectory name (default "av1")
	LangExplicit  bool   // True when -l was explicitly passed; encode filters subtitles only then
}
