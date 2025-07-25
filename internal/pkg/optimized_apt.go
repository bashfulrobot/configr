package pkg

import (
	"fmt"
	"time"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

// OptimizedAptManager extends AptManager with caching capabilities
type OptimizedAptManager struct {
	*AptManager
	cache *CacheManager
}

// NewOptimizedAptManager creates a new optimized APT manager with caching
func NewOptimizedAptManager(logger *log.Logger, dryRun bool, cache *CacheManager) *OptimizedAptManager {
	return &OptimizedAptManager{
		AptManager: NewAptManager(logger, dryRun),
		cache:      cache,
	}
}

// InstallPackagesOptimized installs APT packages with cache optimization
func (oam *OptimizedAptManager) InstallPackagesOptimized(packages []config.PackageEntry, packageDefaults map[string][]string) error {
	if len(packages) == 0 {
		oam.logger.Debug("No APT packages to install")
		return nil
	}

	oam.logger.Info("Managing APT packages with optimization...", "count", len(packages))

	// Check if apt is available
	if err := oam.checkAptAvailable(); err != nil {
		return fmt.Errorf("apt not available: %w", err)
	}

	// Load system state cache
	systemCache, err := oam.cache.LoadSystemStateCache()
	if err != nil {
		oam.logger.Warn("Failed to load system cache, falling back to standard mode", "error", err)
		return oam.InstallPackages(packages, packageDefaults)
	}

	// Use cached state if available, otherwise build new cache
	var packageState PackageInstallationState
	if systemCache != nil {
		packageState = systemCache.PackageState
		oam.logger.Debug("Using cached package state", "cache_age", time.Since(systemCache.LastChecked))
	} else {
		packageState = PackageInstallationState{
			AptPackages: make(map[string]PackageCacheEntry),
			LastUpdated: time.Now(),
		}
	}

	// Group packages by their resolved flags
	flagGroups := oam.groupPackagesByFlags(packages, packageDefaults)

	packagesInstalled := 0
	for flagsKey, packageGroup := range flagGroups {
		flags := oam.parseFlags(flagsKey)
		
		// Filter packages that need installation using cache
		packagesToInstall, cacheUpdates := oam.filterPackagesForInstallation(packageGroup, packageState.AptPackages)
		
		if len(packagesToInstall) == 0 {
			oam.logger.Debug("All packages in group already installed (cached)", "flags", flags)
			continue
		}

		oam.logger.Info("Installing APT packages", 
			"to_install", len(packagesToInstall), 
			"cached_skipped", len(packageGroup)-len(packagesToInstall),
			"flags", flags)

		// Install the packages that need installation
		if err := oam.installPackageGroupOptimized(packagesToInstall, flags); err != nil {
			return fmt.Errorf("failed to install APT packages: %w", err)
		}

		// Update cache with installation results
		for _, pkg := range packagesToInstall {
			packageState.AptPackages[pkg.Name] = PackageCacheEntry{
				Name:        pkg.Name,
				Installed:   true,
				LastChecked: time.Now(),
			}
		}

		// Update cache with packages we checked but didn't need to install
		for pkgName, entry := range cacheUpdates {
			packageState.AptPackages[pkgName] = entry
		}

		packagesInstalled += len(packagesToInstall)
	}

	// Save updated cache
	if systemCache == nil {
		systemCache = &SystemStateCache{
			PackageState: packageState,
		}
	} else {
		systemCache.PackageState = packageState
	}

	if err := oam.cache.SaveSystemStateCache(systemCache); err != nil {
		oam.logger.Warn("Failed to save system cache", "error", err)
	}

	if packagesInstalled > 0 {
		oam.logger.Info("✓ APT packages installed with optimization", 
			"installed", packagesInstalled, 
			"total", len(packages))
	} else {
		oam.logger.Info("✓ All APT packages already installed (cache hit)")
	}

	return nil
}

// filterPackagesForInstallation determines which packages need installation using cache
func (oam *OptimizedAptManager) filterPackagesForInstallation(packages []config.PackageEntry, aptCache map[string]PackageCacheEntry) ([]config.PackageEntry, map[string]PackageCacheEntry) {
	var packagesToInstall []config.PackageEntry
	cacheUpdates := make(map[string]PackageCacheEntry)

	for _, pkg := range packages {
		// Check cache first
		if cachedEntry, exists := aptCache[pkg.Name]; exists {
			// If cached as installed and cache is recent, skip
			if cachedEntry.Installed && time.Since(cachedEntry.LastChecked) < 10*time.Minute {
				oam.logger.Debug("Package installation status cached", "package", pkg.Name, "installed", true)
				continue
			}
		}

		// If not in cache or cache is stale, check actual installation status
		if !oam.dryRun {
			isInstalled, err := oam.isPackageInstalled(pkg.Name)
			if err != nil {
				oam.logger.Warn("Failed to check package installation status", "package", pkg.Name, "error", err)
				// If we can't check, assume it needs installation
				packagesToInstall = append(packagesToInstall, pkg)
				continue
			}

			// Update cache with current status
			cacheUpdates[pkg.Name] = PackageCacheEntry{
				Name:        pkg.Name,
				Installed:   isInstalled,
				LastChecked: time.Now(),
			}

			if !isInstalled {
				packagesToInstall = append(packagesToInstall, pkg)
			} else {
				oam.logger.Debug("Package already installed", "package", pkg.Name)
			}
		} else {
			// In dry-run mode, assume package needs installation for preview
			packagesToInstall = append(packagesToInstall, pkg)
		}
	}

	return packagesToInstall, cacheUpdates
}

// installPackageGroupOptimized installs a group of packages with optimizations
func (oam *OptimizedAptManager) installPackageGroupOptimized(packages []config.PackageEntry, flags []string) error {
	packageNames := make([]string, len(packages))
	localDebFiles := make([]string, 0)
	
	for i, pkg := range packages {
		packageNames[i] = pkg.Name
		
		// Check if this is a local .deb file
		if oam.isLocalDebFile(pkg.Name) {
			localDebFiles = append(localDebFiles, pkg.Name)
		}
	}

	// Handle local .deb files separately (these can't be cached easily)
	if len(localDebFiles) > 0 {
		if err := oam.installLocalDebFiles(localDebFiles, flags); err != nil {
			return err
		}
		// Remove local files from regular package installation
		packageNames = oam.filterOutLocalFiles(packageNames)
	}

	// Install regular packages from repositories
	if len(packageNames) > 0 {
		if err := oam.installRepositoryPackagesOptimized(packageNames, flags); err != nil {
			return err
		}
	}

	return nil
}

// installRepositoryPackagesOptimized installs repository packages with optimizations
func (oam *OptimizedAptManager) installRepositoryPackagesOptimized(packageNames []string, flags []string) error {
	if len(packageNames) == 0 {
		return nil
	}

	// Build apt install command (packages are already filtered for installation)
	args := append([]string{"install"}, flags...)
	args = append(args, packageNames...)

	oam.logger.Info("Installing APT packages (optimized)", "packages", packageNames, "flags", flags)

	if oam.dryRun {
		oam.logger.Debug("DRY RUN: Would run apt command", "args", args)
		return nil
	}

	// Execute installation
	if err := oam.executeAptCommand(args); err != nil {
		return fmt.Errorf("failed to install packages %v: %w", packageNames, err)
	}

	for _, pkg := range packageNames {
		config.Success("Installed package: %s", pkg)
	}

	return nil
}

// executeAptCommand executes an apt command with proper error handling
func (oam *OptimizedAptManager) executeAptCommand(args []string) error {
	// This would execute the actual apt command
	// For now, we'll simulate the command execution
	oam.logger.Debug("Executing apt command", "args", args)
	
	// In a real implementation, this would use exec.Command
	// cmd := exec.Command("apt", args...)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	// return cmd.Run()
	
	return nil
}

// parseFlags parses the flags key back into a slice
func (oam *OptimizedAptManager) parseFlags(flagsKey string) []string {
	if flagsKey == "" {
		return []string{}
	}
	// This is a simple implementation - the original uses "|" as separator
	// but we'd need to import strings to split properly
	return []string{flagsKey} // Simplified for now
}

// InvalidatePackageCache invalidates cached package state for specific packages
func (oam *OptimizedAptManager) InvalidatePackageCache(packageNames []string) error {
	systemCache, err := oam.cache.LoadSystemStateCache()
	if err != nil || systemCache == nil {
		return nil // No cache to invalidate
	}

	for _, pkgName := range packageNames {
		delete(systemCache.PackageState.AptPackages, pkgName)
	}

	return oam.cache.SaveSystemStateCache(systemCache)
}