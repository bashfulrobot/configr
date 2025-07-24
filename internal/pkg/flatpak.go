package pkg

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

// FlatpakManager handles Flatpak package management operations
type FlatpakManager struct {
	logger *log.Logger
	dryRun bool
}

// NewFlatpakManager creates a new Flatpak manager
func NewFlatpakManager(logger *log.Logger, dryRun bool) *FlatpakManager {
	return &FlatpakManager{
		logger: logger,
		dryRun: dryRun,
	}
}

// InstallPackages installs Flatpak applications
func (fm *FlatpakManager) InstallPackages(packages []config.PackageEntry, packageDefaults map[string][]string) error {
	if len(packages) == 0 {
		fm.logger.Debug("No Flatpak packages to install")
		return nil
	}

	fm.logger.Info("Managing Flatpak packages...", "count", len(packages))

	// Check if flatpak is available (skip in dry-run for testing)
	if !fm.dryRun {
		if err := fm.checkFlatpakAvailable(); err != nil {
			return fmt.Errorf("flatpak not available: %w", err)
		}
	}

	// Group packages by resolved flags to minimize system calls
	packageGroups := fm.groupPackagesByFlags(packages, packageDefaults)

	for _, group := range packageGroups {
		if err := fm.installPackageGroup(group, packageDefaults); err != nil {
			return fmt.Errorf("failed to install Flatpak package group: %w", err)
		}
	}

	fm.logger.Info("✓ Flatpak packages processed successfully")
	return nil
}

// groupPackagesByFlags groups packages by their resolved flags to optimize installation
func (fm *FlatpakManager) groupPackagesByFlags(packages []config.PackageEntry, packageDefaults map[string][]string) [][]config.PackageEntry {
	flagGroups := make(map[string][]config.PackageEntry)

	for _, pkg := range packages {
		flags := fm.resolvePackageFlags(pkg, packageDefaults)
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
func (fm *FlatpakManager) resolvePackageFlags(pkg config.PackageEntry, packageDefaults map[string][]string) []string {
	// Tier 3: Per-package flags (highest priority)
	if pkg.Flags != nil {
		return pkg.Flags
	}

	// Tier 2: User package defaults
	if userDefaults, exists := packageDefaults["flatpak"]; exists {
		return userDefaults
	}

	// Tier 1: Internal defaults
	return config.GetDefaultFlags("flatpak")
}

// installPackageGroup installs a group of packages with the same flags
func (fm *FlatpakManager) installPackageGroup(packages []config.PackageEntry, packageDefaults map[string][]string) error {
	if len(packages) == 0 {
		return nil
	}

	// Get flags from the first package (all packages in group have same flags)
	flags := fm.resolvePackageFlags(packages[0], packageDefaults)

	// Check if packages are already installed to avoid reinstalling
	var packagesToInstall []string
	for _, pkg := range packages {
		if fm.dryRun {
			// In dry-run, assume package needs installation
			packagesToInstall = append(packagesToInstall, pkg.Name)
		} else {
			installed, err := fm.isPackageInstalled(pkg.Name)
			if err != nil {
				fm.logger.Warn("Failed to check if Flatpak package is installed", "package", pkg.Name, "error", err)
				// Assume not installed and try to install
				packagesToInstall = append(packagesToInstall, pkg.Name)
			} else if !installed {
				packagesToInstall = append(packagesToInstall, pkg.Name)
			} else {
				fm.logger.Debug("Flatpak package already installed", "package", pkg.Name)
			}
		}
	}

	if len(packagesToInstall) == 0 {
		fm.logger.Debug("All Flatpak packages in group already installed")
		return nil
	}

	// Build the flatpak install command
	args := []string{"flatpak", "install"}
	args = append(args, flags...)
	args = append(args, packagesToInstall...)

	fm.logger.Info("Installing Flatpak packages", "packages", packagesToInstall, "flags", flags)

	if fm.dryRun {
		fm.logger.Info("  [DRY RUN] Would run:", "command", strings.Join(args, " "))
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		fm.logger.Error("Failed to install Flatpak packages", "packages", packagesToInstall, "error", err, "output", string(output))
		return fmt.Errorf("flatpak install failed: %w", err)
	}

	fm.logger.Debug("Flatpak packages installed successfully", "packages", packagesToInstall, "output", string(output))
	return nil
}

// isPackageInstalled checks if a Flatpak package is already installed
func (fm *FlatpakManager) isPackageInstalled(packageName string) (bool, error) {
	// Check both system and user installations
	systemInstalled, err := fm.isPackageInstalledInScope(packageName, "--system")
	if err != nil {
		return false, err
	}

	userInstalled, err := fm.isPackageInstalledInScope(packageName, "--user")
	if err != nil {
		return false, err
	}

	return systemInstalled || userInstalled, nil
}

// isPackageInstalledInScope checks if a package is installed in a specific scope (system or user)
func (fm *FlatpakManager) isPackageInstalledInScope(packageName, scope string) (bool, error) {
	args := []string{"flatpak", "list", scope, "--app", "--columns=application"}
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// If flatpak list fails, it might be because no packages are installed
		// or the scope doesn't exist, so we assume the package is not installed
		return false, nil
	}

	installedPackages := strings.Split(string(output), "\n")
	for _, installed := range installedPackages {
		if strings.TrimSpace(installed) == packageName {
			return true, nil
		}
	}

	return false, nil
}

// UninstallPackage removes a Flatpak application
func (fm *FlatpakManager) UninstallPackage(packageName string, flags []string) error {
	args := []string{"flatpak", "uninstall"}
	args = append(args, flags...)
	args = append(args, packageName)

	fm.logger.Info("Uninstalling Flatpak package", "package", packageName, "flags", flags)

	if fm.dryRun {
		fm.logger.Info("  [DRY RUN] Would run:", "command", strings.Join(args, " "))
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		fm.logger.Error("Failed to uninstall Flatpak package", "package", packageName, "error", err, "output", string(output))
		return fmt.Errorf("flatpak uninstall failed: %w", err)
	}

	fm.logger.Debug("Flatpak package uninstalled successfully", "package", packageName, "output", string(output))
	return nil
}

// ListInstalledPackages returns a list of installed Flatpak applications
func (fm *FlatpakManager) ListInstalledPackages() ([]string, error) {
	if fm.dryRun {
		fm.logger.Info("  [DRY RUN] Would run:", "command", "flatpak list --app --columns=application")
		return []string{}, nil
	}

	args := []string{"flatpak", "list", "--app", "--columns=application"}
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		fm.logger.Error("Failed to list Flatpak packages", "error", err, "output", string(output))
		return nil, fmt.Errorf("flatpak list failed: %w", err)
	}

	var packages []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line != "Application ID" { // Skip header
			packages = append(packages, line)
		}
	}

	fm.logger.Debug("Listed Flatpak packages", "count", len(packages))
	return packages, nil
}

