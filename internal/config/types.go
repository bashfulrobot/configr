package config

// Config represents the main configuration structure
type Config struct {
	Version         string                    `yaml:"version" mapstructure:"version"`
	Includes        []IncludeSpec             `yaml:"includes,omitempty" mapstructure:"includes,omitempty"`
	PackageDefaults map[string][]string       `yaml:"package_defaults,omitempty" mapstructure:"package_defaults,omitempty"`
	BackupPolicy    BackupPolicy              `yaml:"backup_policy,omitempty" mapstructure:"backup_policy,omitempty"`
	Repositories    RepositoryManagement      `yaml:"repositories,omitempty" mapstructure:"repositories,omitempty"`
	Packages        PackageManagement         `yaml:"packages" mapstructure:"packages"`
	Files           map[string]File           `yaml:"files" mapstructure:"files"`
	Binaries        map[string]Binary         `yaml:"binaries,omitempty" mapstructure:"binaries,omitempty"`
	DConf           DConfConfig               `yaml:"dconf" mapstructure:"dconf"`
}

// IncludeSpec represents an include specification with conditional logic and glob support
type IncludeSpec struct {
	Path        string             `yaml:"path,omitempty" mapstructure:"path,omitempty"`           // File path (supports glob patterns)
	Conditions  []IncludeCondition `yaml:"conditions,omitempty" mapstructure:"conditions,omitempty"` // Conditional includes  
	Description string             `yaml:"description,omitempty" mapstructure:"description,omitempty"` // Human-readable description
	Optional    bool               `yaml:"optional,omitempty" mapstructure:"optional,omitempty"`   // Don't fail if file missing
}

// IncludeCondition represents a condition for conditional includes
type IncludeCondition struct {
	Type     string `yaml:"type" mapstructure:"type"`         // Condition type: "os", "hostname", "env", "file_exists"
	Value    string `yaml:"value" mapstructure:"value"`       // Expected value
	Operator string `yaml:"operator,omitempty" mapstructure:"operator,omitempty"` // Comparison operator: "equals", "contains", "matches"
}

// PackageManagement contains all package manager configurations
type PackageManagement struct {
	Apt     []PackageEntry `yaml:"apt" mapstructure:"apt"`
	Flatpak []PackageEntry `yaml:"flatpak" mapstructure:"flatpak"`
	Snap    []PackageEntry `yaml:"snap" mapstructure:"snap"`
}

// PackageEntry represents a package with optional configuration
// Supports both simple string format and complex nested format:
//   Simple: "package-name"
//   Complex: "package-name":
//              flags: ["--flag1", "--flag2"]
type PackageEntry struct {
	Name  string   `yaml:"-" mapstructure:"-"`                           // Package name (from YAML key or string value)
	Flags []string `yaml:"flags,omitempty" mapstructure:"flags,omitempty"` // Optional flags for this package
}

// File represents a file to be managed (dotfile, system file, etc.)
type File struct {
	Source           string `yaml:"source" mapstructure:"source"`
	Destination      string `yaml:"destination" mapstructure:"destination"`
	Owner            string `yaml:"owner,omitempty" mapstructure:"owner,omitempty"`
	Group            string `yaml:"group,omitempty" mapstructure:"group,omitempty"`
	Mode             string `yaml:"mode,omitempty" mapstructure:"mode,omitempty"`
	Backup           bool   `yaml:"backup,omitempty" mapstructure:"backup,omitempty"`
	Copy             bool   `yaml:"copy,omitempty" mapstructure:"copy,omitempty"`
	Interactive      bool   `yaml:"interactive,omitempty" mapstructure:"interactive,omitempty"`           // Prompt for conflicts
	PromptPermissions bool  `yaml:"prompt_permissions,omitempty" mapstructure:"prompt_permissions,omitempty"` // Prompt for permissions
	PromptOwnership  bool   `yaml:"prompt_ownership,omitempty" mapstructure:"prompt_ownership,omitempty"`     // Prompt for ownership
}

