package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestLoadWithIncludes_SimpleConfig(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create a simple config file
	configContent := `version: "1.0"
packages:
  apt:
    - git
    - curl
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Set up viper
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	
	// Load config
	config, err := LoadWithIncludes()
	if err != nil {
		t.Fatalf("LoadWithIncludes failed: %v", err)
	}
	
	// Verify loaded config
	if config.Version != "1.0" {
		t.Errorf("expected version '1.0', got '%s'", config.Version)
	}
	
	if len(config.Packages.Apt) != 2 {
		t.Errorf("expected 2 apt packages, got %d", len(config.Packages.Apt))
	}
	
	expectedPackages := []string{"git", "curl"}
	for i, pkg := range config.Packages.Apt {
		if pkg.Name != expectedPackages[i] {
			t.Errorf("expected package '%s', got '%s'", expectedPackages[i], pkg.Name)
		}
	}
}

func TestLoadWithIncludes_WithIncludes(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create main config
	mainConfig := `version: "1.0"
includes:
  - path: packages.yaml
packages:
  apt:
    - git
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(configPath, []byte(mainConfig), 0644)
	if err != nil {
		t.Fatalf("failed to create main config: %v", err)
	}
	
	// Create included config
	includedConfig := `packages:
  apt:
    - curl
    - vim
  snap:
    - discord
`
	includedPath := filepath.Join(tempDir, "packages.yaml")
	err = os.WriteFile(includedPath, []byte(includedConfig), 0644)
	if err != nil {
		t.Fatalf("failed to create included config: %v", err)
	}
	
	// Set up viper
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	
	// Load config
	config, err := LoadWithIncludes()
	if err != nil {
		t.Fatalf("LoadWithIncludes failed: %v", err)
	}
	
	// Verify merged config
	if len(config.Packages.Apt) != 3 {
		t.Errorf("expected 3 apt packages after merge, got %d", len(config.Packages.Apt))
	}
	
	if len(config.Packages.Snap) != 1 {
		t.Errorf("expected 1 snap package, got %d", len(config.Packages.Snap))
	}
	
	// Check that packages were merged correctly
	aptNames := make([]string, len(config.Packages.Apt))
	for i, pkg := range config.Packages.Apt {
		aptNames[i] = pkg.Name
	}
	
	expectedApt := []string{"git", "curl", "vim"}
	for _, expected := range expectedApt {
		found := false
		for _, actual := range aptNames {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected apt package '%s' not found in merged config", expected)
		}
	}
}

func TestLoadWithIncludes_DirectoryInclude(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create directory structure
	packagesDir := filepath.Join(tempDir, "packages")
	err := os.Mkdir(packagesDir, 0755)
	if err != nil {
		t.Fatalf("failed to create packages directory: %v", err)
	}
	
	// Create main config
	mainConfig := `version: "1.0"
includes:
  - path: packages/
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err = os.WriteFile(configPath, []byte(mainConfig), 0644)
	if err != nil {
		t.Fatalf("failed to create main config: %v", err)
	}
	
	// Create default.yaml in packages directory
	defaultConfig := `packages:
  apt:
    - htop
    - tree
`
	defaultPath := filepath.Join(packagesDir, "default.yaml")
	err = os.WriteFile(defaultPath, []byte(defaultConfig), 0644)
	if err != nil {
		t.Fatalf("failed to create default.yaml: %v", err)
	}
	
	// Set up viper
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	
	// Load config
	config, err := LoadWithIncludes()
	if err != nil {
		t.Fatalf("LoadWithIncludes failed: %v", err)
	}
	
	// Verify loaded config
	if len(config.Packages.Apt) != 2 {
		t.Errorf("expected 2 apt packages, got %d", len(config.Packages.Apt))
	}
}

func TestLoadWithIncludes_CircularInclude(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create main config that includes second.yaml
	mainConfig := `version: "1.0"
includes:
  - path: second.yaml
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(configPath, []byte(mainConfig), 0644)
	if err != nil {
		t.Fatalf("failed to create main config: %v", err)
	}
	
	// Create second config that includes the main config (circular)
	secondConfig := `includes:
  - path: configr.yaml
packages:
  apt:
    - git
`
	secondPath := filepath.Join(tempDir, "second.yaml")
	err = os.WriteFile(secondPath, []byte(secondConfig), 0644)
	if err != nil {
		t.Fatalf("failed to create second config: %v", err)
	}
	
	// Set up viper
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	
	// Load config - should fail with circular include error
	_, err = LoadWithIncludes()
	if err == nil {
		t.Fatal("LoadWithIncludes should fail with circular include")
	}
	
	if !strings.Contains(err.Error(), "circular include") {
		t.Errorf("expected 'circular include' error, got: %v", err)
	}
}

