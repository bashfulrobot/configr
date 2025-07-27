package pkg

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bashfulrobot/configr/internal/config"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
)

// UXManager handles user experience enhancements like progress bars and spinners
type UXManager struct {
	logger *log.Logger
	dryRun bool
	
	// Styles
	progressStyle   lipgloss.Style
	spinnerStyle    lipgloss.Style
	successStyle    lipgloss.Style
	errorStyle      lipgloss.Style
	warningStyle    lipgloss.Style
	noteStyle       lipgloss.Style
	lineNumberStyle lipgloss.Style
}

// NewUXManager creates a new UX manager
func NewUXManager(logger *log.Logger, dryRun bool) *UXManager {
	return &UXManager{
		logger: logger,
		dryRun: dryRun,
		
		// Initialize styles
		progressStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		spinnerStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("69")),
		successStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true),
		errorStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		warningStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
		noteStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true),
		lineNumberStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("243")),
	}
}

// ProgressModel represents a progress bar with a title and current step
type ProgressModel struct {
	title    string
	progress progress.Model
	current  int
	total    int
	done     bool
	err      error
}

func (m ProgressModel) Init() tea.Cmd {
	return nil
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case ProgressUpdateMsg:
		m.current = msg.Current
		m.total = msg.Total
		if m.current >= m.total {
			m.done = true
			return m, tea.Quit
		}
		return m, m.progress.SetPercent(float64(m.current) / float64(m.total))
	case ProgressErrorMsg:
		m.err = msg.Error
		return m, tea.Quit
	}
	
	var cmd tea.Cmd
	progressModel, progressCmd := m.progress.Update(msg)
	m.progress = progressModel.(progress.Model)
	return m, tea.Batch(cmd, progressCmd)
}

func (m ProgressModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("✗ %s: %v\n", m.title, m.err)
	}
	
	if m.done {
		return fmt.Sprintf("✓ %s completed\n", m.title)
	}
	
	return fmt.Sprintf("%s (%d/%d)\n%s\n", 
		m.title, m.current, m.total, 
		m.progress.View(),
	)
}

// SpinnerModel represents a spinner with a message
type SpinnerModel struct {
	spinner  spinner.Model
	message  string
	done     bool
	success  bool
	err      error
}

func (m SpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m SpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case SpinnerDoneMsg:
		m.done = true
		m.success = msg.Success
		m.err = msg.Error
		return m, tea.Quit
	case SpinnerUpdateMsg:
		m.message = msg.Message
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m SpinnerModel) View() string {
	if m.done {
		if m.err != nil {
			return fmt.Sprintf("✗ %s: %v\n", m.message, m.err)
		}
		if m.success {
			return fmt.Sprintf("✓ %s\n", m.message)
		}
		return fmt.Sprintf("- %s\n", m.message)
	}
	
	return fmt.Sprintf("%s %s", m.spinner.View(), m.message)
}

// Progress bar messages
type ProgressUpdateMsg struct {
	Current int
	Total   int
}

type ProgressErrorMsg struct {
	Error error
}

// Spinner messages
type SpinnerDoneMsg struct {
	Success bool
	Error   error
}

type SpinnerUpdateMsg struct {
	Message string
}

// ShowProgressBar displays a progress bar for multi-step operations
func (ux *UXManager) ShowProgressBar(title string, steps int) (*tea.Program, chan ProgressUpdateMsg, chan error) {
	if !ux.IsInteractiveTerminal() {
		// Fallback for non-interactive terminals
		ux.logger.Info(title, "steps", steps)
		return nil, make(chan ProgressUpdateMsg, 10), make(chan error, 1)
	}

	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 60
	prog.ShowPercentage = true
	
	model := ProgressModel{
		title:    title,
		progress: prog,
		current:  0,
		total:    steps,
	}
	
	p := tea.NewProgram(model, tea.WithOutput(os.Stderr))
	updateChan := make(chan ProgressUpdateMsg, 10)
	errorChan := make(chan error, 1)
	
	// Start the program in a goroutine
	go func() {
		if _, err := p.Run(); err != nil {
			ux.logger.Error("Progress bar error", "error", err)
		}
	}()
	
	// Handle updates
	go func() {
		for {
			select {
			case update := <-updateChan:
				p.Send(update)
				if update.Current >= update.Total {
					return
				}
			case err := <-errorChan:
				p.Send(ProgressErrorMsg{Error: err})
				return
			}
		}
	}()
	
	return p, updateChan, errorChan
}

