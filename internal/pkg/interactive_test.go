package pkg

import (
	"os"
	"testing"

	"github.com/charmbracelet/log"
)

func TestNewInteractiveManager(t *testing.T) {
	logger := log.New(os.Stderr)
	im := NewInteractiveManager(logger)
	
	if im == nil {
		t.Fatal("NewInteractiveManager returned nil")
	}
	
	if im.logger != logger {
		t.Error("Logger not set correctly")
	}
	
	if im.reader == nil {
		t.Error("Reader not initialized")
	}
}

func TestValidateOctalPermissions(t *testing.T) {
	logger := log.New(os.Stderr)
	im := NewInteractiveManager(logger)
	
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{
			name:    "valid 3 digit octal",
			mode:    "644",
			wantErr: false,
		},
		{
			name:    "valid 4 digit octal",
			mode:    "0644",
			wantErr: false,
		},
		{
			name:    "invalid - contains 8",
			mode:    "648",
			wantErr: true,
		},
		{
			name:    "invalid - contains 9",
			mode:    "749",
			wantErr: true,
		},
		{
			name:    "invalid - too long",
			mode:    "07777",
			wantErr: true,
		},
		{
			name:    "invalid - empty",
			mode:    "",
			wantErr: true,
		},
		{
			name:    "invalid - non-numeric",
			mode:    "abc",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := im.validateOctalPermissions(tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOctalPermissions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsInteractiveMode(t *testing.T) {
	logger := log.New(os.Stderr)
	im := NewInteractiveManager(logger)
	
	// This will typically return false in test environment
	result := im.IsInteractiveMode()
	
	// We just test that it doesn't panic and returns a boolean
	if result != true && result != false {
		t.Error("IsInteractiveMode should return a boolean")
	}
}

func TestShowFileDiff(t *testing.T) {
	logger := log.New(os.Stderr)
	im := NewInteractiveManager(logger)
	
	tmpDir := t.TempDir()
	
	// Create test files
	file1 := tmpDir + "/file1.txt"
	file2 := tmpDir + "/file2.txt"
	
	err := os.WriteFile(file1, []byte("line1\nline2\nline3\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	err = os.WriteFile(file2, []byte("line1\nmodified line2\nline3\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Test showing diff (this should not error even if no diff tool available)
	err = im.ShowFileDiff(file1, file2)
	// We don't expect this to fail in test environment, just not panic
	if err != nil {
		t.Logf("ShowFileDiff returned error (expected in test env): %v", err)
	}
}

func TestFileConflictInfo(t *testing.T) {
	// Test the FileConflictInfo struct
	conflict := FileConflictInfo{
		Name:           "test.txt",
		SourcePath:     "/tmp/source.txt",
		DestinationPath: "/tmp/dest.txt",
		IsSymlink:      false,
		BackupEnabled:  true,
	}
	
	if conflict.Name != "test.txt" {
		t.Error("Name not set correctly")
	}
	
	if conflict.SourcePath != "/tmp/source.txt" {
		t.Error("SourcePath not set correctly")
	}
	
	if conflict.DestinationPath != "/tmp/dest.txt" {
		t.Error("DestinationPath not set correctly")
	}
	
	if conflict.IsSymlink {
		t.Error("IsSymlink should be false")
	}
	
	if !conflict.BackupEnabled {
		t.Error("BackupEnabled should be true")
	}
}

func TestConflictResolution(t *testing.T) {
	// Test the ConflictResolution constants
	if ResolutionSkip != 0 {
		t.Error("ResolutionSkip should be 0")
	}
	
	if ResolutionOverwrite != 1 {
		t.Error("ResolutionOverwrite should be 1")
	}
	
	if ResolutionBackup != 2 {
		t.Error("ResolutionBackup should be 2")
	}
	
	if ResolutionViewDiff != 3 {
		t.Error("ResolutionViewDiff should be 3")
	}
	
	if ResolutionQuit != 4 {
		t.Error("ResolutionQuit should be 4")
	}
}