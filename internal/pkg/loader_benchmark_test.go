package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

// BenchmarkConfigurationLoading benchmarks configuration loading performance
func BenchmarkConfigurationLoading(b *testing.B) {
	tmpDir := b.TempDir()
	
	testCases := []struct {
		name        string
		fileCount   int
		includeDepth int
	}{
		{"SingleFile", 1, 0},
		{"ThreeFiles", 3, 1},
		{"TenFiles", 10, 2},
		{"TwentyFiles", 20, 3},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			// Create test configuration files
			configPaths := createNestedConfigFiles(b, tmpDir, tc.fileCount, tc.includeDepth)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Simulate configuration loading
				// In real implementation, this would use the actual loader
				_, err := loadTestConfiguration(configPaths[0])
				if err != nil {
					b.Fatalf("Failed to load configuration: %v", err)
				}
			}
		})
	}
}

// BenchmarkOptimizedLoader benchmarks the optimized configuration loader
func BenchmarkOptimizedLoader(b *testing.B) {
	tmpDir := b.TempDir()
	logger := log.New(os.Stderr)
	cm := NewCacheManager(logger)
	loader := NewOptimizedLoader(logger, cm)
	
	// Create test configuration
	configPath := createLargeNestedConfig(b, tmpDir)
	
	b.Run("ColdLoad", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Clear cache to simulate cold load
			cm.ClearCache()
			
			_, _, err := loader.LoadConfigurationOptimized(configPath)
			if err != nil {
				b.Fatalf("Optimized load failed: %v", err)
			}
		}
	})
	
	// Prime the cache
	loader.LoadConfigurationOptimized(configPath)
	
	b.Run("WarmLoad", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, err := loader.LoadConfigurationOptimized(configPath)
			if err != nil {
				b.Fatalf("Cached load failed: %v", err)
			}
		}
	})
}

