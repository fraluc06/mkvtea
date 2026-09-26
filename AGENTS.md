# MKVTea

A blazing-fast batch processing CLI with a BubbleTea TUI for extracting/merging subtitles, audio tracks, and fonts in Anime/TV series MKV libraries.

## Development Setup

```bash
# Toolchain (optional, mise pins Go)
mise install

# Dependencies
go mod download

# Development
go run . extract /path/to/dir -r -l ita

# Build
go build -o ./mkvtea

# Tests
go test ./...          # add -race for the concurrency-sensitive TUI code
```

## Tech Layers

- **Framework**: Cobra (CLI commands/flags) + BubbleTea v2 (TUI)
- **Language**: Go 1.25+ (go.mod toolchain; mise pins Go 1.26)
- **Styling**: Lipgloss v2 with the Catppuccin Mocha palette (`internal/ui/styles.go`)
- **Database**: None — state lives in `.mkvtea_checkpoint.json` (atomic temp-file + rename) inside the scanned directory
- **Testing**: stdlib `testing` with table-driven tests, no external assertion libs

## Project Structure

```
main.go                  # Entry point, only calls cmd.Execute()
cmd/
├── root.go              # Cobra root command, flags, extract/merge/encode factory, processFiles
├── scanner.go           # ScanFiles: mode-aware video discovery (encode adds containers, skips av1/ dirs)
└── scanner_test.go
internal/
├── config/
│   ├── config.go        # Config struct + Version (injected via -ldflags)
│   └── config_test.go
├── mkv/
│   ├── engine.go        # RunExtract/RunMerge, mkvmerge/mkvextract invocation, ErrSkipped
│   ├── metadata.go      # GetInfo: parses mkvmerge -J JSON (Track, Attachment, Info)
│   ├── parser.go        # GetEpisodeNumber: episode regex, compiled once at package level
│   └── *_test.go
├── encode/
│   ├── encode.go        # RunEncode: ffmpeg→SvtAv1EncApp y4m pipe, mkvmerge remux, output paths
│   ├── progress.go      # Parse SvtAv1EncApp stderr progress (\r-delimited, ANSI-colored) → plain ProgressFunc
│   ├── params.go        # ParseParams/ResolveParams: flag > file > .mkvtea-av1-params > defaults
│   └── *_test.go
├── ui/
│   ├── model.go         # ProcessModel state + Init/Update
│   ├── processing.go    # Worker goroutines, mode dispatch, checkpoint recording
│   ├── loglines.go      # One live log line per file: STARTED/ENCODING → final status, in place
│   ├── rendering.go     # Log truncation (rune-safe) + progress bar
│   ├── view.go          # View layout (holds m.mu while rendering)
│   ├── processor.go     # RunProcessTUI entry point, resume prompt, final summary
│   └── styles.go        # Lipgloss styles
└── checkpoint/
    └── checkpoint.go    # Checkpoint Manager: load/save/resume, MD5-based file matching
```

## Code Standards

### General Rules
- Import order: stdlib → external → internal, blank lines between groups
- Run `gofmt ./...` and `go vet ./...` before committing
- Handle every error explicitly — no `_ = err`; wrap with `fmt.Errorf("context: %w", err)`
- Match sentinel errors with `errors.Is` (e.g., `mkv.ErrSkipped`), never `err.Error() == "..."`
- Write tests for new features (table-driven where possible)

### Naming Conventions
- Packages: lowercase, single word (config, mkv, ui, checkpoint)
- Types: PascalCase (Config, Info, ProcessModel)
- Functions: PascalCase exported with doc comment starting with the name, camelCase unexported
- Variables: camelCase locals
- Constants: camelCase for unexported (e.g., `autoCloseDelay`); doc comment explains "why", not "what"

### File Organization
- One responsibility per file, target 50–150 LOC (see tree above for the split)
- Tests colocated as `*_test.go` next to the code they cover
- Keep `main.go` minimal; all logic lives in `cmd/` or `internal/`

## Important Patterns

### Dependency direction (law)
Import flow is strictly one-way: `cmd → ui → {mkv, encode, checkpoint}`. The TUI is a
leaf package; the engines must never know it exists:
- Only `cmd/root.go` may import `internal/ui`; `internal/ui` never imports `cmd`
- `charm.land/*` (bubbletea, bubbles, lipgloss) may appear ONLY inside `internal/ui`
- `internal/mkv`, `internal/encode`, `internal/checkpoint`, `internal/config` take plain
  args and return results — testable without a terminal
- Self-check: `grep -rlE 'charm.land' --include='*.go' cmd internal` lists only `internal/ui/` files

This convention is shared with burnmail (same rules, package names aside); keep both
AGENTS.md files aligned when either layout changes.

