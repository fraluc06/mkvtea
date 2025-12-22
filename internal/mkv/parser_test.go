package mkv

import "testing"

func TestGetEpisodeNumber(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "Standard S01E02 format",
			filename: "Anime Title S01E02.mkv",
			expected: "02",
		},
		{
			name:     "S01E02 with brackets",
			filename: "[SubGroup] Anime Title - S01E02.mkv",
			expected: "02",
		},
		{
			name:     "Episode number with leading zero",
			filename: "episode_01.mkv",
			expected: "01",
		},
		{
			name:     "EP format",
			filename: "anime ep 05 title.mkv",
			expected: "05",
		},
		{
			name:     "Plain two digits",
			filename: "03_anime.mkv",
			expected: "03",
		},
		{
			name:     "Episode 12",
			filename: "Anime - 12 - Title.mkv",
			expected: "12",
		},
		{
			name:     "No episode number",
			filename: "opening.mkv",
			expected: "XX",
		},
		{
			name:     "Single digit (should not match)",
			filename: "anime_5.mkv",
			expected: "XX",
		},
		{
			name:     "Complex title with episode",
			filename: "[GroupName] Series - 10 [1080p].mkv",
			expected: "10",
		},
		{
			name:     "Episode at start of filename",
			filename: "01 - Title of Episode.mkv",
			expected: "01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEpisodeNumber(tt.filename)
			if result != tt.expected {
				t.Errorf("GetEpisodeNumber(%q) = %q; want %q", tt.filename, result, tt.expected)
			}
		})
	}
}
