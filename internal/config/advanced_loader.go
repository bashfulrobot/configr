package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// AdvancedLoader handles loading configurations with advanced include features
type AdvancedLoader struct {
	visited   map[string]bool
	hostname  string
	osName    string
}

// NewAdvancedLoader creates a new advanced configuration loader
func NewAdvancedLoader() *AdvancedLoader {
	hostname, _ := os.Hostname()
	return &AdvancedLoader{
		visited:  make(map[string]bool),
		hostname: hostname,
		osName:   runtime.GOOS,
	}
}

// LoadConfigurationAdvanced loads configuration with advanced include support
func (al *AdvancedLoader) LoadConfigurationAdvanced(configPath string) (*Config, []string, error) {
	al.visited = make(map[string]bool) // Reset visited map
	config, paths, err := al.loadConfigRecursiveAdvanced(configPath, []string{})
	return config, paths, err
}

// loadConfigRecursiveAdvanced loads a config file and processes both simple and advanced includes
func (al *AdvancedLoader) loadConfigRecursiveAdvanced(configPath string, loadedPaths []string) (*Config, []string, error) {
	// Prevent circular includes
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get absolute path for %s: %w", configPath, err)
	}

	if al.visited[absPath] {
		return nil, nil, fmt.Errorf("circular include detected: %s", absPath)
	}
	al.visited[absPath] = true
	defer func() { delete(al.visited, absPath) }()

	// Load the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal config file %s: %w", configPath, err)
	}

	loadedPaths = append(loadedPaths, absPath)
	baseDir := filepath.Dir(configPath)

	// Process includes with advanced features
	if len(config.Includes) > 0 {
		for _, includeSpec := range config.Includes {
			// Check conditions first
			if !al.evaluateConditions(includeSpec.Conditions) {
				continue // Skip this include
			}

			// Resolve include path (supports glob patterns)
			includePaths, err := al.resolveIncludePathAdvanced(baseDir, includeSpec.Path, includeSpec.Optional)

			if err != nil {
				if includeSpec.Optional {
					continue // Skip optional includes that fail
				}
				return nil, nil, fmt.Errorf("failed to resolve include %s: %w", includeSpec.Path, err)
			}

			for _, resolvedPath := range includePaths {
				includedConfig, paths, err := al.loadConfigRecursiveAdvanced(resolvedPath, loadedPaths)
				if err != nil {
					if includeSpec.Optional {
						continue // Skip optional includes that fail to load
					}
					return nil, nil, fmt.Errorf("failed to load included config %s: %w", resolvedPath, err)
				}

				loadedPaths = paths
				if err := mergeConfigs(&config, includedConfig); err != nil {
					return nil, nil, fmt.Errorf("failed to merge config %s: %w", resolvedPath, err)
				}
			}
		}
	}

	return &config, loadedPaths, nil
}

// resolveIncludePathAdvanced resolves include paths with enhanced features
func (al *AdvancedLoader) resolveIncludePathAdvanced(baseDir, includePath string, optional bool) ([]string, error) {
	// Check for glob patterns
	if strings.Contains(includePath, "*") || strings.Contains(includePath, "?") || strings.Contains(includePath, "[") {
		return al.resolveGlobPattern(baseDir, includePath, optional)
	}

	// Use existing resolveIncludePath for non-glob patterns
	resolvedPath, err := resolveIncludePath(baseDir, includePath)
	if err != nil {
		if optional {
			return []string{}, nil
		}
		return nil, err
	}

	return []string{resolvedPath}, nil
}

// resolveGlobPattern resolves glob patterns for include files
func (al *AdvancedLoader) resolveGlobPattern(baseDir, pattern string, optional bool) ([]string, error) {
	fullPattern := filepath.Join(baseDir, pattern)
	
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern %s: %w", pattern, err)
	}

	if len(matches) == 0 {
		if optional {
			return []string{}, nil
		}
		return nil, fmt.Errorf("no files match glob pattern: %s", pattern)
	}

	// Filter out directories unless they contain default.yaml
	var validPaths []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}

		if info.IsDir() {
			// Check for default.yaml in directory
			defaultPath := filepath.Join(match, "default.yaml")
			if _, err := os.Stat(defaultPath); err == nil {
				validPaths = append(validPaths, defaultPath)
			}
		} else if strings.HasSuffix(strings.ToLower(match), ".yaml") || strings.HasSuffix(strings.ToLower(match), ".yml") {
			validPaths = append(validPaths, match)
		}
	}

	if len(validPaths) == 0 && !optional {
		return nil, fmt.Errorf("no valid YAML files found for pattern: %s", pattern)
	}

	return validPaths, nil
}

