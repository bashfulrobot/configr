package pkg

import (
	"os"
	"strings"
	"testing"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

func TestRepositoryManager_AddRepositories(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests

	tests := []struct {
		name         string
		repositories config.RepositoryManagement
		dryRun       bool
		expectError  bool
	}{
		{
			name: "empty repositories",
			repositories: config.RepositoryManagement{
				Apt:     []config.AptRepository{},
				Flatpak: []config.FlatpakRepository{},
			},
			dryRun:      true,
			expectError: false,
		},
		{
			name: "apt ppa repository (legacy) - dry run",
			repositories: config.RepositoryManagement{
				Apt: []config.AptRepository{
					{Name: "python39", PPA: "deadsnakes/ppa"},
				},
			},
			dryRun:      true,
			expectError: false,
		},
		{
			name: "apt legacy repository with key - dry run",
			repositories: config.RepositoryManagement{
				Apt: []config.AptRepository{
					{
						Name: "docker",
						URI:  "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable",
						Key:  "https://download.docker.com/linux/ubuntu/gpg.asc",
					},
				},
			},
			dryRun:      true,
			expectError: false,
		},
		{
			name: "apt deb822 format repository - dry run",
			repositories: config.RepositoryManagement{
				Apt: []config.AptRepository{
					{
						Name:         "vscode",
						URIs:         []string{"https://packages.microsoft.com/repos/code"},
						Suites:       []string{"stable"},
						Components:   []string{"main"},
						Types:        []string{"deb"},
						Architectures: []string{"amd64", "arm64", "armhf"},
						KeyURL:       "https://packages.microsoft.com/keys/microsoft.asc",
						SignedBy:     "/usr/share/keyrings/vscode.gpg",
					},
				},
			},
			dryRun:      true,
			expectError: false,
		},
		{
			name: "flatpak repository - dry run",
			repositories: config.RepositoryManagement{
				Flatpak: []config.FlatpakRepository{
					{
						Name: "flathub",
						URL:  "https://flathub.org/repo/flathub.flatpakrepo",
						User: false,
					},
				},
			},
			dryRun:      true,
			expectError: false,
		},
		{
			name: "mixed repositories - dry run",
			repositories: config.RepositoryManagement{
				Apt: []config.AptRepository{
					{Name: "python39", PPA: "deadsnakes/ppa"},
					{
						Name: "nodejs",
						URI:  "deb https://deb.nodesource.com/node_16.x focal main",
						Key:  "0x9FD3B784BC1C6FC31A8A0A1C1655A0AB68576280",
					},
					{
						Name:         "chrome",
						URIs:         []string{"https://dl.google.com/linux/chrome/deb/"},
						Suites:       []string{"stable"},
						Components:   []string{"main"},
						KeyID:        "0xEB4C1BFD4F042F6DDDCCEC917721F63BD38B4796",
						SignedBy:     "/usr/share/keyrings/chrome.gpg",
					},
				},
				Flatpak: []config.FlatpakRepository{
					{Name: "flathub", URL: "https://flathub.org/repo/flathub.flatpakrepo", User: false},
					{Name: "kde", URL: "https://distribute.kde.org/kdeapps.flatpakrepo", User: true},
				},
			},
			dryRun:      true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := NewRepositoryManager(logger, tt.dryRun)
			err := rm.AddRepositories(tt.repositories)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRepositoryManager_AddAptRepositoryDEB822_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests

	tests := []struct {
		name        string
		repo        config.AptRepository
		expectError bool
	}{
		{
			name:        "legacy ppa repository",
			repo:        config.AptRepository{Name: "python39", PPA: "deadsnakes/ppa"},
			expectError: false,
		},
		{
			name: "legacy custom repository with https key",
			repo: config.AptRepository{
				Name: "docker",
				URI:  "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable",
				Key:  "https://download.docker.com/linux/ubuntu/gpg.asc",
			},
			expectError: false,
		},
		{
			name: "legacy custom repository with keyserver key",
			repo: config.AptRepository{
				Name: "nodejs",
				URI:  "deb https://deb.nodesource.com/node_16.x focal main",
				Key:  "0x9FD3B784BC1C6FC31A8A0A1C1655A0AB68576280",
			},
			expectError: false,
		},
		{
			name: "deb822 format repository",
			repo: config.AptRepository{
				Name:         "vscode",
				URIs:         []string{"https://packages.microsoft.com/repos/code"},
				Suites:       []string{"stable"},
				Components:   []string{"main"},
				Types:        []string{"deb"},
				Architectures: []string{"amd64", "arm64"},
				KeyURL:       "https://packages.microsoft.com/keys/microsoft.asc",
				SignedBy:     "/usr/share/keyrings/vscode.gpg",
			},
			expectError: false,
		},
		{
			name: "deb822 with key id",
			repo: config.AptRepository{
				Name:       "chrome",
				URIs:       []string{"https://dl.google.com/linux/chrome/deb/"},
				Suites:     []string{"stable"},
				Components: []string{"main"},
				KeyID:      "0xEB4C1BFD4F042F6DDDCCEC917721F63BD38B4796",
				SignedBy:   "/usr/share/keyrings/chrome.gpg",
			},
			expectError: false,
		},
		{
			name:        "repository with no configuration",
			repo:        config.AptRepository{Name: "invalid"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := NewRepositoryManager(logger, true) // Always dry run for unit tests
			err := rm.addAptRepositoryDEB822(tt.repo)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRepositoryManager_ConvertLegacyToRepository(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	rm := NewRepositoryManager(logger, true)

	tests := []struct {
		name           string
		input          config.AptRepository
		expectError    bool
		expectedURIs   []string
		expectedSuites []string
	}{
		{
			name:           "ppa conversion",
			input:          config.AptRepository{Name: "test", PPA: "deadsnakes/ppa"},
			expectError:    false,
			expectedURIs:   []string{"https://ppa.launchpadcontent.net/deadsnakes/ppa/ubuntu"},
			expectedSuites: []string{"noble"}, // Will vary based on system, just check conversion works
		},
		{
			name: "legacy uri conversion",
			input: config.AptRepository{
				Name: "test",
				URI:  "deb [arch=amd64] https://example.com/repo stable main",
			},
			expectError:    false,
			expectedURIs:   []string{"https://example.com/repo"},
			expectedSuites: []string{"stable"},
		},
		{
			name: "already deb822 format",
			input: config.AptRepository{
				Name:       "test",
				URIs:       []string{"https://example.com/repo"},
				Suites:     []string{"stable"},
				Components: []string{"main"},
			},
			expectError:    false,
			expectedURIs:   []string{"https://example.com/repo"},
			expectedSuites: []string{"stable"},
		},
		{
			name:        "missing configuration",
			input:       config.AptRepository{Name: "test"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := rm.convertLegacyToRepository(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// For PPA conversion, we can't predict the exact codename, so just check structure
			if tt.input.PPA != "" {
				if len(result.URIs) == 0 || !strings.Contains(result.URIs[0], "ppa.launchpadcontent.net") {
					t.Errorf("PPA conversion failed, got URIs: %v", result.URIs)
				}
				if len(result.Suites) == 0 {
					t.Errorf("PPA conversion failed, no suites set")
				}
				return
			}

			// For other tests, check exact values
			if len(tt.expectedURIs) > 0 && !equalStringSlices(result.URIs, tt.expectedURIs) {
				t.Errorf("expected URIs %v, got %v", tt.expectedURIs, result.URIs)
			}
			if len(tt.expectedSuites) > 0 && !equalStringSlices(result.Suites, tt.expectedSuites) {
				t.Errorf("expected Suites %v, got %v", tt.expectedSuites, result.Suites)
			}
		})
	}
}

func TestRepositoryManager_GenerateDEB822Content(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	rm := NewRepositoryManager(logger, true)

	repo := config.AptRepository{
		Name:         "vscode",
		URIs:         []string{"https://packages.microsoft.com/repos/code"},
		Suites:       []string{"stable"},
		Components:   []string{"main"},
		Types:        []string{"deb"},
		Architectures: []string{"amd64", "arm64", "armhf"},
		SignedBy:     "/usr/share/keyrings/vscode.gpg",
	}

	content := rm.generateDEB822Content(repo)

	// Check that content contains required fields
	expectedLines := []string{
		"Types: deb",
		"URIs: https://packages.microsoft.com/repos/code",
		"Suites: stable",
		"Components: main",
		"Architectures: amd64,arm64,armhf",
		"Signed-By: /usr/share/keyrings/vscode.gpg",
	}

	for _, expected := range expectedLines {
		if !strings.Contains(content, expected) {
			t.Errorf("expected content to contain '%s', got:\n%s", expected, content)
		}
	}

	// Check header comment is present
	if !strings.Contains(content, "### THIS FILE IS AUTOMATICALLY CONFIGURED ###") {
		t.Errorf("expected content to contain header comment")
	}
}

// Helper function to compare string slices
func equalStringSlices(a, b []string) bool {
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

func TestRepositoryManager_AddFlatpakRepository_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests

	tests := []struct {
		name        string
		repo        config.FlatpakRepository
		expectError bool
	}{
		{
			name: "system repository",
			repo: config.FlatpakRepository{
				Name: "flathub",
				URL:  "https://flathub.org/repo/flathub.flatpakrepo",
				User: false,
			},
			expectError: false,
		},
		{
			name: "user repository",
			repo: config.FlatpakRepository{
				Name: "kde",
				URL:  "https://distribute.kde.org/kdeapps.flatpakrepo",
				User: true,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := NewRepositoryManager(logger, true) // Always dry run for unit tests
			err := rm.addFlatpakRepository(tt.repo)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRepositoryManager_CheckCommands(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	
	rm := NewRepositoryManager(logger, true)

	// Test command availability checks
	// Note: These may fail in CI environments without the tools installed
	t.Run("check gpg", func(t *testing.T) {
		err := rm.checkGPGAvailable()
		// Don't fail the test if command is not available, just log it
		if err != nil {
			t.Logf("gpg not available: %v", err)
		}
	})

	t.Run("check flatpak", func(t *testing.T) {
		err := rm.checkFlatpakAvailable()
		// Don't fail the test if command is not available, just log it
		if err != nil {
			t.Logf("flatpak not available: %v", err)
		}
	})
}

func TestRepositoryManager_SystemCompatibility(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests
	
	rm := NewRepositoryManager(logger, true)

	// Test Ubuntu version compatibility check (may not work in all test environments)
	t.Run("check ubuntu version", func(t *testing.T) {
		err := rm.checkUbuntuVersionCompatibility()
		// Don't fail the test if not running on Ubuntu, just log it
		if err != nil {
			t.Logf("Ubuntu version check: %v", err)
		}
	})

	// Test version support checker
	t.Run("version support checker", func(t *testing.T) {
		tests := []struct {
			version   string
			supported bool
		}{
			{"24.04", true},
			{"24.10", true},
			{"25.04", true},
			{"22.04", false},
			{"20.04", false},
			{"18.04", false},
		}

		for _, tt := range tests {
			result := rm.isVersionSupported(tt.version)
			if result != tt.supported {
				t.Errorf("version %s: expected supported=%v, got %v", tt.version, tt.supported, result)
			}
		}
	})
}