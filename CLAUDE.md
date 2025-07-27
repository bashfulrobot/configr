# Configr

## Project Overview

### Summary & Purpose

Written by the staff member "Gopher", this application will be a scaled-down version of desktop Linux (Ubuntu) configuration management, akin to Ansible, but in a single binary.

**Application Identity:**

- configr
- github.com/bashfulrobot/configr

### Current Implementation Status

**✅ Implemented (Production Ready):**

- APT package management (repository + local .deb files)
- Flatpak package management (application installation with reverse domain validation)
- Snap package management (package installation with naming convention validation)
- Package removal system (removes packages when removed from configuration)
- File removal system (removes files/dotfiles when removed from configuration)
- Repository management (APT PPAs/custom repos + Flatpak remotes)
- File management system (symlink/copy modes with backup)
- DConf configuration management (desktop settings for any dconf-using application)
- Configuration validation with Rust-style error reporting
- Three-tier package flag system
- State tracking and management for package and file removal
- Configuration and system state caching for performance optimization
- Professional CLI with charmbracelet/fang integration
- Comprehensive test coverage (170+ tests)

**✅ Recently Implemented:**

- Init command for tool installation verification
- Interactive conflict resolution system
- File diff preview before replacement
- Advanced permission handling with interactive prompts
- Advanced include system with glob patterns and conditional includes
- System detection for OS, hostname, and environment-based configuration

**📋 Planned Features:**

- Backup restoration system
- Enhanced interactive features

### Key Technical Differentiators

- **Professional CLI**: Styled help pages, man pages, shell completions
- **Rust-style Validation**: Clear, actionable error messages with suggestions
- **Three-tier Flag System**: Internal defaults → User defaults → Per-package overrides
- **Unified File Management**: Single interface for dotfiles and system files
- **Exceptional UX**: Color output, emoji icons, spinners, clear feedback

---

## Development Guidelines

### Core Principles

- **Single Feature Implementation**: Implement features one at a time unless dependencies require otherwise
- **Exceptional User Experience**: Color output, emoji icons, spinners, simple-to-read output (not verbose unless requested)
- **Rust-style Error Reporting**: Extremely easy to decipher and act on error messages
- **System Admin Friendly**: Professional CLI suitable for production system administration

### Guardrails & Rules

- **Never put Claude branding in any commit**
- **Commits should use conventional commits, with emojis, and if there is ever a version tag, follow semver.
- **Always write tests when feasible** and run as appropriate
- **Proper error handling** with meaningful stdout/stderr from external tools
- **Favor existing patterns** for consistency, suggest improvements when beneficial
- **Update documentation** when requested: README (end-user POV) + CLAUDE.md (architecture)
- **Always consider the relationship between schema, caching, and actions**

### Development Workflow

- **Follow Cobra best practices** for CLI verbs and structure
- **Shell out to system tools** (apt, flatpak, snap, dconf)
- **Check tool availability** on init with installation offers
- **Maintain consistency** across all implementations

### Testing Requirements

- Comprehensive test coverage for all new features
- Integration tests using actual system commands
- Validation tests for configuration schemas
- Error handling and edge case coverage

---

## Technical Specifications

### Configuration Schema (YAML Structure)

**Top-level Structure:**

```yaml
version: "1.0"     # Configuration schema version
includes: [...]    # Optional: Include additional YAML files
package_defaults:  # Optional: Override internal package manager defaults
  apt: ["-y"]
  flatpak: ["--user"]
  snap: ["--classic"]
repositories:      # Repository management (apt, flatpak)
  apt: [...]
  flatpak: [...]
packages:          # Package management (apt, flatpak, snap)
  apt: [...]
  flatpak: [...]
  snap: [...]
files:             # Unified file management
  name:
    source: "path"
    destination: "path"
    # Optional: owner, group, mode, backup, copy
dconf:             # DConf settings for any dconf-using application
  settings:
    "/path/to/setting": "'value'"
```

