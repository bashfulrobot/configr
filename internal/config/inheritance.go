package config

import (
	"fmt"
	"sort"
	"strings"
)

// InheritancePattern represents different inheritance strategies
type InheritancePattern int

const (
	InheritanceNone InheritancePattern = iota
	InheritanceOverride
	InheritanceMerge
	InheritanceAppend
	InheritancePrepend
)

// InheritanceRule defines how specific configuration sections should be inherited
type InheritanceRule struct {
	Section   string             `yaml:"section"`   // Configuration section (packages.apt, files, etc.)
	Pattern   InheritancePattern `yaml:"pattern"`   // How to inherit
	Priority  int                `yaml:"priority"`  // Priority for conflict resolution
	Condition string             `yaml:"condition"` // Optional condition for rule application
}

// InheritanceConfig defines inheritance rules for configuration merging
type InheritanceConfig struct {
	Rules        []InheritanceRule `yaml:"rules"`
	DefaultRule  InheritancePattern `yaml:"default_rule"`
	AllowOverride bool              `yaml:"allow_override"`
}

// ConfigInheritanceManager handles configuration inheritance
type ConfigInheritanceManager struct {
	rules       map[string]InheritanceRule
	defaultRule InheritancePattern
	allowOverride bool
}

// NewConfigInheritanceManager creates a new inheritance manager
func NewConfigInheritanceManager() *ConfigInheritanceManager {
	return &ConfigInheritanceManager{
		rules:         make(map[string]InheritanceRule),
		defaultRule:   InheritanceMerge, // Safe default
		allowOverride: true,
	}
}

// SetInheritanceRules sets the inheritance rules
func (cim *ConfigInheritanceManager) SetInheritanceRules(config InheritanceConfig) {
	cim.defaultRule = config.DefaultRule
	cim.allowOverride = config.AllowOverride
	
	// Clear existing rules
	cim.rules = make(map[string]InheritanceRule)
	
	// Set new rules
	for _, rule := range config.Rules {
		cim.rules[rule.Section] = rule
	}
}

// GetBuiltinInheritanceRules returns sensible default inheritance rules
func (cim *ConfigInheritanceManager) GetBuiltinInheritanceRules() InheritanceConfig {
	return InheritanceConfig{
		DefaultRule:   InheritanceMerge,
		AllowOverride: true,
		Rules: []InheritanceRule{
			// Package management: Append packages from parent configs
			{Section: "packages.apt", Pattern: InheritanceAppend, Priority: 1},
			{Section: "packages.flatpak", Pattern: InheritanceAppend, Priority: 1},
			{Section: "packages.snap", Pattern: InheritanceAppend, Priority: 1},
			
			// Package defaults: Override (more specific configs take precedence)
			{Section: "package_defaults", Pattern: InheritanceOverride, Priority: 2},
			
			// Files: Override (child configs override parent file configurations)
			{Section: "files", Pattern: InheritanceOverride, Priority: 3},
			
			// DConf settings: Merge (combine settings from all levels)
			{Section: "dconf.settings", Pattern: InheritanceMerge, Priority: 1},
			
			// Repositories: Merge (combine repositories from all levels)
			{Section: "repositories.apt", Pattern: InheritanceMerge, Priority: 1},
			{Section: "repositories.flatpak", Pattern: InheritanceMerge, Priority: 1},
			
			// Backup policy: Override (most specific config wins)
			{Section: "backup_policy", Pattern: InheritanceOverride, Priority: 2},
			
			// Version: Override (child version takes precedence)
			{Section: "version", Pattern: InheritanceOverride, Priority: 3},
		},
	}
}

