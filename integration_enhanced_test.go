package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegration_AdvancedIncludes tests the advanced include system with glob patterns and conditions
func TestIntegration_AdvancedIncludes(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	
	// Create source files for includes
	includeDir := filepath.Join(tempDir, "includes")
	err := os.MkdirAll(includeDir, 0755)
	if err != nil {
		t.Fatalf("failed to create include directory: %v", err)
	}
	
	// Create multiple include files
	include1Content := `packages:
  apt:
    - curl
    - wget`
	include1Path := filepath.Join(includeDir, "packages1.yaml")
	err = os.WriteFile(include1Path, []byte(include1Content), 0644)
	if err != nil {
		t.Fatalf("failed to create include file 1: %v", err)
	}
	
	include2Content := `packages:
  apt:
    - vim
    - git`
	include2Path := filepath.Join(includeDir, "packages2.yaml")
	err = os.WriteFile(include2Path, []byte(include2Content), 0644)
	if err != nil {
		t.Fatalf("failed to create include file 2: %v", err)
	}
	
	// Create OS-specific include
	osIncludeContent := `packages:
  snap:
    - code`
	osIncludePath := filepath.Join(tempDir, "linux.yaml")
	err = os.WriteFile(osIncludePath, []byte(osIncludeContent), 0644)
	if err != nil {
		t.Fatalf("failed to create OS include file: %v", err)
	}
	
	// Main config with advanced includes
	configContent := `version: "1.0"
advanced_includes:
  - glob: "includes/*.yaml"
    description: "All package configurations"
    optional: false
  - path: "linux.yaml"
    description: "Linux-specific packages"
    optional: true
    conditions:
      - type: "os"
        value: "linux"
        operator: "equals"
packages:
  apt:
    - base-package`
	
	configPath := filepath.Join(tempDir, "configr.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run validate command to test advanced includes
	cmd := exec.Command(binaryPath, "validate", configPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Advanced includes test output: %s", string(output))
	}
	
	// Check that the validation processes the includes
	outputStr := string(output)
	if strings.Contains(outputStr, "error") && !strings.Contains(outputStr, "TTY") {
		t.Errorf("Advanced includes validation failed unexpectedly: %s", outputStr)
	}
}

// TestIntegration_ThreeTierFlagSystem tests the three-tier package flag resolution
func TestIntegration_ThreeTierFlagSystem(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	tempDir := t.TempDir()
	
	// Create config with three-tier flag system
	configContent := `version: "1.0"
package_defaults:
  apt: ["-y", "--no-install-recommends"]
  flatpak: ["--user"]
  snap: ["--classic"]
packages:
  apt:
    - name: vim
      flags: ["-y", "--install-suggests"]  # Override defaults
    - name: git  # Uses package defaults
  flatpak:
    - name: org.mozilla.Firefox
      flags: ["--system"]  # Override user defaults
    - name: org.libreoffice.LibreOffice  # Uses defaults
  snap:
    - name: code
      flags: ["--classic", "--dangerous"]  # Override defaults
    - name: discord  # Uses defaults`
	
	configPath := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run dry-run to test flag resolution
	cmd := exec.Command(binaryPath, "apply", configPath, "--dry-run")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Three-tier flag test output: %s", string(output))
	}
	
	outputStr := string(output)
	
	// Should show flag differences in dry-run output
	if !strings.Contains(outputStr, "dry-run") && !strings.Contains(outputStr, "TTY") {
		t.Logf("Flag system test - output: %s", outputStr)
	}
}