**Three-Tier Package Flag System:**

1. **Internal Defaults (Tier 1)**: Built into configr
   - APT: `["-y", "--no-install-recommends"]`
   - Snap: `[]` (interactive by design)
   - Flatpak: `["--system", "--assumeyes"]`
2. **User Package Defaults (Tier 2)**: Global overrides in `package_defaults`
3. **Per-Package Flags (Tier 3)**: Highest priority, package-specific overrides

**Repository Management Schema:**

```yaml
repositories:
  apt:
    python39:                     # Repository name/identifier
      ppa: "deadsnakes/ppa"       # Ubuntu PPA format: "user/repo"
    docker:
      uri: "deb [arch=amd64] https://download.docker.com/linux/ubuntu focal stable"
      key: "https://download.docker.com/linux/ubuntu/gpg"  # GPG key URL or key ID
  flatpak:
    flathub:                      # Remote name
      url: "https://flathub.org/repo/flathub.flatpakrepo"  # Required: repository URL
      user: false                 # Optional: user-only install (default: system)
    kde:
      url: "https://distribute.kde.org/kdeapps.flatpakrepo"
      user: true
```

**File Management Schema:**

```yaml
files:
  filename:
    source: "path/to/source"         # Required: source file path
    destination: "/target/path"      # Required: where to place the file
    owner: "user"                    # Optional: file owner (preserves if omitted)
    group: "group"                   # Optional: file group (preserves if omitted)
    mode: "644"                      # Optional: file permissions (preserves if omitted)
    backup: true                     # Optional: backup existing file before replacing
    copy: true                       # Optional: copy file instead of symlinking (default: false)
    interactive: true                # Optional: enable interactive conflict resolution
    prompt_permissions: true         # Optional: prompt for permission changes
    prompt_ownership: true           # Optional: prompt for ownership changes
```

### Command Interface (CLI Design)

**Command Structure:**

```bash
configr [global-flags] <command> [command-flags] [arguments]
```

**Available Commands:**

- `configr validate [file]` - Validate configuration without applying changes
- `configr apply [file]` - Apply configuration changes to system
- `configr apply --dry-run` - Preview changes without applying
- `configr apply --interactive` - Enable interactive prompts for conflicts and permissions
- `configr apply --remove-packages=false` - Skip package removal operations
- `configr apply --optimize=false` - Disable caching and optimization
- `configr cache stats` - Show cache usage statistics
- `configr cache clear` - Clear all cached data
- `configr help [command]` - Show help for any command
- `configr man` - Generate Unix man pages
- `configr completion [shell]` - Generate shell completions
- `configr --version` - Show version and build information

**Global Flags:**

- `-c, --config <file>` - Specify config file path
- `-v, --verbose` - Enable verbose output
- `--no-color` - Disable colored output

**Config File Discovery (in order):**

1. Explicit path via `--config` flag
2. Environment variable `CONFIGR_CONFIG`
3. Current directory (`./configr.yaml`)
4. XDG config directory (`~/.config/configr/configr.yaml`)
5. Home directory (`~/configr.yaml`)
6. System config (`/etc/configr/configr.yaml`)
7. Local system config (`/usr/local/etc/configr/configr.yaml`)

### Error Handling & UX Standards

**Validation Features:**

- Schema validation (required fields, correct formats)
- File existence checks (source files exist before deployment)
- Permission validation (file modes and ownership)
- Path safety (prevents unsafe destinations like `../../../etc/passwd`)
- Package name validation (manager-specific rules)
- DConf path validation (dconf settings paths and value formats)

**Error Reporting Style (Rust-inspired):**

```
error: source file not found
  --> configr.yaml:15:5
   |
   | files.vimrc.source: dotfiles/vimrc
   |                     ^^^^^^^^^^^^^^ file does not exist
   |
   = help: create the file or check the path
   = note: looked for: /home/user/dotfiles/vimrc
   = suggestion: did you mean "dotfiles/.vimrc"?
```

