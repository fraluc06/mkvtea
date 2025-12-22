package mkv

import "regexp"

// GetEpisodeNumber extracts the episode number from a filename
// Uses regex to match patterns like S01E02, EP02, 02, etc.
func GetEpisodeNumber(filename string) string {
	re := regexp.MustCompile(`(?i)(?:s\d{1,2}|e|ep|\s|\.|_|^)(\d{2})(?:\s|\.|_|v|$)`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) > 1 {
		return matches[1]
	}
	return "XX"
}
