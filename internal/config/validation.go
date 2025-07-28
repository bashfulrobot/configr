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

// FormatValidationResultEnhanced provides enhanced Rust-style error formatting
func FormatValidationResultEnhanced(result *ValidationResult) string {
	formatter := NewEnhancedFormatter()
	return formatter.FormatValidationResultEnhanced(result)
}

// FormatQuickFixEnhanced provides enhanced quick fix suggestions
func FormatQuickFixEnhanced(result *ValidationResult) string {
	formatter := NewEnhancedFormatter()
	return formatter.FormatQuickFixEnhanced(result)
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
		validateIncludes(config, result, nil, configPath)
		validateRepositories(config, result, nil, configPath)
		validatePackages(config, result, nil, configPath)
		validateFiles(config, configPath, result, nil, configPath)
		validateBinaries(config, result, nil, configPath)
		validateDConf(config, result, nil, configPath)
		return result
	}
	
	// Basic structure validation with position information
	validateVersion(config, result, configWithPos, configPath)
	validateIncludes(config, result, configWithPos, configPath)
	validateRepositories(config, result, configWithPos, configPath)
	validatePackages(config, result, configWithPos, configPath)
	validateFiles(config, configPath, result, configWithPos, configPath)
	validateBinaries(config, result, configWithPos, configPath)
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

// validateIncludes checks include configurations
func validateIncludes(config *Config, result *ValidationResult, configPos *ConfigWithPosition, configPath string) {
	if len(config.Includes) == 0 {
		return
	}

	loader := NewAdvancedLoader()
	baseDir := filepath.Dir(configPath)

	for i, includeSpec := range config.Includes {
		field := fmt.Sprintf("includes[%d]", i)
		line, column := 0, 0
		if configPos != nil {
			line, column = configPos.FindFieldPosition(field)
		}

		// Validate the include spec structure
		if err := loader.ValidateIncludeSpec(includeSpec); err != nil {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "invalid include specification",
				File:    configPath,
				Line:    line,
				Column:  column,
				Field:   field,
				Message: err.Error(),
				Help:    "fix the include specification according to the schema",
				Note:    "includes require 'path' to be specified",
			})
			continue
		}

		// Validate conditions
		for j, condition := range includeSpec.Conditions {
			conditionField := fmt.Sprintf("%s.conditions[%d]", field, j)
			if err := loader.validateCondition(condition); err != nil {
				result.Add(ValidationError{
					Type:    "error",
					Title:   "invalid include condition",
					File:    configPath,
					Line:    line,
					Column:  column,
					Field:   conditionField,
					Message: err.Error(),
					Help:    "fix the condition specification",
					Note:    "valid condition types: os, hostname, env, file_exists, dir_exists",
				})
			}
		}

		// Check for potentially dangerous patterns
		if strings.Contains(includeSpec.Path, "..") {
			result.Add(ValidationError{
				Type:    "warning",
				Title:   "potentially unsafe include path",
				File:    configPath,
				Line:    line,
				Column:  column,
				Field:   fmt.Sprintf("%s.path", field),
				Value:   includeSpec.Path,
				Message: "include path contains '..' which may access files outside the configuration directory",
				Help:    "use relative paths within the configuration directory",
				Note:    "while technically valid, this pattern can make configurations less portable",
			})
		}

		// Test glob patterns for syntax validity
		if strings.ContainsAny(includeSpec.Path, "*?[") {
			pattern := filepath.Join(baseDir, includeSpec.Path)
			if _, err := filepath.Glob(pattern); err != nil {
				result.Add(ValidationError{
					Type:    "error",
					Title:   "invalid glob pattern",
					File:    configPath,
					Line:    line,
					Column:  column,
					Field:   fmt.Sprintf("%s.path", field),
					Value:   includeSpec.Path,
					Message: fmt.Sprintf("invalid glob pattern: %v", err),
					Help:    "use valid glob syntax with *, ?, and [] patterns",
					Note:    "glob patterns are relative to the configuration file directory",
				})
			}
		}

		// Validate paths (non-optional includes should exist)
		if !includeSpec.Optional && !strings.ContainsAny(includeSpec.Path, "*?[") {
			fullPath := filepath.Join(baseDir, includeSpec.Path)
			if _, err := os.Stat(fullPath); os.IsNotExist(err) {
				// Try with .yaml extension
				if filepath.Ext(fullPath) == "" {
					yamlPath := fullPath + ".yaml"
					if _, err := os.Stat(yamlPath); err != nil {
						result.Add(ValidationError{
							Type:    "error",
							Title:   "include file not found",
							File:    configPath,
							Line:    line,
							Column:  column,
							Field:   fmt.Sprintf("%s.path", field),
							Value:   includeSpec.Path,
							Message: fmt.Sprintf("include file does not exist: %s", fullPath),
							Help:    "ensure the include file exists, create it, or mark as optional",
							Note:    "set 'optional: true' if the file may not exist",
						})
					}
				}
			}
		}
	}
}

