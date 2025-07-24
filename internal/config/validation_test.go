package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidate_ValidConfig(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a valid source file
	sourceFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(sourceFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	
	config := &Config{
		Version: "1.0",
		Files: map[string]File{
			"test": {
				Source:      "test.txt",
				Destination: "~/test.txt",
				Mode:        "644",
				Backup:      true,
			},
		},
		DConf: DConfConfig{
			Settings: map[string]string{
				"/test/setting": "'value'",
			},
		},
	}
	
	result := Validate(config, filepath.Join(tempDir, "config.yaml"))
	
	if result.HasErrors() {
		t.Errorf("validation should pass for valid config, got errors: %v", result.Errors)
	}
	
	if !result.Valid {
		t.Error("valid config should be marked as valid")
	}
}

func TestValidate_MissingVersion(t *testing.T) {
	config := &Config{
		// Version intentionally missing
		Files: map[string]File{},
	}
	
	result := Validate(config, "test.yaml")
	
	if !result.HasErrors() {
		t.Error("validation should fail when version is missing")
	}
	
	if result.Valid {
		t.Error("config without version should be marked as invalid")
	}
	
	// Check specific error
	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Title, "missing version") {
			found = true
			break
		}
	}
	if !found {
		t.Error("should have specific error about missing version")
	}
}

func TestValidate_InvalidVersion(t *testing.T) {
	config := &Config{
		Version: "not-a-version",
		Files:   map[string]File{},
	}
	
	result := Validate(config, "test.yaml")
	
	if !result.HasErrors() {
		t.Error("validation should fail for invalid version format")
	}
	
	// Check specific error
	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Title, "invalid version") {
			found = true
			break
		}
	}
	if !found {
		t.Error("should have specific error about invalid version format")
	}
}

func TestValidate_FileValidation(t *testing.T) {
	tempDir := t.TempDir()
	
	tests := []struct {
		name        string
		file        File
		shouldError bool
		errorTitle  string
	}{
		{
			name: "missing source",
			file: File{
				Destination: "~/test.txt",
			},
			shouldError: true,
			errorTitle:  "missing source path",
		},
		{
			name: "missing destination",
			file: File{
				Source: "test.txt",
			},
			shouldError: true,
			errorTitle:  "missing destination path",
		},
		{
			name: "non-existent source file",
			file: File{
				Source:      "nonexistent.txt",
				Destination: "~/test.txt",
			},
			shouldError: true,
			errorTitle:  "source file not found",
		},
		{
			name: "invalid file mode",
			file: File{
				Source:      "test.txt",
				Destination: "~/test.txt",
				Mode:        "999",
			},
			shouldError: true,
			errorTitle:  "invalid file mode",
		},
		{
			name: "unsafe destination path",
			file: File{
				Source:      "test.txt",
				Destination: "../../../etc/passwd",
			},
			shouldError: true,
			errorTitle:  "unsafe destination path",
		},
	}
	
	// Create a test source file for valid tests
	sourceFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(sourceFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Version: "1.0",
				Files: map[string]File{
					"test": tt.file,
				},
			}
			
			result := Validate(config, filepath.Join(tempDir, "config.yaml"))
			
			if tt.shouldError {
				if !result.HasErrors() {
					t.Errorf("validation should fail for %s", tt.name)
				}
				
				// Check for specific error title
				found := false
				for _, err := range result.Errors {
					if strings.Contains(err.Title, tt.errorTitle) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("should have error with title containing '%s'", tt.errorTitle)
				}
			} else {
				if result.HasErrors() {
					t.Errorf("validation should pass for %s, got errors: %v", tt.name, result.Errors)
				}
			}
		})
	}
}

