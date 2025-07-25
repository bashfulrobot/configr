package configr

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/bashfulrobot/configr/internal/pkg"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	dryRun          bool
	removePackages  bool
	useOptimization bool
)

var applyCmd = &cobra.Command{
	Use:   "apply [config-file]",
	Short: "Apply configuration changes to the system",
	Long: `Apply loads and applies the configuration to your system.

This command will:
- Remove packages no longer in configuration (if --remove-packages=true)
- Add APT and Flatpak repositories
- Deploy and symlink files to their destinations
- Install APT, Flatpak, and Snap packages
- Apply dconf settings for desktop configuration
- Create backups of existing files when requested
- Track package state for future removal operations

By default, it looks for 'configr.yaml' in standard locations.`,
	Example: `  configr apply                         # Apply default config
  configr apply my-config.yaml          # Apply specific config
  configr apply --dry-run               # Preview changes without applying
  configr apply --remove-packages=false # Skip package removal
  configr apply --optimize=false        # Disable caching and optimization
  configr --config custom.yaml apply    # Use custom config file`,
	Args: cobra.MaximumNArgs(1),
	RunE: runApply,
}

func init() {
	rootCmd.AddCommand(applyCmd)
	
	// Command-specific flags
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without applying them")
	applyCmd.Flags().BoolVar(&removePackages, "remove-packages", true, "remove packages that are no longer in configuration")
	applyCmd.Flags().BoolVar(&useOptimization, "optimize", true, "enable caching and optimization for faster runs")
}

func runApply(cmd *cobra.Command, args []string) error {
	// Initialize logger
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: false,
		Prefix:          "configr",
	})

	// Set log level based on verbose flag
	if viper.GetBool("verbose") {
		logger.SetLevel(log.DebugLevel)
	}

	// Disable color if requested  
	// Note: color profile setting would require termenv import

	// Determine config file path
	var configPath string
	if len(args) > 0 {
		configPath = args[0]
	} else if viper.GetString("config") != "" {
		configPath = viper.GetString("config")
	} else {
		// Find config file in standard locations
		var err error
		configPath, err = findConfigFile()
		if err != nil {
			return fmt.Errorf("failed to find config file: %w", err)
		}
	}

	// Load configuration with optimization if enabled
	var cfg *config.Config
	var configPaths []string
	var err error

	if useOptimization {
		logger.Info("Loading configuration with optimization", "file", configPath)
		
		// Initialize cache manager
		cacheManager := pkg.NewCacheManager(logger)
		optimizedLoader := config.NewOptimizedLoader(logger, cacheManager)
		
		cfg, configPaths, err = optimizedLoader.LoadConfigurationOptimized(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config with optimization: %w", err)
		}
	} else {
		logger.Info("Loading configuration (standard mode)", "file", configPath)
		
		// Standard loading
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		
		cfg, err = config.LoadWithIncludes()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		configPaths = []string{configPath}
	}

	// Validate configuration
	logger.Debug("Validating configuration")
	result := config.Validate(cfg, configPath)
	if result.HasErrors() {
		fmt.Fprint(os.Stderr, config.FormatValidationResultSimple(result))
		return fmt.Errorf("configuration validation failed")
	}

	// Show warnings if any
	if len(result.Warnings) > 0 {
		fmt.Fprint(os.Stderr, config.FormatValidationResultSimple(result))
	}

	if dryRun {
		logger.Info("🏃 Running in dry-run mode - no changes will be made")
	}

	// Get config directory for relative path resolution
	configDir := filepath.Dir(configPath)

	// Apply repository configurations first (may be needed for package installations)
	if err := applyRepositoryConfigurations(cfg, logger, dryRun); err != nil {
		return fmt.Errorf("failed to apply repository configurations: %w", err)
	}

	// Apply file configurations
	var deployedFiles []pkg.ManagedFile
	if len(cfg.Files) > 0 {
		logger.Info("Applying file configurations")
		fileManager := pkg.NewFileManager(logger, dryRun, configDir)
		
		// Validate file permissions before proceeding
		if err := fileManager.ValidateFilePermissions(cfg.Files); err != nil {
			return fmt.Errorf("permission validation failed: %w", err)
		}

		var err error
		deployedFiles, err = fileManager.DeployFiles(cfg.Files)
		if err != nil {
			return fmt.Errorf("failed to deploy files: %w", err)
		}
	}

	// Apply package configurations
	if err := applyPackageConfigurations(cfg, deployedFiles, logger, dryRun, useOptimization); err != nil {
		return fmt.Errorf("failed to apply package configurations: %w", err)
	}

	// Apply dconf configurations
	if len(cfg.DConf.Settings) > 0 {
		logger.Debug("Applying dconf configurations", "count", len(cfg.DConf.Settings))
		dconfManager := pkg.NewDConfManager(logger, dryRun)
		
		// Validate dconf settings before applying
		if err := dconfManager.ValidateSettings(cfg.DConf); err != nil {
			return fmt.Errorf("dconf validation failed: %w", err)
		}
		
		if err := dconfManager.ApplySettings(cfg.DConf); err != nil {
			return fmt.Errorf("failed to apply dconf settings: %w", err)
		}
	}

	if dryRun {
		logger.Info("✓ Dry run completed - no actual changes were made")
	} else {
		logger.Info("✓ Configuration applied successfully")
	}

	return nil
}

