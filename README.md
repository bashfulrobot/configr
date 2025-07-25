# Configr

A single binary configuration management tool for Ubuntu desktop systems. Configr provides package management, configuration file management, and desktop settings management similar to Ansible but contained in a single binary.

✅ **Currently Implemented**: APT, Flatpak, and Snap package management, Repository management, File management, DConf settings, Configuration validation

**Key Differentiators:**
- **Professional CLI**: Styled help pages and documentation via charmbracelet/fang
- **Rust-style Validation**: Clear, actionable error messages with suggestions
- **System Administrator Friendly**: Man pages, shell completions, and clean output

## Features

- **Smart Package Management**: Three-tier flag system with intelligent defaults for APT, Flatpak, and Snap
- **File Management**: Deploy and manage configuration files (dotfiles, system files) with symlinks
- **Desktop Configuration**: DConf settings management for any application using dconf
- **Modular Configuration**: Split configurations across multiple YAML files with includes
- **Backup Support**: Automatic backup of existing files before replacement
- **Professional CLI**: Styled help pages, auto-completion, and man page generation
- **Comprehensive Validation**: Rust-style error reporting with actionable suggestions and flag safety warnings

## Installation

```bash
# Build from source
git clone https://github.com/bashfulrobot/configr
cd configr

# Build with version injection using just
just build

# Or build manually
VERSION=$(git describe --tags --always --dirty)
go build -ldflags "-X github.com/bashfulrobot/configr/cmd/configr.Version=${VERSION}" -o configr .

# Or for a release build with a specific version
go build -ldflags "-X github.com/bashfulrobot/configr/cmd/configr.Version=v1.2.3" -o configr .

# Make it available system-wide
sudo mv configr /usr/local/bin/

# Install man page (optional)
configr man > configr.1
sudo mv configr.1 /usr/local/share/man/man1/

# Install shell completion (optional)
# For bash
configr completion bash > configr_completion.bash
sudo mv configr_completion.bash /etc/bash_completion.d/configr

# For zsh  
configr completion zsh > _configr
sudo mv _configr /usr/local/share/zsh/site-functions/
```

## Quick Start

1. **Create a configuration file** (`configr.yaml`):

```yaml
version: "1.0"

# Optional: Add package repositories
repositories:
  apt:
    python39:
      ppa: "deadsnakes/ppa"          # Ubuntu PPA format
  flatpak:
    flathub:
      url: "https://flathub.org/repo/flathub.flatpakrepo"

# Optional: Customize default flags for package managers
package_defaults:
  apt: ["-y"]                        # Override internal defaults
  flatpak: ["--user"]                # Prefer user installs

packages:
  apt:
    - git                            # Uses: ["-y"] from package_defaults
    - curl
    - vim
    - build-essential
  flatpak:
    - org.mozilla.Firefox
    - com.spotify.Client
  snap:
    - discord
    - code

files:
  vimrc:
    source: "dotfiles/vimrc"
    destination: "~/.vimrc"
    backup: true
  
  hosts:
    source: "system/hosts"
    destination: "/etc/hosts"
    owner: "root"
    group: "root"
    mode: "644"
    backup: true
    copy: true      # Copy for system files

dconf:
  settings:
    "/org/gnome/desktop/interface/gtk-theme": "'Adwaita-dark'"
    "/org/gnome/desktop/interface/icon-theme": "'Adwaita'"
    "/org/gnome/desktop/wm/preferences/button-layout": "'close,minimize,maximize:'"
```

2. **Validate your configuration**:

```bash
configr validate
```

3. **Apply your configuration**:

```bash
# Preview changes first (recommended)
configr apply --dry-run

# Apply the configuration
configr apply
```

4. **Check available commands**:

```bash
configr --help
```

## Configuration

### Basic Structure

Configr uses YAML configuration files with five main sections:

- `repositories`: Package repositories to add (APT PPAs, Flatpak remotes)
- `packages`: Software to install via package managers (APT, Flatpak, and Snap)
- `files`: Configuration files to deploy
- `dconf`: Desktop settings for any application using dconf
- `includes`: Additional configuration files to merge

### Package Management

Configr features a powerful three-tier flag system that provides intelligent defaults while allowing fine-grained control over package installation flags.

**Three-Tier Flag Resolution:**
1. **Internal Defaults** - Built-in sensible defaults for each package manager
2. **Package Defaults** - Your global defaults that override internal ones  
3. **Per-Package Flags** - Specific flags for individual packages (highest priority)

**Basic Package Installation:**

```yaml
packages:
  apt:
    - git                            # Uses intelligent defaults  
    - curl
    - build-essential
  flatpak:
    - org.mozilla.Firefox            # Firefox web browser
    - com.spotify.Client             # Spotify music player
  snap: 
    - discord                        # Discord chat application
    - code                           # Visual Studio Code editor
```

