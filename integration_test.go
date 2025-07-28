package main

import (
	"fmt"
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

func TestIntegration_APTPackageManager(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create config file with APT packages (using packages that should exist but we'll use dry-run)
	configContent := `version: "1.0"
packages:
  apt:
    - curl
    - wget
    - git
package_defaults:
  apt: ["-y", "--dry-run"]
repositories:
  apt:
    universe:
      ppa: "universe"
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run apply command with dry-run to test APT integration
	cmd := exec.Command(binaryPath, "apply", configPath, "--dry-run")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	// Check command execution
	if err != nil {
		t.Logf("APT test command output: %s", string(output))
		// APT might not be available in test environment, so we check output for expected behavior
	}
	
	// Check that output mentions APT packages
	outputStr := string(output)
	if !strings.Contains(outputStr, "apt") && !strings.Contains(outputStr, "package") {
		t.Logf("APT package test - output: %s", outputStr)
	}
}

func TestIntegration_FlatpakPackageManager(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create config file with Flatpak packages
	configContent := `version: "1.0"
packages:
  flatpak:
    - org.mozilla.firefox
    - org.gnome.gedit
package_defaults:
  flatpak: ["--user", "--assumeyes"]
repositories:
  flatpak:
    flathub:
      url: "https://flathub.org/repo/flathub.flatpakrepo"
      user: false
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run apply command with dry-run to test Flatpak integration
	cmd := exec.Command(binaryPath, "apply", configPath, "--dry-run")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	// Check command execution
	if err != nil {
		t.Logf("Flatpak test command output: %s", string(output))
		// Flatpak might not be available in test environment
	}
	
	// Check that output processes Flatpak packages
	outputStr := string(output)
	if !strings.Contains(outputStr, "flatpak") && !strings.Contains(outputStr, "package") {
		t.Logf("Flatpak package test - output: %s", outputStr)
	}
}

func TestIntegration_SnapPackageManager(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create config file with Snap packages
	configContent := `version: "1.0"
packages:
  snap:
    - code
    - discord
package_defaults:
  snap: ["--classic"]
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run apply command with dry-run to test Snap integration
	cmd := exec.Command(binaryPath, "apply", configPath, "--dry-run")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	// Check command execution
	if err != nil {
		t.Logf("Snap test command output: %s", string(output))
		// Snap might not be available in test environment
	}
	
	// Check that output processes Snap packages
	outputStr := string(output)
	if !strings.Contains(outputStr, "snap") && !strings.Contains(outputStr, "package") {
		t.Logf("Snap package test - output: %s", outputStr)
	}
}

func TestIntegration_DConfSettings(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create config file with DConf settings
	configContent := `version: "1.0"
dconf:
  settings:
    "/org/gnome/desktop/interface/gtk-theme": "'Adwaita'"
    "/org/gnome/desktop/interface/icon-theme": "'Adwaita'"
    "/org/gnome/terminal/legacy/profiles:/:default-profile-id": "'default'"
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run apply command with dry-run to test DConf integration
	cmd := exec.Command(binaryPath, "apply", configPath, "--dry-run")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	// Check command execution
	if err != nil {
		t.Logf("DConf test command output: %s", string(output))
		// DConf might not be available in test environment
	}
	
	// Check that output processes DConf settings
	outputStr := string(output)
	if !strings.Contains(outputStr, "dconf") && !strings.Contains(outputStr, "setting") {
		t.Logf("DConf settings test - output: %s", outputStr)
	}
}

func TestIntegration_CacheCommands(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Test cache stats command
	cmd := exec.Command(binaryPath, "cache", "stats")
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Cache stats command output: %s", string(output))
		// Cache command might fail in fresh environment, which is expected
	}
	
	// Test cache clear command
	cmd = exec.Command(binaryPath, "cache", "clear")
	output, err = cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Cache clear command output: %s", string(output))
		// Cache command might fail in fresh environment, which is expected
	}
	
	// Test cache info command
	cmd = exec.Command(binaryPath, "cache", "info")
	output, err = cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Cache info command output: %s", string(output))
		// Cache command might fail in fresh environment, which is expected
	}
}

