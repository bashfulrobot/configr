package pkg

import (
	"os"
	"testing"
	"time"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

func TestStateManager_NewStateManager(t *testing.T) {
	logger := log.New(os.Stderr)
	sm := NewStateManager(logger)
	
	if sm == nil {
		t.Fatal("Expected StateManager, got nil")
	}
}

func TestInteractiveManager_NewInteractiveManager(t *testing.T) {
	logger := log.New(os.Stderr)
	im := NewInteractiveManager(logger)
	
	if im == nil {
		t.Fatal("Expected InteractiveManager, got nil")
	}
}

func TestInteractiveManager_IsInteractiveMode(t *testing.T) {
	logger := log.New(os.Stderr)
	im := NewInteractiveManager(logger)
	
	// In test environment, should return false
	result := im.IsInteractiveMode()
	if result {
		t.Log("Interactive mode detected")
	}
}

func TestCacheManager_NewCacheManager(t *testing.T) {
	logger := log.New(os.Stderr)
	cm := NewCacheManager(logger)
	
	if cm == nil {
		t.Fatal("Expected CacheManager, got nil")
	}
}

func TestOptimizedLoader_NewOptimizedLoader(t *testing.T) {
	logger := log.New(os.Stderr)
	cm := NewCacheManager(logger)
	ol := NewOptimizedLoader(logger, cm)
	
	if ol == nil {
		t.Fatal("Expected OptimizedLoader, got nil")
	}
}

func TestOptimizedAptManager_NewOptimizedAptManager(t *testing.T) {
	logger := log.New(os.Stderr)
	cm := NewCacheManager(logger)
	apt := NewOptimizedAptManager(logger, false, cm)
	
	if apt == nil {
		t.Fatal("Expected OptimizedAptManager, got nil")
	}
}

func TestUXManager_NewUXManager(t *testing.T) {
	logger := log.New(os.Stderr)
	ux := NewUXManager(logger, false)
	
	if ux == nil {
		t.Fatal("Expected UXManager, got nil")
	}
}

func TestUXManager_IsInteractiveTerminal(t *testing.T) {
	logger := log.New(os.Stderr)
	ux := NewUXManager(logger, false)
	
	// Test basic functionality
	result := ux.IsInteractiveTerminal()
	t.Logf("Interactive terminal: %v", result)
}

func TestUXManager_FormatValidationSummary(t *testing.T) {
	logger := log.New(os.Stderr)
	ux := NewUXManager(logger, false)
	
	result := &config.ValidationResult{
		Valid: true,
		Errors: []config.ValidationError{},
		Warnings: []config.ValidationError{},
	}
	
	summary := ux.FormatValidationSummary(result)
	if summary == "" {
		t.Log("Empty summary for valid config")
	}
}

func TestUXManager_FormatValidationSummaryCompact(t *testing.T) {
	logger := log.New(os.Stderr)
	ux := NewUXManager(logger, false)
	
	result := &config.ValidationResult{
		Valid: true,
		Errors: []config.ValidationError{},
		Warnings: []config.ValidationError{},
	}
	
	summary := ux.FormatValidationSummaryCompact(result)
	if summary == "" {
		t.Log("Empty compact summary for valid config")
	}
}

func TestUXManager_SimulateProgress(t *testing.T) {
	logger := log.New(os.Stderr)
	ux := NewUXManager(logger, false)
	
	start := time.Now()
	ux.SimulateProgress("Testing...", 10, 50*time.Millisecond)
	duration := time.Since(start)
	
	if duration < 25*time.Millisecond {
		t.Error("Progress simulation too quick")
	}
}

func TestUXManager_SimulateSpinner(t *testing.T) {
	logger := log.New(os.Stderr)
	ux := NewUXManager(logger, false)
	
	start := time.Now()
	ux.SimulateSpinner("Loading...", 50*time.Millisecond, false)
	duration := time.Since(start)
	
	if duration < 25*time.Millisecond {
		t.Error("Spinner simulation too quick")
	}
}

func TestConflictResolution_Constants(t *testing.T) {
	// Test that conflict resolution constants are defined
	if ResolutionSkip == ResolutionOverwrite {
		t.Error("Conflict resolution constants should be different")
	}
	
	if ResolutionBackup == ResolutionViewDiff {
		t.Error("Conflict resolution constants should be different")
	}
}

func TestFileConflictInfo_Structure(t *testing.T) {
	info := FileConflictInfo{
		Name:            "test",
		SourcePath:      "src",
		DestinationPath: "dest",
		IsSymlink:       true,
		BackupEnabled:   true,
	}
	
	if info.Name != "test" {
		t.Error("FileConflictInfo Name not set correctly")
	}
	if info.SourcePath != "src" {
		t.Error("FileConflictInfo SourcePath not set correctly")
	}
	if !info.IsSymlink {
		t.Error("FileConflictInfo IsSymlink not set correctly")
	}
	if !info.BackupEnabled {
		t.Error("FileConflictInfo BackupEnabled not set correctly")
	}
}

func TestPackageState_Structure(t *testing.T) {
	ps := PackageState{
		Version:     "1.0",
		LastUpdated: time.Now(),
		Packages: ManagedPackages{
			Apt:     []string{"vim"},
			Flatpak: []string{"firefox"},
			Snap:    []string{"code"},
		},
	}
	
	if ps.Version != "1.0" {
		t.Error("PackageState Version not set correctly")
	}
	
	if len(ps.Packages.Apt) != 1 {
		t.Error("PackageState Apt packages not set correctly")
	}
	
	if len(ps.Packages.Flatpak) != 1 {
		t.Error("PackageState Flatpak packages not set correctly")
	}
	
	if len(ps.Packages.Snap) != 1 {
		t.Error("PackageState Snap packages not set correctly")
	}
}

func TestManagedFile_Structure(t *testing.T) {
	mf := ManagedFile{
		Name:        "vimrc",
		Destination: "/home/user/.vimrc",
		IsSymlink:   true,
		BackupPath:  "/home/user/.vimrc.backup",
	}
	
	if mf.Name != "vimrc" {
		t.Error("ManagedFile Name not set correctly")
	}
	
	if mf.Destination != "/home/user/.vimrc" {
		t.Error("ManagedFile Destination not set correctly")
	}
	
	if !mf.IsSymlink {
		t.Error("ManagedFile IsSymlink not set correctly")
	}
	
	if mf.BackupPath != "/home/user/.vimrc.backup" {
		t.Error("ManagedFile BackupPath not set correctly")
	}
}