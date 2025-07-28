package pkg

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

// RepositoryManager handles repository management operations for APT and Flatpak
type RepositoryManager struct {
	logger *log.Logger
	dryRun bool
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(logger *log.Logger, dryRun bool) *RepositoryManager {
	return &RepositoryManager{
		logger: logger,
		dryRun: dryRun,
	}
}

// AddRepositories adds both APT and Flatpak repositories
func (rm *RepositoryManager) AddRepositories(repositories config.RepositoryManagement) error {
	// Add APT repositories first (they may be needed for package installations)
	if err := rm.addAptRepositories(repositories.Apt); err != nil {
		return fmt.Errorf("failed to add APT repositories: %w", err)
	}

	// Add Flatpak repositories
	if err := rm.addFlatpakRepositories(repositories.Flatpak); err != nil {
		return fmt.Errorf("failed to add Flatpak repositories: %w", err)
	}

	return nil
}

// addAptRepositories handles APT repository management using DEB822 format
func (rm *RepositoryManager) addAptRepositories(repos []config.AptRepository) error {
	if len(repos) == 0 {
		rm.logger.Debug("No APT repositories to add")
		return nil
	}

	rm.logger.Info("Managing APT repositories using DEB822 format...", "count", len(repos))

	// Check Ubuntu version compatibility (24.04+)
	if !rm.dryRun {
		if err := rm.checkUbuntuVersionCompatibility(); err != nil {
			return fmt.Errorf("Ubuntu version compatibility check failed: %w", err)
		}
	}

	for _, repo := range repos {
		if err := rm.addAptRepositoryDEB822(repo); err != nil {
			return fmt.Errorf("failed to add APT repository '%s': %w", repo.Name, err)
		}
	}

	rm.logger.Info("✓ APT repositories processed successfully")
	return nil
}

// addAptRepositoryDEB822 adds a single APT repository using DEB822 format
func (rm *RepositoryManager) addAptRepositoryDEB822(repo config.AptRepository) error {
	// Convert legacy format to DEB822 if needed
	repo, err := rm.convertLegacyToRepository(repo)
	if err != nil {
		return fmt.Errorf("failed to convert legacy repository format: %w", err)
	}

	// Handle GPG key installation first
	if err := rm.installGPGKeyDEB822(repo); err != nil {
		return fmt.Errorf("failed to install GPG key: %w", err)
	}

	// Create DEB822 sources file
	if err := rm.createDEB822SourcesFile(repo); err != nil {
		return fmt.Errorf("failed to create sources file: %w", err)
	}

	return nil
}

// convertLegacyToRepository converts legacy repository format to DEB822
func (rm *RepositoryManager) convertLegacyToRepository(repo config.AptRepository) (config.AptRepository, error) {
	// Handle PPA conversion
	if repo.PPA != "" {
		rm.logger.Debug("Converting PPA to DEB822 format", "ppa", repo.PPA)
		return rm.convertPPAToDEB822(repo)
	}

	// Handle legacy URI conversion
	if repo.URI != "" {
		rm.logger.Debug("Converting legacy URI to DEB822 format", "uri", repo.URI)
		return rm.convertLegacyURIToDEB822(repo)
	}

	// Already in DEB822 format, validate required fields
	if len(repo.URIs) == 0 {
		return repo, fmt.Errorf("repository '%s' missing URIs field", repo.Name)
	}
	if len(repo.Suites) == 0 {
		return repo, fmt.Errorf("repository '%s' missing Suites field", repo.Name)
	}
	if len(repo.Components) == 0 {
		return repo, fmt.Errorf("repository '%s' missing Components field", repo.Name)
	}

	// Set defaults for optional fields
	if len(repo.Types) == 0 {
		repo.Types = []string{"deb"}
	}
	if len(repo.Architectures) == 0 {
		repo.Architectures = []string{"amd64"}
	}

	return repo, nil
}

// convertPPAToDEB822 converts a PPA to DEB822 format
func (rm *RepositoryManager) convertPPAToDEB822(repo config.AptRepository) (config.AptRepository, error) {
	// Get Ubuntu codename for PPA
	codename, err := rm.getUbuntuCodename()
	if err != nil {
		return repo, fmt.Errorf("failed to get Ubuntu codename: %w", err)
	}

	// Convert PPA format to DEB822
	repo.URIs = []string{fmt.Sprintf("https://ppa.launchpadcontent.net/%s/ubuntu", repo.PPA)}
	repo.Suites = []string{codename}
	repo.Components = []string{"main"}
	repo.Types = []string{"deb"}
	repo.Architectures = []string{"amd64"}
	
	// PPA GPG key will be fetched from Launchpad
	repo.KeyURL = fmt.Sprintf("https://keyserver.ubuntu.com/pks/lookup?op=get&search=0x%s", rm.getPPAKeyFingerprint(repo.PPA))
	repo.SignedBy = fmt.Sprintf("/usr/share/keyrings/%s-ppa.gpg", strings.ReplaceAll(repo.Name, "_", "-"))

	// Clear legacy fields
	repo.PPA = ""
	repo.Key = ""

	return repo, nil
}

// convertLegacyURIToDEB822 converts legacy URI format to DEB822
func (rm *RepositoryManager) convertLegacyURIToDEB822(repo config.AptRepository) (config.AptRepository, error) {
	// Parse legacy URI format: "deb [arch=amd64] https://example.com/repo stable main"
	parts := strings.Fields(repo.URI)
	if len(parts) < 4 {
		return repo, fmt.Errorf("invalid legacy URI format: %s", repo.URI)
	}

	repo.Types = []string{parts[0]} // "deb" or "deb-src"
	
	// Handle architecture specification
	if strings.HasPrefix(parts[1], "[") {
		// Parse [arch=amd64] format
		archSpec := parts[1]
		if strings.Contains(archSpec, "arch=") {
			archPart := strings.TrimPrefix(strings.TrimSuffix(archSpec, "]"), "[arch=")
			repo.Architectures = strings.Split(archPart, ",")
		}
		repo.URIs = []string{parts[2]}
		repo.Suites = []string{parts[3]}
		if len(parts) > 4 {
			repo.Components = parts[4:]
		}
	} else {
		repo.URIs = []string{parts[1]}
		repo.Suites = []string{parts[2]}
		if len(parts) > 3 {
			repo.Components = parts[3:]
		}
	}

	// Set defaults
	if len(repo.Architectures) == 0 {
		repo.Architectures = []string{"amd64"}
	}
	if len(repo.Components) == 0 {
		repo.Components = []string{"main"}
	}

	// Handle legacy key field
	if repo.Key != "" {
		if strings.HasPrefix(repo.Key, "https://") {
			repo.KeyURL = repo.Key
		} else {
			repo.KeyID = repo.Key
		}
		repo.SignedBy = fmt.Sprintf("/usr/share/keyrings/%s.gpg", strings.ReplaceAll(repo.Name, "_", "-"))
		repo.Key = ""
	}

	// Clear legacy fields
	repo.URI = ""

	return repo, nil
}

// installGPGKeyDEB822 installs GPG key for DEB822 repository
func (rm *RepositoryManager) installGPGKeyDEB822(repo config.AptRepository) error {
	if repo.KeyURL == "" && repo.KeyID == "" {
		rm.logger.Debug("No GPG key specified for repository", "name", repo.Name)
		return nil
	}

	if repo.SignedBy == "" {
		return fmt.Errorf("repository '%s' has key but no SignedBy path specified", repo.Name)
	}

	rm.logger.Info("Installing GPG key for repository", "name", repo.Name, "signed_by", repo.SignedBy)

	if rm.dryRun {
		if repo.KeyURL != "" {
			rm.logger.Info("  [DRY RUN] Would download key from:", "url", repo.KeyURL)
		} else {
			rm.logger.Info("  [DRY RUN] Would fetch key from keyserver:", "key_id", repo.KeyID)
		}
		rm.logger.Info("  [DRY RUN] Would save key to:", "path", repo.SignedBy)
		return nil
	}

	// Ensure keyrings directory exists
	keyringDir := filepath.Dir(repo.SignedBy)
	if err := os.MkdirAll(keyringDir, 0755); err != nil {
		return fmt.Errorf("failed to create keyring directory %s: %w", keyringDir, err)
	}

	if repo.KeyURL != "" {
		return rm.downloadAndInstallKey(repo.KeyURL, repo.SignedBy, repo.Name)
	} else {
		return rm.fetchAndInstallKeyFromKeyserver(repo.KeyID, repo.SignedBy, repo.Name)
	}
}

// downloadAndInstallKey downloads a GPG key from URL and installs it
func (rm *RepositoryManager) downloadAndInstallKey(keyURL, keyPath, repoName string) error {
	rm.logger.Debug("Downloading GPG key from URL", "url", keyURL, "path", keyPath)

	// Download the key
	resp, err := http.Get(keyURL)
	if err != nil {
		return fmt.Errorf("failed to download GPG key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download GPG key: HTTP %d", resp.StatusCode)
	}

	// Create temporary file for processing
	tmpFile, err := os.CreateTemp("", "configr-key-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy downloaded key to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save downloaded key: %w", err)
	}
	tmpFile.Close()

	// Convert to binary format and install
	return rm.convertAndInstallKey(tmpFile.Name(), keyPath, repoName)
}

