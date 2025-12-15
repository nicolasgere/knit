// Package api provides HTTP API handlers.
package api

import (
	"encoding/json"

	"example.com/core"
	"example.com/utils"
)

// Response represents an API response.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// VersionResponse returns version information as JSON.
func VersionResponse() ([]byte, error) {
	resp := Response{
		Success: true,
		Data: map[string]string{
			"version": utils.GetVersionInfo(),
		},
	}
	return json.Marshal(resp)
}

// ConfigResponse returns config information as JSON.
func ConfigResponse(name string) ([]byte, error) {
	cfg := utils.ConfigWithDefaults(name)
	resp := Response{
		Success: true,
		Data:    cfg,
	}
	return json.Marshal(resp)
}

// HealthCheck returns a health check response.
func HealthCheck() Response {
	return Response{
		Success: true,
		Data: map[string]string{
			"status":  "healthy",
			"version": core.Version(),
		},
	}
}