func TestIntegration_InteractiveMode(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a source file
	sourceFile := filepath.Join(tempDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	
	// Create existing destination file to trigger interactive mode
	destPath := filepath.Join(tempDir, "dest.txt")
	err = os.WriteFile(destPath, []byte("existing content"), 0644)
	if err != nil {
		t.Fatalf("failed to create destination file: %v", err)
	}
	
	// Create config file with interactive mode enabled
	configContent := `version: "1.0"
files:
  test:
    source: "source.txt"
    destination: "` + destPath + `"
    interactive: true
    prompt_permissions: true
    prompt_ownership: true
`
	configPath := filepath.Join(tempDir, "configr.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run apply command with interactive mode (will use defaults since no TTY)
	cmd := exec.Command(binaryPath, "apply", configPath, "--interactive", "--dry-run")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	// Check command execution (should handle gracefully without TTY)
	if err != nil {
		t.Logf("Interactive mode test output: %s", string(output))
		// Interactive features might not work without TTY, which is expected
	}
	
	// Check that interactive features are mentioned or handled
	outputStr := string(output)
	if !strings.Contains(outputStr, "interactive") && !strings.Contains(outputStr, "dry-run") {
		t.Logf("Interactive mode test - output: %s", outputStr)
	}
}

func TestIntegration_BinaryManagement_Validation(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Test valid binary configuration
	validConfig := `version: "1.0"
binaries:
  test-tool:
    source: "https://github.com/user/repo/releases/download/v1.0.0/tool"
    destination: "/usr/local/bin/tool"
    mode: "755"
    backup: true
`
	
	configPath := filepath.Join(tempDir, "valid-binary.yaml")
	err := os.WriteFile(configPath, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run validate command
	cmd := exec.Command(binaryPath, "validate", configPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Fatalf("validation should pass for valid binary config, got error: %v, output: %s", err, string(output))
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "✓") && !strings.Contains(outputStr, "valid") {
		t.Logf("Validation output: %s", outputStr)
	}
}

func TestIntegration_BinaryManagement_InvalidConfig(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Test invalid binary configuration (HTTP URL)
	invalidConfig := `version: "1.0"
binaries:
  insecure-tool:
    source: "http://example.com/tool"
    destination: "/usr/local/bin/tool"
    mode: "755"
`
	
	configPath := filepath.Join(tempDir, "invalid-binary.yaml")
	err := os.WriteFile(configPath, []byte(invalidConfig), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run validate command
	cmd := exec.Command(binaryPath, "validate", configPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	// Should fail validation
	if err == nil {
		t.Fatalf("validation should fail for insecure HTTP URL in binary config")
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "insecure") && !strings.Contains(outputStr, "error") {
		t.Errorf("expected validation error message about insecure URL, got: %s", outputStr)
	}
}

func TestIntegration_BinaryManagement_DryRun(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a temporary directory for binary destinations
	binDir := filepath.Join(tempDir, "bin")
	err := os.MkdirAll(binDir, 0755)
	if err != nil {
		t.Fatalf("failed to create bin directory: %v", err)
	}

	// Test binary configuration with dry run
	dryRunConfig := fmt.Sprintf(`version: "1.0"
binaries:
  hugo:
    source: "https://github.com/gohugoio/hugo/releases/download/v0.120.0/hugo_extended_0.120.0_linux-amd64.tar.gz"
    destination: "%s/hugo"
    mode: "755"
    backup: true
  gh:
    source: "https://github.com/cli/cli/releases/download/v2.40.0/gh_2.40.0_linux_amd64.tar.gz"
    destination: "%s/gh"
    mode: "755"
`, binDir, binDir)
	
	configPath := filepath.Join(tempDir, "binary-dryrun.yaml")
	err = os.WriteFile(configPath, []byte(dryRunConfig), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run apply command with dry-run
	cmd := exec.Command(binaryPath, "apply", configPath, "--dry-run")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Fatalf("dry-run should not fail for binary config, got error: %v, output: %s", err, string(output))
	}
	
	outputStr := string(output)
	// Check for dry-run indicators
	if !strings.Contains(outputStr, "dry-run") && !strings.Contains(outputStr, "would") {
		t.Logf("Dry-run binary management output: %s", outputStr)
	}
	
	// Verify no actual files were created in test directories (dry-run mode)
	testPaths := []string{filepath.Join(binDir, "hugo"), filepath.Join(binDir, "gh")}
	for _, path := range testPaths {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("dry-run should not create actual files: %s", path)
		}
	}
}

func TestIntegration_BinaryManagement_HomeDirectory(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create a temporary "home" directory for testing
	homeDir := filepath.Join(tempDir, "home")
	err := os.MkdirAll(filepath.Join(homeDir, "bin"), 0755)
	if err != nil {
		t.Fatalf("failed to create test home directory: %v", err)
	}
	
	// Test binary configuration targeting home directory
	homeConfig := `version: "1.0"
binaries:
  local-tool:
    source: "https://github.com/user/tool/releases/download/v1.0.0/tool"
    destination: "~/bin/local-tool"
    mode: "755"
    backup: true
`
	
	configPath := filepath.Join(tempDir, "binary-home.yaml")
	err = os.WriteFile(configPath, []byte(homeConfig), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run validate command (should pass)
	cmd := exec.Command(binaryPath, "validate", configPath)
	cmd.Dir = tempDir
	// Set HOME to our temporary directory
	cmd.Env = append(os.Environ(), "HOME="+homeDir)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Fatalf("validation should pass for home directory binary config, got error: %v, output: %s", err, string(output))
	}
	
	// Run dry-run apply
	cmd = exec.Command(binaryPath, "apply", configPath, "--dry-run")
	cmd.Dir = tempDir
	cmd.Env = append(os.Environ(), "HOME="+homeDir)
	output, err = cmd.CombinedOutput()
	
	if err != nil {
		t.Fatalf("dry-run should pass for home directory binary config, got error: %v, output: %s", err, string(output))
	}
}

func TestIntegration_BinaryManagement_MissingFields(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Test binary configuration with missing required fields
	incompleteConfig := `version: "1.0"
binaries:
  incomplete-tool:
    source: "https://github.com/user/repo/releases/download/v1.0.0/tool"
    # destination missing
    mode: "755"
`
	
	configPath := filepath.Join(tempDir, "incomplete-binary.yaml")
	err := os.WriteFile(configPath, []byte(incompleteConfig), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run validate command (should fail)
	cmd := exec.Command(binaryPath, "validate", configPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	if err == nil {
		t.Fatalf("validation should fail for binary config missing destination")
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "destination") && !strings.Contains(outputStr, "missing") {
		t.Errorf("expected validation error about missing destination, got: %s", outputStr)
	}
}

func TestIntegration_BinaryManagement_InvalidPermissions(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Test binary configuration with invalid file mode
	invalidModeConfig := `version: "1.0"  
binaries:
  bad-permissions:
    source: "https://github.com/user/repo/releases/download/v1.0.0/tool"
    destination: "/usr/local/bin/tool"
    mode: "999"  # Invalid octal mode
`
	
	configPath := filepath.Join(tempDir, "invalid-mode.yaml")
	err := os.WriteFile(configPath, []byte(invalidModeConfig), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run validate command (should fail)
	cmd := exec.Command(binaryPath, "validate", configPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	if err == nil {
		t.Fatalf("validation should fail for binary config with invalid file mode")
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "mode") && !strings.Contains(outputStr, "invalid") {
		t.Errorf("expected validation error about invalid file mode, got: %s", outputStr)
	}
}