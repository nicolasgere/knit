package core

import "testing"

func TestVersion(t *testing.T) {
	v := Version()
	if v == "" {
		t.Error("version should not be empty")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Name != "default" {
		t.Errorf("expected name 'default', got %q", cfg.Name)
	}
	if cfg.Debug {
		t.Error("debug should be false by default")
	}
	if cfg.MaxSize != 100 {
		t.Errorf("expected max size 100, got %d", cfg.MaxSize)
	}
}
