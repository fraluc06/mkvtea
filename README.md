# 🍵 MKVTea

> A blazing-fast batch processing tool for managing your Anime/TV Series library with beautiful TUI interface.

Extract and merge subtitles, fonts, and chapters from MKV files with ease. Perfect for anime collectors who need to batch process hundreds of episodes.

[![Go](https://img.shields.io/badge/Go-1.25%2B-blue?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Build](https://img.shields.io/badge/Build-Passing-brightgreen)](#-installation)

## ✨ Features

- **🚀 Blazing Fast**: Concurrent processing with customizable worker threads
- **🎨 Beautiful TUI**: Responsive terminal UI with real-time progress tracking
- **📁 Recursive Processing**: Handle massive libraries with one command
- **🎯 Smart Language Selection**: Extract subtitles in any language (ISO 639-2 codes)
- **🔊 Audio Cleaning**: Keep only desired audio language, remove bloat
- **🎬 Directory Mirroring**: Maintains folder structure automatically
- **✅ Dependency Validation**: Clear error messages if MKVToolNix not installed
- **⚡ Fast Metadata Mode**: Quick edits without remuxing with `-f` flag

## 📋 Requirements

- **Go 1.25+** (for building from source)
- **MKVToolNix** (mkvmerge, mkvextract, mkvpropedit)

### Install MKVToolNix

```bash
# macOS
brew install mkvtoolnix

# Ubuntu/Debian
sudo apt install mkvtoolnix

# Fedora/RHEL
sudo dnf install mkvtoolnix

# Arch Linux
sudo pacman -S mkvtoolnix-cli
```

## 🚀 Installation

### From Source

```bash
git clone https://github.com/yourusername/mkvtea.git
cd mkvtea
go build
sudo mv mkvtea /usr/local/bin/  # Optional: add to PATH
```

### Pre-built Binary

Download from [Releases](https://github.com/yourusername/mkvtea/releases)

## 📖 Usage

### Basic Commands

```bash
# Extract subtitles from current directory
./mkvtea e .

# Extract subtitles recursively
./mkvtea e /path/to/anime -r

# Extract specific language
./mkvtea e /path/to/anime -r -l eng

# Merge subtitles back
./mkvtea m /path/to/anime -r

# Merge with audio cleaning (keep only Japanese)
./mkvtea m /path/to/anime -r -a jpn
```

### Global Flags

| Flag            | Short | Default | Description                                          |
|:----------------|:-----:|:-------:|------------------------------------------------------|
| `--lang`        | `-l`  |  `ita`  | Subtitle language code (ita, eng, jpn, etc.)         |
| `--output`      | `-o`  |    -    | Custom output directory                              |
| `--subs-dir`    | `-s`  |    -    | Custom directory for external subtitles (merge only) |
| `--recursive`   | `-r`  | `false` | Process all subdirectories                           |
| `--dry-run`     | `-d`  | `false` | Simulate without modifying files                     |
| `--audio`       | `-a`  |    -    | Keep only this audio language (removes others)       |
| `--concurrency` | `-c`  |   `2`   | Max parallel workers                                 |
| `--fast`        | `-f`  | `false` | Fast metadata-only mode (no remux)                   |

## 💡 Examples

### Extract Italian Subtitles

```bash
./mkvtea e /anime/season1 -r -l ita
```

Creates:
```
/anime/season1/
├── episode01.mkv
├── episode02.mkv
└── subs/ita/
    ├── 01_ita_9.ass
    ├── 02_ita_9.ass
    └── ...
```

### Merge with Audio Cleaning

```bash
./mkvtea m /anime/season1 -r -l ita -a jpn
```

Results in:
- Removes all original subtitles
- Adds Italian subtitles (set as DEFAULT)
- Keeps only Japanese audio
- Creates `/anime/season1_ita/` with processed files
- **File size reduced by ~40-50%**

### Extract English from Multiple Series

```bash
./mkvtea e /anime/downloads -r -l eng -c 4
```

- Processes all MKV files recursively
- Uses 4 parallel workers for faster processing
- Great for SSDs

### Dry-Run Preview

```bash
./mkvtea e /anime -r -d
```

Shows what would happen without modifying files.

### Merge from Custom Subtitle Directory

```bash
./mkvtea m /anime/episodes -r -l ita -s /external/subs
```

Searches for subtitles in `/external/subs/` instead of default location.

### Fast Metadata Edit (No Remux)

```bash
./mkvtea m /anime -r -f
```

Uses `mkvpropedit` for instant metadata changes without remuxing.

## 🎨 TUI Interface

The beautiful terminal UI shows:

```
🍵 MKVTEA - EXTRACT
📦  12 Total  │  ✅  10 Success  │  ⏭️  1 Skipped  │  ❌  1 Failed
█████████░░░░░░░░░░ 50% [6/12]
✨ Processing files...
📋 Processing Log:
✅ SUCCESS: [RigAV1] Saenai Heroine no Sodatekata - S01E01.mkv
✅ SUCCESS: [RigAV1] Saenai Heroine no Sodatekata - S01E02.mkv
⏭️  SKIPPED: opening.mkv
🔄 Window closes in 5 second(s) | Press Q or Ctrl+C to exit now
```

Features:
- **Responsive**: Adapts to any terminal size
- **Real-time Progress**: Updates as files are processed
- **Color-coded Status**: Green for success, yellow for skipped, red for failed
- **Auto-close**: Closes after 5 seconds (or press Q/Ctrl+C)
- **Scrollable Logs**: Full filename visibility with smart truncation

## 🔍 Language Codes

ISO 639-2 three-letter codes:

| Language           | Code  |
|:-------------------|:------|
| Japanese           | `jpn` |
| Italian            | `ita` |
| English            | `eng` |
| German             | `deu` |
| French             | `fra` |
| Spanish            | `spa` |
| Portuguese         | `por` |
| Chinese (Mandarin) | `zho` |
| Korean             | `kor` |
| Russian            | `rus` |

## 🐛 Troubleshooting

### ❌ "Missing required MKV tools"

**Solution**: Install MKVToolNix (see [Requirements](#-requirements))

### ❌ "No MKV files found"

- Verify path exists
- Check file extensions are `.mkv` (case-insensitive)
- Use `-r` flag for recursive search

### ⏭️ "SKIPPED: filename.mkv"

- File doesn't have subtitles in the requested language
- Normal for opening/ending sequences

### ❌ "Corrupted or unreadable MKV file"

- File is damaged
- Try: `mkvmerge -o "fixed.mkv" "corrupted.mkv"`

## 🏗️ Project Structure

```
mkvtea/
├── main.go                         # Entry point
├── cmd/
│   ├── root.go                    # CLI setup, command definitions, flags
│   └── scanner.go                 # File scanning logic (recursive/non-recursive)
├── internal/
│   ├── config/
│   │   └── config.go              # Configuration struct
│   ├── mkv/
│   │   ├── engine.go              # Core extract/merge logic (228 LOC)
│   │   ├── metadata.go            # GetInfo + JSON structs (Track, Attachment, MkvInfo)
│   │   └── parser.go              # Episode number extraction
│   └── ui/
│       ├── model.go               # ProcessModel struct + Init/Update lifecycle (142 LOC)
│       ├── view.go                # View rendering method
│       ├── processing.go          # File processing logic + concurrency (92 LOC)
│       ├── rendering.go           # Log and progress bar rendering (82 LOC)
│       ├── processor.go           # RunProcessTUI entry point
│       └── styles.go              # Catppuccin Mocha theme
├── go.mod / go.sum                # Go dependencies
├── AGENTS.md                       # Development guidelines for agents
└── README.md                       # This file
```

### File Responsibility

- **`cmd/scanner.go`** - Find MKV files in directories
- **`mkv/metadata.go`** - Read MKV file metadata (tracks, attachments)
- **`mkv/parser.go`** - Extract episode numbers from filenames
- **`mkv/engine.go`** - Core MKV operations (extract, merge, property editing)
- **`ui/model.go`** - BubbleTea model state + lifecycle (Init, Update)
- **`ui/processing.go`** - Concurrent file processing logic
- **`ui/rendering.go`** - Progress bars and log rendering
- **`ui/view.go`** - TUI display layout
- **`ui/processor.go`** - Entry point for TUI execution

**Design principle**: Each file has a single, clear responsibility (50-150 LOC target)

## 🎨 Design

### Color Theme: Catppuccin Mocha

Beautiful dark theme with carefully chosen colors:
- Background: `#1e1e2e`
- Text: `#cdd6f4`
- Accent: `#89b4fa` (blue)
- Success: `#a6e3a1` (green)
- Warning: `#fab387` (peach)
- Error: `#f38ba8` (red)

## 📦 Dependencies

```
github.com/charmbracelet/bubbletea    # TUI framework
github.com/charmbracelet/bubbles      # UI components
github.com/charmbracelet/lipgloss     # Styling
github.com/spf13/cobra                # CLI framework
```

## 🔄 Workflow Example

### Scenario: Process 12 anime episodes

```bash
# Step 1: Extract Italian subtitles
./mkvtea e ~/anime/downloads/season1 -r -l ita

# Step 2: Verify subtitles (they're in season1/subs/ita/)
ls ~/anime/downloads/season1/subs/ita/

# Step 3: (Optional) Edit subtitles with external tool
# ...

# Step 4: Merge subtitles back
./mkvtea m ~/anime/downloads/season1 -r -l ita

# Step 5: Organized files are in ~/anime/downloads/season1_ita/
ls ~/anime/downloads/season1_ita/
```

## 🛠️ Development

### Build from Source

```bash
go build
```

### Run Tests

```bash
go test ./...
```

### Format Code

```bash
go fmt ./...
```

## 📄 License

MIT License - see [LICENSE](LICENSE) file

## 🤝 Contributing

Contributions welcome! Please feel free to:
- Report bugs
- Suggest features
- Submit pull requests

## ❓ FAQ

**Q: Can I use different languages for extract and merge?**
A: Extract and merge must use the same language. Extract with `-l eng` then merge with `-l eng`.

**Q: Does it work on Windows?**
A: Yes, if MKVToolNix is installed and in PATH.

**Q: How do I speed up processing?**
A: Use `-c 8` for SSD (increase workers), or `-f` for metadata-only mode.

**Q: Can I merge subtitles from a different folder?**
A: Yes! Use `-s /path/to/subs` flag in merge mode.

**Q: Will it overwrite my original files?**
A: No. Extract creates a `subs/` folder. Merge creates a `directory_lang/` folder.

## 📞 Support

- Open an [Issue](https://github.com/yourusername/mkvtea/issues)
- Check [Discussions](https://github.com/yourusername/mkvtea/discussions)

---

Made with ❤️ for anime enthusiasts everywhere.

**🍵 MKVTea - Batch process your anime library like a pro!**
