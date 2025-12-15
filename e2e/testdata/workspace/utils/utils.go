// Package utils provides utility functions that build on core.
package utils

import "example.com/core"

// GetVersionInfo returns formatted version information.
func GetVersionInfo() string {
	return "Version: " + core.Version()
}

// ConfigWithDefaults returns a config with some customizations.
func ConfigWithDefaults(name string) core.Config {
	cfg := core.DefaultConfig()
	cfg.Name = name
	return cfg
}

// StringSliceContains checks if a slice contains a string.
func StringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
