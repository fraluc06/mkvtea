package ui

import (
	"testing"

	"mkvtea/internal/checkpoint"
	"mkvtea/internal/config"
)

func TestFlushCheckpointLocked(t *testing.T) {
	// A clean run flushes the checkpoint away: nothing left to resume.
	t.Run("clean run clears checkpoint", func(t *testing.T) {
		dir := t.TempDir()
		cfg := config.Config{Dir: dir, Mode: "encode", CheckpointInterval: 10}
		mgr, err := checkpoint.NewManager(cfg)
		if err != nil {
			t.Fatalf("NewManager: %v", err)
		}
		if err := mgr.Create(cfg, 1); err != nil {
			t.Fatalf("Create: %v", err)
		}

		m := &ProcessModel{cfg: cfg, checkpointMgr: mgr}
		m.mu.Lock()
		m.flushCheckpointLocked()
		m.mu.Unlock()

		if checkpoint.Exists(cfg) {
			t.Error("checkpoint file survived a clean run, want it cleared")
		}
		if len(m.logs) != 0 {
			t.Errorf("unexpected warnings on clean flush: %q", m.logs)
		}
	})

	// A run with failures keeps the checkpoint AND persists the tail that
	// interval saves never covered.
	t.Run("failed run flushes the tail", func(t *testing.T) {
		dir := t.TempDir()
		cfg := config.Config{Dir: dir, Mode: "encode", Lang: "ita", CheckpointInterval: 10}
		mgr, err := checkpoint.NewManager(cfg)
		if err != nil {
			t.Fatalf("NewManager: %v", err)
		}
		if err := mgr.Create(cfg, 2); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := mgr.AddSuccess(dir + "/ok.mkv"); err != nil {
			t.Fatalf("AddSuccess: %v", err)
		}
		if err := mgr.AddFailed(dir+"/broken.mkv", "mkvmerge: no good streams"); err != nil {
			t.Fatalf("AddFailed: %v", err)
		}

		m := &ProcessModel{cfg: cfg, checkpointMgr: mgr, errorCount: 1}
		m.mu.Lock()
		m.flushCheckpointLocked()
		m.mu.Unlock()

		if !checkpoint.Exists(cfg) {
			t.Fatal("checkpoint file missing after a failed run, want it kept for resume")
		}
		fresh, err := checkpoint.NewManager(cfg)
		if err != nil {
			t.Fatalf("NewManager: %v", err)
		}
		cp, err := fresh.Load()
		if err != nil || cp == nil {
			t.Fatalf("Load: cp=%v err=%v", cp, err)
		}
		if len(cp.Processed.Successful) != 1 || len(cp.Processed.Failed) != 1 {
			t.Errorf("persisted state lost the tail: %+v", cp.Processed)
		}
		if len(m.logs) != 0 {
			t.Errorf("unexpected warnings on failed flush: %q", m.logs)
		}
	})

	// Interval disabled: silent no-op, nil manager tolerated.
	t.Run("disabled interval is a no-op", func(t *testing.T) {
		m := &ProcessModel{cfg: config.Config{CheckpointInterval: 0}}
		m.mu.Lock()
		m.flushCheckpointLocked()
		m.mu.Unlock()
		if len(m.logs) != 0 {
			t.Errorf("no-op flush logged warnings: %q", m.logs)
		}
	})
}
