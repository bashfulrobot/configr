package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ValidationError represents a single validation error with Rust-style details
type ValidationError struct {
	Type        string // "error", "warning"
	Title       string // Short error title
	File        string // File path where error occurred
	Line        int    // Line number (if available)
	Column      int    // Column number (if available)
	Field       string // YAML field path (e.g., "files.vimrc.source")
	Value       string // The problematic value
	Message     string // Main error message
	Help        string // How to fix it
	Note        string // Additional context
	Suggestion  string // Suggested fix
	Highlighted string // The problematic part to highlight
}

// ValidationResult contains all validation errors and warnings
type ValidationResult struct {
	Errors   []ValidationError
	Warnings []ValidationError
	Valid    bool
}

// ValidationFailedError is returned when configuration validation fails
type ValidationFailedError struct {
	Result *ValidationResult
}

func (e *ValidationFailedError) Error() string {
	return FormatValidationResultSimple(e.Result) + FormatQuickFixSimple(e.Result)
}

// Add adds a validation error to the result
func (vr *ValidationResult) Add(err ValidationError) {
	if err.Type == "warning" {
		vr.Warnings = append(vr.Warnings, err)
	} else {
		vr.Errors = append(vr.Errors, err)
		vr.Valid = false
	}
}

// HasErrors returns true if there are validation errors
func (vr *ValidationResult) HasErrors() bool {
	return len(vr.Errors) > 0
}

// Validate performs comprehensive validation on the configuration
func Validate(config *Config, configPath string) *ValidationResult {
	result := &ValidationResult{Valid: true}
	
	// Parse with position information for better error reporting
	configWithPos, err := ParseConfigWithPosition(configPath)
	if err != nil {
		// If we can't parse with positions, fall back to basic validation
		validateVersion(config, result, nil, configPath)
		validatePackages(config, result, nil, configPath)
		validateFiles(config, configPath, result, nil, configPath)
		validateDConf(config, result, nil, configPath)
		return result
	}
	
	// Basic structure validation with position information
	validateVersion(config, result, configWithPos, configPath)
	validatePackages(config, result, configWithPos, configPath)
	validateFiles(config, configPath, result, configWithPos, configPath)
	validateDConf(config, result, configWithPos, configPath)
	
	return result
}

// validateVersion checks the version field
func validateVersion(config *Config, result *ValidationResult, configPos *ConfigWithPosition, configPath string) {
	if config.Version == "" {
		line, column := 0, 0
		if configPos != nil {
			line, column = configPos.FindFieldPosition("version")
		}
		
		result.Add(ValidationError{
			Type:    "error",
			Title:   "missing version field",
			File:    configPath,
			Line:    line,
			Column:  column,
			Field:   "version",
			Message: "configuration version is required",
			Help:    "add 'version: \"1.0\"' to your configuration",
			Note:    "version helps ensure compatibility with configr updates",
		})
		return
	}
	
	// Simple version format check
	matched, _ := regexp.MatchString(`^\d+\.\d+(\.\d+)?$`, config.Version)
	if !matched {
		line, column := 0, 0
		if configPos != nil {
			line, column = configPos.FindFieldPosition("version")
		}
		
		result.Add(ValidationError{
			Type:       "error",
			Title:      "invalid version format",
			File:       configPath,
			Line:       line,
			Column:     column,
			Field:      "version",
			Value:      config.Version,
			Message:    "version must be in semantic version format",
			Help:       "use format like '1.0' or '1.0.0'",
			Suggestion: "version: \"1.0\"",
		})
	}
}

// validatePackages checks package manager configurations
func validatePackages(config *Config, result *ValidationResult, configPos *ConfigWithPosition, configPath string) {
	// Check for duplicate packages across managers
	allPackages := make(map[string]string) // package -> manager
	
	// Validate apt packages
	validatePackageEntries(config.Packages.Apt, "apt", allPackages, result, configPos, configPath)
	
	// Validate flatpak packages
	validatePackageEntries(config.Packages.Flatpak, "flatpak", allPackages, result, configPos, configPath)
	
	// Validate snap packages
	validatePackageEntries(config.Packages.Snap, "snap", allPackages, result, configPos, configPath)
	
	// Validate package_defaults if present
	if config.PackageDefaults != nil {
		validatePackageDefaults(config.PackageDefaults, result, configPos, configPath)
	}
}

