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
	logger      *log.Logger
	dryRun      bool
	configDir   string
	interactive *InteractiveManager
}

// BackupInfo contains information about available backups
type BackupInfo struct {
	FileName       string    `json:"file_name"`
	BackupPath     string    `json:"backup_path"`
	OriginalPath   string    `json:"original_path"`
	BackupTime     time.Time `json:"backup_time"`
	BackupSize     int64     `json:"backup_size"`
	OriginalExists bool      `json:"original_exists"`
}

// NewFileManager creates a new FileManager instance
func NewFileManager(logger *log.Logger, dryRun bool, configDir string) *FileManager {
	return &FileManager{
		logger:      logger,
		dryRun:      dryRun,
		configDir:   configDir,
		interactive: NewInteractiveManager(logger),
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

	// Handle existing file (backup if needed, with interactive support)
	backupPath, err := fm.handleExistingFile(name, destPath, sourcePath, file)
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

// handleExistingFile handles existing files at the destination, with interactive support
// Returns the backup path if a backup was created, empty string otherwise
func (fm *FileManager) handleExistingFile(name, destPath, sourcePath string, file config.File) (string, error) {
	if _, err := os.Lstat(destPath); os.IsNotExist(err) {
		// File doesn't exist, nothing to handle
		return "", nil
	}

	// Get file info for interactive prompts
	fileInfo, err := os.Lstat(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	isSymlink := fileInfo.Mode()&os.ModeSymlink != 0

	if fm.dryRun {
		if file.Backup {
			fm.logger.Debug("DRY RUN: Would backup existing file", "path", destPath)
			return fmt.Sprintf("%s.backup.%s", destPath, time.Now().Format("20060102-150405")), nil
		} else {
			fm.logger.Debug("DRY RUN: Would remove existing file", "path", destPath)
			return "", nil
		}
	}

	// Check if it's already a symlink to our source (only relevant for symlink mode)
	if isSymlink && !file.Copy {
		if link, err := os.Readlink(destPath); err == nil {
			// If it's already pointing to the right place, we're done
			if filepath.Clean(link) == filepath.Clean(sourcePath) {
				fm.logger.Debug("File already correctly symlinked", "path", destPath)
				return "", nil
			}
		}
	}

	// Interactive conflict resolution
	if file.Interactive && fm.interactive.IsInteractiveMode() {
		conflict := FileConflictInfo{
			Name:            name,
			SourcePath:      sourcePath,
			DestinationPath: destPath,
			ExistingInfo:    fileInfo,
			IsSymlink:       isSymlink,
			BackupEnabled:   file.Backup,
		}

		for {
			resolution, err := fm.interactive.PromptForConflictResolution(conflict)
			if err != nil {
				return "", fmt.Errorf("failed to get user input: %w", err)
			}

			switch resolution {
			case ResolutionSkip:
				fm.logger.Info("⏭ Skipping file", "name", name)
				return "", fmt.Errorf("file skipped by user: %s", name)
			case ResolutionQuit:
				return "", fmt.Errorf("operation cancelled by user")
			case ResolutionViewDiff:
				if err := fm.interactive.ShowFileDiff(sourcePath, destPath); err != nil {
					fm.logger.Warn("Failed to show diff", "error", err)
				}
				continue // Ask again
			case ResolutionOverwrite:
				break // Continue with overwrite
			case ResolutionBackup:
				// Force backup for this file
				break // Continue with backup
			}
			break
		}
	}

	// Determine if we should backup based on config or user choice
	shouldBackup := file.Backup
	
	if shouldBackup {
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
	if fm.dryRun {
		fm.logger.Debug("DRY RUN: Would set file attributes", "path", destPath, "mode", file.Mode, "owner", file.Owner, "group", file.Group)
		return nil
	}
	
	// Get current file info for interactive prompts
	fileInfo, err := os.Lstat(destPath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	owner := file.Owner
	group := file.Group
	mode := file.Mode

	// Interactive permission prompting
	if file.PromptPermissions && fm.interactive.IsInteractiveMode() {
		if promptedMode, err := fm.interactive.PromptForPermissions(destPath, fileInfo.Mode()); err != nil {
			fm.logger.Warn("Failed to prompt for permissions", "error", err)
		} else {
			mode = promptedMode
		}
	}

	// Interactive ownership prompting
	if file.PromptOwnership && fm.interactive.IsInteractiveMode() {
		// Get current ownership info
		currentOwner, currentGroup := fm.getCurrentOwnership(destPath)
		
		if promptedOwner, promptedGroup, err := fm.interactive.PromptForOwnership(destPath, currentOwner, currentGroup); err != nil {
			fm.logger.Warn("Failed to prompt for ownership", "error", err)
		} else {
			owner = promptedOwner
			group = promptedGroup
		}
	}

	// Set ownership if specified or prompted
	if owner != "" || group != "" {
		if err := fm.setOwnership(destPath, owner, group); err != nil {
			return fmt.Errorf("failed to set ownership: %w", err)
		}
	}

	// Set permissions if specified or prompted
	if mode != "" {
		if err := fm.setPermissions(destPath, mode); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	return nil
}

// getCurrentOwnership gets the current owner and group names for a file
func (fm *FileManager) getCurrentOwnership(path string) (string, string) {
	_, err := os.Lstat(path)
	if err != nil {
		return "unknown", "unknown"
	}

	// This is a simplified implementation - in practice you'd want to
	// resolve UIDs/GIDs to names using os/user package
	return "current", "current"
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
	
	// Interactive backup restoration offer
	if fm.interactive.IsInteractiveMode() {
		shouldRestore, err := fm.interactive.PromptYesNo(
			fmt.Sprintf("Restore backup for %s?", file.Name),
			false, // default to no
		)
		if err != nil {
			fm.logger.Warn("Failed to prompt for backup restoration", "error", err)
			return nil
		}
		
		if shouldRestore {
			return fm.RestoreFromBackup(file.BackupPath, file.Destination)
		}
	}
	
	return nil
}

// RestoreFromBackup restores a file from its backup
func (fm *FileManager) RestoreFromBackup(backupPath, originalDestination string) error {
	fm.logger.Info("🔄 Restoring file from backup", "backup", backupPath, "destination", originalDestination)
	
	if fm.dryRun {
		fm.logger.Info("DRY RUN: Would restore file from backup", "backup", backupPath, "destination", originalDestination)
		return nil
	}
	
	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}
	
	// Create destination directory if needed
	destDir := filepath.Dir(originalDestination)
	if err := fm.ensureDirectory(destDir); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	// Remove existing file at destination if it exists
	if _, err := os.Lstat(originalDestination); err == nil {
		fm.logger.Debug("Removing existing file before restore", "path", originalDestination)
		if err := os.Remove(originalDestination); err != nil {
			return fmt.Errorf("failed to remove existing file: %w", err)
		}
	}
	
	// Move backup to original location
	if err := os.Rename(backupPath, originalDestination); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}
	
	fm.logger.Info("✓ File restored from backup successfully", "destination", originalDestination)
	return nil
}

// RestoreAllBackups restores all available backups for managed files
func (fm *FileManager) RestoreAllBackups(managedFiles []ManagedFile) error {
	var restored, failed int
	
	for _, file := range managedFiles {
		if file.BackupPath == "" {
			continue // No backup for this file
		}
		
		// Check if backup still exists
		if _, err := os.Stat(file.BackupPath); os.IsNotExist(err) {
			fm.logger.Debug("Backup no longer exists", "backup", file.BackupPath)
			continue
		}
		
		// Check if original file exists (don't restore if file is still there)
		if _, err := os.Lstat(file.Destination); err == nil {
			fm.logger.Debug("Original file still exists, skipping restore", "destination", file.Destination)
			continue
		}
		
		if err := fm.RestoreFromBackup(file.BackupPath, file.Destination); err != nil {
			fm.logger.Error("Failed to restore backup", "file", file.Name, "backup", file.BackupPath, "error", err)
			failed++
		} else {
			restored++
		}
	}
	
	if restored > 0 {
		fm.logger.Info("✓ Backup restoration completed", "restored", restored, "failed", failed)
	}
	
	if failed > 0 {
		return fmt.Errorf("failed to restore %d backup(s)", failed)
	}
	
	return nil
}

// ListBackups returns information about available backups
func (fm *FileManager) ListBackups(managedFiles []ManagedFile) []BackupInfo {
	var backups []BackupInfo
	
	for _, file := range managedFiles {
		if file.BackupPath == "" {
			continue
		}
		
		backupStat, err := os.Stat(file.BackupPath)
		if os.IsNotExist(err) {
			continue // Backup doesn't exist
		}
		
		// Check if original file exists
		originalExists := true
		if _, err := os.Lstat(file.Destination); os.IsNotExist(err) {
			originalExists = false
		}
		
		backup := BackupInfo{
			FileName:       file.Name,
			BackupPath:     file.BackupPath,
			OriginalPath:   file.Destination,
			BackupTime:     backupStat.ModTime(),
			BackupSize:     backupStat.Size(),
			OriginalExists: originalExists,
		}
		
		backups = append(backups, backup)
	}
	
	return backups
}

// CleanupExpiredBackups removes backups older than the specified duration
func (fm *FileManager) CleanupExpiredBackups(managedFiles []ManagedFile, maxAge time.Duration) error {
	var cleaned, failed int
	cutoffTime := time.Now().Add(-maxAge)
	
	fm.logger.Info("Cleaning up expired backups", "max_age", maxAge, "cutoff", cutoffTime.Format("2006-01-02 15:04:05"))
	
	for _, file := range managedFiles {
		if file.BackupPath == "" {
			continue
		}
		
		backupStat, err := os.Stat(file.BackupPath)
		if os.IsNotExist(err) {
			continue // Backup doesn't exist
		}
		if err != nil {
			fm.logger.Warn("Could not stat backup file", "backup", file.BackupPath, "error", err)
			continue
		}
		
		// Check if backup is older than cutoff
		if backupStat.ModTime().Before(cutoffTime) {
			if fm.dryRun {
				fm.logger.Info("DRY RUN: Would remove expired backup", "backup", file.BackupPath, "age", time.Since(backupStat.ModTime()))
				cleaned++
				continue
			}
			
			fm.logger.Debug("Removing expired backup", "backup", file.BackupPath, "age", time.Since(backupStat.ModTime()))
			if err := os.Remove(file.BackupPath); err != nil {
				fm.logger.Error("Failed to remove expired backup", "backup", file.BackupPath, "error", err)
				failed++
			} else {
				cleaned++
			}
		}
	}
	
	if cleaned > 0 || failed > 0 {
		fm.logger.Info("✓ Backup cleanup completed", "cleaned", cleaned, "failed", failed)
	}
	
	if failed > 0 {
		return fmt.Errorf("failed to clean %d backup(s)", failed)
	}
	
	return nil
}

// FindOrphanedBackups finds backup files that are no longer tracked in state
func (fm *FileManager) FindOrphanedBackups(managedFiles []ManagedFile) ([]string, error) {
	var orphanedBackups []string
	
	// Get all backup paths from managed files
	knownBackups := make(map[string]bool)
	for _, file := range managedFiles {
		if file.BackupPath != "" {
			knownBackups[file.BackupPath] = true
		}
	}
	
	// Search common backup directories for orphaned backups
	searchDirs := []string{
		"~/.config",
		"~/.local",
		"~/",
	}
	
	for _, dir := range searchDirs {
		// Expand home directory
		if dir[0] == '~' {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				continue
			}
			dir = filepath.Join(homeDir, dir[1:])
		}
		
		orphaned, err := fm.searchForOrphanedBackups(dir, knownBackups)
		if err != nil {
			fm.logger.Debug("Could not search for orphaned backups", "dir", dir, "error", err)
			continue
		}
		
		orphanedBackups = append(orphanedBackups, orphaned...)
	}
	
	return orphanedBackups, nil
}

// searchForOrphanedBackups recursively searches for backup files in a directory
func (fm *FileManager) searchForOrphanedBackups(dir string, knownBackups map[string]bool) ([]string, error) {
	var orphaned []string
	
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip directories we can't access
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Check if this looks like a configr backup file
		if fm.isLikelyConfigrBackup(path, info) && !knownBackups[path] {
			orphaned = append(orphaned, path)
		}
		
		return nil
	})
	
	return orphaned, err
}

// isLikelyConfigrBackup checks if a file appears to be a configr backup
func (fm *FileManager) isLikelyConfigrBackup(path string, info os.FileInfo) bool {
	// Check for backup naming pattern: *.backup.YYYYMMDD-HHMMSS
	filename := info.Name()
	
	// Must contain ".backup." and end with timestamp pattern
	if !strings.Contains(filename, ".backup.") {
		return false
	}
	
	// Extract timestamp part
	parts := strings.Split(filename, ".backup.")
	if len(parts) != 2 {
		return false
	}
	
	timestamp := parts[1]
	// Check if it matches our timestamp format: YYYYMMDD-HHMMSS
	if len(timestamp) != 15 {
		return false
	}
	
	// Basic validation of timestamp format
	if timestamp[8] != '-' {
		return false
	}
	
	return true
}

// GetBackupStatistics returns comprehensive backup statistics
func (fm *FileManager) GetBackupStatistics(managedFiles []ManagedFile) (*BackupStatistics, error) {
	stats := &BackupStatistics{
		TotalBackups:    0,
		TotalSize:       0,
		OldestBackup:    time.Now(),
		NewestBackup:    time.Time{},
		BackupsByAge:    make(map[string]int),
		RestorableCount: 0,
	}
	
	now := time.Now()
	
	for _, file := range managedFiles {
		if file.BackupPath == "" {
			continue
		}
		
		backupInfo, err := os.Stat(file.BackupPath)
		if os.IsNotExist(err) {
			continue // Backup doesn't exist
		}
		if err != nil {
			continue // Skip files we can't stat
		}
		
		stats.TotalBackups++
		stats.TotalSize += backupInfo.Size()
		
		// Track oldest and newest
		if backupInfo.ModTime().Before(stats.OldestBackup) {
			stats.OldestBackup = backupInfo.ModTime()
		}
		if backupInfo.ModTime().After(stats.NewestBackup) {
			stats.NewestBackup = backupInfo.ModTime()
		}
		
		// Check if restorable (original doesn't exist)
		if _, err := os.Lstat(file.Destination); os.IsNotExist(err) {
			stats.RestorableCount++
		}
		
		// Categorize by age
		age := now.Sub(backupInfo.ModTime())
		switch {
		case age < 24*time.Hour:
			stats.BackupsByAge["< 1 day"]++
		case age < 7*24*time.Hour:
			stats.BackupsByAge["< 1 week"]++
		case age < 30*24*time.Hour:
			stats.BackupsByAge["< 1 month"]++
		case age < 365*24*time.Hour:
			stats.BackupsByAge["< 1 year"]++
		default:
			stats.BackupsByAge["> 1 year"]++
		}
	}
	
	if stats.TotalBackups == 0 {
		stats.OldestBackup = time.Time{}
	}
	
	return stats, nil
}

// CleanupOrphanedBackups removes backup files that are no longer tracked
func (fm *FileManager) CleanupOrphanedBackups(managedFiles []ManagedFile) error {
	orphaned, err := fm.FindOrphanedBackups(managedFiles)
	if err != nil {
		return fmt.Errorf("failed to find orphaned backups: %w", err)
	}
	
	if len(orphaned) == 0 {
		fm.logger.Info("No orphaned backups found")
		return nil
	}
	
	fm.logger.Info("Found orphaned backups", "count", len(orphaned))
	
	var removed, failed int
	for _, backupPath := range orphaned {
		if fm.dryRun {
			fm.logger.Info("DRY RUN: Would remove orphaned backup", "path", backupPath)
			removed++
			continue
		}
		
		fm.logger.Debug("Removing orphaned backup", "path", backupPath)
		if err := os.Remove(backupPath); err != nil {
			fm.logger.Error("Failed to remove orphaned backup", "path", backupPath, "error", err)
			failed++
		} else {
			removed++
		}
	}
	
	fm.logger.Info("✓ Orphaned backup cleanup completed", "removed", removed, "failed", failed)
	
	if failed > 0 {
		return fmt.Errorf("failed to remove %d orphaned backup(s)", failed)
	}
	
	return nil
}

// BackupStatistics contains comprehensive backup information
type BackupStatistics struct {
	TotalBackups    int               `json:"total_backups"`
	TotalSize       int64             `json:"total_size"`
	OldestBackup    time.Time         `json:"oldest_backup"`
	NewestBackup    time.Time         `json:"newest_backup"`
	BackupsByAge    map[string]int    `json:"backups_by_age"`
	RestorableCount int               `json:"restorable_count"`
}

// ApplyBackupPolicy enforces the configured backup policy
func (fm *FileManager) ApplyBackupPolicy(managedFiles []ManagedFile, policy config.BackupPolicy) error {
	if !policy.AutoCleanup {
		fm.logger.Debug("Backup policy auto-cleanup disabled")
		return nil
	}

	fm.logger.Info("Applying backup policy", "max_age", policy.MaxAge, "max_count", policy.MaxCount, "cleanup_orphaned", policy.CleanupOrphaned)

	var errors []error

	// Clean up orphaned backups if enabled
	if policy.CleanupOrphaned {
		if err := fm.CleanupOrphanedBackups(managedFiles); err != nil {
			errors = append(errors, fmt.Errorf("orphaned backup cleanup failed: %w", err))
		}
	}

	// Apply age-based cleanup
	if policy.MaxAge != "" {
		maxAge, err := fm.parseBackupAge(policy.MaxAge)
		if err != nil {
			errors = append(errors, fmt.Errorf("invalid max_age in backup policy: %w", err))
		} else {
			if err := fm.CleanupExpiredBackups(managedFiles, maxAge); err != nil {
				errors = append(errors, fmt.Errorf("age-based cleanup failed: %w", err))
			}
		}
	}

	// Apply count-based cleanup
	if policy.MaxCount > 0 {
		if err := fm.cleanupByCount(managedFiles, policy.MaxCount, policy.PreserveRecent); err != nil {
			errors = append(errors, fmt.Errorf("count-based cleanup failed: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("backup policy enforcement failed: %v", errors)
	}

	return nil
}

// cleanupByCount removes excess backups keeping only the most recent ones
func (fm *FileManager) cleanupByCount(managedFiles []ManagedFile, maxCount, preserveRecent int) error {
	// Group backups by original file
	fileBackups := make(map[string][]BackupInfo)
	
	for _, file := range managedFiles {
		if file.BackupPath == "" {
			continue
		}
		
		backupStat, err := os.Stat(file.BackupPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			continue
		}
		
		backup := BackupInfo{
			FileName:     file.Name,
			BackupPath:   file.BackupPath,
			OriginalPath: file.Destination,
			BackupTime:   backupStat.ModTime(),
			BackupSize:   backupStat.Size(),
		}
		
		fileBackups[file.Destination] = append(fileBackups[file.Destination], backup)
	}
	
	var cleaned, failed int
	
	// Process each file's backups
	for _, backups := range fileBackups {
		if len(backups) <= maxCount {
			continue // No cleanup needed
		}
		
		// Sort by backup time (newest first)
		for i := 0; i < len(backups)-1; i++ {
			for j := i + 1; j < len(backups); j++ {
				if backups[i].BackupTime.Before(backups[j].BackupTime) {
					backups[i], backups[j] = backups[j], backups[i]
				}
			}
		}
		
		// Determine how many to preserve
		preserveCount := maxCount
		if preserveRecent > preserveCount {
			preserveCount = preserveRecent
		}
		
		// Remove excess backups
		for i := preserveCount; i < len(backups); i++ {
			backup := backups[i]
			
			if fm.dryRun {
				fm.logger.Info("DRY RUN: Would remove excess backup", 
					"file", backup.FileName, 
					"backup", backup.BackupPath,
					"age", time.Since(backup.BackupTime))
				cleaned++
				continue
			}
			
			fm.logger.Debug("Removing excess backup", 
				"file", backup.FileName, 
				"backup", backup.BackupPath, 
				"age", time.Since(backup.BackupTime))
				
			if err := os.Remove(backup.BackupPath); err != nil {
				fm.logger.Error("Failed to remove excess backup", "backup", backup.BackupPath, "error", err)
				failed++
			} else {
				cleaned++
			}
		}
	}
	
	if cleaned > 0 || failed > 0 {
		fm.logger.Info("✓ Count-based backup cleanup completed", "cleaned", cleaned, "failed", failed)
	}
	
	if failed > 0 {
		return fmt.Errorf("failed to clean %d backup(s)", failed)
	}
	
	return nil
}

// parseBackupAge parses duration strings for backup policies
func (fm *FileManager) parseBackupAge(ageStr string) (time.Duration, error) {
	// Handle common suffixes
	switch {
	case len(ageStr) == 0:
		return 0, fmt.Errorf("empty duration")
	case ageStr[len(ageStr)-1] == 'd':
		// Days - convert to hours
		daysStr := ageStr[:len(ageStr)-1]
		days, err := time.ParseDuration(daysStr + "h")
		if err != nil {
			return 0, err
		}
		return days * 24, nil
	default:
		// Standard Go duration parsing
		return time.ParseDuration(ageStr)
	}
}