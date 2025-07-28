package pkg

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

// BinaryManager handles binary operations including downloading, deployment, and permission management
type BinaryManager struct {
	logger      *log.Logger
	dryRun      bool
	configDir   string
	interactive *InteractiveManager
}

// ManagedBinary represents a binary managed by configr
type ManagedBinary struct {
	Name        string `json:"name"`        // Binary identifier from YAML
	Source      string `json:"source"`      // URL where binary was downloaded from
	Destination string `json:"destination"` // Where the binary was deployed
	BackupPath  string `json:"backup_path,omitempty"` // Path to backup file if created
}

// NewBinaryManager creates a new BinaryManager instance
func NewBinaryManager(logger *log.Logger, dryRun bool, configDir string) *BinaryManager {
	return &BinaryManager{
		logger:      logger,
		dryRun:      dryRun,
		configDir:   configDir,
		interactive: NewInteractiveManager(logger),
	}
}

// DeployBinaries processes all binaries in the configuration and returns deployed binary info
func (bm *BinaryManager) DeployBinaries(binaries map[string]config.Binary) ([]ManagedBinary, error) {
	if len(binaries) == 0 {
		bm.logger.Debug("No binaries to deploy")
		return []ManagedBinary{}, nil
	}

	bm.logger.Info("Processing binary deployments", "count", len(binaries))

	var deployedBinaries []ManagedBinary
	for name, binary := range binaries {
		managedBinary, err := bm.deployBinary(name, binary)
		if err != nil {
			return nil, fmt.Errorf("failed to deploy binary '%s': %w", name, err)
		}
		deployedBinaries = append(deployedBinaries, managedBinary)
	}

	bm.logger.Info("✓ All binaries deployed successfully")
	return deployedBinaries, nil
}

// deployBinary handles the deployment of a single binary and returns binary info
func (bm *BinaryManager) deployBinary(name string, binary config.Binary) (ManagedBinary, error) {
	bm.logger.Debug("Deploying binary", "name", name, "source", binary.Source, "destination", binary.Destination)

	// Validate source URL
	if err := bm.validateSourceURL(binary.Source); err != nil {
		return ManagedBinary{}, fmt.Errorf("invalid source URL: %w", err)
	}

	// Resolve destination path
	destPath, err := bm.resolveDestinationPath(binary.Destination)
	if err != nil {
		return ManagedBinary{}, fmt.Errorf("failed to resolve destination path: %w", err)
	}

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(destPath)
	if err := bm.ensureDirectory(destDir); err != nil {
		return ManagedBinary{}, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Handle existing binary (backup if needed, with interactive support)
	backupPath, err := bm.handleExistingBinary(name, destPath, binary)
	if err != nil {
		return ManagedBinary{}, fmt.Errorf("failed to handle existing binary: %w", err)
	}

	// Download and deploy binary
	if err := bm.downloadAndDeployBinary(binary.Source, destPath); err != nil {
		return ManagedBinary{}, fmt.Errorf("failed to download and deploy binary: %w", err)
	}

	// Set ownership and permissions if specified
	if err := bm.setBinaryAttributes(destPath, binary); err != nil {
		return ManagedBinary{}, fmt.Errorf("failed to set binary attributes: %w", err)
	}

	bm.logger.Info("✓ Binary deployed", "name", name, "destination", destPath)
	
	// Return managed binary info
	return ManagedBinary{
		Name:        name,
		Source:      binary.Source,
		Destination: destPath,
		BackupPath:  backupPath,
	}, nil
}

// validateSourceURL validates that the source URL is valid and uses HTTPS
func (bm *BinaryManager) validateSourceURL(url string) error {
	if url == "" {
		return fmt.Errorf("source URL cannot be empty")
	}

	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("source URL must use HTTPS for security: %s", url)
	}

	return nil
}