### External Tool Invocation
All MKV work goes through `mkvmerge`/`mkvextract`/`mkvpropedit` via `os/exec`; encode mode
additionally drives `ffmpeg | SvtAv1EncApp` (y4m pipe) then remuxes with `mkvmerge`.
Validate availability first (mode-aware — `processFiles` picks the right validator):
```go
if err := mkv.ValidateDependencies(); err != nil {
    return err // multi-line install instructions, printed once by Execute
}
// encode mode:
if err := encode.ValidateDependencies(); err != nil { // SvtAv1EncApp, ffmpeg, mkvmerge
    return err
}
```
When piping two commands, start the reader first, wire `cmdA.StdoutPipe()` into
`cmdB.Stdin`, then `Wait` the producer before the consumer (EOF closes the pipe).
SvtAv1EncApp streams live progress on stderr as `\r`-delimited segments with ANSI
color codes and, while piping stdin, without a frame total (`Encoding: 2 Frames
@ 18 fps` → `Encoding: 240/240 Frames @ 1475 fps`): `encode/progress.go` tees every
byte to the `tailBuffer` diagnostics and forwards parsed updates through a plain
`ProgressFunc` callback — the engine stays TUI-agnostic (dependency law). Until the
encoder learns the real total at input EOF (late on slow presets), `estimatedFrames`
derives one from `mkvmerge -J` container duration ÷ video `default_duration`; such
totals carry `Progress.Estimated` and the TUI renders them with `≈`. Because the
encoder only learns the total at input EOF (late on slow presets), `estimatedFrames`
derives one from `mkvmerge -J` container duration / video `default_duration` so the
TUI shows a `≈`-marked percent from frame one; the encoder's real total replaces it at
EOF, and cover-art video tracks without `default_duration` are skipped.

### State Management
- `ProcessModel` is the single source of truth for TUI state
- Worker goroutines mutate `logs`/counters/`viewport` only while holding `m.mu`
- `View` and `Update` (WindowSizeMsg/quit paths) must also hold `m.mu` when touching shared state — the BubbleTea renderer runs on its own goroutine
- Concurrency = buffered semaphore channel (`m.sem`) + `sync.WaitGroup`, worker count from `min(max(NumCPU/2, 2), 8)`
- The Processing Log keeps ONE line per file: `🔄 STARTED` appears when a worker slot
  is acquired, `🔄 ENCODING` rewrites it live, and the final `✅/⏭️/❌` replaces it in
  place via the index in `m.activeLogs` — never append a second line per file. New log
  prefixes must join `logPrefixes` in `ui/rendering.go` or truncation will miss them

## Testing Guidelines

- Write tests alongside implementation (`scanner_test.go`, `parser_test.go`, `metadata_test.go` are the models to follow)
- Focus on behavior: extension case-insensitivity, episode-number formats, JSON struct mapping
- Use `t.TempDir()` for filesystem fixtures; never touch the real filesystem outside it
- Run `go test -race ./...` before merging — the TUI shares state across goroutines

## Common Pitfalls to Avoid

- DON'T: Call `os.Exit` inside `RunE` — return the error and let `Execute` handle it
- DON'T: Byte-slice log lines — prefixes contain multi-byte emoji; use `strings.CutPrefix` and rune-aware truncation
- DON'T: Duplicate timing constants — the auto-close countdown and timer share `autoCloseDelay`
- DON'T: Compile regexes inside functions — hoist them to package level (see `episodePattern`)
- DON'T: Skip `gofmt`/`go vet` for "simple" changes
- DO: Check `internal/mkv` helpers (`getAudioExtension`, `isAudioExt`) before adding new ones
- DO: Keep `%w` for wrapped errors, `%q` for paths in error messages
- DO: Keep functions small and focused

## Performance Considerations

- Worker count auto-detected: 50% of CPUs, clamped to [2, 8]; overridable via config
- Encode mode defaults to 1 worker — SvtAv1EncApp saturates cores on a single video; parallel encodes thrash
- Process-level parallelism only — MKVToolNix does the heavy I/O, Go coordinates
- Preallocate slices when size is known (`make([]string, 0, len(files))`)
- Checkpoint saves are throttled by `--checkpoint-interval` (default every 10 files), not per file

## Deployment

- Tag-driven: `git tag vX.Y.Z && git push origin vX.Y.Z` triggers `.github/workflows/release.yml`
- The tag is the single source of truth for the version, injected via `-ldflags -X mkvtea/internal/config.Version=...` (`config.go` stays at "dev")
- The workflow tests, cross-builds 5 platforms, creates the GitHub Release with changelog + checksums, and pushes the multi-arch image to GHCR
- Local Docker: `docker compose build` → `mkvtea:local`; run via `docker compose run --rm mkvtea <args>` (needs `tty: true` for the TUI; media goes in the `/data` volume)
- Lint the Dockerfile with `droast --no-roast .` (config in `droast.toml`)

## Additional Resources

- Usage, screenshots, and install instructions: `README.md`
- CI pipeline: `.github/workflows/CI.yml`
- Release pipeline: `.github/workflows/release.yml`
- Docker setup and volume notes: `docker-compose.yml` (header comments)
- Sister project sharing the layout conventions: burnmail (its AGENTS.md mirrors the dependency-direction law above)