// Binary represents a binary to be downloaded and installed from a remote source
type Binary struct {
	Source           string `yaml:"source" mapstructure:"source"`                                         // URL to download the binary from
	Destination      string `yaml:"destination" mapstructure:"destination"`                               // Where to place the binary (typically in PATH)
	Owner            string `yaml:"owner,omitempty" mapstructure:"owner,omitempty"`                       // Optional file owner
	Group            string `yaml:"group,omitempty" mapstructure:"group,omitempty"`                       // Optional file group
	Mode             string `yaml:"mode,omitempty" mapstructure:"mode,omitempty"`                         // File permissions (default: 755)
	Backup           bool   `yaml:"backup,omitempty" mapstructure:"backup,omitempty"`                     // Backup existing binary before replacement
	Interactive      bool   `yaml:"interactive,omitempty" mapstructure:"interactive,omitempty"`           // Prompt for conflicts
	PromptPermissions bool  `yaml:"prompt_permissions,omitempty" mapstructure:"prompt_permissions,omitempty"` // Prompt for permissions
	PromptOwnership  bool   `yaml:"prompt_ownership,omitempty" mapstructure:"prompt_ownership,omitempty"`     // Prompt for ownership
}

// BackupPolicy defines automatic backup management policies
type BackupPolicy struct {
	AutoCleanup      bool   `yaml:"auto_cleanup,omitempty" mapstructure:"auto_cleanup,omitempty"`           // Enable automatic backup cleanup
	MaxAge           string `yaml:"max_age,omitempty" mapstructure:"max_age,omitempty"`                     // Maximum age (e.g., "30d", "7d", "24h")
	MaxCount         int    `yaml:"max_count,omitempty" mapstructure:"max_count,omitempty"`                 // Maximum number of backups per file
	CleanupOrphaned  bool   `yaml:"cleanup_orphaned,omitempty" mapstructure:"cleanup_orphaned,omitempty"`   // Remove orphaned backups
	PreserveRecent   int    `yaml:"preserve_recent,omitempty" mapstructure:"preserve_recent,omitempty"`     // Always preserve N most recent backups
}

// DConfConfig manages dconf settings
type DConfConfig struct {
	Settings map[string]string `yaml:"settings" mapstructure:"settings"`
}

// RepositoryManagement contains all repository configurations
type RepositoryManagement struct {
	Apt     []AptRepository     `yaml:"apt,omitempty" mapstructure:"apt,omitempty"`
	Flatpak []FlatpakRepository `yaml:"flatpak,omitempty" mapstructure:"flatpak,omitempty"`
}

// AptRepository represents an APT repository configuration using DEB822 format
// Creates .sources files in /etc/apt/sources.list.d/ (Ubuntu 24.04+ only)
type AptRepository struct {
	Name           string   `yaml:"-" mapstructure:"-"`                                         // Repository name/identifier (from YAML key)
	PPA            string   `yaml:"ppa,omitempty" mapstructure:"ppa,omitempty"`                 // PPA format: "user/repo" (e.g., "deadsnakes/ppa") - converted to DEB822
	URIs           []string `yaml:"uris,omitempty" mapstructure:"uris,omitempty"`               // Repository URIs (DEB822 format)  
	Suites         []string `yaml:"suites,omitempty" mapstructure:"suites,omitempty"`           // Distribution suites (e.g., "focal", "stable")
	Components     []string `yaml:"components,omitempty" mapstructure:"components,omitempty"`   // Repository components (e.g., "main", "contrib")
	Architectures  []string `yaml:"architectures,omitempty" mapstructure:"architectures,omitempty"` // Target architectures (e.g., "amd64", "arm64")
	Types          []string `yaml:"types,omitempty" mapstructure:"types,omitempty"`             // Repository types ("deb", "deb-src")
	SignedBy       string   `yaml:"signed_by,omitempty" mapstructure:"signed_by,omitempty"`     // GPG keyring file path
	KeyURL         string   `yaml:"key_url,omitempty" mapstructure:"key_url,omitempty"`         // GPG key download URL
	KeyID          string   `yaml:"key_id,omitempty" mapstructure:"key_id,omitempty"`           // GPG key ID from keyserver
	Trusted        bool     `yaml:"trusted,omitempty" mapstructure:"trusted,omitempty"`         // Mark repository as trusted (use carefully)
	
	// Legacy fields for backward compatibility (will be converted to DEB822)
	URI string `yaml:"uri,omitempty" mapstructure:"uri,omitempty"`   // Deprecated: use URIs, Suites, Components instead
	Key string `yaml:"key,omitempty" mapstructure:"key,omitempty"`   // Deprecated: use KeyURL or KeyID instead
}

// FlatpakRepository represents a Flatpak remote repository
// Uses flatpak remote-add command for repository management
type FlatpakRepository struct {
	Name string `yaml:"-" mapstructure:"-"`                           // Remote name (from YAML key)
	URL  string `yaml:"url" mapstructure:"url"`                       // Repository URL (required)
	User bool   `yaml:"user,omitempty" mapstructure:"user,omitempty"` // Install for user only (default: system-wide)
}