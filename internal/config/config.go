package config

type Config struct {
	Dir       string
	Lang      string
	OutDir    string
	SubsDir   string // Custom directory for external subtitles
	Mode      string // "extract", "merge"
	Recursive bool
	DryRun    bool
	KeepAudio string
	MaxProcs  int  // Concurrency workers
	FastEdit  bool // Use mkvpropedit (no remux)
}
