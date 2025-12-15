// Package main is the application entry point.
package main

import (
	"fmt"
	"os"

	"example.com/api"
	"example.com/core"
)

func main() {
	fmt.Println("App starting...")
	fmt.Println("Core version:", core.Version())

	resp := api.HealthCheck()
	if resp.Success {
		fmt.Println("Health check passed!")
	} else {
		fmt.Println("Health check failed!")
		os.Exit(1)
	}
}

// GetAppInfo returns application information.
func GetAppInfo() string {
	return fmt.Sprintf("App v%s", core.Version())
}
