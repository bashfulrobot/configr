package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigTemplate represents a configuration template
type ConfigTemplate struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Version     string            `yaml:"version"`
	Author      string            `yaml:"author"`
	Variables   map[string]string `yaml:"variables"`
	Files       map[string]string `yaml:"files"`
}

// TemplateScaffolder handles configuration template scaffolding
type TemplateScaffolder struct {
	baseDir   string
	templates map[string]*ConfigTemplate
}

// NewTemplateScaffolder creates a new template scaffolder
func NewTemplateScaffolder(baseDir string) *TemplateScaffolder {
	return &TemplateScaffolder{
		baseDir:   baseDir,
		templates: make(map[string]*ConfigTemplate),
	}
}

// RegisterBuiltinTemplates registers built-in configuration templates
func (ts *TemplateScaffolder) RegisterBuiltinTemplates() {
	ts.templates["minimal"] = &ConfigTemplate{
		Name:        "minimal",
		Description: "Minimal configuration with basic packages",
		Version:     "1.0",
		Author:      "configr",
		Variables: map[string]string{
			"username": "user",
			"email":    "user@example.com",
		},
		Files: map[string]string{
			"configr.yaml": minimalTemplate,
		},
	}

	ts.templates["developer"] = &ConfigTemplate{
		Name:        "developer",
		Description: "Developer workstation configuration",
		Version:     "1.0",
		Author:      "configr",
		Variables: map[string]string{
			"username":     "developer",
			"email":        "dev@example.com",
			"github_user":  "username",
			"editor":       "code",
			"shell":        "bash",
		},
		Files: map[string]string{
			"configr.yaml":              developerMainTemplate,
			"packages/development.yaml": developerPackagesTemplate,
			"files/dotfiles.yaml":       developerDotfilesTemplate,
			"repositories.yaml":         developerRepositoriesTemplate,
		},
	}

	ts.templates["server"] = &ConfigTemplate{
		Name:        "server",
		Description: "Server configuration with essential packages",
		Version:     "1.0",
		Author:      "configr",
		Variables: map[string]string{
			"hostname":    "server",
			"environment": "production",
			"admin_user":  "admin",
		},
		Files: map[string]string{
			"configr.yaml":        serverMainTemplate,
			"packages/server.yaml": serverPackagesTemplate,
			"files/system.yaml":   serverFilesTemplate,
		},
	}

	ts.templates["desktop"] = &ConfigTemplate{
		Name:        "desktop",
		Description: "Desktop environment configuration",
		Version:     "1.0",
		Author:      "configr",
		Variables: map[string]string{
			"username":     "user",
			"desktop_env":  "gnome",
			"theme":        "default",
			"icon_theme":   "default",
		},
		Files: map[string]string{
			"configr.yaml":           desktopMainTemplate,
			"packages/desktop.yaml":  desktopPackagesTemplate,
			"packages/media.yaml":    desktopMediaTemplate,
			"dconf/gnome.yaml":       desktopDconfTemplate,
			"files/dotfiles.yaml":    desktopDotfilesTemplate,
		},
	}

	ts.templates["advanced"] = &ConfigTemplate{
		Name:        "advanced",
		Description: "Advanced configuration with includes and conditions",
		Version:     "1.0",
		Author:      "configr",
		Variables: map[string]string{
			"username":     "user",
			"environment":  "development",
			"hostname":     "workstation",
		},
		Files: map[string]string{
			"configr.yaml":                     advancedMainTemplate,
			"common/base.yaml":                 advancedBaseTemplate,
			"environments/development.yaml":    advancedDevelopmentTemplate,
			"environments/production.yaml":     advancedProductionTemplate,
			"hosts/workstation.yaml":          advancedWorkstationTemplate,
			"hosts/laptop.yaml":               advancedLaptopTemplate,
		},
	}
}

// GetTemplate returns a template by name
func (ts *TemplateScaffolder) GetTemplate(name string) (*ConfigTemplate, error) {
	template, exists := ts.templates[name]
	if !exists {
		return nil, fmt.Errorf("template '%s' not found", name)
	}
	return template, nil
}

// ListTemplates returns a list of available templates
func (ts *TemplateScaffolder) ListTemplates() []string {
	var names []string
	for name := range ts.templates {
		names = append(names, name)
	}
	return names
}

