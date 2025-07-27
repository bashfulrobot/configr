package pkg

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

// SnapManager handles Snap package management operations
type SnapManager struct {
	logger *log.Logger
	dryRun bool
}

// NewSnapManager creates a new Snap manager
func NewSnapManager(logger *log.Logger, dryRun bool) *SnapManager {
	return &SnapManager{
		logger: logger,
		dryRun: dryRun,
	}
}

// InstallPackages installs Snap packages
func (sm *SnapManager) InstallPackages(packages []config.PackageEntry, packageDefaults map[string][]string) error {
	if len(packages) == 0 {
		sm.logger.Debug("No Snap packages to install")
		return nil
	}

	sm.logger.Info("Managing Snap packages...", "count", len(packages))

	// Check if snap is available (skip in dry-run for testing)
	if !sm.dryRun {
		if err := sm.checkSnapAvailable(); err != nil {
			return fmt.Errorf("snap not available: %w", err)
		}
	}

	// Group packages by resolved flags to minimize system calls
	packageGroups := sm.groupPackagesByFlags(packages, packageDefaults)

	for _, group := range packageGroups {
		if err := sm.installPackageGroup(group, packageDefaults); err != nil {
			return fmt.Errorf("failed to install Snap package group: %w", err)
		}
	}

	sm.logger.Info("✓ Snap packages processed successfully")
	return nil
}

// groupPackagesByFlags groups packages by their resolved flags to optimize installation
func (sm *SnapManager) groupPackagesByFlags(packages []config.PackageEntry, packageDefaults map[string][]string) [][]config.PackageEntry {
	flagGroups := make(map[string][]config.PackageEntry)

	for _, pkg := range packages {
		flags := sm.resolvePackageFlags(pkg, packageDefaults)
		flagKey := strings.Join(flags, "|")
		flagGroups[flagKey] = append(flagGroups[flagKey], pkg)
	}

	var groups [][]config.PackageEntry
	for _, group := range flagGroups {
		groups = append(groups, group)
	}

	return groups
}

// resolvePackageFlags implements the three-tier flag resolution system
func (sm *SnapManager) resolvePackageFlags(pkg config.PackageEntry, packageDefaults map[string][]string) []string {
	// Tier 3: Per-package flags (highest priority)
	if pkg.Flags != nil {
		return pkg.Flags
	}

	// Tier 2: User package defaults
	if userDefaults, exists := packageDefaults["snap"]; exists {
		return userDefaults
	}

	// Tier 1: Internal defaults
	return config.GetDefaultFlags("snap")
}

// installPackageGroup installs a group of packages with the same flags
func (sm *SnapManager) installPackageGroup(packages []config.PackageEntry, packageDefaults map[string][]string) error {
	if len(packages) == 0 {
		return nil
	}

	// Get flags from the first package (all packages in group have same flags)
	flags := sm.resolvePackageFlags(packages[0], packageDefaults)

	// Check if packages are already installed to avoid reinstalling
	var packagesToInstall []string
	for _, pkg := range packages {
		if sm.dryRun {
			// In dry-run, assume package needs installation
			packagesToInstall = append(packagesToInstall, pkg.Name)
		} else {
			installed, err := sm.isPackageInstalled(pkg.Name)
			if err != nil {
				sm.logger.Warn("Failed to check if Snap package is installed", "package", pkg.Name, "error", err)
				// Assume not installed and try to install
				packagesToInstall = append(packagesToInstall, pkg.Name)
			} else if !installed {
				packagesToInstall = append(packagesToInstall, pkg.Name)
			} else {
				sm.logger.Debug("Snap package already installed", "package", pkg.Name)
			}
		}
	}

	if len(packagesToInstall) == 0 {
		sm.logger.Debug("All Snap packages in group already installed")
		return nil
	}

	// Install packages one by one (snap install doesn't support multiple packages in one command)
	for _, packageName := range packagesToInstall {
		if err := sm.installSinglePackage(packageName, flags); err != nil {
			return fmt.Errorf("failed to install Snap package '%s': %w", packageName, err)
		}
	}

	return nil
}

