package configr

import (
	"fmt"
	"os"

	"github.com/bashfulrobot/configr/internal/pkg"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage configuration and system state cache",
	Long: `Cache management commands for configr optimization system.

The cache stores parsed configurations and system state to speed up repeated runs.
Cache data is stored in ~/.cache/configr/ by default.`,
	Example: `  configr cache stats    # Show cache statistics
  configr cache clear    # Clear all cached data
  configr cache info     # Show cache information`,
}

var cacheStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show cache usage statistics",
	Long: `Display statistics about cache usage including file count, total size,
and last modification time.`,
	RunE: runCacheStats,
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all cached data",
	Long: `Remove all cached configuration and system state data.
This will force the next run to rebuild all caches.`,
	RunE: runCacheClear,
}

var cacheInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show cache system information",
	Long: `Display information about the cache system including cache directory
location and current optimization settings.`,
	RunE: runCacheInfo,
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	
	// Add subcommands
	cacheCmd.AddCommand(cacheStatsCmd)
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheInfoCmd)
}

func runCacheStats(cmd *cobra.Command, args []string) error {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: false,
		Prefix:          "configr",
	})

	cacheManager := pkg.NewCacheManager(logger)
	stats, err := cacheManager.GetCacheStats()
	if err != nil {
		return fmt.Errorf("failed to get cache stats: %w", err)
	}

	// Display cache statistics
	fmt.Printf("Cache Statistics\n")
	fmt.Printf("================\n\n")
	fmt.Printf("Cache Directory: %s\n", stats.CacheDir)
	fmt.Printf("Total Files:     %d\n", stats.TotalFiles)
	fmt.Printf("Total Size:      %s\n", formatBytes(stats.TotalSize))
	
	if !stats.LastModified.IsZero() {
		fmt.Printf("Last Modified:   %s\n", stats.LastModified.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("Last Modified:   Never\n")
	}

	if stats.TotalFiles == 0 {
		fmt.Printf("\n💡 Cache is empty. Run 'configr apply --optimize' to build cache.\n")
	} else {
		fmt.Printf("\n✓ Cache contains %d files (%s total)\n", stats.TotalFiles, formatBytes(stats.TotalSize))
	}

	return nil
}

func runCacheClear(cmd *cobra.Command, args []string) error {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: false,
		Prefix:          "configr",
	})

	cacheManager := pkg.NewCacheManager(logger)
	
	// Get stats before clearing
	stats, err := cacheManager.GetCacheStats()
	if err != nil {
		logger.Warn("Failed to get cache stats before clearing", "error", err)
	}

	if err := cacheManager.ClearCache(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	if stats != nil && stats.TotalFiles > 0 {
		fmt.Printf("✓ Cleared cache: %d files (%s) removed\n", stats.TotalFiles, formatBytes(stats.TotalSize))
	} else {
		fmt.Printf("✓ Cache cleared (was already empty)\n")
	}

	return nil
}

func runCacheInfo(cmd *cobra.Command, args []string) error {
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportCaller:    false,
		ReportTimestamp: false,
		Prefix:          "configr",
	})

	cacheManager := pkg.NewCacheManager(logger)
	stats, err := cacheManager.GetCacheStats()
	if err != nil {
		return fmt.Errorf("failed to get cache info: %w", err)
	}

	fmt.Printf("Cache System Information\n")
	fmt.Printf("========================\n\n")
	fmt.Printf("Cache Directory:     %s\n", stats.CacheDir)
	fmt.Printf("Optimization:        Enabled by default\n")
	fmt.Printf("Cache TTL:           1 hour (system state)\n")
	fmt.Printf("Config Cache:        Persistent until files change\n")
	fmt.Printf("Package Cache:       10 minutes (installation status)\n")
	
	fmt.Printf("\nCache Types:\n")
	fmt.Printf("- Configuration Cache: Parsed YAML configurations\n")
	fmt.Printf("- System State Cache:  Package installation status\n")
	fmt.Printf("- File State Cache:    File modification tracking\n")
	
	fmt.Printf("\nCommands:\n")
	fmt.Printf("- Enable:  configr apply --optimize=true (default)\n")
	fmt.Printf("- Disable: configr apply --optimize=false\n")
	fmt.Printf("- Clear:   configr cache clear\n")
	fmt.Printf("- Stats:   configr cache stats\n")

	return nil
}

// formatBytes formats byte count as human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}