// ShowPackageProgress displays a specialized progress bar for package operations
func (ux *UXManager) ShowPackageProgress(manager string, packages []string) (*tea.Program, chan ProgressUpdateMsg, chan error) {
	title := fmt.Sprintf("Installing %s packages", manager)
	if len(packages) > 0 {
		title = fmt.Sprintf("Installing %d %s packages", len(packages), manager)
	}
	return ux.ShowProgressBar(title, len(packages))
}

// ShowFileProgress displays a specialized progress bar for file operations
func (ux *UXManager) ShowFileProgress(operation string, files map[string]interface{}) (*tea.Program, chan ProgressUpdateMsg, chan error) {
	title := fmt.Sprintf("%s files", operation)
	if len(files) > 0 {
		title = fmt.Sprintf("%s %d files", operation, len(files))
	}
	return ux.ShowProgressBar(title, len(files))
}

// ShowSpinner displays a spinner for single operations
func (ux *UXManager) ShowSpinner(message string) (*tea.Program, chan string, chan SpinnerDoneMsg) {
	if !ux.IsInteractiveTerminal() {
		// Fallback for non-interactive terminals
		ux.logger.Info(message)
		return nil, make(chan string, 10), make(chan SpinnerDoneMsg, 1)
	}

	s := spinner.New()
	s.Spinner = spinner.Line
	s.Style = ux.spinnerStyle
	
	model := SpinnerModel{
		spinner: s,
		message: message,
	}
	
	p := tea.NewProgram(model, tea.WithOutput(os.Stderr))
	updateChan := make(chan string, 10)
	doneChan := make(chan SpinnerDoneMsg, 1)
	
	// Start the program in a goroutine
	go func() {
		if _, err := p.Run(); err != nil {
			ux.logger.Error("Spinner error", "error", err)
		}
	}()
	
	// Handle updates
	go func() {
		for {
			select {
			case msg := <-updateChan:
				p.Send(SpinnerUpdateMsg{Message: msg})
			case done := <-doneChan:
				p.Send(done)
				return
			}
		}
	}()
	
	return p, updateChan, doneChan
}

// ShowValidationSpinner displays a spinner specifically for validation operations
func (ux *UXManager) ShowValidationSpinner() (*tea.Program, chan SpinnerDoneMsg) {
	_, _, doneChan := ux.ShowSpinner("Validating configuration...")
	p, _, doneChan2 := ux.ShowSpinner("Validating configuration...")
	
	// Merge channels for simplified interface
	go func() {
		select {
		case msg := <-doneChan2:
			doneChan <- msg
		}
	}()
	
	return p, doneChan
}

// ShowRepositorySpinner displays a spinner for repository operations
func (ux *UXManager) ShowRepositorySpinner(operation string) (*tea.Program, chan string, chan SpinnerDoneMsg) {
	message := fmt.Sprintf("%s repositories...", operation)
	return ux.ShowSpinner(message)
}

// ShowConfigLoadSpinner displays a spinner for configuration loading
func (ux *UXManager) ShowConfigLoadSpinner(optimized bool) (*tea.Program, chan SpinnerDoneMsg) {
	message := "Loading configuration..."
	if optimized {
		message = "Loading configuration (optimized)..."
	}
	_, _, doneChan := ux.ShowSpinner(message)
	p, _, doneChan2 := ux.ShowSpinner(message)
	
	// Merge channels for simplified interface
	go func() {
		select {
		case msg := <-doneChan2:
			doneChan <- msg
		}
	}()
	
	return p, doneChan
}

