package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// EnhancedFormatter provides Rust-style error formatting with colors and enhanced presentation
type EnhancedFormatter struct {
	// Color styles
	errorStyle     lipgloss.Style
	warningStyle   lipgloss.Style
	noteStyle      lipgloss.Style
	helpStyle      lipgloss.Style
	lineNumberStyle lipgloss.Style
	highlightStyle lipgloss.Style
	codeStyle      lipgloss.Style
	titleStyle     lipgloss.Style
}

// NewEnhancedFormatter creates a new enhanced formatter with color support
func NewEnhancedFormatter() *EnhancedFormatter {
	return &EnhancedFormatter{
		errorStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		warningStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
		noteStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true),
		helpStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true),
		lineNumberStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("243")),
		highlightStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Underline(true),
		codeStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("235")).Padding(0, 1),
		titleStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true),
	}
}

// FormatValidationResultEnhanced provides enhanced Rust-style error formatting
func (ef *EnhancedFormatter) FormatValidationResultEnhanced(result *ValidationResult) string {
	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		return ef.helpStyle.Render("✅ Configuration validation passed") + "\n"
	}

	var output strings.Builder

	// Format errors with enhanced styling
	for i, err := range result.Errors {
		if i > 0 {
			output.WriteString(ef.lineNumberStyle.Render("─────────────────────────────────────────────────────────") + "\n\n")
		}
		output.WriteString(ef.formatErrorEnhanced(err))
	}

	// Add separator between errors and warnings
	if len(result.Errors) > 0 && len(result.Warnings) > 0 {
		output.WriteString(ef.lineNumberStyle.Render("─────────────────────────────────────────────────────────") + "\n\n")
	}

	// Format warnings with enhanced styling
	for i, warning := range result.Warnings {
		if i > 0 {
			output.WriteString(ef.lineNumberStyle.Render("─────────────────────────────────────────────────────────") + "\n\n")
		}
		output.WriteString(ef.formatWarningEnhanced(warning))
	}

	// Enhanced summary with better visual hierarchy
	if len(result.Errors) > 0 || len(result.Warnings) > 0 {
		output.WriteString(ef.lineNumberStyle.Render("═════════════════════════════════════════════════════════") + "\n")
		
		if len(result.Errors) > 0 {
			summary := fmt.Sprintf("🚨 Validation failed: %d error(s) found", len(result.Errors))
			if len(result.Warnings) > 0 {
				summary += fmt.Sprintf(" (+ %d warning(s))", len(result.Warnings))
			}
			output.WriteString(ef.errorStyle.Render(summary) + "\n")
		} else if len(result.Warnings) > 0 {
			summary := fmt.Sprintf("⚠️  Validation completed with %d warning(s)", len(result.Warnings))
			output.WriteString(ef.warningStyle.Render(summary) + "\n")
		}
		
		output.WriteString(ef.lineNumberStyle.Render("═════════════════════════════════════════════════════════") + "\n")
	}

	return output.String()
}

func (ef *EnhancedFormatter) formatErrorEnhanced(err ValidationError) string {
	var output strings.Builder

	// Error header with enhanced styling and emoji
	output.WriteString(ef.errorStyle.Render("❌ error") + ": " + ef.titleStyle.Render(err.Title) + "\n")

	// Location with file context
	if err.File != "" && err.Line > 0 {
		location := fmt.Sprintf("%s:%d:%d", err.File, err.Line, err.Column)
		output.WriteString("  " + ef.noteStyle.Render("┌─>") + " " + location + "\n")
		
		// Show actual file content if available
		if content := ef.getFileContext(err.File, err.Line, err.Column); content != "" {
			output.WriteString(content)
		}
	} else if err.File != "" {
		output.WriteString("  " + ef.noteStyle.Render("┌─>") + " " + err.File + "\n")
	}

	// Field path and value with enhanced highlighting
	if err.Field != "" {
		output.WriteString("  │\n")
		if err.Value != "" {
			// Highlight problematic value with better context
			highlighted := ef.highlightStyle.Render(err.Value)
			output.WriteString(fmt.Sprintf("  │ %s: %s\n", err.Field, highlighted))
			
			// Show underline highlighting
			fieldLen := len(err.Field) + 2 // ": " length
			valueStart := fieldLen
			underline := strings.Repeat(" ", valueStart) + ef.highlightStyle.Render(strings.Repeat("^", len(err.Value)))
			output.WriteString("  │ " + underline + " " + ef.errorStyle.Render(err.Message) + "\n")
		} else {
			output.WriteString(fmt.Sprintf("  │ %s\n", err.Field))
			if err.Message != "" {
				output.WriteString("  │ " + strings.Repeat(" ", len(err.Field)) + " " + ef.errorStyle.Render("^-- " + err.Message) + "\n")
			}
		}
		output.WriteString("  │\n")
	}

	// Help section with enhanced styling and icon
	if err.Help != "" {
		output.WriteString("  " + ef.helpStyle.Render("💡 help:") + " " + err.Help + "\n")
	}

	// Note section with enhanced styling and icon
	if err.Note != "" {
		output.WriteString("  " + ef.noteStyle.Render("📝 note:") + " " + err.Note + "\n")
	}

	// Suggestion section with enhanced code formatting
	if err.Suggestion != "" {
		output.WriteString("  " + ef.helpStyle.Render("🔧 suggestion:") + " ")
		if strings.Contains(err.Suggestion, ":") || strings.Contains(err.Suggestion, "{") {
			// Format as code if it looks like YAML
			output.WriteString("\n" + ef.formatCodeSuggestion(err.Suggestion))
		} else {
			output.WriteString(err.Suggestion + "\n")
		}
	}

	output.WriteString("\n") // Add spacing between errors

	return output.String()
}