// validateFiles checks file configurations
func validateFiles(config *Config, configPath string, result *ValidationResult, configPos *ConfigWithPosition, configFile string) {
	configDir := filepath.Dir(configPath)
	
	for name, file := range config.Files {
		fieldPrefix := fmt.Sprintf("files.%s", name)
		
		// Validate required fields
		if file.Source == "" {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "missing source path",
				Field:   fieldPrefix + ".source",
				Message: "source file path is required",
				Help:    "specify the path to your source file",
				Note:    "source paths are relative to your config file",
			})
			continue
		}
		
		if file.Destination == "" {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "missing destination path",
				Field:   fieldPrefix + ".destination",
				Message: "destination path is required", 
				Help:    "specify where the file should be placed",
				Note:    "use ~ for home directory (e.g., ~/.vimrc)",
			})
			continue
		}
		
		// Check if source file exists
		sourcePath := file.Source
		if !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(configDir, file.Source)
		}
		
		if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
			// Try to suggest alternatives
			suggestion := suggestAlternativeFile(sourcePath)
			
			result.Add(ValidationError{
				Type:       "error",
				Title:      "source file not found",
				Field:      fieldPrefix + ".source",
				Value:      file.Source,
				Message:    "source file does not exist",
				Help:       "create the file or check the path",
				Note:       fmt.Sprintf("looked for: %s", sourcePath),
				Suggestion: suggestion,
			})
		}
		
		// Validate file mode if provided
		if file.Mode != "" {
			if !isValidFileMode(file.Mode) {
				result.Add(ValidationError{
					Type:       "error",
					Title:      "invalid file mode",
					Field:      fieldPrefix + ".mode",
					Value:      file.Mode,
					Message:    "file mode must be valid octal (e.g., '644', '755')",
					Help:       "use '644' for regular files, '755' for executables",
					Suggestion: "mode: \"644\"",
				})
			} else if isOverlyPermissive(file.Mode) {
				result.Add(ValidationError{
					Type:    "warning",
					Title:   "overly permissive mode",
					Field:   fieldPrefix + ".mode",
					Value:   file.Mode,
					Message: "file mode allows write access for others",
					Help:    "consider using '644' for better security",
					Note:    "mode '777' or '666' can be security risks",
				})
			}
		}
		
		// Validate destination path
		if strings.Contains(file.Destination, "..") {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "unsafe destination path",
				Field:   fieldPrefix + ".destination",
				Value:   file.Destination,
				Message: "destination path contains '..' which is not allowed",
				Help:    "use absolute paths or paths relative to home (~)",
				Note:    "this prevents accidental file overwrites outside intended directories",
			})
		}
	}
}

// validateDConf checks dconf settings
func validateDConf(config *Config, result *ValidationResult, configPos *ConfigWithPosition, configPath string) {
	for path, value := range config.DConf.Settings {
		fieldPrefix := fmt.Sprintf("dconf.settings[\"%s\"]", path)
		
		// Validate dconf path format
		if !strings.HasPrefix(path, "/") {
			result.Add(ValidationError{
				Type:       "error",
				Title:      "invalid dconf path",
				Field:      fieldPrefix,
				Value:      path,
				Message:    "dconf path must start with '/'",
				Help:       "prefix the path with '/'",
				Suggestion: fmt.Sprintf("\"%s\": \"%s\"", "/"+path, value),
			})
		}
		
		// Check for common dconf path mistakes
		if strings.Contains(path, "//") {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "malformed dconf path",
				Field:   fieldPrefix,
				Value:   path,
				Message: "dconf path contains double slashes",
				Help:    "use single slashes to separate path segments",
			})
		}
	}
}

// Helper functions
func isValidPackageName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9\-\.\+]*$`, name)
	return matched
}

func sanitizePackageName(name string) string {
	return strings.ToLower(regexp.MustCompile(`[^a-z0-9\-\.]`).ReplaceAllString(name, "-"))
}

// isValidPackageNameForManager validates package names based on the specific package manager
func isValidPackageNameForManager(name, manager string) bool {
	switch manager {
	case "apt":
		// Check if it's a local .deb file
		if strings.HasSuffix(name, ".deb") {
			// If it ends with .deb, it must be a valid file path
			return isValidDebFilePath(name)
		}
		// APT package names: lowercase, numbers, hyphens, dots, plus signs
		matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9\-\.\+]*$`, name)
		return matched
	case "flatpak":
		// Flatpak app IDs: reverse domain notation with dots, letters, numbers
		matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9\-\._]*[a-zA-Z0-9]$`, name)
		return matched
	case "snap":
		// Snap package names: lowercase, numbers, hyphens
		matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9\-]*$`, name)
		return matched
	default:
		// Fallback to original validation
		return isValidPackageName(name)
	}
}

// getPackageNameValidationMessage returns validation message for specific package manager
func getPackageNameValidationMessage(manager string) string {
	switch manager {
	case "apt":
		return "APT package name or .deb file path contains invalid characters"
	case "flatpak":
		return "Flatpak app ID contains invalid characters"
	case "snap":
		return "Snap package name contains invalid characters"
	default:
		return "package name contains invalid characters"
	}
}

