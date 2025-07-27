package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigSplitter handles configuration splitting strategies
type ConfigSplitter struct {
	baseDir string
}

// NewConfigSplitter creates a new configuration splitter
func NewConfigSplitter(baseDir string) *ConfigSplitter {
	return &ConfigSplitter{
		baseDir: baseDir,
	}
}

// SplitStrategy represents different configuration splitting strategies
type SplitStrategy int

const (
	SplitByPackageManager SplitStrategy = iota
	SplitByDomain
	SplitByEnvironment
	SplitByHost
	SplitByFunction
)

// SplitConfig splits a configuration into multiple files based on the specified strategy
func (cs *ConfigSplitter) SplitConfig(config *Config, strategy SplitStrategy) (map[string]*Config, error) {
	switch strategy {
	case SplitByPackageManager:
		return cs.splitByPackageManager(config)
	case SplitByDomain:
		return cs.splitByDomain(config)
	case SplitByEnvironment:
		return cs.splitByEnvironment(config)
	case SplitByHost:
		return cs.splitByHost(config)
	case SplitByFunction:
		return cs.splitByFunction(config)
	default:
		return nil, fmt.Errorf("unknown split strategy: %d", strategy)
	}
}

// splitByPackageManager splits configuration by package managers
func (cs *ConfigSplitter) splitByPackageManager(config *Config) (map[string]*Config, error) {
	result := make(map[string]*Config)

	// Create base config with common settings
	baseConfig := &Config{
		Version:         config.Version,
		PackageDefaults: config.PackageDefaults,
		BackupPolicy:    config.BackupPolicy,
		Includes:        []IncludeSpec{},
	}

	// Split APT packages
	if len(config.Packages.Apt) > 0 {
		aptConfig := &Config{
			Version: config.Version,
			Packages: PackageManagement{
				Apt: config.Packages.Apt,
			},
		}
		result["packages/apt.yaml"] = aptConfig
		baseConfig.Includes = append(baseConfig.Includes, IncludeSpec{
			Path:        "packages/apt.yaml",
			Description: "APT package management",
		})
	}

	// Split Flatpak packages
	if len(config.Packages.Flatpak) > 0 {
		flatpakConfig := &Config{
			Version: config.Version,
			Packages: PackageManagement{
				Flatpak: config.Packages.Flatpak,
			},
		}
		result["packages/flatpak.yaml"] = flatpakConfig
		baseConfig.Includes = append(baseConfig.Includes, IncludeSpec{
			Path:        "packages/flatpak.yaml",
			Description: "Flatpak package management",
		})
	}

	// Split Snap packages
	if len(config.Packages.Snap) > 0 {
		snapConfig := &Config{
			Version: config.Version,
			Packages: PackageManagement{
				Snap: config.Packages.Snap,
			},
		}
		result["packages/snap.yaml"] = snapConfig
		baseConfig.Includes = append(baseConfig.Includes, IncludeSpec{
			Path:        "packages/snap.yaml",
			Description: "Snap package management",
		})
	}

	// Split repositories
	if len(config.Repositories.Apt) > 0 || len(config.Repositories.Flatpak) > 0 {
		repoConfig := &Config{
			Version:      config.Version,
			Repositories: config.Repositories,
		}
		result["repositories.yaml"] = repoConfig
		baseConfig.Includes = append(baseConfig.Includes, IncludeSpec{
			Path:        "repositories.yaml",
			Description: "Repository management",
		})
	}

	// Split files
	if len(config.Files) > 0 {
		filesConfig := &Config{
			Version: config.Version,
			Files:   config.Files,
		}
		result["files.yaml"] = filesConfig
		baseConfig.Includes = append(baseConfig.Includes, IncludeSpec{
			Path:        "files.yaml",
			Description: "File management",
		})
	}

	// Split DConf settings
	if len(config.DConf.Settings) > 0 {
		dconfConfig := &Config{
			Version: config.Version,
			DConf:   config.DConf,
		}
		result["dconf.yaml"] = dconfConfig
		baseConfig.Includes = append(baseConfig.Includes, IncludeSpec{
			Path:        "dconf.yaml",
			Description: "DConf settings management",
		})
	}

	result["configr.yaml"] = baseConfig
	return result, nil
}

