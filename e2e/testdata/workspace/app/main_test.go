package main

import "testing"

func TestGetAppInfo(t *testing.T) {
	info := GetAppInfo()
	if info == "" {
		t.Error("app info should not be empty")
	}
	if len(info) < 5 {
		t.Error("app info seems too short")
	}
}
