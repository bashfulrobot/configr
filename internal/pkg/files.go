package pkg

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

// FileManager handles file operations including deployment, backup, and permission management
type FileManager struct {
	logger   *log.Logger
	dryRun   bool
	configDir string
}

// NewFileManager creates a new FileManager instance
func NewFileManager(logger *log.Logger, dryRun bool, configDir string) *FileManager {
	return &FileManager{
		logger:    logger,
		dryRun:    dryRun,
		configDir: configDir,
	}
}

// DeployFiles processes all files in the configuration and returns deployed file info
func (fm *FileManager) DeployFiles(files map[string]config.File) ([]ManagedFile, error) {
	if len(files) == 0 {
		fm.logger.Debug("No files to deploy")
		return []ManagedFile{}, nil
	}

	fm.logger.Info("Processing file deployments", "count", len(files))

	var deployedFiles []ManagedFile
	for name, file := range files {
		managedFile, err := fm.deployFile(name, file)
		if err != nil {
			return nil, fmt.Errorf("failed to deploy file '%s': %w", name, err)
		}
		deployedFiles = append(deployedFiles, managedFile)
	}

	fm.logger.Info("✓ All files deployed successfully")
	return deployedFiles, nil
}

// deployFile handles the deployment of a single file and returns file info
func (fm *FileManager) deployFile(name string, file config.File) (ManagedFile, error) {
	fm.logger.Debug("Deploying file", "name", name, "source", file.Source, "destination", file.Destination)

	// Resolve source path
	sourcePath, err := fm.resolveSourcePath(file.Source)
	if err != nil {
		return ManagedFile{}, fmt.Errorf("failed to resolve source path: %w", err)
	}

	// Resolve destination path
	destPath, err := fm.resolveDestinationPath(file.Destination)
	if err != nil {
		return ManagedFile{}, fmt.Errorf("failed to resolve destination path: %w", err)
	}

	// Check if source file exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return ManagedFile{}, fmt.Errorf("source file does not exist: %s", sourcePath)
	}

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(destPath)
	if err := fm.ensureDirectory(destDir); err != nil {
		return ManagedFile{}, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Handle existing file (backup if needed)
	backupPath, err := fm.handleExistingFile(destPath, file.Backup)
	if err != nil {
		return ManagedFile{}, fmt.Errorf("failed to handle existing file: %w", err)
	}

	// Deploy file (either copy or symlink)
	isSymlink := !file.Copy
	if file.Copy {
		if err := fm.copyFile(sourcePath, destPath); err != nil {
			return ManagedFile{}, fmt.Errorf("failed to copy file: %w", err)
		}
	} else {
		if err := fm.createSymlink(sourcePath, destPath); err != nil {
			return ManagedFile{}, fmt.Errorf("failed to create symlink: %w", err)
		}
	}

	// Set ownership and permissions if specified
	if err := fm.setFileAttributes(destPath, file); err != nil {
		return ManagedFile{}, fmt.Errorf("failed to set file attributes: %w", err)
	}

	fm.logger.Info("✓ File deployed", "name", name, "destination", destPath)
	
	// Return managed file info
	return ManagedFile{
		Name:        name,
		Destination: destPath,
		IsSymlink:   isSymlink,
		BackupPath:  backupPath,
	}, nil
}

// resolveSourcePath resolves the source file path, handling relative paths
func (fm *FileManager) resolveSourcePath(source string) (string, error) {
	if filepath.IsAbs(source) {
		return source, nil
	}

	// Relative to config directory
	return filepath.Join(fm.configDir, source), nil
}