// splitByDomain splits configuration by functional domains
func (cs *ConfigSplitter) splitByDomain(config *Config) (map[string]*Config, error) {
	result := make(map[string]*Config)

	// Create base config
	baseConfig := &Config{
		Version:         config.Version,
		PackageDefaults: config.PackageDefaults,
		BackupPolicy:    config.BackupPolicy,
		Includes:        []IncludeSpec{},
	}

	// Development tools domain
	devPackages := cs.filterPackagesByDomain(config.Packages, "development")
	if len(devPackages.Apt) > 0 || len(devPackages.Flatpak) > 0 || len(devPackages.Snap) > 0 {
		devConfig := &Config{
			Version:  config.Version,
			Packages: devPackages,
		}
		result["domains/development.yaml"] = devConfig
		baseConfig.Includes = append(baseConfig.Includes, IncludeSpec{
			Path:        "domains/development.yaml",
			Description: "Development tools and environments",
		})
	}

	// Media domain
	mediaPackages := cs.filterPackagesByDomain(config.Packages, "media")
	if len(mediaPackages.Apt) > 0 || len(mediaPackages.Flatpak) > 0 || len(mediaPackages.Snap) > 0 {
		mediaConfig := &Config{
			Version:  config.Version,
			Packages: mediaPackages,
		}
		result["domains/media.yaml"] = mediaConfig
		baseConfig.Includes = append(baseConfig.Includes, IncludeSpec{
			Path:        "domains/media.yaml",
			Description: "Media tools and applications",
		})
	}

	// System utilities domain
	systemPackages := cs.filterPackagesByDomain(config.Packages, "system")
	if len(systemPackages.Apt) > 0 || len(systemPackages.Flatpak) > 0 || len(systemPackages.Snap) > 0 {
		systemConfig := &Config{
			Version:  config.Version,
			Packages: systemPackages,
		}
		result["domains/system.yaml"] = systemConfig
		baseConfig.Includes = append(baseConfig.Includes, IncludeSpec{
			Path:        "domains/system.yaml",
			Description: "System utilities and tools",
		})
	}

	result["configr.yaml"] = baseConfig
	return result, nil
}

// splitByEnvironment splits configuration by environment (dev, staging, prod)
func (cs *ConfigSplitter) splitByEnvironment(config *Config) (map[string]*Config, error) {
	result := make(map[string]*Config)

	// Create base config with common settings
	baseConfig := &Config{
		Version:         config.Version,
		PackageDefaults: config.PackageDefaults,
		BackupPolicy:    config.BackupPolicy,
		Repositories:    config.Repositories,
		Includes: []IncludeSpec{
			{
				Path:        "environments/common.yaml",
				Description: "Common packages and settings",
			},
			{
				Path:        "environments/development.yaml",
				Description: "Development environment specific settings",
				Optional:    true,
				Conditions: []IncludeCondition{
					{
						Type:     "env",
						Value:    "NODE_ENV=development",
						Operator: "equals",
					},
				},
			},
			{
				Path:        "environments/production.yaml",
				Description: "Production environment specific settings",
				Optional:    true,
				Conditions: []IncludeCondition{
					{
						Type:     "env",
						Value:    "NODE_ENV=production",
						Operator: "equals",
					},
				},
			},
		},
	}

	// Common packages (subset of original)
	commonPackages := cs.getCommonPackages(config.Packages)
	commonConfig := &Config{
		Version:  config.Version,
		Packages: commonPackages,
		Files:    config.Files, // Most files are common
		DConf:    config.DConf, // DConf settings are usually common
	}
	result["environments/common.yaml"] = commonConfig

	// Development-specific packages
	devPackages := cs.getDevelopmentPackages(config.Packages)
	if len(devPackages.Apt) > 0 || len(devPackages.Flatpak) > 0 || len(devPackages.Snap) > 0 {
		devConfig := &Config{
			Version:  config.Version,
			Packages: devPackages,
		}
		result["environments/development.yaml"] = devConfig
	}

	// Production-specific packages (minimal)
	prodPackages := cs.getProductionPackages(config.Packages)
	if len(prodPackages.Apt) > 0 || len(prodPackages.Flatpak) > 0 || len(prodPackages.Snap) > 0 {
		prodConfig := &Config{
			Version:  config.Version,
			Packages: prodPackages,
		}
		result["environments/production.yaml"] = prodConfig
	}

	result["configr.yaml"] = baseConfig
	return result, nil
}

