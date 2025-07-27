package pkg

import (
	"fmt"
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

// BenchmarkAdvancedIncludeSystem benchmarks the advanced include system performance
func BenchmarkAdvancedIncludeSystem(b *testing.B) {
	// Create temporary directory
	tmpDir := b.TempDir()
	
	// Create many include files
	_ = createManyIncludeFiles(b, tmpDir, 50)
	
	// Create main config with advanced includes
	mainConfig := filepath.Join(tmpDir, "main.yaml")
	mainContent := `version: "1.0"
advanced_includes:
  - glob: "includes/*.yaml"
    optional: false
  - path: "specific/config.yaml"
    optional: true
    conditions:
      - type: "os"
        value: "linux"
        operator: "equals"
packages:
  apt:
    - git
    - curl
`
	if err := os.WriteFile(mainConfig, []byte(mainContent), 0644); err != nil {
		b.Fatalf("Failed to create main config: %v", err)
	}
	
	b.ResetTimer()
	
	// Benchmark loading with advanced includes
	for i := 0; i < b.N; i++ {
		_, err := config.LoadWithIncludes()
		if err != nil {
			b.Fatalf("Failed to load configuration: %v", err)
		}
	}
}

// BenchmarkInteractiveFeatures benchmarks interactive feature performance
func BenchmarkInteractiveFeatures(b *testing.B) {
	logger := log.New(os.Stderr)
	_ = NewInteractiveManager(logger) // Create but don't use in benchmark
	
	// Create test scenarios
	testPrompts := []string{
		"Do you want to overwrite the existing file?",
		"Apply new permissions (644) to the file?",
		"Change ownership to user:group?",
		"Continue with deployment?",
		"Skip this file and continue?",
	}
	
	b.ResetTimer()
	
	// Benchmark prompt preparation (no actual TTY interaction)
	for i := 0; i < b.N; i++ {
		prompt := testPrompts[i%len(testPrompts)]
		// Simulate prompt preparation overhead
		_ = prompt + " (default)"
	}
}

// BenchmarkPackageStateChecking benchmarks package state checking performance
func BenchmarkPackageStateChecking(b *testing.B) {
	// Create temporary directory
	tmpDir := b.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")
	
	logger := log.New(os.Stderr)
	cm := NewCacheManagerWithPath(logger, cacheDir)
	
	// Create large system state cache
	stateCache := createLargeSystemStateCache()
	
	// Save the cache first
	if err := cm.SaveSystemStateCache(stateCache); err != nil {
		b.Fatalf("Failed to save system state cache: %v", err)
	}
	
	b.ResetTimer()
	
	// Benchmark loading system state cache
	for i := 0; i < b.N; i++ {
		_, err := cm.LoadSystemStateCache()
		if err != nil {
			b.Fatalf("Failed to load system state cache: %v", err)
		}
	}
}

// BenchmarkComplexConfigValidation benchmarks validation of complex configurations
func BenchmarkComplexConfigValidation(b *testing.B) {
	// Create a complex configuration for validation benchmarking
	cfg := createComplexTestConfig()
	
	b.ResetTimer()
	
	// Benchmark validation performance
	for i := 0; i < b.N; i++ {
		result := config.Validate(cfg, "benchmark.yaml")
		if result.HasErrors() {
			b.Fatalf("Validation failed: %v", result.Errors)
		}
	}
}

// createManyIncludeFiles creates many include files for benchmarking
func createManyIncludeFiles(b *testing.B, tmpDir string, count int) []string {
	includesDir := filepath.Join(tmpDir, "includes")
	if err := os.MkdirAll(includesDir, 0755); err != nil {
		b.Fatalf("Failed to create includes directory: %v", err)
	}
	
	var paths []string
	
	for i := 0; i < count; i++ {
		filename := filepath.Join(includesDir, fmt.Sprintf("include-%03d.yaml", i))
		content := fmt.Sprintf(`# Include file %d
packages:
  apt:
    - pkg-%03d-a
    - pkg-%03d-b
  flatpak:
    - org.example.App%03d
files:
  file-%03d:
    source: "/source/file-%03d"
    destination: "/dest/file-%03d"
dconf:
  settings:
    "/org/example/setting-%03d": "'value-%03d'"
`, i, i, i, i, i, i, i, i, i)
		
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			b.Fatalf("Failed to create include file %s: %v", filename, err)
		}
		
		paths = append(paths, filename)
	}
	
	return paths
}

// createComplexTestConfig creates a complex configuration for validation benchmarking
func createComplexTestConfig() *config.Config {
	cfg := &config.Config{
		Version: "1.0",
		PackageDefaults: map[string][]string{
			"apt":     {"-y", "--no-install-recommends"},
			"flatpak": {"--user", "--assumeyes"},
			"snap":    {"--classic"},
		},
		Repositories: config.RepositoryManagement{
			Apt: []config.AptRepository{
				{Name: "python39", PPA: "deadsnakes/ppa"},
				{Name: "docker", URI: "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable", Key: "https://download.docker.com/linux/ubuntu/gpg"},
			},
			Flatpak: []config.FlatpakRepository{
				{Name: "flathub", URL: "https://flathub.org/repo/flathub.flatpakrepo"},
				{Name: "kde", URL: "https://distribute.kde.org/kdeapps.flatpakrepo", User: true},
			},
		},
		Packages: config.PackageManagement{
			Apt: make([]config.PackageEntry, 200),
			Flatpak: make([]config.PackageEntry, 100),
			Snap: make([]config.PackageEntry, 50),
		},
		Files: make(map[string]config.File),
		DConf: config.DConfConfig{
			Settings: make(map[string]string),
		},
	}
	
	// Add many packages with various flag configurations
	for i := 0; i < 200; i++ {
		cfg.Packages.Apt[i] = config.PackageEntry{
			Name: generatePackageName("apt-pkg", i),
			Flags: []string{"-y"},
		}
	}
	
	for i := 0; i < 100; i++ {
		cfg.Packages.Flatpak[i] = config.PackageEntry{
			Name: generateFlatpakName("com.example.App", i),
			Flags: []string{"--user"},
		}
	}
	
	for i := 0; i < 50; i++ {
		cfg.Packages.Snap[i] = config.PackageEntry{
			Name: generatePackageName("snap-pkg", i),
			Flags: []string{"--classic"},
		}
	}
	
	// Add many files with interactive features
	for i := 0; i < 100; i++ {
		fileName := generatePackageName("file", i)
		cfg.Files[fileName] = config.File{
			Source:           "/source/" + fileName,
			Destination:      "/dest/" + fileName,
			Mode:             "644",
			Owner:            "user",
			Group:            "group",
			Backup:           true,
			Interactive:      i%10 == 0, // Every 10th file is interactive
			PromptPermissions: i%20 == 0, // Every 20th file prompts for permissions
			PromptOwnership:  i%30 == 0, // Every 30th file prompts for ownership
		}
	}
	
	// Add many DConf settings
	for i := 0; i < 150; i++ {
		settingPath := fmt.Sprintf("/org/example/setting-%03d", i)
		settingValue := fmt.Sprintf("'value-%03d'", i)
		cfg.DConf.Settings[settingPath] = settingValue
	}
	
	return cfg
}