// validateRepositories checks repository configurations
func validateRepositories(config *Config, result *ValidationResult, configPos *ConfigWithPosition, configPath string) {
	// Validate APT repositories
	validateAptRepositories(config.Repositories.Apt, result, configPos, configPath)
	
	// Validate Flatpak repositories
	validateFlatpakRepositories(config.Repositories.Flatpak, result, configPos, configPath)
}

// validateAptRepositories validates APT repository configurations using DEB822 format
func validateAptRepositories(repos []AptRepository, result *ValidationResult, configPos *ConfigWithPosition, configPath string) {
	for _, repo := range repos {
		fieldPrefix := fmt.Sprintf("repositories.apt.%s", repo.Name)
		
		// Check for legacy format usage
		hasLegacy := repo.PPA != "" || repo.URI != "" || repo.Key != ""
		hasDEB822 := len(repo.URIs) > 0 || len(repo.Suites) > 0 || len(repo.Components) > 0
		
		// Validate repository format specification
		if !hasLegacy && !hasDEB822 {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "missing repository configuration",
				Field:   fieldPrefix,
				Message: "repository must specify configuration in DEB822 format or legacy format",
				Help:    "add 'ppa: \"user/repo\"' for PPA, legacy 'uri:' field, or DEB822 'uris:', 'suites:', 'components:' fields",
				Note:    "DEB822 format is recommended for Ubuntu 24.04+",
			})
			continue
		}
		
		// Validate legacy format if used
		if hasLegacy {
			validateLegacyAptRepository(repo, result, fieldPrefix)
		}
		
		// Validate DEB822 format if used
		if hasDEB822 {
			validateDEB822AptRepository(repo, result, fieldPrefix)
		}
		
		// Validate mixed format usage (warning)
		if hasLegacy && hasDEB822 {
			result.Add(ValidationError{
				Type:    "warning",
				Title:   "mixed repository format",
				Field:   fieldPrefix,
				Message: "repository uses both legacy and DEB822 format fields",
				Help:    "use either legacy format (ppa/uri/key) or DEB822 format (uris/suites/components)",
				Note:    "legacy fields will be converted to DEB822 format",
			})
		}
		
		// Validate GPG key configuration
		validateAptRepositoryKeys(repo, result, fieldPrefix)
	}
}

