package utils

import "testing"

func TestGetVersionInfo(t *testing.T) {
	info := GetVersionInfo()
	if info == "" {
		t.Error("version info should not be empty")
	}
	if len(info) < 10 {
		t.Error("version info seems too short")
	}
}

func TestConfigWithDefaults(t *testing.T) {
	cfg := ConfigWithDefaults("myapp")
	if cfg.Name != "myapp" {
		t.Errorf("expected name 'myapp', got %q", cfg.Name)
	}
}

func TestStringSliceContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !StringSliceContains(slice, "b") {
		t.Error("should contain 'b'")
	}
	if StringSliceContains(slice, "x") {
		t.Error("should not contain 'x'")
	}
}