// MergeConfigsWithInheritance merges configurations using inheritance rules
func (cim *ConfigInheritanceManager) MergeConfigsWithInheritance(parent, child *Config) (*Config, error) {
	// Start with a copy of the child config
	result := cim.deepCopyConfig(child)
	
	// Apply inheritance rules for each section
	if err := cim.inheritPackages(&result.Packages, &parent.Packages); err != nil {
		return nil, fmt.Errorf("failed to inherit packages: %w", err)
	}
	
	if err := cim.inheritPackageDefaults(&result.PackageDefaults, parent.PackageDefaults); err != nil {
		return nil, fmt.Errorf("failed to inherit package defaults: %w", err)
	}
	
	if err := cim.inheritFiles(&result.Files, parent.Files); err != nil {
		return nil, fmt.Errorf("failed to inherit files: %w", err)
	}
	
	if err := cim.inheritDConfSettings(&result.DConf.Settings, parent.DConf.Settings); err != nil {
		return nil, fmt.Errorf("failed to inherit dconf settings: %w", err)
	}
	
	if err := cim.inheritRepositories(&result.Repositories, &parent.Repositories); err != nil {
		return nil, fmt.Errorf("failed to inherit repositories: %w", err)
	}
	
	if err := cim.inheritBackupPolicy(&result.BackupPolicy, parent.BackupPolicy); err != nil {
		return nil, fmt.Errorf("failed to inherit backup policy: %w", err)
	}
	
	if err := cim.inheritVersion(&result.Version, parent.Version); err != nil {
		return nil, fmt.Errorf("failed to inherit version: %w", err)
	}
	
	return result, nil
}

// inheritPackages handles package inheritance
func (cim *ConfigInheritanceManager) inheritPackages(child, parent *PackageManagement) error {
	// APT packages
	aptRule := cim.getRule("packages.apt")
	switch aptRule.Pattern {
	case InheritanceAppend:
		child.Apt = cim.appendUniquePackages(parent.Apt, child.Apt)
	case InheritancePrepend:
		child.Apt = cim.appendUniquePackages(child.Apt, parent.Apt)
	case InheritanceOverride:
		// Child already has precedence, nothing to do
	case InheritanceMerge:
		child.Apt = cim.mergePackages(parent.Apt, child.Apt)
	}
	
	// Flatpak packages
	flatpakRule := cim.getRule("packages.flatpak")
	switch flatpakRule.Pattern {
	case InheritanceAppend:
		child.Flatpak = cim.appendUniquePackages(parent.Flatpak, child.Flatpak)
	case InheritancePrepend:
		child.Flatpak = cim.appendUniquePackages(child.Flatpak, parent.Flatpak)
	case InheritanceOverride:
		// Child already has precedence, nothing to do
	case InheritanceMerge:
		child.Flatpak = cim.mergePackages(parent.Flatpak, child.Flatpak)
	}
	
	// Snap packages
	snapRule := cim.getRule("packages.snap")
	switch snapRule.Pattern {
	case InheritanceAppend:
		child.Snap = cim.appendUniquePackages(parent.Snap, child.Snap)
	case InheritancePrepend:
		child.Snap = cim.appendUniquePackages(child.Snap, parent.Snap)
	case InheritanceOverride:
		// Child already has precedence, nothing to do
	case InheritanceMerge:
		child.Snap = cim.mergePackages(parent.Snap, child.Snap)
	}
	
	return nil
}

