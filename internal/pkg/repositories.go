package pkg

import (
	"fmt"
	"os/exec"
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

// addAptRepositories handles APT repository management
func (rm *RepositoryManager) addAptRepositories(repos []config.AptRepository) error {
	if len(repos) == 0 {
		rm.logger.Debug("No APT repositories to add")
		return nil
	}

	rm.logger.Info("Managing APT repositories...", "count", len(repos))

	// Check if add-apt-repository is available (skip in dry-run for testing)
	if !rm.dryRun {
		if err := rm.checkAddAptRepositoryAvailable(); err != nil {
			return fmt.Errorf("add-apt-repository not available: %w", err)
		}
	}

	for _, repo := range repos {
		if err := rm.addAptRepository(repo); err != nil {
			return fmt.Errorf("failed to add APT repository '%s': %w", repo.Name, err)
		}
	}

	rm.logger.Info("✓ APT repositories processed successfully")
	return nil
}

// addAptRepository adds a single APT repository
func (rm *RepositoryManager) addAptRepository(repo config.AptRepository) error {
	// Handle PPA repositories
	if repo.PPA != "" {
		return rm.addPPARepository(repo)
	}

	// Handle custom URI repositories
	if repo.URI != "" {
		return rm.addCustomRepository(repo)
	}

	return fmt.Errorf("repository '%s' has no PPA or URI specified", repo.Name)
}

// addPPARepository adds a PPA repository using add-apt-repository
func (rm *RepositoryManager) addPPARepository(repo config.AptRepository) error {
	ppaArg := fmt.Sprintf("ppa:%s", repo.PPA)
	args := []string{"add-apt-repository", "-y", ppaArg}

	rm.logger.Info("Adding PPA repository", "name", repo.Name, "ppa", repo.PPA)

	if rm.dryRun {
		rm.logger.Info("  [DRY RUN] Would run:", "command", fmt.Sprintf("sudo %s", strings.Join(args, " ")))
		return nil
	}

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		rm.logger.Error("Failed to add PPA repository", "name", repo.Name, "error", err, "output", string(output))
		return fmt.Errorf("add-apt-repository failed: %w", err)
	}

	rm.logger.Debug("PPA repository added successfully", "name", repo.Name, "output", string(output))
	return nil
}

// addCustomRepository adds a custom repository with optional GPG key
func (rm *RepositoryManager) addCustomRepository(repo config.AptRepository) error {
	rm.logger.Info("Adding custom APT repository", "name", repo.Name, "uri", repo.URI)

	// Add GPG key first if provided
	if repo.Key != "" {
		if err := rm.addGPGKey(repo.Key, repo.Name); err != nil {
			return fmt.Errorf("failed to add GPG key for repository '%s': %w", repo.Name, err)
		}
	}

	// Add the repository
	args := []string{"add-apt-repository", "-y", repo.URI}

	if rm.dryRun {
		rm.logger.Info("  [DRY RUN] Would run:", "command", fmt.Sprintf("sudo %s", strings.Join(args, " ")))
		return nil
	}

	cmd := exec.Command("sudo", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		rm.logger.Error("Failed to add custom repository", "name", repo.Name, "error", err, "output", string(output))
		return fmt.Errorf("add-apt-repository failed: %w", err)
	}

	rm.logger.Debug("Custom repository added successfully", "name", repo.Name, "output", string(output))
	return nil
}

// addGPGKey adds a GPG key for repository authentication
func (rm *RepositoryManager) addGPGKey(key, repoName string) error {
	rm.logger.Info("Adding GPG key", "repo", repoName, "key", key)

	if rm.dryRun {
		if strings.HasPrefix(key, "https://") {
			rm.logger.Info("  [DRY RUN] Would run:", "command", fmt.Sprintf("wget -qO- %s | sudo apt-key add -", key))
		} else {
			rm.logger.Info("  [DRY RUN] Would run:", "command", fmt.Sprintf("sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys %s", strings.TrimPrefix(key, "0x")))
		}
		return nil
	}

	var cmd *exec.Cmd
	var cmdDescription string

	if strings.HasPrefix(key, "https://") {
		// Handle URL-based keys
		cmdDescription = fmt.Sprintf("wget -qO- %s | sudo apt-key add -", key)
		cmd = exec.Command("bash", "-c", cmdDescription)
	} else {
		// Handle keyserver-based keys
		keyID := strings.TrimPrefix(key, "0x")
		args := []string{"apt-key", "adv", "--keyserver", "keyserver.ubuntu.com", "--recv-keys", keyID}
		cmdDescription = fmt.Sprintf("sudo %s", strings.Join(args, " "))
		cmd = exec.Command("sudo", args...)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		rm.logger.Error("Failed to add GPG key", "repo", repoName, "key", key, "error", err, "output", string(output))
		return fmt.Errorf("GPG key installation failed: %w", err)
	}

	rm.logger.Debug("GPG key added successfully", "repo", repoName, "output", string(output))
	return nil
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

// checkAddAptRepositoryAvailable checks if add-apt-repository command is available
func (rm *RepositoryManager) checkAddAptRepositoryAvailable() error {
	if _, err := exec.LookPath("add-apt-repository"); err != nil {
		return fmt.Errorf("add-apt-repository command not found - install software-properties-common package")
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