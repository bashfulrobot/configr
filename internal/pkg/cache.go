package pkg

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

// CacheManager handles configuration and system state caching for performance optimization
type CacheManager struct {
	logger    *log.Logger
	cacheDir  string
}

// CachedConfig represents a cached configuration with metadata
type CachedConfig struct {
	Config       *config.Config    `json:"config"`
	ConfigHash   string            `json:"config_hash"`
	ConfigPaths  []string          `json:"config_paths"`
	ModTimes     map[string]int64  `json:"mod_times"`
	CachedAt     time.Time         `json:"cached_at"`
	Version      string            `json:"version"`
}

// SystemStateCache represents cached system state information
type SystemStateCache struct {
	PackageState    PackageInstallationState `json:"package_state"`
	FileState       FileDeploymentState      `json:"file_state"`
	LastChecked     time.Time                `json:"last_checked"`
	SystemHash      string                   `json:"system_hash"`
	Version         string                   `json:"version"`
}

// PackageInstallationState caches package manager state
type PackageInstallationState struct {
	AptPackages     map[string]PackageCacheEntry `json:"apt_packages"`
	FlatpakPackages map[string]PackageCacheEntry `json:"flatpak_packages"`
	SnapPackages    map[string]PackageCacheEntry `json:"snap_packages"`
	LastUpdated     time.Time                    `json:"last_updated"`
}

// FileDeploymentState caches file system state
type FileDeploymentState struct {
	Files       map[string]FileCacheEntry `json:"files"`
	LastUpdated time.Time                 `json:"last_updated"`
}

// PackageCacheEntry represents a cached package state
type PackageCacheEntry struct {
	Name        string    `json:"name"`
	Installed   bool      `json:"installed"`
	Version     string    `json:"version,omitempty"`
	LastChecked time.Time `json:"last_checked"`
}

// FileCacheEntry represents a cached file state
type FileCacheEntry struct {
	Path         string    `json:"path"`
	ModTime      time.Time `json:"mod_time"`
	Size         int64     `json:"size"`
	IsSymlink    bool      `json:"is_symlink"`
	Target       string    `json:"target,omitempty"`
	Hash         string    `json:"hash,omitempty"`
	LastChecked  time.Time `json:"last_checked"`
}

// NewCacheManager creates a new cache manager
func NewCacheManager(logger *log.Logger) *CacheManager {
	// Default cache directory: ~/.cache/configr/
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Warn("Could not determine home directory, using /tmp for cache", "error", err)
		homeDir = "/tmp"
	}
	
	cacheDir := filepath.Join(homeDir, ".cache", "configr")
	
	return &CacheManager{
		logger:   logger,
		cacheDir: cacheDir,
	}
}

// NewCacheManagerWithPath creates a cache manager with custom cache directory
func NewCacheManagerWithPath(logger *log.Logger, cacheDir string) *CacheManager {
	return &CacheManager{
		logger:   logger,
		cacheDir: cacheDir,
	}
}

// LoadCachedConfig attempts to load a cached configuration
func (cm *CacheManager) LoadCachedConfig(configPaths []string) (*CachedConfig, error) {
	cm.logger.Debug("Attempting to load cached configuration", "paths", configPaths)
	
	// Generate cache key from config paths
	cacheKey := cm.generateConfigCacheKey(configPaths)
	cachePath := filepath.Join(cm.cacheDir, "config", cacheKey+".json")
	
	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		cm.logger.Debug("No cached configuration found", "cache_path", cachePath)
		return nil, nil
	}
	
	// Load cache file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}
	
	var cached CachedConfig
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("failed to parse cache file: %w", err)
	}
	
	// Validate cache is still valid
	if !cm.isCacheValid(&cached, configPaths) {
		cm.logger.Debug("Cached configuration is invalid, will regenerate")
		return nil, nil
	}
	
	cm.logger.Debug("Loaded cached configuration successfully", "cached_at", cached.CachedAt)
	return &cached, nil
}

// SaveCachedConfig saves a configuration to cache
func (cm *CacheManager) SaveCachedConfig(cfg *config.Config, configPaths []string) error {
	cm.logger.Debug("Saving configuration to cache", "paths", configPaths)
	
	// Ensure cache directory exists
	cacheConfigDir := filepath.Join(cm.cacheDir, "config")
	if err := os.MkdirAll(cacheConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	
	// Generate modification time map
	modTimes := make(map[string]int64)
	for _, path := range configPaths {
		if info, err := os.Stat(path); err == nil {
			modTimes[path] = info.ModTime().Unix()
		}
	}
	
	// Create cached config
	configHash := cm.generateConfigHash(cfg, configPaths)
	cached := CachedConfig{
		Config:      cfg,
		ConfigHash:  configHash,
		ConfigPaths: configPaths,
		ModTimes:    modTimes,
		CachedAt:    time.Now(),
		Version:     "1.0",
	}
	
	// Marshal to JSON
	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}
	
	// Write to cache file
	cacheKey := cm.generateConfigCacheKey(configPaths)
	cachePath := filepath.Join(cacheConfigDir, cacheKey+".json")
	
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}
	
	cm.logger.Debug("Configuration cached successfully", "cache_path", cachePath)
	return nil
}

