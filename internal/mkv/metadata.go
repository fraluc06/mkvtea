package mkv

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Track represents a track entry from mkvmerge JSON output
type Track struct {
	ID    int    `json:"id"`
	Type  string `json:"type"`
	Codec string `json:"codec"`
	Props struct {
		Lang      string `json:"language"`
		TrackName string `json:"track_name"`
		Forced    bool   `json:"forced_track"`
	} `json:"properties"`
}

// Attachment represents an attachment (e.g., font) in an MKV file
type Attachment struct {
	ID          int    `json:"id"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
}

// MkvInfo contains metadata about an MKV file
type MkvInfo struct {
	Tracks      []Track       `json:"tracks"`
	Attachments []Attachment  `json:"attachments"`
	Chapters    []interface{} `json:"chapters"`
}

// GetInfo analyzes MKV file metadata using mkvmerge
func GetInfo(path string) (*MkvInfo, error) {
	// Verify file exists and is accessible
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("file inaccessible (permissions?): %s", path)
	}

	cmd := exec.Command("mkvmerge", "-J", path)
	out, err := cmd.Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("corrupted or unreadable MKV file: %s", path)
		}
		return nil, fmt.Errorf("failed to analyze MKV: %v", err)
	}

	var info MkvInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("failed to parse MKV metadata: %v", err)
	}
	return &info, nil
}
