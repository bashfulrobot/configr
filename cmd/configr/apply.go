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
	interactiveMode bool
	showPreview     bool
)

var applyCmd = &cobra.Command{
	Use:   "apply [config-file]",
	Short: "Apply configuration changes to the system",
	Long: `Apply loads and applies the configuration to your system.

This command will:
- Remove packages no longer in configuration (if --remove-packages=true)
- Add APT and Flatpak repositories
- Deploy and symlink files to their destinations
- Download and deploy binaries from remote repositories
- Install APT, Flatpak, and Snap packages
- Apply dconf settings for desktop configuration
- Create backups of existing files and binaries when requested
- Track package state for future removal operations
- Interactively resolve file and binary conflicts when --interactive flag is used

Interactive features include:
- Conflict resolution prompts for existing files and binaries
- File diff preview before replacement
- Interactive permission and ownership configuration

By default, it looks for 'configr.yaml' in standard locations.`,
	Example: `  configr apply                         # Apply default config
  configr apply my-config.yaml          # Apply specific config
  configr apply --dry-run               # Preview changes without applying
  configr apply --interactive           # Enable interactive prompts
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
	applyCmd.Flags().BoolVar(&interactiveMode, "interactive", false, "enable interactive prompts for conflicts and permissions")
	applyCmd.Flags().BoolVar(&showPreview, "preview", false, "show configuration preview before applying")
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

	// Initialize UX manager for enhanced user experience
	uxManager := pkg.NewUXManager(logger, dryRun)

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

	// Load configuration with enhanced UX
	var cfg *config.Config
	var err error
	configDir := filepath.Dir(configPath)

	// Show loading spinner for configuration
	loadSpinner, loadDone := uxManager.ShowConfigLoadSpinner(useOptimization)
	
	if useOptimization {
		logger.Debug("Loading configuration with optimization", "file", configPath)
		
		// Initialize cache manager
		cacheManager := pkg.NewCacheManager(logger)
		optimizedLoader := pkg.NewOptimizedLoader(logger, cacheManager)
		
		cfg, _, err = optimizedLoader.LoadConfigurationOptimized(configPath)
		if err != nil {
			if loadSpinner != nil {
				loadDone <- pkg.SpinnerDoneMsg{Success: false, Error: err}
				loadSpinner.Kill()
			}
			return fmt.Errorf("failed to load config with optimization: %w", err)
		}
	} else {
		logger.Debug("Loading configuration (standard mode)", "file", configPath)
		
		// Standard loading
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			if loadSpinner != nil {
				loadDone <- pkg.SpinnerDoneMsg{Success: false, Error: err}
				loadSpinner.Kill()
			}
			return fmt.Errorf("failed to read config file: %w", err)
		}
		
		cfg, err = config.LoadWithIncludes()
		if err != nil {
			if loadSpinner != nil {
				loadDone <- pkg.SpinnerDoneMsg{Success: false, Error: err}
				loadSpinner.Kill()
			}
			return fmt.Errorf("failed to load config: %w", err)
		}
	}
	
	// Complete loading spinner
	if loadSpinner != nil {
		loadDone <- pkg.SpinnerDoneMsg{Success: true}
		loadSpinner.Kill()
	}

	// Validate configuration with enhanced UX
	validationSpinner, validationDone := uxManager.ShowValidationSpinner()
	
	logger.Debug("Validating configuration")
	result := config.Validate(cfg, configPath)
	
	// Complete validation spinner
	if validationSpinner != nil {
		validationDone <- pkg.SpinnerDoneMsg{Success: !result.HasErrors()}
		validationSpinner.Kill()
	}
	
	// Show enhanced validation summary
	if result.HasErrors() || len(result.Warnings) > 0 {
		fmt.Fprint(os.Stderr, uxManager.FormatValidationSummary(result))
		if result.HasErrors() {
			return fmt.Errorf("configuration validation failed")
		}
	}

	// Show configuration preview if requested
	if showPreview {
		fmt.Print(uxManager.ShowConfigPreview(cfg))
		fmt.Print("\nDo you want to continue? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" && response != "yes" && response != "Yes" {
			logger.Info("Operation cancelled by user")
			return nil
		}
	}

	if dryRun {
		logger.Info("🏃 Running in dry-run mode - no changes will be made")
	}

	// Get config directory for relative path resolution
	// configDir already declared above

	// Apply repository configurations first (may be needed for package installations)
	if err := applyRepositoryConfigurations(cfg, logger, dryRun); err != nil {
		return fmt.Errorf("failed to apply repository configurations: %w", err)
	}

	// Apply file configurations
	var deployedFiles []pkg.ManagedFile
	if len(cfg.Files) > 0 {
		logger.Info("Applying file configurations")
		fileManager := pkg.NewFileManager(logger, dryRun, configDir)
		
		// Enable interactive mode on all files if global flag is set
		if interactiveMode {
			cfg.Files = enableInteractiveModeOnFiles(cfg.Files)
		}
		
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

	// Apply binary configurations
	var deployedBinaries []pkg.ManagedBinary
	if len(cfg.Binaries) > 0 {
		logger.Info("Applying binary configurations")
		binaryManager := pkg.NewBinaryManager(logger, dryRun, configDir)
		
		// Enable interactive mode on all binaries if global flag is set
		if interactiveMode {
			cfg.Binaries = enableInteractiveModeOnBinaries(cfg.Binaries)
		}
		
		// Validate binary permissions before proceeding
		if err := binaryManager.ValidateBinaryPermissions(cfg.Binaries); err != nil {
			return fmt.Errorf("binary permission validation failed: %w", err)
		}

		var err error
		deployedBinaries, err = binaryManager.DeployBinaries(cfg.Binaries)
		if err != nil {
			return fmt.Errorf("failed to deploy binaries: %w", err)
		}
	}

	// Apply package configurations
	if err := applyPackageConfigurations(cfg, deployedFiles, deployedBinaries, logger, dryRun, useOptimization, configDir); err != nil {
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

	// Apply backup policy if configured (only in non-dry-run mode)
	if !dryRun && len(deployedFiles) > 0 {
		// Load current state to get all managed files for policy enforcement
		stateManager := pkg.NewStateManager(logger)
		state, err := stateManager.LoadState()
		if err == nil && cfg.BackupPolicy.AutoCleanup {
			logger.Debug("Applying backup policy")
			fileManager := pkg.NewFileManager(logger, dryRun, configDir)
			if err := fileManager.ApplyBackupPolicy(state.Files, cfg.BackupPolicy); err != nil {
				logger.Warn("Backup policy enforcement failed", "error", err)
				// Don't fail the entire operation for backup policy issues
			}
		}
	}

	if dryRun {
		logger.Info("✓ Dry run completed - no actual changes were made")
	} else {
		logger.Info("✓ Configuration applied successfully")
	}

	return nil
}

// enableInteractiveModeOnFiles enables interactive features on all files
func enableInteractiveModeOnFiles(files map[string]config.File) map[string]config.File {
	for name, file := range files {
		file.Interactive = true
		files[name] = file
	}
	return files
}

// enableInteractiveModeOnBinaries enables interactive features on all binaries
func enableInteractiveModeOnBinaries(binaries map[string]config.Binary) map[string]config.Binary {
	for name, binary := range binaries {
		binary.Interactive = true
		binaries[name] = binary
	}
	return binaries
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
func applyPackageConfigurations(cfg *config.Config, deployedFiles []pkg.ManagedFile, deployedBinaries []pkg.ManagedBinary, logger *log.Logger, dryRun bool, useOptimization bool, configDir string) error {
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
	
	// Get binaries to remove (binaries in previous state but not in current config)
	binariesToRemove, err := stateManager.GetBinariesToRemove(cfg)
	if err != nil {
		logger.Warn("Could not determine binaries to remove", "error", err)
		// Continue with deployment even if binary removal tracking fails
		binariesToRemove = []pkg.ManagedBinary{}
	}
	
	// Remove packages, files, and binaries that are no longer in configuration (if enabled)
	if removePackages {
		if err := removePackagesNotInConfig(packagesToRemove, logger, dryRun); err != nil {
			return fmt.Errorf("failed to remove packages: %w", err)
		}
		if err := removeFilesNotInConfig(filesToRemove, configDir, logger, dryRun); err != nil {
			return fmt.Errorf("failed to remove files: %w", err)
		}
		if err := removeBinariesNotInConfig(binariesToRemove, logger, dryRun); err != nil {
			return fmt.Errorf("failed to remove binaries: %w", err)
		}
	} else {
		logger.Debug("Package, file, and binary removal disabled by --remove-packages=false flag")
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
		if err := stateManager.UpdateStateWithBinaries(cfg, deployedFiles, deployedBinaries); err != nil {
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

// removeBinariesNotInConfig removes binaries that are no longer in the configuration
func removeBinariesNotInConfig(binariesToRemove []pkg.ManagedBinary, logger *log.Logger, dryRun bool) error {
	if len(binariesToRemove) == 0 {
		return nil
	}
	logger.Info("Removing binaries no longer in configuration", "count", len(binariesToRemove))
	binaryManager := pkg.NewBinaryManager(logger, dryRun, "")
	
	return binaryManager.RemoveBinaries(binariesToRemove)
}