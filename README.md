#  MKVTea

> A blazing-fast batch processing tool for managing your Anime/TV Series library with beautiful TUI interface.

[![Go](https://img.shields.io/badge/Go-1.25%2B-blue?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)
[![Build](https://img.shields.io/badge/Build-Passing-brightgreen)](#-installation)

##  TUI Interface

![MKVTea TUI](assets/mkvtea_TUI.png)

> *Screenshot example showing the tool in action with sample anime files*

## ✨ Features

- **🚀 Blazing Fast**: Concurrent processing with customizable worker threads
- **🎨 Beautiful TUI**: Responsive terminal UI with real-time progress tracking
- **📁 Recursive Processing**: Handle massive libraries with one command
- **🎯 Smart Language Selection**: Extract subtitles in any language (ISO 639-2 codes)
- **🔊 Audio Cleaning**: Keep only desired audio language, remove bloat
- **🎬 Directory Mirroring**: Maintains folder structure automatically
- **✅ Dependency Validation**: Clear error messages if MKVToolNix not installed

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

Download from [Releases](https://github.com/fraluc06/mkvtea/releases)

## 📖 Usage

### Basic Commands

```bash
# Extract subtitles from current directory
./mkvtea e .

# Extract subtitles recursively
./mkvtea e /path/to/anime -r

# Extract specific language
./mkvtea e /path/to/anime -r -l eng

# Extract multiple languages at once
./mkvtea e /path/to/anime -r -l ita,eng,jpn

# Merge subtitles back
./mkvtea m /path/to/anime -r

# Merge with audio cleaning (keep only Japanese)
./mkvtea m /path/to/anime -r -a jpn

```

### Global Flags

| Flag                    | Short | Default | Description                                                       |
|:------------------------|:-----:|:-------:|-------------------------------------------------------------------|
| `--lang`                | `-l`  |  `ita`  | Subtitle language code(s): single (eng) or multiple (ita,eng,jpn) |
| `--output`              | `-o`  |    -    | Custom output directory                                           |
| `--subs-dir`            | `-s`  |    -    | Custom directory for external subtitles (merge only)              |
| `--recursive`           | `-r`  | `false` | Process all subdirectories                                        |
| `--audio`               | `-a`  |    -    | Keep only this audio language (removes others)                    |
| `--checkpoint-interval` |   -   |  `10`   | Save checkpoint every N files (0 to disable)                      |

### Performance Tuning

**Parallel Processing**: MKVTea automatically detects the optimal number of worker threads based on your CPU count:
- Uses **50% of available CPU cores** (e.g., 4 cores → 2 workers)
- Minimum: **2 workers** (for slower systems)
- Maximum: **8 workers** (to avoid overwhelming your system)

## 💡 Examples of Use

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

### Merge from Custom Subtitle Directory

```bash
./mkvtea m /anime/episodes -r -l ita -s /external/subs
```

Searches for subtitles in `/external/subs/` instead of default location.

### Resume Interrupted Processing with Checkpoints

Process failed mid-way? Pick up where you left off:

**How checkpoints work:**
- ✅ Saves progress every N files (default: 10)
- 💾 Stores `.mkvtea_checkpoint.json` in target directory
- 🔄 Auto-detects previous checkpoints on next run
- 🗑️ Clear checkpoint and restart: select `n` at prompt
- 🔐 Tracks by filename + MD5 hash (detects renamed files)

**Example checkpoint file:**
```json
{
  "mode": "extract",
  "languages": ["ita"],
  "directory": "/anime/library",
  "total_files": 1000,
  "started_at": "2025-12-22T10:30:00Z",
  "processed": {
    "successful": 350,
    "failed": 12,
    "skipped": 15
  }
}
```

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

## 🤝 Contributing

Contributions welcome! Please feel free to:
- Report bugs
- Suggest features
- Submit pull requests

## 🐛 Troubleshooting

### ❌ "No MKV files found"

- Verify path exists
- Check file extensions are `.mkv` (case-insensitive)
- Use `-r` flag for recursive search

### ⏭️ "SKIPPED: filename.mkv"

- File doesn't have subtitles in the requested language
- Normal for opening/ending sequences

## ⚠️ Disclaimers

- Screenshots and examples shown are for demonstration purposes only
- File names and content displayed are sample data to illustrate functionality
- MKVTea is a processing tool designed to work with media files on your system
- Users should only process media files they have the legal right to modify
- This tool does not distribute, stream, or handle copyrighted content - it simply processes local files

##  License

MIT License - see [LICENSE](LICENSE) file

---

Made with ❤️ by [fraluc06](https://github.com/fraluc06)
