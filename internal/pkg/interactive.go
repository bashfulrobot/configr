package pkg

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

// InteractiveManager handles interactive features like prompts and file diffs
type InteractiveManager struct {
	logger *log.Logger
	reader *bufio.Reader
}

// NewInteractiveManager creates a new InteractiveManager instance
func NewInteractiveManager(logger *log.Logger) *InteractiveManager {
	return &InteractiveManager{
		logger: logger,
		reader: bufio.NewReader(os.Stdin),
	}
}

// ConflictResolution represents the user's choice for conflict resolution
type ConflictResolution int

const (
	ResolutionSkip ConflictResolution = iota
	ResolutionOverwrite
	ResolutionBackup
	ResolutionViewDiff
	ResolutionQuit
)

// FileConflictInfo contains information about a file conflict
type FileConflictInfo struct {
	Name           string
	SourcePath     string
	DestinationPath string
	ExistingInfo   os.FileInfo
	IsSymlink      bool
	BackupEnabled  bool
}

// PromptForConflictResolution prompts the user to resolve a file conflict
func (im *InteractiveManager) PromptForConflictResolution(conflict FileConflictInfo) (ConflictResolution, error) {
	im.logger.Info("📁 File conflict detected", "file", conflict.Name, "destination", conflict.DestinationPath)
	
	// Show file information
	if conflict.IsSymlink {
		target, err := os.Readlink(conflict.DestinationPath)
		if err != nil {
			im.logger.Warn("Could not read symlink target", "error", err)
		} else {
			im.logger.Info("  Current: symlink → %s", target)
		}
	} else {
		im.logger.Info("  Current: regular file (modified %s)", conflict.ExistingInfo.ModTime().Format("2006-01-02 15:04:05"))
	}
	
	im.logger.Info("  New: %s", conflict.SourcePath)
	
	fmt.Print("\nHow would you like to proceed?\n")
	fmt.Print("  [o] Overwrite existing file\n")
	if conflict.BackupEnabled {
		fmt.Print("  [b] Backup existing file and overwrite\n")
	}
	fmt.Print("  [d] Show diff between files\n")
	fmt.Print("  [s] Skip this file\n")
	fmt.Print("  [q] Quit configuration\n")
	fmt.Print("\nChoice: ")

	input, err := im.reader.ReadString('\n')
	if err != nil {
		return ResolutionQuit, fmt.Errorf("failed to read user input: %w", err)
	}

	choice := strings.ToLower(strings.TrimSpace(input))
	switch choice {
	case "o", "overwrite":
		return ResolutionOverwrite, nil
	case "b", "backup":
		if conflict.BackupEnabled {
			return ResolutionBackup, nil
		}
		im.logger.Warn("Backup is not enabled for this file")
		return im.PromptForConflictResolution(conflict) // Ask again
	case "d", "diff":
		return ResolutionViewDiff, nil
	case "s", "skip":
		return ResolutionSkip, nil
	case "q", "quit":
		return ResolutionQuit, nil
	default:
		im.logger.Warn("Invalid choice: %s", choice)
		return im.PromptForConflictResolution(conflict) // Ask again
	}
}

// ShowFileDiff displays a diff between source and destination files
func (im *InteractiveManager) ShowFileDiff(sourcePath, destPath string) error {
	im.logger.Info("📊 Showing diff between files")
	fmt.Printf("\n--- %s\n+++ %s\n", destPath, sourcePath)
	
	// Try to use diff command first
	if err := im.showSystemDiff(destPath, sourcePath); err != nil {
		im.logger.Debug("System diff failed, using built-in diff", "error", err)
		return im.showBuiltinDiff(destPath, sourcePath)
	}
	
	return nil
}

// showSystemDiff uses the system diff command
func (im *InteractiveManager) showSystemDiff(file1, file2 string) error {
	cmd := exec.Command("diff", "-u", file1, file2)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// diff returns exit code 1 when files differ, which is normal
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				return nil // Files differ, but diff succeeded
			}
		}
		return err
	}
	
	return nil
}

// showBuiltinDiff provides a basic built-in diff implementation
func (im *InteractiveManager) showBuiltinDiff(file1, file2 string) error {
	content1, err := os.ReadFile(file1)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", file1, err)
	}
	
	content2, err := os.ReadFile(file2)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", file2, err)
	}
	
	lines1 := strings.Split(string(content1), "\n")
	lines2 := strings.Split(string(content2), "\n")
	
	// Simple line-by-line comparison
	maxLines := len(lines1)
	if len(lines2) > maxLines {
		maxLines = len(lines2)
	}
	
	for i := 0; i < maxLines; i++ {
		var line1, line2 string
		if i < len(lines1) {
			line1 = lines1[i]
		}
		if i < len(lines2) {
			line2 = lines2[i]
		}
		
		if line1 != line2 {
			if line1 != "" {
				fmt.Printf("-%s\n", line1)
			}
			if line2 != "" {
				fmt.Printf("+%s\n", line2)
			}
		}
	}
	
	return nil
}

