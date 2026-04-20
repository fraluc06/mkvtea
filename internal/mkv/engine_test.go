package mkv

import "testing"

func TestGetAudioExtension(t *testing.T) {
	tests := []struct {
		codec    string
		expected string
	}{
		{"AC-3", ".ac3"},
		{"E-AC-3", ".eac3"},
		{"DTS", ".dts"},
		{"FLAC", ".flac"},
		{"AAC", ".aac"},
		{"MP3", ".mp3"},
		{"MPEG-1 Layer 3", ".mp3"},
		{"Opus", ".opus"},
		{"Vorbis", ".ogg"},
		{"PCM", ".wav"},
		{"Unknown", ".mka"},
	}

	for _, tt := range tests {
		got := getAudioExtension(tt.codec)
		if got != tt.expected {
			t.Errorf("getAudioExtension(%q) = %q; want %q", tt.codec, got, tt.expected)
		}
	}
}

func TestIsAudioExt(t *testing.T) {
	tests := []struct {
		ext      string
		expected bool
	}{
		{".ac3", true},
		{".eac3", true},
		{".dts", true},
		{".flac", true},
		{".aac", true},
		{".m4a", true},
		{".mp3", true},
		{".opus", true},
		{".ogg", true},
		{".wav", true},
		{".mka", true},
		{".srt", false},
		{".ass", false},
		{".txt", false},
		{".mp4", false},
		{".mkv", false},
	}

	for _, tt := range tests {
		got := isAudioExt(tt.ext)
		if got != tt.expected {
			t.Errorf("isAudioExt(%q) = %v; want %v", tt.ext, got, tt.expected)
		}
	}
}
