# Configr Quick Reference

A concise reference for configr commands and configuration syntax.

## Commands

### Core Operations
```bash
configr validate                    # Validate configuration
configr apply                       # Apply configuration
configr apply --dry-run            # Preview changes
configr apply --interactive        # Enable interactive prompts
configr init                        # Check system dependencies
```

### Cache Management  
```bash
configr cache stats                 # Show cache statistics
configr cache clear                 # Clear all cache data
configr cache info                  # Show cache information
```

### Advanced Features
```bash
configr includes                    # Debug include system
configr packages                    # Package management operations
configr restore                     # Restore from backups
```

### Documentation
```bash
configr --help                      # General help
configr [command] --help           # Command-specific help
configr man                         # Generate man pages
configr completion bash             # Shell completion
```

## Configuration Syntax

### Basic Structure
```yaml
version: "1.0"
packages:
  apt: [...]
  flatpak: [...]
  snap: [...]
files:
  name:
    source: "path"
    destination: "path"
dconf:
  settings:
    "/path/to/setting": "'value'"
```

### Package Management
```yaml
# Simple packages
packages:
  apt:
    - git
    - curl
  flatpak:
    - org.mozilla.Firefox
  snap:
    - code

# With custom flags
packages:
  apt:
    - "docker.io":
        flags: ["-y", "--install-suggests"]

# Package defaults
package_defaults:
  apt: ["-y"]
  flatpak: ["--user"]
```

### File Management
```yaml
files:
  # Basic dotfile (symlink mode)
  bashrc:
    source: "dotfiles/bashrc"
    destination: "~/.bashrc"
    backup: true
  
  # System file (copy mode)
  nginx_config:
    source: "system/nginx.conf"
    destination: "/etc/nginx/nginx.conf"
    owner: "root"
    group: "root"
    mode: "644"
    copy: true
    backup: true
    interactive: true
```

### Repository Management
```yaml
repositories:
  apt:
    python39:
      ppa: "deadsnakes/ppa"
    docker:
      uri: "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"
      key: "https://download.docker.com/linux/ubuntu/gpg"
  
  flatpak:
    flathub:
      url: "https://flathub.org/repo/flathub.flatpakrepo"
      user: false
```

### Include System
```yaml
includes:
  # Simple includes
  - path: "packages.yaml"
  - path: "packages/"              # Loads packages/default.yaml
  
  # Glob patterns
  - glob: "packages/*.yaml"
  
  # Conditional includes
  - path: "environments/dev.yaml"
    optional: true
    conditions:
      - type: "env"
        value: "NODE_ENV=development"
      - type: "hostname"
        value: "workstation"
        operator: "contains"
```

### DConf Settings
```yaml
dconf:
  settings:
    "/org/gnome/desktop/interface/gtk-theme": "'Adwaita-dark'"
    "/org/gnome/terminal/legacy/profiles:/:default/font": "'Monospace 12'"
    "/org/gnome/desktop/interface/clock-show-seconds": "true"
```

## Common Patterns

### Development Environment
```yaml
version: "1.0"
packages:
  apt:
    - git
    - curl
    - vim
    - build-essential
  flatpak:
    - org.mozilla.Firefox
  snap:
    - code

files:
  bashrc:
    source: "dotfiles/bashrc"
    destination: "~/.bashrc"
    backup: true
```

### Modular Configuration
```yaml
version: "1.0"
includes:
  - path: "common/base.yaml"
  - glob: "packages/*.yaml"
  - path: "hosts/${HOSTNAME}.yaml"
    optional: true
```

### Interactive Deployment
```yaml
files:
  ssh_config:
    source: "dotfiles/ssh_config"
    destination: "~/.ssh/config"
    mode: "600"
    backup: true
    interactive: true
    prompt_permissions: true
```

## Flags Reference

### Global Flags
- `-c, --config <file>` - Config file path
- `-v, --verbose` - Verbose output  
- `--no-color` - Disable colors

### Apply Command Flags
- `--dry-run` - Preview changes
- `--interactive` - Enable interactive prompts
- `--remove-packages=false` - Skip package removal
- `--optimize=false` - Disable caching
- `--preview` - Show config preview

### Package Manager Flags

**APT (Internal defaults: `["-y", "--no-install-recommends"]`)**
- `--install-suggests` - Install suggested packages
- `--allow-unauthenticated` - Allow unauthenticated packages
- `--force-depends` - Force dependency resolution

**Flatpak (Internal defaults: `["--system", "--assumeyes"]`)**
- `--user` - User-only installation
- `--system` - System-wide installation  
- `--or-update` - Update if already installed

**Snap (Internal defaults: `[]`)**
- `--classic` - Classic confinement (required for many desktop apps)
- `--devmode` - Development mode
- `--dangerous` - Install unsigned packages

## File Locations

### Config Files (searched in order)
1. `--config` flag path
2. `$CONFIGR_CONFIG` environment variable
3. `./configr.yaml`
4. `~/.config/configr/configr.yaml`
5. `~/configr.yaml`
6. `/etc/configr/configr.yaml`
7. `/usr/local/etc/configr/configr.yaml`

### Data Locations
- **State tracking**: `~/.config/configr/state.json`
- **Cache data**: `~/.cache/configr/`
- **Backups**: `~/.config/configr/backups/`

## Troubleshooting

### Common Issues
```bash
# Validation errors
configr validate --verbose

# Include resolution problems  
configr includes --verbose

# Performance issues
configr cache stats
configr cache clear

# Package installation failures
configr init
configr apply --dry-run

# Permission problems
configr apply --interactive
```

### Debug Commands
```bash
# Show detailed help
configr [command] --help

# Preview all changes
configr apply --dry-run --verbose

# Check system state
configr init --verbose

# Analyze includes
configr includes --verbose

# Clear cache and retry
configr cache clear && configr apply
```

## Examples

All examples available in `examples/` directory:
- `examples/desktop-dev.yaml` - Development setup
- `examples/interactive-configuration.yaml` - Interactive features
- `examples/repository-management.yaml` - Repository setup
- `examples/performance-optimization.yaml` - Caching demo
- `examples/state-management.yaml` - Package removal demo