// getPackageNameValidationHelp returns validation help for specific package manager
func getPackageNameValidationHelp(manager string) string {
	switch manager {
	case "apt":
		return "use only lowercase letters, numbers, hyphens, dots, and plus signs"
	case "flatpak":
		return "use reverse domain notation like org.app.Name or com.company.App"
	case "snap":
		return "use only lowercase letters, numbers, and hyphens"
	default:
		return "use only lowercase letters, numbers, hyphens, and dots"
	}
}

// sanitizePackageNameForManager sanitizes package names based on the specific package manager
func sanitizePackageNameForManager(name, manager string) string {
	switch manager {
	case "apt":
		return strings.ToLower(regexp.MustCompile(`[^a-z0-9\-\.\+]`).ReplaceAllString(name, "-"))
	case "flatpak":
		// For flatpak, preserve case and dots, replace invalid chars with dots
		return regexp.MustCompile(`[^a-zA-Z0-9\-\._]`).ReplaceAllString(name, ".")
	case "snap":
		return strings.ToLower(regexp.MustCompile(`[^a-z0-9\-]`).ReplaceAllString(name, "-"))
	default:
		return sanitizePackageName(name)
	}
}

// validatePackageEntries validates a list of PackageEntry instances
func validatePackageEntries(packages []PackageEntry, manager string, allPackages map[string]string, result *ValidationResult, configPos *ConfigWithPosition, configPath string) {
	for _, pkg := range packages {
		// Validate package name
		if pkg.Name == "" {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "empty package name",
				Field:   fmt.Sprintf("packages.%s", manager),
				Message: "package name cannot be empty",
				Help:    "remove empty entries or provide valid package names",
			})
			continue
		}
		
		if !isValidPackageNameForManager(pkg.Name, manager) {
			result.Add(ValidationError{
				Type:       "error",
				Title:      "invalid package name",
				Field:      fmt.Sprintf("packages.%s", manager),
				Value:      pkg.Name,
				Message:    getPackageNameValidationMessage(manager),
				Help:       getPackageNameValidationHelp(manager),
				Suggestion: fmt.Sprintf("did you mean \"%s\"?", sanitizePackageNameForManager(pkg.Name, manager)),
			})
		}
		
		// Check for duplicates
		if existing, found := allPackages[pkg.Name]; found {
			result.Add(ValidationError{
				Type:    "warning",
				Title:   "duplicate package",
				Field:   fmt.Sprintf("packages.%s", manager),
				Value:   pkg.Name,
				Message: fmt.Sprintf("package '%s' is already listed in %s", pkg.Name, existing),
				Help:    "remove the duplicate entry",
				Note:    "duplicate packages are ignored but clutter configuration",
			})
		} else {
			allPackages[pkg.Name] = manager
		}
		
		// Validate package flags
		validatePackageFlags(pkg, manager, result)
	}
}

// validatePackageFlags validates the flags for a specific package entry
func validatePackageFlags(pkg PackageEntry, manager string, result *ValidationResult) {
	if len(pkg.Flags) == 0 {
		return // No flags to validate
	}
	
	// Check for dangerous flag combinations
	validateFlagSafety(pkg.Flags, pkg.Name, manager, result)
	
	// Check for conflicting flags
	validateFlagConflicts(pkg.Flags, pkg.Name, manager, result)
	
	// Suggest common patterns for specific packages
	suggestCommonFlags(pkg, manager, result)
}

// validateFlagSafety checks for potentially dangerous flags
func validateFlagSafety(flags []string, packageName, manager string, result *ValidationResult) {
	dangerousFlags := map[string]string{
		"--allow-unauthenticated": "installs packages without authentication",
		"--force":                 "bypasses safety checks",
		"--dangerous":             "bypasses snap security",
	}
	
	for _, flag := range flags {
		if warning, isDangerous := dangerousFlags[flag]; isDangerous {
			result.Add(ValidationError{
				Type:    "warning",
				Title:   "potentially dangerous flag",
				Field:   fmt.Sprintf("packages.%s", manager),
				Value:   packageName,
				Message: fmt.Sprintf("flag '%s' %s", flag, warning),
				Help:    "ensure you understand the security implications",
				Note:    "this flag reduces security but may be necessary for your use case",
			})
		}
	}
}