// findConfigFile searches for a config file in standard locations
func findConfigFile() (string, error) {
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
			return path, nil
		}
	}

	return "", fmt.Errorf("no config file found in standard locations: %v", searchPaths)
}

// applyPackageConfigurations handles package management for all supported package managers
func applyPackageConfigurations(cfg *config.Config, deployedFiles []pkg.ManagedFile, logger *log.Logger, dryRun bool, useOptimization bool) error {
	// Initialize state manager for package removal tracking
	stateManager := pkg.NewStateManager(logger)
	
	// Get packages to remove (packages in previous state but not in current config)
	packagesToRemove, err := stateManager.GetPackagesToRemove(cfg)
	if err != nil {
		logger.Warn("Could not determine packages to remove", "error", err)
		// Continue with installation even if removal tracking fails
		packagesToRemove = &pkg.ManagedPackages{}
	}

	// Get files to remove (files in previous state but not in current config)
	filesToRemove, err := stateManager.GetFilesToRemove(cfg)
	if err != nil {
		logger.Warn("Could not determine files to remove", "error", err)
		// Continue with deployment even if file removal tracking fails
		filesToRemove = []pkg.ManagedFile{}
	}
	
	// Remove packages and files that are no longer in configuration (if enabled)
	if removePackages {
		if err := removePackagesNotInConfig(packagesToRemove, logger, dryRun); err != nil {
			return fmt.Errorf("failed to remove packages: %w", err)
		}
		if err := removeFilesNotInConfig(filesToRemove, configDir, logger, dryRun); err != nil {
			return fmt.Errorf("failed to remove files: %w", err)
		}
	} else {
		logger.Debug("Package and file removal disabled by --remove-packages=false flag")
	}
	// Handle APT packages
	if len(cfg.Packages.Apt) > 0 {
		logger.Debug("Applying APT package configurations", "count", len(cfg.Packages.Apt))
		
		if useOptimization {
			cacheManager := pkg.NewCacheManager(logger)
			aptManager := pkg.NewOptimizedAptManager(logger, dryRun, cacheManager)
			if err := aptManager.InstallPackagesOptimized(cfg.Packages.Apt, cfg.PackageDefaults); err != nil {
				return fmt.Errorf("APT package installation failed: %w", err)
			}
		} else {
			aptManager := pkg.NewAptManager(logger, dryRun)
			if err := aptManager.InstallPackages(cfg.Packages.Apt, cfg.PackageDefaults); err != nil {
				return fmt.Errorf("APT package installation failed: %w", err)
			}
		}
	}

	// Handle Flatpak packages
	if len(cfg.Packages.Flatpak) > 0 {
		logger.Debug("Applying Flatpak package configurations", "count", len(cfg.Packages.Flatpak))
		flatpakManager := pkg.NewFlatpakManager(logger, dryRun)
		
		// Validate Flatpak package names
		if err := flatpakManager.ValidatePackageNames(cfg.Packages.Flatpak); err != nil {
			return fmt.Errorf("Flatpak package validation failed: %w", err)
		}
		
		if err := flatpakManager.InstallPackages(cfg.Packages.Flatpak, cfg.PackageDefaults); err != nil {
			return fmt.Errorf("Flatpak package installation failed: %w", err)
		}
	}

	// Handle Snap packages
	if len(cfg.Packages.Snap) > 0 {
		logger.Debug("Applying Snap package configurations", "count", len(cfg.Packages.Snap))
		snapManager := pkg.NewSnapManager(logger, dryRun)
		
		// Validate Snap package names
		if err := snapManager.ValidatePackageNames(cfg.Packages.Snap); err != nil {
			return fmt.Errorf("Snap package validation failed: %w", err)
		}
		
		if err := snapManager.InstallPackages(cfg.Packages.Snap, cfg.PackageDefaults); err != nil {
			return fmt.Errorf("Snap package installation failed: %w", err)
		}
	}

	// Update state file with current configuration (only if not dry-run)
	if !dryRun {
		if err := stateManager.UpdateState(cfg, deployedFiles); err != nil {
			logger.Warn("Failed to update state", "error", err)
			// Don't fail the entire operation for state tracking issues
		}
	}

	return nil
}