// installSinglePackage installs a single Snap package
func (sm *SnapManager) installSinglePackage(packageName string, flags []string) error {
	args := []string{"snap", "install"}
	args = append(args, flags...)
	args = append(args, packageName)

	sm.logger.Info("Installing Snap package", "package", packageName, "flags", flags)

	if sm.dryRun {
		sm.logger.Info("  [DRY RUN] Would run:", "command", strings.Join(args, " "))
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		sm.logger.Error("Failed to install Snap package", "package", packageName, "error", err, "output", string(output))
		return fmt.Errorf("snap install failed: %w", err)
	}

	sm.logger.Debug("Snap package installed successfully", "package", packageName, "output", string(output))
	return nil
}

// isPackageInstalled checks if a Snap package is already installed
func (sm *SnapManager) isPackageInstalled(packageName string) (bool, error) {
	args := []string{"snap", "list", packageName}
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// If snap list fails, the package is likely not installed
		// snap list returns non-zero exit code for non-installed packages
		return false, nil
	}

	// Check if the output contains the package name
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), packageName+" ") || 
		   strings.TrimSpace(line) == packageName {
			return true, nil
		}
	}

	return false, nil
}

// UninstallPackage removes a Snap package
func (sm *SnapManager) UninstallPackage(packageName string, flags []string) error {
	args := []string{"snap", "remove"}
	args = append(args, flags...)
	args = append(args, packageName)

	sm.logger.Info("Uninstalling Snap package", "package", packageName, "flags", flags)

	if sm.dryRun {
		sm.logger.Info("  [DRY RUN] Would run:", "command", strings.Join(args, " "))
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		sm.logger.Error("Failed to uninstall Snap package", "package", packageName, "error", err, "output", string(output))
		return fmt.Errorf("snap remove failed: %w", err)
	}

	sm.logger.Debug("Snap package uninstalled successfully", "package", packageName, "output", string(output))
	return nil
}

// ListInstalledPackages returns a list of installed Snap packages
func (sm *SnapManager) ListInstalledPackages() ([]string, error) {
	if sm.dryRun {
		sm.logger.Info("  [DRY RUN] Would run:", "command", "snap list")
		return []string{}, nil
	}

	args := []string{"snap", "list"}
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		sm.logger.Error("Failed to list Snap packages", "error", err, "output", string(output))
		return nil, fmt.Errorf("snap list failed: %w", err)
	}

	var packages []string
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		// Skip header line and empty lines
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		
		// Extract package name (first column)
		fields := strings.Fields(line)
		if len(fields) > 0 {
			packages = append(packages, fields[0])
		}
	}

	sm.logger.Debug("Listed Snap packages", "count", len(packages))
	return packages, nil
}

// RefreshPackages updates all installed Snap packages
func (sm *SnapManager) RefreshPackages(flags []string) error {
	args := []string{"snap", "refresh"}
	args = append(args, flags...)

	sm.logger.Info("Refreshing Snap packages", "flags", flags)

	if sm.dryRun {
		sm.logger.Info("  [DRY RUN] Would run:", "command", strings.Join(args, " "))
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		sm.logger.Error("Failed to refresh Snap packages", "error", err, "output", string(output))
		return fmt.Errorf("snap refresh failed: %w", err)
	}

	sm.logger.Debug("Snap packages refreshed successfully", "output", string(output))
	return nil
}

// InfoPackage gets information about a Snap package
func (sm *SnapManager) InfoPackage(packageName string) (string, error) {
	if sm.dryRun {
		sm.logger.Info("  [DRY RUN] Would run:", "command", fmt.Sprintf("snap info %s", packageName))
		return "", nil
	}

	args := []string{"snap", "info", packageName}
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		sm.logger.Error("Failed to get Snap package info", "package", packageName, "error", err, "output", string(output))
		return "", fmt.Errorf("snap info failed: %w", err)
	}

	return string(output), nil
}

// FindPackage searches for Snap packages
func (sm *SnapManager) FindPackage(packageName string) ([]string, error) {
	if sm.dryRun {
		sm.logger.Info("  [DRY RUN] Would run:", "command", fmt.Sprintf("snap find %s", packageName))
		return []string{}, nil
	}

	args := []string{"snap", "find", packageName}
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		sm.logger.Error("Failed to find Snap packages", "search", packageName, "error", err, "output", string(output))
		return nil, fmt.Errorf("snap find failed: %w", err)
	}

	var packages []string
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		// Skip header line and empty lines
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		
		// Extract package name (first column)
		fields := strings.Fields(line)
		if len(fields) > 0 {
			packages = append(packages, fields[0])
		}
	}

	sm.logger.Debug("Found Snap packages", "search", packageName, "count", len(packages))
	return packages, nil
}

