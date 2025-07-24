package config

// Config represents the main configuration structure
type Config struct {
	Version         string                    `yaml:"version" mapstructure:"version"`
	Includes        []string                  `yaml:"includes,omitempty" mapstructure:"includes,omitempty"`
	PackageDefaults map[string][]string       `yaml:"package_defaults,omitempty" mapstructure:"package_defaults,omitempty"`
	Repositories    RepositoryManagement      `yaml:"repositories,omitempty" mapstructure:"repositories,omitempty"`
	Packages        PackageManagement         `yaml:"packages" mapstructure:"packages"`
	Files           map[string]File           `yaml:"files" mapstructure:"files"`
	DConf           DConfConfig               `yaml:"dconf" mapstructure:"dconf"`
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
	Source      string `yaml:"source" mapstructure:"source"`
	Destination string `yaml:"destination" mapstructure:"destination"`
	Owner       string `yaml:"owner,omitempty" mapstructure:"owner,omitempty"`
	Group       string `yaml:"group,omitempty" mapstructure:"group,omitempty"`
	Mode        string `yaml:"mode,omitempty" mapstructure:"mode,omitempty"`
	Backup      bool   `yaml:"backup,omitempty" mapstructure:"backup,omitempty"`
	Copy        bool   `yaml:"copy,omitempty" mapstructure:"copy,omitempty"`
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

// AptRepository represents an APT repository configuration
// Uses add-apt-repository command for PPA and repository management
type AptRepository struct {
	Name string `yaml:"-" mapstructure:"-"`                           // Repository name/identifier (from YAML key)
	PPA  string `yaml:"ppa,omitempty" mapstructure:"ppa,omitempty"`   // PPA format: "user/repo" (e.g., "deadsnakes/ppa")
	URI  string `yaml:"uri,omitempty" mapstructure:"uri,omitempty"`   // Full repository URI for non-PPA repos
	Key  string `yaml:"key,omitempty" mapstructure:"key,omitempty"`   // GPG key URL or keyserver key ID
}

// FlatpakRepository represents a Flatpak remote repository
// Uses flatpak remote-add command for repository management
type FlatpakRepository struct {
	Name string `yaml:"-" mapstructure:"-"`                           // Remote name (from YAML key)
	URL  string `yaml:"url" mapstructure:"url"`                       // Repository URL (required)
	User bool   `yaml:"user,omitempty" mapstructure:"user,omitempty"` // Install for user only (default: system-wide)
}