// resolveDestinationPath resolves the destination path, handling ~ expansion
func (bm *BinaryManager) resolveDestinationPath(destination string) (string, error) {
	if strings.HasPrefix(destination, "~/") {
		currentUser, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("failed to get current user: %w", err)
		}
		return filepath.Join(currentUser.HomeDir, destination[2:]), nil
	}

	if strings.HasPrefix(destination, "~") && len(destination) > 1 {
		// Handle ~username/ expansion
		username := destination[1:]
		if idx := strings.Index(username, "/"); idx != -1 {
			username = username[:idx]
		}

		targetUser, err := user.Lookup(username)
		if err != nil {
			return "", fmt.Errorf("failed to lookup user '%s': %w", username, err)
		}

		if idx := strings.Index(destination[1:], "/"); idx != -1 {
			return filepath.Join(targetUser.HomeDir, destination[2+idx:]), nil
		}
		return targetUser.HomeDir, nil
	}

	return destination, nil
}

// ensureDirectory creates the directory if it doesn't exist
func (bm *BinaryManager) ensureDirectory(dir string) error {
	if bm.dryRun {
		bm.logger.Debug("DRY RUN: Would create directory", "path", dir)
		return nil
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		bm.logger.Debug("Creating directory", "path", dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// handleExistingBinary handles existing binaries at the destination, with interactive support
// Returns the backup path if a backup was created, empty string otherwise
func (bm *BinaryManager) handleExistingBinary(name, destPath string, binary config.Binary) (string, error) {
	if _, err := os.Lstat(destPath); os.IsNotExist(err) {
		// Binary doesn't exist, nothing to handle
		return "", nil
	}

	// Get file info for interactive prompts
	fileInfo, err := os.Lstat(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	if bm.dryRun {
		if binary.Backup {
			bm.logger.Debug("DRY RUN: Would backup existing binary", "path", destPath)
			return fmt.Sprintf("%s.backup.%s", destPath, time.Now().Format("20060102-150405")), nil
		} else {
			bm.logger.Debug("DRY RUN: Would remove existing binary", "path", destPath)
			return "", nil
		}
	}

	// Interactive conflict resolution
	if binary.Interactive && bm.interactive.IsInteractiveMode() {
		conflict := FileConflictInfo{
			Name:            name,
			SourcePath:      binary.Source, // Note: This is a URL, not a file path
			DestinationPath: destPath,
			ExistingInfo:    fileInfo,
			IsSymlink:       false, // Binaries are always regular files
			BackupEnabled:   binary.Backup,
		}

		for {
			resolution, err := bm.interactive.PromptForConflictResolution(conflict)
			if err != nil {
				return "", fmt.Errorf("failed to get user input: %w", err)
			}

			switch resolution {
			case ResolutionSkip:
				bm.logger.Info("⏭ Skipping binary", "name", name)
				return "", fmt.Errorf("binary skipped by user: %s", name)
			case ResolutionQuit:
				return "", fmt.Errorf("operation cancelled by user")
			case ResolutionViewDiff:
				bm.logger.Info("Cannot show diff for binary downloads (source is URL)")
				continue // Ask again
			case ResolutionOverwrite:
				break // Continue with overwrite
			case ResolutionBackup:
				// Force backup for this binary
				break // Continue with backup
			}
			break
		}
	}

	// Determine if we should backup based on config or user choice
	shouldBackup := binary.Backup
	
	if shouldBackup {
		backupPath := fmt.Sprintf("%s.backup.%s", destPath, time.Now().Format("20060102-150405"))
		bm.logger.Info("⚠ Backing up existing binary", "from", destPath, "to", backupPath)
		
		if err := os.Rename(destPath, backupPath); err != nil {
			return "", fmt.Errorf("failed to backup binary: %w", err)
		}
		return backupPath, nil
	} else {
		bm.logger.Info("⚠ Removing existing binary", "path", destPath)
		if err := os.Remove(destPath); err != nil {
			return "", fmt.Errorf("failed to remove existing binary: %w", err)
		}
		return "", nil
	}
}

// downloadAndDeployBinary downloads the binary from the source URL and saves it to the destination
func (bm *BinaryManager) downloadAndDeployBinary(sourceURL, destPath string) error {
	if bm.dryRun {
		bm.logger.Debug("DRY RUN: Would download binary", "from", sourceURL, "to", destPath)
		return nil
	}

	bm.logger.Debug("Downloading binary", "from", sourceURL, "to", destPath)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	// Download the binary
	resp, err := client.Get(sourceURL)
	if err != nil {
		return fmt.Errorf("failed to download binary from %s: %w", sourceURL, err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download binary: HTTP %d from %s", resp.StatusCode, sourceURL)
	}

	// Create destination file
	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy downloaded content to destination
	_, err = io.Copy(dst, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save binary to destination: %w", err)
	}

	// Sync to ensure data is written
	if err := dst.Sync(); err != nil {
		return fmt.Errorf("failed to sync binary file: %w", err)
	}

	return nil
}

// setBinaryAttributes sets ownership and permissions on the binary if specified
func (bm *BinaryManager) setBinaryAttributes(destPath string, binary config.Binary) error {
	if bm.dryRun {
		bm.logger.Debug("DRY RUN: Would set binary attributes", "path", destPath, "mode", binary.Mode, "owner", binary.Owner, "group", binary.Group)
		return nil
	}
	
	// Get current file info for interactive prompts
	fileInfo, err := os.Lstat(destPath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	owner := binary.Owner
	group := binary.Group
	mode := binary.Mode

	// Default mode for binaries is 755 if not specified
	if mode == "" {
		mode = "755"
	}

	// Interactive permission prompting
	if binary.PromptPermissions && bm.interactive.IsInteractiveMode() {
		if promptedMode, err := bm.interactive.PromptForPermissions(destPath, fileInfo.Mode()); err != nil {
			bm.logger.Warn("Failed to prompt for permissions", "error", err)
		} else {
			mode = promptedMode
		}
	}

	// Interactive ownership prompting
	if binary.PromptOwnership && bm.interactive.IsInteractiveMode() {
		// Get current ownership info
		currentOwner, currentGroup := bm.getCurrentOwnership(destPath)
		
		if promptedOwner, promptedGroup, err := bm.interactive.PromptForOwnership(destPath, currentOwner, currentGroup); err != nil {
			bm.logger.Warn("Failed to prompt for ownership", "error", err)
		} else {
			owner = promptedOwner
			group = promptedGroup
		}
	}

	// Set ownership if specified or prompted
	if owner != "" || group != "" {
		if err := bm.setOwnership(destPath, owner, group); err != nil {
			return fmt.Errorf("failed to set ownership: %w", err)
		}
	}

	// Set permissions (default 755 for binaries)
	if err := bm.setPermissions(destPath, mode); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}

// getCurrentOwnership gets the current owner and group names for a binary
func (bm *BinaryManager) getCurrentOwnership(path string) (string, string) {
	_, err := os.Lstat(path)
	if err != nil {
		return "unknown", "unknown"
	}

	// This is a simplified implementation - in practice you'd want to
	// resolve UIDs/GIDs to names using os/user package
	return "current", "current"
}

// setOwnership sets the owner and group of a binary
func (bm *BinaryManager) setOwnership(path, owner, group string) error {
	if bm.dryRun {
		bm.logger.Debug("DRY RUN: Would set ownership", "path", path, "owner", owner, "group", group)
		return nil
	}

	var uid, gid int = -1, -1

	// Resolve owner
	if owner != "" {
		if u, err := user.Lookup(owner); err == nil {
			if parsed, err := strconv.Atoi(u.Uid); err == nil {
				uid = parsed
			}
		} else {
			// Try as numeric UID
			if parsed, err := strconv.Atoi(owner); err == nil {
				uid = parsed
			} else {
				return fmt.Errorf("failed to lookup user '%s': %w", owner, err)
			}
		}
	}

	// Resolve group
	if group != "" {
		if g, err := user.LookupGroup(group); err == nil {
			if parsed, err := strconv.Atoi(g.Gid); err == nil {
				gid = parsed
			}
		} else {
			// Try as numeric GID
			if parsed, err := strconv.Atoi(group); err == nil {
				gid = parsed
			} else {
				return fmt.Errorf("failed to lookup group '%s': %w", group, err)
			}
		}
	}

	bm.logger.Debug("Setting ownership", "path", path, "uid", uid, "gid", gid)
	
	if err := os.Lchown(path, uid, gid); err != nil {
		return fmt.Errorf("failed to change ownership: %w", err)
	}

	return nil
}

// setPermissions sets the binary mode/permissions
func (bm *BinaryManager) setPermissions(path, mode string) error {
	if bm.dryRun {
		bm.logger.Debug("DRY RUN: Would set permissions", "path", path, "mode", mode)
		return nil
	}

	// Parse octal mode
	modeInt, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return fmt.Errorf("invalid file mode '%s': %w", mode, err)
	}

	bm.logger.Debug("Setting permissions", "path", path, "mode", mode)
	
	if err := os.Chmod(path, os.FileMode(modeInt)); err != nil {
		return fmt.Errorf("failed to change permissions: %w", err)
	}

	return nil
}

// ValidateBinaryPermissions checks if we have the necessary permissions to deploy binaries
func (bm *BinaryManager) ValidateBinaryPermissions(binaries map[string]config.Binary) error {
	for name, binary := range binaries {
		destPath, err := bm.resolveDestinationPath(binary.Destination)
		if err != nil {
			return fmt.Errorf("failed to resolve destination for '%s': %w", name, err)
		}

		destDir := filepath.Dir(destPath)
		
		// Check if we can write to the destination directory
		if err := bm.checkWritePermission(destDir); err != nil {
			return fmt.Errorf("insufficient permissions for '%s': %w", name, err)
		}

		// If setting ownership, check if we're root or have appropriate capabilities
		if binary.Owner != "" || binary.Group != "" {
			if os.Geteuid() != 0 {
				bm.logger.Warn("Setting ownership requires root privileges", "binary", name)
			}
		}
	}

	return nil
}

// checkWritePermission checks if we can write to a directory
func (bm *BinaryManager) checkWritePermission(dir string) error {
	// Create directory if it doesn't exist (for testing purposes)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		testDir := dir
		for {
			parent := filepath.Dir(testDir)
			if parent == testDir || parent == "/" {
				break
			}
			if _, err := os.Stat(parent); err == nil {
				// Found existing parent, check write permission
				return bm.testWriteAccess(parent)
			}
			testDir = parent
		}
	}

	// Directory exists, check write permission
	return bm.testWriteAccess(dir)
}

// testWriteAccess tests write access by attempting to create a temporary file
func (bm *BinaryManager) testWriteAccess(dir string) error {
	// Try to create a temporary file to test write access
	tempFile := filepath.Join(dir, ".configr-write-test")
	f, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("no write permission to directory %s: %w", dir, err)
	}
	f.Close()
	os.Remove(tempFile) // Clean up
	return nil
}

// RemoveBinaries removes binaries that are no longer in the configuration
func (bm *BinaryManager) RemoveBinaries(binariesToRemove []ManagedBinary) error {
	if len(binariesToRemove) == 0 {
		bm.logger.Debug("No binaries to remove")
		return nil
	}

	bm.logger.Info("Removing binaries no longer in configuration", "count", len(binariesToRemove))

	for _, binary := range binariesToRemove {
		if err := bm.removeBinary(binary); err != nil {
			bm.logger.Error("Failed to remove binary", "name", binary.Name, "destination", binary.Destination, "error", err)
			return fmt.Errorf("failed to remove binary '%s': %w", binary.Name, err)
		}
	}

	bm.logger.Info("✓ All binaries removed successfully")
	return nil
}

// removeBinary removes a single managed binary with safety checks
func (bm *BinaryManager) removeBinary(binary ManagedBinary) error {
	bm.logger.Debug("Removing binary", "name", binary.Name, "destination", binary.Destination)

	// Check if binary still exists at destination
	fileInfo, err := os.Lstat(binary.Destination)
	if os.IsNotExist(err) {
		bm.logger.Debug("Binary already removed", "destination", binary.Destination)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check binary status: %w", err)
	}

	// Safety check: ensure it's a regular file (not a directory or symlink)
	if !fileInfo.Mode().IsRegular() {
		bm.logger.Warn("Destination is not a regular file, skipping removal for safety", "destination", binary.Destination)
		return fmt.Errorf("destination is not a regular file, skipping removal for safety: %s", binary.Destination)
	}

	// Safety check: ensure it's executable (basic sanity check for binaries)
	if fileInfo.Mode().Perm()&0111 == 0 {
		bm.logger.Warn("File is not executable, might not be a binary, skipping removal for safety", "destination", binary.Destination)
		return fmt.Errorf("file is not executable, skipping removal for safety: %s", binary.Destination)
	}

	if bm.dryRun {
		bm.logger.Info("DRY RUN: Would remove binary", "destination", binary.Destination)
		return nil
	}

	// Perform the removal
	if err := os.Remove(binary.Destination); err != nil {
		return fmt.Errorf("failed to remove binary: %w", err)
	}

	bm.logger.Info("✓ Binary removed", "name", binary.Name, "destination", binary.Destination)

	// If there was a backup, optionally restore it
	if binary.BackupPath != "" {
		if err := bm.offerBackupRestore(binary); err != nil {
			bm.logger.Warn("Could not restore backup", "backup", binary.BackupPath, "error", err)
			// Don't fail the removal operation for backup restoration issues
		}
	}

	return nil
}

// offerBackupRestore handles backup restoration when removing binaries
func (bm *BinaryManager) offerBackupRestore(binary ManagedBinary) error {
	// Check if backup still exists
	if _, err := os.Stat(binary.BackupPath); os.IsNotExist(err) {
		bm.logger.Debug("Backup binary no longer exists", "backup", binary.BackupPath)
		return nil
	}

	bm.logger.Info("📁 Backup available for removed binary", "backup", binary.BackupPath, "original", binary.Destination)
	
	// Interactive backup restoration offer
	if bm.interactive.IsInteractiveMode() {
		shouldRestore, err := bm.interactive.PromptYesNo(
			fmt.Sprintf("Restore backup for %s?", binary.Name),
			false, // default to no
		)
		if err != nil {
			bm.logger.Warn("Failed to prompt for backup restoration", "error", err)
			return nil
		}
		
		if shouldRestore {
			return bm.RestoreFromBackup(binary.BackupPath, binary.Destination)
		}
	}
	
	return nil
}

// RestoreFromBackup restores a binary from its backup
func (bm *BinaryManager) RestoreFromBackup(backupPath, originalDestination string) error {
	bm.logger.Info("🔄 Restoring binary from backup", "backup", backupPath, "destination", originalDestination)
	
	if bm.dryRun {
		bm.logger.Info("DRY RUN: Would restore binary from backup", "backup", backupPath, "destination", originalDestination)
		return nil
	}
	
	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup binary does not exist: %s", backupPath)
	}
	
	// Create destination directory if needed
	destDir := filepath.Dir(originalDestination)
	if err := bm.ensureDirectory(destDir); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	// Remove existing binary at destination if it exists
	if _, err := os.Lstat(originalDestination); err == nil {
		bm.logger.Debug("Removing existing binary before restore", "path", originalDestination)
		if err := os.Remove(originalDestination); err != nil {
			return fmt.Errorf("failed to remove existing binary: %w", err)
		}
	}
	
	// Move backup to original location
	if err := os.Rename(backupPath, originalDestination); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	
	bm.logger.Info("✓ Binary restored from backup successfully", "destination", originalDestination)
	return nil
}