func TestLoadWithIncludes_MissingInclude(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create main config with non-existent include
	mainConfig := `version: "1.0"
includes:
  - path: nonexistent.yaml
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(configPath, []byte(mainConfig), 0644)
	if err != nil {
		t.Fatalf("failed to create main config: %v", err)
	}
	
	// Set up viper
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	
	// Load config - should fail
	_, err = LoadWithIncludes()
	if err == nil {
		t.Fatal("LoadWithIncludes should fail with missing include")
	}
}

func TestResolveIncludePath(t *testing.T) {
	tempDir := t.TempDir()
	
	// Create test files and directories
	err := os.WriteFile(filepath.Join(tempDir, "explicit.yaml"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("failed to create explicit.yaml: %v", err)
	}
	
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}
	
	err = os.WriteFile(filepath.Join(subDir, "default.yaml"), []byte("test"), 0644)
	if err != nil {
		t.Fatalf("failed to create default.yaml: %v", err)
	}
	
	tests := []struct {
		name         string
		includePath  string
		expectedFile string
		shouldError  bool
	}{
		{
			name:         "explicit file",
			includePath:  "explicit.yaml",
			expectedFile: "explicit.yaml",
			shouldError:  false,
		},
		{
			name:         "directory with trailing slash",
			includePath:  "subdir/",
			expectedFile: "subdir/default.yaml",
			shouldError:  false,
		},
		{
			name:         "directory without trailing slash",
			includePath:  "subdir",
			expectedFile: "subdir/default.yaml",
			shouldError:  false,
		},
		{
			name:        "nonexistent file",
			includePath: "nonexistent.yaml",
			shouldError: true,
		},
		{
			name:        "directory without default.yaml",
			includePath: "empty/",
			shouldError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := resolveIncludePath(tempDir, tt.includePath)
			
			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error for %s", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %s: %v", tt.name, err)
				}
				
				expectedPath := filepath.Join(tempDir, tt.expectedFile)
				if resolved != expectedPath {
					t.Errorf("expected %s, got %s", expectedPath, resolved)
				}
			}
		})
	}
}

func TestRemoveDuplicatePackages(t *testing.T) {
	packages := []PackageEntry{
		{Name: "git"},
		{Name: "curl"},
		{Name: "git"}, // Duplicate
		{Name: "vim"},
		{Name: "curl"}, // Duplicate
	}
	
	result := removeDuplicatePackages(packages)
	
	if len(result) != 3 {
		t.Errorf("expected 3 unique packages, got %d", len(result))
	}
	
	expectedNames := []string{"git", "curl", "vim"}
	for i, pkg := range result {
		if pkg.Name != expectedNames[i] {
			t.Errorf("expected package %s at index %d, got %s", expectedNames[i], i, pkg.Name)
		}
	}
}

func TestMergeConfigs(t *testing.T) {
	dst := &Config{
		Packages: PackageManagement{
			Apt: []PackageEntry{{Name: "git"}},
		},
		Files: map[string]File{
			"file1": {Source: "src1", Destination: "dest1"},
		},
		DConf: DConfConfig{
			Settings: map[string]string{
				"/setting1": "'value1'",
			},
		},
	}
	
	src := &Config{
		Packages: PackageManagement{
			Apt:  []PackageEntry{{Name: "curl"}},
			Snap: []PackageEntry{{Name: "discord"}},
		},
		Files: map[string]File{
			"file1": {Source: "new_src1", Destination: "new_dest1"}, // Override
			"file2": {Source: "src2", Destination: "dest2"},         // New
		},
		DConf: DConfConfig{
			Settings: map[string]string{
				"/setting1": "'new_value1'", // Override
				"/setting2": "'value2'",     // New
			},
		},
	}
	
	err := mergeConfigs(dst, src)
	if err != nil {
		t.Fatalf("mergeConfigs failed: %v", err)
	}
	
	// Check packages were merged
	if len(dst.Packages.Apt) != 2 {
		t.Errorf("expected 2 apt packages after merge, got %d", len(dst.Packages.Apt))
	}
	
	if len(dst.Packages.Snap) != 1 {
		t.Errorf("expected 1 snap package after merge, got %d", len(dst.Packages.Snap))
	}
	
	// Check files were merged (src should override dst)
	if len(dst.Files) != 2 {
		t.Errorf("expected 2 files after merge, got %d", len(dst.Files))
	}
	
	if dst.Files["file1"].Source != "new_src1" {
		t.Errorf("file1 should be overridden by src config")
	}
	
	// Check dconf was merged (src should override dst)
	if len(dst.DConf.Settings) != 2 {
		t.Errorf("expected 2 dconf settings after merge, got %d", len(dst.DConf.Settings))
	}
	
	if dst.DConf.Settings["/setting1"] != "'new_value1'" {
		t.Errorf("setting1 should be overridden by src config")
	}
}

func TestConfigFileDiscoveryOrder(t *testing.T) {
	// Test the configuration file discovery order as implemented in root.go
	tempDir := t.TempDir()
	
	// Create test config files
	configContent := `version: "1.0"
packages:
  apt:
    - git
`
	
	// Create config in different locations
	currentDirConfig := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(currentDirConfig, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create current dir config: %v", err)
	}
	
	// Test explicit config file (highest priority)
	explicitConfig := filepath.Join(tempDir, "explicit.yaml")
	err = os.WriteFile(explicitConfig, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create explicit config: %v", err)
	}
	
	// Test CONFIGR_CONFIG environment variable (second priority)
	envConfig := filepath.Join(tempDir, "env.yaml")
	err = os.WriteFile(envConfig, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create env config: %v", err)
	}
	
	tests := []struct {
		name           string
		explicitConfig string
		envConfig      string
		expectedFile   string
	}{
		{
			name:           "explicit config file takes precedence",
			explicitConfig: explicitConfig,
			envConfig:      envConfig,
			expectedFile:   explicitConfig,
		},
		{
			name:           "env config used when no explicit file",
			explicitConfig: "",
			envConfig:      envConfig,
			expectedFile:   envConfig,
		},
		{
			name:           "fallback to search when neither explicit nor env",
			explicitConfig: "",
			envConfig:      "",
			expectedFile:   "", // Will use search path
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear viper state
			viper.Reset()
			
			// Set up environment
			if tt.envConfig != "" {
				os.Setenv("CONFIGR_CONFIG", tt.envConfig)
				defer os.Unsetenv("CONFIGR_CONFIG")
			}
			
			// Simulate the initConfig logic from root.go
			if tt.explicitConfig != "" {
				viper.SetConfigFile(tt.explicitConfig)
			} else if configEnv := os.Getenv("CONFIGR_CONFIG"); configEnv != "" {
				viper.SetConfigFile(configEnv)
			} else {
				// For testing, we'll just use the current directory
				viper.SetConfigName("configr")
				viper.SetConfigType("yaml")
				viper.AddConfigPath(tempDir)
			}
			
			err := viper.ReadInConfig()
			if err != nil && tt.expectedFile != "" {
				t.Fatalf("failed to read config: %v", err)
			}
			
			if tt.expectedFile != "" {
				used := viper.ConfigFileUsed()
				if used != tt.expectedFile {
					t.Errorf("expected config file %s, got %s", tt.expectedFile, used)
				}
			}
		})
	}
}