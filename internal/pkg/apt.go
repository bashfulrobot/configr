package pkg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

// AptManager handles APT package management operations
type AptManager struct {
	logger *log.Logger
	dryRun bool
}

// NewAptManager creates a new APT package manager
func NewAptManager(logger *log.Logger, dryRun bool) *AptManager {
	return &AptManager{
		logger: logger,
		dryRun: dryRun,
	}
}

// InstallPackages installs the specified APT packages
func (am *AptManager) InstallPackages(packages []config.PackageEntry, packageDefaults map[string][]string) error {
	if len(packages) == 0 {
		am.logger.Debug("No APT packages to install")
		return nil
	}

	am.logger.Info("Managing APT packages...")

	// Check if apt is available
	if err := am.checkAptAvailable(); err != nil {
		return fmt.Errorf("apt not available: %w", err)
	}

	// Group packages by their resolved flags to minimize apt calls
	flagGroups := am.groupPackagesByFlags(packages, packageDefaults)

	for flagsKey, packageGroup := range flagGroups {
		if err := am.installPackageGroup(packageGroup, strings.Split(flagsKey, "|")); err != nil {
			return err
		}
	}

	return nil
}

// checkAptAvailable verifies that apt is available on the system
func (am *AptManager) checkAptAvailable() error {
	_, err := exec.LookPath("apt")
	if err != nil {
		return fmt.Errorf("apt command not found - is this a Debian/Ubuntu system?")
	}
	return nil
}

// groupPackagesByFlags groups packages with the same resolved flags together
func (am *AptManager) groupPackagesByFlags(packages []config.PackageEntry, packageDefaults map[string][]string) map[string][]config.PackageEntry {
	flagGroups := make(map[string][]config.PackageEntry)

	for _, pkg := range packages {
		flags := am.resolvePackageFlags(pkg, packageDefaults)
		flagsKey := strings.Join(flags, "|") // Use "|" as separator since it's not valid in flags
		flagGroups[flagsKey] = append(flagGroups[flagsKey], pkg)
	}

	return flagGroups
}

// resolvePackageFlags implements the three-tier flag resolution system
func (am *AptManager) resolvePackageFlags(pkg config.PackageEntry, packageDefaults map[string][]string) []string {
	// Tier 3: Per-package flags (highest priority)
	// Note: pkg.Flags != nil means the user explicitly set flags (even if empty)
	if pkg.Flags != nil {
		am.logger.Debug("Using per-package flags", "package", pkg.Name, "flags", pkg.Flags)
		return pkg.Flags
	}

	// Tier 2: User package defaults
	if userDefaults, exists := packageDefaults["apt"]; exists {
		am.logger.Debug("Using user default flags", "package", pkg.Name, "flags", userDefaults)
		return userDefaults
	}

	// Tier 1: Internal defaults
	internalDefaults := config.GetDefaultFlags("apt")
	am.logger.Debug("Using internal default flags", "package", pkg.Name, "flags", internalDefaults)
	return internalDefaults
}

// installPackageGroup installs a group of packages with the same flags
func (am *AptManager) installPackageGroup(packages []config.PackageEntry, flags []string) error {
	packageNames := make([]string, len(packages))
	localDebFiles := make([]string, 0)
	
	for i, pkg := range packages {
		packageNames[i] = pkg.Name
		
		// Check if this is a local .deb file
		if am.isLocalDebFile(pkg.Name) {
			localDebFiles = append(localDebFiles, pkg.Name)
		}
	}

	// Handle local .deb files separately
	if len(localDebFiles) > 0 {
		if err := am.installLocalDebFiles(localDebFiles, flags); err != nil {
			return err
		}
		// Remove local files from regular package installation
		packageNames = am.filterOutLocalFiles(packageNames)
	}

	// Install regular packages from repositories
	if len(packageNames) > 0 {
		if err := am.installRepositoryPackages(packageNames, flags); err != nil {
			return err
		}
	}

	return nil
}