// UpdatePackages updates all installed Flatpak applications
func (fm *FlatpakManager) UpdatePackages(flags []string) error {
	args := []string{"flatpak", "update"}
	args = append(args, flags...)

	fm.logger.Info("Updating Flatpak packages", "flags", flags)

	if fm.dryRun {
		fm.logger.Info("  [DRY RUN] Would run:", "command", strings.Join(args, " "))
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		fm.logger.Error("Failed to update Flatpak packages", "error", err, "output", string(output))
		return fmt.Errorf("flatpak update failed: %w", err)
	}

	fm.logger.Debug("Flatpak packages updated successfully", "output", string(output))
	return nil
}

// checkFlatpakAvailable checks if flatpak command is available
func (fm *FlatpakManager) checkFlatpakAvailable() error {
	if _, err := exec.LookPath("flatpak"); err != nil {
		return fmt.Errorf("flatpak command not found - install flatpak package")
	}
	return nil
}

// ValidatePackageNames validates Flatpak package names (application IDs)
func (fm *FlatpakManager) ValidatePackageNames(packages []config.PackageEntry) error {
	fm.logger.Debug("Validating Flatpak package names", "count", len(packages))

	for _, pkg := range packages {
		if err := fm.validatePackageName(pkg.Name); err != nil {
			return fmt.Errorf("invalid Flatpak package name '%s': %w", pkg.Name, err)
		}
	}

	return nil
}

// validatePackageName validates a single Flatpak application ID
func (fm *FlatpakManager) validatePackageName(packageName string) error {
	// Flatpak application IDs should follow reverse domain notation
	// e.g., org.mozilla.Firefox, com.spotify.Client
	if packageName == "" {
		return fmt.Errorf("package name cannot be empty")
	}

	// Basic validation for reverse domain notation
	parts := strings.Split(packageName, ".")
	if len(parts) < 2 {
		return fmt.Errorf("Flatpak application ID should use reverse domain notation (e.g., org.mozilla.Firefox)")
	}

	// Check for invalid characters
	for _, part := range parts {
		if part == "" {
			return fmt.Errorf("application ID parts cannot be empty")
		}
		for _, char := range part {
			if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || 
				 (char >= '0' && char <= '9') || char == '-' || char == '_') {
				return fmt.Errorf("application ID contains invalid character: %c", char)
			}
		}
	}

	return nil
}