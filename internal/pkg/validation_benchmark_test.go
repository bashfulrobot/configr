package pkg

import (
	"fmt"
	"testing"

	"github.com/bashfulrobot/configr/internal/config"
)

// BenchmarkValidation benchmarks configuration validation performance
func BenchmarkValidation(b *testing.B) {
	// Create test configurations of varying sizes
	testCases := []struct {
		name        string
		packageCount int
		fileCount   int
	}{
		{"Small", 10, 5},
		{"Medium", 50, 25},
		{"Large", 200, 100},
		{"XLarge", 500, 250},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			cfg := createTestConfigForValidation(tc.packageCount, tc.fileCount)
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := config.Validate(cfg, "test.yaml")
				if result.HasErrors() {
					b.Fatalf("Validation failed: %v", result.Errors)
				}
			}
		})
	}
}

// BenchmarkPackageNameValidation benchmarks package name validation specifically
func BenchmarkPackageNameValidation(b *testing.B) {
	packageNames := generateTestPackageNames(1000)
	
	b.ResetTimer()
	
	b.Run("APT", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, name := range packageNames {
				_ = isValidPackageNameForManager(name, "apt")
			}
		}
	})
	
	b.Run("Flatpak", func(b *testing.B) {
		flatpakNames := generateTestFlatpakNames(1000)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, name := range flatpakNames {
				_ = isValidPackageNameForManager(name, "flatpak")
			}
		}
	})
	
	b.Run("Snap", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, name := range packageNames {
				_ = isValidPackageNameForManager(name, "snap")
			}
		}
	})
}

// BenchmarkFileValidation benchmarks file configuration validation
func BenchmarkFileValidation(b *testing.B) {
	testCases := []struct {
		name     string
		fileCount int
	}{
		{"Small", 10},
		{"Medium", 50},
		{"Large", 200},
		{"XLarge", 500},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			files := generateTestFileConfigs(tc.fileCount)
			cfg := &config.Config{
				Version: "1.0",
				Files:   files,
			}
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := config.Validate(cfg, "test.yaml")
				_ = result.Valid
			}
		})
	}
}

// BenchmarkComplexValidation benchmarks validation of complex configurations
func BenchmarkComplexValidation(b *testing.B) {
	cfg := createComplexTestConfig()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := config.Validate(cfg, "complex.yaml")
		if result.HasErrors() {
			b.Fatalf("Complex validation failed: %v", result.Errors)
		}
	}
}

// Helper functions

func createTestConfigForValidation(packageCount, fileCount int) *config.Config {
	cfg := &config.Config{
		Version: "1.0",
		Packages: config.PackageManagement{
			Apt:     make([]config.PackageEntry, packageCount/3),
			Flatpak: make([]config.PackageEntry, packageCount/3),
			Snap:    make([]config.PackageEntry, packageCount/3),
		},
		// Skip files for validation benchmarks to avoid file existence checks
		Files: make(map[string]config.File),
	}

	// Add APT packages
	for i := 0; i < packageCount/3; i++ {
		cfg.Packages.Apt[i] = config.PackageEntry{
			Name: fmt.Sprintf("apt-package-%d", i),
		}
	}

	// Add Flatpak packages
	for i := 0; i < packageCount/3; i++ {
		cfg.Packages.Flatpak[i] = config.PackageEntry{
			Name: fmt.Sprintf("com.example.App%d", i),
		}
	}

	// Add Snap packages
	for i := 0; i < packageCount/3; i++ {
		cfg.Packages.Snap[i] = config.PackageEntry{
			Name: fmt.Sprintf("snap-package-%d", i),
		}
	}

	return cfg
}

func generateTestPackageNames(count int) []string {
	names := make([]string, count)
	for i := 0; i < count; i++ {
		names[i] = fmt.Sprintf("package-%d", i)
	}
	return names
}

func generateTestFlatpakNames(count int) []string {
	names := make([]string, count)
	for i := 0; i < count; i++ {
		names[i] = fmt.Sprintf("com.example.App%d", i)
	}
	return names
}

func generateTestFileConfigs(count int) map[string]config.File {
	files := make(map[string]config.File)
	for i := 0; i < count; i++ {
		fileName := fmt.Sprintf("file-%d", i)
		files[fileName] = config.File{
			Source:      fmt.Sprintf("/src/%s", fileName),
			Destination: fmt.Sprintf("/dst/%s", fileName),
			Mode:        "644",
			Owner:       "user",
			Group:       "group",
		}
	}
	return files
}


// isValidPackageNameForManager is a mock function to test package validation
// In the real implementation, this would be in the validation package
func isValidPackageNameForManager(name, manager string) bool {
	if name == "" {
		return false
	}
	
	switch manager {
	case "apt":
		// Simple APT package name validation
		return len(name) > 0 && len(name) < 100
	case "flatpak":
		// Simple Flatpak reverse domain validation
		return len(name) > 0 && len(name) < 200
	case "snap":
		// Simple Snap package name validation
		return len(name) > 0 && len(name) < 50
	default:
		return false
	}
}