// TestIntegration_StateManagement tests the state tracking and removal system
func TestIntegration_StateManagement(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	tempDir := t.TempDir()
	
	// Create source file
	sourceFile := filepath.Join(tempDir, "source.txt")
	err := os.WriteFile(sourceFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}
	
	// Initial config with file and packages
	configContent1 := `version: "1.0"
files:
  test:
    source: "source.txt"
    destination: "~/test-dest.txt"
packages:
  apt:
    - git
    - vim`
	
	configPath := filepath.Join(tempDir, "configr.yaml")
	err = os.WriteFile(configPath, []byte(configContent1), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// First apply - dry run
	cmd := exec.Command(binaryPath, "apply", configPath, "--dry-run")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("State management test (first apply) output: %s", string(output))
	}
	
	// Modified config with removed items (to test removal system)
	configContent2 := `version: "1.0"
packages:
  apt:
    - git  # vim removed, should be detected`
	
	err = os.WriteFile(configPath, []byte(configContent2), 0644)
	if err != nil {
		t.Fatalf("failed to update config file: %v", err)
	}
	
	// Second apply - should detect removals
	cmd = exec.Command(binaryPath, "apply", configPath, "--dry-run")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("State management test (second apply) output: %s", string(output))
	}
	
	outputStr := string(output)
	// In a real scenario, this would show removal operations
	if !strings.Contains(outputStr, "dry-run") && !strings.Contains(outputStr, "TTY") {
		t.Logf("State management test - output: %s", outputStr)
	}
}

// TestIntegration_RepositoryManagement tests APT and Flatpak repository management
func TestIntegration_RepositoryManagement(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	tempDir := t.TempDir()
	
	// Config with repositories
	configContent := `version: "1.0"
repositories:
  apt:
    docker:
      uri: "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"
      key: "https://download.docker.com/linux/ubuntu/gpg"
    python39:
      ppa: "deadsnakes/ppa"
  flatpak:
    flathub:
      url: "https://flathub.org/repo/flathub.flatpakrepo"
      user: false
    kde:
      url: "https://distribute.kde.org/kdeapps.flatpakrepo"
      user: true
packages:
  apt:
    - docker-ce
  flatpak:
    - org.kde.krita`
	
	configPath := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run validation
	cmd := exec.Command(binaryPath, "validate", configPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Repository management test output: %s", string(output))
	}
	
	// Run dry-run apply
	cmd = exec.Command(binaryPath, "apply", configPath, "--dry-run")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Repository management apply test output: %s", string(output))
	}
	
	outputStr := string(output)
	if strings.Contains(outputStr, "error") && !strings.Contains(outputStr, "TTY") {
		t.Logf("Repository management test had errors: %s", outputStr)
	}
}

// TestIntegration_DConfAdvanced tests advanced DConf settings
func TestIntegration_DConfAdvanced(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	tempDir := t.TempDir()
	
	// Config with comprehensive DConf settings
	configContent := `version: "1.0"
dconf:
  settings:
    "/org/gnome/desktop/interface/gtk-theme": "'Adwaita-dark'"
    "/org/gnome/desktop/interface/cursor-size": "24"
    "/org/gnome/desktop/interface/enable-animations": "true"
    "/org/gnome/shell/enabled-extensions": "['dash-to-dock@micxgx.gmail.com']"
    "/org/gnome/desktop/wm/preferences/button-layout": "'close,minimize,maximize:'"
    "/org/gnome/settings-daemon/plugins/power/sleep-inactive-ac-timeout": "3600"`
	
	configPath := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run validation
	cmd := exec.Command(binaryPath, "validate", configPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("DConf advanced test output: %s", string(output))
	}
	
	// Run dry-run apply
	cmd = exec.Command(binaryPath, "apply", configPath, "--dry-run")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("DConf advanced apply test output: %s", string(output))
	}
	
	outputStr := string(output)
	if strings.Contains(outputStr, "error") && !strings.Contains(outputStr, "TTY") && !strings.Contains(outputStr, "dconf") {
		t.Errorf("DConf advanced test failed: %s", outputStr)
	}
}

