package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// OptimizedLoader handles configuration loading with caching optimization
type OptimizedLoader struct {
	logger *log.Logger
	cache  *CacheManager
}

// NewOptimizedLoader creates a new optimized configuration loader
func NewOptimizedLoader(logger *log.Logger, cache *CacheManager) *OptimizedLoader {
	return &OptimizedLoader{
		logger: logger,
		cache:  cache,
	}
}

// LoadConfigurationOptimized loads configuration with caching optimization
func (ol *OptimizedLoader) LoadConfigurationOptimized(configPath string) (*config.Config, []string, error) {
	startTime := time.Now()
	ol.logger.Debug("Loading configuration with optimization", "config_path", configPath)

	// Resolve the main config path
	resolvedPath, err := ol.resolveConfigPath(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve config path: %w", err)
	}

	// Collect all config paths (including includes)
	configPaths, err := ol.collectAllConfigPaths(resolvedPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to collect config paths: %w", err)
	}

	// Try to load from cache first
	cachedConfig, err := ol.cache.LoadCachedConfig(configPaths)
	if err != nil {
		ol.logger.Warn("Failed to load cached config, falling back to standard loading", "error", err)
	}

	if cachedConfig != nil {
		loadTime := time.Since(startTime)
		ol.logger.Info("✓ Configuration loaded from cache", 
			"load_time", loadTime,
			"files", len(configPaths),
			"cache_age", time.Since(cachedConfig.CachedAt))
		return cachedConfig.Config, configPaths, nil
	}

	// Cache miss - load configuration normally
	ol.logger.Debug("Cache miss, loading configuration from files", "files", len(configPaths))
	
	// Set the main config file in viper
	viper.SetConfigFile(resolvedPath)
	if err := viper.ReadInConfig(); err != nil {
		return nil, nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Load configuration with advanced includes
	advancedLoader := config.NewAdvancedLoader()
	configResult, actualPaths, err := advancedLoader.LoadConfigurationAdvanced(resolvedPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config with includes: %w", err)
	}
	
	// Update configPaths with actually loaded paths
	configPaths = actualPaths

	// Cache the loaded configuration
	if err := ol.cache.SaveCachedConfig(configResult, configPaths); err != nil {
		ol.logger.Warn("Failed to cache configuration", "error", err)
		// Don't fail the operation for caching issues
	}

	loadTime := time.Since(startTime)
	ol.logger.Info("✓ Configuration loaded and cached", 
		"load_time", loadTime,
		"files", len(configPaths))

	return configResult, configPaths, nil
}

// resolveConfigPath resolves the configuration file path
func (ol *OptimizedLoader) resolveConfigPath(configPath string) (string, error) {
	if configPath != "" {
		// Explicit path provided
		abs, err := filepath.Abs(configPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path: %w", err)
		}
		return abs, nil
	}

	// Use viper's config file discovery
	if viper.ConfigFileUsed() != "" {
		return viper.ConfigFileUsed(), nil
	}

	// Fall back to standard search locations
	searchPaths := []string{
		"./configr.yaml",
		"~/.config/configr/configr.yaml",
		"~/configr.yaml",
		"/etc/configr/configr.yaml",
		"/usr/local/etc/configr/configr.yaml",
	}

	for _, path := range searchPaths {
		// Expand ~ if present
		if path[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			path = filepath.Join(home, path[1:])
		}

		if _, err := os.Stat(path); err == nil {
			abs, err := filepath.Abs(path)
			if err != nil {
				continue
			}
			return abs, nil
		}
	}

	return "", fmt.Errorf("no config file found in standard locations")
}

// collectAllConfigPaths collects all configuration file paths including includes
func (ol *OptimizedLoader) collectAllConfigPaths(mainConfigPath string) ([]string, error) {
	paths := []string{mainConfigPath}
	visited := make(map[string]bool)
	visited[mainConfigPath] = true

	// Parse the main config to find includes
	includePaths, err := ol.extractIncludePaths(mainConfigPath)
	if err != nil {
		ol.logger.Warn("Failed to extract include paths", "error", err)
		return paths, nil // Return what we have
	}

	// Recursively collect include paths
	if err := ol.collectIncludePaths(includePaths, filepath.Dir(mainConfigPath), &paths, visited); err != nil {
		ol.logger.Warn("Failed to collect all include paths", "error", err)
	}

	return paths, nil
}

// extractIncludePaths extracts include paths from a configuration file
func (ol *OptimizedLoader) extractIncludePaths(configPath string) ([]string, error) {
	// Parse the config file to get include specs
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var configData config.Config
	if err := yaml.Unmarshal(data, &configData); err != nil {
		return nil, err
	}

	// Extract paths from include specs
	var paths []string
	for _, spec := range configData.Includes {
		if spec.Path != "" {
			paths = append(paths, spec.Path)
		}
	}
	
	return paths, nil
}

// collectIncludePaths recursively collects include file paths
func (ol *OptimizedLoader) collectIncludePaths(includes []string, baseDir string, allPaths *[]string, visited map[string]bool) error {
	for _, include := range includes {
		// Convert include path to absolute
		var includePath string
		if filepath.IsAbs(include) {
			includePath = include
		} else {
			includePath = filepath.Join(baseDir, include)
		}

		// Handle directory includes (load default.yaml)
		if info, err := os.Stat(includePath); err == nil && info.IsDir() {
			includePath = filepath.Join(includePath, "default.yaml")
		} else if !strings.HasSuffix(includePath, ".yaml") && !strings.HasSuffix(includePath, ".yml") {
			// Try adding .yaml extension
			if _, err := os.Stat(includePath + ".yaml"); err == nil {
				includePath = includePath + ".yaml"
			}
		}

		// Convert to absolute path
		absPath, err := filepath.Abs(includePath)
		if err != nil {
			ol.logger.Warn("Failed to resolve include path", "path", includePath, "error", err)
			continue
		}

		// Check for circular includes
		if visited[absPath] {
			continue
		}

		// Check if file exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			ol.logger.Warn("Include file not found", "path", absPath)
			continue
		}

		// Add to paths and mark as visited
		*allPaths = append(*allPaths, absPath)
		visited[absPath] = true

		// Recursively process includes in this file
		nestedIncludes, err := ol.extractIncludePaths(absPath)
		if err != nil {
			ol.logger.Warn("Failed to extract nested includes", "path", absPath, "error", err)
			continue
		}

		if len(nestedIncludes) > 0 {
			if err := ol.collectIncludePaths(nestedIncludes, filepath.Dir(absPath), allPaths, visited); err != nil {
				ol.logger.Warn("Failed to collect nested includes", "error", err)
			}
		}
	}

	return nil
}

// InvalidateConfigCache invalidates cached configuration
func (ol *OptimizedLoader) InvalidateConfigCache(configPaths []string) error {
	ol.logger.Debug("Invalidating config cache", "paths", configPaths)
	
	// For now, we'll clear the entire cache since individual file invalidation
	// is complex with includes. A more sophisticated implementation could
	// track which cached configs depend on which files.
	return ol.cache.ClearCache()
}

// GetCacheStats returns cache statistics
func (ol *OptimizedLoader) GetCacheStats() (*CacheStats, error) {
	return ol.cache.GetCacheStats()
}