package pkg

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

func TestFileManager_areFilesIdentical(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel)
	fm := NewFileManager(logger, false, tempDir)

	// Create two identical files
	content := "This is test content\nwith multiple lines\nfor testing file comparison"
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	
	err := os.WriteFile(file1, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file1: %v", err)
	}
	
	err = os.WriteFile(file2, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file2: %v", err)
	}

	// Test identical files
	if !fm.areFilesIdentical(file1, file2) {
		t.Error("Expected identical files to be detected as identical")
	}

	// Create a different file
	file3 := filepath.Join(tempDir, "file3.txt")
	err = os.WriteFile(file3, []byte("Different content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file3: %v", err)
	}

	// Test different files
	if fm.areFilesIdentical(file1, file3) {
		t.Error("Expected different files to be detected as different")
	}

	// Test with non-existent file
	nonExistent := filepath.Join(tempDir, "nonexistent.txt")
	if fm.areFilesIdentical(file1, nonExistent) {
		t.Error("Expected comparison with non-existent file to return false")
	}
}

func TestFileManager_calculateFileHash(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel)
	fm := NewFileManager(logger, false, tempDir)

	content := "Test content for hashing"
	testFile := filepath.Join(tempDir, "hashtest.txt")
	
	err := os.WriteFile(testFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash, err := fm.calculateFileHash(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate hash: %v", err)
	}

	// Calculate expected hash manually
	hasher := sha256.New()
	hasher.Write([]byte(content))
	expectedHash := fmt.Sprintf("%x", hasher.Sum(nil))

	if hash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, hash)
	}

	// Test with non-existent file
	nonExistent := filepath.Join(tempDir, "nonexistent.txt")
	_, err = fm.calculateFileHash(nonExistent)
	if err == nil {
		t.Error("Expected error when hashing non-existent file")
	}
}