// inheritPackageDefaults handles package defaults inheritance
func (cim *ConfigInheritanceManager) inheritPackageDefaults(child *map[string][]string, parent map[string][]string) error {
	rule := cim.getRule("package_defaults")
	
	if *child == nil {
		*child = make(map[string][]string)
	}
	
	switch rule.Pattern {
	case InheritanceOverride:
		// Child takes precedence, only inherit missing keys
		for key, value := range parent {
			if _, exists := (*child)[key]; !exists {
				(*child)[key] = make([]string, len(value))
				copy((*child)[key], value)
			}
		}
	case InheritanceMerge:
		// Merge values for each package manager
		for key, parentValues := range parent {
			if childValues, exists := (*child)[key]; exists {
				// Merge arrays
				merged := make([]string, 0, len(parentValues)+len(childValues))
				merged = append(merged, parentValues...)
				for _, childValue := range childValues {
					if !contains(merged, childValue) {
						merged = append(merged, childValue)
					}
				}
				(*child)[key] = merged
			} else {
				// Copy parent values
				(*child)[key] = make([]string, len(parentValues))
				copy((*child)[key], parentValues)
			}
		}
	case InheritanceAppend:
		// Append parent values to child values
		for key, parentValues := range parent {
			if childValues, exists := (*child)[key]; exists {
				for _, parentValue := range parentValues {
					if !contains(childValues, parentValue) {
						(*child)[key] = append((*child)[key], parentValue)
					}
				}
			} else {
				(*child)[key] = make([]string, len(parentValues))
				copy((*child)[key], parentValues)
			}
		}
	case InheritancePrepend:
		// Prepend parent values to child values
		for key, parentValues := range parent {
			if childValues, exists := (*child)[key]; exists {
				newValues := make([]string, 0, len(parentValues)+len(childValues))
				newValues = append(newValues, parentValues...)
				for _, childValue := range childValues {
					if !contains(newValues, childValue) {
						newValues = append(newValues, childValue)
					}
				}
				(*child)[key] = newValues
			} else {
				(*child)[key] = make([]string, len(parentValues))
				copy((*child)[key], parentValues)
			}
		}
	}
	
	return nil
}

// inheritFiles handles file inheritance
func (cim *ConfigInheritanceManager) inheritFiles(child *map[string]File, parent map[string]File) error {
	rule := cim.getRule("files")
	
	if *child == nil {
		*child = make(map[string]File)
	}
	
	switch rule.Pattern {
	case InheritanceOverride:
		// Child takes precedence, only inherit missing files
		for key, value := range parent {
			if _, exists := (*child)[key]; !exists {
				(*child)[key] = value
			}
		}
	case InheritanceMerge:
		// Merge file configurations (child properties override parent)
		for key, parentFile := range parent {
			if childFile, exists := (*child)[key]; exists {
				// Merge file properties
				merged := parentFile
				if childFile.Source != "" {
					merged.Source = childFile.Source
				}
				if childFile.Destination != "" {
					merged.Destination = childFile.Destination
				}
				if childFile.Owner != "" {
					merged.Owner = childFile.Owner
				}
				if childFile.Group != "" {
					merged.Group = childFile.Group
				}
				if childFile.Mode != "" {
					merged.Mode = childFile.Mode
				}
				// Boolean fields: child value takes precedence if explicitly set
				if childFile.Backup {
					merged.Backup = childFile.Backup
				}
				if childFile.Copy {
					merged.Copy = childFile.Copy
				}
				if childFile.Interactive {
					merged.Interactive = childFile.Interactive
				}
				(*child)[key] = merged
			} else {
				(*child)[key] = parentFile
			}
		}
	case InheritanceAppend, InheritancePrepend:
		// For files, append/prepend is the same as merge
		return cim.inheritFiles(child, parent) // Recursive call with merge logic
	}
	
	return nil
}

// inheritDConfSettings handles DConf settings inheritance
func (cim *ConfigInheritanceManager) inheritDConfSettings(child *map[string]string, parent map[string]string) error {
	rule := cim.getRule("dconf.settings")
	
	if *child == nil {
		*child = make(map[string]string)
	}
	
	switch rule.Pattern {
	case InheritanceOverride:
		// Child takes precedence, only inherit missing settings
		for key, value := range parent {
			if _, exists := (*child)[key]; !exists {
				(*child)[key] = value
			}
		}
	case InheritanceMerge, InheritanceAppend, InheritancePrepend:
		// All merge for settings (child overrides parent for same key)
		for key, value := range parent {
			if _, exists := (*child)[key]; !exists {
				(*child)[key] = value
			}
		}
	}
	
	return nil
}

