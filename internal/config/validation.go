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
	
	for _, pkg := range config.Packages.Apt {
		if pkg == "" {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "empty package name",
				Field:   "packages.apt",
				Message: "package name cannot be empty",
				Help:    "remove empty entries or provide valid package names",
			})
			continue
		}
		
		if !isValidPackageName(pkg) {
			result.Add(ValidationError{
				Type:       "error",
				Title:      "invalid package name",
				Field:      "packages.apt",
				Value:      pkg,
				Message:    "package name contains invalid characters",
				Help:       "use only lowercase letters, numbers, hyphens, and dots",
				Suggestion: fmt.Sprintf("did you mean \"%s\"?", sanitizePackageName(pkg)),
			})
		}
		
		if existing, found := allPackages[pkg]; found {
			result.Add(ValidationError{
				Type:    "warning",
				Title:   "duplicate package",
				Field:   "packages.apt",
				Value:   pkg,
				Message: fmt.Sprintf("package '%s' is already listed in %s", pkg, existing),
				Help:    "remove the duplicate entry",
				Note:    "duplicate packages are ignored but clutter configuration",
			})
		} else {
			allPackages[pkg] = "apt"
		}
	}
	
	// Similar validation for flatpak and snap...
	validatePackageList(config.Packages.Flatpak, "flatpak", allPackages, result, configPos, configPath)
	validatePackageList(config.Packages.Snap, "snap", allPackages, result, configPos, configPath)
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

func validatePackageList(packages []string, manager string, allPackages map[string]string, result *ValidationResult, configPos *ConfigWithPosition, configPath string) {
	for _, pkg := range packages {
		if existing, found := allPackages[pkg]; found {
			result.Add(ValidationError{
				Type:    "warning", 
				Title:   "duplicate package",
				Field:   fmt.Sprintf("packages.%s", manager),
				Value:   pkg,
				Message: fmt.Sprintf("package '%s' is already listed in %s", pkg, existing),
				Help:    "remove the duplicate entry",
			})
		} else {
			allPackages[pkg] = manager
		}
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