// ShowConfigPreview displays a comprehensive preview of the configuration changes
func (ux *UXManager) ShowConfigPreview(cfg *config.Config) string {
	var preview strings.Builder
	
	// Header with enhanced styling
	headerTitle := ux.successStyle.Render("📋 Configuration Preview")
	preview.WriteString(headerTitle + "\n")
	preview.WriteString(ux.lineNumberStyle.Render(strings.Repeat("═", 60)) + "\n\n")
	
	// Version and metadata
	if cfg.Version != "" {
		preview.WriteString(ux.noteStyle.Render("📌 Version: ") + cfg.Version + "\n")
	}
	
	// Includes summary
	if len(cfg.Includes) > 0 {
		preview.WriteString(ux.noteStyle.Render("📁 Includes: ") + fmt.Sprintf("%d files\n", len(cfg.Includes)))
		for _, include := range cfg.Includes {
			if include.Path != "" {
				preview.WriteString(fmt.Sprintf("   ↳ %s", include.Path))
				if include.Optional {
					preview.WriteString(ux.warningStyle.Render(" (optional)"))
				}
				preview.WriteString("\n")
			}
		}
	}
	preview.WriteString("\n")
	
	// Repositories with enhanced details
	totalRepos := len(cfg.Repositories.Apt) + len(cfg.Repositories.Flatpak)
	if totalRepos > 0 {
		preview.WriteString(ux.warningStyle.Render("📦 Repositories ") + fmt.Sprintf("(%d total)", totalRepos) + "\n")
		
		if len(cfg.Repositories.Apt) > 0 {
			preview.WriteString(ux.noteStyle.Render("  APT:") + "\n")
			for _, repo := range cfg.Repositories.Apt {
				if repo.PPA != "" {
					preview.WriteString(fmt.Sprintf("    • PPA: %s", repo.PPA))
					if repo.Key != "" {
						preview.WriteString(ux.lineNumberStyle.Render(" (with GPG key)"))
					}
					preview.WriteString("\n")
				} else if repo.URI != "" {
					preview.WriteString(fmt.Sprintf("    • Custom: %s", repo.URI))
					if repo.Key != "" {
						preview.WriteString(ux.lineNumberStyle.Render(" (with GPG key)"))
					}
					preview.WriteString("\n")
				}
			}
		}
		
		if len(cfg.Repositories.Flatpak) > 0 {
			preview.WriteString(ux.noteStyle.Render("  Flatpak:") + "\n")
			for _, repo := range cfg.Repositories.Flatpak {
				preview.WriteString(fmt.Sprintf("    • %s: %s", repo.Name, repo.URL))
				if repo.User {
					preview.WriteString(ux.lineNumberStyle.Render(" (user scope)"))
				} else {
					preview.WriteString(ux.lineNumberStyle.Render(" (system scope)"))
				}
				preview.WriteString("\n")
			}
		}
		preview.WriteString("\n")
	}
	
	// Packages with enhanced breakdown
	totalPackages := len(cfg.Packages.Apt) + len(cfg.Packages.Flatpak) + len(cfg.Packages.Snap)
	if totalPackages > 0 {
		preview.WriteString(ux.warningStyle.Render("📱 Packages ") + fmt.Sprintf("(%d total)", totalPackages) + "\n")
		
		if len(cfg.Packages.Apt) > 0 {
			preview.WriteString(fmt.Sprintf("  • APT: %s packages", ux.successStyle.Render(fmt.Sprintf("%d", len(cfg.Packages.Apt)))))
			debFiles := 0
			for _, pkg := range cfg.Packages.Apt {
				if strings.HasSuffix(pkg.Name, ".deb") {
					debFiles++
				}
			}
			if debFiles > 0 {
				preview.WriteString(fmt.Sprintf(" (%d local .deb files)", debFiles))
			}
			preview.WriteString("\n")
		}
		
		if len(cfg.Packages.Flatpak) > 0 {
			preview.WriteString(fmt.Sprintf("  • Flatpak: %s packages\n", ux.successStyle.Render(fmt.Sprintf("%d", len(cfg.Packages.Flatpak)))))
		}
		
		if len(cfg.Packages.Snap) > 0 {
			preview.WriteString(fmt.Sprintf("  • Snap: %s packages", ux.successStyle.Render(fmt.Sprintf("%d", len(cfg.Packages.Snap)))))
			classicSnaps := 0
			for _, pkg := range cfg.Packages.Snap {
				for _, flag := range pkg.Flags {
					if flag == "--classic" {
						classicSnaps++
						break
					}
				}
			}
			if classicSnaps > 0 {
				preview.WriteString(fmt.Sprintf(" (%d classic mode)", classicSnaps))
			}
			preview.WriteString("\n")
		}
		preview.WriteString("\n")
	}
	
	// Files with enhanced details
	if len(cfg.Files) > 0 {
		preview.WriteString(ux.warningStyle.Render("📄 Files ") + fmt.Sprintf("(%d total)", len(cfg.Files)) + "\n")
		
		symlinkCount := 0
		copyCount := 0
		interactiveCount := 0
		
		for name, file := range cfg.Files {
			mode := ux.noteStyle.Render("symlink")
			if file.Copy {
				mode = ux.warningStyle.Render("copy")
				copyCount++
			} else {
				symlinkCount++
			}
			
			if file.Interactive {
				interactiveCount++
			}
			
			preview.WriteString(fmt.Sprintf("  • %s: %s → %s (%s)", 
				name, file.Source, file.Destination, mode))
			
			if file.Mode != "" || file.Owner != "" || file.Group != "" {
				permissions := []string{}
				if file.Mode != "" {
					permissions = append(permissions, fmt.Sprintf("mode:%s", file.Mode))
				}
				if file.Owner != "" {
					permissions = append(permissions, fmt.Sprintf("owner:%s", file.Owner))
				}
				if file.Group != "" {
					permissions = append(permissions, fmt.Sprintf("group:%s", file.Group))
				}
				preview.WriteString(ux.lineNumberStyle.Render(fmt.Sprintf(" [%s]", strings.Join(permissions, " "))))
			}
			
			if file.Interactive {
				preview.WriteString(ux.warningStyle.Render(" (interactive)"))
			}
			
			preview.WriteString("\n")
		}
		
		// Summary
		preview.WriteString(ux.lineNumberStyle.Render(fmt.Sprintf("    Summary: %d symlinks, %d copies", symlinkCount, copyCount)))
		if interactiveCount > 0 {
			preview.WriteString(ux.lineNumberStyle.Render(fmt.Sprintf(", %d interactive", interactiveCount)))
		}
		preview.WriteString("\n\n")
	}
	
	// DConf settings with enhanced details
	if len(cfg.DConf.Settings) > 0 {
		preview.WriteString(ux.warningStyle.Render("⚙️  DConf Settings ") + fmt.Sprintf("(%d settings)", len(cfg.DConf.Settings)) + "\n")
		
		// Show a few example settings
		count := 0
		for path, value := range cfg.DConf.Settings {
			if count < 3 {
				truncatedValue := value
				if len(value) > 40 {
					truncatedValue = value[:37] + "..."
				}
				preview.WriteString(fmt.Sprintf("  • %s = %s\n", path, truncatedValue))
				count++
			} else {
				remaining := len(cfg.DConf.Settings) - 3
				preview.WriteString(ux.lineNumberStyle.Render(fmt.Sprintf("  ... and %d more settings\n", remaining)))
				break
			}
		}
		preview.WriteString("\n")
	}
	
	// Package defaults summary
	if cfg.PackageDefaults != nil && len(cfg.PackageDefaults) > 0 {
		preview.WriteString(ux.noteStyle.Render("🔧 Package Defaults:") + "\n")
		for manager, flags := range cfg.PackageDefaults {
			preview.WriteString(fmt.Sprintf("  • %s: %s\n", manager, strings.Join(flags, " ")))
		}
		preview.WriteString("\n")
	}
	
	// Footer with status and next steps
	preview.WriteString(ux.lineNumberStyle.Render(strings.Repeat("─", 60)) + "\n")
	if ux.dryRun {
		preview.WriteString(ux.spinnerStyle.Render("🔍 DRY RUN MODE") + " - No changes will be made to your system\n")
	} else {
		preview.WriteString(ux.errorStyle.Render("🚀 READY TO APPLY") + " - Configuration will be applied to your system\n")
	}
	
	return preview.String()
}