// inheritRepositories handles repository inheritance
func (cim *ConfigInheritanceManager) inheritRepositories(child, parent *RepositoryManagement) error {
	aptRule := cim.getRule("repositories.apt")
	flatpakRule := cim.getRule("repositories.flatpak")
	
	// Initialize child slices if nil
	if child.Apt == nil {
		child.Apt = make([]AptRepository, 0)
	}
	if child.Flatpak == nil {
		child.Flatpak = make([]FlatpakRepository, 0)
	}
	
	// Inherit APT repositories
	switch aptRule.Pattern {
	case InheritanceOverride:
		// Add parent repos that don't exist in child
		for _, parentRepo := range parent.Apt {
			found := false
			for _, childRepo := range child.Apt {
				if childRepo.Name == parentRepo.Name {
					found = true
					break
				}
			}
			if !found {
				child.Apt = append(child.Apt, parentRepo)
			}
		}
	case InheritanceMerge, InheritanceAppend, InheritancePrepend:
		// Add all parent repos that don't exist in child
		for _, parentRepo := range parent.Apt {
			found := false
			for _, childRepo := range child.Apt {
				if childRepo.Name == parentRepo.Name {
					found = true
					break
				}
			}
			if !found {
				child.Apt = append(child.Apt, parentRepo)
			}
		}
	}
	
	// Inherit Flatpak repositories
	switch flatpakRule.Pattern {
	case InheritanceOverride:
		// Add parent repos that don't exist in child
		for _, parentRepo := range parent.Flatpak {
			found := false
			for _, childRepo := range child.Flatpak {
				if childRepo.Name == parentRepo.Name {
					found = true
					break
				}
			}
			if !found {
				child.Flatpak = append(child.Flatpak, parentRepo)
			}
		}
	case InheritanceMerge, InheritanceAppend, InheritancePrepend:
		// Add all parent repos that don't exist in child
		for _, parentRepo := range parent.Flatpak {
			found := false
			for _, childRepo := range child.Flatpak {
				if childRepo.Name == parentRepo.Name {
					found = true
					break
				}
			}
			if !found {
				child.Flatpak = append(child.Flatpak, parentRepo)
			}
		}
	}
	
	return nil
}

// inheritBackupPolicy handles backup policy inheritance
func (cim *ConfigInheritanceManager) inheritBackupPolicy(child *BackupPolicy, parent BackupPolicy) error {
	rule := cim.getRule("backup_policy")
	
	switch rule.Pattern {
	case InheritanceOverride:
		// Only inherit values that are not set in child
		if !child.AutoCleanup && parent.AutoCleanup {
			child.AutoCleanup = parent.AutoCleanup
		}
		if child.MaxAge == "" && parent.MaxAge != "" {
			child.MaxAge = parent.MaxAge
		}
		if child.MaxCount == 0 && parent.MaxCount > 0 {
			child.MaxCount = parent.MaxCount
		}
		if !child.CleanupOrphaned && parent.CleanupOrphaned {
			child.CleanupOrphaned = parent.CleanupOrphaned
		}
		if child.PreserveRecent == 0 && parent.PreserveRecent > 0 {
			child.PreserveRecent = parent.PreserveRecent
		}
	case InheritanceMerge:
		// Merge policies (child takes precedence for set values)
		if !child.AutoCleanup {
			child.AutoCleanup = parent.AutoCleanup
		}
		if child.MaxAge == "" {
			child.MaxAge = parent.MaxAge
		}
		if child.MaxCount == 0 {
			child.MaxCount = parent.MaxCount
		}
		if !child.CleanupOrphaned {
			child.CleanupOrphaned = parent.CleanupOrphaned
		}
		if child.PreserveRecent == 0 {
			child.PreserveRecent = parent.PreserveRecent
		}
	}
	
	return nil
}

// inheritVersion handles version inheritance
func (cim *ConfigInheritanceManager) inheritVersion(child *string, parent string) error {
	rule := cim.getRule("version")
	
	switch rule.Pattern {
	case InheritanceOverride:
		// Child version takes precedence
		if *child == "" && parent != "" {
			*child = parent
		}
	case InheritanceMerge:
		// Use parent version if child doesn't specify one
		if *child == "" {
			*child = parent
		}
	}
	
	return nil
}