// fetchAndInstallKeyFromKeyserver fetches a GPG key from keyserver and installs it
func (rm *RepositoryManager) fetchAndInstallKeyFromKeyserver(keyID, keyPath, repoName string) error {
	rm.logger.Debug("Fetching GPG key from keyserver", "key_id", keyID, "path", keyPath)

	// Clean key ID
	cleanKeyID := strings.TrimPrefix(keyID, "0x")
	
	// Create temporary file for processing
	tmpFile, err := os.CreateTemp("", "configr-key-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Fetch key from keyserver using gpg
	cmd := exec.Command("gpg", "--batch", "--quiet", "--keyserver", "keyserver.ubuntu.com", 
		"--recv-keys", cleanKeyID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to fetch key %s from keyserver: %w", cleanKeyID, err)
	}

	// Export key to temporary file
	cmd = exec.Command("gpg", "--batch", "--quiet", "--armor", "--export", cleanKeyID)
	cmd.Stdout = tmpFile
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to export key %s: %w", cleanKeyID, err)
	}
	tmpFile.Close()

	// Convert to binary format and install
	return rm.convertAndInstallKey(tmpFile.Name(), keyPath, repoName)
}

// convertAndInstallKey converts ASCII armored key to binary format and installs it
func (rm *RepositoryManager) convertAndInstallKey(tmpPath, keyPath, repoName string) error {
	// Convert ASCII armored key to binary format using gpg --dearmor
	cmd := exec.Command("gpg", "--batch", "--quiet", "--dearmor", "--output", keyPath, tmpPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to convert key to binary format: %w", err)
	}

	// Set appropriate permissions
	if err := os.Chmod(keyPath, 0644); err != nil {
		return fmt.Errorf("failed to set key permissions: %w", err)
	}

	rm.logger.Debug("GPG key installed successfully", "repo", repoName, "path", keyPath)
	return nil
}

