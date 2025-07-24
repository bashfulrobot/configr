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
	dryRun bool
)

var applyCmd = &cobra.Command{
	Use:   "apply [config-file]",
	Short: "Apply configuration changes to the system",
	Long: `Apply loads and applies the configuration to your system.

This command will:
- Add APT and Flatpak repositories
- Deploy and symlink files to their destinations
- Install APT packages
- Apply dconf settings for desktop configuration
- Create backups of existing files when requested

Note: Flatpak packages and Snap packages are not yet implemented.

By default, it looks for 'configr.yaml' in standard locations.`,
	Example: `  configr apply                       # Apply default config
  configr apply my-config.yaml        # Apply specific config
  configr apply --dry-run             # Preview changes without applying
  configr --config custom.yaml apply  # Use custom config file`,
	Args: cobra.MaximumNArgs(1),
	RunE: runApply,
}

func init() {
	rootCmd.AddCommand(applyCmd)
	
	// Command-specific flags
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without applying them")
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

	// Set config file in viper and load configuration  
	logger.Info("Loading configuration", "file", configPath)
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	
	cfg, err := config.LoadWithIncludes()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
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
	if len(cfg.Files) > 0 {
		logger.Info("Applying file configurations")
		fileManager := pkg.NewFileManager(logger, dryRun, configDir)
		
		// Validate file permissions before proceeding
		if err := fileManager.ValidateFilePermissions(cfg.Files); err != nil {
			return fmt.Errorf("permission validation failed: %w", err)
		}

		if err := fileManager.DeployFiles(cfg.Files); err != nil {
			return fmt.Errorf("failed to deploy files: %w", err)
		}
	}

	// Apply package configurations
	if err := applyPackageConfigurations(cfg, logger, dryRun); err != nil {
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
func applyPackageConfigurations(cfg *config.Config, logger *log.Logger, dryRun bool) error {
	// Handle APT packages
	if len(cfg.Packages.Apt) > 0 {
		logger.Debug("Applying APT package configurations", "count", len(cfg.Packages.Apt))
		aptManager := pkg.NewAptManager(logger, dryRun)
		if err := aptManager.InstallPackages(cfg.Packages.Apt, cfg.PackageDefaults); err != nil {
			return fmt.Errorf("APT package installation failed: %w", err)
		}
	}

	// TODO: Handle Flatpak packages (when implemented)
	if len(cfg.Packages.Flatpak) > 0 {
		logger.Warn("Flatpak management not yet implemented - skipping flatpak packages")
	}

	// TODO: Handle Snap packages (when implemented)  
	if len(cfg.Packages.Snap) > 0 {
		logger.Warn("Snap management not yet implemented - skipping snap packages")
	}

	return nil
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