// splitByHost splits configuration by hostname patterns
func (cs *ConfigSplitter) splitByHost(config *Config) (map[string]*Config, error) {
	result := make(map[string]*Config)

	// Create base config with common settings
	baseConfig := &Config{
		Version:         config.Version,
		PackageDefaults: config.PackageDefaults,
		BackupPolicy:    config.BackupPolicy,
		Repositories:    config.Repositories,
		Includes: []IncludeSpec{
			{
				Path:        "hosts/common.yaml",
				Description: "Common configuration for all hosts",
			},
			{
				Path:        "hosts/workstation-*.yaml",
				Description: "Workstation-specific configuration",
				Optional:    true,
				Conditions: []IncludeCondition{
					{
						Type:     "hostname",
						Value:    "workstation",
						Operator: "contains",
					},
				},
			},
			{
				Path:        "hosts/laptop-*.yaml",
				Description: "Laptop-specific configuration",
				Optional:    true,
				Conditions: []IncludeCondition{
					{
						Type:     "hostname",
						Value:    "laptop",
						Operator: "contains",
					},
				},
			},
			{
				Path:        "hosts/server-*.yaml",
				Description: "Server-specific configuration",
				Optional:    true,
				Conditions: []IncludeCondition{
					{
						Type:     "hostname",
						Value:    "server",
						Operator: "contains",
					},
				},
			},
		},
	}

	// Common configuration
	commonConfig := &Config{
		Version:  config.Version,
		Packages: cs.getCommonPackages(config.Packages),
		Files:    cs.getCommonFiles(config.Files),
		DConf:    config.DConf,
	}
	result["hosts/common.yaml"] = commonConfig

	// Workstation-specific
	workstationConfig := &Config{
		Version:  config.Version,
		Packages: cs.getWorkstationPackages(config.Packages),
		Files:    cs.getWorkstationFiles(config.Files),
	}
	result["hosts/workstation-packages.yaml"] = workstationConfig

	result["configr.yaml"] = baseConfig
	return result, nil
}

// splitByFunction splits configuration by functional areas
func (cs *ConfigSplitter) splitByFunction(config *Config) (map[string]*Config, error) {
	result := make(map[string]*Config)

	// Create base config
	baseConfig := &Config{
		Version:         config.Version,
		PackageDefaults: config.PackageDefaults,
		BackupPolicy:    config.BackupPolicy,
		Includes: []IncludeSpec{
			{Path: "functions/repositories.yaml", Description: "Repository management"},
			{Path: "functions/system-packages.yaml", Description: "Core system packages"},
			{Path: "functions/development.yaml", Description: "Development tools"},
			{Path: "functions/desktop.yaml", Description: "Desktop applications"},
			{Path: "functions/dotfiles.yaml", Description: "Dotfiles and configuration files"},
			{Path: "functions/desktop-settings.yaml", Description: "Desktop environment settings"},
		},
	}

	// Split by function
	result["functions/repositories.yaml"] = &Config{Version: config.Version, Repositories: config.Repositories}
	result["functions/system-packages.yaml"] = &Config{Version: config.Version, Packages: cs.getSystemPackages(config.Packages)}
	result["functions/development.yaml"] = &Config{Version: config.Version, Packages: cs.getDevelopmentPackages(config.Packages)}
	result["functions/desktop.yaml"] = &Config{Version: config.Version, Packages: cs.getDesktopPackages(config.Packages)}
	result["functions/dotfiles.yaml"] = &Config{Version: config.Version, Files: config.Files}
	result["functions/desktop-settings.yaml"] = &Config{Version: config.Version, DConf: config.DConf}

	result["configr.yaml"] = baseConfig
	return result, nil
}

// WriteConfigFiles writes the split configuration files to disk
func (cs *ConfigSplitter) WriteConfigFiles(configs map[string]*Config) error {
	for fileName, config := range configs {
		filePath := filepath.Join(cs.baseDir, fileName)
		
		// Create directory if needed
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		// Marshal config to YAML
		data, err := yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal config for %s: %w", fileName, err)
		}

		// Write file
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write config file %s: %w", filePath, err)
		}
	}

	return nil
}

// Helper methods for package filtering

func (cs *ConfigSplitter) filterPackagesByDomain(packages PackageManagement, domain string) PackageManagement {
	// This is a simplified implementation. In a real scenario, you'd have
	// a database or configuration mapping packages to domains.
	domainPackages := map[string][]string{
		"development": {"git", "vim", "code", "nodejs", "python3", "build-essential", "docker.io"},
		"media":       {"vlc", "gimp", "audacity", "ffmpeg", "obs-studio"},
		"system":      {"htop", "tree", "curl", "wget", "unzip", "neofetch"},
	}

	packageList := domainPackages[domain]
	if packageList == nil {
		return PackageManagement{}
	}

	result := PackageManagement{}
	
	// Filter APT packages
	for _, pkg := range packages.Apt {
		if contains(packageList, pkg.Name) {
			result.Apt = append(result.Apt, pkg)
		}
	}

	// Filter Flatpak packages
	for _, pkg := range packages.Flatpak {
		if contains(packageList, pkg.Name) {
			result.Flatpak = append(result.Flatpak, pkg)
		}
	}

	// Filter Snap packages
	for _, pkg := range packages.Snap {
		if contains(packageList, pkg.Name) {
			result.Snap = append(result.Snap, pkg)
		}
	}

	return result
}

func (cs *ConfigSplitter) getCommonPackages(packages PackageManagement) PackageManagement {
	// Return essential packages that should be on all systems
	commonPkgs := []string{"git", "curl", "wget", "unzip", "htop", "tree"}
	return cs.filterPackagesByNames(packages, commonPkgs)
}