**User Experience Standards:**

- Structured logging with charmbracelet/log
- Clear success/warning/error indicators (✓, ⚠, ✗)
- Position-aware error reporting with line/column numbers
- Verbose mode for detailed operation insights
- Quick fix suggestions with immediate actionable solutions

### External Dependencies

**Required Libraries:**

- **Viper**: Configuration file management
- **Cobra**: CLI interface framework
- **charmbracelet/fang**: Enhanced CLI presentation and tooling
- **charmbracelet/log**: Structured, colorful logging

**Optional Libraries (for consideration):**

- [glow](https://github.com/charmbracelet/glow) - Markdown rendering
- [charmbracelet/huh](https://github.com/charmbracelet/huh) - Terminal forms and prompts
- [charmbracelet/skate](https://github.com/charmbracelet/skate) - Key/value store if needed

---

## Implementation Status & Architecture

### ✅ Implemented Features

#### File Management System

**Core Components:**

- **FileManager (`internal/pkg/files.go`)** - Central orchestrator for all file operations
- **Dual deployment modes**: Symlink (default) and copy modes
- **Backup system**: Timestamped backups of existing files before replacement
- **Permission management**: Sets owner, group, and mode when specified
- **Path resolution**: Handles relative, absolute, and `~` user expansion
- **Safety checks**: Permission validation and path safety verification

**Key Implementation Details:**

- **Symlink Mode (default)**: Live updates, clear ownership, safe removal, backup restoration
- **Copy Mode**: Static snapshots, independence, standard files, no symlink overhead
- **Path Resolution Hierarchy**: Absolute paths → Relative to config dir → User home expansion
- **Comprehensive validation**: Source file existence, destination safety, permission checks

#### APT Package Management

**Core Components:**

- **AptManager (`internal/pkg/apt.go`)** - Central orchestrator for all APT operations
- **Repository packages**: Standard Ubuntu/Debian package installation
- **Local .deb files**: Installation from local filesystem paths with security validation
- **Smart grouping**: Groups packages by resolved flags to minimize system calls
- **State management**: Checks existing package status to avoid unnecessary operations

**Three-Tier Flag Resolution Implementation:**

```go
// Tier 3: Per-package flags (highest priority)
if pkg.Flags != nil {
    return pkg.Flags
}
// Tier 2: User package defaults
if userDefaults, exists := packageDefaults["apt"]; exists {
    return userDefaults
}
// Tier 1: Internal defaults
return config.GetDefaultFlags("apt")
```

**Local .deb File Support:**

- Path validation with security checks (prevents `../../../etc/passwd.deb`)
- Relative path resolution to absolute paths
- Mixed installations (repository + local packages seamlessly)
- File existence verification before installation

**Installation Logic:**

1. Check apt command availability
2. Group packages by resolved flags
3. Separate local .deb files from repository packages
4. Check installation status to avoid duplicates
5. Install in optimized batches
6. Provide clear success/failure feedback

#### Configuration Validation System

**Validation Integration:**

- APT-specific validation extends existing framework
- Package name validation (repository packages follow apt naming conventions)
- .deb file validation (local files must have valid paths with security checks)
- Flag safety warnings (potentially dangerous flags like `--force-yes`)
- Availability checking (verifies apt command exists on system)

**Advanced Features:**

- Rust-style error reporting with actionable suggestions
- Quick fix recommendations with specific file paths
- Circular include detection for configuration files
- Position-aware error reporting with line/column numbers

#### Professional CLI Integration

**Fang Integration Benefits:**

- **Styled help pages**: Professional, visually appealing help output
- **Automatic version handling**: Built-in version command with styling
- **Man page generation**: Automatic Unix man page creation
- **Shell completions**: Auto-completion for bash, zsh, fish
- **Minimal boilerplate**: Reduced CLI code by ~60% while adding features

**Enhanced User Experience:**

- Professional formatting with clear sections (USAGE, EXAMPLES, COMMANDS, FLAGS)
- Consistent visual structure across all help pages
- Clean, readable output suitable for system administration
- Automatic features without additional maintenance code

#### Repository Management System

**Core Components:**

- **RepositoryManager (`internal/pkg/repositories.go`)** - Central orchestrator for repository operations
- **APT repository support**: PPA and custom repository management via `add-apt-repository`
- **Flatpak repository support**: Remote management via `flatpak remote-add`
- **GPG key handling**: Automatic key installation from URLs or keyservers
- **Command availability checks**: Validates required tools are installed before operation

**Key Implementation Details:**

- **PPA Support**: Uses `add-apt-repository ppa:user/repo` for Ubuntu PPAs
- **Custom APT Repos**: Supports full `sources.list` format with optional GPG keys
- **Flatpak Remotes**: Manages `.flatpakrepo` files with user/system installation options
- **GPG Key Management**: Handles both HTTPS URLs (`.gpg`/`.asc`) and keyserver key IDs
- **Dry-run Support**: Preview repository changes without making system modifications
- **Error Handling**: Clear feedback with specific installation requirements

**APT Repository Features:**

- PPA format validation (`user/repo`)
- Custom repository URI validation (must start with `deb`/`deb-src`)
- GPG key validation (HTTPS URLs or hex key IDs)
- Automatic key installation before repository addition

**Flatpak Repository Features:**

- Repository URL validation (HTTPS enforcement for security)
- Remote name validation (alphanumeric with hyphens/underscores)
- User vs system installation control
- `--if-not-exists` flag to prevent duplicate remotes

#### DConf Configuration Management

**Core Components:**

- **DConfManager (`internal/pkg/dconf.go`)** - Central orchestrator for dconf settings management
- **Universal compatibility**: Works with any application using dconf (GNOME, GTK apps, etc.)
- **Comprehensive validation**: Path format, value type, and structure validation
- **Multiple operations**: Set, get, reset, list, and dump dconf settings
- **Type safety**: Supports strings, booleans, numbers, and complex data structures

**Key Implementation Details:**

- **Setting Management**: Uses `dconf write` for applying configuration changes
- **Value Validation**: Comprehensive checks for proper quoting and data types
- **Path Validation**: Ensures dconf paths start with `/` and have proper structure
- **Dry-run Support**: Preview changes without modifying system settings
- **Error Handling**: Clear feedback with specific dconf command requirements

**DConf Features:**

- Settings path validation (must start with `/`, no double slashes)
- Value format validation with helpful warnings for unquoted strings
- Support for all dconf data types (strings, booleans, integers, arrays)
- Immediate application without application restart required
- Integration with existing configuration validation system

**Advanced Operations:**

- `GetSetting()`: Retrieve current values from dconf database
- `ResetSetting()`: Reset settings to application defaults
- `ListSettings()`: List all settings under a given path
- `DumpSettings()`: Export settings in ini-like format for backup
- `ValidateSettings()`: Pre-validate settings before application

**Application Coverage:**

- GNOME Desktop Environment (themes, wallpapers, behavior)
- GNOME Applications (Terminal, Nautilus, Text Editor, etc.)
- GTK Applications (any app using GSettings/dconf)
- Third-party applications (Guake, etc.)

#### Flatpak Package Management

**Core Components:**

- **FlatpakManager (`internal/pkg/flatpak.go`)** - Central orchestrator for Flatpak application management
- **Universal application support**: Manages any Flatpak application with reverse domain notation
- **Three-tier flag resolution**: Supports per-package, user defaults, and internal defaults
- **Smart grouping**: Groups applications by flags to minimize system calls
- **State management**: Checks installation status to avoid unnecessary operations

**Key Implementation Details:**

- **Application Installation**: Uses `flatpak install` with comprehensive flag support
- **Reverse Domain Validation**: Enforces proper application ID format (org.mozilla.Firefox)
- **Scope Management**: Supports both user (`--user`) and system (`--system`) installations
- **Update Handling**: Supports `--or-update` flag for existing application updates
- **Dry-run Support**: Preview changes without modifying system state

**Flatpak Features:**

- Application ID validation with reverse domain notation enforcement
- User vs system installation scope control
- Smart package grouping by resolved flags
- State checking to prevent duplicate installations
- Integration with existing three-tier flag system

**Advanced Operations:**

- `InstallPackages()`: Install applications with flag resolution
- `UninstallPackage()`: Remove applications with custom flags
- `ListInstalledPackages()`: Enumerate installed applications
- `UpdatePackages()`: Update all installed applications
- `ValidatePackageNames()`: Pre-validate application IDs

#### Snap Package Management

**Core Components:**

- **SnapManager (`internal/pkg/snap.go`)** - Central orchestrator for Snap package management
- **Universal package support**: Manages any Snap package with naming convention validation
- **Three-tier flag resolution**: Supports per-package, user defaults, and internal defaults
- **Individual installation**: Handles packages one at a time (Snap design requirement)
- **State management**: Checks installation status to avoid unnecessary operations

**Key Implementation Details:**

- **Package Installation**: Uses `snap install` with comprehensive flag support
- **Name Validation**: Enforces Snap naming conventions (lowercase, hyphens, length limits)
- **Classic Confinement**: Special handling for applications requiring `--classic` flag
- **Interactive Model**: Respects Snap's interactive permission prompting design
- **Dry-run Support**: Preview changes without modifying system state

**Snap Features:**

- Package name validation with Snap naming convention enforcement
- Classic confinement support for desktop applications
- Individual package installation (Snap limitation)
- State checking to prevent duplicate installations
- Integration with existing three-tier flag system

**Advanced Operations:**

- `InstallPackages()`: Install packages with flag resolution and individual handling
- `UninstallPackage()`: Remove packages with custom flags
- `ListInstalledPackages()`: Enumerate installed packages
- `RefreshPackages()`: Update all installed packages
- `InfoPackage()`: Get detailed package information
- `FindPackage()`: Search for available packages
- `ValidatePackageNames()`: Pre-validate package names

#### Package and File Removal System

**Core Components:**

- **StateManager (`internal/pkg/state.go`)** - Central orchestrator for package and file state tracking
- **Automatic removal**: Removes packages and files when they are removed from configuration
- **State persistence**: Tracks installed packages and deployed files in `~/.config/configr/state.json`
- **Cross-manager support**: Works with APT, Flatpak, and Snap packages
- **File type support**: Handles both symlinked and copied files
- **Safety checks**: Only removes packages that are actually installed and files that are safe to remove
- **Configurable**: Can be disabled with `--remove-packages=false` flag

**Key Implementation Details:**

- **State tracking**: JSON file tracks all packages and files managed by configr
- **Differential analysis**: Compares current state with new configuration to determine removals
- **Manager integration**: Uses existing `RemovePackages()` methods on each package manager
- **File removal integration**: Uses `RemoveFiles()` method on FileManager with safety checks
- **Dry-run support**: Preview removals without making changes
- **Error handling**: Graceful degradation if state tracking fails

**Package and File Removal Features:**

- State file format with version tracking and timestamps
- Removal detection by comparing previous and current package/file lists
- Batch removal operations for APT packages
- Individual removal for Flatpak and Snap packages (manager limitations)
- File removal with symlink/copy detection and safety checks
- Integration with existing three-tier flag system for removal operations

**Advanced Operations:**

- `LoadState()`: Read current package and file state from disk
- `SaveState()`: Persist package and file state with timestamps
- `UpdateState()`: Update state after successful configuration application
- `GetPackagesToRemove()`: Calculate packages that need removal
- `GetFilesToRemove()`: Calculate files that need removal
- State file location: `~/.config/configr/state.json`

**Safety and Error Handling:**

- Only removes packages that are actually installed on the system
- Only removes files that match expected type (symlink vs copy)
- Skips removal of files that appear to be modified by users
- Performs safety checks on symlinks to prevent system file removal
- Graceful handling of missing or corrupted state files
- Continues with installation/deployment even if removal tracking fails
- Configurable via command-line flag for safety

#### Interactive Features System

**Core Components:**

- **InteractiveManager (`internal/pkg/interactive.go`)** - Central orchestrator for all interactive features
- **Conflict resolution prompts**: User-friendly y/n prompts for file conflicts
- **File diff preview**: Shows differences between source and destination files before replacement
- **Advanced permission handling**: Interactive prompts for file permissions and ownership
- **Terminal detection**: Automatically detects if running in interactive mode

**Key Implementation Details:**

- **Conflict Resolution**: Prompts users when files already exist at destination paths
- **Diff Preview**: Uses system `diff` command with fallback to built-in diff implementation
- **Permission Prompts**: Interactive validation and modification of file permissions (octal format)
- **Ownership Prompts**: Interactive confirmation and modification of file ownership
- **Terminal Safety**: Only activates in interactive terminal environments

**Interactive Features:**

- File conflict resolution with options: overwrite, backup, skip, view diff, quit
- Real-time file diff preview using unified diff format
- Permission validation with octal format checking (644, 755, etc.)
- Ownership prompting with username/UID and groupname/GID support
- Preview summaries showing all changes before application
- Graceful degradation when not in interactive mode

**Advanced Operations:**

- `PromptForConflictResolution()`: Handle file conflicts with user input
- `ShowFileDiff()`: Display differences between files using system or built-in diff
- `PromptYesNo()`: Generic yes/no prompting with default value support
- `PromptForPermissions()`: Interactive permission setting with validation
- `PromptForOwnership()`: Interactive ownership configuration
- `ShowPreviewSummary()`: Comprehensive preview of all planned changes
- `IsInteractiveMode()`: Terminal detection for interactive capabilities

**Configuration Integration:**

- **Per-file control**: `interactive: true` enables prompts for specific files
- **Global flag**: `--interactive` enables interactive mode for all file operations
- **Granular prompts**: `prompt_permissions` and `prompt_ownership` for specific interactions
- **Backward compatibility**: All interactive features are opt-in and don't affect existing configurations

**Safety and Error Handling:**

- **Terminal detection**: Only activates in proper terminal environments
- **Input validation**: Validates octal permissions and ownership formats
- **Graceful fallback**: Continues with standard behavior if interactive features fail
- **User control**: Multiple exit strategies (skip, quit) to prevent unwanted changes
- **Error isolation**: Interactive failures don't prevent core functionality

#### State Caching & Optimization System

**Core Components:**

- **CacheManager (`internal/pkg/cache.go`)** - Central orchestrator for all caching operations
- **Configuration caching**: Converts YAML to faster binary format with modification tracking
- **System state caching**: Caches package installation status and file deployment state
- **Change detection**: Automatic cache invalidation when source files change
- **Performance optimization**: Dramatically reduces repeated run times

**Key Implementation Details:**

- **Configuration Cache**: Parsed configurations stored as JSON with modification time tracking
- **System State Cache**: Package installation status cached for 10 minutes, system state for 1 hour
- **File State Cache**: Tracks deployed files with checksums and modification times
- **Cache invalidation**: Automatic invalidation when config files or system state changes
- **Smart loading**: Falls back to standard loading if cache is invalid or missing

**Caching Features:**

- Multi-level caching (config, packages, files, system state)
- Automatic cache expiration and validation
- File modification time tracking for cache invalidation
- Configurable cache TTL values
- Cross-run optimization for repeated apply operations
- Cache statistics and management commands

**Advanced Operations:**

- `LoadCachedConfig()`: Load configuration from cache with validation
- `SaveCachedConfig()`: Store parsed configuration with metadata
- `LoadSystemStateCache()`: Load cached system state information
- `SaveSystemStateCache()`: Store package and file state data
- `ClearCache()`: Remove all cached data
- `GetCacheStats()`: Retrieve cache usage statistics
- Cache location: `~/.cache/configr/`

**Performance Benefits:**

- **Configuration loading**: 5-10x faster for large configs with includes
- **Package checking**: Skip installation status queries for recently cached packages
- **File deployment**: Avoid re-checking file states that haven't changed
- **System queries**: Reduce calls to apt/flatpak/snap for known package states
- **Overall speedup**: 2-5x faster repeated runs depending on configuration size

**Cache Management:**

- **Enable/Disable**: `--optimize=true/false` flag (enabled by default)
- **Cache statistics**: `configr cache stats` shows usage and performance data
- **Cache clearing**: `configr cache clear` removes all cached data
- **Cache information**: `configr cache info` shows system and configuration details

**Safety and Error Handling:**

- **Graceful degradation**: Falls back to standard mode if caching fails
- **Cache validation**: Ensures cached data is current and valid
- **Modification tracking**: Detects file changes and invalidates stale cache
- **Error isolation**: Cache failures don't affect core functionality
- **Atomic operations**: Cache updates are atomic to prevent corruption

### 🚧 In Development Features

*Reserved for future implementation status updates*

### 📋 Planned Features

#### Advanced Package Management

- **Package search and discovery**: Enhanced package finding capabilities
- **Package version management**: Pin and update specific package versions

#### Enhanced File Management

- **Backup restoration**: Restore backed-up files when configurations are removed
- **Interactive conflict resolution**: Yes/no prompts for file conflicts
- **Advanced permission handling**: More sophisticated ownership management
- **File modification detection**: Better algorithms for detecting user-modified files

#### System Integration

- **Init command**: Ensure required tools (apt, flatpak, snapd) are installed
- **Tool availability checking**: Verify and offer to install missing dependencies
- **System state management**: Track what configr manages vs. external changes

#### Configuration System Enhancements

- **Include system expansion**: More sophisticated file inclusion patterns
- **Configuration splitting**: Advanced strategies for modular configurations
- **Validation improvements**: More comprehensive safety and compatibility checks

---

## Implementation Details (Reference)

### Code Organization

```
configr/
├── cmd/configr/           # CLI command implementations
│   ├── root.go           # Root command and global flags
│   ├── validate.go       # Configuration validation command
│   └── apply.go          # Configuration application command
├── internal/
│   ├── config/           # Configuration management
│   │   ├── types.go      # YAML schema definitions
│   │   ├── validation.go # Validation logic and error reporting
│   │   ├── loader.go     # Configuration loading with includes
│   │   └── defaults.go   # Package manager default flags
│   └── pkg/              # Feature implementations
│       ├── files.go      # File management system
│       └── apt.go        # APT package management
├── examples/             # Configuration examples
└── integration_test.go   # End-to-end integration tests
```

### Design Patterns Used

- **Manager Pattern**: FileManager, AptManager for feature encapsulation
- **Three-Tier Resolution**: Hierarchical flag resolution system
- **Validation Pipeline**: Comprehensive validation with actionable error reporting
- **Command Pattern**: Cobra-based CLI with clear separation of concerns
- **Strategy Pattern**: Symlink vs. copy deployment strategies

### Integration Points

- **Viper Integration**: Configuration loading with multiple file format support
- **Cobra Integration**: CLI command structure with professional help pages
- **System Command Integration**: Shelling out to apt, dpkg with proper error handling
- **Validation Integration**: Unified validation system across all configuration types

### Testing Strategies

- **Unit Tests**: Individual component testing (AptManager, FileManager, validation)
- **Integration Tests**: End-to-end testing with actual binary execution
- **Validation Tests**: Comprehensive schema and error condition testing
- **Cross-platform Considerations**: Path handling and command availability testing

### Advanced Include System

**Core Components:**

- **AdvancedLoader (`internal/config/advanced_loader.go`)** - Central orchestrator for advanced include processing
- **Glob pattern support**: Include multiple files using wildcards (*.yaml, packages/*.yaml)
- **Conditional includes**: System-aware configuration loading based on OS, hostname, environment
- **Enhanced validation**: Comprehensive validation with circular dependency prevention
- **Backward compatibility**: Full support for existing simple include syntax

**Key Implementation Details:**

- **Glob Pattern Resolution**: Uses `filepath.Glob()` for reliable pattern matching with YAML file filtering
- **System Detection**: Automatic detection of OS, hostname, and environment variables
- **Condition Evaluation**: Multiple condition types with configurable operators
- **Optional Includes**: Graceful handling of missing optional configuration files
- **Path Safety**: Validation against unsafe path traversal patterns

**Advanced Include Features:**

```yaml
# Simple includes (backward compatibility)
includes:
  - "common/base.yaml"
  - "packages/"

# Advanced includes with conditions and patterns
advanced_includes:
  # Glob patterns
  - glob: "packages/*.yaml"
    description: "All package configurations"
    optional: false
  
  # Conditional includes
  - path: "os-specific/linux.yaml"
    optional: true
    conditions:
      - type: "os"
        value: "linux"
        operator: "equals"
  
  # Environment-based includes
  - path: "environments/development.yaml"
    optional: true
    conditions:
      - type: "env"
        value: "NODE_ENV=development"
  
  # Multiple conditions (all must be true)
  - glob: "hosts/*.yaml"
    optional: true
    conditions:
      - type: "hostname"
        value: "workstation"
        operator: "contains"
      - type: "file_exists"
        value: "/etc/workstation-config"
```

**Condition Types:**

- **`os`**: Operating system detection (linux, darwin, windows)
- **`hostname`**: System hostname matching with operators
- **`env`**: Environment variable existence and value checking
- **`file_exists`**: File system existence checks
- **`dir_exists`**: Directory existence validation

**Operators:**

- **`equals`**: Exact string matching (default)
- **`contains`**: Substring matching
- **`matches`**: Regular expression matching
- **`not_equals`**: Negated exact matching
- **`not_contains`**: Negated substring matching

**Advanced Operations:**

- `LoadConfigurationAdvanced()`: Main loading entry point with full feature support
- `resolveGlobPattern()`: Pattern matching with YAML file filtering
- `evaluateConditions()`: Multi-condition evaluation with AND logic
- `ValidateIncludeSpec()`: Comprehensive validation of include specifications
- `GetSystemInfo()`: System information for debugging conditional includes

**Path Resolution Rules:**

1. **Simple includes**: `packages.yaml` → loads `packages.yaml`
2. **Directory includes**: `packages/` → loads `packages/default.yaml`
3. **Glob patterns**: `packages/*.yaml` → loads all matching YAML files
4. **Conditional paths**: Evaluated only when conditions are met
5. **Optional includes**: Skip gracefully when files don't exist

**Validation and Safety:**

- **Circular dependency detection**: Prevents infinite include loops
- **Path traversal protection**: Validates against `../` patterns
- **Glob syntax validation**: Tests pattern validity before evaluation
- **Condition validation**: Ensures valid condition types and operators
- **Optional include handling**: Graceful degradation for missing files

**Integration:**

- **Optimized loading**: Seamless integration with existing cache system
- **Validation system**: Enhanced validation with position-aware error reporting
- **Backward compatibility**: Existing configurations continue to work unchanged
- **Error reporting**: Rust-style error messages with actionable suggestions

**Performance Considerations:**

- **Lazy evaluation**: Conditions evaluated only when needed
- **Cached system info**: OS and hostname cached per loader instance
- **Efficient glob matching**: Uses Go's optimized filepath.Glob
- **Path deduplication**: Prevents loading same file multiple times

This advanced include system provides powerful configuration organization capabilities while maintaining configr's commitment to exceptional UX and system administrator-friendly operation.
