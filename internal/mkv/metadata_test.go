package mkv

import (
	"encoding/json"
	"testing"
)

func TestTrackStructure(t *testing.T) {
	// Test that Track struct properly deserializes JSON
	jsonData := `{
		"id": 0,
		"type": "video",
		"codec": "h264",
		"properties": {
			"language": "und",
			"track_name": "Main Video",
			"forced_track": false
		}
	}`

	var track Track
	err := json.Unmarshal([]byte(jsonData), &track)
	if err != nil {
		t.Fatalf("Failed to unmarshal track: %v", err)
	}

	if track.ID != 0 {
		t.Errorf("Expected ID 0, got %d", track.ID)
	}
	if track.Type != "video" {
		t.Errorf("Expected type 'video', got %s", track.Type)
	}
	if track.Props.Lang != "und" {
		t.Errorf("Expected language 'und', got %s", track.Props.Lang)
	}
}

func TestAttachmentStructure(t *testing.T) {
	// Test that Attachment struct properly deserializes JSON
	jsonData := `{
		"id": 1,
		"file_name": "Arial.ttf",
		"content_type": "application/x-truetype-font"
	}`

	var attachment Attachment
	err := json.Unmarshal([]byte(jsonData), &attachment)
	if err != nil {
		t.Fatalf("Failed to unmarshal attachment: %v", err)
	}

	if attachment.ID != 1 {
		t.Errorf("Expected ID 1, got %d", attachment.ID)
	}
	if attachment.FileName != "Arial.ttf" {
		t.Errorf("Expected FileName 'Arial.ttf', got %s", attachment.FileName)
	}
}

func TestInfoStructure(t *testing.T) {
	// Test that Info struct properly deserializes complete MKV metadata
	jsonData := `{
		"tracks": [
			{
				"id": 0,
				"type": "video",
				"codec": "h264",
				"properties": {
					"language": "und",
					"track_name": "",
					"forced_track": false
				}
			},
			{
				"id": 1,
				"type": "audio",
				"codec": "aac",
				"properties": {
					"language": "jpn",
					"track_name": "Japanese",
					"forced_track": false
				}
			},
			{
				"id": 2,
				"type": "subtitles",
				"codec": "ass",
				"properties": {
					"language": "ita",
					"track_name": "Italian",
					"forced_track": false
				}
			}
		],
		"attachments": [
			{
				"id": 1,
				"file_name": "Roboto.ttf",
				"content_type": "application/x-truetype-font"
			}
		],
		"chapters": []
	}`

	var info Info
	err := json.Unmarshal([]byte(jsonData), &info)
	if err != nil {
		t.Fatalf("Failed to unmarshal info: %v", err)
	}

	if len(info.Tracks) != 3 {
		t.Errorf("Expected 3 tracks, got %d", len(info.Tracks))
	}
	if len(info.Attachments) != 1 {
		t.Errorf("Expected 1 attachment, got %d", len(info.Attachments))
	}

	// Verify track types
	if info.Tracks[0].Type != "video" {
		t.Errorf("Expected first track to be 'video', got %s", info.Tracks[0].Type)
	}
	if info.Tracks[1].Type != "audio" {
		t.Errorf("Expected second track to be 'audio', got %s", info.Tracks[1].Type)
	}
	if info.Tracks[2].Type != "subtitles" {
		t.Errorf("Expected third track to be 'subtitles', got %s", info.Tracks[2].Type)
	}

	// Verify language codes
	if info.Tracks[1].Props.Lang != "jpn" {
		t.Errorf("Expected audio track language 'jpn', got %s", info.Tracks[1].Props.Lang)
	}
	if info.Tracks[2].Props.Lang != "ita" {
		t.Errorf("Expected subtitle track language 'ita', got %s", info.Tracks[2].Props.Lang)
	}
}

func TestMultipleAudioTracks(t *testing.T) {
	// Test parsing multiple audio tracks with different languages
	jsonData := `{
		"tracks": [
			{
				"id": 0,
				"type": "audio",
				"codec": "aac",
				"properties": {
					"language": "jpn",
					"track_name": "Japanese",
					"forced_track": false
				}
			},
			{
				"id": 1,
				"type": "audio",
				"codec": "aac",
				"properties": {
					"language": "eng",
					"track_name": "English",
					"forced_track": false
				}
			},
			{
				"id": 2,
				"type": "audio",
				"codec": "aac",
				"properties": {
					"language": "ita",
					"track_name": "Italian",
					"forced_track": false
				}
			}
		],
		"attachments": [],
		"chapters": []
	}`

	var info Info
	err := json.Unmarshal([]byte(jsonData), &info)
	if err != nil {
		t.Fatalf("Failed to unmarshal info: %v", err)
	}

	if len(info.Tracks) != 3 {
		t.Errorf("Expected 3 audio tracks, got %d", len(info.Tracks))
	}

	languages := []string{"jpn", "eng", "ita"}
	for i, expectedLang := range languages {
		if info.Tracks[i].Props.Lang != expectedLang {
			t.Errorf("Track %d: expected language '%s', got '%s'", i, expectedLang, info.Tracks[i].Props.Lang)
		}
	}
}

func TestMultipleSubtitleTracks(t *testing.T) {
	// Test parsing multiple subtitle tracks with different languages
	jsonData := `{
		"tracks": [
			{
				"id": 0,
				"type": "subtitles",
				"codec": "ass",
				"properties": {
					"language": "ita",
					"track_name": "Italian",
					"forced_track": false
				}
			},
			{
				"id": 1,
				"type": "subtitles",
				"codec": "ass",
				"properties": {
					"language": "ita",
					"track_name": "Italian (Sign&Song)",
					"forced_track": true
				}
			},
			{
				"id": 2,
				"type": "subtitles",
				"codec": "ass",
				"properties": {
					"language": "eng",
					"track_name": "English",
					"forced_track": false
				}
			}
		],
		"attachments": [],
		"chapters": []
	}`

	var info Info
	err := json.Unmarshal([]byte(jsonData), &info)
	if err != nil {
		t.Fatalf("Failed to unmarshal info: %v", err)
	}

	if len(info.Tracks) != 3 {
		t.Errorf("Expected 3 subtitle tracks, got %d", len(info.Tracks))
	}

	// Check forced track flag
	if !info.Tracks[1].Props.Forced {
		t.Errorf("Expected track 1 to have forced=true")
	}
	if info.Tracks[0].Props.Forced {
		t.Errorf("Expected track 0 to have forced=false")
	}
}
