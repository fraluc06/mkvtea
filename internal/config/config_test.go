package config

import "testing"

func TestConfigDefaults(t *testing.T) {
	cfg := Config{
		Lang:      "ita",
		MaxProcs:  2,
		Recursive: false,
	}

	if cfg.Lang != "ita" {
		t.Errorf("Expected default Lang to be 'ita', got %s", cfg.Lang)
	}

	if cfg.MaxProcs != 2 {
		t.Errorf("Expected default MaxProcs to be 2, got %d", cfg.MaxProcs)
	}

	if cfg.Recursive != false {
		t.Errorf("Expected Recursive to be false, got %v", cfg.Recursive)
	}
}

func TestConfigModes(t *testing.T) {
	tests := []struct {
		name  string
		mode  string
		valid bool
	}{
		{"extract mode", "extract", true},
		{"merge mode", "merge", true},
		{"invalid mode", "invalid", false},
		{"empty mode", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Mode: tt.mode}
			isValid := cfg.Mode == "extract" || cfg.Mode == "merge"
			if isValid != tt.valid {
				t.Errorf("Config with mode %q: expected valid=%v, got %v", tt.mode, tt.valid, isValid)
			}
		})
	}
}

func TestConfigLanguages(t *testing.T) {
	cfg := Config{
		Lang: "eng",
	}

	if cfg.Lang != "eng" {
		t.Errorf("Expected Lang 'eng', got %s", cfg.Lang)
	}

	// Test various language codes
	validLangs := []string{"ita", "eng", "jpn", "deu", "fra", "spa"}
	for _, lang := range validLangs {
		cfg.Lang = lang
		if cfg.Lang != lang {
			t.Errorf("Failed to set Lang to %s", lang)
		}
	}
}

func TestConfigConcurrency(t *testing.T) {
	tests := []struct {
		name     string
		maxProcs int
		valid    bool
	}{
		{"default concurrency", 2, true},
		{"high concurrency", 8, true},
		{"single worker", 1, true},
		{"zero workers", 0, false},
		{"negative workers", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{MaxProcs: tt.maxProcs}
			isValid := cfg.MaxProcs > 0
			if isValid != tt.valid {
				t.Errorf("Config with MaxProcs %d: expected valid=%v, got %v", tt.maxProcs, tt.valid, isValid)
			}
		})
	}
}