func (ef *EnhancedFormatter) formatWarningEnhanced(warning ValidationError) string {
	var output strings.Builder

	// Warning header with enhanced styling and emoji
	output.WriteString(ef.warningStyle.Render("⚠️  warning") + ": " + ef.titleStyle.Render(warning.Title) + "\n")

	// Location with enhanced styling
	if warning.File != "" && warning.Line > 0 {
		location := fmt.Sprintf("%s:%d:%d", warning.File, warning.Line, warning.Column)
		output.WriteString("  " + ef.noteStyle.Render("┌─>") + " " + location + "\n")
		
		// Show actual file content if available
		if content := ef.getFileContext(warning.File, warning.Line, warning.Column); content != "" {
			output.WriteString(content)
		}
	} else if warning.File != "" {
		output.WriteString("  " + ef.noteStyle.Render("┌─>") + " " + warning.File + "\n")
	}

	// Field and value with highlighting
	if warning.Field != "" {
		output.WriteString("  │\n")
		if warning.Value != "" {
			highlighted := ef.warningStyle.Render(warning.Value)
			output.WriteString(fmt.Sprintf("  │ %s: %s\n", warning.Field, highlighted))
			
			// Show underline for warnings too
			fieldLen := len(warning.Field) + 2
			underline := strings.Repeat(" ", fieldLen) + ef.warningStyle.Render(strings.Repeat("~", len(warning.Value)))
			output.WriteString("  │ " + underline + " " + ef.warningStyle.Render(warning.Message) + "\n")
		} else {
			output.WriteString(fmt.Sprintf("  │ %s\n", warning.Field))
			if warning.Message != "" {
				output.WriteString("  │ " + strings.Repeat(" ", len(warning.Field)) + " " + ef.warningStyle.Render("~-- " + warning.Message) + "\n")
			}
		}
		output.WriteString("  │\n")
	}

	// Help section with enhanced styling and icon
	if warning.Help != "" {
		output.WriteString("  " + ef.helpStyle.Render("💡 help:") + " " + warning.Help + "\n")
	}

	// Note section for warnings
	if warning.Note != "" {
		output.WriteString("  " + ef.noteStyle.Render("📝 note:") + " " + warning.Note + "\n")
	}

	output.WriteString("\n") // Add spacing between warnings

	return output.String()
}

// getFileContext reads and formats the relevant lines from the source file
func (ef *EnhancedFormatter) getFileContext(filename string, line, column int) string {
	content, err := os.ReadFile(filename)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(content), "\n")
	if line <= 0 || line > len(lines) {
		return ""
	}

	var output strings.Builder
	
	// Show context: 2 lines before, the error line, 2 lines after for better context
	startLine := max(1, line-2)
	endLine := min(len(lines), line+2)
	
	// Calculate line number width for alignment
	lineNumWidth := len(strconv.Itoa(endLine))
	
	for i := startLine; i <= endLine; i++ {
		lineContent := ""
		if i <= len(lines) {
			lineContent = lines[i-1]
		}
		
		lineNum := ef.lineNumberStyle.Render(fmt.Sprintf("%*d", lineNumWidth, i))
		
		if i == line {
			// Highlight the error line with special styling
			output.WriteString(fmt.Sprintf("   %s │ %s\n", 
				ef.errorStyle.Render(fmt.Sprintf("%*d", lineNumWidth, i)), 
				lineContent))
			
			// Add pointer to specific column if available
			if column > 0 && column <= len(lineContent) {
				pointer := strings.Repeat(" ", column-1) + ef.highlightStyle.Render("^")
				if column > 1 {
					pointer = strings.Repeat(" ", column-1) + ef.highlightStyle.Render("^^^")
				}
				output.WriteString(fmt.Sprintf("   %s │ %s\n", 
					strings.Repeat(" ", lineNumWidth), pointer))
			}
		} else {
			// Context lines with subdued styling
			output.WriteString(fmt.Sprintf("   %s │ %s\n", lineNum, 
				ef.lineNumberStyle.Render(lineContent)))
		}
	}
	
	return output.String()
}

