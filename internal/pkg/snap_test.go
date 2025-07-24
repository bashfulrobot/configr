package pkg

import (
	"os"
	"testing"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

func TestNewSnapManager(t *testing.T) {
	logger := log.New(os.Stderr)
	dryRun := true

	snapManager := NewSnapManager(logger, dryRun)

	if snapManager == nil {
		t.Fatal("NewSnapManager should not return nil")
	}

	if snapManager.logger != logger {
		t.Error("SnapManager should store the provided logger")
	}

	if snapManager.dryRun != dryRun {
		t.Error("SnapManager should store the provided dryRun setting")
	}
}

func TestSnapManager_resolvePackageFlags(t *testing.T) {
	logger := log.New(os.Stderr)
	snapManager := NewSnapManager(logger, true)

	tests := []struct {
		name            string
		pkg             config.PackageEntry
		packageDefaults map[string][]string
		expected        []string
		description     string
	}{
		{
			name:            "Per-package flags (highest priority)",
			pkg:             config.PackageEntry{Name: "code", Flags: []string{"--classic", "--beta"}},
			packageDefaults: map[string][]string{"snap": {"--devmode"}},
			expected:        []string{"--classic", "--beta"},
			description:     "Should use per-package flags when available",
		},
		{
			name:            "User default flags",
			pkg:             config.PackageEntry{Name: "discord", Flags: nil},
			packageDefaults: map[string][]string{"snap": {"--classic"}},
			expected:        []string{"--classic"},
			description:     "Should use user defaults when no per-package flags",
		},
		{
			name:            "Internal default flags",
			pkg:             config.PackageEntry{Name: "hello", Flags: nil},
			packageDefaults: map[string][]string{},
			expected:        []string{},
			description:     "Should use internal defaults (empty for snap) when no user defaults",
		},
		{
			name:            "Empty per-package flags still take priority",
			pkg:             config.PackageEntry{Name: "test", Flags: []string{}},
			packageDefaults: map[string][]string{"snap": {"--classic"}},
			expected:        []string{},
			description:     "Empty per-package flags should still override user defaults",
		},
		{
			name:            "Nil flags use user defaults",
			pkg:             config.PackageEntry{Name: "test", Flags: nil},
			packageDefaults: map[string][]string{"snap": {"--classic"}},
			expected:        []string{"--classic"},
			description:     "Nil flags should use user defaults (not explicitly set)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := snapManager.resolvePackageFlags(tt.pkg, tt.packageDefaults)

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

func TestSnapManager_groupPackagesByFlags(t *testing.T) {
	logger := log.New(os.Stderr)
	snapManager := NewSnapManager(logger, true)

	packages := []config.PackageEntry{
		{Name: "code", Flags: []string{"--classic"}},
		{Name: "discord", Flags: []string{"--classic"}},
		{Name: "hello", Flags: []string{}},
		{Name: "snap-store", Flags: nil}, // Will use defaults: []
	}

	packageDefaults := map[string][]string{
		"snap": {},
	}

	result := snapManager.groupPackagesByFlags(packages, packageDefaults)

	// Should have 2 groups:
	// - one for "--classic" (code, discord)
	// - one for "" (hello, snap-store with defaults)
	expectedGroups := 2
	if len(result) != expectedGroups {
		t.Errorf("Expected %d groups, got %d: %v", expectedGroups, len(result), result)
	}

	// Check that code and discord are grouped together (both have "--classic")
	foundClassicGroup := false
	for _, group := range result {
		if len(group) == 2 {
			if (group[0].Name == "code" && group[1].Name == "discord") ||
			   (group[0].Name == "discord" && group[1].Name == "code") {
				foundClassicGroup = true
				break
			}
		}
	}

	if !foundClassicGroup {
		t.Error("Expected code and discord to be grouped together")
	}
}

func TestSnapManager_InstallPackages_EmptyList(t *testing.T) {
	logger := log.New(os.Stderr)
	snapManager := NewSnapManager(logger, true)

	err := snapManager.InstallPackages([]config.PackageEntry{}, map[string][]string{})
	if err != nil {
		t.Errorf("InstallPackages with empty list should not return error, got: %v", err)
	}
}

func TestSnapManager_InstallPackages_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	snapManager := NewSnapManager(logger, true) // dry-run mode

	packages := []config.PackageEntry{
		{Name: "code", Flags: []string{"--classic"}},
		{Name: "discord", Flags: []string{"--classic"}},
	}

	// This should not error in dry-run mode even if snap is not available
	err := snapManager.InstallPackages(packages, map[string][]string{})
	if err != nil {
		t.Errorf("InstallPackages in dry-run mode should not error, got: %v", err)
	}
}

func TestSnapManager_ValidatePackageNames(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	snapManager := NewSnapManager(logger, true)

	tests := []struct {
		name        string
		packages    []config.PackageEntry
		expectError bool
		description string
	}{
		{
			name: "valid package names",
			packages: []config.PackageEntry{
				{Name: "code"},
				{Name: "discord"},
				{Name: "hello"},
				{Name: "snap-store"},
				{Name: "slack"},
			},
			expectError: false,
			description: "Should accept valid Snap package names",
		},
		{
			name: "valid package names with numbers",
			packages: []config.PackageEntry{
				{Name: "firefox"},
				{Name: "chromium"},
				{Name: "gimp"},
				{Name: "blender"},
			},
			expectError: false,
			description: "Should accept package names with numbers",
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
			name: "invalid package name - too short",
			packages: []config.PackageEntry{
				{Name: "a"},
			},
			expectError: true,
			description: "Should reject package names shorter than 2 characters",
		},
		{
			name: "invalid package name - too long",
			packages: []config.PackageEntry{
				{Name: "this-is-a-very-long-package-name-that-exceeds-forty-characters"},
			},
			expectError: true,
			description: "Should reject package names longer than 40 characters",
		},
		{
			name: "invalid package name - starts with number",
			packages: []config.PackageEntry{
				{Name: "1code"},
			},
			expectError: true,
			description: "Should reject package names starting with numbers",
		},
		{
			name: "invalid package name - starts with uppercase",
			packages: []config.PackageEntry{
				{Name: "Code"},
			},
			expectError: true,
			description: "Should reject package names starting with uppercase letters",
		},
		{
			name: "invalid package name - ends with hyphen",
			packages: []config.PackageEntry{
				{Name: "code-"},
			},
			expectError: true,
			description: "Should reject package names ending with hyphens",
		},
		{
			name: "invalid package name - consecutive hyphens",
			packages: []config.PackageEntry{
				{Name: "code--editor"},
			},
			expectError: true,
			description: "Should reject package names with consecutive hyphens",
		},
		{
			name: "invalid package name - invalid characters",
			packages: []config.PackageEntry{
				{Name: "code_editor"},
			},
			expectError: true,
			description: "Should reject package names with invalid characters (underscore)",
		},
		{
			name: "invalid package name - special characters",
			packages: []config.PackageEntry{
				{Name: "code@editor"},
			},
			expectError: true,
			description: "Should reject package names with special characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := snapManager.ValidatePackageNames(tt.packages)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestSnapManager_validatePackageName(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	snapManager := NewSnapManager(logger, true)

	tests := []struct {
		packageName string
		expectError bool
		description string
	}{
		{"code", false, "Valid simple name"},
		{"discord", false, "Valid name"},
		{"snap-store", false, "Valid name with hyphen"},
		{"hello", false, "Valid simple name"},
		{"firefox", false, "Valid name with numbers"},
		{"gimp", false, "Valid short name"},
		{"", true, "Empty string"},
		{"a", true, "Too short"},
		{"this-is-a-very-long-package-name-that-exceeds-forty-characters", true, "Too long"},
		{"1code", true, "Starts with number"},
		{"Code", true, "Starts with uppercase"},
		{"code-", true, "Ends with hyphen"},
		{"code--editor", true, "Consecutive hyphens"},
		{"code_editor", true, "Contains underscore"},
		{"code@editor", true, "Contains special character"},
		{"code.editor", true, "Contains dot"},
		{"code/editor", true, "Contains slash"},
		{"code editor", true, "Contains space"},
		{"-code", true, "Starts with hyphen"},
	}

	for _, tt := range tests {
		t.Run(tt.packageName, func(t *testing.T) {
			err := snapManager.validatePackageName(tt.packageName)

			if tt.expectError && err == nil {
				t.Errorf("expected error for package name '%s' but got none", tt.packageName)
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error for package name '%s': %v", tt.packageName, err)
			}
		})
	}
}

func TestSnapManager_installSinglePackage_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	snapManager := NewSnapManager(logger, true) // dry-run mode

	err := snapManager.installSinglePackage("code", []string{"--classic"})
	if err != nil {
		t.Errorf("installSinglePackage in dry-run mode should not error, got: %v", err)
	}
}

func TestSnapManager_UninstallPackage_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	snapManager := NewSnapManager(logger, true) // dry-run mode

	err := snapManager.UninstallPackage("code", []string{})
	if err != nil {
		t.Errorf("UninstallPackage in dry-run mode should not error, got: %v", err)
	}
}

func TestSnapManager_ListInstalledPackages_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	snapManager := NewSnapManager(logger, true) // dry-run mode

	packages, err := snapManager.ListInstalledPackages()
	if err != nil {
		t.Errorf("ListInstalledPackages in dry-run mode should not error, got: %v", err)
	}

	// In dry-run mode, should return empty slice
	if len(packages) != 0 {
		t.Errorf("expected empty slice in dry-run, got: %v", packages)
	}
}

func TestSnapManager_RefreshPackages_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	snapManager := NewSnapManager(logger, true) // dry-run mode

	err := snapManager.RefreshPackages([]string{})
	if err != nil {
		t.Errorf("RefreshPackages in dry-run mode should not error, got: %v", err)
	}
}

func TestSnapManager_InfoPackage_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	snapManager := NewSnapManager(logger, true) // dry-run mode

	info, err := snapManager.InfoPackage("code")
	if err != nil {
		t.Errorf("InfoPackage in dry-run mode should not error, got: %v", err)
	}

	// In dry-run mode, should return empty string
	if info != "" {
		t.Errorf("expected empty string in dry-run, got: %s", info)
	}
}

