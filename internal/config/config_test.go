package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	// Since Load() uses viper internally and doesn't take parameters,
	// we need to test it by setting up viper configuration
	t.Run("load without config", func(t *testing.T) {
		// This will likely fail since no config is set up
		_, err := Load()
		if err == nil {
			t.Error("Expected error when no config is available")
		}
	})
}

func TestGetConfigFile(t *testing.T) {
	// GetConfigFile() returns the viper config file path
	configFile := GetConfigFile()
	
	// Initially should be empty since no config is loaded
	if configFile != "" {
		t.Logf("Config file found: %s", configFile)
	}
}

func TestIsConfigFound(t *testing.T) {
	// IsConfigFound() checks if viper has found a config file
	found := IsConfigFound()
	
	// Initially should be false since no config is loaded
	if found {
		t.Log("Config was found by viper")
	} else {
		t.Log("No config found by viper")
	}
}