// validateFlagConflicts checks for conflicting flags
func validateFlagConflicts(flags []string, packageName, manager string, result *ValidationResult) {
	conflicts := map[string][]string{
		"flatpak": {"--user", "--system"}, // Can't install both user and system
	}
	
	if conflictingFlags, exists := conflicts[manager]; exists {
		found := make([]string, 0)
		for _, flag := range flags {
			for _, conflictFlag := range conflictingFlags {
				if flag == conflictFlag {
					found = append(found, flag)
				}
			}
		}
		
		if len(found) > 1 {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "conflicting flags",
				Field:   fmt.Sprintf("packages.%s", manager),
				Value:   packageName,
				Message: fmt.Sprintf("conflicting flags: %v", found),
				Help:    "choose either --user OR --system, not both",
			})
		}
	}
}

// suggestCommonFlags suggests commonly needed flags for specific packages
func suggestCommonFlags(pkg PackageEntry, manager string, result *ValidationResult) {
	suggestions := map[string]map[string][]string{
		"snap": {
			"code":          {"--classic"},
			"discord":       {},
			"slack":         {"--classic"},
			"postman":       {"--classic"},
			"android-studio": {"--classic"},
		},
	}
	
	if managerSuggestions, exists := suggestions[manager]; exists {
		if suggestedFlags, exists := managerSuggestions[pkg.Name]; exists && len(suggestedFlags) > 0 {
			// Check if package is missing commonly needed flags
			hasAllSuggested := true
			for _, suggested := range suggestedFlags {
				found := false
				for _, actual := range pkg.Flags {
					if actual == suggested {
						found = true
						break
					}
				}
				if !found {
					hasAllSuggested = false
					break
				}
			}
			
			if !hasAllSuggested && len(pkg.Flags) == 0 {
				result.Add(ValidationError{
					Type:       "warning",
					Title:      "missing common flags",
					Field:      fmt.Sprintf("packages.%s", manager),
					Value:      pkg.Name,
					Message:    fmt.Sprintf("'%s' commonly needs flags: %v", pkg.Name, suggestedFlags),
					Help:       "consider adding the suggested flags if the package fails to install",
					Suggestion: fmt.Sprintf("\"%s\":\n  flags: %v", pkg.Name, suggestedFlags),
				})
			}
		}
	}
}

// validatePackageDefaults validates the package_defaults section
func validatePackageDefaults(defaults map[string][]string, result *ValidationResult, configPos *ConfigWithPosition, configPath string) {
	supportedManagers := GetSupportedPackageManagers()
	
	for manager, flags := range defaults {
		// Check if manager is supported
		supported := false
		for _, supported_manager := range supportedManagers {
			if manager == supported_manager {
				supported = true
				break
			}
		}
		
		if !supported {
			result.Add(ValidationError{
				Type:       "error",
				Title:      "unsupported package manager",
				Field:      fmt.Sprintf("package_defaults.%s", manager),
				Value:      manager,
				Message:    fmt.Sprintf("'%s' is not a supported package manager", manager),
				Help:       fmt.Sprintf("use one of: %v", supportedManagers),
				Suggestion: "remove this entry or check for typos",
			})
		}
		
		// Validate the flags themselves
		validateFlagSafety(flags, fmt.Sprintf("(defaults for %s)", manager), manager, result)
		validateFlagConflicts(flags, fmt.Sprintf("(defaults for %s)", manager), manager, result)
	}
}

func isValidFileMode(mode string) bool {
	if len(mode) != 3 && len(mode) != 4 {
		return false
	}
	_, err := strconv.ParseInt(mode, 8, 32)
	return err == nil
}

func isOverlyPermissive(mode string) bool {
	// Check for world-writable permissions
	return strings.HasSuffix(mode, "6") || strings.HasSuffix(mode, "7") ||
		   strings.Contains(mode, "66") || strings.Contains(mode, "77")
}

func suggestAlternativeFile(path string) string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	
	// Try common alternatives
	alternatives := []string{
		filepath.Join(dir, "."+base), // Hidden file
		filepath.Join(dir, base+".example"),
		filepath.Join(dir, base+".template"),
	}
	
	for _, alt := range alternatives {
		if _, err := os.Stat(alt); err == nil {
			return fmt.Sprintf("did you mean \"%s\"?", strings.TrimPrefix(alt, filepath.Dir(path)+"/"))
		}
	}
	
	return ""
}

// isValidDebFilePath validates a local .deb file path
func isValidDebFilePath(debPath string) bool {
	// Must end with .deb
	if !strings.HasSuffix(debPath, ".deb") {
		return false
	}
	
	// Must contain a path separator (absolute or relative path)
	if !strings.Contains(debPath, "/") {
		return false
	}
	
	// Basic path sanitization - no path traversal
	if strings.Contains(debPath, "..") {
		return false
	}
	
	// Extract filename and check it's not just ".deb"
	parts := strings.Split(debPath, "/")
	filename := parts[len(parts)-1]
	if filename == ".deb" {
		return false
	}
	
	return true
}