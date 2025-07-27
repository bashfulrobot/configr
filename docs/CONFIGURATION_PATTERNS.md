# Configuration Patterns and Best Practices

This document outlines recommended patterns and best practices for structuring configr configurations.

## Table of Contents

1. [Configuration Organization](#configuration-organization)
2. [Package Management Patterns](#package-management-patterns)
3. [File Management Patterns](#file-management-patterns)
4. [Include System Patterns](#include-system-patterns)
5. [Repository Management](#repository-management)
6. [Interactive Features](#interactive-features)
7. [Performance Optimization](#performance-optimization)
8. [Security Best Practices](#security-best-practices)
9. [Common Anti-Patterns](#common-anti-patterns)
10. [Troubleshooting](#troubleshooting)

## Configuration Organization

### Single User Configuration

For personal desktop configuration:

```yaml
version: "1.0"

# Keep it simple for single-user setups
packages:
  apt:
    - git
    - curl
    - vim
  flatpak:
    - org.mozilla.Firefox
  snap:
    - code

files:
  bashrc:
    source: "dotfiles/bashrc"
    destination: "~/.bashrc"
    backup: true

dconf:
  settings:
    "/org/gnome/desktop/interface/gtk-theme": "'Adwaita-dark'"
```

### Multi-Environment Configuration

For configurations that need to work across different environments:

```yaml
version: "1.0"

# Use conditional includes for environment-specific configuration
includes:
  - path: "common/base.yaml"
  - path: "environments/development.yaml"
    optional: true
    conditions:
      - type: "env"
        value: "NODE_ENV=development"
  - path: "environments/production.yaml"
    optional: true
    conditions:
      - type: "env"
        value: "NODE_ENV=production"
  - path: "hosts/workstation.yaml"
    optional: true
    conditions:
      - type: "hostname"
        value: "workstation"
        operator: "contains"

# Base configuration that applies everywhere
packages:
  apt:
    - git
    - curl
```

### Team/Organization Configuration

For shared configurations across a team:

```yaml
version: "1.0"

# Modular approach for team configurations
includes:
  - path: "team/base-tools.yaml"
  - path: "team/development-tools.yaml"
  - glob: "projects/*.yaml"
    optional: true
  - path: "personal/customizations.yaml"
    optional: true

# Organization-wide package defaults
package_defaults:
  apt: ["-y", "--no-install-recommends"]
  flatpak: ["--system"]
  snap: ["--classic"]
```

## Package Management Patterns

### Three-Tier Flag Strategy

Use the three-tier flag system effectively:

```yaml
# Level 1: Internal defaults (built-in)
# APT: ["-y", "--no-install-recommends"]
# Flatpak: ["--system", "--assumeyes"] 
# Snap: []

# Level 2: Your global defaults (override internal)
package_defaults:
  apt: ["-y"]                    # Less opinionated than internal
  flatpak: ["--user"]            # Prefer user installs
  snap: ["--classic"]            # Default to classic confinement

packages:
  apt:
    # Level 3: Per-package flags (highest priority)
    - git                        # Uses: ["-y"] from package_defaults
    - "docker.io":
        flags: ["-y", "--install-suggests"]  # Override for this package
  
  flatpak:
    - org.mozilla.Firefox        # Uses: ["--user"] from package_defaults
    - "org.gimp.GIMP":
        flags: ["--system"]      # Override for this package
```

### Package Grouping by Purpose

Organize packages by function:

```yaml
packages:
  apt:
    # Base system tools
    - git
    - curl
    - wget
    - htop
    
    # Development tools
    - build-essential
    - python3
    - python3-pip
    - nodejs
    - npm
    
    # System administration
    - docker.io
    - nginx
    - postgresql
    
  flatpak:
    # Desktop applications
    - org.mozilla.Firefox
    - org.libreoffice.LibreOffice
    - org.gimp.GIMP
    
    # Media applications
    - org.videolan.VLC
    - org.audacityteam.Audacity
```

### Local Package Management

Handle local .deb files properly:

```yaml
packages:
  apt:
    # Repository packages
    - git
    - curl
    
    # Local packages with relative paths
    - "./packages/custom-app.deb":
        flags: ["-y", "--force-depends"]
    
    # Local packages with absolute paths
    - "/opt/packages/proprietary.deb"
    
    # Mixed with repository packages
    - vim
```

## File Management Patterns

### Dotfile Management

Best practices for managing dotfiles:

```yaml
files:
  # Simple dotfiles with symlink mode (default)
  bashrc:
    source: "dotfiles/bashrc"
    destination: "~/.bashrc"
    backup: true                 # Always backup existing files
  
  # Nested configuration directories
  vim_config:
    source: "dotfiles/vimrc"
    destination: "~/.config/vim/vimrc"
    backup: true
  
  # Template files that need to be independent
  app_config:
    source: "templates/app.conf"
    destination: "~/.config/app/config"
    copy: true                   # Copy mode for templates
    backup: true
```

### System File Management

For system-level files:

```yaml
files:
  # System configuration with proper ownership
  nginx_config:
    source: "system/nginx.conf"
    destination: "/etc/nginx/nginx.conf"
    owner: "root"
    group: "root"
    mode: "644"
    copy: true                   # Always copy system files
    backup: true
    
  # Service configuration
  systemd_service:
    source: "system/my-service.service"
    destination: "/etc/systemd/system/my-service.service"
    owner: "root"
    group: "root"
    mode: "644"
    copy: true
    backup: true
```

### Interactive File Deployment

Use interactive features for sensitive files:

```yaml
files:
  # Critical configuration with interactive prompts
  ssh_config:
    source: "dotfiles/ssh_config"
    destination: "~/.ssh/config"
    mode: "600"                  # Secure permissions
    backup: true
    interactive: true            # Prompt for conflicts
    prompt_permissions: true     # Confirm permission changes
  
  # System file with ownership prompts
  system_config:
    source: "system/app.conf"
    destination: "/etc/app/config"
    owner: "root"
    group: "app"
    mode: "640"
    copy: true
    backup: true
    interactive: true
    prompt_ownership: true       # Confirm ownership changes
```

## Include System Patterns

### Modular Configuration Structure

Organize large configurations:

```
configr-config/
├── configr.yaml              # Main configuration
├── common/
│   ├── base.yaml             # Base packages/settings
│   └── development.yaml      # Development tools
├── environments/
│   ├── development.yaml      # Dev-specific config
│   └── production.yaml       # Prod-specific config
├── hosts/
│   ├── workstation.yaml      # Workstation-specific
│   └── laptop.yaml           # Laptop-specific
└── packages/
    ├── apt.yaml              # APT packages
    ├── flatpak.yaml          # Flatpak applications
    └── snap.yaml             # Snap packages
```

Main configuration file:

```yaml
# configr.yaml
version: "1.0"

includes:
  # Always include base configuration
  - path: "common/base.yaml"
  
  # Conditionally include environment-specific config
  - path: "environments/development.yaml"
    optional: true
    conditions:
      - type: "env"
        value: "NODE_ENV=development"
  
  # Include all package files
  - glob: "packages/*.yaml"
    description: "All package configurations"
  
  # Host-specific configuration
  - path: "hosts/workstation.yaml"
    optional: true
    conditions:
      - type: "hostname"
        value: "workstation"
        operator: "equals"

# Local overrides can still be included inline
packages:
  apt:
    - personal-tool
```

### Conditional Includes

Use conditions effectively:

```yaml
includes:
  # OS-specific configuration
  - path: "os-specific/linux.yaml"
    optional: true
    conditions:
      - type: "os"
        value: "linux"
  
  # Development environment
  - path: "dev-tools.yaml"
    optional: true
    conditions:
      - type: "env"
        value: "CONFIGR_ENV=development"
  
  # Work vs home configuration
  - path: "work/corporate.yaml"
    optional: true
    conditions:
      - type: "hostname"
        value: "corp"
        operator: "contains"
      - type: "file_exists"
        value: "/etc/corporate-config"
  
  # User-specific customizations
  - path: "users/${USER}.yaml"
    optional: true
    conditions:
      - type: "env"
        value: "USER"  # Check if USER env var exists
```

## Repository Management

### APT Repository Patterns

Organize APT repositories by purpose:

```yaml
repositories:
  apt:
    # Development tools
    nodejs:
      uri: "deb https://deb.nodesource.com/node_16.x focal main"
      key: "0x9FD3B784BC1C6FC31A8A0A1C1655A0AB68576280"
    
    docker:
      uri: "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"
      key: "https://download.docker.com/linux/ubuntu/gpg"
    
    # Language-specific repositories
    python39:
      ppa: "deadsnakes/ppa"
    
    # Multimedia
    multimedia:
      ppa: "ubuntuhandbook1/apps"

packages:
  apt:
    # These packages will be available from the repositories above
    - nodejs
    - docker-ce
    - python3.9
    - vlc
```

### Flatpak Repository Management

```yaml
repositories:
  flatpak:
    # Main application hub (system-wide)
    flathub:
      url: "https://flathub.org/repo/flathub.flatpakrepo"
      user: false
    
    # Development applications (user-only)
    kde:
      url: "https://distribute.kde.org/kdeapps.flatpakrepo"
      user: true
    
    # Nightly builds (user-only, optional)
    gnome-nightly:
      url: "https://nightly.gnome.org/gnome-nightly.flatpakrepo"
      user: true

packages:
  flatpak:
    # From flathub (system-wide)
    - org.mozilla.Firefox
    
    # From KDE repository (user-only due to repo config)
    - org.kde.krita
```

## Interactive Features

### Progressive Interactivity

Start with basic automation, add interactivity where needed:

```yaml
files:
  # Fully automated for safe files
  bashrc:
    source: "dotfiles/bashrc"
    destination: "~/.bashrc"
    backup: true
  
  # Interactive for potentially conflicting files
  ssh_config:
    source: "dotfiles/ssh_config"
    destination: "~/.ssh/config"
    backup: true
    interactive: true            # Enable conflict resolution
  
  # Interactive for sensitive system files
  system_config:
    source: "system/app.conf"
    destination: "/etc/app/config"
    owner: "root"
    group: "root"
    mode: "640"
    copy: true
    backup: true
    interactive: true
    prompt_permissions: true     # Confirm permission changes
    prompt_ownership: true       # Confirm ownership changes
```

### Global vs Per-File Interactivity

```bash
# Enable interactivity globally
configr apply --interactive

# Or configure per-file in YAML (more granular control)
# Use the interactive: true option in file configuration
```

## Performance Optimization

### Cache-Friendly Configuration Patterns

Structure configurations for optimal caching:

```yaml
# Use includes to maximize cache benefits
includes:
  - path: "base.yaml"           # Changes rarely
  - path: "packages.yaml"       # Changes occasionally  
  - path: "development.yaml"    # Changes frequently

# Separate stable from volatile configuration
```

### Large Configuration Management

For configurations with many packages:

```yaml
# Split large package lists across files
includes:
  - glob: "packages/*.yaml"
    description: "All package files"

# Use package defaults to reduce repetition
package_defaults:
  apt: ["-y", "--no-install-recommends"]
  flatpak: ["--user"]

# Group related packages together for cache efficiency
packages:
  apt:
    # Base tools (stable)
    - git
    - curl
    - vim
```

### Cache Management Strategy

```bash
# Regular cache monitoring
configr cache stats

# Clear cache when troubleshooting
configr cache clear

# Disable caching for debugging
configr apply --optimize=false
```

## Security Best Practices

### File Permissions

Always specify appropriate permissions:

```yaml
files:
  # Private configuration
  ssh_config:
    source: "dotfiles/ssh_config"
    destination: "~/.ssh/config"
    mode: "600"                  # User read/write only
    backup: true
  
  # Shared configuration
  app_config:
    source: "configs/app.conf"
    destination: "~/.config/app/config"
    mode: "644"                  # User write, others read
    backup: true
  
  # System service
  service_config:
    source: "system/service.conf"
    destination: "/etc/service/config"
    owner: "root"
    group: "service"
    mode: "640"                  # Root write, group read
    copy: true
    backup: true
```

### Repository Security

Use secure repository configuration:

```yaml
repositories:
  apt:
    # Always specify GPG keys for custom repositories
    docker:
      uri: "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"
      key: "https://download.docker.com/linux/ubuntu/gpg"  # HTTPS URL
    
    nodejs:
      uri: "deb https://deb.nodesource.com/node_16.x focal main"
      key: "0x9FD3B784BC1C6FC31A8A0A1C1655A0AB68576280"    # Key ID

  flatpak:
    # Stick to well-known repositories
    flathub:
      url: "https://flathub.org/repo/flathub.flatpakrepo"  # HTTPS only
```

### Path Safety

Avoid dangerous path patterns:

```yaml
files:
  # Good: Explicit paths
  config:
    source: "configs/app.conf"
    destination: "~/.config/app/config"
  
  # Bad: Relative path traversal (configr will reject this)
  # malicious:
  #   source: "../../../etc/passwd"
  #   destination: "~/.passwd"
  
  # Good: Absolute paths when needed
  system_file:
    source: "/opt/configs/system.conf"
    destination: "/etc/app/system.conf"
    copy: true
    backup: true
```

## Common Anti-Patterns

### What to Avoid

1. **Overly Complex Flag Overrides**

```yaml
# Avoid: Too many per-package overrides
packages:
  apt:
    - "package1":
        flags: ["-y", "--force-yes", "--allow-unauthenticated"]
    - "package2":
        flags: ["-y", "--force-yes", "--allow-unauthenticated"]
    - "package3":
        flags: ["-y", "--force-yes", "--allow-unauthenticated"]

# Better: Use package defaults
package_defaults:
  apt: ["-y", "--force-yes", "--allow-unauthenticated"]
packages:
  apt:
    - package1
    - package2
    - package3
```

2. **No Backup Strategy**

```yaml
# Avoid: No backups for important files
files:
  important_config:
    source: "configs/important.conf"
    destination: "~/.important"
    # Missing backup: true

# Better: Always backup
files:
  important_config:
    source: "configs/important.conf"
    destination: "~/.important"
    backup: true
```

3. **Symlinks for System Files**

```yaml
# Avoid: Symlinks for system files
files:
  system_config:
    source: "system/app.conf"
    destination: "/etc/app/config"
    # Missing copy: true (defaults to symlink)

# Better: Copy system files
files:
  system_config:
    source: "system/app.conf"
    destination: "/etc/app/config"
    copy: true
    backup: true
```

4. **Ignoring Validation**

```bash
# Avoid: Applying without validation
configr apply

# Better: Always validate first
configr validate
configr apply --dry-run
configr apply
```

## Troubleshooting

### Common Issues and Solutions

1. **Configuration Not Loading**

```bash
# Check configuration search order
configr validate --verbose

# Use explicit config path
configr --config /path/to/config.yaml validate
```

2. **Include Resolution Problems**

```bash
# Debug include system
configr includes --verbose

# Check file paths and permissions
ls -la includes/
```

3. **Performance Issues**

```bash
# Check cache status
configr cache stats

# Clear cache and retry
configr cache clear
configr apply

# Disable caching temporarily
configr apply --optimize=false
```

4. **Permission Problems**

```bash
# Use interactive mode
configr apply --interactive

# Check file ownership and permissions
ls -la destination/path
```

5. **Package Installation Failures**

```bash
# Check package manager availability
configr init

# Use dry-run to see what would happen
configr apply --dry-run

# Check repository configuration
configr validate
```

### Debugging Techniques

1. **Verbose Output**

```bash
configr --verbose validate
configr --verbose apply --dry-run
```

2. **Step-by-Step Application**

```bash
# Validate first
configr validate

# Preview changes
configr apply --dry-run

# Apply interactively
configr apply --interactive
```

3. **Selective Features**

```bash
# Skip package removal
configr apply --remove-packages=false

# Disable optimization
configr apply --optimize=false
```

This guide provides a foundation for creating maintainable, secure, and efficient configr configurations. Adapt these patterns to your specific needs and environment.