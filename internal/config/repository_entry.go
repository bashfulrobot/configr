package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML implements custom unmarshaling for RepositoryManagement
// Supports map format: apt: { repo-name: { ppa: "..." } }
func (rm *RepositoryManagement) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("repositories must be a mapping")
	}

	// Initialize slices
	rm.Apt = []AptRepository{}
	rm.Flatpak = []FlatpakRepository{}

	// Process each key-value pair
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		if keyNode.Kind != yaml.ScalarNode {
			continue
		}

		switch keyNode.Value {
		case "apt":
			if err := rm.unmarshalAptRepositories(valueNode); err != nil {
				return fmt.Errorf("failed to unmarshal apt repositories: %w", err)
			}
		case "flatpak":
			if err := rm.unmarshalFlatpakRepositories(valueNode); err != nil {
				return fmt.Errorf("failed to unmarshal flatpak repositories: %w", err)
			}
		}
	}

	return nil
}

func (rm *RepositoryManagement) unmarshalAptRepositories(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("apt repositories must be a mapping")
	}

	for i := 0; i < len(node.Content); i += 2 {
		nameNode := node.Content[i]
		configNode := node.Content[i+1]

		if nameNode.Kind != yaml.ScalarNode {
			continue
		}

		var repo AptRepository
		repo.Name = nameNode.Value

		if configNode.Kind == yaml.MappingNode {
			var config struct {
				PPA string `yaml:"ppa,omitempty"`
				URI string `yaml:"uri,omitempty"`
				Key string `yaml:"key,omitempty"`
			}
			if err := configNode.Decode(&config); err != nil {
				return fmt.Errorf("failed to decode apt repository %s: %w", repo.Name, err)
			}
			repo.PPA = config.PPA
			repo.URI = config.URI
			repo.Key = config.Key
		}

		rm.Apt = append(rm.Apt, repo)
	}

	return nil
}

func (rm *RepositoryManagement) unmarshalFlatpakRepositories(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("flatpak repositories must be a mapping")
	}

	for i := 0; i < len(node.Content); i += 2 {
		nameNode := node.Content[i]
		configNode := node.Content[i+1]

		if nameNode.Kind != yaml.ScalarNode {
			continue
		}

		var repo FlatpakRepository
		repo.Name = nameNode.Value

		if configNode.Kind == yaml.MappingNode {
			var config struct {
				URL  string `yaml:"url"`
				User bool   `yaml:"user,omitempty"`
			}
			if err := configNode.Decode(&config); err != nil {
				return fmt.Errorf("failed to decode flatpak repository %s: %w", repo.Name, err)
			}
			repo.URL = config.URL
			repo.User = config.User
		}

		rm.Flatpak = append(rm.Flatpak, repo)
	}

	return nil
}