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
	Mode               string // "extract", "merge"
	Recursive          bool
	KeepOnlyAudio      string
	Audio              bool
	MaxProcs           int // Concurrency workers (auto-detected based on CPU count, 50% with min 2 and max 8)
	CheckpointInterval int // Save checkpoint every N files (0 = disabled)
}