// LoadSystemStateCache loads cached system state
func (cm *CacheManager) LoadSystemStateCache() (*SystemStateCache, error) {
	cachePath := filepath.Join(cm.cacheDir, "system_state.json")
	
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		cm.logger.Debug("No system state cache found")
		return nil, nil
	}
	
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read system state cache: %w", err)
	}
	
	var cache SystemStateCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse system state cache: %w", err)
	}
	
	// Check if cache is too old (default: 1 hour)
	if time.Since(cache.LastChecked) > time.Hour {
		cm.logger.Debug("System state cache is stale, will refresh")
		return nil, nil
	}
	
	cm.logger.Debug("Loaded system state cache successfully", "last_checked", cache.LastChecked)
	return &cache, nil
}

// SaveSystemStateCache saves system state to cache
func (cm *CacheManager) SaveSystemStateCache(cache *SystemStateCache) error {
	if err := os.MkdirAll(cm.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	
	cache.LastChecked = time.Now()
	cache.Version = "1.0"
	
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal system state cache: %w", err)
	}
	
	cachePath := filepath.Join(cm.cacheDir, "system_state.json")
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write system state cache: %w", err)
	}
	
	cm.logger.Debug("System state cache saved successfully")
	return nil
}

// ClearCache removes all cached data
func (cm *CacheManager) ClearCache() error {
	cm.logger.Info("Clearing all cache data", "cache_dir", cm.cacheDir)
	
	if err := os.RemoveAll(cm.cacheDir); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}
	
	return nil
}

// GetCacheStats returns information about cache usage
func (cm *CacheManager) GetCacheStats() (*CacheStats, error) {
	stats := &CacheStats{
		CacheDir: cm.cacheDir,
	}
	
	// Check if cache directory exists
	if _, err := os.Stat(cm.cacheDir); os.IsNotExist(err) {
		return stats, nil
	}
	
	// Walk cache directory to collect stats
	err := filepath.Walk(cm.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			stats.TotalFiles++
			stats.TotalSize += info.Size()
			
			if info.ModTime().After(stats.LastModified) {
				stats.LastModified = info.ModTime()
			}
		}
		
		return nil
	})
	
	return stats, err
}

// CacheStats represents cache usage statistics
type CacheStats struct {
	CacheDir     string    `json:"cache_dir"`
	TotalFiles   int       `json:"total_files"`
	TotalSize    int64     `json:"total_size"`
	LastModified time.Time `json:"last_modified"`
}

// isCacheValid checks if a cached configuration is still valid
func (cm *CacheManager) isCacheValid(cached *CachedConfig, configPaths []string) bool {
	// Check if config paths have changed
	if len(cached.ConfigPaths) != len(configPaths) {
		return false
	}
	
	for i, path := range configPaths {
		if i >= len(cached.ConfigPaths) || cached.ConfigPaths[i] != path {
			return false
		}
	}
	
	// Check modification times
	for _, path := range configPaths {
		info, err := os.Stat(path)
		if err != nil {
			return false
		}
		
		cachedModTime, exists := cached.ModTimes[path]
		if !exists || info.ModTime().Unix() != cachedModTime {
			return false
		}
	}
	
	return true
}

// generateConfigCacheKey creates a unique cache key for configuration paths
func (cm *CacheManager) generateConfigCacheKey(configPaths []string) string {
	hasher := sha256.New()
	for _, path := range configPaths {
		hasher.Write([]byte(path))
	}
	return hex.EncodeToString(hasher.Sum(nil))[:16] // Use first 16 chars
}

// generateConfigHash creates a hash of the configuration content
func (cm *CacheManager) generateConfigHash(cfg *config.Config, configPaths []string) string {
	hasher := sha256.New()
	
	// Hash config content
	if data, err := json.Marshal(cfg); err == nil {
		hasher.Write(data)
	}
	
	// Hash file modification times
	for _, path := range configPaths {
		hasher.Write([]byte(path))
		if info, err := os.Stat(path); err == nil {
			hasher.Write([]byte(fmt.Sprintf("%d", info.ModTime().Unix())))
		}
	}
	
	return hex.EncodeToString(hasher.Sum(nil))
}