// evaluateConditions evaluates all conditions and returns true if all pass
func (al *AdvancedLoader) evaluateConditions(conditions []IncludeCondition) bool {
	if len(conditions) == 0 {
		return true // No conditions means always include
	}

	for _, condition := range conditions {
		if !al.evaluateCondition(condition) {
			return false // All conditions must pass
		}
	}

	return true
}

// evaluateCondition evaluates a single condition
func (al *AdvancedLoader) evaluateCondition(condition IncludeCondition) bool {
	operator := condition.Operator
	if operator == "" {
		operator = "equals" // Default operator
	}

	var actualValue string

	switch condition.Type {
	case "os":
		actualValue = al.osName
	case "hostname":
		actualValue = al.hostname
	case "env":
		// For env conditions, the Value should be "VAR_NAME=expected_value"
		if strings.Contains(condition.Value, "=") {
			parts := strings.SplitN(condition.Value, "=", 2)
			varName := parts[0]
			expectedValue := parts[1]
			actualValue = os.Getenv(varName)
			return al.compareValues(actualValue, expectedValue, operator)
		}
		// If no =, treat as checking if env var exists
		varName := condition.Value
		_, exists := os.LookupEnv(varName)
		return exists
	case "file_exists":
		// Check if file exists
		_, err := os.Stat(condition.Value)
		return err == nil
	case "dir_exists":
		// Check if directory exists
		info, err := os.Stat(condition.Value)
		return err == nil && info.IsDir()
	default:
		return false // Unknown condition type
	}

	return al.compareValues(actualValue, condition.Value, operator)
}

// compareValues compares actual and expected values using the specified operator
func (al *AdvancedLoader) compareValues(actual, expected, operator string) bool {
	switch operator {
	case "equals":
		return actual == expected
	case "contains":
		return strings.Contains(actual, expected)
	case "matches":
		matched, err := regexp.MatchString(expected, actual)
		return err == nil && matched
	case "not_equals":
		return actual != expected
	case "not_contains":
		return !strings.Contains(actual, expected)
	default:
		return actual == expected // Default to equals
	}
}

// ValidateIncludeSpec validates an include specification
func (al *AdvancedLoader) ValidateIncludeSpec(spec IncludeSpec) error {
	// Path must be specified
	if spec.Path == "" {
		return fmt.Errorf("include spec must have 'path' specified")
	}

	// Validate conditions
	for i, condition := range spec.Conditions {
		if err := al.validateCondition(condition); err != nil {
			return fmt.Errorf("invalid condition %d: %w", i, err)
		}
	}

	return nil
}

// validateCondition validates a single condition
func (al *AdvancedLoader) validateCondition(condition IncludeCondition) error {
	if condition.Type == "" {
		return fmt.Errorf("condition type is required")
	}

	validTypes := []string{"os", "hostname", "env", "file_exists", "dir_exists"}
	typeValid := false
	for _, validType := range validTypes {
		if condition.Type == validType {
			typeValid = true
			break
		}
	}

	if !typeValid {
		return fmt.Errorf("invalid condition type: %s (valid types: %s)", condition.Type, strings.Join(validTypes, ", "))
	}

	if condition.Value == "" {
		return fmt.Errorf("condition value is required")
	}

	// Validate operators
	if condition.Operator != "" {
		validOperators := []string{"equals", "contains", "matches", "not_equals", "not_contains"}
		operatorValid := false
		for _, validOp := range validOperators {
			if condition.Operator == validOp {
				operatorValid = true
				break
			}
		}

		if !operatorValid {
			return fmt.Errorf("invalid operator: %s (valid operators: %s)", condition.Operator, strings.Join(validOperators, ", "))
		}
	}

	return nil
}

// GetSystemInfo returns information about the current system for debugging
func (al *AdvancedLoader) GetSystemInfo() map[string]string {
	info := make(map[string]string)
	info["os"] = al.osName
	info["hostname"] = al.hostname
	info["goos"] = runtime.GOOS
	info["goarch"] = runtime.GOARCH
	
	// Add some common environment variables
	commonVars := []string{"USER", "HOME", "PATH", "SHELL", "DESKTOP_SESSION"}
	for _, varName := range commonVars {
		if value := os.Getenv(varName); value != "" {
			info["env_"+strings.ToLower(varName)] = value
		}
	}

	return info
}