func TestValidate_DConfValidation(t *testing.T) {
	tests := []struct {
		name        string
		settings    map[string]string
		shouldError bool
		errorTitle  string
	}{
		{
			name: "valid dconf path",
			settings: map[string]string{
				"/org/gnome/desktop/interface/gtk-theme": "'Adwaita'",
			},
			shouldError: false,
		},
		{
			name: "invalid dconf path - missing leading slash",
			settings: map[string]string{
				"org/gnome/desktop/interface/gtk-theme": "'Adwaita'",
			},
			shouldError: true,
			errorTitle:  "invalid dconf path",
		},
		{
			name: "malformed dconf path - double slashes",
			settings: map[string]string{
				"/org//gnome/desktop": "'value'",
			},
			shouldError: true,
			errorTitle:  "malformed dconf path",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Version: "1.0",
				DConf: DConfConfig{
					Settings: tt.settings,
				},
			}
			
			result := Validate(config, "test.yaml")
			
			if tt.shouldError {
				if !result.HasErrors() {
					t.Errorf("validation should fail for %s", tt.name)
				}
				
				// Check for specific error title
				found := false
				for _, err := range result.Errors {
					if strings.Contains(err.Title, tt.errorTitle) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("should have error with title containing '%s'", tt.errorTitle)
				}
			} else {
				if result.HasErrors() {
					t.Errorf("validation should pass for %s, got errors: %v", tt.name, result.Errors)
				}
			}
		})
	}
}

func TestValidate_PackageValidation(t *testing.T) {
	tests := []struct {
		name        string
		packages    PackageManagement
		shouldError bool
		errorTitle  string
	}{
		{
			name: "valid package names",
			packages: PackageManagement{
				Apt: []PackageEntry{
					{Name: "git"},
					{Name: "curl"},
				},
				Flatpak: []PackageEntry{
					{Name: "org.mozilla.firefox"},
				},
				Snap: []PackageEntry{
					{Name: "discord"},
				},
			},
			shouldError: false,
		},
		{
			name: "empty package name",
			packages: PackageManagement{
				Apt: []PackageEntry{
					{Name: ""}, // Empty name
				},
			},
			shouldError: true,
			errorTitle:  "empty package name",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Version:  "1.0",
				Packages: tt.packages,
			}
			
			result := Validate(config, "test.yaml")
			
			if tt.shouldError {
				if !result.HasErrors() {
					t.Errorf("validation should fail for %s", tt.name)
				}
				
				// Check for specific error title
				found := false
				for _, err := range result.Errors {
					if strings.Contains(err.Title, tt.errorTitle) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("should have error with title containing '%s'", tt.errorTitle)
				}
			} else {
				if result.HasErrors() {
					t.Errorf("validation should pass for %s, got errors: %v", tt.name, result.Errors)
				}
			}
		})
	}
}

func TestValidationResult_Add(t *testing.T) {
	result := &ValidationResult{Valid: true}
	
	// Add error
	result.Add(ValidationError{
		Type:  "error",
		Title: "test error",
	})
	
	if result.Valid {
		t.Error("result should be invalid after adding error")
	}
	
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors))
	}
	
	// Add warning
	result.Add(ValidationError{
		Type:  "warning",
		Title: "test warning",
	})
	
	if len(result.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(result.Warnings))
	}
}

func TestIsValidFileMode(t *testing.T) {
	tests := []struct {
		mode  string
		valid bool
	}{
		{"644", true},
		{"755", true},
		{"600", true},
		{"777", true},
		{"999", false}, // Invalid octal
		{"64", false},  // Too short
		{"64444", false}, // Too long
		{"abc", false}, // Not numeric
	}
	
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			result := isValidFileMode(tt.mode)
			if result != tt.valid {
				t.Errorf("isValidFileMode(%s) = %v, want %v", tt.mode, result, tt.valid)
			}
		})
	}
}

