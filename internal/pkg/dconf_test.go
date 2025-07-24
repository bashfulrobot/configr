package pkg

import (
	"os"
	"testing"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

func TestNewDConfManager(t *testing.T) {
	logger := log.New(os.Stderr)
	dryRun := true

	dconfManager := NewDConfManager(logger, dryRun)

	if dconfManager == nil {
		t.Fatal("NewDConfManager should not return nil")
	}

	if dconfManager.logger != logger {
		t.Error("DConfManager should store the provided logger")
	}

	if dconfManager.dryRun != dryRun {
		t.Error("DConfManager should store the provided dryRun setting")
	}
}

func TestDConfManager_ApplySettings(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests

	tests := []struct {
		name        string
		settings    map[string]string
		dryRun      bool
		expectError bool
	}{
		{
			name:        "empty settings",
			settings:    map[string]string{},
			dryRun:      true,
			expectError: false,
		},
		{
			name: "basic dconf settings - dry run",
			settings: map[string]string{
				"/org/gnome/desktop/interface/gtk-theme":       "'Adwaita-dark'",
				"/org/gnome/desktop/interface/icon-theme":      "'Adwaita'",
				"/org/gnome/desktop/wm/preferences/button-layout": "'close,minimize,maximize:'",
			},
			dryRun:      true,
			expectError: false,
		},
		{
			name: "mixed value types - dry run",
			settings: map[string]string{
				"/org/gnome/desktop/interface/gtk-theme":           "'Adwaita-dark'",
				"/org/gnome/desktop/interface/clock-show-seconds":  "true",
				"/org/gnome/desktop/interface/cursor-blink-timeout": "1200",
				"/org/gnome/terminal/legacy/profiles:/:default/background-color": "'rgb(23,20,33)'",
			},
			dryRun:      true,
			expectError: false,
		},
		{
			name: "application-specific settings - dry run",
			settings: map[string]string{
				"/org/gnome/nautilus/preferences/default-folder-viewer": "'icon-view'",
				"/org/gnome/gedit/preferences/editor/scheme":            "'oblivion'",
				"/apps/guake/general/window-height":                     "40",
			},
			dryRun:      true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := NewDConfManager(logger, tt.dryRun)
			dconfConfig := config.DConfConfig{Settings: tt.settings}
			err := dm.ApplySettings(dconfConfig)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDConfManager_SetSetting_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests

	tests := []struct {
		name        string
		path        string
		value       string
		expectError bool
	}{
		{
			name:        "gnome interface setting",
			path:        "/org/gnome/desktop/interface/gtk-theme",
			value:       "'Adwaita-dark'",
			expectError: false,
		},
		{
			name:        "boolean setting",
			path:        "/org/gnome/desktop/interface/clock-show-seconds",
			value:       "true",
			expectError: false,
		},
		{
			name:        "numeric setting",
			path:        "/org/gnome/desktop/interface/cursor-blink-timeout",
			value:       "1200",
			expectError: false,
		},
		{
			name:        "array setting",
			path:        "/org/gnome/desktop/input-sources/sources",
			value:       "[('xkb', 'us')]",
			expectError: false,
		},
		{
			name:        "complex path with colons",
			path:        "/org/gnome/terminal/legacy/profiles:/:b1dcc9dd-5262-4d8d-a863-c897e6d979b9/background-color",
			value:       "'rgb(23,20,33)'",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := NewDConfManager(logger, true) // Always dry run for unit tests
			err := dm.setSetting(tt.path, tt.value)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDConfManager_GetSetting_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests

	dm := NewDConfManager(logger, true) // Always dry run for unit tests

	// Test getting a setting - should not error in dry run
	value, err := dm.GetSetting("/org/gnome/desktop/interface/gtk-theme")
	if err != nil {
		t.Errorf("unexpected error in dry run mode: %v", err)
	}

	// In dry run, should return empty string
	if value != "" {
		t.Errorf("expected empty string in dry run, got: %s", value)
	}
}

func TestDConfManager_ResetSetting_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests

	dm := NewDConfManager(logger, true) // Always dry run for unit tests

	// Test resetting a setting - should not error in dry run
	err := dm.ResetSetting("/org/gnome/desktop/interface/gtk-theme")
	if err != nil {
		t.Errorf("unexpected error in dry run mode: %v", err)
	}
}

func TestDConfManager_ListSettings_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests

	dm := NewDConfManager(logger, true) // Always dry run for unit tests

	// Test listing settings - should not error in dry run
	settings, err := dm.ListSettings("/org/gnome/desktop/interface/")
	if err != nil {
		t.Errorf("unexpected error in dry run mode: %v", err)
	}

	// In dry run, should return empty slice
	if len(settings) != 0 {
		t.Errorf("expected empty slice in dry run, got: %v", settings)
	}
}

func TestDConfManager_DumpSettings_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests

	dm := NewDConfManager(logger, true) // Always dry run for unit tests

	// Test dumping settings - should not error in dry run
	settings, err := dm.DumpSettings("/org/gnome/desktop/")
	if err != nil {
		t.Errorf("unexpected error in dry run mode: %v", err)
	}

	// In dry run, should return empty map
	if len(settings) != 0 {
		t.Errorf("expected empty map in dry run, got: %v", settings)
	}
}

func TestDConfManager_ValidateSettings(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests

	tests := []struct {
		name        string
		settings    map[string]string
		expectError bool
		description string
	}{
		{
			name:        "valid settings",
			settings: map[string]string{
				"/org/gnome/desktop/interface/gtk-theme": "'Adwaita-dark'",
				"/org/gnome/desktop/interface/clock-show-seconds": "true",
				"/org/gnome/desktop/interface/cursor-blink-timeout": "1200",
			},
			expectError: false,
			description: "Should accept valid dconf paths and values",
		},
		{
			name:        "invalid path - no leading slash",
			settings: map[string]string{
				"org/gnome/desktop/interface/gtk-theme": "'Adwaita-dark'",
			},
			expectError: true,
			description: "Should reject paths without leading slash",
		},
		{
			name:        "invalid path - double slashes",
			settings: map[string]string{
				"/org/gnome//desktop/interface/gtk-theme": "'Adwaita-dark'",
			},
			expectError: true,
			description: "Should reject paths with double slashes",
		},
		{
			name:        "empty settings",
			settings:    map[string]string{},
			expectError: false,
			description: "Should handle empty settings gracefully",
		},
		{
			name:        "complex valid paths",
			settings: map[string]string{
				"/org/gnome/terminal/legacy/profiles:/:default/background-color": "'rgb(23,20,33)'",
				"/apps/guake/general/window-height": "40",
				"/org/gnome/desktop/input-sources/sources": "[('xkb', 'us')]",
			},
			expectError: false,
			description: "Should handle complex but valid dconf paths",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := NewDConfManager(logger, true)
			dconfConfig := config.DConfConfig{Settings: tt.settings}
			err := dm.ValidateSettings(dconfConfig)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDConfManager_CheckCommands(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests

	dm := NewDConfManager(logger, true)

	// Test command availability check
	// Note: This may fail in CI environments without dconf installed
	t.Run("check dconf", func(t *testing.T) {
		err := dm.checkDConfAvailable()
		// Don't fail the test if command is not available, just log it
		if err != nil {
			t.Logf("dconf not available: %v", err)
		}
	})
}

func TestIsNumericValue(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"123", true},
		{"123.45", true},
		{"-123", true},
		{"+123", true},
		{"1.23e-4", true},
		{"1.23E+4", true},
		{"'123'", false},
		{"true", false},
		{"false", false},
		{"abc", false},
		{"", false},
		{"12.34.56", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := isNumericValue(tt.value)
			if result != tt.expected {
				t.Errorf("isNumericValue(%q) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestIsBooleanValue(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{"true", true},
		{"false", true},
		{"True", true},
		{"False", true},
		{"TRUE", true},
		{"FALSE", true},
		{"1", false},
		{"0", false},
		{"'true'", false},
		{"yes", false},
		{"no", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := isBooleanValue(tt.value)
			if result != tt.expected {
				t.Errorf("isBooleanValue(%q) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}
}