// removePackagesNotInConfig removes packages that are no longer in the configuration
func removePackagesNotInConfig(packagesToRemove *pkg.ManagedPackages, logger *log.Logger, dryRun bool) error {
	// Remove APT packages
	if len(packagesToRemove.Apt) > 0 {
		logger.Info("Removing APT packages no longer in configuration", "count", len(packagesToRemove.Apt))
		aptManager := pkg.NewAptManager(logger, dryRun)
		if err := aptManager.RemovePackages(packagesToRemove.Apt); err != nil {
			return fmt.Errorf("APT package removal failed: %w", err)
		}
	}

	// Remove Flatpak packages
	if len(packagesToRemove.Flatpak) > 0 {
		logger.Info("Removing Flatpak packages no longer in configuration", "count", len(packagesToRemove.Flatpak))
		flatpakManager := pkg.NewFlatpakManager(logger, dryRun)
		if err := flatpakManager.RemovePackages(packagesToRemove.Flatpak); err != nil {
			return fmt.Errorf("Flatpak package removal failed: %w", err)
		}
	}

	// Remove Snap packages
	if len(packagesToRemove.Snap) > 0 {
		logger.Info("Removing Snap packages no longer in configuration", "count", len(packagesToRemove.Snap))
		snapManager := pkg.NewSnapManager(logger, dryRun)
		if err := snapManager.RemovePackages(packagesToRemove.Snap); err != nil {
			return fmt.Errorf("Snap package removal failed: %w", err)
		}
	}

	return nil
}

// removeFilesNotInConfig removes files that are no longer in the configuration
func removeFilesNotInConfig(filesToRemove []pkg.ManagedFile, configDir string, logger *log.Logger, dryRun bool) error {
	if len(filesToRemove) == 0 {
		return nil
	}

	logger.Info("Removing files no longer in configuration", "count", len(filesToRemove))
	fileManager := pkg.NewFileManager(logger, dryRun, configDir)
	
	return fileManager.RemoveFiles(filesToRemove)
}

// applyRepositoryConfigurations handles repository management for all supported repository types
func applyRepositoryConfigurations(cfg *config.Config, logger *log.Logger, dryRun bool) error {
	// Check if there are any repositories to process
	if len(cfg.Repositories.Apt) == 0 && len(cfg.Repositories.Flatpak) == 0 {
		logger.Debug("No repositories to process")
		return nil
	}

	logger.Debug("Applying repository configurations", 
		"apt_count", len(cfg.Repositories.Apt), 
		"flatpak_count", len(cfg.Repositories.Flatpak))
	
	repoManager := pkg.NewRepositoryManager(logger, dryRun)
	if err := repoManager.AddRepositories(cfg.Repositories); err != nil {
		return fmt.Errorf("repository management failed: %w", err)
	}

	return nil
}