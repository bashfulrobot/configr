package pkg

import (
	"os"
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
			name: "apt ppa repository - dry run",
			repositories: config.RepositoryManagement{
				Apt: []config.AptRepository{
					{Name: "python39", PPA: "deadsnakes/ppa"},
				},
			},
			dryRun:      true,
			expectError: false,
		},
		{
			name: "apt custom repository with key - dry run",
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

func TestRepositoryManager_AddAptRepository_DryRun(t *testing.T) {
	logger := log.New(os.Stderr)
	logger.SetLevel(log.FatalLevel) // Silence logs during tests

	tests := []struct {
		name        string
		repo        config.AptRepository
		expectError bool
	}{
		{
			name:        "ppa repository",
			repo:        config.AptRepository{Name: "python39", PPA: "deadsnakes/ppa"},
			expectError: false,
		},
		{
			name: "custom repository with https key",
			repo: config.AptRepository{
				Name: "docker",
				URI:  "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable",
				Key:  "https://download.docker.com/linux/ubuntu/gpg.asc",
			},
			expectError: false,
		},
		{
			name: "custom repository with keyserver key",
			repo: config.AptRepository{
				Name: "nodejs",
				URI:  "deb https://deb.nodesource.com/node_16.x focal main",
				Key:  "0x9FD3B784BC1C6FC31A8A0A1C1655A0AB68576280",
			},
			expectError: false,
		},
		{
			name:        "repository with no ppa or uri",
			repo:        config.AptRepository{Name: "invalid"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := NewRepositoryManager(logger, true) // Always dry run for unit tests
			err := rm.addAptRepository(tt.repo)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
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
	t.Run("check add-apt-repository", func(t *testing.T) {
		err := rm.checkAddAptRepositoryAvailable()
		// Don't fail the test if command is not available, just log it
		if err != nil {
			t.Logf("add-apt-repository not available: %v", err)
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