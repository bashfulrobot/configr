package pkg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

// StateManager handles tracking of packages managed by configr
type StateManager struct {
	logger   *log.Logger
	statePath string
}

// PackageState represents the state of packages, files, and binaries managed by configr
type PackageState struct {
	Version     string            `json:"version"`
	LastUpdated time.Time         `json:"last_updated"`
	Packages    ManagedPackages   `json:"packages"`
	Files       []ManagedFile     `json:"files"`
	Binaries    []ManagedBinary   `json:"binaries"`
}

// ManagedPackages tracks packages by manager type
type ManagedPackages struct {
	Apt     []string `json:"apt"`
	Flatpak []string `json:"flatpak"`
	Snap    []string `json:"snap"`
}

// ManagedFile represents a file managed by configr
type ManagedFile struct {
	Name        string `json:"name"`        // File identifier from YAML
	Destination string `json:"destination"` // Where the file was deployed
	IsSymlink   bool   `json:"is_symlink"`  // Whether it was deployed as symlink or copy
	BackupPath  string `json:"backup_path,omitempty"` // Path to backup file if created
}

// NewStateManager creates a new state manager
func NewStateManager(logger *log.Logger) *StateManager {
	// Default state file location: ~/.config/configr/state.json
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Warn("Could not determine home directory, using /tmp for state file", "error", err)
		homeDir = "/tmp"
	}
	
	configDir := filepath.Join(homeDir, ".config", "configr")
	statePath := filepath.Join(configDir, "state.json")
	
	return &StateManager{
		logger:    logger,
		statePath: statePath,
	}
}

// NewStateManagerWithPath creates a state manager with a custom state file path
func NewStateManagerWithPath(logger *log.Logger, statePath string) *StateManager {
	return &StateManager{
		logger:    logger,
		statePath: statePath,
	}
}

// LoadState loads the current package state from disk
func (sm *StateManager) LoadState() (*PackageState, error) {
	sm.logger.Debug("Loading package state", "path", sm.statePath)
	
	// If state file doesn't exist, return empty state
	if _, err := os.Stat(sm.statePath); os.IsNotExist(err) {
		sm.logger.Debug("State file does not exist, returning empty state")
		return &PackageState{
			Version:     "1.0",
			LastUpdated: time.Now(),
			Packages:    ManagedPackages{},
			Files:       []ManagedFile{},
			Binaries:    []ManagedBinary{},
		}, nil
	}
	
	data, err := os.ReadFile(sm.statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}
	
	var state PackageState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}
	
	sm.logger.Debug("Loaded package state", "apt_count", len(state.Packages.Apt), 
		"flatpak_count", len(state.Packages.Flatpak), "snap_count", len(state.Packages.Snap),
		"files_count", len(state.Files), "binaries_count", len(state.Binaries))
	
	return &state, nil
}

// SaveState saves the current package state to disk
func (sm *StateManager) SaveState(state *PackageState) error {
	sm.logger.Debug("Saving package state", "path", sm.statePath)
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(sm.statePath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Update timestamp
	state.LastUpdated = time.Now()
	
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	
	if err := os.WriteFile(sm.statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}
	
	sm.logger.Debug("Saved package state successfully")
	return nil
}

// UpdateState updates the state with current configuration packages, files, and binaries
func (sm *StateManager) UpdateState(cfg *config.Config, deployedFiles []ManagedFile) error {
	return sm.UpdateStateWithBinaries(cfg, deployedFiles, []ManagedBinary{})
}

// UpdateStateWithBinaries updates the state with current configuration packages, files, and binaries
func (sm *StateManager) UpdateStateWithBinaries(cfg *config.Config, deployedFiles []ManagedFile, deployedBinaries []ManagedBinary) error {
	state, err := sm.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load current state: %w", err)
	}
	
	// Extract package names from configuration
	state.Packages.Apt = extractPackageNames(cfg.Packages.Apt)
	state.Packages.Flatpak = extractPackageNames(cfg.Packages.Flatpak)
	state.Packages.Snap = extractPackageNames(cfg.Packages.Snap)
	
	// Update file state
	state.Files = deployedFiles
	
	// Update binary state
	state.Binaries = deployedBinaries
	
	return sm.SaveState(state)
}

// UpdatePackageState updates only the package state (for backward compatibility)
func (sm *StateManager) UpdatePackageState(cfg *config.Config) error {
	return sm.UpdateState(cfg, []ManagedFile{})
}

