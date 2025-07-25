package configr

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version is set at build time via -ldflags
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "configr",
	Short: "Desktop Linux configuration management",
	Long: `Configr is a single binary configuration management tool for Ubuntu desktop systems.

It provides package management, configuration management, and dotfile management
capabilities similar to Ansible but contained in a single binary.

Use 'configr validate' to check your configuration or 'configr --help' for more commands.`,
	Example: `  configr validate                    # Validate default config
  configr validate my-config.yaml     # Validate specific file  
  configr --config custom.yaml validate # Use custom config file`,
	Version: Version,
}

// Execute is kept for backward compatibility but deprecated
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// NewRootCmd returns the root command for use with fang
func NewRootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	cobra.OnInitialize(initConfig)
	
	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable colored output")
	
	// Bind flags to viper
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("no-color", rootCmd.PersistentFlags().Lookup("no-color"))
}

func initConfig() {
	// Use explicit config file if provided
	if configFile := viper.GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		// Search for config file in standard locations
		viper.SetConfigName("configr")
		viper.SetConfigType("yaml")
		
		// Add search paths in order of preference
		viper.AddConfigPath(".")                    // Current directory
		viper.AddConfigPath("$HOME/.config/configr") // XDG config dir
		viper.AddConfigPath("$HOME")                // Home directory  
		viper.AddConfigPath("/etc/configr")         // System config
		viper.AddConfigPath("/usr/local/etc/configr") // Local system config
	}

	// Environment variable support
	viper.SetEnvPrefix("CONFIGR")
	viper.AutomaticEnv()

	// Don't error here - let individual commands handle config loading
}