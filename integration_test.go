package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func getBinaryPath(t *testing.T) string {
	// Get the absolute path to the binary
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	binaryPath := filepath.Join(wd, "configr")
	
	// Skip if configr binary doesn't exist
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("configr binary not found, run 'go build' first")
	}
	
	return binaryPath
}

func TestIntegration_ValidateCommand(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a source file
	sourceFile := filepath.Join(tempDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	
	// Create valid config file
	configContent := `version: "1.0"
files:
  test:
    source: "source.txt"
    destination: "~/test-dest.txt"
    mode: "644"
    backup: true
packages:
  apt:
    - git
    - curl
dconf:
  settings:
    "/org/gnome/test": "'value'"
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run validate command
	cmd := exec.Command(binaryPath, "validate", configPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	// Check that command succeeded
	if err != nil {
		t.Errorf("validate command should succeed, got error: %v\nOutput: %s", err, string(output))
	}
	
	// Check that output contains success message
	outputStr := string(output)
	if !strings.Contains(outputStr, "✓") || !strings.Contains(outputStr, "valid") {
		t.Errorf("output should contain validation success message, got: %s", outputStr)
	}
}

func TestIntegration_ValidateCommand_InvalidConfig(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create invalid config file (missing version, nonexistent source file)
	configContent := `files:
  test:
    source: "nonexistent.txt"
    destination: "~/dest.txt"
packages:
  apt:
    - ""  # Invalid empty package name
dconf:
  settings:
    "invalid-path": "'value'"  # Missing leading slash
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run validate command
	cmd := exec.Command(binaryPath, "validate", configPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	// Check that command failed
	if err == nil {
		t.Error("validate command should fail for invalid config")
	}
	
	// Check that output contains error details
	outputStr := string(output)
	if !strings.Contains(outputStr, "error:") {
		t.Errorf("output should contain error details, got: %s", outputStr)
	}
}

func TestIntegration_ApplyCommand_DryRun(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a source file
	sourceFile := filepath.Join(tempDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	
	// Create config file
	configContent := `version: "1.0"
files:
  test:
    source: "source.txt"
    destination: "` + filepath.Join(tempDir, "dest.txt") + `"
    backup: true
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run apply command with dry-run
	cmd := exec.Command(binaryPath, "apply", configPath, "--dry-run")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	// Check that command succeeded
	if err != nil {
		t.Errorf("apply --dry-run should succeed, got error: %v\nOutput: %s", err, string(output))
	}
	
	// Check that output contains dry-run message
	outputStr := string(output)
	if !strings.Contains(outputStr, "dry-run") || !strings.Contains(outputStr, "no changes") {
		t.Errorf("output should contain dry-run message, got: %s", outputStr)
	}
	
	// Verify no actual file was created
	destPath := filepath.Join(tempDir, "dest.txt")
	if _, err := os.Stat(destPath); err == nil {
		t.Error("destination file should not exist in dry-run mode")
	}
}

func TestIntegration_ApplyCommand_RealDeploy(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a source file
	sourceFile := filepath.Join(tempDir, "source.txt")
	testContent := []byte("test content for symlink")
	err := os.WriteFile(sourceFile, testContent, 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	
	// Create config file with symlink mode (default)
	destPath := filepath.Join(tempDir, "dest-symlink.txt")
	configContent := `version: "1.0"
files:
  test-symlink:
    source: "source.txt"
    destination: "` + destPath + `"
    backup: true
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run apply command
	cmd := exec.Command(binaryPath, "apply", configPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	// Check that command succeeded
	if err != nil {
		t.Errorf("apply command should succeed, got error: %v\nOutput: %s", err, string(output))
	}
	
	// Check that output contains success message
	outputStr := string(output)
	if !strings.Contains(outputStr, "✓") {
		t.Errorf("output should contain success indicators, got: %s", outputStr)
	}
	
	// Verify the symlink was created
	if _, err := os.Lstat(destPath); err != nil {
		t.Errorf("destination symlink should exist: %v", err)
	}
	
	// Verify it's actually a symlink
	info, err := os.Lstat(destPath)
	if err != nil {
		t.Fatalf("failed to stat destination: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("destination should be a symlink")
	}
	
	// Verify symlink points to correct file
	target, err := os.Readlink(destPath)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}
	if target != sourceFile {
		t.Errorf("symlink should point to %s, got %s", sourceFile, target)
	}
}

func TestIntegration_ApplyCommand_CopyMode(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a source file
	sourceFile := filepath.Join(tempDir, "source.txt")
	testContent := []byte("test content for copy mode")
	err := os.WriteFile(sourceFile, testContent, 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	
	// Create config file with copy mode
	destPath := filepath.Join(tempDir, "dest-copy.txt")
	configContent := `version: "1.0"
files:
  test-copy:
    source: "source.txt"
    destination: "` + destPath + `"
    copy: true
    backup: true
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run apply command
	cmd := exec.Command(binaryPath, "apply", configPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	// Check that command succeeded
	if err != nil {
		t.Errorf("apply command should succeed, got error: %v\nOutput: %s", err, string(output))
	}
	
	// Verify the file was created (not as symlink)
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("destination file should exist: %v", err)
	}
	
	// Verify it's NOT a symlink
	info, err := os.Lstat(destPath)
	if err != nil {
		t.Fatalf("failed to stat destination: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("destination should not be a symlink in copy mode")
	}
	
	// Verify content was copied correctly
	copiedContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read copied file: %v", err)
	}
	if string(copiedContent) != string(testContent) {
		t.Errorf("copied content mismatch: expected %s, got %s", string(testContent), string(copiedContent))
	}
	
	// Modify source file to verify independence in copy mode
	modifiedContent := []byte("modified content")
	err = os.WriteFile(sourceFile, modifiedContent, 0644)
	if err != nil {
		t.Fatalf("failed to modify source file: %v", err)
	}
	
	// Verify copied file still has original content (independence)
	copiedContentAfter, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read copied file after source modification: %v", err)
	}
	if string(copiedContentAfter) != string(testContent) {
		t.Error("copied file should retain original content after source modification")
	}
}

func TestIntegration_ApplyCommand_InvalidConfig(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create invalid config file (missing version, missing source file)
	configContent := `files:
  test:
    source: "nonexistent.txt"
    destination: "~/dest.txt"
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run apply command
	cmd := exec.Command(binaryPath, "apply", configPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	// Check that command failed
	if err == nil {
		t.Error("apply command should fail for invalid config")
	}
	
	// Check that output mentions validation failure
	outputStr := string(output)
	if !strings.Contains(outputStr, "validation") || !strings.Contains(outputStr, "error") {
		t.Errorf("output should mention validation errors, got: %s", outputStr)
	}
}

func TestIntegration_HelpCommand(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Run help command
	cmd := exec.Command(binaryPath, "--help")
	output, err := cmd.CombinedOutput()
	
	// Check that command succeeded
	if err != nil {
		t.Errorf("help command should succeed, got error: %v", err)
	}
	
	// Check that output contains expected sections
	outputStr := string(output)
	expectedSections := []string{"USAGE", "COMMANDS", "FLAGS", "validate", "apply"}
	
	for _, section := range expectedSections {
		if !strings.Contains(outputStr, section) {
			t.Errorf("help output should contain '%s', got: %s", section, outputStr)
		}
	}
}

func TestIntegration_VersionCommand(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Run version command
	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.CombinedOutput()
	
	// Check that command succeeded
	if err != nil {
		t.Errorf("version command should succeed, got error: %v", err)
	}
	
	// Check that output contains version info
	outputStr := string(output)
	if !strings.Contains(outputStr, "configr") {
		t.Errorf("version output should contain 'configr', got: %s", outputStr)
	}
}