package pkg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

func TestStateManager_LoadState_NewState(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	logger := log.New(os.Stderr)
	sm := NewStateManagerWithPath(logger, statePath)

	// Load state when file doesn't exist
	state, err := sm.LoadState()
	if err != nil {
		t.Fatalf("LoadState() failed: %v", err)
	}

	// Verify empty state structure
	if state.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", state.Version)
	}

	if len(state.Packages.Apt) != 0 || len(state.Packages.Flatpak) != 0 || len(state.Packages.Snap) != 0 {
		t.Errorf("Expected empty packages, got %+v", state.Packages)
	}
}

func TestStateManager_SaveAndLoadState(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	logger := log.New(os.Stderr)
	sm := NewStateManagerWithPath(logger, statePath)

	// Create test state
	originalState := &PackageState{
		Version:     "1.0",
		LastUpdated: time.Now().Truncate(time.Second), // Truncate for comparison
		Packages: ManagedPackages{
			Apt:     []string{"vim", "git", "curl"},
			Flatpak: []string{"org.mozilla.Firefox", "com.spotify.Client"},
			Snap:    []string{"code", "discord"},
		},
	}

	// Save state
	if err := sm.SaveState(originalState); err != nil {
		t.Fatalf("SaveState() failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Fatalf("State file was not created")
	}

	// Load state
	loadedState, err := sm.LoadState()
	if err != nil {
		t.Fatalf("LoadState() failed: %v", err)
	}

	// Compare states
	if loadedState.Version != originalState.Version {
		t.Errorf("Version mismatch: expected %s, got %s", originalState.Version, loadedState.Version)
	}

	if !stringSlicesEqual(loadedState.Packages.Apt, originalState.Packages.Apt) {
		t.Errorf("APT packages mismatch: expected %v, got %v", originalState.Packages.Apt, loadedState.Packages.Apt)
	}

	if !stringSlicesEqual(loadedState.Packages.Flatpak, originalState.Packages.Flatpak) {
		t.Errorf("Flatpak packages mismatch: expected %v, got %v", originalState.Packages.Flatpak, loadedState.Packages.Flatpak)
	}

	if !stringSlicesEqual(loadedState.Packages.Snap, originalState.Packages.Snap) {
		t.Errorf("Snap packages mismatch: expected %v, got %v", originalState.Packages.Snap, loadedState.Packages.Snap)
	}
}

func TestStateManager_UpdatePackageState(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	logger := log.New(os.Stderr)
	sm := NewStateManagerWithPath(logger, statePath)

	// Create test configuration
	cfg := &config.Config{
		Packages: config.PackageManagement{
			Apt: []config.PackageEntry{
				{Name: "vim"},
				{Name: "git"},
			},
			Flatpak: []config.PackageEntry{
				{Name: "org.mozilla.Firefox"},
			},
			Snap: []config.PackageEntry{
				{Name: "code"},
			},
		},
	}

	// Update state from configuration
	if err := sm.UpdatePackageState(cfg); err != nil {
		t.Fatalf("UpdatePackageState() failed: %v", err)
	}

	// Load and verify state
	state, err := sm.LoadState()
	if err != nil {
		t.Fatalf("LoadState() failed: %v", err)
	}

	expectedApt := []string{"vim", "git"}
	expectedFlatpak := []string{"org.mozilla.Firefox"}
	expectedSnap := []string{"code"}

	if !stringSlicesEqual(state.Packages.Apt, expectedApt) {
		t.Errorf("APT packages mismatch: expected %v, got %v", expectedApt, state.Packages.Apt)
	}

	if !stringSlicesEqual(state.Packages.Flatpak, expectedFlatpak) {
		t.Errorf("Flatpak packages mismatch: expected %v, got %v", expectedFlatpak, state.Packages.Flatpak)
	}

	if !stringSlicesEqual(state.Packages.Snap, expectedSnap) {
		t.Errorf("Snap packages mismatch: expected %v, got %v", expectedSnap, state.Packages.Snap)
	}
}

func TestStateManager_GetPackagesToRemove(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	logger := log.New(os.Stderr)
	sm := NewStateManagerWithPath(logger, statePath)

	// Create initial state with packages
	initialState := &PackageState{
		Version:     "1.0",
		LastUpdated: time.Now(),
		Packages: ManagedPackages{
			Apt:     []string{"vim", "git", "curl", "removed-package"},
			Flatpak: []string{"org.mozilla.Firefox", "com.spotify.Client", "org.removed.App"},
			Snap:    []string{"code", "discord", "removed-snap"},
		},
	}

	// Save initial state
	if err := sm.SaveState(initialState); err != nil {
		t.Fatalf("SaveState() failed: %v", err)
	}

	// Create new configuration (missing some packages)
	newCfg := &config.Config{
		Packages: config.PackageManagement{
			Apt: []config.PackageEntry{
				{Name: "vim"},
				{Name: "git"},
				// "curl" and "removed-package" are missing
			},
			Flatpak: []config.PackageEntry{
				{Name: "org.mozilla.Firefox"},
				// "com.spotify.Client" and "org.removed.App" are missing
			},
			Snap: []config.PackageEntry{
				{Name: "code"},
				// "discord" and "removed-snap" are missing
			},
		},
	}

	// Get packages to remove
	toRemove, err := sm.GetPackagesToRemove(newCfg)
	if err != nil {
		t.Fatalf("GetPackagesToRemove() failed: %v", err)
	}

	// Verify packages to remove
	expectedAptRemove := []string{"curl", "removed-package"}
	expectedFlatpakRemove := []string{"com.spotify.Client", "org.removed.App"}
	expectedSnapRemove := []string{"discord", "removed-snap"}

	if !stringSlicesEqualUnordered(toRemove.Apt, expectedAptRemove) {
		t.Errorf("APT packages to remove mismatch: expected %v, got %v", expectedAptRemove, toRemove.Apt)
	}

	if !stringSlicesEqualUnordered(toRemove.Flatpak, expectedFlatpakRemove) {
		t.Errorf("Flatpak packages to remove mismatch: expected %v, got %v", expectedFlatpakRemove, toRemove.Flatpak)
	}

	if !stringSlicesEqualUnordered(toRemove.Snap, expectedSnapRemove) {
		t.Errorf("Snap packages to remove mismatch: expected %v, got %v", expectedSnapRemove, toRemove.Snap)
	}
}

func TestStringSliceDiff(t *testing.T) {
	tests := []struct {
		name     string
		slice1   []string
		slice2   []string
		expected []string
	}{
		{
			name:     "no differences",
			slice1:   []string{"a", "b", "c"},
			slice2:   []string{"a", "b", "c"},
			expected: []string{},
		},
		{
			name:     "some differences",
			slice1:   []string{"a", "b", "c", "d"},
			slice2:   []string{"b", "c"},
			expected: []string{"a", "d"},
		},
		{
			name:     "all different",
			slice1:   []string{"a", "b"},
			slice2:   []string{"x", "y"},
			expected: []string{"a", "b"},
		},
		{
			name:     "empty first slice",
			slice1:   []string{},
			slice2:   []string{"a", "b"},
			expected: []string{},
		},
		{
			name:     "empty second slice",
			slice1:   []string{"a", "b"},
			slice2:   []string{},
			expected: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringSliceDiff(tt.slice1, tt.slice2)
			if !stringSlicesEqualUnordered(result, tt.expected) {
				t.Errorf("stringSliceDiff(%v, %v) = %v, expected %v", tt.slice1, tt.slice2, result, tt.expected)
			}
		})
	}
}

