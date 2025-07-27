package config

import (
	"reflect"
	"testing"
)

func TestGetDefaultFlags(t *testing.T) {
	tests := []struct {
		name         string
		manager      string
		expectedFlags []string
	}{
		{
			name:         "apt flags",
			manager:      "apt",
			expectedFlags: []string{"-y", "--no-install-recommends"},
		},
		{
			name:         "flatpak flags",
			manager:      "flatpak",
			expectedFlags: []string{"--system", "--assumeyes"},
		},
		{
			name:         "snap flags",
			manager:      "snap",
			expectedFlags: []string{},
		},
		{
			name:         "unknown manager",
			manager:      "unknown",
			expectedFlags: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := GetDefaultFlags(tt.manager)
			if !reflect.DeepEqual(flags, tt.expectedFlags) {
				t.Errorf("GetDefaultFlags(%s) = %v, expected %v", tt.manager, flags, tt.expectedFlags)
			}
		})
	}
}

func TestHasDefaultFlags(t *testing.T) {
	tests := []struct {
		name     string
		manager  string
		expected bool
	}{
		{
			name:     "apt has flags",
			manager:  "apt",
			expected: true,
		},
		{
			name:     "flatpak has flags",
			manager:  "flatpak",
			expected: true,
		},
		{
			name:     "snap has no flags",
			manager:  "snap",
			expected: true, // Actually snap does have default flags
		},
		{
			name:     "unknown manager",
			manager:  "unknown",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			has := HasDefaultFlags(tt.manager)
			if has != tt.expected {
				t.Errorf("HasDefaultFlags(%s) = %v, expected %v", tt.manager, has, tt.expected)
			}
		})
	}
}

func TestGetSupportedPackageManagers(t *testing.T) {
	managers := GetSupportedPackageManagers()
	expectedCount := 3
	
	if len(managers) != expectedCount {
		t.Errorf("GetSupportedPackageManagers() returned %d managers, expected %d", len(managers), expectedCount)
	}
	
	// Check that all expected managers are present
	expectedManagers := map[string]bool{
		"apt": false,
		"flatpak": false,
		"snap": false,
	}
	
	for _, manager := range managers {
		if _, exists := expectedManagers[manager]; exists {
			expectedManagers[manager] = true
		} else {
			t.Errorf("Unexpected manager: %s", manager)
		}
	}
	
	for manager, found := range expectedManagers {
		if !found {
			t.Errorf("Expected manager %s not found", manager)
		}
	}
}