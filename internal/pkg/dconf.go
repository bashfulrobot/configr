package pkg

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/log"
)

// DConfManager handles dconf configuration management operations
type DConfManager struct {
	logger *log.Logger
	dryRun bool
}

// NewDConfManager creates a new dconf manager
func NewDConfManager(logger *log.Logger, dryRun bool) *DConfManager {
	return &DConfManager{
		logger: logger,
		dryRun: dryRun,
	}
}

// ApplySettings applies all dconf settings
func (dm *DConfManager) ApplySettings(dconfConfig config.DConfConfig) error {
	if len(dconfConfig.Settings) == 0 {
		dm.logger.Debug("No dconf settings to apply")
		return nil
	}

	dm.logger.Info("Applying dconf settings...", "count", len(dconfConfig.Settings))

	// Check if dconf is available (skip in dry-run for testing)
	if !dm.dryRun {
		if err := dm.checkDConfAvailable(); err != nil {
			return fmt.Errorf("dconf not available: %w", err)
		}
	}

	for path, value := range dconfConfig.Settings {
		if err := dm.setSetting(path, value); err != nil {
			return fmt.Errorf("failed to set dconf setting '%s': %w", path, err)
		}
	}

	dm.logger.Info("✓ DConf settings applied successfully")
	return nil
}

// setSetting sets a single dconf setting
func (dm *DConfManager) setSetting(path, value string) error {
	args := []string{"dconf", "write", path, value}

	dm.logger.Info("Setting dconf value", "path", path, "value", value)

	if dm.dryRun {
		dm.logger.Info("  [DRY RUN] Would run:", "command", strings.Join(args, " "))
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		dm.logger.Error("Failed to set dconf value", "path", path, "error", err, "output", string(output))
		return fmt.Errorf("dconf write failed: %w", err)
	}

	dm.logger.Debug("DConf value set successfully", "path", path, "output", string(output))
	return nil
}

// GetSetting retrieves a single dconf setting value
func (dm *DConfManager) GetSetting(path string) (string, error) {
	if dm.dryRun {
		dm.logger.Info("  [DRY RUN] Would run:", "command", fmt.Sprintf("dconf read %s", path))
		return "", nil
	}

	args := []string{"dconf", "read", path}
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		dm.logger.Error("Failed to read dconf value", "path", path, "error", err, "output", string(output))
		return "", fmt.Errorf("dconf read failed: %w", err)
	}

	value := strings.TrimSpace(string(output))
	dm.logger.Debug("DConf value retrieved", "path", path, "value", value)
	return value, nil
}

// ResetSetting resets a dconf setting to its default value
func (dm *DConfManager) ResetSetting(path string) error {
	args := []string{"dconf", "reset", path}

	dm.logger.Info("Resetting dconf value", "path", path)

	if dm.dryRun {
		dm.logger.Info("  [DRY RUN] Would run:", "command", strings.Join(args, " "))
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		dm.logger.Error("Failed to reset dconf value", "path", path, "error", err, "output", string(output))
		return fmt.Errorf("dconf reset failed: %w", err)
	}

	dm.logger.Debug("DConf value reset successfully", "path", path, "output", string(output))
	return nil
}

// ListSettings lists all dconf settings under a given path
func (dm *DConfManager) ListSettings(path string) ([]string, error) {
	if dm.dryRun {
		dm.logger.Info("  [DRY RUN] Would run:", "command", fmt.Sprintf("dconf list %s", path))
		return []string{}, nil
	}

	args := []string{"dconf", "list", path}
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		dm.logger.Error("Failed to list dconf settings", "path", path, "error", err, "output", string(output))
		return nil, fmt.Errorf("dconf list failed: %w", err)
	}

	// Split output into lines and filter out empty lines
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var settings []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			settings = append(settings, line)
		}
	}

	dm.logger.Debug("DConf settings listed", "path", path, "count", len(settings))
	return settings, nil
}

// DumpSettings dumps all dconf settings under a given path
func (dm *DConfManager) DumpSettings(path string) (map[string]string, error) {
	if dm.dryRun {
		dm.logger.Info("  [DRY RUN] Would run:", "command", fmt.Sprintf("dconf dump %s", path))
		return map[string]string{}, nil
	}

	args := []string{"dconf", "dump", path}
	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		dm.logger.Error("Failed to dump dconf settings", "path", path, "error", err, "output", string(output))
		return nil, fmt.Errorf("dconf dump failed: %w", err)
	}

	// Parse the ini-like output format
	settings := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	var currentSection string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle section headers like [org/gnome/desktop/interface]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.Trim(line, "[]")
			// Convert section format to dconf path format
			if !strings.HasPrefix(currentSection, "/") {
				currentSection = "/" + strings.ReplaceAll(currentSection, "/", "/")
			}
			continue
		}

		// Handle key=value pairs
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				fullPath := currentSection + "/" + key
				settings[fullPath] = value
			}
		}
	}

	dm.logger.Debug("DConf settings dumped", "path", path, "count", len(settings))
	return settings, nil
}

// checkDConfAvailable checks if dconf command is available
func (dm *DConfManager) checkDConfAvailable() error {
	if _, err := exec.LookPath("dconf"); err != nil {
		return fmt.Errorf("dconf command not found - install dconf package")
	}
	return nil
}

// ValidateSettings validates dconf settings before applying them
func (dm *DConfManager) ValidateSettings(dconfConfig config.DConfConfig) error {
	dm.logger.Debug("Validating dconf settings", "count", len(dconfConfig.Settings))

	for path, value := range dconfConfig.Settings {
		// Validate path format
		if !strings.HasPrefix(path, "/") {
			return fmt.Errorf("dconf path '%s' must start with '/'", path)
		}

		if strings.Contains(path, "//") {
			return fmt.Errorf("dconf path '%s' contains double slashes", path)
		}

		// Validate value format (basic checks)
		if value == "" {
			dm.logger.Warn("Empty dconf value", "path", path, "value", value)
		}

		// Check if value looks like it should be quoted
		if !strings.HasPrefix(value, "'") && !strings.HasPrefix(value, "\"") && 
		   !strings.HasPrefix(value, "[") && !isNumericValue(value) && !isBooleanValue(value) {
			dm.logger.Warn("DConf value may need quotes", "path", path, "value", value, 
				"suggestion", fmt.Sprintf("'%s'", value))
		}
	}

	return nil
}

// isNumericValue checks if a value looks like a number
func isNumericValue(value string) bool {
	if len(value) == 0 {
		return false
	}
	
	dotCount := 0
	eCount := 0
	
	for i, char := range value {
		switch {
		case char >= '0' && char <= '9':
			// Numbers are always valid
			continue
		case char == '.':
			dotCount++
			if dotCount > 1 {
				return false // Multiple decimal points
			}
		case char == '-' || char == '+':
			// Signs only valid at start or after 'e'/'E'
			if i != 0 && (value[i-1] != 'e' && value[i-1] != 'E') {
				return false
			}
		case char == 'e' || char == 'E':
			eCount++
			if eCount > 1 || i == 0 || i == len(value)-1 {
				return false // Multiple 'e' or at start/end
			}
		default:
			return false // Invalid character
		}
	}
	
	return true
}

// isBooleanValue checks if a value looks like a boolean
func isBooleanValue(value string) bool {
	lower := strings.ToLower(value)
	return lower == "true" || lower == "false"
}