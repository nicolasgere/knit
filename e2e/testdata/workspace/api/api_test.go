package api

import (
	"encoding/json"
	"testing"
)

func TestVersionResponse(t *testing.T) {
	data, err := VersionResponse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !resp.Success {
		t.Error("expected success to be true")
	}
}

func TestConfigResponse(t *testing.T) {
	data, err := ConfigResponse("testapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if !resp.Success {
		t.Error("expected success to be true")
	}
}

func TestHealthCheck(t *testing.T) {
	resp := HealthCheck()
	if !resp.Success {
		t.Error("expected success to be true")
	}

	data, ok := resp.Data.(map[string]string)
	if !ok {
		t.Fatal("expected Data to be map[string]string")
	}

	if data["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got %q", data["status"])
	}
}
