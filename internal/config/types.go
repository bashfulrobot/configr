package config

// Config represents the main configuration structure
type Config struct {
	Version         string                    `yaml:"version" mapstructure:"version"`
	Includes        []string                  `yaml:"includes,omitempty" mapstructure:"includes,omitempty"`
	PackageDefaults map[string][]string       `yaml:"package_defaults,omitempty" mapstructure:"package_defaults,omitempty"`
	Packages        PackageManagement         `yaml:"packages" mapstructure:"packages"`
	Files           map[string]File           `yaml:"files" mapstructure:"files"`
	DConf           DConfConfig               `yaml:"dconf" mapstructure:"dconf"`
}

// PackageManagement contains all package manager configurations
type PackageManagement struct {
	Apt     []string `yaml:"apt" mapstructure:"apt"`
	Flatpak []string `yaml:"flatpak" mapstructure:"flatpak"`
	Snap    []string `yaml:"snap" mapstructure:"snap"`
}

// File represents a file to be managed (dotfile, system file, etc.)
type File struct {
	Source      string `yaml:"source" mapstructure:"source"`
	Destination string `yaml:"destination" mapstructure:"destination"`
	Owner       string `yaml:"owner,omitempty" mapstructure:"owner,omitempty"`
	Group       string `yaml:"group,omitempty" mapstructure:"group,omitempty"`
	Mode        string `yaml:"mode,omitempty" mapstructure:"mode,omitempty"`
	Backup      bool   `yaml:"backup,omitempty" mapstructure:"backup,omitempty"`
}

// DConfConfig manages dconf settings
type DConfConfig struct {
	Settings map[string]string `yaml:"settings" mapstructure:"settings"`
}