package pkg

import (
	"os"
	"testing"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

func TestNewFlatpakManager(t *testing.T) {
	logger := log.New(os.Stderr)
	dryRun := true

	flatpakManager := NewFlatpakManager(logger, dryRun)

	if flatpakManager == nil {
		t.Fatal("NewFlatpakManager should not return nil")
	}

	if flatpakManager.logger != logger {
		t.Error("FlatpakManager should store the provided logger")
	}

	if flatpakManager.dryRun != dryRun {
		t.Error("FlatpakManager should store the provided dryRun setting")
	}
}

func TestFlatpakManager_resolvePackageFlags(t *testing.T) {
	logger := log.New(os.Stderr)
	flatpakManager := NewFlatpakManager(logger, true)

	tests := []struct {
		name            string
		pkg             config.PackageEntry
		packageDefaults map[string][]string
		expected        []string
		description     string
	}{
		{
			name:            "Per-package flags (highest priority)",
			pkg:             config.PackageEntry{Name: "org.mozilla.Firefox", Flags: []string{"--user", "--or-update"}},
			packageDefaults: map[string][]string{"flatpak": {"--system"}},
			expected:        []string{"--user", "--or-update"},
			description:     "Should use per-package flags when available",
		},
		{
			name:            "User default flags",
			pkg:             config.PackageEntry{Name: "com.spotify.Client", Flags: nil},
			packageDefaults: map[string][]string{"flatpak": {"--user", "--or-update"}},
			expected:        []string{"--user", "--or-update"},
			description:     "Should use user defaults when no per-package flags",
		},
		{
			name:            "Internal default flags",
			pkg:             config.PackageEntry{Name: "org.gimp.GIMP", Flags: nil},
			packageDefaults: map[string][]string{},
			expected:        []string{"--system", "--assumeyes"},
			description:     "Should use internal defaults when no user defaults",
		},
		{
			name:            "Empty per-package flags still take priority",
			pkg:             config.PackageEntry{Name: "org.test.App", Flags: []string{}},
			packageDefaults: map[string][]string{"flatpak": {"--user"}},
			expected:        []string{},
			description:     "Empty per-package flags should still override user defaults",
		},
		{
			name:            "Nil flags use user defaults",
			pkg:             config.PackageEntry{Name: "org.test.App", Flags: nil},
			packageDefaults: map[string][]string{"flatpak": {"--user"}},
			expected:        []string{"--user"},
			description:     "Nil flags should use user defaults (not explicitly set)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flatpakManager.resolvePackageFlags(tt.pkg, tt.packageDefaults)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d flags, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			for i, flag := range result {
				if flag != tt.expected[i] {
					t.Errorf("Expected flag %d to be %s, got %s", i, tt.expected[i], flag)
				}
			}
		})
	}
}

func TestFlatpakManager_groupPackagesByFlags(t *testing.T) {
	logger := log.New(os.Stderr)
	flatpakManager := NewFlatpakManager(logger, true)

	packages := []config.PackageEntry{
		{Name: "org.mozilla.Firefox", Flags: []string{"--user"}},
		{Name: "com.spotify.Client", Flags: []string{"--user"}},
		{Name: "org.gimp.GIMP", Flags: []string{"--system", "--assumeyes"}},
		{Name: "org.blender.Blender", Flags: nil}, // Will use defaults: {"--system", "--assumeyes"}
	}

	packageDefaults := map[string][]string{
		"flatpak": {"--system", "--assumeyes"},
	}

	result := flatpakManager.groupPackagesByFlags(packages, packageDefaults)

	// Should have 2 groups:
	// - one for "--user" (Firefox, Spotify)
	// - one for "--system|--assumeyes" (GIMP, Blender with defaults)
	expectedGroups := 2
	if len(result) != expectedGroups {
		t.Errorf("Expected %d groups, got %d: %v", expectedGroups, len(result), result)
	}

	// Check that Firefox and Spotify are grouped together (both have "--user")
	foundUserGroup := false
	for _, group := range result {
		if len(group) == 2 {
			if (group[0].Name == "org.mozilla.Firefox" && group[1].Name == "com.spotify.Client") ||
			   (group[0].Name == "com.spotify.Client" && group[1].Name == "org.mozilla.Firefox") {
				foundUserGroup = true
				break
			}
		}
	}

	if !foundUserGroup {
		t.Error("Expected Firefox and Spotify to be grouped together")
	}
}

func TestFlatpakManager_InstallPackages_EmptyList(t *testing.T) {
	logger := log.New(os.Stderr)
	flatpakManager := NewFlatpakManager(logger, true)

	err := flatpakManager.InstallPackages([]config.PackageEntry{}, map[string][]string{})
	if err != nil {
		t.Errorf("InstallPackages with empty list should not return error, got: %v", err)
	}
}

func TestFlatpakManager_InstallPackages_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	flatpakManager := NewFlatpakManager(logger, true) // dry-run mode

	packages := []config.PackageEntry{
		{Name: "org.mozilla.Firefox", Flags: []string{"--user"}},
		{Name: "com.spotify.Client", Flags: []string{"--system"}},
	}

	// This should not error in dry-run mode even if flatpak is not available
	err := flatpakManager.InstallPackages(packages, map[string][]string{})
	if err != nil {
		t.Errorf("InstallPackages in dry-run mode should not error, got: %v", err)
	}
}