// FormatValidationSummary creates a comprehensive visual summary of validation results
func (ux *UXManager) FormatValidationSummary(result *config.ValidationResult) string {
	if result == nil {
		return ""
	}

	var output strings.Builder
	
	// Header for validation results
	if len(result.Errors) > 0 || len(result.Warnings) > 0 {
		output.WriteString(ux.errorStyle.Render("🔍 Configuration Validation Report") + "\n")
		output.WriteString(ux.lineNumberStyle.Render(strings.Repeat("═", 60)) + "\n\n")
	}
	
	// Summary statistics
	totalIssues := len(result.Errors) + len(result.Warnings)
	if totalIssues > 0 {
		summaryParts := []string{}
		if len(result.Errors) > 0 {
			summaryParts = append(summaryParts, ux.errorStyle.Render(fmt.Sprintf("%d error(s)", len(result.Errors))))
		}
		if len(result.Warnings) > 0 {
			summaryParts = append(summaryParts, ux.warningStyle.Render(fmt.Sprintf("%d warning(s)", len(result.Warnings))))
		}
		
		output.WriteString(ux.noteStyle.Render("📊 Summary: ") + strings.Join(summaryParts, ", ") + " found\n\n")
	}

	// Use enhanced Rust-style formatting for detailed errors
	enhancedOutput := config.FormatValidationResultEnhanced(result)
	if enhancedOutput != "" {
		output.WriteString(enhancedOutput)
	}
	
	// Add quick fixes section
	quickFixes := config.FormatQuickFixEnhanced(result)
	if quickFixes != "" {
		output.WriteString(quickFixes)
	}
	
	// Footer with next steps
	if len(result.Errors) > 0 {
		output.WriteString("\n" + ux.lineNumberStyle.Render(strings.Repeat("─", 60)) + "\n")
		output.WriteString(ux.errorStyle.Render("❌ Validation failed") + " - Please fix the above errors before applying configuration\n")
	} else if len(result.Warnings) > 0 {
		output.WriteString("\n" + ux.lineNumberStyle.Render(strings.Repeat("─", 60)) + "\n")
		output.WriteString(ux.warningStyle.Render("⚠️  Warnings detected") + " - Configuration is valid but consider addressing these issues\n")
	}
	
	return output.String()
}

