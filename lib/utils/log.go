package utils

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"
)

type LogLevel int

var STARTED = time.Now()

const (
	DEBUG LogLevel = 0
	INFO  LogLevel = 1
	WARN  LogLevel = 2
	ERROR LogLevel = 3
)

const LOG_LEVEL = INFO

// ANSI color codes
const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
)

// Bright colors for task differentiation
var taskColors = []string{
	"\033[36m", // Cyan
	"\033[33m", // Yellow
	"\033[35m", // Magenta
	"\033[32m", // Green
	"\033[34m", // Blue
	"\033[91m", // Bright Red
	"\033[92m", // Bright Green
	"\033[93m", // Bright Yellow
	"\033[94m", // Bright Blue
	"\033[95m", // Bright Magenta
	"\033[96m", // Bright Cyan
}

var (
	colorEnabled = false
	colorMu      sync.RWMutex
	taskColorMap = make(map[string]string)
	colorIndex   = 0
)

// SetColorEnabled enables or disables color output
func SetColorEnabled(enabled bool) {
	colorMu.Lock()
	defer colorMu.Unlock()
	colorEnabled = enabled
}

// IsColorEnabled returns whether color output is enabled
func IsColorEnabled() bool {
	colorMu.RLock()
	defer colorMu.RUnlock()
	return colorEnabled
}

// getColorForTask returns a consistent color for a task ID
func getColorForTask(id string) string {
	colorMu.Lock()
	defer colorMu.Unlock()

	if color, exists := taskColorMap[id]; exists {
		return color
	}

	// Use hash to get consistent color for same task ID
	h := fnv.New32a()
	h.Write([]byte(id))
	idx := int(h.Sum32()) % len(taskColors)
	color := taskColors[idx]
	taskColorMap[id] = color
	return color
}

func LogWithTaskId(id string, msg string, level LogLevel) {
	Since := time.Since(STARTED)
	if level >= LOG_LEVEL {
		if IsColorEnabled() {
			color := getColorForTask(id)
			timeColor := Dim
			fmt.Printf("%s%.1f%s %s[%s]%s %s\n", timeColor, Since.Seconds(), Reset, color, id, Reset, msg)
		} else {
			fmt.Printf("%.1f [%s] %s\n", Since.Seconds(), id, msg)
		}
	}
}

// LogStatus logs a status message with appropriate color
func LogStatus(id string, status string, isSuccess bool) {
	Since := time.Since(STARTED)
	if IsColorEnabled() {
		color := getColorForTask(id)
		statusColor := Green
		if !isSuccess {
			statusColor = Red
		}
		timeColor := Dim
		fmt.Printf("%s%.1f%s %s[%s]%s %s%s%s\n", timeColor, Since.Seconds(), Reset, color, id, Reset, statusColor, status, Reset)
	} else {
		fmt.Printf("%.1f [%s] %s\n", Since.Seconds(), id, status)
	}
}

// LogTaskStart logs when a task starts with highlighted command
func LogTaskStart(id string, cmd string) {
	Since := time.Since(STARTED)
	if IsColorEnabled() {
		color := getColorForTask(id)
		timeColor := Dim
		fmt.Printf("%s%.1f%s %s[%s]%s %s▶ Run%s %s%s%s\n",
			timeColor, Since.Seconds(), Reset,
			color, id, Reset,
			Bold, Reset,
			Cyan, cmd, Reset)
	} else {
		fmt.Printf("%.1f [%s] Run task -> %s\n", Since.Seconds(), id, cmd)
	}
}
