package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewAdvancedLoader(t *testing.T) {
	loader := NewAdvancedLoader()
	
	if loader == nil {
		t.Fatal("NewAdvancedLoader returned nil")
	}
	
	if loader.visited == nil {
		t.Error("visited map not initialized")
	}
	
	if loader.osName != runtime.GOOS {
		t.Errorf("Expected osName to be %s, got %s", runtime.GOOS, loader.osName)
	}
	
	if loader.hostname == "" {
		t.Error("hostname should not be empty")
	}
}

func TestResolveGlobPattern(t *testing.T) {
	loader := NewAdvancedLoader()
	tmpDir := t.TempDir()
	
	// Create test files
	testFiles := []string{
		"config1.yaml",
		"config2.yaml",
		"packages.yaml",
		"other.txt",
	}
	
	for _, file := range testFiles {
		path := filepath.Join(tmpDir, file)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}
	
	tests := []struct {
		name     string
		pattern  string
		optional bool
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "match all yaml files",
			pattern:  "*.yaml",
			optional: false,
			wantLen:  3,
			wantErr:  false,
		},
		{
			name:     "match specific pattern",
			pattern:  "config*.yaml",
			optional: false,
			wantLen:  2,
			wantErr:  false,
		},
		{
			name:     "no matches - optional",
			pattern:  "nonexistent*.yaml",
			optional: true,
			wantLen:  0,
			wantErr:  false,
		},
		{
			name:     "no matches - required",
			pattern:  "nonexistent*.yaml",
			optional: false,
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "invalid pattern",
			pattern:  "[",
			optional: false,
			wantLen:  0,
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := loader.resolveGlobPattern(tmpDir, tt.pattern, tt.optional)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveGlobPattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if len(result) != tt.wantLen {
				t.Errorf("resolveGlobPattern() returned %d files, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestEvaluateCondition(t *testing.T) {
	loader := NewAdvancedLoader()
	
	// Create a temporary file for file_exists tests
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	tests := []struct {
		name      string
		condition IncludeCondition
		want      bool
	}{
		{
			name: "os condition matches",
			condition: IncludeCondition{
				Type:  "os",
				Value: runtime.GOOS,
			},
			want: true,
		},
		{
			name: "os condition doesn't match",
			condition: IncludeCondition{
				Type:  "os",
				Value: "nonexistent-os",
			},
			want: false,
		},
		{
			name: "file_exists condition - file exists",
			condition: IncludeCondition{
				Type:  "file_exists",
				Value: tmpFile,
			},
			want: true,
		},
		{
			name: "file_exists condition - file doesn't exist",
			condition: IncludeCondition{
				Type:  "file_exists",
				Value: "/nonexistent/file.txt",
			},
			want: false,
		},
		{
			name: "env condition - var exists",
			condition: IncludeCondition{
				Type:  "env",
				Value: "PATH",
			},
			want: true,
		},
		{
			name: "env condition - var doesn't exist",
			condition: IncludeCondition{
				Type:  "env",
				Value: "NONEXISTENT_VAR_12345",
			},
			want: false,
		},
		{
			name: "hostname condition",
			condition: IncludeCondition{
				Type:  "hostname",
				Value: loader.hostname,
			},
			want: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.evaluateCondition(tt.condition)
			if result != tt.want {
				t.Errorf("evaluateCondition() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestCompareValues(t *testing.T) {
	loader := NewAdvancedLoader()
	
	tests := []struct {
		name     string
		actual   string
		expected string
		operator string
		want     bool
	}{
		{
			name:     "equals - match",
			actual:   "linux",
			expected: "linux",
			operator: "equals",
			want:     true,
		},
		{
			name:     "equals - no match",
			actual:   "linux",
			expected: "windows",
			operator: "equals",
			want:     false,
		},
		{
			name:     "contains - match",
			actual:   "ubuntu-desktop",
			expected: "ubuntu",
			operator: "contains",
			want:     true,
		},
		{
			name:     "contains - no match",
			actual:   "fedora",
			expected: "ubuntu",
			operator: "contains",
			want:     false,
		},
		{
			name:     "matches - regex match",
			actual:   "test123",
			expected: "test\\d+",
			operator: "matches",
			want:     true,
		},
		{
			name:     "matches - regex no match",
			actual:   "testabc",
			expected: "test\\d+",
			operator: "matches",
			want:     false,
		},
		{
			name:     "not_equals - match",
			actual:   "linux",
			expected: "windows",
			operator: "not_equals",
			want:     true,
		},
		{
			name:     "not_contains - match",
			actual:   "fedora",
			expected: "ubuntu",
			operator: "not_contains",
			want:     true,
		},
		{
			name:     "default operator (equals)",
			actual:   "test",
			expected: "test",
			operator: "",
			want:     true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := loader.compareValues(tt.actual, tt.expected, tt.operator)
			if result != tt.want {
				t.Errorf("compareValues() = %v, want %v", result, tt.want)
			}
		})
	}
}

func TestValidateIncludeSpec(t *testing.T) {
	loader := NewAdvancedLoader()
	
	tests := []struct {
		name    string
		spec    IncludeSpec
		wantErr bool
	}{
		{
			name: "valid spec with path",
			spec: IncludeSpec{
				Path: "config.yaml",
			},
			wantErr: false,
		},
		{
			name: "valid spec with glob pattern",
			spec: IncludeSpec{
				Path: "*.yaml",
			},
			wantErr: false,
		},
		{
			name: "invalid spec - no path",
			spec: IncludeSpec{
				Description: "test",
			},
			wantErr: true,
		},
		{
			name: "valid spec with conditions",
			spec: IncludeSpec{
				Path: "config.yaml",
				Conditions: []IncludeCondition{
					{
						Type:  "os",
						Value: "linux",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid spec - invalid condition",
			spec: IncludeSpec{
				Path: "config.yaml",
				Conditions: []IncludeCondition{
					{
						Type: "invalid_type",
						Value: "test",
					},
				},
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.ValidateIncludeSpec(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIncludeSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCondition(t *testing.T) {
	loader := NewAdvancedLoader()
	
	tests := []struct {
		name      string
		condition IncludeCondition
		wantErr   bool
	}{
		{
			name: "valid condition",
			condition: IncludeCondition{
				Type:  "os",
				Value: "linux",
			},
			wantErr: false,
		},
		{
			name: "invalid condition - no type",
			condition: IncludeCondition{
				Value: "linux",
			},
			wantErr: true,
		},
		{
			name: "invalid condition - no value",
			condition: IncludeCondition{
				Type: "os",
			},
			wantErr: true,
		},
		{
			name: "invalid condition - invalid type",
			condition: IncludeCondition{
				Type:  "invalid_type",
				Value: "test",
			},
			wantErr: true,
		},
		{
			name: "invalid condition - invalid operator",
			condition: IncludeCondition{
				Type:     "os",
				Value:    "linux",
				Operator: "invalid_operator",
			},
			wantErr: true,
		},
		{
			name: "valid condition with operator",
			condition: IncludeCondition{
				Type:     "hostname",
				Value:    "test",
				Operator: "contains",
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loader.validateCondition(tt.condition)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCondition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetSystemInfo(t *testing.T) {
	loader := NewAdvancedLoader()
	
	info := loader.GetSystemInfo()
	
	// Check required fields
	if info["os"] != runtime.GOOS {
		t.Errorf("Expected os to be %s, got %s", runtime.GOOS, info["os"])
	}
	
	if info["goos"] != runtime.GOOS {
		t.Errorf("Expected goos to be %s, got %s", runtime.GOOS, info["goos"])
	}
	
	if info["goarch"] != runtime.GOARCH {
		t.Errorf("Expected goarch to be %s, got %s", runtime.GOARCH, info["goarch"])
	}
	
	if info["hostname"] == "" {
		t.Error("hostname should not be empty")
	}
	
	// Check that at least some environment variables are present
	if len(info) < 4 {
		t.Errorf("Expected at least 4 system info entries, got %d", len(info))
	}
}

func TestLoadConfigurationAdvanced(t *testing.T) {
	loader := NewAdvancedLoader()
	tmpDir := t.TempDir()
	
	// Create main config file
	mainConfig := `version: "1.0"
packages:
  apt:
    - "curl"
includes:
  - path: "packages.yaml"
    description: "Additional packages"
`
	
	mainConfigPath := filepath.Join(tmpDir, "main.yaml")
	if err := os.WriteFile(mainConfigPath, []byte(mainConfig), 0644); err != nil {
		t.Fatalf("Failed to create main config file: %v", err)
	}
	
	// Create included config file
	includedConfig := `packages:
  apt:
    - "git"
    - "vim"
`
	
	includedConfigPath := filepath.Join(tmpDir, "packages.yaml")
	if err := os.WriteFile(includedConfigPath, []byte(includedConfig), 0644); err != nil {
		t.Fatalf("Failed to create included config file: %v", err)
	}
	
	// Load configuration
	config, paths, err := loader.LoadConfigurationAdvanced(mainConfigPath)
	if err != nil {
		t.Fatalf("LoadConfigurationAdvanced() failed: %v", err)
	}
	
	// Verify config was loaded correctly
	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}
	
	if config.Version != "1.0" {
		t.Errorf("Expected version to be '1.0', got '%s'", config.Version)
	}
	
	// Verify packages were merged
	expectedPackages := []string{"curl", "git", "vim"}
	if len(config.Packages.Apt) != len(expectedPackages) {
		t.Errorf("Expected %d APT packages, got %d", len(expectedPackages), len(config.Packages.Apt))
	}
	
	// Verify paths were tracked
	if len(paths) < 2 {
		t.Errorf("Expected at least 2 paths to be tracked, got %d", len(paths))
	}
}

// Benchmark tests
func BenchmarkEvaluateCondition(b *testing.B) {
	loader := NewAdvancedLoader()
	condition := IncludeCondition{
		Type:  "os",
		Value: runtime.GOOS,
	}
	
	for i := 0; i < b.N; i++ {
		loader.evaluateCondition(condition)
	}
}

func BenchmarkResolveGlobPattern(b *testing.B) {
	loader := NewAdvancedLoader()
	tmpDir := b.TempDir()
	
	// Create test files
	for i := 0; i < 10; i++ {
		path := filepath.Join(tmpDir, fmt.Sprintf("config%d.yaml", i))
		os.WriteFile(path, []byte("test"), 0644)
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		loader.resolveGlobPattern(tmpDir, "*.yaml", false)
	}
}

// Enhanced integration tests for advanced include system
func TestAdvancedIncludeSystem_ComplexScenarios(t *testing.T) {
	loader := NewAdvancedLoader()
	tmpDir := t.TempDir()
	
	// Create directory structure
	dirsToCreate := []string{"os-specific", "environments", "hosts", "packages"}
	for _, dir := range dirsToCreate {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}
	
	// Create main config with advanced includes
	mainConfig := `version: "1.0"
advanced_includes:
  # Glob pattern includes
  - glob: "packages/*.yaml"
    description: "All package configurations"
    optional: false
  
  # Conditional includes based on OS
  - path: "os-specific/linux.yaml"
    optional: true
    conditions:
      - type: "os"
        value: "linux"
        operator: "equals"
  
  # Environment-based includes
  - path: "environments/development.yaml"
    optional: true
    conditions:
      - type: "env"
        value: "NODE_ENV=development"
  
  # Hostname-based includes
  - glob: "hosts/*.yaml"
    optional: true
    conditions:
      - type: "hostname"
        value: "test"
        operator: "contains"

packages:
  apt:
    - base-package
`
	
	mainConfigPath := filepath.Join(tmpDir, "main.yaml")
	if err := os.WriteFile(mainConfigPath, []byte(mainConfig), 0644); err != nil {
		t.Fatalf("Failed to create main config: %v", err)
	}
	
	// Create package configs
	packageConfigs := []struct {
		path    string
		content string
	}{
		{
			"packages/development.yaml",
			`packages:
  apt:
    - dev-package-1
    - dev-package-2
files:
  dev-file:
    source: "/src/dev"
    destination: "/dst/dev"
`,
		},
		{
			"packages/production.yaml",
			`packages:
  apt:
    - prod-package-1
  flatpak:
    - org.example.ProdApp
`,
		},
		{
			"os-specific/linux.yaml",
			`packages:
  apt:
    - linux-specific-package
dconf:
  settings:
    "/org/gnome/setting": "'linux-value'"
`,
		},
		{
			"hosts/test-machine.yaml",
			`packages:
  snap:
    - test-snap
files:
  test-file:
    source: "/src/test"
    destination: "/dst/test"
`,
		},
	}
	
	for _, cfg := range packageConfigs {
		fullPath := filepath.Join(tmpDir, cfg.path)
		if err := os.WriteFile(fullPath, []byte(cfg.content), 0644); err != nil {
			t.Fatalf("Failed to create config file %s: %v", cfg.path, err)
		}
	}
	
	// Test loading configuration
	config, paths, err := loader.LoadConfigurationAdvanced(mainConfigPath)
	if err != nil {
		t.Fatalf("LoadConfigurationAdvanced() failed: %v", err)
	}
	
	// Verify main config was loaded
	if config == nil {
		t.Fatal("Expected config to be non-nil")
	}
	
	if config.Version != "1.0" {
		t.Errorf("Expected version to be '1.0', got '%s'", config.Version)
	}
	
	// Verify packages were merged from glob includes
	if len(config.Packages.Apt) < 3 { // base + at least packages from glob
		t.Errorf("Expected at least 3 APT packages, got %d", len(config.Packages.Apt))
	}
	
	// Verify paths tracking
	if len(paths) < 2 {
		t.Errorf("Expected at least 2 paths tracked, got %d", len(paths))
	}
	
	// Verify files were merged
	if len(config.Files) == 0 {
		t.Error("Expected files to be merged from includes")
	}
}

func TestAdvancedIncludeSystem_ConditionalIncludes(t *testing.T) {
	loader := NewAdvancedLoader()
	tmpDir := t.TempDir()
	
	tests := []struct {
		name      string
		condition IncludeCondition
		content   string
		shouldLoad bool
	}{
		{
			name: "os condition matches",
			condition: IncludeCondition{
				Type:     "os",
				Value:    runtime.GOOS,
				Operator: "equals",
			},
			content: `packages:
  apt:
    - os-specific-package
`,
			shouldLoad: true,
		},
		{
			name: "os condition doesn't match",
			condition: IncludeCondition{
				Type:     "os",
				Value:    "nonexistent-os",
				Operator: "equals",
			},
			content: `packages:
  apt:
    - should-not-load
`,
			shouldLoad: false,
		},
		{
			name: "hostname contains condition",
			condition: IncludeCondition{
				Type:     "hostname",
				Value:    loader.hostname[:3], // First 3 chars of hostname
				Operator: "contains",
			},
			content: `packages:
  flatpak:
    - org.example.HostnameApp
`,
			shouldLoad: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create main config
			mainConfig := fmt.Sprintf(`version: "1.0"
advanced_includes:
  - path: "conditional.yaml"
    optional: true
    conditions:
      - type: "%s"
        value: "%s"
        operator: "%s"
packages:
  apt:
    - base-package
`, tt.condition.Type, tt.condition.Value, tt.condition.Operator)
			
			mainConfigPath := filepath.Join(tmpDir, "main-"+tt.name+".yaml")
			if err := os.WriteFile(mainConfigPath, []byte(mainConfig), 0644); err != nil {
				t.Fatalf("Failed to create main config: %v", err)
			}
			
			// Create conditional config
			conditionalPath := filepath.Join(tmpDir, "conditional.yaml")
			if err := os.WriteFile(conditionalPath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create conditional config: %v", err)
			}
			
			// Load configuration
			config, _, err := loader.LoadConfigurationAdvanced(mainConfigPath)
			if err != nil {
				t.Fatalf("LoadConfigurationAdvanced() failed: %v", err)
			}
			
			// Check if conditional content was loaded
			basePackageFound := false
			conditionalPackageFound := false
			
			for _, pkg := range config.Packages.Apt {
				if pkg.Name == "base-package" {
					basePackageFound = true
				}
				if pkg.Name == "os-specific-package" || pkg.Name == "should-not-load" {
					conditionalPackageFound = true
				}
			}
			
			for _, pkg := range config.Packages.Flatpak {
				if pkg.Name == "org.example.HostnameApp" {
					conditionalPackageFound = true
				}
			}
			
			if !basePackageFound {
				t.Error("Base package should always be loaded")
			}
			
			if conditionalPackageFound != tt.shouldLoad {
				t.Errorf("Conditional package loading mismatch: expected %v, got %v", tt.shouldLoad, conditionalPackageFound)
			}
		})
	}
}

func TestAdvancedIncludeSystem_ErrorHandling(t *testing.T) {
	loader := NewAdvancedLoader()
	tmpDir := t.TempDir()
	
	tests := []struct {
		name        string
		config      string
		expectError bool
		errorSubstr string
	}{
		{
			name: "invalid glob pattern",
			config: `version: "1.0"
advanced_includes:
  - glob: "[invalid-glob"
    optional: false
`,
			expectError: true,
			errorSubstr: "invalid glob pattern",
		},
		{
			name: "missing required include",
			config: `version: "1.0"
advanced_includes:
  - path: "nonexistent.yaml"
    optional: false
`,
			expectError: true,
			errorSubstr: "required include not found",
		},
		{
			name: "invalid condition type",
			config: `version: "1.0"
advanced_includes:
  - path: "test.yaml"
    optional: true
    conditions:
      - type: "invalid_type"
        value: "test"
`,
			expectError: true,
			errorSubstr: "invalid condition type",
		},
		{
			name: "missing optional include",
			config: `version: "1.0"
advanced_includes:
  - path: "optional-missing.yaml"
    optional: true
packages:
  apt:
    - test-package
`,
			expectError: false, // Optional includes shouldn't cause errors
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, "test-"+tt.name+".yaml")
			if err := os.WriteFile(configPath, []byte(tt.config), 0644); err != nil {
				t.Fatalf("Failed to create test config: %v", err)
			}
			
			_, _, err := loader.LoadConfigurationAdvanced(configPath)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errorSubstr != "" && !strings.Contains(err.Error(), tt.errorSubstr) {
					t.Errorf("Expected error to contain '%s', got: %v", tt.errorSubstr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestAdvancedIncludeSystem_CircularDependencyDetection(t *testing.T) {
	loader := NewAdvancedLoader()
	tmpDir := t.TempDir()
	
	// Create circular dependency: main -> a -> b -> a
	mainConfig := `version: "1.0"
includes:
  - "a.yaml"
packages:
  apt:
    - main-package
`
	
	configA := `includes:
  - "b.yaml"
packages:
  apt:
    - package-a
`
	
	configB := `includes:
  - "a.yaml"
packages:
  apt:
    - package-b
`
	
	if err := os.WriteFile(filepath.Join(tmpDir, "main.yaml"), []byte(mainConfig), 0644); err != nil {
		t.Fatalf("Failed to create main config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "a.yaml"), []byte(configA), 0644); err != nil {
		t.Fatalf("Failed to create config a: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "b.yaml"), []byte(configB), 0644); err != nil {
		t.Fatalf("Failed to create config b: %v", err)
	}
	
	_, _, err := loader.LoadConfigurationAdvanced(filepath.Join(tmpDir, "main.yaml"))
	
	if err == nil {
		t.Error("Expected circular dependency error but got none")
	} else if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Expected circular dependency error, got: %v", err)
	}
}

