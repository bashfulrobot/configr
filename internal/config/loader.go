package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// LoadWithIncludes reads and parses the configuration file with include support
func LoadWithIncludes() (*Config, error) {
	rootConfigPath := viper.ConfigFileUsed()
	if rootConfigPath == "" {
		return nil, fmt.Errorf("no config file found")
	}

	visited := make(map[string]bool)
	return loadConfigRecursive(rootConfigPath, visited)
}

// loadConfigRecursive loads a config file and processes its includes
func loadConfigRecursive(configPath string, visited map[string]bool) (*Config, error) {
	// Prevent circular includes
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for %s: %w", configPath, err)
	}

	if visited[absPath] {
		return nil, fmt.Errorf("circular include detected: %s", absPath)
	}
	visited[absPath] = true

	// Load the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file %s: %w", configPath, err)
	}

	// Set ConfigDir for all file and binary entries in this config
	configDir := filepath.Dir(configPath)
	for name, file := range config.Files {
		file.ConfigDir = configDir
		config.Files[name] = file
	}
	for name, binary := range config.Binaries {
		binary.ConfigDir = configDir
		config.Binaries[name] = binary
	}

	// Process includes
	if len(config.Includes) > 0 {
		baseDir := filepath.Dir(configPath)
		
		for _, includePath := range config.Includes {
			resolvedPath, err := resolveIncludePath(baseDir, includePath.Path)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve include path %s: %w", includePath.Path, err)
			}

			includedConfig, err := loadConfigRecursive(resolvedPath, visited)
			if err != nil {
				return nil, fmt.Errorf("failed to load included config %s: %w", resolvedPath, err)
			}

			// Merge the included config into the current config
			if err := mergeConfigs(&config, includedConfig); err != nil {
				return nil, fmt.Errorf("failed to merge config %s: %w", resolvedPath, err)
			}
		}
	}

	// Clean up the visited map for this branch
	delete(visited, absPath)

	return &config, nil
}

// resolveIncludePath resolves include paths with support for directories and default.yaml
func resolveIncludePath(baseDir, includePath string) (string, error) {
	// Normalize trailing slash for consistency
	normalizedPath := includePath
	if strings.HasSuffix(includePath, "/") {
		// Remove trailing slash to avoid double slashes when joining
		normalizedPath = strings.TrimSuffix(includePath, "/")
	}
	
	fullPath := filepath.Join(baseDir, normalizedPath)

	// If the original path had a trailing slash, treat as directory
	if strings.HasSuffix(includePath, "/") {
		// Must be a directory, look for default.yaml
		if info, err := os.Stat(fullPath); err != nil || !info.IsDir() {
			return "", fmt.Errorf("directory %s not found", fullPath)
		}
		defaultPath := filepath.Join(fullPath, "default.yaml")
		if _, err := os.Stat(defaultPath); err == nil {
			return defaultPath, nil
		}
		return "", fmt.Errorf("directory %s exists but no default.yaml found", fullPath)
	}

	// Check if it's a directory (without trailing slash)
	if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
		// If directory, look for default.yaml
		defaultPath := filepath.Join(fullPath, "default.yaml")
		if _, err := os.Stat(defaultPath); err == nil {
			return defaultPath, nil
		}
		return "", fmt.Errorf("directory %s exists but no default.yaml found", fullPath)
	}

	// Check if explicit file exists
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath, nil
	}

	// If no extension, assume .yaml
	if filepath.Ext(fullPath) == "" {
		yamlPath := fullPath + ".yaml"
		if _, err := os.Stat(yamlPath); err == nil {
			return yamlPath, nil
		}
	}

	return "", fmt.Errorf("include path not found: %s", includePath)
}

// mergeConfigs merges src config into dst config
func mergeConfigs(dst, src *Config) error {
	// Merge packages (remove duplicates)
	dst.Packages.Apt = removeDuplicatePackages(append(dst.Packages.Apt, src.Packages.Apt...))
	dst.Packages.Flatpak = removeDuplicatePackages(append(dst.Packages.Flatpak, src.Packages.Flatpak...))
	dst.Packages.Snap = removeDuplicatePackages(append(dst.Packages.Snap, src.Packages.Snap...))

	// Merge files (src overwrites dst if same key)
	if dst.Files == nil {
		dst.Files = make(map[string]File)
	}
	for key, file := range src.Files {
		dst.Files[key] = file
	}

	// Merge repositories (append without duplicates by name)
	dst.Repositories.Apt = removeDuplicateRepositories(append(dst.Repositories.Apt, src.Repositories.Apt...))
	dst.Repositories.Flatpak = removeDuplicateFlatpakRepositories(append(dst.Repositories.Flatpak, src.Repositories.Flatpak...))

	// Merge binaries (src overwrites dst if same key)
	if dst.Binaries == nil {
		dst.Binaries = make(map[string]Binary)
	}
	for key, binary := range src.Binaries {
		dst.Binaries[key] = binary
	}

	// Merge dconf settings (src overwrites dst if same key)
	if dst.DConf.Settings == nil {
		dst.DConf.Settings = make(map[string]string)
	}
	for key, value := range src.DConf.Settings {
		dst.DConf.Settings[key] = value
	}

	return nil
}

// removeDuplicates removes duplicate strings from a slice while preserving order
func removeDuplicates(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))
	
	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// removeDuplicatePackages removes duplicate PackageEntry instances from a slice while preserving order
// Duplicates are determined by package name only (flags can differ)
func removeDuplicatePackages(slice []PackageEntry) []PackageEntry {
	seen := make(map[string]bool)
	result := make([]PackageEntry, 0, len(slice))
	
	for _, item := range slice {
		if !seen[item.Name] {
			seen[item.Name] = true
			result = append(result, item)
		}
	}
	
	return result
}

// removeDuplicateRepositories removes duplicate APT repositories by name while preserving order
// Duplicates are determined by repository name only
func removeDuplicateRepositories(slice []AptRepository) []AptRepository {
	seen := make(map[string]bool)
	result := make([]AptRepository, 0, len(slice))
	
	for _, item := range slice {
		if !seen[item.Name] {
			seen[item.Name] = true
			result = append(result, item)
		}
	}
	
	return result
}

// removeDuplicateFlatpakRepositories removes duplicate Flatpak repositories by name while preserving order
// Duplicates are determined by repository name only
func removeDuplicateFlatpakRepositories(slice []FlatpakRepository) []FlatpakRepository {
	seen := make(map[string]bool)
	result := make([]FlatpakRepository, 0, len(slice))
	
	for _, item := range slice {
		if !seen[item.Name] {
			seen[item.Name] = true
			result = append(result, item)
		}
	}
	
	return result
}