// isLocalDebFile checks if a package name refers to a local .deb file
func (am *AptManager) isLocalDebFile(packageName string) bool {
	return strings.HasSuffix(packageName, ".deb") && (strings.HasPrefix(packageName, "/") || strings.Contains(packageName, "/"))
}

// filterOutLocalFiles removes local .deb files from the package list
func (am *AptManager) filterOutLocalFiles(packageNames []string) []string {
	filtered := make([]string, 0, len(packageNames))
	for _, name := range packageNames {
		if !am.isLocalDebFile(name) {
			filtered = append(filtered, name)
		}
	}
	return filtered
}

// installLocalDebFiles installs local .deb files
func (am *AptManager) installLocalDebFiles(debFiles []string, flags []string) error {
	for _, debFile := range debFiles {
		if err := am.installSingleDebFile(debFile, flags); err != nil {
			return err
		}
	}
	return nil
}

// installSingleDebFile installs a single local .deb file
func (am *AptManager) installSingleDebFile(debFile string, flags []string) error {
	// Resolve relative paths
	if !filepath.IsAbs(debFile) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		debFile = filepath.Join(wd, debFile)
	}

	// Verify file exists
	if _, err := os.Stat(debFile); os.IsNotExist(err) {
		return fmt.Errorf("local .deb file not found: %s", debFile)
	}

	// Build apt install command for local file
	args := append([]string{"install"}, flags...)
	args = append(args, debFile)

	am.logger.Info("Installing local .deb file", "file", debFile, "flags", flags)

	if am.dryRun {
		am.logger.Debug("DRY RUN: Would run apt command", "args", args)
		return nil
	}

	cmd := exec.Command("apt", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install local .deb file %s: %w", debFile, err)
	}

	config.Success("Installed local .deb file: %s", filepath.Base(debFile))
	return nil
}

// installRepositoryPackages installs packages from repositories
func (am *AptManager) installRepositoryPackages(packageNames []string, flags []string) error {
	// Check which packages are already installed
	installedPackages, err := am.getInstalledPackages(packageNames)
	if err != nil {
		am.logger.Warn("Failed to check installed packages, proceeding anyway", "error", err)
		installedPackages = make(map[string]bool) // Empty map means check all packages
	}

	// Filter out already installed packages
	packagesToInstall := make([]string, 0, len(packageNames))
	for _, pkg := range packageNames {
		if !installedPackages[pkg] {
			packagesToInstall = append(packagesToInstall, pkg)
		} else {
			am.logger.Debug("Package already installed", "package", pkg)
		}
	}

	if len(packagesToInstall) == 0 {
		am.logger.Info("All APT packages already installed")
		return nil
	}

	// Build apt install command
	args := append([]string{"install"}, flags...)
	args = append(args, packagesToInstall...)

	am.logger.Info("Installing APT packages", "packages", packagesToInstall, "flags", flags)

	if am.dryRun {
		am.logger.Debug("DRY RUN: Would run apt command", "args", args)
		return nil
	}

	cmd := exec.Command("apt", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install packages %v: %w", packagesToInstall, err)
	}

	for _, pkg := range packagesToInstall {
		config.Success("Installed package: %s", pkg)
	}

	return nil
}

// getInstalledPackages checks which packages from the list are already installed
func (am *AptManager) getInstalledPackages(packageNames []string) (map[string]bool, error) {
	installed := make(map[string]bool)

	for _, pkg := range packageNames {
		isInstalled, err := am.isPackageInstalled(pkg)
		if err != nil {
			return nil, fmt.Errorf("failed to check if package %s is installed: %w", pkg, err)
		}
		installed[pkg] = isInstalled
	}

	return installed, nil
}

// isPackageInstalled checks if a single package is installed
func (am *AptManager) isPackageInstalled(packageName string) (bool, error) {
	cmd := exec.Command("dpkg", "-s", packageName)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		// dpkg returns non-zero if package is not installed
		return false, nil
	}

	// Check if the package status indicates it's installed
	outputStr := string(output)
	return strings.Contains(outputStr, "Status: install ok installed"), nil
}