func TestFlatpakManager_ValidatePackageNames(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	flatpakManager := NewFlatpakManager(logger, true)

	tests := []struct {
		name        string
		packages    []config.PackageEntry
		expectError bool
		description string
	}{
		{
			name: "valid package names",
			packages: []config.PackageEntry{
				{Name: "org.mozilla.Firefox"},
				{Name: "com.spotify.Client"},
				{Name: "org.gimp.GIMP"},
				{Name: "org.blender.Blender"},
			},
			expectError: false,
			description: "Should accept valid Flatpak application IDs",
		},
		{
			name: "valid package names with numbers and hyphens",
			packages: []config.PackageEntry{
				{Name: "org.kde.krita-4"},
				{Name: "com.github.user_app"},
				{Name: "io.github.project-name"},
			},
			expectError: false,
			description: "Should accept application IDs with numbers, hyphens, and underscores",
		},
		{
			name: "invalid package name - no domain",
			packages: []config.PackageEntry{
				{Name: "firefox"},
			},
			expectError: true,
			description: "Should reject package names without reverse domain notation",
		},
		{
			name: "invalid package name - empty",
			packages: []config.PackageEntry{
				{Name: ""},
			},
			expectError: true,
			description: "Should reject empty package names",
		},
		{
			name: "invalid package name - empty part",
			packages: []config.PackageEntry{
				{Name: "org..Firefox"},
			},
			expectError: true,
			description: "Should reject package names with empty parts",
		},
		{
			name: "invalid package name - invalid characters",
			packages: []config.PackageEntry{
				{Name: "org.mozilla.Fire@fox"},
			},
			expectError: true,
			description: "Should reject package names with invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := flatpakManager.ValidatePackageNames(tt.packages)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFlatpakManager_validatePackageName(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	flatpakManager := NewFlatpakManager(logger, true)

	tests := []struct {
		packageName string
		expectError bool
		description string
	}{
		{"org.mozilla.Firefox", false, "Valid reverse domain notation"},
		{"com.spotify.Client", false, "Valid commercial domain"},
		{"io.github.project", false, "Valid GitHub domain"},
		{"org.kde.krita-4", false, "Valid with hyphen and number"},
		{"com.github.user_app", false, "Valid with underscore"},
		{"firefox", true, "Missing domain parts"},
		{"", true, "Empty string"},
		{"org..Firefox", true, "Empty domain part"},
		{"org.mozilla.Fire@fox", true, "Invalid character (@)"},
		{"org.mozilla.Fire fox", true, "Invalid character (space)"},
		{"org.mozilla.Fire/fox", true, "Invalid character (/)"},
		{"org", true, "Only one part"},
	}

	for _, tt := range tests {
		t.Run(tt.packageName, func(t *testing.T) {
			err := flatpakManager.validatePackageName(tt.packageName)

			if tt.expectError && err == nil {
				t.Errorf("expected error for package name '%s' but got none", tt.packageName)
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error for package name '%s': %v", tt.packageName, err)
			}
		})
	}
}

func TestFlatpakManager_UninstallPackage_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	flatpakManager := NewFlatpakManager(logger, true) // dry-run mode

	err := flatpakManager.UninstallPackage("org.mozilla.Firefox", []string{"--user"})
	if err != nil {
		t.Errorf("UninstallPackage in dry-run mode should not error, got: %v", err)
	}
}

func TestFlatpakManager_ListInstalledPackages_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	flatpakManager := NewFlatpakManager(logger, true) // dry-run mode

	packages, err := flatpakManager.ListInstalledPackages()
	if err != nil {
		t.Errorf("ListInstalledPackages in dry-run mode should not error, got: %v", err)
	}

	// In dry-run mode, should return empty slice
	if len(packages) != 0 {
		t.Errorf("expected empty slice in dry-run, got: %v", packages)
	}
}

func TestFlatpakManager_UpdatePackages_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	flatpakManager := NewFlatpakManager(logger, true) // dry-run mode

	err := flatpakManager.UpdatePackages([]string{"--user"})
	if err != nil {
		t.Errorf("UpdatePackages in dry-run mode should not error, got: %v", err)
	}
}

func TestFlatpakManager_CheckCommands(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests

	flatpakManager := NewFlatpakManager(logger, true)

	// Test command availability check
	// Note: This may fail in CI environments without flatpak installed
	t.Run("check flatpak", func(t *testing.T) {
		err := flatpakManager.checkFlatpakAvailable()
		// Don't fail the test if command is not available, just log it
		if err != nil {
			t.Logf("flatpak not available: %v", err)
		}
	})
}

func TestFlatpakManager_isPackageInstalledInScope_DryRunSkip(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	flatpakManager := NewFlatpakManager(logger, false) // Not dry-run to test actual logic

	// This test is mainly to ensure the function exists and handles errors gracefully
	// We don't want to test actual flatpak commands in unit tests, but we can test error handling
	installed, err := flatpakManager.isPackageInstalledInScope("org.nonexistent.App", "--system")
	
	// Should not error even if flatpak command fails
	if err != nil {
		t.Errorf("isPackageInstalledInScope should handle command errors gracefully, got: %v", err)
	}
	
	// Should return false for nonexistent packages
	if installed {
		t.Error("isPackageInstalledInScope should return false for nonexistent packages")
	}
}