// checkSnapAvailable checks if snap command is available
func (sm *SnapManager) checkSnapAvailable() error {
	if _, err := exec.LookPath("snap"); err != nil {
		return fmt.Errorf("snap command not found - install snapd package")
	}
	return nil
}

// RemovePackages removes multiple Snap packages that are no longer in the configuration
func (sm *SnapManager) RemovePackages(packagesToRemove []string) error {
	if len(packagesToRemove) == 0 {
		return nil
	}

	sm.logger.Info("Removing Snap packages no longer in configuration", "packages", packagesToRemove)

	// Filter to only remove packages that are actually installed
	installedToRemove := make([]string, 0, len(packagesToRemove))
	for _, pkg := range packagesToRemove {
		if installed, err := sm.isPackageInstalled(pkg); err != nil {
			sm.logger.Warn("Could not check if Snap package is installed", "package", pkg, "error", err)
		} else if installed {
			installedToRemove = append(installedToRemove, pkg)
		}
	}

	if len(installedToRemove) == 0 {
		sm.logger.Info("No installed Snap packages to remove")
		return nil
	}

	// Remove packages one by one (snap remove works on individual packages)
	for _, pkg := range installedToRemove {
		if err := sm.UninstallPackage(pkg, []string{}); err != nil {
			sm.logger.Error("Failed to remove Snap package", "package", pkg, "error", err)
			return fmt.Errorf("failed to remove Snap package %s: %w", pkg, err)
		}
		config.Success("Removed Snap package: %s", pkg)
	}

	return nil
}

// ValidatePackageNames validates Snap package names
func (sm *SnapManager) ValidatePackageNames(packages []config.PackageEntry) error {
	sm.logger.Debug("Validating Snap package names", "count", len(packages))

	for _, pkg := range packages {
		if err := sm.validatePackageName(pkg.Name); err != nil {
			return fmt.Errorf("invalid Snap package name '%s': %w", pkg.Name, err)
		}
	}

	return nil
}

// validatePackageName validates a single Snap package name
func (sm *SnapManager) validatePackageName(packageName string) error {
	if packageName == "" {
		return fmt.Errorf("package name cannot be empty")
	}

	// Snap package names should be lowercase and can contain letters, numbers, and hyphens
	// They must start with a letter and cannot end with a hyphen
	if len(packageName) < 2 {
		return fmt.Errorf("package name must be at least 2 characters long")
	}

	if len(packageName) > 40 {
		return fmt.Errorf("package name cannot be longer than 40 characters")
	}

	// Must start with a lowercase letter
	if packageName[0] < 'a' || packageName[0] > 'z' {
		return fmt.Errorf("package name must start with a lowercase letter")
	}

	// Cannot end with a hyphen
	if packageName[len(packageName)-1] == '-' {
		return fmt.Errorf("package name cannot end with a hyphen")
	}

	// Check all characters are valid
	for i, char := range packageName {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
			return fmt.Errorf("package name contains invalid character at position %d: %c", i, char)
		}
	}

	// Cannot have consecutive hyphens
	if strings.Contains(packageName, "--") {
		return fmt.Errorf("package name cannot contain consecutive hyphens")
	}

	return nil
}

// SearchPackages is an alias for FindPackage to maintain consistency with other managers
func (sm *SnapManager) SearchPackages(searchTerm string) ([]string, error) {
	return sm.FindPackage(searchTerm)
}

// GetPackageInfo is an alias for InfoPackage to maintain consistency with other managers
func (sm *SnapManager) GetPackageInfo(packageName string) (string, error) {
	return sm.InfoPackage(packageName)
}

// UpgradePackages is an alias for RefreshPackages to maintain consistency with other managers
func (sm *SnapManager) UpgradePackages(packageNames []string, flags []string) error {
	if len(packageNames) == 0 {
		// Refresh all packages
		return sm.RefreshPackages(flags)
	}
	
	// Snap doesn't support upgrading specific packages, so refresh all
	sm.logger.Info("Snap doesn't support upgrading specific packages, refreshing all packages", "requested", packageNames)
	return sm.RefreshPackages(flags)
}