**Advanced Flag Control:**

```yaml
# Optional: Override default flags globally
package_defaults:
  apt: ["-y"]                        # Less opinionated than internal defaults
  flatpak: ["--user"]                # Install Flatpaks for user only
  snap: ["--classic"]                # Enable classic confinement by default

packages:
  apt:
    - git                            # Uses: ["-y"] from package_defaults
    - "docker.io":                   # Per-package override
        flags: ["-y", "--install-suggests"]
    - "./custom-app.deb":            # Local .deb file installation
        flags: ["-y", "--force-depends"]
    - "/opt/downloads/package.deb"   # Absolute path .deb file
        
  flatpak:
    - org.mozilla.Firefox            # Uses: ["--user"] from package_defaults
    - "com.spotify.Client":          # Override to system install  
        flags: ["--system"]
    - "org.gimp.GIMP":               # GIMP image editor
        flags: ["--user", "--or-update"]
        
  snap:
    - discord                        # Uses: ["--classic"] from package_defaults
    - "code":                        # Visual Studio Code
        flags: ["--classic"]         # Explicit classic confinement
    - "hello":                       # Simple snap (no special flags needed)
        flags: []
```

**Internal Default Flags (no configuration needed):**
- **APT**: `["-y", "--no-install-recommends"]` - Non-interactive, minimal installs
- **Flatpak**: `["--system", "--assumeyes"]` - System-wide, non-interactive installs
- **Snap**: `[]` - No defaults, snaps are interactive by design to prompt for permissions

**APT Package Management:**

Configr provides comprehensive APT support including local .deb file installation:

```yaml
packages:
  apt:
    # Repository packages (standard)
    - git
    - curl
    - build-essential
    
    # Repository packages with custom flags
    - "nginx":
        flags: ["-y", "--install-suggests"]
    
    # Local .deb files (relative paths)
    - "./downloads/custom-app.deb":
        flags: ["-y", "--force-depends"]
    
    # Local .deb files (absolute paths)
    - "/home/user/packages/proprietary.deb"
```

**APT Features:**
- **Repository packages**: Standard Ubuntu/Debian package installation
- **Local .deb files**: Install packages from filesystem paths
- **Mixed installations**: Seamlessly combine repository and local packages
- **Smart grouping**: Groups packages by flags to minimize system calls
- **State checking**: Avoids reinstalling already installed packages
- **Path validation**: Prevents malicious .deb paths with security checks

**Flatpak Package Management:**

Configr provides comprehensive Flatpak support for application installation:

```yaml
packages:
  flatpak:
    # Standard applications
    - org.mozilla.Firefox
    - com.spotify.Client
    - org.gimp.GIMP
    
    # With custom flags
    - "org.blender.Blender":
        flags: ["--user", "--or-update"]
    
    # KDE applications
    - org.kde.krita
    - org.kde.kdenlive
```

**Flatpak Features:**
- **Application ID validation**: Enforces reverse domain notation (org.mozilla.Firefox)
- **User vs system installation**: Control installation scope with `--user` or `--system`
- **Update handling**: Use `--or-update` to update existing installations
- **Smart grouping**: Groups applications by flags to minimize system calls
- **State checking**: Avoids reinstalling already installed applications

**Snap Package Management:**

Configr provides comprehensive Snap support for package installation:

```yaml
packages:
  snap:
    # Standard packages
    - discord
    - hello
    - snap-store
    
    # Applications requiring classic confinement
    - "code":
        flags: ["--classic"]
    - "slack":
        flags: ["--classic"]
    
    # Development tools
    - "postman":
        flags: ["--classic"]
```

**Snap Features:**
- **Package name validation**: Enforces Snap naming conventions (lowercase, hyphens)
- **Classic confinement support**: Many desktop apps need `--classic` for filesystem access
- **Individual installation**: Handles one package at a time (Snap design limitation)
- **State checking**: Avoids reinstalling already installed packages
- **Interactive prompts**: Respects Snap's interactive permission model

**Common Flag Examples:**
- **APT**: `--install-suggests`, `--allow-unauthenticated`, `--force-depends`
- **Flatpak**: `--user` vs `--system`, `--or-update`, `--assumeyes`
- **Snap**: `--classic` for desktop apps, `--devmode` for development, `--dangerous` for local installs

### Repository Management

Configr supports managing package repositories for both APT and Flatpak:

