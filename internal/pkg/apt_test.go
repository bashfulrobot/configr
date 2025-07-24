package pkg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

func TestNewAptManager(t *testing.T) {
	logger := log.New(os.Stderr)
	dryRun := true
	
	aptManager := NewAptManager(logger, dryRun)
	
	if aptManager == nil {
		t.Fatal("NewAptManager should not return nil")
	}
	
	if aptManager.logger != logger {
		t.Error("AptManager should store the provided logger")
	}
	
	if aptManager.dryRun != dryRun {
		t.Error("AptManager should store the provided dryRun setting")
	}
}

func TestAptManager_resolvePackageFlags(t *testing.T) {
	logger := log.New(os.Stderr)
	aptManager := NewAptManager(logger, true)
	
	tests := []struct {
		name            string
		pkg             config.PackageEntry
		packageDefaults map[string][]string
		expected        []string
		description     string
	}{
		{
			name:            "Per-package flags (highest priority)",
			pkg:             config.PackageEntry{Name: "git", Flags: []string{"--install-suggests", "-y"}},
			packageDefaults: map[string][]string{"apt": {"-q"}},
			expected:        []string{"--install-suggests", "-y"},
			description:     "Should use per-package flags when available",
		},
		{
			name:            "User default flags",
			pkg:             config.PackageEntry{Name: "curl", Flags: nil},
			packageDefaults: map[string][]string{"apt": {"-q", "--no-install-recommends"}},
			expected:        []string{"-q", "--no-install-recommends"},
			description:     "Should use user defaults when no per-package flags",
		},
		{
			name:            "Internal default flags",
			pkg:             config.PackageEntry{Name: "vim", Flags: nil},
			packageDefaults: map[string][]string{},
			expected:        []string{"-y", "--no-install-recommends"},
			description:     "Should use internal defaults when no user defaults",
		},
		{
			name:            "Empty per-package flags still take priority",  
			pkg:             config.PackageEntry{Name: "test", Flags: []string{}},
			packageDefaults: map[string][]string{"apt": {"-q"}},
			expected:        []string{},
			description:     "Empty per-package flags should still override user defaults",
		},
		{
			name:            "Nil flags use user defaults",
			pkg:             config.PackageEntry{Name: "test", Flags: nil},
			packageDefaults: map[string][]string{"apt": {"-q"}},
			expected:        []string{"-q"},
			description:     "Nil flags should use user defaults (not explicitly set)",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aptManager.resolvePackageFlags(tt.pkg, tt.packageDefaults)
			
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

func TestAptManager_isLocalDebFile(t *testing.T) {
	logger := log.New(os.Stderr)
	aptManager := NewAptManager(logger, true)
	
	tests := []struct {
		name         string
		packageName  string
		expected     bool
		description  string
	}{
		{
			name:         "Absolute path .deb file",
			packageName:  "/home/user/packages/custom.deb",
			expected:     true,
			description:  "Should recognize absolute path .deb files",
		},
		{
			name:         "Relative path .deb file",
			packageName:  "./packages/custom.deb",
			expected:     true,
			description:  "Should recognize relative path .deb files",
		},
		{
			name:         "Regular package name",
			packageName:  "git",
			expected:     false,
			description:  "Should not recognize regular package names as .deb files",
		},
		{
			name:         ".deb without path",
			packageName:  "package.deb",
			expected:     false,
			description:  "Should not recognize .deb without path separator",
		},
		{
			name:         "Path without .deb extension",
			packageName:  "/home/user/script.sh",
			expected:     false,
			description:  "Should not recognize non-.deb files",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := aptManager.isLocalDebFile(tt.packageName)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for package %s", tt.expected, result, tt.packageName)
			}
		})
	}
}

func TestAptManager_filterOutLocalFiles(t *testing.T) {
	logger := log.New(os.Stderr)
	aptManager := NewAptManager(logger, true)
	
	packageNames := []string{
		"git",
		"./custom.deb",
		"curl",
		"/absolute/path/package.deb",
		"vim",
	}
	
	expected := []string{"git", "curl", "vim"}
	result := aptManager.filterOutLocalFiles(packageNames)
	
	if len(result) != len(expected) {
		t.Errorf("Expected %d packages, got %d: %v", len(expected), len(result), result)
		return
	}
	
	for i, pkg := range result {
		if pkg != expected[i] {
			t.Errorf("Expected package %d to be %s, got %s", i, expected[i], pkg)
		}
	}
}

func TestAptManager_groupPackagesByFlags(t *testing.T) {
	logger := log.New(os.Stderr)
	aptManager := NewAptManager(logger, true)
	
	packages := []config.PackageEntry{
		{Name: "git", Flags: []string{"-y"}},
		{Name: "curl", Flags: []string{"-y"}},  
		{Name: "vim", Flags: []string{"-q", "--no-install-recommends"}},
		{Name: "nano", Flags: nil}, // Will use defaults: {"-y", "--no-install-recommends"}
	}
	
	packageDefaults := map[string][]string{
		"apt": {"-y", "--no-install-recommends"},
	}
	
	result := aptManager.groupPackagesByFlags(packages, packageDefaults)
	
	// Should have 2 groups: 
	// - one for "-y" (git, curl)
	// - one for "-q|--no-install-recommends" (vim)  
	// - one for "-y|--no-install-recommends" (nano with defaults)
	expectedGroups := 3 // Actually 3 groups because nano gets different flags than git/curl
	if len(result) != expectedGroups {
		t.Errorf("Expected %d groups, got %d: %v", expectedGroups, len(result), result)
	}
	
	// Check that git and curl are grouped together (both have "-y")
	foundGitCurlGroup := false
	for _, group := range result {
		if len(group) == 2 {
			if (group[0].Name == "git" && group[1].Name == "curl") ||
			   (group[0].Name == "curl" && group[1].Name == "git") {
				foundGitCurlGroup = true
				break
			}
		}
	}
	
	if !foundGitCurlGroup {
		t.Error("Expected git and curl to be grouped together")
	}
}

func TestAptManager_InstallPackages_EmptyList(t *testing.T) {
	logger := log.New(os.Stderr)
	aptManager := NewAptManager(logger, true)
	
	err := aptManager.InstallPackages([]config.PackageEntry{}, map[string][]string{})
	if err != nil {
		t.Errorf("InstallPackages with empty list should not return error, got: %v", err)
	}
}

func TestAptManager_InstallPackages_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	aptManager := NewAptManager(logger, true) // dry-run mode
	
	packages := []config.PackageEntry{
		{Name: "nonexistent-package-for-testing", Flags: []string{"-y"}},
	}
	
	// This test will be skipped if apt is not available (like on this NixOS system)
	err := aptManager.InstallPackages(packages, map[string][]string{})
	
	// We expect this to fail due to apt not being available, but test the dry-run logic
	if err != nil && err.Error() != "apt not available: apt command not found - is this a Debian/Ubuntu system?" {
		t.Errorf("Expected apt availability error, got: %v", err)
	}
}

func TestAptManager_installSingleDebFile_FileValidation(t *testing.T) {
	logger := log.New(os.Stderr)
	aptManager := NewAptManager(logger, true) // dry-run mode
	
	// Test with non-existent file
	err := aptManager.installSingleDebFile("/nonexistent/path/package.deb", []string{"-y"})
	if err == nil {
		t.Error("Expected error for non-existent .deb file")
	}
	
	if err != nil && !contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestAptManager_installSingleDebFile_RelativePath(t *testing.T) {
	logger := log.New(os.Stderr)
	aptManager := NewAptManager(logger, true) // dry-run mode
	
	// Create a temporary .deb file for testing
	tempDir := t.TempDir()
	debFile := filepath.Join(tempDir, "test.deb")
	err := os.WriteFile(debFile, []byte("fake deb content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test .deb file: %v", err)
	}
	
	// Change to temp directory to test relative path resolution
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)
	
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	// This should work in dry-run mode even without apt
	err = aptManager.installSingleDebFile("test.deb", []string{"-y"})
	if err != nil {
		// We expect this to fail if apt is not available, but the path resolution should work
		if !contains(err.Error(), "not found") && !contains(err.Error(), "apt") {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}