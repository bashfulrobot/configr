# justfile for configr

# Get version from git tags, fallback to commit hash
version := `git describe --tags --always --dirty 2>/dev/null || echo "unknown"`

# Default recipe
default: build

# Build configr with version injection
build:
    go build -ldflags "-X github.com/bashfulrobot/configr/cmd/configr.Version={{version}}" -o configr .

# Development build (keeps "dev" version)
build-dev:
    go build -o configr .

# Install to system
install: build
    sudo mv configr /usr/local/bin/

# Run tests
test:
    go test ./...

# Clean build artifacts
clean:
    rm -f configr

# Show current version that would be used
show-version:
    @echo "Version: {{version}}"

# List available recipes
list:
    @just --list

# Show help
help:
    @echo "Available recipes:"
    @echo "  build      - Build configr with version injection (default)"
    @echo "  build-dev  - Build configr without version injection (keeps 'dev')"
    @echo "  install    - Build and install to /usr/local/bin/"
    @echo "  test       - Run tests"
    @echo "  clean      - Remove build artifacts"
    @echo "  show-version - Show version that would be injected"
    @echo "  list       - List all available recipes"
    @echo "  help       - Show this help"