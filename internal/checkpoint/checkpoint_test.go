package checkpoint

import (
	"os"
	"path/filepath"
	"testing"

	"mkvtea/internal/config"
)

func TestExists(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Config{Dir: dir, Mode: "encode", CheckpointInterval: 10}

	if Exists(cfg) {
		t.Fatal("Exists = true before any checkpoint was created")
	}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := mgr.Create(cfg, 1); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !Exists(cfg) {
		t.Fatal("Exists = false after Create/Save, want true")
	}
	if err := mgr.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if Exists(cfg) {
		t.Fatal("Exists = true after Clear, want false")
	}
}

func TestCheckpointDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "ep01.mkv")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	missing := filepath.Join(dir, "gone.mkv")

	tests := []struct {
		name string
		cfg  config.Config
		want string
	}{
		{
			name: "scanned folder stays as-is",
			cfg:  config.Config{Dir: dir},
			want: dir,
		},
		{
			name: "single-file mode resolves to parent",
			cfg:  config.Config{Dir: file},
			want: dir,
		},
		{
			name: "missing path keeps its original meaning",
			cfg:  config.Config{Dir: missing},
			want: missing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkpointDir(tt.cfg); got != tt.want {
				t.Errorf("checkpointDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewManagerLocations(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "ep01.mkv")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	// Regression: in single-file mode the checkpoint used to be joined
	// INSIDE the .mkv path, making every checkpoint operation fail ENOTDIR.
	mgr, err := NewManager(config.Config{Dir: file})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if want := filepath.Join(dir, checkpointFileName); mgr.checkpointFile != want {
		t.Errorf("single-file checkpointFile = %q, want %q", mgr.checkpointFile, want)
	}

	mgr, err = NewManager(config.Config{Dir: dir})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if want := filepath.Join(dir, checkpointFileName); mgr.checkpointFile != want {
		t.Errorf("dir-mode checkpointFile = %q, want %q", mgr.checkpointFile, want)
	}
}

func TestCreateSaveLoadRoundtripSingleFileMode(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "ep01.mkv")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	cfg := config.Config{Dir: file, Mode: "encode", Lang: "ita", CheckpointInterval: 1}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	if err := mgr.Create(cfg, 2); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := mgr.AddSuccess(file); err != nil {
		t.Fatalf("AddSuccess: %v", err)
	}
	if err := mgr.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, checkpointFileName)); err != nil {
		t.Fatalf("checkpoint file not in the containing folder: %v", err)
	}

	fresh, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	cp, err := fresh.Load()
	if err != nil || cp == nil {
		t.Fatalf("Load: cp=%v err=%v", cp, err)
	}
	if cp.Directory != dir {
		t.Errorf("stored Directory = %q, want resolved folder %q", cp.Directory, dir)
	}
	if !fresh.IsProcessed(file) {
		t.Error("IsProcessed(file) = false after reload, want true")
	}

	canResume, err := CanResume(cfg)
	if err != nil {
		t.Errorf("CanResume: %v", err)
	}
	if !canResume {
		t.Error("CanResume = false after save, want true")
	}
}
