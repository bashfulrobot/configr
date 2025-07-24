package pkg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

func TestFileManager_resolveSourcePath(t *testing.T) {
	fm := &FileManager{configDir: "/home/user/config"}

	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name:     "absolute path",
			source:   "/absolute/path/file.txt",
			expected: "/absolute/path/file.txt",
		},
		{
			name:     "relative path",
			source:   "dotfiles/vimrc",
			expected: "/home/user/config/dotfiles/vimrc",
		},
		{
			name:     "current directory file",
			source:   "config.yaml",
			expected: "/home/user/config/config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fm.resolveSourcePath(tt.source)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFileManager_resolveDestinationPath(t *testing.T) {
	fm := &FileManager{}

	// Get current user's home directory for testing
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name     string
		dest     string
		expected string
	}{
		{
			name:     "absolute path",
			dest:     "/etc/config/file.conf",
			expected: "/etc/config/file.conf",
		},
		{
			name:     "home directory expansion",
			dest:     "~/.vimrc",
			expected: filepath.Join(home, ".vimrc"),
		},
		{
			name:     "home directory root",
			dest:     "~/",
			expected: filepath.Join(home, ""),
		},
		{
			name:     "relative to home",
			dest:     "~/Documents/file.txt",
			expected: filepath.Join(home, "Documents/file.txt"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := fm.resolveDestinationPath(tt.dest)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFileManager_DeployFiles_DryRun(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a logger for testing
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel) // Suppress output during tests

	fm := NewFileManager(logger, true, tempDir) // dry-run mode

	// Create a test source file
	sourceFile := filepath.Join(tempDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Create test configuration
	files := map[string]config.File{
		"test-file": {
			Source:      "source.txt",
			Destination: filepath.Join(tempDir, "dest.txt"),
			Mode:        "644",
			Backup:      true,
		},
	}

	// Deploy files in dry-run mode
	err = fm.DeployFiles(files)
	if err != nil {
		t.Fatalf("unexpected error in dry-run: %v", err)
	}

	// Verify no actual file was created
	destFile := filepath.Join(tempDir, "dest.txt")
	if _, err := os.Stat(destFile); err == nil {
		t.Error("destination file should not exist in dry-run mode")
	}
}

func TestFileManager_DeployFiles_RealDeploy(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a logger for testing
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel) // Suppress output during tests

	fm := NewFileManager(logger, false, tempDir) // real deployment

	// Create a test source file
	sourceFile := filepath.Join(tempDir, "source.txt")
	testContent := []byte("test content")
	err := os.WriteFile(sourceFile, testContent, 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Create test configuration
	destFile := filepath.Join(tempDir, "dest.txt")
	files := map[string]config.File{
		"test-file": {
			Source:      "source.txt",
			Destination: destFile,
			Mode:        "644",
			Backup:      false,
		},
	}

	// Deploy files
	err = fm.DeployFiles(files)
	if err != nil {
		t.Fatalf("unexpected error during deployment: %v", err)
	}

	// Verify symlink was created
	if _, err := os.Lstat(destFile); err != nil {
		t.Fatalf("destination file was not created: %v", err)
	}

	// Verify it's a symlink
	info, err := os.Lstat(destFile)
	if err != nil {
		t.Fatalf("failed to stat destination file: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("destination file is not a symlink")
	}

	// Verify symlink points to correct file
	target, err := os.Readlink(destFile)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}
	if target != sourceFile {
		t.Errorf("symlink points to %s, expected %s", target, sourceFile)
	}
}

func TestFileManager_BackupExistingFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a logger for testing
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel) // Suppress output during tests

	fm := NewFileManager(logger, false, tempDir)

	// Create existing file at destination
	destFile := filepath.Join(tempDir, "dest.txt")
	existingContent := []byte("existing content")
	err := os.WriteFile(destFile, existingContent, 0644)
	if err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	// Create a test source file
	sourceFile := filepath.Join(tempDir, "source.txt")
	testContent := []byte("new content")
	err = os.WriteFile(sourceFile, testContent, 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Create test configuration with backup enabled
	files := map[string]config.File{
		"test-file": {
			Source:      "source.txt",
			Destination: destFile,
			Backup:      true,
		},
	}

	// Deploy files
	err = fm.DeployFiles(files)
	if err != nil {
		t.Fatalf("unexpected error during deployment: %v", err)
	}

	// Verify backup was created
	backupPattern := destFile + ".backup.*"
	matches, err := filepath.Glob(backupPattern)
	if err != nil {
		t.Fatalf("failed to search for backup files: %v", err)
	}
	if len(matches) == 0 {
		t.Error("no backup file was created")
	}

	// Verify backup contains original content
	if len(matches) > 0 {
		backupContent, err := os.ReadFile(matches[0])
		if err != nil {
			t.Fatalf("failed to read backup file: %v", err)
		}
		if string(backupContent) != string(existingContent) {
			t.Errorf("backup content mismatch: got %s, expected %s", string(backupContent), string(existingContent))
		}
	}

	// Verify destination is now a symlink
	info, err := os.Lstat(destFile)
	if err != nil {
		t.Fatalf("failed to stat destination file: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("destination file is not a symlink after deployment")
	}
}

func TestFileManager_ValidateFilePermissions(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a logger for testing
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel) // Suppress output during tests

	fm := NewFileManager(logger, false, tempDir)

	// Create test configuration
	files := map[string]config.File{
		"test-file": {
			Source:      "source.txt",
			Destination: filepath.Join(tempDir, "dest.txt"),
		},
		"invalid-dest": {
			Source:      "source.txt",
			Destination: "/root/restricted/file.txt", // Likely no permission
		},
	}

	// Test validation - should pass for temp dir but may fail for restricted
	err := fm.ValidateFilePermissions(map[string]config.File{
		"test-file": files["test-file"],
	})
	if err != nil {
		t.Errorf("validation should pass for temp directory: %v", err)
	}

	// Note: Testing restricted directory permission would require specific setup
	// and might not be portable across different test environments
}

func TestFileManager_handleExistingFile_NoBackup(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a logger for testing
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel) // Suppress output during tests

	fm := NewFileManager(logger, false, tempDir)

	// Create existing file
	existingFile := filepath.Join(tempDir, "existing.txt")
	err := os.WriteFile(existingFile, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	// Handle existing file without backup
	err = fm.handleExistingFile(existingFile, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was removed
	if _, err := os.Stat(existingFile); err == nil {
		t.Error("existing file should have been removed")
	}
}

func TestFileManager_setFileAttributes_DryRun(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a logger for testing
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel) // Suppress output during tests

	fm := NewFileManager(logger, true, tempDir) // dry-run mode

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Test setting attributes in dry-run mode
	file := config.File{
		Mode:  "755",
		Owner: "root",
		Group: "root",
	}

	err = fm.setFileAttributes(testFile, file)
	if err != nil {
		t.Errorf("dry-run should not fail: %v", err)
	}

	// Verify file attributes weren't actually changed
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Mode().Perm() != 0644 {
		t.Error("file permissions should not have changed in dry-run mode")
	}
}