// Package core provides fundamental types and functions.
package core

// Version returns the current version of the core module.
func Version() string {
	return "1.0.0"
}

// Config holds application configuration.
type Config struct {
	Name    string
	Debug   bool
	MaxSize int
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Name:    "default",
		Debug:   false,
		MaxSize: 100,
	}
}