// createDEB822SourcesFile creates a DEB822 format sources file
func (rm *RepositoryManager) createDEB822SourcesFile(repo config.AptRepository) error {
	sourcesPath := fmt.Sprintf("/etc/apt/sources.list.d/%s.sources", strings.ReplaceAll(repo.Name, "_", "-"))
	
	rm.logger.Info("Creating DEB822 sources file", "name", repo.Name, "path", sourcesPath)

	content := rm.generateDEB822Content(repo)

	if rm.dryRun {
		rm.logger.Info("  [DRY RUN] Would create file:", "path", sourcesPath)
		rm.logger.Info("  [DRY RUN] File content:\n" + content)
		return nil
	}

	// Create the sources file
	if err := os.WriteFile(sourcesPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create sources file %s: %w", sourcesPath, err)
	}

	rm.logger.Debug("DEB822 sources file created successfully", "path", sourcesPath)
	return nil
}

// generateDEB822Content generates the content for a DEB822 sources file
func (rm *RepositoryManager) generateDEB822Content(repo config.AptRepository) string {
	var content strings.Builder
	
	content.WriteString("### THIS FILE IS AUTOMATICALLY CONFIGURED ###\n")
	content.WriteString("# You may comment out this entry, but any other modifications may be lost.\n")
	
	// Types field
	content.WriteString(fmt.Sprintf("Types: %s\n", strings.Join(repo.Types, " ")))
	
	// URIs field  
	content.WriteString(fmt.Sprintf("URIs: %s\n", strings.Join(repo.URIs, " ")))
	
	// Suites field
	content.WriteString(fmt.Sprintf("Suites: %s\n", strings.Join(repo.Suites, " ")))
	
	// Components field
	content.WriteString(fmt.Sprintf("Components: %s\n", strings.Join(repo.Components, " ")))
	
	// Architectures field
	content.WriteString(fmt.Sprintf("Architectures: %s\n", strings.Join(repo.Architectures, ",")))
	
	// Signed-By field (if specified)
	if repo.SignedBy != "" {
		content.WriteString(fmt.Sprintf("Signed-By: %s\n", repo.SignedBy))
	}
	
	// Trusted field (if enabled)
	if repo.Trusted {
		content.WriteString("Trusted: yes\n")
	}
	
	return content.String()
}

// Helper methods for system compatibility and information