// validateLegacyAptRepository validates legacy APT repository format
func validateLegacyAptRepository(repo AptRepository, result *ValidationResult, fieldPrefix string) {
	hasPPA := repo.PPA != ""
	hasURI := repo.URI != ""
	
	// Validate that either PPA or URI is provided, but not both
	if hasPPA && hasURI {
		result.Add(ValidationError{
			Type:    "error",
			Title:   "conflicting legacy repository configuration",
			Field:   fieldPrefix,
			Message: "repository cannot specify both 'ppa' and 'uri'",
			Help:    "use 'ppa' for Ubuntu PPAs or 'uri' for custom repositories",
			Note:    "legacy format will be converted to DEB822",
		})
		return
	}
	
	// Validate PPA format
	if hasPPA {
		if !isValidPPAFormat(repo.PPA) {
			result.Add(ValidationError{
				Type:       "error",
				Title:      "invalid PPA format",
				Field:      fieldPrefix + ".ppa",
				Value:      repo.PPA,
				Message:    "PPA must be in 'user/repo' format",
				Help:       "use format like 'deadsnakes/ppa' or 'ubuntu-toolchain-r/test'",
				Suggestion: suggestPPAFormat(repo.PPA),
			})
		}
	}
	
	// Validate URI format
	if hasURI {
		if !isValidAPTRepositoryURI(repo.URI) {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "invalid repository URI",
				Field:   fieldPrefix + ".uri",
				Value:   repo.URI,
				Message: "repository URI format is invalid",
				Help:    "use format: 'deb [arch=amd64] https://example.com/repo stable main'",
				Note:    "URI should start with 'deb' or 'deb-src'",
			})
		}
	}
}

// validateDEB822AptRepository validates DEB822 APT repository format
func validateDEB822AptRepository(repo AptRepository, result *ValidationResult, fieldPrefix string) {
	// Validate required URIs field
	if len(repo.URIs) == 0 {
		result.Add(ValidationError{
			Type:    "error",
			Title:   "missing URIs field",
			Field:   fieldPrefix + ".uris",
			Message: "DEB822 repository must specify at least one URI",
			Help:    "add 'uris: [\"https://example.com/repo\"]'",
			Note:    "URIs specify the repository locations",
		})
	} else {
		// Validate each URI
		for i, uri := range repo.URIs {
			if !isValidRepositoryURI(uri) {
				result.Add(ValidationError{
					Type:    "error",
					Title:   "invalid repository URI",
					Field:   fmt.Sprintf("%s.uris[%d]", fieldPrefix, i),
					Value:   uri,
					Message: "repository URI must be a valid HTTPS URL",
					Help:    "use format: 'https://example.com/repo' or 'https://ppa.launchpadcontent.net/user/repo/ubuntu'",
					Note:    "HTTPS is required for security",
				})
			}
		}
	}
	
	// Validate required Suites field
	if len(repo.Suites) == 0 {
		result.Add(ValidationError{
			Type:    "error",
			Title:   "missing Suites field",
			Field:   fieldPrefix + ".suites",
			Message: "DEB822 repository must specify at least one suite",
			Help:    "add 'suites: [\"focal\"]' or 'suites: [\"stable\"]'",
			Note:    "suites specify the distribution releases",
		})
	}
	
	// Validate required Components field
	if len(repo.Components) == 0 {
		result.Add(ValidationError{
			Type:    "error",
			Title:   "missing Components field",
			Field:   fieldPrefix + ".components",
			Message: "DEB822 repository must specify at least one component",
			Help:    "add 'components: [\"main\"]' or 'components: [\"main\", \"contrib\"]'",
			Note:    "components specify the repository sections",
		})
	}
	
	// Validate optional Types field
	if len(repo.Types) > 0 {
		for i, repoType := range repo.Types {
			if !isValidRepositoryType(repoType) {
				result.Add(ValidationError{
					Type:    "error",
					Title:   "invalid repository type",
					Field:   fmt.Sprintf("%s.types[%d]", fieldPrefix, i),
					Value:   repoType,
					Message: "repository type must be 'deb' or 'deb-src'",
					Help:    "use 'deb' for binary packages or 'deb-src' for source packages",
					Note:    "defaults to 'deb' if not specified",
				})
			}
		}
	}
	
	// Validate optional Architectures field
	if len(repo.Architectures) > 0 {
		for i, arch := range repo.Architectures {
			if !isValidArchitecture(arch) {
				result.Add(ValidationError{
					Type:    "error",
					Title:   "invalid architecture",
					Field:   fmt.Sprintf("%s.architectures[%d]", fieldPrefix, i),
					Value:   arch,
					Message: "unsupported architecture",
					Help:    "use common architectures like 'amd64', 'arm64', 'armhf'",
					Note:    "defaults to 'amd64' if not specified",
				})
			}
		}
	}
}