func TestFileManager_requiresElevatedPrivileges(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel)
	fm := NewFileManager(logger, false, "/tmp")

	tests := []struct {
		path     string
		expected bool
		name     string
	}{
		{"/etc/config.conf", true, "etc directory"},
		{"/usr/local/bin/binary", true, "usr directory"},
		{"/opt/app/config", true, "opt directory"},
		{"/var/log/app.log", true, "var directory"},
		{"/bin/executable", true, "bin directory"},
		{"/sbin/service", true, "sbin directory"},
		{"/lib/library.so", true, "lib directory"},
		{"/home/user/config", false, "home directory"},
		{"/tmp/tempfile", false, "tmp directory"},
		{"/home/user/.config/app.conf", false, "user config directory"},
		{"relative/path", false, "relative path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fm.requiresElevatedPrivileges(tt.path)
			if result != tt.expected {
				t.Errorf("requiresElevatedPrivileges(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFileManager_handleExistingFile_AlreadyCorrectSymlink(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel)
	fm := NewFileManager(logger, false, tempDir)

	// Create source file
	sourceFile := filepath.Join(tempDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte("source content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create destination symlink pointing to source
	destFile := filepath.Join(tempDir, "dest_link.txt")
	err = os.Symlink(sourceFile, destFile)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	// Test handleExistingFile with correct symlink
	fileConfig := config.File{
		Source:      sourceFile,
		Destination: destFile,
		Backup:      false,
		Copy:        false, // Symlink mode
	}

	backupPath, err := fm.handleExistingFile("test", destFile, sourceFile, fileConfig)
	if err != nil {
		t.Fatalf("handleExistingFile failed: %v", err)
	}

	// Should return empty backup path since file was already correct
	if backupPath != "" {
		t.Errorf("Expected empty backup path for already correct symlink, got: %s", backupPath)
	}

	// Verify symlink still exists and points to correct source
	link, err := os.Readlink(destFile)
	if err != nil {
		t.Fatalf("Failed to read symlink: %v", err)
	}

	if filepath.Clean(link) != filepath.Clean(sourceFile) {
		t.Errorf("Symlink points to %s, expected %s", link, sourceFile)
	}
}

func TestFileManager_handleExistingFile_AlreadyCorrectCopy(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel)
	fm := NewFileManager(logger, false, tempDir)

	// Create source file
	content := "identical content for testing"
	sourceFile := filepath.Join(tempDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create destination file with identical content
	destFile := filepath.Join(tempDir, "dest_copy.txt")
	err = os.WriteFile(destFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create destination file: %v", err)
	}

	// Test handleExistingFile with identical copy
	fileConfig := config.File{
		Source:      sourceFile,
		Destination: destFile,
		Backup:      false,
		Copy:        true, // Copy mode
	}

	backupPath, err := fm.handleExistingFile("test", destFile, sourceFile, fileConfig)
	if err != nil {
		t.Fatalf("handleExistingFile failed: %v", err)
	}

	// Should return empty backup path since file was already identical
	if backupPath != "" {
		t.Errorf("Expected empty backup path for already identical copy, got: %s", backupPath)
	}

	// Verify file still exists with same content
	actualContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(actualContent) != content {
		t.Errorf("Destination file content changed, expected %s, got %s", content, string(actualContent))
	}
}

func TestFileManager_resolveSourcePath_WithConfigDir(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel)
	fm := NewFileManager(logger, false, "/main/config")

	tests := []struct {
		name        string
		source      string
		configDir   string
		expected    string
	}{
		{
			name:      "absolute path ignores config dir",
			source:    "/absolute/path/file.txt",
			configDir: "/included/config",
			expected:  "/absolute/path/file.txt",
		},
		{
			name:      "relative path uses config dir",
			source:    "files/config.txt",
			configDir: "/included/config",
			expected:  "/included/config/files/config.txt",
		},
		{
			name:      "relative path falls back to main config dir",
			source:    "files/config.txt",
			configDir: "", // Empty config dir
			expected:  "/main/config/files/config.txt",
		},
		{
			name:      "current directory file with config dir",
			source:    "config.yaml",
			configDir: "/included/config",
			expected:  "/included/config/config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := config.File{
				Source:    tt.source,
				ConfigDir: tt.configDir,
			}

			result, err := fm.resolveSourcePath(tt.source, file)
			if err != nil {
				t.Fatalf("resolveSourcePath failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("resolveSourcePath(%s, configDir=%s) = %s, expected %s", 
					tt.source, tt.configDir, result, tt.expected)
			}
		})
	}
}

func TestFileManager_ensureDirectory_SystemPaths(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel)
	fm := NewFileManager(logger, false, "/tmp")

	// Test creating a directory that requires privileges
	systemDir := "/etc/test-configr-dir"
	err := fm.ensureDirectory(systemDir)
	
	// Should fail with a helpful error message
	if err == nil {
		// Clean up if it somehow succeeded
		os.RemoveAll(systemDir)
		t.Error("Expected error when creating system directory without privileges")
	} else {
		// Check that error message mentions privileges
		if !containsAny(err.Error(), []string{"permission denied", "privileges", "sudo"}) {
			t.Errorf("Error message should mention privileges, got: %v", err)
		}
	}
}

func TestFileManager_ensureDirectory_Success(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr)
	logger.SetLevel(log.ErrorLevel)
	fm := NewFileManager(logger, false, tempDir)

	// Test creating a directory in temp space
	testDir := filepath.Join(tempDir, "nested", "deep", "directory")
	err := fm.ensureDirectory(testDir)
	if err != nil {
		t.Fatalf("ensureDirectory failed: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(testDir)
	if err != nil {
		t.Fatalf("Created directory does not exist: %v", err)
	}

	if !info.IsDir() {
		t.Error("Created path is not a directory")
	}

	// Test with already existing directory
	err = fm.ensureDirectory(testDir)
	if err != nil {
		t.Errorf("ensureDirectory should succeed for existing directory: %v", err)
	}
}

// Helper function to check if a string contains any of the given substrings
func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}