// GetPackagesToRemove compares current state with new configuration and returns packages to remove
func (sm *StateManager) GetPackagesToRemove(cfg *config.Config) (*ManagedPackages, error) {
	currentState, err := sm.LoadState()
	if err != nil {
		return nil, fmt.Errorf("failed to load current state: %w", err)
	}
	
	// Get package names from new configuration
	newApt := extractPackageNames(cfg.Packages.Apt)
	newFlatpak := extractPackageNames(cfg.Packages.Flatpak)
	newSnap := extractPackageNames(cfg.Packages.Snap)
	
	// Find packages to remove (in old state but not in new config)
	toRemove := &ManagedPackages{
		Apt:     stringSliceDiff(currentState.Packages.Apt, newApt),
		Flatpak: stringSliceDiff(currentState.Packages.Flatpak, newFlatpak),
		Snap:    stringSliceDiff(currentState.Packages.Snap, newSnap),
	}
	
	sm.logger.Debug("Determined packages to remove", 
		"apt", len(toRemove.Apt), "flatpak", len(toRemove.Flatpak), "snap", len(toRemove.Snap))
	
	if len(toRemove.Apt) > 0 {
		sm.logger.Debug("APT packages to remove", "packages", toRemove.Apt)
	}
	if len(toRemove.Flatpak) > 0 {
		sm.logger.Debug("Flatpak packages to remove", "packages", toRemove.Flatpak)
	}
	if len(toRemove.Snap) > 0 {
		sm.logger.Debug("Snap packages to remove", "packages", toRemove.Snap)
	}
	
	return toRemove, nil
}

// GetFilesToRemove compares current state with new configuration and returns files to remove
func (sm *StateManager) GetFilesToRemove(cfg *config.Config) ([]ManagedFile, error) {
	currentState, err := sm.LoadState()
	if err != nil {
		return nil, fmt.Errorf("failed to load current state: %w", err)
	}
	
	// Create a map of current configuration file names for quick lookup
	currentFiles := make(map[string]bool)
	for fileName := range cfg.Files {
		currentFiles[fileName] = true
	}
	
	// Find files in state that are not in current configuration
	var filesToRemove []ManagedFile
	for _, file := range currentState.Files {
		if !currentFiles[file.Name] {
			filesToRemove = append(filesToRemove, file)
		}
	}
	
	sm.logger.Debug("Determined files to remove", "count", len(filesToRemove))
	if len(filesToRemove) > 0 {
		fileNames := make([]string, len(filesToRemove))
		for i, file := range filesToRemove {
			fileNames[i] = file.Name
		}
		sm.logger.Debug("Files to remove", "files", fileNames)
	}
	
	return filesToRemove, nil
}

// GetBinariesToRemove compares current state with new configuration and returns binaries to remove
func (sm *StateManager) GetBinariesToRemove(cfg *config.Config) ([]ManagedBinary, error) {
	currentState, err := sm.LoadState()
	if err != nil {
		return nil, fmt.Errorf("failed to load current state: %w", err)
	}
	
	// Create a map of current configuration binary names for quick lookup
	currentBinaries := make(map[string]bool)
	for binaryName := range cfg.Binaries {
		currentBinaries[binaryName] = true
	}
	
	// Find binaries in state that are not in current configuration
	var binariesToRemove []ManagedBinary
	for _, binary := range currentState.Binaries {
		if !currentBinaries[binary.Name] {
			binariesToRemove = append(binariesToRemove, binary)
		}
	}
	
	sm.logger.Debug("Determined binaries to remove", "count", len(binariesToRemove))
	if len(binariesToRemove) > 0 {
		binaryNames := make([]string, len(binariesToRemove))
		for i, binary := range binariesToRemove {
			binaryNames[i] = binary.Name
		}
		sm.logger.Debug("Binaries to remove", "binaries", binaryNames)
	}
	
	return binariesToRemove, nil
}

// extractPackageNames extracts package names from PackageEntry slices
func extractPackageNames(packages []config.PackageEntry) []string {
	names := make([]string, len(packages))
	for i, pkg := range packages {
		names[i] = pkg.Name
	}
	return names
}

// stringSliceDiff returns elements in slice1 that are not in slice2
func stringSliceDiff(slice1, slice2 []string) []string {
	set2 := make(map[string]bool)
	for _, item := range slice2 {
		set2[item] = true
	}
	
	var diff []string
	for _, item := range slice1 {
		if !set2[item] {
			diff = append(diff, item)
		}
	}
	
	return diff
}