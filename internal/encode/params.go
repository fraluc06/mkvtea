package encode

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"mkvtea/internal/config"
)

// ParamsFileName is the per-folder file mkvtea auto-discovers encoder params from.
const ParamsFileName = ".mkvtea-av1-params"

// reservedParams are owned by mkvtea for pipe wiring; users cannot override them.
var reservedParams = map[string]bool{
	"i": true, "input": true,
	"b": true, "output": true,
}

// paramKeyPattern matches SvtAv1EncApp long option names (after dashes are stripped).
var paramKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`)

// ParseParams parses ffmpeg-style params ("crf=30:tune=0:film-grain=8") into
// SvtAv1EncApp CLI arguments ("--crf", "30", ...). A bare key becomes a
// standalone flag. Values cannot contain colons (no SVT-AV1 option needs them).
func ParseParams(input string) ([]string, error) {
	var args []string
	for _, token := range strings.Split(input, ":") {
		key, value, hasValue, skip, err := parseToken(token)
		if err != nil {
			return nil, err
		}
		if skip {
			continue
		}
		args = append(args, "--"+key)
		if hasValue {
			args = append(args, value)
		}
	}
	return args, nil
}

// ParseParamsFile parses params file content: one "key=value" per line with
// # comments and blank lines allowed. Colons still separate multiple pairs,
// so a flag-style one-liner pasted into a file keeps working.
func ParseParamsFile(content string) ([]string, error) {
	var args []string
	for lineNo, line := range strings.Split(content, "\n") {
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		for _, token := range strings.Split(line, ":") {
			key, value, hasValue, skip, err := parseToken(token)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo+1, err)
			}
			if skip {
				continue
			}
			args = append(args, "--"+key)
			if hasValue {
				args = append(args, value)
			}
		}
	}
	return args, nil
}

// parseToken validates one "key=value" (or bare "key") token.
// skip reports an empty token left over by splitting/comment stripping.
func parseToken(token string) (key, value string, hasValue, skip bool, err error) {
	token = strings.TrimSpace(token)
	token = strings.TrimLeft(token, "-") // courtesy: accidental leading dashes
	if token == "" {
		return "", "", false, true, nil
	}

	key, value, hasValue = strings.Cut(token, "=")
	if !paramKeyPattern.MatchString(key) {
		return "", "", false, false, fmt.Errorf("invalid parameter %q: expected key=value (no dashes needed)", token)
	}
	if reservedParams[key] {
		return "", "", false, false, fmt.Errorf("parameter %q is reserved: mkvtea controls input/output", key)
	}
	if hasValue && value == "" {
		return "", "", false, false, fmt.Errorf("parameter %q has an empty value", key)
	}
	return key, value, hasValue, false, nil
}

// ResolveParams returns encoder args from the highest-precedence source:
// --svtav1-params flag > --av1-params-file > nearest .mkvtea-av1-params
// (source file's folder, walking up to the scan root) > encoder defaults.
// One source wins whole; sources never merge. source describes the winner
// for logging ("" when encoder defaults apply).
func ResolveParams(cfg config.Config, srcFile string) (args []string, source string, err error) {
	if cfg.SvtAv1Params != "" {
		args, err := ParseParams(cfg.SvtAv1Params)
		if err != nil {
			return nil, "", fmt.Errorf("invalid --svtav1-params: %w", err)
		}
		return args, "--svtav1-params", nil
	}

	if cfg.Av1ParamsFile != "" {
		args, err := parseParamsFilePath(cfg.Av1ParamsFile)
		if err != nil {
			return nil, "", err
		}
		return args, cfg.Av1ParamsFile, nil
	}

	if discovered := discoverParamsFile(cfg.Dir, srcFile); discovered != "" {
		args, err := parseParamsFilePath(discovered)
		if err != nil {
			return nil, "", err
		}
		return args, discovered, nil
	}

	return nil, "", nil
}

// ValidateParams checks the run-wide param sources (flag and explicit file)
// so a typo fails before processing starts instead of failing every file.
// Per-folder discovery files are validated lazily by ResolveParams.
func ValidateParams(cfg config.Config) error {
	if cfg.SvtAv1Params != "" {
		if _, err := ParseParams(cfg.SvtAv1Params); err != nil {
			return fmt.Errorf("invalid --svtav1-params: %w", err)
		}
	}
	if cfg.Av1ParamsFile != "" {
		if _, err := parseParamsFilePath(cfg.Av1ParamsFile); err != nil {
			return err
		}
	}
	return nil
}

func parseParamsFilePath(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read params file %q: %w", path, err)
	}
	args, err := ParseParamsFile(string(content))
	if err != nil {
		return nil, fmt.Errorf("invalid params in %q: %w", path, err)
	}
	return args, nil
}

// discoverParamsFile walks from the source file's directory up to (and
// including) the scan root, returning the first params file found.
// The nearest folder wins, so per-series tuning overrides a library-wide file.
func discoverParamsFile(root, srcFile string) string {
	// Single-file mode: cfg.Dir is the file itself, walk from its folder.
	if info, err := os.Stat(root); err == nil && !info.IsDir() {
		root = filepath.Dir(root)
	}
	root = filepath.Clean(root)

	dir := filepath.Dir(srcFile)
	for {
		candidate := filepath.Join(dir, ParamsFileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		if dir == root {
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir { // filesystem root, defensive stop
			return ""
		}
		dir = parent
	}
}
