package pkg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

// BenchmarkConfigCaching benchmarks configuration caching performance
func BenchmarkConfigCaching(b *testing.B) {
	// Create temporary directory
	tmpDir := b.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	logger := log.New(os.Stderr)
	cm := NewCacheManagerWithPath(logger, cacheDir)

	// Create a realistic configuration
	cfg := createLargeTestConfig()

	// Create config files
	configPaths := createTestConfigFiles(b, tmpDir, 5) // 5 config files with includes

	b.ResetTimer()

	// Benchmark cache save performance
	b.Run("SaveCache", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := cm.SaveCachedConfig(cfg, configPaths); err != nil {
				b.Fatalf("SaveCachedConfig failed: %v", err)
			}
		}
	})

	// Ensure cache exists for load benchmark
	if err := cm.SaveCachedConfig(cfg, configPaths); err != nil {
		b.Fatalf("Failed to setup cache for load benchmark: %v", err)
	}

	// Benchmark cache load performance
	b.Run("LoadCache", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cached, err := cm.LoadCachedConfig(configPaths)
			if err != nil {
				b.Fatalf("LoadCachedConfig failed: %v", err)
			}
			if cached == nil {
				b.Fatal("Expected cached config, got nil")
			}
		}
	})
}

// BenchmarkSystemStateCaching benchmarks system state caching performance
func BenchmarkSystemStateCaching(b *testing.B) {
	// Create temporary directory
	tmpDir := b.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	logger := log.New(os.Stderr)
	cm := NewCacheManagerWithPath(logger, cacheDir)

	// Create a large system state cache
	cache := createLargeSystemStateCache()

	b.ResetTimer()

	// Benchmark system state save performance
	b.Run("SaveSystemState", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if err := cm.SaveSystemStateCache(cache); err != nil {
				b.Fatalf("SaveSystemStateCache failed: %v", err)
			}
		}
	})

	// Ensure cache exists for load benchmark
	if err := cm.SaveSystemStateCache(cache); err != nil {
		b.Fatalf("Failed to setup system cache: %v", err)
	}

	// Benchmark system state load performance
	b.Run("LoadSystemState", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			loaded, err := cm.LoadSystemStateCache()
			if err != nil {
				b.Fatalf("LoadSystemStateCache failed: %v", err)
			}
			if loaded == nil {
				b.Fatal("Expected cached system state, got nil")
			}
		}
	})
}

// BenchmarkCacheOverhead compares cached vs non-cached performance
func BenchmarkCacheOverhead(b *testing.B) {
	// Create temporary directory
	tmpDir := b.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	logger := log.New(os.Stderr)
	cm := NewCacheManagerWithPath(logger, cacheDir)

	cfg := createLargeTestConfig()
	configPaths := createTestConfigFiles(b, tmpDir, 3)

	// Prime the cache
	if err := cm.SaveCachedConfig(cfg, configPaths); err != nil {
		b.Fatalf("Failed to prime cache: %v", err)
	}

	b.ResetTimer()

	// Benchmark cache hit scenario
	b.Run("CacheHit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cached, err := cm.LoadCachedConfig(configPaths)
			if err != nil {
				b.Fatalf("LoadCachedConfig failed: %v", err)
			}
			if cached == nil {
				b.Fatal("Expected cache hit, got miss")
			}
			
			// Simulate using the configuration
			_ = len(cached.Config.Packages.Apt)
		}
	})

	// Benchmark cache miss scenario (simulate standard loading)
	b.Run("CacheMiss", func(b *testing.B) {
		// Clear cache before each run to simulate miss
		for i := 0; i < b.N; i++ {
			// Clear cache to force miss
			if err := cm.ClearCache(); err != nil {
				b.Fatalf("Failed to clear cache: %v", err)
			}
			
			// This simulates the work that would be done on cache miss
			// In reality, this would involve YAML parsing, includes resolution, etc.
			_, err := cm.LoadCachedConfig(configPaths)
			if err != nil {
				b.Fatalf("LoadCachedConfig failed: %v", err)
			}
			// Would be nil due to cache miss, so we'd need to do full parsing
		}
	})
}