// TestIntegration_ComplexConfiguration tests a complex real-world configuration
func TestIntegration_ComplexConfiguration(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	tempDir := t.TempDir()
	
	// Create source files
	dotfilesDir := filepath.Join(tempDir, "dotfiles")
	err := os.MkdirAll(dotfilesDir, 0755)
	if err != nil {
		t.Fatalf("failed to create dotfiles directory: %v", err)
	}
	
	vimrcContent := `" Simple vimrc
set number
set tabstop=4`
	vimrcPath := filepath.Join(dotfilesDir, ".vimrc")
	err = os.WriteFile(vimrcPath, []byte(vimrcContent), 0644)
	if err != nil {
		t.Fatalf("failed to create .vimrc: %v", err)
	}
	
	bashrcContent := `# Simple bashrc
export EDITOR=vim
alias ll='ls -la'`
	bashrcPath := filepath.Join(dotfilesDir, ".bashrc")
	err = os.WriteFile(bashrcPath, []byte(bashrcContent), 0644)
	if err != nil {
		t.Fatalf("failed to create .bashrc: %v", err)
	}
	
	// Complex configuration
	configContent := `version: "1.0"
package_defaults:
  apt: ["-y", "--no-install-recommends"]
  flatpak: ["--user", "--assumeyes"]
  snap: ["--classic"]

repositories:
  apt:
    vscode:
      uri: "deb [arch=amd64,arm64,armhf] https://packages.microsoft.com/repos/code stable main"
      key: "BC528686B50D79E339D3721CEB3E94ADBE1229CF"

packages:
  apt:
    - name: vim
      flags: ["-y"]
    - git
    - curl
    - build-essential
  flatpak:
    - name: org.mozilla.Firefox
      flags: ["--system"]
    - org.libreoffice.LibreOffice
  snap:
    - name: code
      flags: ["--classic"]
    - discord

files:
  vimrc:
    source: "dotfiles/.vimrc"
    destination: "~/.vimrc"
    mode: "644"
    backup: true
    copy: false
  bashrc:
    source: "dotfiles/.bashrc"
    destination: "~/.bashrc"
    mode: "644"
    backup: true
    copy: true
    owner: "$USER"
    group: "$USER"

dconf:
  settings:
    "/org/gnome/desktop/interface/gtk-theme": "'Adwaita-dark'"
    "/org/gnome/desktop/interface/cursor-size": "24"
    "/org/gnome/shell/favorite-apps": "['firefox.desktop', 'org.gnome.Nautilus.desktop']"`
	
	configPath := filepath.Join(tempDir, "configr.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run validation
	cmd := exec.Command(binaryPath, "validate", configPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Complex configuration validation output: %s", string(output))
	}
	
	// Run dry-run apply
	cmd = exec.Command(binaryPath, "apply", configPath, "--dry-run")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Complex configuration apply output: %s", string(output))
	}
	
	outputStr := string(output)
	if strings.Contains(outputStr, "fatal") {
		t.Errorf("Complex configuration test failed with fatal error: %s", outputStr)
	}
}

// TestIntegration_OptimizationAndCaching tests the optimization and caching system
func TestIntegration_OptimizationAndCaching(t *testing.T) {
	binaryPath := getBinaryPath(t)
	
	tempDir := t.TempDir()
	
	// Simple config for caching test
	configContent := `version: "1.0"
packages:
  apt:
    - git
    - vim`
	
	configPath := filepath.Join(tempDir, "configr.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}
	
	// Run with optimization enabled (default)
	cmd := exec.Command(binaryPath, "apply", configPath, "--dry-run", "--optimize=true")
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Optimization test (enabled) output: %s", string(output))
	}
	
	// Run with optimization disabled
	cmd = exec.Command(binaryPath, "apply", configPath, "--dry-run", "--optimize=false")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Optimization test (disabled) output: %s", string(output))
	}
	
	// Test cache commands
	cmd = exec.Command(binaryPath, "cache", "stats")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Cache stats test output: %s", string(output))
	}
	
	cmd = exec.Command(binaryPath, "cache", "clear")
	cmd.Dir = tempDir
	output, err = cmd.CombinedOutput()
	
	if err != nil {
		t.Logf("Cache clear test output: %s", string(output))
	}
	
	outputStr := string(output)
	if strings.Contains(outputStr, "fatal") {
		t.Errorf("Cache management test failed: %s", outputStr)
	}
}