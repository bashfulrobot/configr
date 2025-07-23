package configr

import (
	"fmt"
	"os"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var validateCmd = &cobra.Command{
	Use:   "validate [config-file]",
	Short: "Validate configuration file",
	Long: `Validate the configuration file for syntax errors, missing files, 
and other issues without making any changes to the system.`,
	Example: `  configr validate                    # Validate default config
  configr validate my-config.yaml     # Validate specific file
  configr validate --verbose          # Show detailed validation info`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Set verbose mode if requested
		verbose, _ := cmd.Flags().GetBool("verbose")
		config.SetVerbose(verbose)

		// Use provided config file or search for default
		if len(args) > 0 {
			viper.SetConfigFile(args[0])
		}

		// Try to read config
		if err := viper.ReadInConfig(); err != nil {
			config.Error("Failed to read config file: %v", err)
			os.Exit(1)
		}

		config.Info("Validating configuration: %s", viper.ConfigFileUsed())

		// Load and validate config
		cfg, err := config.LoadWithIncludes()
		if err != nil {
			config.Error("Failed to load config: %v", err)
			os.Exit(1)
		}

		// Validate configuration
		result := config.Validate(cfg, viper.ConfigFileUsed())

		// Show results
		if result.HasErrors() {
			fmt.Print(config.FormatValidationResultSimple(result))
			fmt.Print(config.FormatQuickFixSimple(result))
			os.Exit(1)
		}

		if len(result.Warnings) > 0 {
			fmt.Print(config.FormatValidationResultSimple(result))
		}

		config.Success("Configuration is valid")
		
		if verbose {
			config.Debug("Found %d package definitions", 
				len(cfg.Packages.Apt)+len(cfg.Packages.Flatpak)+len(cfg.Packages.Snap))
			config.Debug("Found %d file definitions", len(cfg.Files))
			config.Debug("Found %d dconf settings", len(cfg.DConf.Settings))
		}
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().BoolP("verbose", "v", false, "Show detailed validation information")
}