// createLargeTestConfig creates a configuration with many packages for benchmarking
func createLargeTestConfig() *config.Config {
	cfg := &config.Config{
		Version: "1.0",
		Packages: config.PackageManagement{
			Apt: make([]config.PackageEntry, 100),
			Flatpak: make([]config.PackageEntry, 50),
			Snap: make([]config.PackageEntry, 25),
		},
		Files: make(map[string]config.File),
	}

	// Add APT packages
	for i := 0; i < 100; i++ {
		cfg.Packages.Apt[i] = config.PackageEntry{
			Name: generatePackageName("apt-pkg", i),
		}
	}

	// Add Flatpak packages
	for i := 0; i < 50; i++ {
		cfg.Packages.Flatpak[i] = config.PackageEntry{
			Name: generateFlatpakName("com.example.App", i),
		}
	}

	// Add Snap packages
	for i := 0; i < 25; i++ {
		cfg.Packages.Snap[i] = config.PackageEntry{
			Name: generatePackageName("snap-pkg", i),
		}
	}

	// Add files
	for i := 0; i < 50; i++ {
		fileName := generatePackageName("file", i)
		cfg.Files[fileName] = config.File{
			Source:      "/source/" + fileName,
			Destination: "/dest/" + fileName,
		}
	}

	return cfg
}

// createLargeSystemStateCache creates a large system state cache for benchmarking
func createLargeSystemStateCache() *SystemStateCache {
	cache := &SystemStateCache{
		PackageState: PackageInstallationState{
			AptPackages:     make(map[string]PackageCacheEntry),
			FlatpakPackages: make(map[string]PackageCacheEntry),
			SnapPackages:    make(map[string]PackageCacheEntry),
		},
		FileState: FileDeploymentState{
			Files: make(map[string]FileCacheEntry),
		},
	}

	// Add many package entries
	for i := 0; i < 200; i++ {
		pkgName := generatePackageName("pkg", i)
		cache.PackageState.AptPackages[pkgName] = PackageCacheEntry{
			Name:      pkgName,
			Installed: i%2 == 0, // Alternate installed/not installed
		}
	}

	// Add Flatpak entries
	for i := 0; i < 100; i++ {
		pkgName := generateFlatpakName("com.example.App", i)
		cache.PackageState.FlatpakPackages[pkgName] = PackageCacheEntry{
			Name:      pkgName,
			Installed: i%3 == 0, // Every third installed
		}
	}

	// Add file entries
	for i := 0; i < 150; i++ {
		fileName := generatePackageName("file", i)
		cache.FileState.Files[fileName] = FileCacheEntry{
			Path:      "/path/to/" + fileName,
			Size:      int64(i * 1024),
			IsSymlink: i%2 == 0,
		}
	}

	return cache
}

// createTestConfigFiles creates multiple config files for testing includes
func createTestConfigFiles(b *testing.B, tmpDir string, count int) []string {
	var paths []string
	
	for i := 0; i < count; i++ {
		filename := filepath.Join(tmpDir, generatePackageName("config", i)+".yaml")
		content := generateConfigContent(i)
		
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to create test config file %s: %v", filename, err)
		}
		
		paths = append(paths, filename)
	}
	
	return paths
}

// generatePackageName generates a package name for testing
func generatePackageName(prefix string, i int) string {
	return fmt.Sprintf("%s-%03d", prefix, i)
}

// generateFlatpakName generates a Flatpak application ID for testing
func generateFlatpakName(base string, i int) string {
	return fmt.Sprintf("%s%03d", base, i)
}

// generateConfigContent generates YAML content for test config files
func generateConfigContent(i int) string {
	return fmt.Sprintf(`version: "1.0"
packages:
  apt:
    - package-%03d
    - tool-%03d
files:
  file-%03d:
    source: "/src/file-%03d"
    destination: "/dst/file-%03d"
`, i, i, i, i, i)
}

// We need to import fmt for the benchmark functions
import "fmt"