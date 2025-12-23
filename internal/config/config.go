package config

const Version = "0.1.2"

type Config struct {
	Dir                string
	Lang               string   // Single language (backward compatibility)
	Languages          []string // Multiple languages for extraction
	OutDir             string
	SubsDir            string // Custom directory for external subtitles
	Mode               string // "extract", "merge"
	Recursive          bool
	KeepAudio          string
	MaxProcs           int // Concurrency workers (auto-detected based on CPU count, 50% with min 2 and max 8)
	CheckpointInterval int // Save checkpoint every N files (0 = disabled)
}
