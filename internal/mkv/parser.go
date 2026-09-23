package mkv

import "regexp"

// episodePattern matches patterns like S01E02, EP02, 02, etc.
// Compiled once at package level instead of on every call.
var episodePattern = regexp.MustCompile(`(?i)(?:s\d{1,2}|e|ep|\s|\.|_|^)(\d{2})(?:\s|\.|_|v|$)`)

// GetEpisodeNumber extracts the episode number from a filename
func GetEpisodeNumber(filename string) string {
	matches := episodePattern.FindStringSubmatch(filename)
	if len(matches) > 1 {
		return matches[1]
	}
	return "XX"
}