// PromptYesNo prompts the user for a yes/no question
func (im *InteractiveManager) PromptYesNo(question string, defaultYes bool) (bool, error) {
	defaultStr := "y/N"
	if defaultYes {
		defaultStr = "Y/n"
	}
	
	fmt.Printf("%s [%s]: ", question, defaultStr)
	
	input, err := im.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}
	
	response := strings.ToLower(strings.TrimSpace(input))
	if response == "" {
		return defaultYes, nil
	}
	
	switch response {
	case "y", "yes", "true", "1":
		return true, nil
	case "n", "no", "false", "0":
		return false, nil
	default:
		im.logger.Warn("Please answer yes or no")
		return im.PromptYesNo(question, defaultYes)
	}
}

// PromptForPermissions prompts for file permissions with validation
func (im *InteractiveManager) PromptForPermissions(fileName string, currentMode os.FileMode) (string, error) {
	currentOctal := fmt.Sprintf("%04o", currentMode.Perm())
	
	fmt.Printf("Current permissions for %s: %s (%s)\n", fileName, currentMode.String(), currentOctal)
	fmt.Print("Enter new permissions (octal, e.g., 644, 755) or press Enter to keep current: ")
	
	input, err := im.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read user input: %w", err)
	}
	
	mode := strings.TrimSpace(input)
	if mode == "" {
		return currentOctal, nil
	}
	
	// Validate octal format
	if err := im.validateOctalPermissions(mode); err != nil {
		im.logger.Warn("Invalid permissions: %v", err)
		return im.PromptForPermissions(fileName, currentMode)
	}
	
	return mode, nil
}

// validateOctalPermissions validates that a string is a valid octal permission
func (im *InteractiveManager) validateOctalPermissions(mode string) error {
	if len(mode) < 3 || len(mode) > 4 {
		return fmt.Errorf("permissions must be 3 or 4 digits")
	}
	
	for _, char := range mode {
		if char < '0' || char > '7' {
			return fmt.Errorf("permissions must contain only octal digits (0-7)")
		}
	}
	
	return nil
}

// PromptForOwnership prompts for file ownership
func (im *InteractiveManager) PromptForOwnership(fileName, currentOwner, currentGroup string) (string, string, error) {
	fmt.Printf("Current ownership for %s: %s:%s\n", fileName, currentOwner, currentGroup)
	
	fmt.Print("Enter new owner (username or UID) or press Enter to keep current: ")
	ownerInput, err := im.reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("failed to read owner input: %w", err)
	}
	
	newOwner := strings.TrimSpace(ownerInput)
	if newOwner == "" {
		newOwner = currentOwner
	}
	
	fmt.Print("Enter new group (groupname or GID) or press Enter to keep current: ")
	groupInput, err := im.reader.ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("failed to read group input: %w", err)
	}
	
	newGroup := strings.TrimSpace(groupInput)
	if newGroup == "" {
		newGroup = currentGroup
	}
	
	return newOwner, newGroup, nil
}

// ShowPreviewSummary shows a summary of all changes that will be made
func (im *InteractiveManager) ShowPreviewSummary(files map[string]config.File, conflicts []FileConflictInfo) error {
	fmt.Print("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Print("CONFIGURATION PREVIEW\n")
	fmt.Print(strings.Repeat("=", 60) + "\n\n")
	
	// Show files to be deployed
	if len(files) > 0 {
		fmt.Printf("Files to be deployed (%d):\n", len(files))
		for name, file := range files {
			action := "symlink"
			if file.Copy {
				action = "copy"
			}
			
			fmt.Printf("  • %s: %s → %s (%s)\n", name, file.Source, file.Destination, action)
			
			if file.Owner != "" || file.Group != "" {
				fmt.Printf("    ownership: %s:%s\n", file.Owner, file.Group)
			}
			if file.Mode != "" {
				fmt.Printf("    permissions: %s\n", file.Mode)
			}
			if file.Backup {
				fmt.Print("    backup: enabled\n")
			}
		}
		fmt.Print("\n")
	}
	
	// Show conflicts
	if len(conflicts) > 0 {
		fmt.Printf("Conflicts detected (%d):\n", len(conflicts))
		for _, conflict := range conflicts {
			fmt.Printf("  ⚠ %s: %s (exists)\n", conflict.Name, conflict.DestinationPath)
		}
		fmt.Print("\n")
	}
	
	fmt.Print(strings.Repeat("=", 60) + "\n")
	return nil
}

// WaitForUser waits for user to press Enter to continue
func (im *InteractiveManager) WaitForUser(message string) error {
	if message == "" {
		message = "Press Enter to continue..."
	}
	
	fmt.Print(message)
	_, err := im.reader.ReadString('\n')
	return err
}

// IsInteractiveMode checks if we're running in an interactive terminal
func (im *InteractiveManager) IsInteractiveMode() bool {
	// Check if stdin is a terminal
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	
	return (stat.Mode() & os.ModeCharDevice) != 0
}