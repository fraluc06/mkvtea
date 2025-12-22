package config

const Version = "0.1.0"

type Config struct {
	Dir                string
	Lang               string   // Single language (backward compatibility)
	Languages          []string // Multiple languages for extraction
	OutDir             string
	SubsDir            string // Custom directory for external subtitles
	Mode               string // "extract", "merge"
	Recursive          bool
	DryRun             bool
	KeepAudio          string
	MaxProcs           int // Concurrency workers
	CheckpointInterval int // Save checkpoint every N files (0 = disabled)
}
