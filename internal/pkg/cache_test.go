package pkg

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

func TestCacheManager_ConfigCache(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	logger := log.New(os.Stderr)
	cm := NewCacheManagerWithPath(logger, cacheDir)

	// Create test configuration
	cfg := &config.Config{
		Version: "1.0",
		Packages: config.PackageManagement{
			Apt: []config.PackageEntry{
				{Name: "vim"},
				{Name: "git"},
			},
		},
	}

	// Create dummy config file
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte("version: '1.0'\npackages:\n  apt:\n    - vim\n    - git"), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	configPaths := []string{configPath}

	// Test saving configuration to cache
	if err := cm.SaveCachedConfig(cfg, configPaths); err != nil {
		t.Fatalf("SaveCachedConfig() failed: %v", err)
	}

	// Test loading configuration from cache
	cached, err := cm.LoadCachedConfig(configPaths)
	if err != nil {
		t.Fatalf("LoadCachedConfig() failed: %v", err)
	}

	if cached == nil {
		t.Fatal("Expected cached config, got nil")
	}

	// Verify cached configuration
	if cached.Config.Version != cfg.Version {
		t.Errorf("Version mismatch: expected %s, got %s", cfg.Version, cached.Config.Version)
	}

	if len(cached.Config.Packages.Apt) != len(cfg.Packages.Apt) {
		t.Errorf("APT packages mismatch: expected %d, got %d", len(cfg.Packages.Apt), len(cached.Config.Packages.Apt))
	}

	// Verify cache metadata
	if len(cached.ConfigPaths) != 1 || cached.ConfigPaths[0] != configPath {
		t.Errorf("Config paths mismatch: expected [%s], got %v", configPath, cached.ConfigPaths)
	}

	if cached.ConfigHash == "" {
		t.Error("Expected config hash, got empty string")
	}
}

func TestCacheManager_InvalidateOnFileChange(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	logger := log.New(os.Stderr)
	cm := NewCacheManagerWithPath(logger, cacheDir)

	// Create test configuration
	cfg := &config.Config{Version: "1.0"}

	// Create dummy config file
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte("version: '1.0'"), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	configPaths := []string{configPath}

	// Save to cache
	if err := cm.SaveCachedConfig(cfg, configPaths); err != nil {
		t.Fatalf("SaveCachedConfig() failed: %v", err)
	}

	// Verify cache hit
	cached, err := cm.LoadCachedConfig(configPaths)
	if err != nil {
		t.Fatalf("LoadCachedConfig() failed: %v", err)
	}
	if cached == nil {
		t.Fatal("Expected cache hit, got nil")
	}

	// Wait a bit to ensure different modification time
	time.Sleep(10 * time.Millisecond)

	// Modify config file
	if err := os.WriteFile(configPath, []byte("version: '2.0'"), 0644); err != nil {
		t.Fatalf("Failed to modify config file: %v", err)
	}

	// Verify cache miss after file modification
	cached, err = cm.LoadCachedConfig(configPaths)
	if err != nil {
		t.Fatalf("LoadCachedConfig() failed: %v", err)
	}
	if cached != nil {
		t.Error("Expected cache miss after file modification, got cache hit")
	}
}

func TestCacheManager_SystemStateCache(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	logger := log.New(os.Stderr)
	cm := NewCacheManagerWithPath(logger, cacheDir)

	// Create test system state cache
	cache := &SystemStateCache{
		PackageState: PackageInstallationState{
			AptPackages: map[string]PackageCacheEntry{
				"vim": {
					Name:        "vim",
					Installed:   true,
					LastChecked: time.Now(),
				},
				"git": {
					Name:        "git",
					Installed:   false,
					LastChecked: time.Now(),
				},
			},
			LastUpdated: time.Now(),
		},
		LastChecked: time.Now(),
		SystemHash:  "test-hash",
		Version:     "1.0",
	}

	// Test saving system state cache
	if err := cm.SaveSystemStateCache(cache); err != nil {
		t.Fatalf("SaveSystemStateCache() failed: %v", err)
	}

	// Test loading system state cache
	loaded, err := cm.LoadSystemStateCache()
	if err != nil {
		t.Fatalf("LoadSystemStateCache() failed: %v", err)
	}

	if loaded == nil {
		t.Fatal("Expected loaded cache, got nil")
	}

	// Verify cache contents
	if loaded.SystemHash != cache.SystemHash {
		t.Errorf("System hash mismatch: expected %s, got %s", cache.SystemHash, loaded.SystemHash)
	}

	if len(loaded.PackageState.AptPackages) != len(cache.PackageState.AptPackages) {
		t.Errorf("APT packages count mismatch: expected %d, got %d", 
			len(cache.PackageState.AptPackages), len(loaded.PackageState.AptPackages))
	}

	// Check specific package entries
	if entry, exists := loaded.PackageState.AptPackages["vim"]; !exists {
		t.Error("Expected vim package in cache, not found")
	} else if !entry.Installed {
		t.Error("Expected vim to be marked as installed")
	}

	if entry, exists := loaded.PackageState.AptPackages["git"]; !exists {
		t.Error("Expected git package in cache, not found")
	} else if entry.Installed {
		t.Error("Expected git to be marked as not installed")
	}
}