// RemovePackages removes packages that are no longer in the configuration
func (am *AptManager) RemovePackages(packagesToRemove []string) error {
	if len(packagesToRemove) == 0 {
		return nil
	}

	am.logger.Info("Removing APT packages no longer in configuration", "packages", packagesToRemove)

	// Filter to only remove packages that are actually installed
	installedToRemove := make([]string, 0, len(packagesToRemove))
	for _, pkg := range packagesToRemove {
		if installed, err := am.isPackageInstalled(pkg); err != nil {
			am.logger.Warn("Could not check if package is installed", "package", pkg, "error", err)
		} else if installed {
			installedToRemove = append(installedToRemove, pkg)
		}
	}

	if len(installedToRemove) == 0 {
		am.logger.Info("No installed APT packages to remove")
		return nil
	}

	// Build apt remove command
	args := append([]string{"remove", "-y"}, installedToRemove...)

	if am.dryRun {
		am.logger.Debug("DRY RUN: Would run apt command", "args", args)
		return nil
	}

	cmd := exec.Command("apt", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove packages %v: %w", installedToRemove, err)
	}

	for _, pkg := range installedToRemove {
		config.Success("Removed package: %s", pkg)
	}

	return nil
}

// SearchPackages searches for APT packages using apt search
func (am *AptManager) SearchPackages(searchTerm string) ([]string, error) {
	if searchTerm == "" {
		return nil, fmt.Errorf("search term cannot be empty")
	}

	if am.dryRun {
		am.logger.Info("DRY RUN: Would search for packages", "term", searchTerm)
		return []string{}, nil
	}

	// Check if apt is available
	if err := am.checkAptAvailable(); err != nil {
		return nil, fmt.Errorf("apt not available: %w", err)
	}

	args := []string{"search", searchTerm}
	cmd := exec.Command("apt", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil, fmt.Errorf("apt search failed: %w", err)
	}

	var packages []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "WARNING:") || strings.HasPrefix(line, "NOTE:") {
			continue
		}
		
		// APT search output format: "packagename/repository - description"
		if strings.Contains(line, "/") && strings.Contains(line, " - ") {
			parts := strings.Split(line, "/")
			if len(parts) > 0 {
				packageName := strings.TrimSpace(parts[0])
				if packageName != "" {
					packages = append(packages, packageName)
				}
			}
		}
	}

	am.logger.Debug("Found APT packages", "search", searchTerm, "count", len(packages))
	return packages, nil
}

// GetPackageInfo returns detailed information about an APT package
func (am *AptManager) GetPackageInfo(packageName string) (string, error) {
	if packageName == "" {
		return "", fmt.Errorf("package name cannot be empty")
	}

	if am.dryRun {
		am.logger.Info("DRY RUN: Would get package info", "package", packageName)
		return "", nil
	}

	// Check if apt is available
	if err := am.checkAptAvailable(); err != nil {
		return "", fmt.Errorf("apt not available: %w", err)
	}

	args := []string{"show", packageName}
	cmd := exec.Command("apt", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("apt show failed for package %s: %w", packageName, err)
	}

	return string(output), nil
}

// ListUpgradablePackages returns a list of packages that can be upgraded
func (am *AptManager) ListUpgradablePackages() ([]string, error) {
	if am.dryRun {
		am.logger.Info("DRY RUN: Would list upgradable packages")
		return []string{}, nil
	}

	// Check if apt is available
	if err := am.checkAptAvailable(); err != nil {
		return nil, fmt.Errorf("apt not available: %w", err)
	}

	args := []string{"list", "--upgradable"}
	cmd := exec.Command("apt", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil, fmt.Errorf("apt list --upgradable failed: %w", err)
	}

	var packages []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "WARNING:") || strings.HasPrefix(line, "Listing...") {
			continue
		}
		
		// APT list output format: "packagename/repository version [upgradable from: oldversion]"
		if strings.Contains(line, "/") && strings.Contains(line, "[upgradable from:") {
			parts := strings.Split(line, "/")
			if len(parts) > 0 {
				packageName := strings.TrimSpace(parts[0])
				if packageName != "" {
					packages = append(packages, packageName)
				}
			}
		}
	}

	am.logger.Debug("Found upgradable APT packages", "count", len(packages))
	return packages, nil
}

