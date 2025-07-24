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
		Repositories: RepositoryManagement{
			Apt: []AptRepository{
				{Name: "python39", PPA: "deadsnakes/ppa"},
			},
			Flatpak: []FlatpakRepository{
				{Name: "flathub", URL: "https://flathub.org/repo/flathub.flatpakrepo"},
			},
		},
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

// Repository validation tests

func TestValidateRepositories_ValidAPTRepositories(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Repositories: RepositoryManagement{
			Apt: []AptRepository{
				{Name: "python39", PPA: "deadsnakes/ppa"},
				{Name: "docker", URI: "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable", Key: "https://download.docker.com/linux/ubuntu/gpg.asc"},
				{Name: "nodejs", URI: "deb https://deb.nodesource.com/node_16.x focal main", Key: "0x9FD3B784BC1C6FC31A8A0A1C1655A0AB68576280"},
			},
		},
	}
	
	result := Validate(config, "config.yaml")
	
	if result.HasErrors() {
		t.Errorf("validation should pass for valid APT repositories, got errors: %v", result.Errors)
	}
}

func TestValidateRepositories_ValidFlatpakRepositories(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Repositories: RepositoryManagement{
			Flatpak: []FlatpakRepository{
				{Name: "flathub", URL: "https://flathub.org/repo/flathub.flatpakrepo"},
				{Name: "kde", URL: "https://distribute.kde.org/kdeapps.flatpakrepo", User: true},
				{Name: "gnome", URL: "https://nightly.gnome.org/gnome-nightly.flatpakrepo"},
			},
		},
	}
	
	result := Validate(config, "config.yaml")
	
	if result.HasErrors() {
		t.Errorf("validation should pass for valid Flatpak repositories, got errors: %v", result.Errors)
	}
}

func TestValidateRepositories_InvalidAPTRepositories(t *testing.T) {
	tests := []struct {
		name     string
		repo     AptRepository
		errorMsg string
	}{
		{
			name:     "missing ppa and uri",
			repo:     AptRepository{Name: "test"},
			errorMsg: "missing repository configuration",
		},
		{
			name:     "both ppa and uri",
			repo:     AptRepository{Name: "test", PPA: "user/repo", URI: "deb https://example.com/repo stable main"},
			errorMsg: "conflicting repository configuration",
		},
		{
			name:     "invalid ppa format",
			repo:     AptRepository{Name: "test", PPA: "invalid-ppa-format"},
			errorMsg: "invalid PPA format",
		},
		{
			name:     "invalid uri format",
			repo:     AptRepository{Name: "test", URI: "invalid-uri-format"},
			errorMsg: "invalid repository URI",
		},
		{
			name:     "invalid gpg key",
			repo:     AptRepository{Name: "test", PPA: "user/repo", Key: "invalid-key"},
			errorMsg: "invalid GPG key reference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Version: "1.0",
				Repositories: RepositoryManagement{
					Apt: []AptRepository{tt.repo},
				},
			}
			
			result := Validate(config, "config.yaml")
			
			if !result.HasErrors() {
				t.Errorf("validation should fail for %s", tt.name)
				return
			}
			
			found := false
			for _, err := range result.Errors {
				if strings.Contains(err.Title, tt.errorMsg) {
					found = true
					break
				}
			}
			
			if !found {
				t.Errorf("expected error message containing '%s', got errors: %v", tt.errorMsg, result.Errors)
			}
		})
	}
}

func TestValidateRepositories_InvalidFlatpakRepositories(t *testing.T) {
	tests := []struct {
		name     string
		repo     FlatpakRepository
		errorMsg string
	}{
		{
			name:     "missing url",
			repo:     FlatpakRepository{Name: "test"},
			errorMsg: "missing repository URL",
		},
		{
			name:     "invalid url format",
			repo:     FlatpakRepository{Name: "test", URL: "ftp://invalid.com/repo"},
			errorMsg: "invalid repository URL",
		},
		{
			name:     "invalid remote name",
			repo:     FlatpakRepository{Name: "test@invalid!", URL: "https://example.com/repo.flatpakrepo"},
			errorMsg: "invalid remote name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Version: "1.0",
				Repositories: RepositoryManagement{
					Flatpak: []FlatpakRepository{tt.repo},
				},
			}
			
			result := Validate(config, "config.yaml")
			
			if !result.HasErrors() {
				t.Errorf("validation should fail for %s", tt.name)
				return
			}
			
			found := false
			for _, err := range result.Errors {
				if strings.Contains(err.Title, tt.errorMsg) {
					found = true
					break
				}
			}
			
			if !found {
				t.Errorf("expected error message containing '%s', got errors: %v", tt.errorMsg, result.Errors)
			}
		})
	}
}