// ScaffoldProject creates a new project from a template
func (ts *TemplateScaffolder) ScaffoldProject(templateName string, variables map[string]string, outputDir string) error {
	tmpl, err := ts.GetTemplate(templateName)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	// Merge template variables with provided variables
	allVars := make(map[string]string)
	for k, v := range tmpl.Variables {
		allVars[k] = v
	}
	for k, v := range variables {
		allVars[k] = v
	}

	// Add system variables
	allVars["timestamp"] = time.Now().Format("2006-01-02 15:04:05")
	allVars["date"] = time.Now().Format("2006-01-02")
	allVars["year"] = time.Now().Format("2006")

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Process each file in the template
	for fileName, content := range tmpl.Files {
		if err := ts.createFileFromTemplate(fileName, content, allVars, outputDir); err != nil {
			return fmt.Errorf("failed to create file %s: %w", fileName, err)
		}
	}

	return nil
}

// createFileFromTemplate creates a file from a template
func (ts *TemplateScaffolder) createFileFromTemplate(fileName, templateContent string, variables map[string]string, outputDir string) error {
	// Parse template
	tmpl, err := template.New(fileName).Parse(templateContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create file path
	filePath := filepath.Join(outputDir, fileName)
	fileDir := filepath.Dir(filePath)

	// Create directory if needed
	if err := os.MkdirAll(fileDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", fileDir, err)
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, variables); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// ValidateTemplate validates a template configuration
func (ts *TemplateScaffolder) ValidateTemplate(tmpl *ConfigTemplate) error {
	if tmpl.Name == "" {
		return fmt.Errorf("template name is required")
	}

	if tmpl.Version == "" {
		return fmt.Errorf("template version is required")
	}

	if len(tmpl.Files) == 0 {
		return fmt.Errorf("template must have at least one file")
	}

	// Validate that template files are valid YAML templates
	for fileName, content := range tmpl.Files {
		if strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml") {
			// Try to parse the template
			tmplObj, err := template.New(fileName).Parse(content)
			if err != nil {
				return fmt.Errorf("invalid template syntax in %s: %w", fileName, err)
			}

			// Try to execute with sample variables
			sampleVars := make(map[string]string)
			for k, v := range tmpl.Variables {
				sampleVars[k] = v
			}

			var buf strings.Builder
			if err := tmplObj.Execute(&buf, sampleVars); err != nil {
				return fmt.Errorf("template execution failed for %s: %w", fileName, err)
			}

			// Try to parse as YAML
			var testConfig interface{}
			if err := yaml.Unmarshal([]byte(buf.String()), &testConfig); err != nil {
				return fmt.Errorf("template %s does not produce valid YAML: %w", fileName, err)
			}
		}
	}

	return nil
}

// Template definitions

const minimalTemplate = `version: "1.0"

packages:
  apt:
    - git
    - curl
    - htop
    - tree

files:
  bashrc:
    source: "dotfiles/.bashrc"
    destination: "~/.bashrc"
    backup: true

dconf:
  settings:
    "/org/gnome/desktop/interface/clock-show-seconds": "true"
`

const developerMainTemplate = `version: "1.0"

# Developer workstation configuration
# Generated on {{.timestamp}} for {{.username}}

includes:
  - path: "repositories.yaml"
    description: "Development repositories"
  - path: "packages/development.yaml"
    description: "Development packages and tools"
  - path: "files/dotfiles.yaml"
    description: "Developer dotfiles and configurations"

package_defaults:
  apt: ["-y", "--no-install-recommends"]
  flatpak: ["--user", "--assumeyes"]

backup_policy:
  enabled: true
  retention_days: 30
  backup_location: "~/.config/configr/backups"
`

const developerPackagesTemplate = `version: "1.0"

packages:
  apt:
    - git
    - {{.editor}}
    - curl
    - wget
    - htop
    - tree
    - neofetch
    - build-essential
    - nodejs
    - npm
    - python3
    - python3-pip
    - docker.io
    - docker-compose
  
  flatpak:
    - com.visualstudio.code
    - org.mozilla.Firefox
  
  snap:
    - discord
    - slack
`

const developerDotfilesTemplate = `version: "1.0"

files:
  bashrc:
    source: "dotfiles/.bashrc"
    destination: "~/.bashrc"
    backup: true
    interactive: true

  vimrc:
    source: "dotfiles/.vimrc"
    destination: "~/.vimrc"
    backup: true

  gitconfig:
    source: "dotfiles/.gitconfig"
    destination: "~/.gitconfig"
    backup: true

  ssh_config:
    source: "dotfiles/.ssh/config"
    destination: "~/.ssh/config"
    mode: "600"
    backup: true
`

const developerRepositoriesTemplate = `version: "1.0"

repositories:
  apt:
    docker:
      uri: "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"
      key: "https://download.docker.com/linux/ubuntu/gpg"
    
    nodejs:
      uri: "deb https://deb.nodesource.com/node_16.x focal main"
      key: "https://deb.nodesource.com/gpgkey/nodesource.gpg.key"

  flatpak:
    flathub:
      url: "https://flathub.org/repo/flathub.flatpakrepo"
      user: false
`

const serverMainTemplate = `version: "1.0"

# Server configuration for {{.hostname}}
# Environment: {{.environment}}
# Generated on {{.timestamp}}

includes:
  - path: "packages/server.yaml"
    description: "Server packages"
  - path: "files/system.yaml"
    description: "System configuration files"

package_defaults:
  apt: ["-y", "--no-install-recommends"]

backup_policy:
  enabled: true
  retention_days: 90
  backup_location: "/var/backups/configr"
`

const serverPackagesTemplate = `version: "1.0"

packages:
  apt:
    - htop
    - tree
    - curl
    - wget
    - unzip
    - git
    - openssh-server
    - ufw
    - fail2ban
    - logrotate
    - rsync
    - cron
`

const serverFilesTemplate = `version: "1.0"

files:
  sshd_config:
    source: "system/sshd_config"
    destination: "/etc/ssh/sshd_config"
    owner: "root"
    group: "root"
    mode: "644"
    backup: true

  ufw_rules:
    source: "system/ufw.rules"
    destination: "/etc/ufw/user.rules"
    owner: "root"
    group: "root"
    mode: "640"
    backup: true
`

const desktopMainTemplate = `version: "1.0"

# Desktop configuration for {{.username}}
# Desktop Environment: {{.desktop_env}}
# Generated on {{.timestamp}}

includes:
  - path: "packages/desktop.yaml"
    description: "Desktop applications"
  - path: "packages/media.yaml"
    description: "Media applications"
  - path: "dconf/gnome.yaml"
    description: "GNOME desktop settings"
    conditions:
      - type: "env"
        value: "DESKTOP_SESSION"
        operator: "contains"
  - path: "files/dotfiles.yaml"
    description: "Desktop dotfiles"

package_defaults:
  flatpak: ["--user", "--assumeyes"]
`

const desktopPackagesTemplate = `version: "1.0"

packages:
  apt:
    - firefox
    - thunderbird
    - libreoffice
    - gimp
    - vlc
    - htop
    - tree
    - curl
    - wget
    - git

  flatpak:
    - org.mozilla.Firefox
    - org.mozilla.Thunderbird
    - org.libreoffice.LibreOffice
    - org.gimp.GIMP

  snap:
    - discord
    - slack
    - spotify
`

const desktopMediaTemplate = `version: "1.0"

packages:
  apt:
    - vlc
    - audacity
    - ffmpeg
    - imagemagick

  flatpak:
    - org.videolan.VLC
    - org.audacityteam.Audacity
    - org.blender.Blender

  snap:
    - obs-studio
`

const desktopDconfTemplate = `version: "1.0"

dconf:
  settings:
    # Desktop interface settings
    "/org/gnome/desktop/interface/clock-show-seconds": "true"
    "/org/gnome/desktop/interface/show-battery-percentage": "true"
    "/org/gnome/desktop/interface/gtk-theme": "'{{.theme}}'"
    "/org/gnome/desktop/interface/icon-theme": "'{{.icon_theme}}'"
    
    # Window manager settings
    "/org/gnome/desktop/wm/preferences/button-layout": "'close,minimize,maximize:'"
    "/org/gnome/desktop/wm/preferences/focus-mode": "'click'"
    
    # Keyboard shortcuts
    "/org/gnome/settings-daemon/plugins/media-keys/terminal": "['<Super>t']"
    "/org/gnome/settings-daemon/plugins/media-keys/home": "['<Super>e']"
`

const desktopDotfilesTemplate = `version: "1.0"

files:
  bashrc:
    source: "dotfiles/.bashrc"
    destination: "~/.bashrc"
    backup: true

  profile:
    source: "dotfiles/.profile"
    destination: "~/.profile"
    backup: true

  gitconfig:
    source: "dotfiles/.gitconfig"
    destination: "~/.gitconfig"
    backup: true
`

const advancedMainTemplate = `version: "1.0"

# Advanced configuration with conditional includes
# User: {{.username}}
# Environment: {{.environment}}
# Hostname: {{.hostname}}
# Generated on {{.timestamp}}

includes:
  # Base configuration (always loaded)
  - path: "common/base.yaml"
    description: "Base configuration for all systems"

  # Environment-specific configuration
  - path: "environments/{{.environment}}.yaml"
    description: "{{.environment}} environment configuration"
    optional: true
    conditions:
      - type: "env"
        value: "ENVIRONMENT={{.environment}}"
        operator: "equals"

  # Host-specific configuration
  - path: "hosts/workstation.yaml"
    description: "Workstation-specific configuration"
    optional: true
    conditions:
      - type: "hostname"
        value: "workstation"
        operator: "contains"

  - path: "hosts/laptop.yaml"
    description: "Laptop-specific configuration"
    optional: true
    conditions:
      - type: "hostname"
        value: "laptop"
        operator: "contains"

  # OS-specific configuration
  - path: "os-specific/*.yaml"
    description: "OS-specific configuration files"
    optional: true
    conditions:
      - type: "os"
        value: "linux"
        operator: "equals"

package_defaults:
  apt: ["-y", "--no-install-recommends"]
  flatpak: ["--user", "--assumeyes"]

backup_policy:
  enabled: true
  retention_days: 30
  backup_location: "~/.config/configr/backups"
`

const advancedBaseTemplate = `version: "1.0"

# Base configuration included by all systems

packages:
  apt:
    - git
    - curl
    - wget
    - htop
    - tree
    - unzip

files:
  bashrc:
    source: "dotfiles/.bashrc"
    destination: "~/.bashrc"
    backup: true

  gitconfig:
    source: "dotfiles/.gitconfig"
    destination: "~/.gitconfig"
    backup: true
`

const advancedDevelopmentTemplate = `version: "1.0"

# Development environment specific configuration

packages:
  apt:
    - build-essential
    - nodejs
    - npm
    - python3
    - python3-pip
    - docker.io
    - docker-compose

  flatpak:
    - com.visualstudio.code

repositories:
  apt:
    docker:
      uri: "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"
      key: "https://download.docker.com/linux/ubuntu/gpg"
`

const advancedProductionTemplate = `version: "1.0"

# Production environment specific configuration

packages:
  apt:
    - openssh-server
    - ufw
    - fail2ban
    - logrotate

files:
  sshd_config:
    source: "system/sshd_config"
    destination: "/etc/ssh/sshd_config"
    owner: "root"
    group: "root"
    mode: "644"
    backup: true
`

const advancedWorkstationTemplate = `version: "1.0"

# Workstation-specific configuration

packages:
  apt:
    - code
    - docker.io
    - kubernetes-cli

  snap:
    - discord
    - slack

dconf:
  settings:
    "/org/gnome/desktop/interface/clock-show-seconds": "true"
    "/org/gnome/settings-daemon/plugins/media-keys/terminal": "['<Super>t']"
`

const advancedLaptopTemplate = `version: "1.0"

# Laptop-specific configuration

packages:
  apt:
    - tlp
    - powertop
    - acpi

files:
  tlp_config:
    source: "system/tlp.conf"
    destination: "/etc/tlp.conf"
    owner: "root"
    group: "root"
    mode: "644"
    backup: true

dconf:
  settings:
    "/org/gnome/settings-daemon/plugins/power/sleep-inactive-ac-timeout": "3600"
    "/org/gnome/settings-daemon/plugins/power/sleep-inactive-battery-timeout": "1800"
`