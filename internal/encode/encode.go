package encode

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"mkvtea/internal/config"
	"mkvtea/internal/mkv"
)

// DefaultOutSubdir is the per-folder output directory name for encoded files.
const DefaultOutSubdir = "av1"

// stderrTail caps captured tool output so long encodes cannot grow memory.
const stderrTail = 4096

// ValidateDependencies checks that the AV1 encoding toolchain is installed.
func ValidateDependencies() error {
	tools := []string{"SvtAv1EncApp", "ffmpeg", "mkvmerge"}
	var missingTools []string

	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			missingTools = append(missingTools, tool)
		}
	}

	if len(missingTools) > 0 {
		return fmt.Errorf(`❌ Missing required encoding tools: %s

macOS — install the tap's FFmpeg (bundled with SVT-AV1-Essential) and CLI:

  brew install fraluc06/ffmpeg-svt-av1-essential/ffmpeg
  brew install fraluc06/ffmpeg-svt-av1-essential/svt-av1-essential

  (tap: https://github.com/fraluc06/homebrew-ffmpeg-svt-av1-essential)

Linux:

  sudo apt install ffmpeg mkvtoolnix   # or your distro's equivalent
  SvtAv1EncApp → build SVT-AV1-Essential: https://github.com/nekotrix/SVT-AV1-Essential

Ensure they are in your PATH and try again`, strings.Join(missingTools, ", "))
	}

	return nil
}

// RunEncode encodes one video file to AV1 with SvtAv1EncApp, then remuxes all
// non-video streams (audio, subtitles, attachments, chapters) from the source.
// onProgress (nil when unused) receives live status updates from the video
// pass, throttled to one per progressEmitInterval.
// Returns mkv.ErrSkipped when the output already exists (idempotent reruns).
func RunEncode(path string, cfg config.Config, onProgress ProgressFunc) error {
	info, err := mkv.GetInfo(path)
	if err != nil {
		return err
	}

	outPath := outputPath(path, cfg)
	if _, err := os.Stat(outPath); err == nil {
		return mkv.ErrSkipped
	}
	if err := os.MkdirAll(filepath.Dir(outPath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	params, _, err := ResolveParams(cfg, path)
	if err != nil {
		return err
	}

	// Temp bitstream lives in the (scanner-excluded) output dir and is
	// removed on every path, like the zsh script's *.tmp.mkv handling.
	tmp, err := os.CreateTemp(filepath.Dir(outPath), ".tmp-*.ivf")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpPath) }() // best-effort cleanup

	if err := encodeVideo(path, tmpPath, params, onProgress); err != nil {
		return err
	}

	return remux(path, tmpPath, outPath, info, cfg)
}

// outputPath returns the destination: <folder>/<OutSubdir>/<name>.mkv, or
// mirrored under -o when set (consistent with merge mode). Always .mkv.
func outputPath(path string, cfg config.Config) string {
	base := filepath.Base(path)
	if ext := filepath.Ext(base); ext != "" {
		base = strings.TrimSuffix(base, ext) + ".mkv"
	}

	if cfg.OutDir != "" {
		rel, err := filepath.Rel(cfg.Dir, path)
		if err != nil || strings.HasPrefix(rel, "..") {
			rel = filepath.Base(path)
		}
		return filepath.Join(cfg.OutDir, filepath.Dir(rel), base)
	}

	subdir := cfg.OutSubdir
	if subdir == "" {
		subdir = DefaultOutSubdir
	}
	return filepath.Join(filepath.Dir(path), subdir, base)
}

