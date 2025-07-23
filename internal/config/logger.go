package config

import (
	"os"

	"github.com/charmbracelet/log"
)

var Logger *log.Logger

func init() {
	Logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: false,
		Prefix:         "configr",
	})
	
	// Set default level to info (can be overridden)
	Logger.SetLevel(log.InfoLevel)
}

// SetVerbose enables verbose logging
func SetVerbose(verbose bool) {
	if verbose {
		Logger.SetLevel(log.DebugLevel)
	} else {
		Logger.SetLevel(log.InfoLevel)
	}
}

// Success logs a success message with checkmark
func Success(msg string, args ...interface{}) {
	Logger.Info("✓ "+msg, args...)
}

// Warning logs a warning message
func Warning(msg string, args ...interface{}) {
	Logger.Warn("⚠ "+msg, args...)
}

// Error logs an error message
func Error(msg string, args ...interface{}) {
	Logger.Error("✗ "+msg, args...)
}

// Info logs an info message
func Info(msg string, args ...interface{}) {
	Logger.Info(msg, args...)
}

// Debug logs a debug message (only shown in verbose mode)
func Debug(msg string, args ...interface{}) {
	Logger.Debug(msg, args...)
}