func TestIsValidPPAFormat(t *testing.T) {
	tests := []struct {
		ppa      string
		expected bool
		name     string
	}{
		{"deadsnakes/ppa", true, "valid ppa"},
		{"user/repo", true, "simple user/repo"},
		{"ubuntu-toolchain-r/test", true, "ppa with hyphens"},
		{"user", false, "missing slash"},
		{"user/", false, "missing repo name"},
		{"/repo", false, "missing user name"},
		{"user@invalid/repo", false, "invalid characters in user"},
		{"user/repo@invalid", false, "invalid characters in repo"},
		{"", false, "empty string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidPPAFormat(tt.ppa)
			if result != tt.expected {
				t.Errorf("isValidPPAFormat(%s) = %v, expected %v", tt.ppa, result, tt.expected)
			}
		})
	}
}

func TestIsValidAPTRepositoryURI(t *testing.T) {
	tests := []struct {
		uri      string
		expected bool
		name     string
	}{
		{"deb https://example.com/repo stable main", true, "valid deb uri"},
		{"deb-src https://example.com/repo stable main", true, "valid deb-src uri"},
		{"deb [arch=amd64] https://example.com/repo stable main", true, "deb with architecture"},
		{"deb http://example.com/repo stable main", true, "deb with http"},
		{"deb file:///path/to/repo stable main", true, "deb with file protocol"},
		{"rpm https://example.com/repo", false, "non-deb format"},
		{"https://example.com/repo", false, "missing deb prefix"},
		{"deb example.com/repo stable main", false, "missing protocol"},
		{"", false, "empty string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidAPTRepositoryURI(tt.uri)
			if result != tt.expected {
				t.Errorf("isValidAPTRepositoryURI(%s) = %v, expected %v", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestIsValidGPGKeyReference(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
		name     string
	}{
		{"https://example.com/key.gpg", true, "valid gpg url"},
		{"https://example.com/key.asc", true, "valid asc url"},
		{"0x1234567890ABCDEF", true, "valid key id with 0x prefix"},
		{"1234567890ABCDEF", true, "valid key id without prefix"},
		{"9FD3B784BC1C6FC31A8A0A1C1655A0AB68576280", true, "long key id"},
		{"12345678", true, "short key id"},
		{"http://example.com/key.gpg", false, "http instead of https"},
		{"https://example.com/key.txt", false, "wrong file extension"},
		{"0xGHIJKLMN", false, "invalid hex characters"},
		{"12345", false, "too short key id"},
		{"", false, "empty string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidGPGKeyReference(tt.key)
			if result != tt.expected {
				t.Errorf("isValidGPGKeyReference(%s) = %v, expected %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestIsValidFlatpakRepositoryURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
		name     string
	}{
		{"https://flathub.org/repo/flathub.flatpakrepo", true, "valid flatpakrepo url"},
		{"https://example.com/repo/test.flatpakrepo", true, "custom flatpakrepo url"},
		{"https://example.com/repo/", true, "repository directory url"},
		{"http://localhost/repo/test.flatpakrepo", true, "http for local testing"},
		{"ftp://example.com/repo.flatpakrepo", false, "ftp protocol not allowed"},
		{"https://example.com/file.txt", false, "not a flatpakrepo file"},
		{"example.com/repo.flatpakrepo", false, "missing protocol"},
		{"", false, "empty string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidFlatpakRepositoryURL(tt.url)
			if result != tt.expected {
				t.Errorf("isValidFlatpakRepositoryURL(%s) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestIsValidFlatpakRemoteName(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
		testName string
	}{
		{"flathub", true, "simple name"},
		{"kde-apps", true, "name with hyphen"},
		{"gnome_nightly", true, "name with underscore"},
		{"test123", true, "name with numbers"},
		{"Test", true, "name with capital letters"},
		{"@invalid", false, "name starting with special character"},
		{"test@invalid", false, "name with invalid character"},
		{"test-", false, "name ending with hyphen"},
		{"", false, "empty string"},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			result := isValidFlatpakRemoteName(tt.name)
			if result != tt.expected {
				t.Errorf("isValidFlatpakRemoteName(%s) = %v, expected %v", tt.name, result, tt.expected)
			}
		})
	}
}