// checkUbuntuVersionCompatibility checks if running on Ubuntu 24.04+
func (rm *RepositoryManager) checkUbuntuVersionCompatibility() error {
	// Check /etc/os-release for Ubuntu version
	content, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return fmt.Errorf("failed to read /etc/os-release: %w", err)
	}

	osInfo := string(content)
	if !strings.Contains(osInfo, "Ubuntu") {
		return fmt.Errorf("DEB822 format requires Ubuntu 24.04+ (detected non-Ubuntu system)")
	}

	// Extract version
	for _, line := range strings.Split(osInfo, "\n") {
		if strings.HasPrefix(line, "VERSION_ID=") {
			version := strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
			if rm.isVersionSupported(version) {
				return nil
			}
			return fmt.Errorf("DEB822 format requires Ubuntu 24.04+ (detected Ubuntu %s)", version)
		}
	}

	return fmt.Errorf("could not determine Ubuntu version")
}

// isVersionSupported checks if Ubuntu version supports DEB822 format
func (rm *RepositoryManager) isVersionSupported(version string) bool {
	// Support Ubuntu 24.04 and newer
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}

	major := parts[0]
	minor := parts[1]
	
	// 24.04 and newer
	if major == "24" && minor >= "04" {
		return true
	}
	
	// Versions 25+ and beyond
	if major > "24" {
		return true
	}
	
	return false
}

// getUbuntuCodename gets the Ubuntu codename for PPA conversion
func (rm *RepositoryManager) getUbuntuCodename() (string, error) {
	// Check /etc/os-release for codename
	content, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("failed to read /etc/os-release: %w", err)
	}

	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(line, "VERSION_CODENAME=") {
			return strings.Trim(strings.TrimPrefix(line, "VERSION_CODENAME="), "\""), nil
		}
	}

	// Fallback: try lsb_release
	cmd := exec.Command("lsb_release", "-cs")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Ubuntu codename: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// getPPAKeyFingerprint gets the key fingerprint for a PPA (simplified approach)
func (rm *RepositoryManager) getPPAKeyFingerprint(ppa string) string {
	// For demo purposes, return a placeholder
	// In real implementation, this would query Launchpad API
	h := sha256.Sum256([]byte(ppa))
	return fmt.Sprintf("%X", h[:8])
}

// addFlatpakRepositories handles Flatpak repository management
func (rm *RepositoryManager) addFlatpakRepositories(repos []config.FlatpakRepository) error {
	if len(repos) == 0 {
		rm.logger.Debug("No Flatpak repositories to add")
		return nil
	}

	rm.logger.Info("Managing Flatpak repositories...", "count", len(repos))

	// Check if flatpak is available (skip in dry-run for testing)
	if !rm.dryRun {
		if err := rm.checkFlatpakAvailable(); err != nil {
			return fmt.Errorf("flatpak not available: %w", err)
		}
	}

	for _, repo := range repos {
		if err := rm.addFlatpakRepository(repo); err != nil {
			return fmt.Errorf("failed to add Flatpak repository '%s': %w", repo.Name, err)
		}
	}

	rm.logger.Info("✓ Flatpak repositories processed successfully")
	return nil
}

// addFlatpakRepository adds a single Flatpak repository
func (rm *RepositoryManager) addFlatpakRepository(repo config.FlatpakRepository) error {
	args := []string{"flatpak", "remote-add", "--if-not-exists"}

	// Add user or system flag
	if repo.User {
		args = append(args, "--user")
	} else {
		args = append(args, "--system")
	}

	args = append(args, repo.Name, repo.URL)

	rm.logger.Info("Adding Flatpak repository", "name", repo.Name, "url", repo.URL, "user", repo.User)

	if rm.dryRun {
		rm.logger.Info("  [DRY RUN] Would run:", "command", strings.Join(args, " "))
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		rm.logger.Error("Failed to add Flatpak repository", "name", repo.Name, "error", err, "output", string(output))
		return fmt.Errorf("flatpak remote-add failed: %w", err)
	}

	rm.logger.Debug("Flatpak repository added successfully", "name", repo.Name, "output", string(output))
	return nil
}

// checkGPGAvailable checks if gpg command is available for key management
func (rm *RepositoryManager) checkGPGAvailable() error {
	if _, err := exec.LookPath("gpg"); err != nil {
		return fmt.Errorf("gpg command not found - install gnupg package")
	}
	return nil
}

// checkFlatpakAvailable checks if flatpak command is available
func (rm *RepositoryManager) checkFlatpakAvailable() error {
	if _, err := exec.LookPath("flatpak"); err != nil {
		return fmt.Errorf("flatpak command not found - install flatpak package")
	}
	return nil
}