// UpgradePackages upgrades all upgradable packages or specific packages
func (am *AptManager) UpgradePackages(packageNames []string, flags []string) error {
	// Check if apt is available
	if err := am.checkAptAvailable(); err != nil {
		return fmt.Errorf("apt not available: %w", err)
	}

	var args []string
	if len(packageNames) == 0 {
		// Upgrade all packages
		args = append([]string{"upgrade"}, flags...)
		am.logger.Info("Upgrading all APT packages", "flags", flags)
	} else {
		// Upgrade specific packages
		args = append([]string{"install", "--only-upgrade"}, flags...)
		args = append(args, packageNames...)
		am.logger.Info("Upgrading specific APT packages", "packages", packageNames, "flags", flags)
	}

	if am.dryRun {
		am.logger.Debug("DRY RUN: Would run apt command", "args", args)
		return nil
	}

	cmd := exec.Command("apt", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to upgrade packages: %w", err)
	}

	if len(packageNames) == 0 {
		config.Success("All APT packages upgraded successfully")
	} else {
		for _, pkg := range packageNames {
			config.Success("Upgraded package: %s", pkg)
		}
	}

	return nil
}

// GetPackageDependencies returns the dependencies of an APT package
func (am *AptManager) GetPackageDependencies(packageName string) ([]string, error) {
	if packageName == "" {
		return nil, fmt.Errorf("package name cannot be empty")
	}

	if am.dryRun {
		am.logger.Info("DRY RUN: Would get package dependencies", "package", packageName)
		return []string{}, nil
	}

	// Check if apt is available
	if err := am.checkAptAvailable(); err != nil {
		return nil, fmt.Errorf("apt not available: %w", err)
	}

	args := []string{"depends", packageName}
	cmd := exec.Command("apt", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil, fmt.Errorf("apt depends failed for package %s: %w", packageName, err)
	}

	var dependencies []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, packageName+":") {
			continue
		}
		
		// APT depends output format: "  Depends: packagename"
		if strings.HasPrefix(line, "Depends:") || strings.HasPrefix(line, "  ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Remove "Depends:" prefix and extract package name
				depName := parts[len(parts)-1]
				// Clean up package name (remove version constraints)
				if idx := strings.Index(depName, "("); idx != -1 {
					depName = depName[:idx]
				}
				if depName != "" && depName != packageName {
					dependencies = append(dependencies, depName)
				}
			}
		}
	}

	am.logger.Debug("Found package dependencies", "package", packageName, "count", len(dependencies))
	return dependencies, nil
}

// SimulateInstall simulates package installation to show what would be installed
func (am *AptManager) SimulateInstall(packageNames []string, flags []string) ([]string, error) {
	if len(packageNames) == 0 {
		return nil, fmt.Errorf("no packages specified")
	}

	if am.dryRun {
		am.logger.Info("DRY RUN: Would simulate package installation", "packages", packageNames)
		return []string{}, nil
	}

	// Check if apt is available
	if err := am.checkAptAvailable(); err != nil {
		return nil, fmt.Errorf("apt not available: %w", err)
	}

	// Build apt install command with simulation flag
	args := append([]string{"install", "--simulate"}, flags...)
	args = append(args, packageNames...)

	cmd := exec.Command("apt", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return nil, fmt.Errorf("apt simulate failed: %w", err)
	}

	var willInstall []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for "Inst" lines which show what would be installed
		if strings.HasPrefix(line, "Inst ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				packageName := parts[1]
				willInstall = append(willInstall, packageName)
			}
		}
	}

	am.logger.Debug("Simulated package installation", "packages", packageNames, "will_install", len(willInstall))
	return willInstall, nil
}