func TestIsOverlyPermissive(t *testing.T) {
	tests := []struct {
		mode       string
		permissive bool
	}{
		{"644", false},
		{"755", false},
		{"600", false},
		{"666", true}, // World writable
		{"777", true}, // World writable
		{"646", true}, // Group writable includes 6
		{"676", true}, // World writable
	}
	
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			result := isOverlyPermissive(tt.mode)
			if result != tt.permissive {
				t.Errorf("isOverlyPermissive(%s) = %v, want %v", tt.mode, result, tt.permissive)
			}
		})
	}
}

func TestValidate_AptPackageValidation(t *testing.T) {
	tests := []struct {
		name        string
		packages    []PackageEntry
		expectError bool
		errorTitle  string
		description string
	}{
		{
			name: "valid apt package names",
			packages: []PackageEntry{
				{Name: "git"},
				{Name: "nodejs"},
				{Name: "build-essential"},
				{Name: "libssl-dev"},
				{Name: "python3.9"},
			},
			expectError: false,
			description: "Should accept valid apt package names",
		},
		{
			name: "valid local .deb file paths",
			packages: []PackageEntry{
				{Name: "/home/user/packages/custom.deb"},
				{Name: "./local/package.deb"},
				{Name: "subfolder/app.deb"},
			},
			expectError: false,
			description: "Should accept valid local .deb file paths",
		},
		{
			name: "invalid apt package names",
			packages: []PackageEntry{
				{Name: "Invalid_Package"}, // Uppercase and underscore
				{Name: "package with spaces"},
			},
			expectError: true,
			errorTitle:  "invalid package name",
			description: "Should reject invalid apt package names",
		},
		{
			name: "empty package name",
			packages: []PackageEntry{
				{Name: ""},
			},
			expectError: true,
			errorTitle:  "empty package name",
			description: "Should reject empty package names",
		},
		{
			name: "invalid .deb file paths",
			packages: []PackageEntry{
				{Name: "package.deb"}, // No path separator
			},
			expectError: true,
			errorTitle:  "invalid package name",
			description: "Should reject invalid .deb file paths",
		},
		{
			name: "path traversal in .deb",
			packages: []PackageEntry{
				{Name: "../../../etc/passwd.deb"}, // Path traversal
			},
			expectError: true,
			errorTitle:  "invalid package name",
			description: "Should reject .deb paths with traversal",
		},
		{
			name: "mixed valid packages",
			packages: []PackageEntry{
				{Name: "git"},
				{Name: "./custom.deb"},
				{Name: "curl", Flags: []string{"--install-suggests"}},
			},
			expectError: false,
			description: "Should accept mix of repository packages and local .deb files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Version: "1.0",
				Packages: PackageManagement{
					Apt: tt.packages,
				},
			}

			result := Validate(config, "test-config.yaml")

			if tt.expectError {
				if !result.HasErrors() {
					t.Errorf("Expected validation errors for %s, but got none", tt.description)
				} else {
					// Check if we got the expected error type
					foundExpectedError := false
					for _, err := range result.Errors {
						if err.Title == tt.errorTitle {
							foundExpectedError = true
							break
						}
					}
					if !foundExpectedError {
						t.Errorf("Expected error with title '%s' for %s, but got errors: %v", 
							tt.errorTitle, tt.description, result.Errors)
					}
				}
			} else {
				if result.HasErrors() {
					t.Errorf("Expected no validation errors for %s, but got: %v", 
						tt.description, result.Errors)
				}
			}
		})
	}
}

func TestIsValidDebFilePath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
		name     string
	}{
		{"/home/user/package.deb", true, "absolute path .deb file"},
		{"./local/package.deb", true, "relative path .deb file"},
		{"subfolder/custom.deb", true, "subdirectory .deb file"},
		{"package.deb", false, "no path separator"},
		{"/path/to/file.txt", false, "not a .deb file"},
		{"../../../etc/passwd.deb", false, "path traversal attempt"},
		{"", false, "empty path"},
		{".deb", false, "just extension"},
		{"/home/user/.deb", false, "no filename before extension"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidDebFilePath(tt.path)
			if result != tt.expected {
				t.Errorf("isValidDebFilePath(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}