```yaml
repositories:
  apt:
    python39:                     # Repository identifier
      ppa: "deadsnakes/ppa"       # Ubuntu PPA format
    docker:
      uri: "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"
      key: "https://download.docker.com/linux/ubuntu/gpg.asc"  # GPG key (optional)
    nodejs:
      uri: "deb https://deb.nodesource.com/node_16.x focal main"
      key: "0x9FD3B784BC1C6FC31A8A0A1C1655A0AB68576280"  # Key ID format
  
  flatpak:
    flathub:                      # Remote name
      url: "https://flathub.org/repo/flathub.flatpakrepo"  # Required
      user: false                 # Optional: system-wide (default)
    kde:
      url: "https://distribute.kde.org/kdeapps.flatpakrepo"
      user: true                  # User-only installation
```

**APT Repository Options:**
- `ppa`: Ubuntu PPA in `user/repo` format (uses `add-apt-repository`)
- `uri`: Custom repository URI in standard sources.list format
- `key`: GPG key URL (HTTPS) or keyserver key ID (hex format)

**Flatpak Repository Options:**
- `url`: Repository URL (required) - typically `.flatpakrepo` files
- `user`: Install for user only vs system-wide (default: false)

**Repository Features:**
- **Validation**: Comprehensive format checking with helpful error messages
- **Security**: HTTPS enforcement for keys, path safety validation
- **Flexibility**: Support both PPA shortcuts and full repository URIs
- **Integration**: Works with existing three-tier package flag system

### File Management

Deploy any file to any location with optional permissions and backup. Configr supports both symlink and copy modes:

```yaml
files:
  # Dotfile example (default symlink mode)
  bashrc:
    source: "dotfiles/bashrc"
    destination: "~/.bashrc"
    backup: true

  # System file example with copy mode
  docker_daemon:
    source: "system/docker/daemon.json"
    destination: "/etc/docker/daemon.json"
    owner: "root"
    group: "root"
    mode: "644"
    backup: true
    copy: true    # Copy instead of symlink for system files

  # Configuration that needs to be independent
  app_config:
    source: "configs/app.conf"
    destination: "~/.config/app/config"
    copy: true    # Ensures config won't change if source is modified
```

**Symlink vs Copy Mode:**

- **Symlink (default)**: Changes to source files are immediately reflected. Best for dotfiles where you want live updates.
- **Copy mode**: Creates independent file copies. Best for system files or when you need stable configurations.

**File Options:**
- `source` (required): Path to source file
- `destination` (required): Where to place the file
- `owner` (optional): File owner (preserves existing if omitted)
- `group` (optional): File group (preserves existing if omitted)  
- `mode` (optional): File permissions (preserves existing if omitted)
- `backup` (optional): Backup existing file before replacement
- `copy` (optional): Copy file instead of creating symlink (default: false)

### Desktop Settings

Configure any application that uses dconf for settings storage. This includes GNOME desktop environment, many GTK applications, and other desktop applications:

```yaml
dconf:
  settings:
    # GNOME Desktop Environment
    "/org/gnome/desktop/interface/gtk-theme": "'Adwaita-dark'"
    "/org/gnome/desktop/interface/icon-theme": "'Adwaita'"
    "/org/gnome/desktop/wm/preferences/button-layout": "'close,minimize,maximize:'"
    "/org/gnome/desktop/interface/clock-show-seconds": "true"
    
    # GNOME Terminal
    "/org/gnome/terminal/legacy/profiles:/:default/background-color": "'rgb(23,20,33)'"
    "/org/gnome/terminal/legacy/profiles:/:default/foreground-color": "'rgb(208,207,204)'"
    
    # Nautilus File Manager
    "/org/gnome/nautilus/preferences/default-folder-viewer": "'icon-view'"
    "/org/gnome/nautilus/preferences/show-hidden-files": "true"
    
    # Text Editor (gedit)
    "/org/gnome/gedit/preferences/editor/scheme": "'oblivion'"
    
    # Application-specific (e.g., Guake terminal)
    "/apps/guake/general/window-height": "40"
    "/apps/guake/general/use-default-font": "false"
```

**DConf Features:**
- **Universal compatibility**: Works with any application using dconf
- **Immediate application**: Settings applied instantly without restart
- **Type safety**: Supports strings, booleans, numbers, and arrays
- **Validation**: Comprehensive path and value validation
- **Dry-run support**: Preview changes before applying

### Modular Configuration

Split large configurations across multiple files:

```yaml
# configr.yaml
version: "1.0"
includes:
  - "packages.yaml"           # Explicit file
  - "packages/"               # Directory with default.yaml
  - "packages/apt/"           # Subdirectory with default.yaml
  - "dotfiles/vim.yaml"       # Specific configuration

packages:
  apt: ["base-tools"]         # Can still have inline config
```