// validateAptRepositoryKeys validates GPG key configuration
func validateAptRepositoryKeys(repo AptRepository, result *ValidationResult, fieldPrefix string) {
	hasLegacyKey := repo.Key != ""
	hasKeyURL := repo.KeyURL != ""
	hasKeyID := repo.KeyID != ""
	
	// Check for conflicting key specifications
	keyCount := 0
	if hasLegacyKey { keyCount++ }
	if hasKeyURL { keyCount++ }
	if hasKeyID { keyCount++ }
	
	if keyCount > 1 {
		result.Add(ValidationError{
			Type:    "error",
			Title:   "conflicting key specifications",
			Field:   fieldPrefix,
			Message: "repository can only specify one key method",
			Help:    "use either 'key' (legacy), 'key_url', or 'key_id'",
			Note:    "multiple key specifications are not allowed",
		})
		return
	}
	
	// Validate legacy key format
	if hasLegacyKey {
		if !isValidGPGKeyReference(repo.Key) {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "invalid GPG key reference",
				Field:   fieldPrefix + ".key",
				Value:   repo.Key,
				Message: "GPG key must be a URL or keyserver key ID",
				Help:    "use a HTTPS URL to .gpg file or keyserver key ID",
				Note:    "example: 'https://example.com/key.gpg' or '0x1234567890ABCDEF'",
			})
		}
	}
	
	// Validate key URL
	if hasKeyURL {
		if !isValidKeyURL(repo.KeyURL) {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "invalid key URL",
				Field:   fieldPrefix + ".key_url",
				Value:   repo.KeyURL,
				Message: "key URL must be a valid HTTPS URL",
				Help:    "use format: 'https://example.com/key.gpg' or 'https://example.com/key.asc'",
				Note:    "HTTPS is required for security",
			})
		}
	}
	
	// Validate key ID
	if hasKeyID {
		if !isValidKeyID(repo.KeyID) {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "invalid key ID",
				Field:   fieldPrefix + ".key_id",
				Value:   repo.KeyID,
				Message: "key ID must be a valid GPG key fingerprint or short ID",
				Help:    "use format: '0x1234567890ABCDEF' or '1234567890ABCDEF'",
				Note:    "long key IDs are recommended for security",
			})
		}
	}
	
	// Validate SignedBy path when key is specified
	if (hasKeyURL || hasKeyID) && repo.SignedBy == "" {
		result.Add(ValidationError{
			Type:    "warning",
			Title:   "missing signed_by path",
			Field:   fieldPrefix + ".signed_by",
			Message: "SignedBy path recommended when using key_url or key_id",
			Help:    "add 'signed_by: \"/usr/share/keyrings/repo-name.gpg\"'",
			Note:    "SignedBy specifies where to store the GPG key",
		})
	}
	
	// Validate SignedBy path format
	if repo.SignedBy != "" {
		if !isValidSignedByPath(repo.SignedBy) {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "invalid SignedBy path",
				Field:   fieldPrefix + ".signed_by",
				Value:   repo.SignedBy,
				Message: "SignedBy path must point to /usr/share/keyrings/",
				Help:    "use format: '/usr/share/keyrings/repo-name.gpg'",
				Note:    "keyring files should be stored in the standard location",
			})
		}
	}
}

