package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigWithPosition contains config data with line/column position information
type ConfigWithPosition struct {
	Config   *Config
	YAMLNode *yaml.Node
	FilePath string
}

// ParseConfigWithPosition parses YAML and retains position information for better error reporting
func ParseConfigWithPosition(configPath string) (*ConfigWithPosition, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var rootNode yaml.Node
	if err := yaml.Unmarshal(data, &rootNode); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", configPath, err)
	}

	var config Config
	if err := rootNode.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config in %s: %w", configPath, err)
	}

	return &ConfigWithPosition{
		Config:   &config,
		YAMLNode: &rootNode,
		FilePath: configPath,
	}, nil
}

// FindFieldPosition finds the line and column for a given field path
func (cp *ConfigWithPosition) FindFieldPosition(fieldPath string) (line, column int) {
	if cp.YAMLNode == nil || len(cp.YAMLNode.Content) == 0 {
		return 0, 0
	}

	// Split field path (e.g., "files.vimrc.source")
	parts := strings.Split(fieldPath, ".")
	
	// Start with the document root
	node := cp.YAMLNode.Content[0] // Document node
	
	for _, part := range parts {
		node = findMapValue(node, part)
		if node == nil {
			return 0, 0
		}
	}

	return node.Line, node.Column
}

// findMapValue finds a value in a YAML mapping node
func findMapValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}

	// YAML mapping nodes have alternating key-value content
	for i := 0; i < len(node.Content); i += 2 {
		if i+1 < len(node.Content) && node.Content[i].Value == key {
			return node.Content[i+1] // Return the value node
		}
	}

	return nil
}

// GetValueAtPosition gets the actual YAML value at a specific field position
func (cp *ConfigWithPosition) GetValueAtPosition(fieldPath string) string {
	parts := strings.Split(fieldPath, ".")
	node := cp.YAMLNode.Content[0] // Document root

	for _, part := range parts {
		node = findMapValue(node, part)
		if node == nil {
			return ""
		}
	}

	return node.Value
}

// ValidateFieldExists checks if a field path exists in the parsed YAML
func (cp *ConfigWithPosition) ValidateFieldExists(fieldPath string) bool {
	parts := strings.Split(fieldPath, ".")
	node := cp.YAMLNode.Content[0]

	for _, part := range parts {
		node = findMapValue(node, part)
		if node == nil {
			return false
		}
	}

	return true
}