// FormatValidationSummaryCompact creates a compact validation summary for quick feedback
func (ux *UXManager) FormatValidationSummaryCompact(result *config.ValidationResult) string {
	if result == nil || (len(result.Errors) == 0 && len(result.Warnings) == 0) {
		return ux.successStyle.Render("✓ Configuration is valid")
	}
	
	var summary strings.Builder
	
	if len(result.Errors) > 0 {
		summary.WriteString(ux.errorStyle.Render(fmt.Sprintf("✗ %d error(s)", len(result.Errors))))
		if len(result.Warnings) > 0 {
			summary.WriteString(", ")
		}
	}
	
	if len(result.Warnings) > 0 {
		summary.WriteString(ux.warningStyle.Render(fmt.Sprintf("⚠ %d warning(s)", len(result.Warnings))))
	}
	
	return summary.String()
}

// IsInteractiveTerminal checks if we're running in an interactive terminal
func (ux *UXManager) IsInteractiveTerminal() bool {
	// Check if we're in a TTY
	if fileInfo, _ := os.Stdin.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		return false
	}
	
	// Check for CI environment variables
	ciVars := []string{"CI", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "TRAVIS"}
	for _, ciVar := range ciVars {
		if os.Getenv(ciVar) != "" {
			return false
		}
	}
	
	return true
}

// SimulateProgress simulates a progress bar for demonstration
func (ux *UXManager) SimulateProgress(title string, steps int, duration time.Duration) {
	if !ux.IsInteractiveTerminal() {
		ux.logger.Info(title)
		return
	}
	
	p, updateChan, _ := ux.ShowProgressBar(title, steps)
	defer p.Kill()
	
	stepDuration := duration / time.Duration(steps)
	
	go func() {
		for i := 0; i <= steps; i++ {
			select {
			case updateChan <- ProgressUpdateMsg{Current: i, Total: steps}:
			case <-time.After(time.Second):
				// Timeout protection
				return
			}
			
			if i < steps {
				time.Sleep(stepDuration)
			}
		}
	}()
	
	// Let the progress bar finish
	time.Sleep(duration + time.Millisecond*500)
}

// SimulateSpinner simulates a spinner for demonstration
func (ux *UXManager) SimulateSpinner(message string, duration time.Duration, success bool) {
	if !ux.IsInteractiveTerminal() {
		ux.logger.Info(message)
		return
	}
	
	p, _, doneChan := ux.ShowSpinner(message)
	defer p.Kill()
	
	go func() {
		time.Sleep(duration)
		doneChan <- SpinnerDoneMsg{Success: success}
	}()
	
	// Let the spinner finish
	time.Sleep(duration + time.Millisecond*500)
}