// encodeVideo decodes the source to 10-bit y4m with ffmpeg and pipes it into
// SvtAv1EncApp, replacing the zsh `<(...)` process substitution. onProgress
// (may be nil) is fed the encoder's live stderr progress segments.
func encodeVideo(srcPath, tmpPath string, params []string, onProgress ProgressFunc) error {
	ffmpegCmd := exec.Command("ffmpeg",
		"-hide_banner", "-loglevel", "error",
		"-i", srcPath,
		"-f", "yuv4mpegpipe",
		"-pix_fmt", "yuv420p10le",
		"-strict", "-1",
		"-",
	)
	encArgs := []string{"-i", "stdin", "-b", tmpPath}
	encCmd := exec.Command("SvtAv1EncApp", append(encArgs, params...)...)

	var ffmpegErr, encErr tailBuffer
	ffmpegErr.max = stderrTail
	encErr.max = stderrTail
	ffmpegCmd.Stderr = &ffmpegErr
	encCmd.Stderr = &progressWriter{
		tail:        &encErr,
		onProgress:  onProgress,
		minInterval: progressEmitInterval,
	}

	pipe, err := ffmpegCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create ffmpeg pipe: %w", err)
	}
	encCmd.Stdin = pipe

	// The encoder blocks reading stdin; start it before ffmpeg.
	if err := encCmd.Start(); err != nil {
		return fmt.Errorf("failed to start SvtAv1EncApp: %w", err)
	}
	if err := ffmpegCmd.Start(); err != nil {
		_ = encCmd.Process.Kill()
		_ = encCmd.Wait()
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// ffmpeg exiting closes the pipe, signalling EOF to the encoder.
	ffErr := ffmpegCmd.Wait()
	enErr := encCmd.Wait()

	// When the encoder dies, ffmpeg follows via SIGPIPE: report the encoder.
	if enErr != nil {
		return fmt.Errorf("encoding failed (SvtAv1EncApp): %w: %s", enErr, encErr.String())
	}
	if ffErr != nil {
		return fmt.Errorf("decoding failed (ffmpeg): %w: %s", ffErr, ffmpegErr.String())
	}
	return nil
}

// remux muxes the encoded video with every non-video stream of the source
// (audio, subtitles, attachments, chapters, tags) — mkvmerge's default.
func remux(srcPath, tmpPath, outPath string, info *mkv.Info, cfg config.Config) error {
	args := []string{"-o", outPath}

	// Preserve the source video track's language/name on the AV1 track,
	// which would otherwise come out of the raw bitstream as "und".
	for _, t := range info.Tracks {
		if t.Type == "video" {
			if t.Props.Lang != "" && t.Props.Lang != "und" {
				args = append(args, "--language", "0:"+t.Props.Lang)
			}
			if t.Props.TrackName != "" {
				args = append(args, "--track-name", "0:"+t.Props.TrackName)
			}
			break
		}
	}

	args = append(args, tmpPath, "--no-video")

	// Explicit -l: keep only subtitle tracks in those languages (+ und).
	if cfg.LangExplicit && len(cfg.Languages) > 0 {
		var subIDs []string
		for _, t := range info.Tracks {
			if t.Type == "subtitles" && (slices.Contains(cfg.Languages, t.Props.Lang) || t.Props.Lang == "und") {
				subIDs = append(subIDs, strconv.Itoa(t.ID))
			}
		}
		if len(subIDs) > 0 {
			args = append(args, "--subtitle-tracks", strings.Join(subIDs, ","))
		} else {
			args = append(args, "--no-subtitles")
		}
	}

	// Like merge mode, the audio filter only applies when it matches:
	// silently dropping every audio track on a typo would be too destructive.
	if cfg.KeepOnlyAudio != "" {
		var audioIDs []string
		for _, t := range info.Tracks {
			if t.Type == "audio" && t.Props.Lang == cfg.KeepOnlyAudio {
				audioIDs = append(audioIDs, strconv.Itoa(t.ID))
			}
		}
		if len(audioIDs) > 0 {
			args = append(args, "--audio-tracks", strings.Join(audioIDs, ","))
		}
	}

	args = append(args, srcPath)

	cmd := exec.Command("mkvmerge", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.Remove(outPath) // don't leave a broken mux behind; muxing error wins
		return fmt.Errorf("muxing failed (mkvmerge): %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// tailBuffer keeps only the last max bytes written, for stderr diagnostics.
type tailBuffer struct {
	buf []byte
	max int
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.max {
		b.buf = append([]byte(nil), b.buf[len(b.buf)-b.max:]...)
	}
	return len(p), nil
}

func (b *tailBuffer) String() string {
	return strings.TrimSpace(string(b.buf))
}