// validateFlatpakRepositories validates Flatpak repository configurations
func validateFlatpakRepositories(repos []FlatpakRepository, result *ValidationResult, configPos *ConfigWithPosition, configPath string) {
	for _, repo := range repos {
		fieldPrefix := fmt.Sprintf("repositories.flatpak.%s", repo.Name)
		
		// Validate required URL field
		if repo.URL == "" {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "missing repository URL",
				Field:   fieldPrefix + ".url",
				Message: "Flatpak repository URL is required",
				Help:    "specify the .flatpakrepo URL or repository location",
				Note:    "example: 'https://flathub.org/repo/flathub.flatpakrepo'",
			})
			continue
		}
		
		// Validate URL format
		if !isValidFlatpakRepositoryURL(repo.URL) {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "invalid repository URL",
				Field:   fieldPrefix + ".url",
				Value:   repo.URL,
				Message: "Flatpak repository URL format is invalid",
				Help:    "use HTTPS URL to .flatpakrepo file or repository",
				Note:    "URLs should use HTTPS for security",
			})
		}
		
		// Validate repository name
		if !isValidFlatpakRemoteName(repo.Name) {
			result.Add(ValidationError{
				Type:       "error",
				Title:      "invalid remote name",
				Field:      fieldPrefix,
				Value:      repo.Name,
				Message:    "Flatpak remote name contains invalid characters",
				Help:       "use only letters, numbers, hyphens, and underscores",
				Suggestion: suggestFlatpakRemoteName(repo.Name),
			})
		}
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