// Helper methods

func (cim *ConfigInheritanceManager) getRule(section string) InheritanceRule {
	if rule, exists := cim.rules[section]; exists {
		return rule
	}
	// Return default rule
	return InheritanceRule{
		Section:  section,
		Pattern:  cim.defaultRule,
		Priority: 1,
	}
}

func (cim *ConfigInheritanceManager) appendUniquePackages(first, second []PackageEntry) []PackageEntry {
	result := make([]PackageEntry, len(first))
	copy(result, first)
	
	existingNames := make(map[string]bool)
	for _, pkg := range first {
		existingNames[pkg.Name] = true
	}
	
	for _, pkg := range second {
		if !existingNames[pkg.Name] {
			result = append(result, pkg)
			existingNames[pkg.Name] = true
		}
	}
	
	return result
}

func (cim *ConfigInheritanceManager) mergePackages(parent, child []PackageEntry) []PackageEntry {
	// Create a map for quick lookup
	childMap := make(map[string]PackageEntry)
	for _, pkg := range child {
		childMap[pkg.Name] = pkg
	}
	
	// Start with parent packages
	result := make([]PackageEntry, 0, len(parent)+len(child))
	for _, parentPkg := range parent {
		if childPkg, exists := childMap[parentPkg.Name]; exists {
			// Child overrides parent
			result = append(result, childPkg)
		} else {
			// No child override, use parent
			result = append(result, parentPkg)
		}
	}
	
	// Add child packages that weren't in parent
	for _, childPkg := range child {
		found := false
		for _, parentPkg := range parent {
			if parentPkg.Name == childPkg.Name {
				found = true
				break
			}
		}
		if !found {
			result = append(result, childPkg)
		}
	}
	
	return result
}

func (cim *ConfigInheritanceManager) deepCopyConfig(original *Config) *Config {
	if original == nil {
		return nil
	}
	
	result := &Config{
		Version:         original.Version,
		PackageDefaults: make(map[string][]string),
		BackupPolicy:    original.BackupPolicy,
		Files:           make(map[string]File),
		DConf: DConfConfig{
			Settings: make(map[string]string),
		},
		Repositories: RepositoryManagement{
			Apt:     make([]AptRepository, 0),
			Flatpak: make([]FlatpakRepository, 0),
		},
		Packages: PackageManagement{
			Apt:     make([]PackageEntry, len(original.Packages.Apt)),
			Flatpak: make([]PackageEntry, len(original.Packages.Flatpak)),
			Snap:    make([]PackageEntry, len(original.Packages.Snap)),
		},
		Includes: make([]IncludeSpec, len(original.Includes)),
	}
	
	// Deep copy slices and maps
	copy(result.Packages.Apt, original.Packages.Apt)
	copy(result.Packages.Flatpak, original.Packages.Flatpak)
	copy(result.Packages.Snap, original.Packages.Snap)
	copy(result.Includes, original.Includes)
	
	for k, v := range original.PackageDefaults {
		result.PackageDefaults[k] = make([]string, len(v))
		copy(result.PackageDefaults[k], v)
	}
	
	for k, v := range original.Files {
		result.Files[k] = v
	}
	
	for k, v := range original.DConf.Settings {
		result.DConf.Settings[k] = v
	}
	
	result.Repositories.Apt = make([]AptRepository, len(original.Repositories.Apt))
	copy(result.Repositories.Apt, original.Repositories.Apt)
	
	result.Repositories.Flatpak = make([]FlatpakRepository, len(original.Repositories.Flatpak))
	copy(result.Repositories.Flatpak, original.Repositories.Flatpak)
	
	return result
}

