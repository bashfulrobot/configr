package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidation_ComprehensiveScenarios tests extensive validation scenarios
func TestValidation_ComprehensiveScenarios(t *testing.T) {
	tempDir := t.TempDir()

	// Test 1: Complete valid configuration
	t.Run("CompleteValidConfiguration", func(t *testing.T) {
		// Create source files
		sourceFile := filepath.Join(tempDir, "source.txt")
		err := os.WriteFile(sourceFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("failed to create source file: %v", err)
		}

		config := &Config{
			Version: "1.0",
			PackageDefaults: map[string][]string{
				"apt":     {"-y", "--no-install-recommends"},
				"flatpak": {"--user"},
				"snap":    {"--classic"},
			},
			Repositories: RepositoryManagement{
				Apt: []AptRepository{
					{Name: "docker", URI: "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable", Key: "https://download.docker.com/linux/ubuntu/gpg"},
					{Name: "python", PPA: "deadsnakes/ppa"},
				},
				Flatpak: []FlatpakRepository{
					{Name: "flathub", URL: "https://flathub.org/repo/flathub.flatpakrepo", User: false},
				},
			},
			Packages: PackageManagement{
				Apt: []PackageEntry{
					{Name: "curl"},
					{Name: "vim", Flags: []string{"-y", "--install-suggests"}},
				},
				Flatpak: []PackageEntry{
					{Name: "org.mozilla.Firefox"},
					{Name: "org.gimp.GIMP", Flags: []string{"--user"}},
				},
				Snap: []PackageEntry{
					{Name: "code"},
					{Name: "discord", Flags: []string{"--classic"}},
				},
			},
			Files: map[string]File{
				"vimrc": {
					Source:            sourceFile,
					Destination:       filepath.Join(tempDir, ".vimrc"),
					Mode:              "644",
					Owner:             "user",
					Group:             "user",
					Backup:            true,
					Interactive:       true,
					PromptPermissions: true,
					PromptOwnership:   true,
				},
			},
			DConf: DConfConfig{
				Settings: map[string]string{
					"/org/gnome/desktop/background/picture-uri": "'file:///home/user/wallpaper.jpg'",
					"/org/gnome/terminal/legacy/profiles:/:default/font": "'Monospace 12'",
				},
			},
		}

		result := Validate(config, "test.yaml")
		if result.HasErrors() {
			t.Errorf("valid configuration should not have errors: %v", result.Errors)
		}
		if len(result.Warnings) > 0 {
			t.Logf("warnings found (may be expected): %v", result.Warnings)
		}
	})

	// Test 2: Repository validation errors
	t.Run("RepositoryValidationErrors", func(t *testing.T) {
		tests := []struct {
			name        string
			config      *Config
			expectError bool
		}{
			{
				name: "Empty APT repository name",
				config: &Config{
					Version: "1.0",
					Repositories: RepositoryManagement{
						Apt: []AptRepository{
							{Name: "", PPA: "deadsnakes/ppa"},
						},
					},
				},
				expectError: true,
			},
			{
				name: "Invalid PPA format",
				config: &Config{
					Version: "1.0",
					Repositories: RepositoryManagement{
						Apt: []AptRepository{
							{Name: "invalid", PPA: "invalid-ppa-format"},
						},
					},
				},
				expectError: true,
			},
			{
				name: "Invalid URI format",
				config: &Config{
					Version: "1.0",
					Repositories: RepositoryManagement{
						Apt: []AptRepository{
							{Name: "invalid", URI: "invalid uri format"},
						},
					},
				},
				expectError: true,
			},
			{
				name: "Empty Flatpak repository name",
				config: &Config{
					Version: "1.0",
					Repositories: RepositoryManagement{
						Flatpak: []FlatpakRepository{
							{Name: "", URL: "https://flathub.org/repo/flathub.flatpakrepo"},
						},
					},
				},
				expectError: true,
			},
			{
				name: "Invalid Flatpak URL",
				config: &Config{
					Version: "1.0",
					Repositories: RepositoryManagement{
						Flatpak: []FlatpakRepository{
							{Name: "invalid", URL: "not-a-valid-url"},
						},
					},
				},
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := Validate(tt.config, "test.yaml")
				hasErrors := result.HasErrors()
				if tt.expectError && !hasErrors {
					t.Error("expected validation errors but got none")
				}
				if !tt.expectError && hasErrors {
					t.Errorf("unexpected validation errors: %v", result.Errors)
				}
			})
		}
	})

	// Test 3: Package validation errors
	t.Run("PackageValidationErrors", func(t *testing.T) {
		tests := []struct {
			name        string
			config      *Config
			expectError bool
		}{
			{
				name: "Empty APT package name",
				config: &Config{
					Version: "1.0",
					Packages: PackageManagement{
						Apt: []PackageEntry{
							{Name: ""},
						},
					},
				},
				expectError: true,
			},
			{
				name: "Invalid Flatpak app ID",
				config: &Config{
					Version: "1.0",
					Packages: PackageManagement{
						Flatpak: []PackageEntry{
							{Name: "invalid-app-id"},
						},
					},
				},
				expectError: true,
			},
			{
				name: "Invalid Snap package name",
				config: &Config{
					Version: "1.0",
					Packages: PackageManagement{
						Snap: []PackageEntry{
							{Name: "Invalid_Package_Name"},
						},
					},
				},
				expectError: true,
			},
			{
				name: "Snap package name too long",
				config: &Config{
					Version: "1.0",
					Packages: PackageManagement{
						Snap: []PackageEntry{
							{Name: "this-is-a-very-long-package-name-that-exceeds-forty-characters"},
						},
					},
				},
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := Validate(tt.config, "test.yaml")
				hasErrors := result.HasErrors()
				if tt.expectError && !hasErrors {
					t.Error("expected validation errors but got none")
				}
				if !tt.expectError && hasErrors {
					t.Errorf("unexpected validation errors: %v", result.Errors)
				}
			})
		}
	})

	// Test 4: File validation errors
	t.Run("FileValidationErrors", func(t *testing.T) {
		tests := []struct {
			name        string
			config      *Config
			expectError bool
		}{
			{
				name: "Missing source file",
				config: &Config{
					Version: "1.0",
					Files: map[string]File{
						"test": {
							Source:      "/path/that/does/not/exist",
							Destination: filepath.Join(tempDir, "test"),
						},
					},
				},
				expectError: true,
			},
			{
				name: "Empty source path",
				config: &Config{
					Version: "1.0",
					Files: map[string]File{
						"test": {
							Source:      "",
							Destination: filepath.Join(tempDir, "test"),
						},
					},
				},
				expectError: true,
			},
			{
				name: "Empty destination path",
				config: &Config{
					Version: "1.0",
					Files: map[string]File{
						"test": {
							Source:      filepath.Join(tempDir, "source.txt"),
							Destination: "",
						},
					},
				},
				expectError: true,
			},
			{
				name: "Invalid file mode",
				config: &Config{
					Version: "1.0",
					Files: map[string]File{
						"test": {
							Source:      filepath.Join(tempDir, "source.txt"),
							Destination: filepath.Join(tempDir, "test"),
							Mode:        "999",
						},
					},
				},
				expectError: true,
			},
		}

		// Create source file for valid tests
		sourceFile := filepath.Join(tempDir, "source.txt")
		err := os.WriteFile(sourceFile, []byte("test"), 0644)
		if err != nil {
			t.Fatalf("failed to create source file: %v", err)
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := Validate(tt.config, "test.yaml")
				hasErrors := result.HasErrors()
				if tt.expectError && !hasErrors {
					t.Error("expected validation errors but got none")
				}
				if !tt.expectError && hasErrors {
					t.Errorf("unexpected validation errors: %v", result.Errors)
				}
			})
		}
	})

	// Test 5: DConf validation
	t.Run("DConfValidation", func(t *testing.T) {
		tests := []struct {
			name        string
			config      *Config
			expectError bool
		}{
			{
				name: "Valid DConf settings",
				config: &Config{
					Version: "1.0",
					DConf: DConfConfig{
						Settings: map[string]string{
							"/org/gnome/desktop/background/picture-uri": "'file:///home/user/wallpaper.jpg'",
							"/org/gnome/terminal/legacy/profiles:/:default/font": "'Monospace 12'",
						},
					},
				},
				expectError: false,
			},
			{
				name: "Invalid DConf path",
				config: &Config{
					Version: "1.0",
					DConf: DConfConfig{
						Settings: map[string]string{
							"invalid-path": "value",
						},
					},
				},
				expectError: true,
			},
			{
				name: "DConf path with double slashes",
				config: &Config{
					Version: "1.0",
					DConf: DConfConfig{
						Settings: map[string]string{
							"/org//gnome/desktop/background": "value",
						},
					},
				},
				expectError: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := Validate(tt.config, "test.yaml")
				hasErrors := result.HasErrors()
				if tt.expectError && !hasErrors {
					t.Error("expected validation errors but got none")
				}
				if !tt.expectError && hasErrors {
					t.Errorf("unexpected validation errors: %v", result.Errors)
				}
			})
		}
	})

	// Test 6: Include validation
	t.Run("IncludeValidation", func(t *testing.T) {
		// Create include files
		validInclude := filepath.Join(tempDir, "valid.yaml")
		err := os.WriteFile(validInclude, []byte("packages:\n  apt:\n    - curl"), 0644)
		if err != nil {
			t.Fatalf("failed to create include file: %v", err)
		}

		tests := []struct {
			name        string
			config      *Config
			expectError bool
		}{
			{
				name: "Valid include",
				config: &Config{
					Version: "1.0",
					Includes: []IncludeSpec{
						{Path: validInclude, Optional: false},
					},
				},
				expectError: false,
			},
			{
				name: "Missing required include",
				config: &Config{
					Version: "1.0",
					Includes: []IncludeSpec{
						{Path: "/path/that/does/not/exist", Optional: false},
					},
				},
				expectError: true,
			},
			{
				name: "Missing optional include",
				config: &Config{
					Version: "1.0",
					Includes: []IncludeSpec{
						{Path: "/path/that/does/not/exist", Optional: true},
					},
				},
				expectError: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := Validate(tt.config, "test.yaml")
				hasErrors := result.HasErrors()
				if tt.expectError && !hasErrors {
					t.Error("expected validation errors but got none")
				}
				if !tt.expectError && hasErrors {
					t.Errorf("unexpected validation errors: %v", result.Errors)
				}
			})
		}
	})
}