// formatCodeSuggestion formats YAML code suggestions with proper indentation
func (ef *EnhancedFormatter) formatCodeSuggestion(suggestion string) string {
	lines := strings.Split(suggestion, "\n")
	var formatted strings.Builder
	
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			formatted.WriteString("       " + ef.codeStyle.Render(line) + "\n")
		}
	}
	
	return formatted.String()
}

// FormatQuickFixEnhanced provides enhanced quick fix suggestions
func (ef *EnhancedFormatter) FormatQuickFixEnhanced(result *ValidationResult) string {
	if len(result.Errors) == 0 {
		return ""
	}

	var output strings.Builder
	output.WriteString("\n" + ef.helpStyle.Render("🚀 Quick Fixes") + "\n")
	output.WriteString(ef.lineNumberStyle.Render("┌─ Try these solutions to resolve the errors:") + "\n")

	fixCount := 0
	for _, err := range result.Errors {
		if fixCount >= 5 { // Limit to first 5 quick fixes
			remaining := len(result.Errors) - fixCount
			output.WriteString(ef.lineNumberStyle.Render("├─ ") + 
				ef.noteStyle.Render(fmt.Sprintf("... and %d more errors need attention", remaining)) + "\n")
			break
		}

		fix := ef.getQuickFixEnhanced(err)
		if fix != "" {
			prefix := "├─"
			if fixCount == len(result.Errors)-1 || fixCount == 4 {
				prefix = "└─"
			}
			output.WriteString(ef.lineNumberStyle.Render(prefix + " ") + fix + "\n")
			fixCount++
		}
	}

	if fixCount == 0 {
		output.WriteString(ef.lineNumberStyle.Render("└─ ") + 
			ef.noteStyle.Render("Review the error details above for specific guidance") + "\n")
	}

	return output.String()
}

func (ef *EnhancedFormatter) getQuickFixEnhanced(err ValidationError) string {
	switch {
	case strings.Contains(err.Title, "missing version"):
		return fmt.Sprintf("🔧 Add %s to the top of your config file", 
			ef.codeStyle.Render(`version: "1.0"`))
	case strings.Contains(err.Title, "source file not found"):
		return fmt.Sprintf("📁 Create missing file: %s", ef.codeStyle.Render(err.Value))
	case strings.Contains(err.Title, "invalid file mode"):
		return fmt.Sprintf("🔐 Change mode to %s in %s", 
			ef.codeStyle.Render(`"644"`), ef.codeStyle.Render(err.Field))
	case strings.Contains(err.Title, "empty package name"):
		return "🗑️  Remove empty package entries from your lists"
	case strings.Contains(err.Title, "invalid dconf path"):
		return fmt.Sprintf("🛤️  Add '/' prefix to dconf path: %s", 
			ef.codeStyle.Render("/"+err.Value))
	case strings.Contains(err.Title, "invalid package name"):
		if err.Suggestion != "" {
			return fmt.Sprintf("📦 Fix package name format (suggested: %s)", 
				ef.codeStyle.Render(err.Suggestion))
		}
		return "📦 Fix package name format according to manager conventions"
	case strings.Contains(err.Title, "missing repository"):
		return "📚 Add repository configuration before installing packages"
	case strings.Contains(err.Title, "invalid PPA"):
		return fmt.Sprintf("🔗 Use correct PPA format: %s", 
			ef.codeStyle.Render("user/repo"))
	case strings.Contains(err.Title, "unsafe destination"):
		return "🛡️  Use absolute paths or home-relative paths (~/) for destinations"
	case strings.Contains(err.Title, "conflicting flags"):
		return "⚡ Remove conflicting flags - choose one option per category"
	case strings.Contains(err.Title, "dangerous flag"):
		return "⚠️  Consider removing dangerous flags or ensure you understand the security implications"
	case strings.Contains(err.Title, "missing destination"):
		return fmt.Sprintf("🎯 Add destination path: %s", 
			ef.codeStyle.Render(`destination: "/path/to/target"`))
	case strings.Contains(err.Title, "missing source"):
		return fmt.Sprintf("📄 Add source path: %s", 
			ef.codeStyle.Render(`source: "path/to/file"`))
	case strings.Contains(err.Title, "include file not found"):
		return "📂 Create the missing include file or mark as optional with 'optional: true'"
	case strings.Contains(err.Title, "invalid glob"):
		return "🔍 Check glob pattern syntax - use *, ?, and [] correctly"
	case strings.Contains(err.Title, "circular"):
		return "🔄 Remove circular dependencies between include files"
	case strings.Contains(err.Title, "malformed dconf"):
		return "⚙️  Fix dconf path format - use single slashes to separate segments"
	default:
		if err.Help != "" {
			return "💡 " + err.Help
		}
		return "🔧 Check the configuration documentation for guidance"
	}
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}