// CreateInheritanceChain creates a chain of configurations for inheritance processing
func (cim *ConfigInheritanceManager) CreateInheritanceChain(configs []*Config) (*Config, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("no configurations provided")
	}
	
	if len(configs) == 1 {
		return configs[0], nil
	}
	
	// Sort configs by priority (most general to most specific)
	// For now, we'll use the order provided, but this could be enhanced
	// with explicit priority fields in the configuration
	
	result := configs[0]
	for i := 1; i < len(configs); i++ {
		merged, err := cim.MergeConfigsWithInheritance(result, configs[i])
		if err != nil {
			return nil, fmt.Errorf("failed to merge config %d: %w", i, err)
		}
		result = merged
	}
	
	return result, nil
}

// ValidateInheritanceRules validates inheritance rules
func (cim *ConfigInheritanceManager) ValidateInheritanceRules(config InheritanceConfig) error {
	validSections := map[string]bool{
		"packages.apt":        true,
		"packages.flatpak":    true,
		"packages.snap":       true,
		"package_defaults":    true,
		"files":               true,
		"dconf.settings":      true,
		"repositories.apt":    true,
		"repositories.flatpak": true,
		"backup_policy":       true,
		"version":             true,
	}
	
	for _, rule := range config.Rules {
		if !validSections[rule.Section] {
			return fmt.Errorf("invalid section in inheritance rule: %s", rule.Section)
		}
		
		if rule.Priority < 1 || rule.Priority > 10 {
			return fmt.Errorf("invalid priority %d for section %s (must be 1-10)", rule.Priority, rule.Section)
		}
	}
	
	return nil
}

// GetInheritanceReport generates a report of how configurations were inherited
func (cim *ConfigInheritanceManager) GetInheritanceReport(configs []*Config, result *Config) string {
	var report strings.Builder
	
	report.WriteString("Configuration Inheritance Report\n")
	report.WriteString("================================\n\n")
	
	report.WriteString(fmt.Sprintf("Merged %d configuration(s) using inheritance rules:\n\n", len(configs)))
	
	// Sort rules by priority for display
	var rules []InheritanceRule
	for _, rule := range cim.rules {
		rules = append(rules, rule)
	}
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].Priority < rules[j].Priority
	})
	
	report.WriteString("Inheritance Rules Applied:\n")
	for _, rule := range rules {
		patternName := cim.getPatternName(rule.Pattern)
		report.WriteString(fmt.Sprintf("  %s: %s (priority %d)\n", rule.Section, patternName, rule.Priority))
	}
	
	report.WriteString(fmt.Sprintf("\nDefault Rule: %s\n", cim.getPatternName(cim.defaultRule)))
	report.WriteString(fmt.Sprintf("Allow Override: %t\n\n", cim.allowOverride))
	
	// Summary of merged configuration
	aptCount := len(result.Packages.Apt)
	flatpakCount := len(result.Packages.Flatpak)
	snapCount := len(result.Packages.Snap)
	totalPackages := aptCount + flatpakCount + snapCount
	
	report.WriteString("Final Configuration Summary:\n")
	report.WriteString(fmt.Sprintf("  Version: %s\n", result.Version))
	report.WriteString(fmt.Sprintf("  Total Packages: %d (APT: %d, Flatpak: %d, Snap: %d)\n", 
		totalPackages, aptCount, flatpakCount, snapCount))
	report.WriteString(fmt.Sprintf("  Files: %d\n", len(result.Files)))
	report.WriteString(fmt.Sprintf("  DConf Settings: %d\n", len(result.DConf.Settings)))
	report.WriteString(fmt.Sprintf("  APT Repositories: %d\n", len(result.Repositories.Apt)))
	report.WriteString(fmt.Sprintf("  Flatpak Repositories: %d\n", len(result.Repositories.Flatpak)))
	
	return report.String()
}

func (cim *ConfigInheritanceManager) getPatternName(pattern InheritancePattern) string {
	switch pattern {
	case InheritanceNone:
		return "None"
	case InheritanceOverride:
		return "Override"
	case InheritanceMerge:
		return "Merge"
	case InheritanceAppend:
		return "Append"
	case InheritancePrepend:
		return "Prepend"
	default:
		return "Unknown"
	}
}