**Include Path Resolution:**
- `packages.yaml` → loads `packages.yaml`
- `packages/` → loads `packages/default.yaml`
- `packages/apt/` → loads `packages/apt/default.yaml`
- Auto-extension: `packages` → tries `packages.yaml`

**Note**: Use trailing slashes (`/`) for directories to improve readability.

## Configuration Validation

Configr provides comprehensive validation with Rust-inspired error reporting for excellent user experience:

### Validation Features

- **Schema validation** - Ensures required fields and correct formats
- **File existence checks** - Verifies source files exist before deployment  
- **Permission validation** - Checks file modes and ownership
- **Path safety** - Prevents unsafe destinations like `../../../etc/passwd`
- **Package validation** - Ensures valid package names
- **DConf validation** - Validates dconf paths and value formats

### Error Reporting

When validation fails, you get clear, actionable error messages:

```
error: source file not found
  --> configr.yaml:15:5
   |
   | files.vimrc.source: dotfiles/vimrc
   |                     ^^^^^^^^^^^^^^ file does not exist
   |
   = help: create the file or check the path
   = note: looked for: /home/user/dotfiles/vimrc

Quick fixes:
  • Create missing file: dotfiles/vimrc
  • Check if path is correct
```

**Benefits:**
- **Immediate feedback** - Catch errors before deployment
- **Clear guidance** - Know exactly what to fix and how
- **Safe operations** - Prevent accidental system damage
- **Better experience** - Less time debugging configuration issues

### CLI Visual Improvements

Configr's professional CLI interface stands out from typical command-line tools:

**Standard CLI Help vs Configr:**

```
# Typical CLI tool help
Usage:
  tool [flags]
  tool [command]
Flags:
  -h, --help   help for tool

# Configr with fang integration
  USAGE

    configr [command] [--flags]

  EXAMPLES

    configr validate # Validate default config
    configr --config custom.yaml validate # Use custom config

  COMMANDS

    validate [config-file] [--flags]  Validate configuration file

  FLAGS

    -c --config     Config file path
    -v --verbose    Verbose output
```

The improved formatting makes configr feel trustworthy and professional for system administration tasks.

## Examples

See the `examples/` directory for complete configuration examples:

- `examples/desktop-dev.yaml` - Development environment setup
- `examples/apt-simple.yaml` - Basic APT package management
- `examples/apt-packages.yaml` - Comprehensive APT features showcase  
- `examples/advanced-flags.yaml` - Three-tier flag system demonstration

## CLI Commands

Configr features a professional CLI interface with styled help pages and comprehensive tooling.

### Core Commands

- `configr validate [file]` - Validate configuration without applying changes
- `configr apply [file]` - Apply configuration changes to your system
- `configr help [command]` - Show help for any command

### Documentation & Setup

- `configr man` - Generate Unix man pages for system installation
- `configr completion [shell]` - Generate shell completions (bash, zsh, fish)
- `configr --version` - Show version and build information

### Professional CLI Features

- **Styled Help Pages**: Clean, formatted help output with clear sections
- **Automatic Completions**: Tab completion for commands, flags, and file paths
- **Man Page Generation**: Standard Unix documentation via `man configr`
- **Consistent Formatting**: Professional appearance suitable for system administration

### Global Flags

- `-c, --config <file>` - Specify config file path
- `-v, --verbose` - Enable verbose output
- `--no-color` - Disable colored output

### Usage Examples

**Basic Operations:**
```bash
# Validate default configuration
configr validate

# Apply configuration changes to system
configr apply

# Preview changes without applying them (dry-run)
configr apply --dry-run

# Apply specific configuration file
configr apply my-config.yaml --verbose

# Use custom config file location
configr --config /path/to/config.yaml apply
```

**Documentation & Setup:**
```bash
# View styled help (much prettier than standard CLI tools)
configr --help
configr validate --help

# Generate and install man page
configr man > configr.1
sudo mv configr.1 /usr/local/share/man/man1/
man configr

# Set up shell completions for better UX
configr completion bash > /etc/bash_completion.d/configr
# Restart shell or source: source /etc/bash_completion.d/configr
```

**Professional Features:**
```bash
# All commands feature consistent, styled output
configr --version        # Styled version information
configr validate --help  # Professional help formatting
```

## Configuration File Locations

Configr searches for configuration files in order:

1. Explicit path via `--config` flag
2. Environment variable `CONFIGR_CONFIG`  
3. Current directory (`./configr.yaml`)
4. XDG config directory (`~/.config/configr/configr.yaml`)
5. Home directory (`~/configr.yaml`)
6. System config (`/etc/configr/configr.yaml`)
7. Local system config (`/usr/local/etc/configr/configr.yaml`)

## License

MIT License - see LICENSE file for details.