func TestCacheManager_CacheExpiration(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	logger := log.New(os.Stderr)
	cm := NewCacheManagerWithPath(logger, cacheDir)

	// Create old system state cache (older than 1 hour)
	oldCache := &SystemStateCache{
		PackageState: PackageInstallationState{
			AptPackages: make(map[string]PackageCacheEntry),
			LastUpdated: time.Now().Add(-2 * time.Hour),
		},
		LastChecked: time.Now().Add(-2 * time.Hour),
		SystemHash:  "old-hash",
		Version:     "1.0",
	}

	// Save old cache
	if err := cm.SaveSystemStateCache(oldCache); err != nil {
		t.Fatalf("SaveSystemStateCache() failed: %v", err)
	}

	// Try to load - should return nil due to expiration
	loaded, err := cm.LoadSystemStateCache()
	if err != nil {
		t.Fatalf("LoadSystemStateCache() failed: %v", err)
	}

	if loaded != nil {
		t.Error("Expected nil for expired cache, got cache data")
	}
}

func TestCacheManager_CacheStats(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	logger := log.New(os.Stderr)
	cm := NewCacheManagerWithPath(logger, cacheDir)

	// Get stats for empty cache
	stats, err := cm.GetCacheStats()
	if err != nil {
		t.Fatalf("GetCacheStats() failed: %v", err)
	}

	if stats.TotalFiles != 0 {
		t.Errorf("Expected 0 files in empty cache, got %d", stats.TotalFiles)
	}

	if stats.TotalSize != 0 {
		t.Errorf("Expected 0 size for empty cache, got %d", stats.TotalSize)
	}

	// Add some cache data
	cfg := &config.Config{Version: "1.0"}
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte("version: '1.0'"), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	if err := cm.SaveCachedConfig(cfg, []string{configPath}); err != nil {
		t.Fatalf("SaveCachedConfig() failed: %v", err)
	}

	cache := &SystemStateCache{
		PackageState: PackageInstallationState{
			AptPackages: make(map[string]PackageCacheEntry),
			LastUpdated: time.Now(),
		},
		LastChecked: time.Now(),
		SystemHash:  "test-hash",
		Version:     "1.0",
	}

	if err := cm.SaveSystemStateCache(cache); err != nil {
		t.Fatalf("SaveSystemStateCache() failed: %v", err)
	}

	// Get stats with cache data
	stats, err = cm.GetCacheStats()
	if err != nil {
		t.Fatalf("GetCacheStats() failed: %v", err)
	}

	if stats.TotalFiles == 0 {
		t.Error("Expected cache files, got 0")
	}

	if stats.TotalSize == 0 {
		t.Error("Expected cache size > 0, got 0")
	}

	if stats.CacheDir != cacheDir {
		t.Errorf("Cache dir mismatch: expected %s, got %s", cacheDir, stats.CacheDir)
	}
}

func TestCacheManager_ClearCache(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	cacheDir := filepath.Join(tmpDir, "cache")

	logger := log.New(os.Stderr)
	cm := NewCacheManagerWithPath(logger, cacheDir)

	// Add cache data
	cfg := &config.Config{Version: "1.0"}
	configPath := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(configPath, []byte("version: '1.0'"), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	if err := cm.SaveCachedConfig(cfg, []string{configPath}); err != nil {
		t.Fatalf("SaveCachedConfig() failed: %v", err)
	}

	// Verify cache exists
	stats, err := cm.GetCacheStats()
	if err != nil {
		t.Fatalf("GetCacheStats() failed: %v", err)
	}
	if stats.TotalFiles == 0 {
		t.Fatal("Expected cache files before clearing")
	}

	// Clear cache
	if err := cm.ClearCache(); err != nil {
		t.Fatalf("ClearCache() failed: %v", err)
	}

	// Verify cache is cleared
	stats, err = cm.GetCacheStats()
	if err != nil {
		t.Fatalf("GetCacheStats() failed: %v", err)
	}
	if stats.TotalFiles != 0 {
		t.Errorf("Expected 0 files after clearing cache, got %d", stats.TotalFiles)
	}

	// Verify cache directory no longer exists
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Error("Expected cache directory to be removed")
	}
}