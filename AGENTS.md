# AGENTS.md - MKVTea Development Guide

## Build & Test Commands
- `go build -o ./mkvtea` - Build binary
- `go test ./...` - Run all tests
- `go test -v ./internal/mkv -run TestName` - Run single test
- `go fmt ./...` - Format all code
- `go vet ./...` - Lint checks
- `go mod tidy` - Clean dependencies

## Code Style Guidelines

### Imports & Formatting
- Order: stdlib → external → internal (blank lines between groups)
- Use `gofmt` before every commit
- No import aliases unless necessary

### Naming Conventions
- **Packages**: lowercase, single word (config, mkv, ui)
- **Types**: PascalCase (Config, MkvInfo, ProcessModel)
- **Functions**: PascalCase exported, camelCase unexported
- **Variables**: camelCase locals, PascalCase exported constants
- **Constants**: ALL_CAPS with underscores
- **Interface methods**: Receiver pointers for Update/modification operations

### Error Handling
- Always handle errors explicitly (no `_ = err`)
- Return `error` as last parameter
- Wrap errors with context before returning
- Early returns on errors (no nested conditionals)

### Project Structure
- `cmd/` - CLI entry point (Cobra commands) + file scanning
- `internal/config/` - Configuration struct
- `internal/mkv/` - Core MKV processing (engine, metadata, parser)
- `internal/ui/` - BubbleTea TUI (model, view, processor, styles)

### File Organization by Responsibility
- `engine.go` (228 LOC) - Core extract/merge logic
- `metadata.go` (60 LOC) - GetInfo + JSON structs (Track, Attachment, MkvInfo)
- `parser.go` (14 LOC) - Episode number extraction
- `model.go` (142 LOC) - ProcessModel struct + Init/Update/lifecycle
- `processing.go` (92 LOC) - File processing logic + concurrency
- `rendering.go` (82 LOC) - Log and progress bar rendering
- `view.go` (94 LOC) - View rendering method
- `processor.go` (68 LOC) - RunProcessTUI entry point
- Target: 50-150 LOC per file for clarity and maintainability

### BubbleTea Patterns
- Update: handles messages, returns (Model, Cmd). View: renders state (string)
- Use pointer receivers for Update methods
- Follow tea.Msg patterns for custom messages

### Dependencies
- **TUI**: charmbracelet/bubbletea, bubbles, lipgloss
- **CLI**: spf13/cobra
- **External**: mkvmerge, mkvextract, mkvpropedit (os/exec)

### Language & Documentation
- All code comments and identifiers in English
- Comments explain "why" not "what" (code shows what)
- Exported functions must have doc comments starting with function name
