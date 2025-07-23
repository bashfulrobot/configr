package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Load reads and parses the configuration file using Viper with include support and validation
func Load() (*Config, error) {
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Use the new include-aware loader
	config, err := LoadWithIncludes()
	if err != nil {
		return nil, err
	}

	// Validate the configuration
	result := Validate(config, viper.ConfigFileUsed())
	if result.HasErrors() {
		return nil, &ValidationFailedError{
			Result: result,
		}
	}

	// Show warnings if any (but don't fail)
	if len(result.Warnings) > 0 {
		fmt.Print(FormatValidationResultSimple(&ValidationResult{
			Warnings: result.Warnings,
			Valid:    true,
		}))
	}

	return config, nil
}

// GetConfigFile returns the path of the config file being used
func GetConfigFile() string {
	return viper.ConfigFileUsed()
}

// IsConfigFound returns true if a config file was found and loaded
func IsConfigFound() bool {
	return viper.ConfigFileUsed() != ""
}