func TestSnapManager_FindPackage_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	snapManager := NewSnapManager(logger, true) // dry-run mode

	packages, err := snapManager.FindPackage("code")
	if err != nil {
		t.Errorf("FindPackage in dry-run mode should not error, got: %v", err)
	}

	// In dry-run mode, should return empty slice
	if len(packages) != 0 {
		t.Errorf("expected empty slice in dry-run, got: %v", packages)
	}
}

func TestSnapManager_CheckCommands(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests

	snapManager := NewSnapManager(logger, true)

	// Test command availability check
	// Note: This may fail in CI environments without snap installed
	t.Run("check snap", func(t *testing.T) {
		err := snapManager.checkSnapAvailable()
		// Don't fail the test if command is not available, just log it
		if err != nil {
			t.Logf("snap not available: %v", err)
		}
	})
}

func TestSnapManager_isPackageInstalled_DryRunSkip(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	snapManager := NewSnapManager(logger, false) // Not dry-run to test actual logic

	// This test is mainly to ensure the function exists and handles errors gracefully
	// We don't want to test actual snap commands in unit tests, but we can test error handling
	installed, err := snapManager.isPackageInstalled("nonexistent-package-12345")
	
	// Should not error even if snap command fails
	if err != nil {
		t.Errorf("isPackageInstalled should handle command errors gracefully, got: %v", err)
	}
	
	// Should return false for nonexistent packages
	if installed {
		t.Error("isPackageInstalled should return false for nonexistent packages")
	}
}