func (cs *ConfigSplitter) getDevelopmentPackages(packages PackageManagement) PackageManagement {
	devPkgs := []string{"code", "nodejs", "python3", "build-essential", "docker.io", "vim"}
	return cs.filterPackagesByNames(packages, devPkgs)
}

func (cs *ConfigSplitter) getProductionPackages(packages PackageManagement) PackageManagement {
	prodPkgs := []string{"htop", "curl", "wget"}
	return cs.filterPackagesByNames(packages, prodPkgs)
}

func (cs *ConfigSplitter) getSystemPackages(packages PackageManagement) PackageManagement {
	systemPkgs := []string{"htop", "tree", "curl", "wget", "unzip", "neofetch"}
	return cs.filterPackagesByNames(packages, systemPkgs)
}

func (cs *ConfigSplitter) getDesktopPackages(packages PackageManagement) PackageManagement {
	desktopPkgs := []string{"firefox", "thunderbird", "libreoffice", "gimp"}
	return cs.filterPackagesByNames(packages, desktopPkgs)
}

func (cs *ConfigSplitter) getWorkstationPackages(packages PackageManagement) PackageManagement {
	workstationPkgs := []string{"code", "docker.io", "kubernetes", "terraform"}
	return cs.filterPackagesByNames(packages, workstationPkgs)
}

func (cs *ConfigSplitter) filterPackagesByNames(packages PackageManagement, names []string) PackageManagement {
	result := PackageManagement{}

	for _, pkg := range packages.Apt {
		if contains(names, pkg.Name) {
			result.Apt = append(result.Apt, pkg)
		}
	}

	for _, pkg := range packages.Flatpak {
		if contains(names, pkg.Name) {
			result.Flatpak = append(result.Flatpak, pkg)
		}
	}

	for _, pkg := range packages.Snap {
		if contains(names, pkg.Name) {
			result.Snap = append(result.Snap, pkg)
		}
	}

	return result
}

func (cs *ConfigSplitter) getCommonFiles(files map[string]File) map[string]File {
	// Filter files that are common across all hosts
	common := make(map[string]File)
	for name, file := range files {
		// Basic heuristic: files in home directory are usually common
		if strings.HasPrefix(file.Destination, "~/") || strings.Contains(name, "bashrc") || strings.Contains(name, "vimrc") {
			common[name] = file
		}
	}
	return common
}

func (cs *ConfigSplitter) getWorkstationFiles(files map[string]File) map[string]File {
	// Filter files specific to workstations
	workstation := make(map[string]File)
	for name, file := range files {
		// Basic heuristic: development-related files
		if strings.Contains(name, "code") || strings.Contains(name, "docker") || strings.Contains(file.Source, "dev") {
			workstation[name] = file
		}
	}
	return workstation
}

// GenerateSplitReport generates a report of the split configuration
func (cs *ConfigSplitter) GenerateSplitReport(configs map[string]*Config) string {
	var report strings.Builder
	
	report.WriteString("Configuration Split Report\n")
	report.WriteString("==========================\n\n")

	// Sort file names for consistent output
	var fileNames []string
	for fileName := range configs {
		fileNames = append(fileNames, fileName)
	}
	sort.Strings(fileNames)

	for _, fileName := range fileNames {
		config := configs[fileName]
		report.WriteString(fmt.Sprintf("File: %s\n", fileName))
		report.WriteString(fmt.Sprintf("  Version: %s\n", config.Version))
		
		if len(config.Includes) > 0 {
			report.WriteString(fmt.Sprintf("  Includes: %d files\n", len(config.Includes)))
		}
		
		aptCount := len(config.Packages.Apt)
		flatpakCount := len(config.Packages.Flatpak)
		snapCount := len(config.Packages.Snap)
		totalPackages := aptCount + flatpakCount + snapCount
		
		if totalPackages > 0 {
			report.WriteString(fmt.Sprintf("  Packages: %d total (APT: %d, Flatpak: %d, Snap: %d)\n", 
				totalPackages, aptCount, flatpakCount, snapCount))
		}
		
		if len(config.Files) > 0 {
			report.WriteString(fmt.Sprintf("  Files: %d\n", len(config.Files)))
		}
		
		if len(config.DConf.Settings) > 0 {
			report.WriteString(fmt.Sprintf("  DConf Settings: %d\n", len(config.DConf.Settings)))
		}
		
		if len(config.Repositories.Apt) > 0 || len(config.Repositories.Flatpak) > 0 {
			report.WriteString(fmt.Sprintf("  Repositories: APT: %d, Flatpak: %d\n", 
				len(config.Repositories.Apt), len(config.Repositories.Flatpak)))
		}
		
		report.WriteString("\n")
	}

	return report.String()
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}