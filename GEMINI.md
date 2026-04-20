# MKVTea - Project Context for Gemini

MKVTea is a blazing-fast batch processing tool for managing MKV and MP4 media libraries. It provides a terminal UI (TUI) for extracting and merging subtitles, audio tracks, and fonts.

## Project Overview

- **Core Technology:** Go (Golang) 1.25+
- **CLI Framework:** [Cobra](https://github.com/spf13/cobra)
- **TUI Framework:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) (via charm.land forks)
- **External Dependencies:** [MKVToolNix](https://mkvtoolnix.download/) (`mkvmerge`, `mkvextract`, `mkvpropedit`)

### Architecture

- `cmd/`: CLI entry points and file scanning logic.
- `internal/config/`: Configuration structure and validation.
- `internal/mkv/`: Logic for interacting with MKVToolNix binaries, metadata parsing, and codec detection.
- `internal/ui/`: TUI implementation (Model-View-Update pattern).
- `internal/checkpoint/`: Persistence logic for resuming interrupted batch jobs.

## Building and Running

### Commands

- **Build:** `go build -o mkvtea main.go`
- **Test:** `go test ./...`
- **Run (Extract):** `./mkvtea extract [dir|file] [flags]`
- **Run (Merge):** `./mkvtea merge [dir|file] [flags]`

### Key Flags

- `-l, --lang`: Target language code (e.g., `ita`, `eng`). Supports multiple codes: `ita,eng`.
- `-a, --audio`: Boolean. If set, extracts/merges audio tracks matching the target language.
- `-r, --recursive`: Process subdirectories recursively.
- `-s, --subs-dir`: Custom directory for external subtitles (merge mode).
- `--audio-dir`: Custom directory for external audio tracks (merge mode).
- `-o, --output`: Custom output directory for merged files.
- `--keep-only-audio`: Filter to keep only a specific audio language.

## Development Conventions

- **File Scanning:** Supports both `.mkv` and `.mp4` files. MP4 files are automatically converted to MKV during merge.
- **Metadata:** Uses `mkvmerge -J` to parse file structure as JSON.
- **Audio Detection:** Maps codecs to extensions (e.g., AAC -> `.aac`, AC3 -> `.ac3`, DTS -> `.dts`).
- **Parallelism:** Automatically uses ~50% of available CPU cores (min 2, max 8) for parallel processing.
- **Error Handling:** Files failing or missing target assets are marked as "SKIPPED" or "FAILED" in the TUI without stopping the entire batch.
- **Checkpoints:** Automatically saves progress to `.mkvtea_checkpoint.json` to allow resuming long tasks.

## Code Responsibilities

- `internal/mkv/engine.go`: Orchestrates calls to `mkvmerge` and `mkvextract`.
- `internal/mkv/metadata.go`: Defines the JSON structure returned by `mkvmerge -J`.
- `internal/ui/processing.go`: Manages the worker pool and synchronization between the TUI and the background tasks.
- `cmd/scanner.go`: Handles file system discovery for media files.