// resolveDestinationPath resolves the destination path, handling ~ expansion
func (fm *FileManager) resolveDestinationPath(destination string) (string, error) {
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
func (fm *FileManager) ensureDirectory(dir string) error {
	if fm.dryRun {
		fm.logger.Debug("DRY RUN: Would create directory", "path", dir)
		return nil
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		fm.logger.Debug("Creating directory", "path", dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// handleExistingFile handles existing files at the destination, optionally creating backups
// Returns the backup path if a backup was created, empty string otherwise
func (fm *FileManager) handleExistingFile(destPath string, backup bool) (string, error) {
	if _, err := os.Lstat(destPath); os.IsNotExist(err) {
		// File doesn't exist, nothing to handle
		return "", nil
	}

	if fm.dryRun {
		if backup {
			fm.logger.Debug("DRY RUN: Would backup existing file", "path", destPath)
			return fmt.Sprintf("%s.backup.%s", destPath, time.Now().Format("20060102-150405")), nil
		} else {
			fm.logger.Debug("DRY RUN: Would remove existing file", "path", destPath)
			return "", nil
		}
	}

	// Check if it's already a symlink to our source (only relevant for symlink mode)
	if link, err := os.Readlink(destPath); err == nil {
		sourcePath, _ := fm.resolveSourcePath("")
		sourceDir := filepath.Dir(sourcePath)
		
		// If it's already pointing to the right place, we're done
		if filepath.Clean(link) == filepath.Clean(filepath.Join(sourceDir, filepath.Base(link))) {
			fm.logger.Debug("File already correctly symlinked", "path", destPath)
			return "", nil
		}
	}

	if backup {
		backupPath := fmt.Sprintf("%s.backup.%s", destPath, time.Now().Format("20060102-150405"))
		fm.logger.Info("⚠ Backing up existing file", "from", destPath, "to", backupPath)
		
		if err := os.Rename(destPath, backupPath); err != nil {
			return "", fmt.Errorf("failed to backup file: %w", err)
		}
		return backupPath, nil
	} else {
		fm.logger.Info("⚠ Removing existing file", "path", destPath)
		if err := os.Remove(destPath); err != nil {
			return "", fmt.Errorf("failed to remove existing file: %w", err)
		}
		return "", nil
	}
}

// createSymlink creates a symlink from source to destination
func (fm *FileManager) createSymlink(sourcePath, destPath string) error {
	if fm.dryRun {
		fm.logger.Debug("DRY RUN: Would create symlink", "from", sourcePath, "to", destPath)
		return nil
	}

	fm.logger.Debug("Creating symlink", "from", sourcePath, "to", destPath)
	
	if err := os.Symlink(sourcePath, destPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// copyFile copies a file from source to destination
func (fm *FileManager) copyFile(sourcePath, destPath string) error {
	if fm.dryRun {
		fm.logger.Debug("DRY RUN: Would copy file", "from", sourcePath, "to", destPath)
		return nil
	}

	fm.logger.Debug("Copying file", "from", sourcePath, "to", destPath)

	// Open source file
	src, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy file contents
	_, err = dst.ReadFrom(src)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Sync to ensure data is written
	if err := dst.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// setFileAttributes sets ownership and permissions on the file if specified
func (fm *FileManager) setFileAttributes(destPath string, file config.File) error {
	// Set ownership if specified
	if file.Owner != "" || file.Group != "" {
		if err := fm.setOwnership(destPath, file.Owner, file.Group); err != nil {
			return fmt.Errorf("failed to set ownership: %w", err)
		}
	}

	// Set permissions if specified
	if file.Mode != "" {
		if err := fm.setPermissions(destPath, file.Mode); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	return nil
}

// setOwnership sets the owner and group of a file
func (fm *FileManager) setOwnership(path, owner, group string) error {
	if fm.dryRun {
		fm.logger.Debug("DRY RUN: Would set ownership", "path", path, "owner", owner, "group", group)
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

	fm.logger.Debug("Setting ownership", "path", path, "uid", uid, "gid", gid)
	
	if err := os.Lchown(path, uid, gid); err != nil {
		return fmt.Errorf("failed to change ownership: %w", err)
	}

	return nil
}

// setPermissions sets the file mode/permissions
func (fm *FileManager) setPermissions(path, mode string) error {
	if fm.dryRun {
		fm.logger.Debug("DRY RUN: Would set permissions", "path", path, "mode", mode)
		return nil
	}

	// Parse octal mode
	modeInt, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return fmt.Errorf("invalid file mode '%s': %w", mode, err)
	}

	fm.logger.Debug("Setting permissions", "path", path, "mode", mode)
	
	if err := os.Chmod(path, os.FileMode(modeInt)); err != nil {
		return fmt.Errorf("failed to change permissions: %w", err)
	}

	return nil
}

// RemoveFiles removes files that are no longer in the configuration
func (fm *FileManager) RemoveFiles(oldFiles, newFiles map[string]config.File) error {
	// Find files that were removed from config
	removedFiles := make(map[string]config.File)
	for name, file := range oldFiles {
		if _, exists := newFiles[name]; !exists {
			removedFiles[name] = file
		}
	}

	if len(removedFiles) == 0 {
		fm.logger.Debug("No files to remove")
		return nil
	}

	fm.logger.Info("Processing file removals", "count", len(removedFiles))

	for name, file := range removedFiles {
		if err := fm.removeFile(name, file); err != nil {
			fm.logger.Error("Failed to remove file", "name", name, "error", err)
			// Continue with other files rather than failing completely
		}
	}

	return nil
}

// removeFile removes a single file and optionally restores backup
func (fm *FileManager) removeFile(name string, file config.File) error {
	destPath, err := fm.resolveDestinationPath(file.Destination)
	if err != nil {
		return fmt.Errorf("failed to resolve destination path: %w", err)
	}

	if fm.dryRun {
		fm.logger.Debug("DRY RUN: Would remove file", "name", name, "path", destPath)
		return nil
	}

	// Check if file exists 
	if info, err := os.Lstat(destPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			fm.logger.Info("✗ Removing symlink", "name", name, "path", destPath)
			if err := os.Remove(destPath); err != nil {
				return fmt.Errorf("failed to remove symlink: %w", err)
			}
		} else if file.Copy {
			// This was a copied file, safe to remove if it was managed by configr
			fm.logger.Info("✗ Removing copied file", "name", name, "path", destPath)
			if err := os.Remove(destPath); err != nil {
				return fmt.Errorf("failed to remove copied file: %w", err)
			}
		} else {
			fm.logger.Warn("File exists but is not a symlink or copied file, skipping removal", "path", destPath)
		}

		// Try to restore backup if it exists (for both symlinks and copied files)
		if err := fm.restoreBackup(destPath); err != nil {
			fm.logger.Warn("Failed to restore backup", "path", destPath, "error", err)
		}
	}

	return nil
}

// restoreBackup restores the most recent backup file if it exists
func (fm *FileManager) restoreBackup(originalPath string) error {
	backupPattern := originalPath + ".backup.*"
	matches, err := filepath.Glob(backupPattern)
	if err != nil || len(matches) == 0 {
		fm.logger.Debug("No backup files found", "pattern", backupPattern)
		return nil
	}

	// Find the most recent backup (assumes timestamp format)
	var mostRecent string
	var mostRecentTime time.Time
	
	for _, match := range matches {
		parts := strings.Split(match, ".backup.")
		if len(parts) < 2 {
			continue
		}
		
		timeStr := parts[len(parts)-1]
		if t, err := time.Parse("20060102-150405", timeStr); err == nil {
			if mostRecent == "" || t.After(mostRecentTime) {
				mostRecent = match
				mostRecentTime = t
			}
		}
	}

	if mostRecent != "" {
		fm.logger.Info("↶ Restoring backup", "from", mostRecent, "to", originalPath)
		if err := os.Rename(mostRecent, originalPath); err != nil {
			return fmt.Errorf("failed to restore backup: %w", err)
		}
	}

	return nil
}

// ValidateFilePermissions checks if we have the necessary permissions to deploy files
func (fm *FileManager) ValidateFilePermissions(files map[string]config.File) error {
	for name, file := range files {
		destPath, err := fm.resolveDestinationPath(file.Destination)
		if err != nil {
			return fmt.Errorf("failed to resolve destination for '%s': %w", name, err)
		}

		destDir := filepath.Dir(destPath)
		
		// Check if we can write to the destination directory
		if err := fm.checkWritePermission(destDir); err != nil {
			return fmt.Errorf("insufficient permissions for '%s': %w", name, err)
		}

		// If setting ownership, check if we're root or have appropriate capabilities
		if file.Owner != "" || file.Group != "" {
			if os.Geteuid() != 0 {
				fm.logger.Warn("Setting ownership requires root privileges", "file", name)
			}
		}
	}

	return nil
}

// checkWritePermission checks if we can write to a directory
func (fm *FileManager) checkWritePermission(dir string) error {
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
				return fm.testWriteAccess(parent)
			}
			testDir = parent
		}
	}

	// Directory exists, check write permission
	return fm.testWriteAccess(dir)
}

// testWriteAccess tests write access by attempting to create a temporary file
func (fm *FileManager) testWriteAccess(dir string) error {
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

// RemoveFiles removes files that are no longer in the configuration
func (fm *FileManager) RemoveFiles(filesToRemove []ManagedFile) error {
	if len(filesToRemove) == 0 {
		fm.logger.Debug("No files to remove")
		return nil
	}

	fm.logger.Info("Removing files no longer in configuration", "count", len(filesToRemove))

	for _, file := range filesToRemove {
		if err := fm.removeFile(file); err != nil {
			fm.logger.Error("Failed to remove file", "name", file.Name, "destination", file.Destination, "error", err)
			return fmt.Errorf("failed to remove file '%s': %w", file.Name, err)
		}
	}

	fm.logger.Info("✓ All files removed successfully")
	return nil
}

// removeFile removes a single managed file with safety checks
func (fm *FileManager) removeFile(file ManagedFile) error {
	fm.logger.Debug("Removing file", "name", file.Name, "destination", file.Destination, "is_symlink", file.IsSymlink)

	// Check if file still exists at destination
	fileInfo, err := os.Lstat(file.Destination)
	if os.IsNotExist(err) {
		fm.logger.Debug("File already removed", "destination", file.Destination)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to check file status: %w", err)
	}

	// Safety check: verify file type matches what we expect
	isCurrentlySymlink := fileInfo.Mode()&os.ModeSymlink != 0
	if isCurrentlySymlink != file.IsSymlink {
		fm.logger.Warn("File type changed since deployment", 
			"destination", file.Destination, 
			"expected_symlink", file.IsSymlink, 
			"actual_symlink", isCurrentlySymlink)
		
		// For safety, don't remove files that changed type
		return fmt.Errorf("file type changed since deployment, skipping removal for safety: %s", file.Destination)
	}

	// Additional safety check for symlinks - verify they point to expected location
	if file.IsSymlink {
		if err := fm.verifySymlinkSafety(file.Destination); err != nil {
			fm.logger.Warn("Symlink safety check failed", "destination", file.Destination, "error", err)
			return fmt.Errorf("symlink safety check failed, skipping removal: %w", err)
		}
	}

	// For copied files, check if user has modified the file
	if !file.IsSymlink {
		if modified, err := fm.isFileModifiedByUser(file.Destination, file); err != nil {
			fm.logger.Warn("Could not check if file was modified", "destination", file.Destination, "error", err)
			// Continue with removal but log the warning
		} else if modified {
			fm.logger.Warn("File appears to be modified by user, skipping removal for safety", "destination", file.Destination)
			return fmt.Errorf("file appears modified by user, skipping removal for safety: %s", file.Destination)
		}
	}

	if fm.dryRun {
		fm.logger.Info("DRY RUN: Would remove file", "destination", file.Destination)
		return nil
	}

	// Perform the removal
	if err := os.Remove(file.Destination); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	fm.logger.Info("✓ File removed", "name", file.Name, "destination", file.Destination)

	// If there was a backup, optionally restore it
	if file.BackupPath != "" {
		if err := fm.offerBackupRestore(file); err != nil {
			fm.logger.Warn("Could not restore backup", "backup", file.BackupPath, "error", err)
			// Don't fail the removal operation for backup restoration issues
		}
	}

	return nil
}

// verifySymlinkSafety checks if a symlink is safe to remove
func (fm *FileManager) verifySymlinkSafety(symlinkPath string) error {
	target, err := os.Readlink(symlinkPath)
	if err != nil {
		return fmt.Errorf("failed to read symlink: %w", err)
	}

	// Basic safety check - ensure it's not pointing to system files
	if strings.HasPrefix(target, "/etc/") || strings.HasPrefix(target, "/usr/") || strings.HasPrefix(target, "/bin/") {
		return fmt.Errorf("symlink points to system directory, unsafe to remove: %s", target)
	}

	return nil
}

// isFileModifiedByUser attempts to detect if a copied file was modified by the user
func (fm *FileManager) isFileModifiedByUser(filePath string, file ManagedFile) (bool, error) {
	// This is a basic heuristic - in a full implementation, you might:
	// 1. Store checksums of deployed files
	// 2. Compare modification times
	// 3. Use more sophisticated change detection
	
	// For now, we'll be conservative and assume files might be modified
	// A more sophisticated implementation would store file hashes in the state
	
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}

	// If file is very recent (deployed in last few minutes), probably not modified
	if time.Since(fileInfo.ModTime()) < 5*time.Minute {
		return false, nil
	}

	// For now, assume older copied files might have been modified
	// This is conservative but safe
	return true, nil
}

// offerBackupRestore handles backup restoration when removing files
func (fm *FileManager) offerBackupRestore(file ManagedFile) error {
	// Check if backup still exists
	if _, err := os.Stat(file.BackupPath); os.IsNotExist(err) {
		fm.logger.Debug("Backup file no longer exists", "backup", file.BackupPath)
		return nil
	}

	fm.logger.Info("📁 Backup available for removed file", "backup", file.BackupPath, "original", file.Destination)
	// In a full implementation, you might:
	// 1. Automatically restore backups
	// 2. Prompt user for restoration
	// 3. Move backups to a cleanup directory
	
	// For now, just log that backups are available
	return nil
}