package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML implements custom unmarshaling for PackageEntry
// Supports both simple string and complex nested formats:
//   Simple: - "package-name"
//   Complex: - "package-name":
//              flags: ["--flag1", "--flag2"]
func (pe *PackageEntry) UnmarshalYAML(node *yaml.Node) error {
	// Handle simple string format: - "package-name"
	if node.Kind == yaml.ScalarNode {
		pe.Name = node.Value
		pe.Flags = []string{}
		return nil
	}

	// Handle complex nested format: - "package-name": { flags: [...] }
	if node.Kind == yaml.MappingNode {
		if len(node.Content) != 2 {
			return fmt.Errorf("package entry must have exactly one key-value pair")
		}

		// First node is the package name (key)
		if node.Content[0].Kind != yaml.ScalarNode {
			return fmt.Errorf("package name must be a string")
		}
		pe.Name = node.Content[0].Value

		// Second node is the configuration (value)
		configNode := node.Content[1]
		
		// Initialize with empty flags
		pe.Flags = []string{}

		// If the value is a mapping, parse the configuration
		if configNode.Kind == yaml.MappingNode {
			var config struct {
				Flags []string `yaml:"flags,omitempty"`
			}
			if err := configNode.Decode(&config); err != nil {
				return fmt.Errorf("failed to decode package configuration for %s: %w", pe.Name, err)
			}
			pe.Flags = config.Flags
		}

		return nil
	}

	return fmt.Errorf("package entry must be either a string or a mapping")
}

// MarshalYAML implements custom marshaling for PackageEntry
// Outputs simple format if no flags, complex format if flags exist
func (pe PackageEntry) MarshalYAML() (interface{}, error) {
	// Simple format if no flags
	if len(pe.Flags) == 0 {
		return pe.Name, nil
	}

	// Complex format with flags
	return map[string]interface{}{
		pe.Name: map[string]interface{}{
			"flags": pe.Flags,
		},
	}, nil
}

// GetEffectiveFlags returns the flags that should be used for this package
// considering the three-tier hierarchy: internal defaults -> user defaults -> package flags
func (pe *PackageEntry) GetEffectiveFlags(packageManager string, userDefaults map[string][]string) []string {
	// Tier 3: If package has specific flags, use those (highest priority)
	if len(pe.Flags) > 0 {
		result := make([]string, len(pe.Flags))
		copy(result, pe.Flags)
		return result
	}

	// Tier 2: If user has defaults for this package manager, use those
	if userDefaults != nil {
		if userFlags, exists := userDefaults[packageManager]; exists {
			result := make([]string, len(userFlags))
			copy(result, userFlags)
			return result
		}
	}

	// Tier 1: Use internal defaults (lowest priority)
	return GetDefaultFlags(packageManager)
}

// HasCustomFlags returns true if this package entry has custom flags defined
func (pe *PackageEntry) HasCustomFlags() bool {
	return len(pe.Flags) > 0
}

// String returns a string representation of the package entry for debugging
func (pe PackageEntry) String() string {
	if len(pe.Flags) == 0 {
		return pe.Name
	}
	return fmt.Sprintf("%s (flags: %v)", pe.Name, pe.Flags)
}