func TestStateManager_UpdateStateWithFiles(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	logger := log.New(os.Stderr)
	sm := NewStateManagerWithPath(logger, statePath)

	// Create test configuration
	cfg := &config.Config{
		Packages: config.PackageManagement{
			Apt: []config.PackageEntry{
				{Name: "vim"},
			},
		},
		Files: map[string]config.File{
			"vimrc": {Source: "dotfiles/.vimrc", Destination: "~/.vimrc"},
		},
	}

	// Create deployed files info
	deployedFiles := []ManagedFile{
		{
			Name:        "vimrc",
			Destination: "/home/user/.vimrc",
			IsSymlink:   true,
			BackupPath:  "",
		},
	}

	// Update state with files
	if err := sm.UpdateState(cfg, deployedFiles); err != nil {
		t.Fatalf("UpdateState() failed: %v", err)
	}

	// Load and verify state
	state, err := sm.LoadState()
	if err != nil {
		t.Fatalf("LoadState() failed: %v", err)
	}

	// Check packages
	expectedApt := []string{"vim"}
	if !stringSlicesEqual(state.Packages.Apt, expectedApt) {
		t.Errorf("APT packages mismatch: expected %v, got %v", expectedApt, state.Packages.Apt)
	}

	// Check files
	if len(state.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(state.Files))
	}

	file := state.Files[0]
	if file.Name != "vimrc" {
		t.Errorf("Expected file name 'vimrc', got %s", file.Name)
	}
	if file.Destination != "/home/user/.vimrc" {
		t.Errorf("Expected destination '/home/user/.vimrc', got %s", file.Destination)
	}
	if !file.IsSymlink {
		t.Errorf("Expected file to be symlink, got %v", file.IsSymlink)
	}
}