// validateBinaries checks binary configurations
func validateBinaries(config *Config, result *ValidationResult, configPos *ConfigWithPosition, configPath string) {
	for name, binary := range config.Binaries {
		fieldPrefix := fmt.Sprintf("binaries.%s", name)
		
		// Validate required fields
		if binary.Source == "" {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "missing source URL",
				Field:   fieldPrefix + ".source",
				Message: "source URL is required for binary download",
				Help:    "specify the HTTPS URL to download the binary from",
				Note:    "only HTTPS URLs are allowed for security",
			})
			continue
		}
		
		if binary.Destination == "" {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "missing destination path",
				Field:   fieldPrefix + ".destination",
				Message: "destination path is required", 
				Help:    "specify where the binary should be placed",
				Note:    "use paths like /usr/local/bin/mybinary or ~/bin/mybinary",
			})
			continue
		}
		
		// Validate source URL format and security
		if !strings.HasPrefix(binary.Source, "https://") {
			result.Add(ValidationError{
				Type:       "error",
				Title:      "insecure source URL",
				Field:      fieldPrefix + ".source",
				Value:      binary.Source,
				Message:    "source URL must use HTTPS for security",
				Help:       "change http:// to https://",
				Suggestion: fmt.Sprintf("source: \"%s\"", strings.Replace(binary.Source, "http://", "https://", 1)),
			})
		}
		
		// Basic URL format validation
		if !isValidURL(binary.Source) {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "invalid URL format",
				Field:   fieldPrefix + ".source",
				Value:   binary.Source,
				Message: "source URL format is invalid",
				Help:    "ensure the URL is properly formatted",
				Note:    "example: https://github.com/user/repo/releases/download/v1.0.0/binary",
			})
		}
		
		// Validate binary mode if provided
		if binary.Mode != "" {
			if !isValidFileMode(binary.Mode) {
				result.Add(ValidationError{
					Type:       "error",
					Title:      "invalid file mode",
					Field:      fieldPrefix + ".mode",
					Value:      binary.Mode,
					Message:    "file mode must be valid octal (e.g., '755', '644')",
					Help:       "use '755' for executables, '644' for non-executables",
					Suggestion: "mode: \"755\"",
				})
			} else if !isExecutableMode(binary.Mode) {
				result.Add(ValidationError{
					Type:    "warning",
					Title:   "non-executable binary mode",
					Field:   fieldPrefix + ".mode",
					Value:   binary.Mode,
					Message: "binary file mode should typically be executable",
					Help:    "consider using '755' for executable binaries",
					Note:    "binaries usually need execute permissions to run",
				})
			}
		}
		
		// Validate destination path
		if strings.Contains(binary.Destination, "..") {
			result.Add(ValidationError{
				Type:    "error",
				Title:   "unsafe destination path",
				Field:   fieldPrefix + ".destination",
				Value:   binary.Destination,
				Message: "destination path contains '..' which is not allowed",
				Help:    "use absolute paths or paths relative to home (~)",
				Note:    "this prevents accidental binary overwrites outside intended directories",
			})
		}
		
		// Warn about non-standard binary locations
		if !isStandardBinaryLocation(binary.Destination) {
			result.Add(ValidationError{
				Type:    "warning",
				Title:   "non-standard binary location",
				Field:   fieldPrefix + ".destination",
				Value:   binary.Destination,
				Message: "binary is not being placed in a standard PATH location",
				Help:    "consider using /usr/local/bin/, ~/bin/, or ~/.local/bin/",
				Note:    "binaries outside PATH may not be easily accessible",
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

// Repository validation helper functions

// isValidPPAFormat validates PPA format (user/repo)
func isValidPPAFormat(ppa string) bool {
	// PPA format: user/repo
	matched, _ := regexp.MatchString(`^[a-z0-9][a-z0-9\-]*\/[a-z0-9][a-z0-9\-]*$`, ppa)
	return matched
}

// suggestPPAFormat suggests a corrected PPA format
func suggestPPAFormat(ppa string) string {
	// Remove common prefixes that users might add
	ppa = strings.TrimPrefix(ppa, "ppa:")
	ppa = strings.TrimSpace(ppa)
	
	// If it doesn't contain a slash, suggest adding one
	if !strings.Contains(ppa, "/") {
		return fmt.Sprintf("did you mean \"%s/ppa\"?", ppa)
	}
	
	// Clean up the format
	parts := strings.Split(ppa, "/")
	if len(parts) == 2 {
		user := strings.ToLower(regexp.MustCompile(`[^a-z0-9\-]`).ReplaceAllString(parts[0], "-"))
		repo := strings.ToLower(regexp.MustCompile(`[^a-z0-9\-]`).ReplaceAllString(parts[1], "-"))
		return fmt.Sprintf("did you mean \"%s/%s\"?", user, repo)
	}
	
	return ""
}

// isValidAPTRepositoryURI validates APT repository URI format
func isValidAPTRepositoryURI(uri string) bool {
	// Must start with "deb" or "deb-src"
	if !strings.HasPrefix(uri, "deb ") && !strings.HasPrefix(uri, "deb-src ") {
		return false
	}
	
	// Basic format check - should contain URL
	return strings.Contains(uri, "http://") || strings.Contains(uri, "https://") || strings.Contains(uri, "file://")
}

// isValidGPGKeyReference validates GPG key URL or key ID
func isValidGPGKeyReference(key string) bool {
	// Check if it's a URL
	if strings.HasPrefix(key, "https://") && (strings.HasSuffix(key, ".gpg") || strings.HasSuffix(key, ".asc")) {
		return true
	}
	
	// Check if it's a keyserver key ID (hex format, optionally prefixed with 0x)
	keyID := strings.TrimPrefix(key, "0x")
	matched, _ := regexp.MatchString(`^[A-Fa-f0-9]{8,40}$`, keyID)
	return matched
}

// isValidFlatpakRepositoryURL validates Flatpak repository URL
func isValidFlatpakRepositoryURL(url string) bool {
	// Must be HTTPS for security (or HTTP for local testing)
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return false
	}
	
	// Should end with .flatpakrepo or be a valid repository URL
	return strings.HasSuffix(url, ".flatpakrepo") || strings.Contains(url, "/repo/")
}

// isValidFlatpakRemoteName validates Flatpak remote name
func isValidFlatpakRemoteName(name string) bool {
	// Remote names: letters, numbers, hyphens, underscores
	// Must start and end with alphanumeric character
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9\-_]*[a-zA-Z0-9]$|^[a-zA-Z0-9]$`, name)
	return matched
}

// suggestFlatpakRemoteName suggests a corrected remote name
func suggestFlatpakRemoteName(name string) string {
	// Clean up the name
	suggested := regexp.MustCompile(`[^a-zA-Z0-9\-_]`).ReplaceAllString(name, "-")
	suggested = strings.Trim(suggested, "-_")
	
	if suggested != "" && suggested != name {
		return fmt.Sprintf("did you mean \"%s\"?", suggested)
	}
	
	return ""
}

// isValidURL validates basic URL format
func isValidURL(url string) bool {
	// Basic URL validation - must contain valid scheme and host
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return false
	}
	
	// Must have something after the scheme
	stripped := strings.TrimPrefix(url, "https://")
	stripped = strings.TrimPrefix(stripped, "http://")
	
	// Must contain at least a domain
	if len(stripped) < 3 || !strings.Contains(stripped, ".") {
		return false
	}
	
	return true
}

// isExecutableMode checks if a file mode includes execute permissions
func isExecutableMode(mode string) bool {
	if len(mode) < 3 {
		return false
	}
	
	// Check if any of the execute bits are set (user, group, or other)
	// For octal: 1=execute, 2=write, 4=read
	// So modes ending in 1, 3, 5, 7 have execute for owner
	// Modes with 1 in middle position have execute for group
	// Modes with 1 in last position have execute for other
	lastChar := mode[len(mode)-1:]
	return lastChar == "1" || lastChar == "3" || lastChar == "5" || lastChar == "7"
}

// isStandardBinaryLocation checks if the destination is in a standard PATH location
func isStandardBinaryLocation(destination string) bool {
	standardPaths := []string{
		"/usr/bin/",
		"/usr/local/bin/",
		"/bin/",
		"/sbin/",
		"/usr/sbin/",
		"/usr/local/sbin/",
		"~/bin/",
		"~/.local/bin/",
	}
	
	for _, path := range standardPaths {
		if strings.HasPrefix(destination, path) {
			return true
		}
	}
	
	return false
}

// DEB822 validation helper functions

// isValidRepositoryURI validates a repository URI for DEB822 format
func isValidRepositoryURI(uri string) bool {
	// Must be HTTPS for security
	if !strings.HasPrefix(uri, "https://") {
		return false
	}
	
	// Basic URL validation
	return isValidURL(uri)
}

// isValidRepositoryType validates repository type (deb or deb-src)
func isValidRepositoryType(repoType string) bool {
	return repoType == "deb" || repoType == "deb-src"
}

// isValidArchitecture validates system architecture
func isValidArchitecture(arch string) bool {
	validArchs := []string{
		"amd64", "arm64", "armhf", "armel", "i386", "mips", "mipsel", 
		"mips64el", "ppc64el", "s390x", "all", "any",
	}
	
	for _, validArch := range validArchs {
		if arch == validArch {
			return true
		}
	}
	
	return false
}

// isValidKeyURL validates a GPG key URL
func isValidKeyURL(keyURL string) bool {
	// Must be HTTPS for security
	if !strings.HasPrefix(keyURL, "https://") {
		return false
	}
	
	// Should end with common key file extensions
	if !strings.HasSuffix(keyURL, ".gpg") && !strings.HasSuffix(keyURL, ".asc") && 
	   !strings.HasSuffix(keyURL, ".key") {
		return false
	}
	
	return isValidURL(keyURL)
}

// isValidKeyID validates a GPG key ID
func isValidKeyID(keyID string) bool {
	// Clean key ID
	cleanKeyID := strings.TrimPrefix(keyID, "0x")
	
	// Must be hexadecimal and proper length
	// Short key IDs: 8 chars, Long key IDs: 16 chars, Fingerprints: 40 chars
	if len(cleanKeyID) != 8 && len(cleanKeyID) != 16 && len(cleanKeyID) != 40 {
		return false
	}
	
	// Must be valid hex
	matched, _ := regexp.MatchString(`^[A-Fa-f0-9]+$`, cleanKeyID)
	return matched
}

// isValidSignedByPath validates the SignedBy keyring path
func isValidSignedByPath(signedBy string) bool {
	// Must be in standard keyring directory
	if !strings.HasPrefix(signedBy, "/usr/share/keyrings/") {
		return false
	}
	
	// Must end with .gpg
	if !strings.HasSuffix(signedBy, ".gpg") {
		return false
	}
	
	// Validate filename (no path traversal)
	filename := strings.TrimPrefix(signedBy, "/usr/share/keyrings/")
	if strings.Contains(filename, "/") || strings.Contains(filename, "..") {
		return false
	}
	
	return true
}