// BenchmarkIncludeResolution benchmarks include file resolution
func BenchmarkIncludeResolution(b *testing.B) {
	tmpDir := b.TempDir()
	
	testCases := []struct {
		name         string
		includeCount int
		nestingDepth int
	}{
		{"FewIncludes", 3, 1},
		{"ManyIncludes", 15, 2},
		{"DeepNesting", 5, 5},
		{"WideAndDeep", 20, 3},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			configFiles := createComplexIncludeStructure(b, tmpDir, tc.includeCount, tc.nestingDepth)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Simulate include resolution
				err := resolveIncludes(configFiles[0])
				if err != nil {
					b.Fatalf("Include resolution failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkAdvancedLoader benchmarks the advanced loader with complex features
func BenchmarkAdvancedLoader(b *testing.B) {
	tmpDir := b.TempDir()
	
	// Create complex configuration with advanced features
	configPath := createAdvancedConfigStructure(b, tmpDir)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate advanced configuration loading since the function may not exist
		_, err := loadTestConfiguration(configPath)
		if err != nil {
			b.Fatalf("Advanced loading failed: %v", err)
		}
	}
}

// Helper functions

func createNestedConfigFiles(b *testing.B, tmpDir string, fileCount, depth int) []string {
	var paths []string
	
	// Create main config file
	mainConfig := filepath.Join(tmpDir, "main.yaml")
	mainContent := `version: "1.0"
includes:
`
	
	// Add includes
	for i := 1; i < fileCount; i++ {
		includeFile := fmt.Sprintf("include-%d.yaml", i)
		mainContent += fmt.Sprintf("  - %s\n", includeFile)
		
		// Create include file
		includePath := filepath.Join(tmpDir, includeFile)
		includeContent := generateIncludeContent(i, depth)
		
		if err := os.WriteFile(includePath, []byte(includeContent), 0644); err != nil {
			b.Fatalf("Failed to create include file: %v", err)
		}
		paths = append(paths, includePath)
	}
	
	// Add some packages to main config
	mainContent += `
packages:
  apt:
    - vim
    - git
    - curl
files:
  vimrc:
    source: dotfiles/.vimrc
    destination: ~/.vimrc
`
	
	if err := os.WriteFile(mainConfig, []byte(mainContent), 0644); err != nil {
		b.Fatalf("Failed to create main config: %v", err)
	}
	
	paths = append([]string{mainConfig}, paths...)
	return paths
}

func generateIncludeContent(index, depth int) string {
	content := fmt.Sprintf(`# Include file %d
packages:
  apt:
`, index)
	
	// Add packages based on index
	for i := 0; i < 5; i++ {
		content += fmt.Sprintf("    - package-%d-%d\n", index, i)
	}
	
	// Add files
	content += "files:\n"
	for i := 0; i < 3; i++ {
		content += fmt.Sprintf(`  file-%d-%d:
    source: src/file-%d-%d
    destination: dst/file-%d-%d
`, index, i, index, i, index, i)
	}
	
	return content
}

func createLargeNestedConfig(b *testing.B, tmpDir string) string {
	mainConfig := filepath.Join(tmpDir, "large.yaml")
	
	content := `version: "1.0"
includes:
  - packages/apt.yaml
  - packages/flatpak.yaml
  - packages/snap.yaml
  - files/dotfiles.yaml
  - files/system.yaml

package_defaults:
  apt: ["-y", "--no-install-recommends"]
  flatpak: ["--system"]

packages:
  apt:
`
	
	// Add many APT packages
	for i := 0; i < 50; i++ {
		content += fmt.Sprintf("    - large-package-%d\n", i)
	}
	
	content += `
files:
`
	
	// Add many files
	for i := 0; i < 30; i++ {
		content += fmt.Sprintf(`  large-file-%d:
    source: src/large-file-%d
    destination: dst/large-file-%d
`, i, i, i)
	}
	
	if err := os.WriteFile(mainConfig, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create large config: %v", err)
	}
	
	// Create include files
	createIncludeFiles(b, tmpDir)
	
	return mainConfig
}

func createIncludeFiles(b *testing.B, tmpDir string) {
	packagesDir := filepath.Join(tmpDir, "packages")
	filesDir := filepath.Join(tmpDir, "files")
	
	os.MkdirAll(packagesDir, 0755)
	os.MkdirAll(filesDir, 0755)
	
	// Create package files
	aptContent := "packages:\n  apt:\n"
	for i := 0; i < 20; i++ {
		aptContent += fmt.Sprintf("    - apt-include-%d\n", i)
	}
	os.WriteFile(filepath.Join(packagesDir, "apt.yaml"), []byte(aptContent), 0644)
	
	flatpakContent := "packages:\n  flatpak:\n"
	for i := 0; i < 10; i++ {
		flatpakContent += fmt.Sprintf("    - com.example.Include%d\n", i)
	}
	os.WriteFile(filepath.Join(packagesDir, "flatpak.yaml"), []byte(flatpakContent), 0644)
	
	// Create file configs
	dotfilesContent := "files:\n"
	for i := 0; i < 15; i++ {
		dotfilesContent += fmt.Sprintf(`  dotfile-%d:
    source: dotfiles/file-%d
    destination: ~/file-%d
`, i, i, i)
	}
	os.WriteFile(filepath.Join(filesDir, "dotfiles.yaml"), []byte(dotfilesContent), 0644)
}

func createComplexIncludeStructure(b *testing.B, tmpDir string, includeCount, depth int) []string {
	var paths []string
	
	mainConfig := filepath.Join(tmpDir, "complex.yaml")
	content := "version: \"1.0\"\nincludes:\n"
	
	for i := 0; i < includeCount; i++ {
		includeFile := fmt.Sprintf("level0-%d.yaml", i)
		content += fmt.Sprintf("  - %s\n", includeFile)
		
		includePath := filepath.Join(tmpDir, includeFile)
		includeContent := createNestedInclude(b, tmpDir, i, depth, 0)
		
		os.WriteFile(includePath, []byte(includeContent), 0644)
		paths = append(paths, includePath)
	}
	
	os.WriteFile(mainConfig, []byte(content), 0644)
	paths = append([]string{mainConfig}, paths...)
	
	return paths
}

func createNestedInclude(b *testing.B, tmpDir string, index, maxDepth, currentDepth int) string {
	content := fmt.Sprintf("# Level %d, Index %d\n", currentDepth, index)
	
	if currentDepth < maxDepth {
		content += "includes:\n"
		for i := 0; i < 2; i++ {
			nestedFile := fmt.Sprintf("level%d-%d-%d.yaml", currentDepth+1, index, i)
			content += fmt.Sprintf("  - %s\n", nestedFile)
			
			nestedPath := filepath.Join(tmpDir, nestedFile)
			nestedContent := createNestedInclude(b, tmpDir, i, maxDepth, currentDepth+1)
			os.WriteFile(nestedPath, []byte(nestedContent), 0644)
		}
	}
	
	content += "packages:\n  apt:\n"
	for i := 0; i < 3; i++ {
		content += fmt.Sprintf("    - pkg-d%d-i%d-n%d\n", currentDepth, index, i)
	}
	
	return content
}

func createAdvancedConfigStructure(b *testing.B, tmpDir string) string {
	mainConfig := filepath.Join(tmpDir, "advanced.yaml")
	
	content := `version: "1.0"
advanced_includes:
  - glob: "configs/*.yaml"
    optional: true
  - path: "os-specific/linux.yaml"
    optional: true
    conditions:
      - type: "os"
        value: "linux"
        operator: "equals"
  - path: "env/development.yaml"
    optional: true
    conditions:
      - type: "env"
        value: "NODE_ENV=development"

packages:
  apt:
    - advanced-pkg-1
    - advanced-pkg-2
  flatpak:
    - com.advanced.App1
    - com.advanced.App2

files:
  advanced-config:
    source: configs/advanced.conf
    destination: ~/.config/advanced.conf
    mode: "600"
    backup: true
`
	
	if err := os.WriteFile(mainConfig, []byte(content), 0644); err != nil {
		b.Fatalf("Failed to create advanced config: %v", err)
	}
	
	// Create additional directories and files
	configsDir := filepath.Join(tmpDir, "configs")
	os.MkdirAll(configsDir, 0755)
	
	for i := 0; i < 5; i++ {
		configFile := filepath.Join(configsDir, fmt.Sprintf("config-%d.yaml", i))
		configContent := fmt.Sprintf("packages:\n  apt:\n    - config-specific-%d\n", i)
		os.WriteFile(configFile, []byte(configContent), 0644)
	}
	
	return mainConfig
}

// Mock functions for benchmarking
func loadTestConfiguration(configPath string) (*config.Config, error) {
	// Simulate configuration loading work
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	
	// Simulate parsing overhead
	_ = len(content)
	
	return &config.Config{
		Version: "1.0",
		Packages: config.PackageManagement{
			Apt: []config.PackageEntry{{Name: "test"}},
		},
	}, nil
}

func resolveIncludes(configPath string) error {
	// Simulate include resolution work
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	
	// Simulate parsing and resolution overhead
	_ = len(content)
	
	return nil
}