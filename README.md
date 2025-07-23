# Configr

** WORK IN PROGRESS - NOT WORKING, HEAVY DEV. **

-----------------------------------------------------

A single binary configuration management tool for Ubuntu desktop systems. Configr provides package management, configuration file management, and desktop settings management similar to Ansible but contained in a single binary.

**Key Differentiators:**
- **Professional CLI**: Styled help pages and documentation via charmbracelet/fang
- **Rust-style Validation**: Clear, actionable error messages with suggestions
- **System Administrator Friendly**: Man pages, shell completions, and clean output

## Features

- **Package Management**: Install and manage packages via APT, Flatpak, and Snap
- **File Management**: Deploy and manage configuration files (dotfiles, system files) with symlinks
- **Desktop Configuration**: Manage GNOME dconf settings
- **Modular Configuration**: Split configurations across multiple YAML files with includes
- **Backup Support**: Automatic backup of existing files before replacement
- **Professional CLI**: Styled help pages, auto-completion, and man page generation
- **Comprehensive Validation**: Rust-style error reporting with actionable suggestions

## Installation

```bash
# Build from source
git clone https://github.com/bashfulrobot/configr
cd configr
go build -o configr .

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

packages:
  apt:
    - git
    - curl
    - vim
  flatpak:
    - org.mozilla.firefox
    - com.visualstudio.code
  snap:
    - discord

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

dconf:
  settings:
    "/org/gnome/desktop/interface/gtk-theme": "'Adwaita-dark'"
    "/org/gnome/desktop/interface/icon-theme": "'Adwaita'"
```

2. **Validate your configuration**:

```bash
configr validate
```

3. **Check available commands**:

```bash
configr --help
```

## Configuration

### Basic Structure

Configr uses YAML configuration files with four main sections:

- `packages`: Software to install via package managers
- `files`: Configuration files to deploy
- `dconf`: GNOME desktop settings
- `includes`: Additional configuration files to merge

### Package Management

Install packages from multiple sources:

```yaml
packages:
  apt:
    - git
    - curl
    - build-essential
  flatpak:
    - org.mozilla.firefox
    - com.visualstudio.code
  snap:
    - discord
    - slack
```

### File Management

Deploy any file to any location with optional permissions and backup:

```yaml
files:
  # Dotfile example
  bashrc:
    source: "dotfiles/bashrc"
    destination: "~/.bashrc"
    backup: true

  # System file example  
  docker_daemon:
    source: "system/docker/daemon.json"
    destination: "/etc/docker/daemon.json"
    owner: "root"
    group: "root"
    mode: "644"
    backup: true
```

**File Options:**
- `source` (required): Path to source file
- `destination` (required): Where to place the file
- `owner` (optional): File owner (preserves existing if omitted)
- `group` (optional): File group (preserves existing if omitted)  
- `mode` (optional): File permissions (preserves existing if omitted)
- `backup` (optional): Backup existing file before replacement

### Desktop Settings

Configure GNOME settings via dconf:

```yaml
dconf:
  settings:
    "/org/gnome/desktop/interface/gtk-theme": "'Adwaita-dark'"
    "/org/gnome/desktop/wm/preferences/button-layout": "'close,minimize,maximize:'"
    "/org/gnome/terminal/legacy/profiles:/:b1dcc9dd-5262-4d8d-a863-c897e6d979b9/background-color": "'rgb(23,20,33)'"
```

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
- **DConf validation** - Validates GNOME settings paths

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

## CLI Commands

Configr features a professional CLI interface with styled help pages and comprehensive tooling.

### Core Commands

- `configr validate [file]` - Validate configuration without applying changes
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

# Validate specific file with verbose output
configr validate my-config.yaml --verbose

# Use custom config file location
configr --config /path/to/config.yaml validate
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