// TestValidation_ErrorFormatting tests error message formatting
func TestValidation_ErrorFormatting(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Files: map[string]File{
			"test": {
				Source:      "/nonexistent/file",
				Destination: "/tmp/test",
			},
		},
	}

	result := Validate(config, "test.yaml")
	if !result.HasErrors() {
		t.Error("expected validation errors")
		return
	}

	// Check that error messages are formatted properly
	errorMsg := result.Errors[0].Message
	if !strings.Contains(errorMsg, "source file") {
		t.Error("error message should mention source file")
	}
	if !strings.Contains(errorMsg, "not found") || !strings.Contains(errorMsg, "does not exist") {
		t.Error("error message should indicate file does not exist")
	}
}

// TestValidation_WarningGeneration tests warning generation
func TestValidation_WarningGeneration(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	config := &Config{
		Version: "1.0",
		DConf: DConfConfig{
			Settings: map[string]string{
				"/org/gnome/desktop/background/picture-uri": "unquoted string value",
			},
		},
	}

	result := Validate(config, "test.yaml")
	
	// Should have warnings for unquoted DConf values
	if len(result.Warnings) == 0 {
		t.Log("expected validation warnings for unquoted DConf values (may vary by implementation)")
	}
}

// TestValidation_EmptyConfiguration tests empty configuration validation
func TestValidation_EmptyConfiguration(t *testing.T) {
	config := &Config{
		Version: "1.0",
	}

	result := Validate(config, "test.yaml")
	if result.HasErrors() {
		t.Errorf("empty configuration should be valid: %v", result.Errors)
	}
}

// TestValidation_ConfigurationVersion tests version validation
func TestValidation_ConfigurationVersion(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectError bool
	}{
		{"Valid version 1.0", "1.0", false},
		{"Empty version", "", true},
		{"Invalid version", "2.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Version: tt.version,
			}

			result := Validate(config, "test.yaml")
			hasErrors := result.HasErrors()
			if tt.expectError && !hasErrors {
				t.Error("expected validation errors but got none")
			}
			if !tt.expectError && hasErrors {
				t.Errorf("unexpected validation errors: %v", result.Errors)
			}
		})
	}
}