func TestStateManager_GetFilesToRemove(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	logger := log.New(os.Stderr)
	sm := NewStateManagerWithPath(logger, statePath)

	// Create initial state with files
	initialState := &PackageState{
		Version:     "1.0",
		LastUpdated: time.Now(),
		Packages:    ManagedPackages{},
		Files: []ManagedFile{
			{
				Name:        "vimrc",
				Destination: "/home/user/.vimrc",
				IsSymlink:   true,
				BackupPath:  "",
			},
			{
				Name:        "bashrc",
				Destination: "/home/user/.bashrc",
				IsSymlink:   false,
				BackupPath:  "/home/user/.bashrc.backup.20240101-120000",
			},
			{
				Name:        "removed-file",
				Destination: "/home/user/.removed",
				IsSymlink:   true,
				BackupPath:  "",
			},
		},
	}

	// Save initial state
	if err := sm.SaveState(initialState); err != nil {
		t.Fatalf("SaveState() failed: %v", err)
	}

	// Create new configuration (missing some files)
	newCfg := &config.Config{
		Files: map[string]config.File{
			"vimrc":  {Source: "dotfiles/.vimrc", Destination: "~/.vimrc"},
			"bashrc": {Source: "dotfiles/.bashrc", Destination: "~/.bashrc"},
			// "removed-file" is missing
		},
	}

	// Get files to remove
	toRemove, err := sm.GetFilesToRemove(newCfg)
	if err != nil {
		t.Fatalf("GetFilesToRemove() failed: %v", err)
	}

	// Verify files to remove
	if len(toRemove) != 1 {
		t.Errorf("Expected 1 file to remove, got %d", len(toRemove))
	}

	if toRemove[0].Name != "removed-file" {
		t.Errorf("Expected to remove 'removed-file', got %s", toRemove[0].Name)
	}
}

func TestStateManager_GetFilesToRemove_EmptyConfig(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	logger := log.New(os.Stderr)
	sm := NewStateManagerWithPath(logger, statePath)

	// Create initial state with files
	initialState := &PackageState{
		Version:     "1.0",
		LastUpdated: time.Now(),
		Packages:    ManagedPackages{},
		Files: []ManagedFile{
			{
				Name:        "file1",
				Destination: "/home/user/.file1",
				IsSymlink:   true,
			},
			{
				Name:        "file2",
				Destination: "/home/user/.file2",
				IsSymlink:   false,
			},
		},
	}

	// Save initial state
	if err := sm.SaveState(initialState); err != nil {
		t.Fatalf("SaveState() failed: %v", err)
	}

	// Create configuration with no files
	newCfg := &config.Config{
		Files: map[string]config.File{},
	}

	// Get files to remove
	toRemove, err := sm.GetFilesToRemove(newCfg)
	if err != nil {
		t.Fatalf("GetFilesToRemove() failed: %v", err)
	}

	// All files should be removed
	if len(toRemove) != 2 {
		t.Errorf("Expected 2 files to remove, got %d", len(toRemove))
	}

	fileNames := make([]string, len(toRemove))
	for i, file := range toRemove {
		fileNames[i] = file.Name
	}

	expectedNames := []string{"file1", "file2"}
	if !stringSlicesEqualUnordered(fileNames, expectedNames) {
		t.Errorf("Expected to remove %v, got %v", expectedNames, fileNames)
	}
}

func TestExtractPackageNames(t *testing.T) {
	packages := []config.PackageEntry{
		{Name: "vim"},
		{Name: "git"},
		{Name: "code"},
	}

	expected := []string{"vim", "git", "code"}
	result := extractPackageNames(packages)

	if !stringSlicesEqual(result, expected) {
		t.Errorf("extractPackageNames() = %v, expected %v", result, expected)
	}
}

func TestStateManager_InvalidJSON(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Write invalid JSON to state file
	invalidJSON := `{"version": "1.0", "packages": invalid}`
	if err := os.WriteFile(statePath, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	logger := log.New(os.Stderr)
	sm := NewStateManagerWithPath(logger, statePath)

	// Loading should fail with invalid JSON
	_, err := sm.LoadState()
	if err == nil {
		t.Errorf("Expected error when loading invalid JSON, got nil")
	}
}

// Helper functions
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func stringSlicesEqualUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aMap := make(map[string]int)
	bMap := make(map[string]int)

	for _, v := range a {
		aMap[v]++
	}
	for _, v := range b {
		bMap[v]++
	}

	for k, v := range aMap {
		if bMap[k] != v {
			return false
		}
	}

	return true
}