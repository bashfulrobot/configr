package config

import (
	"fmt"
	"strings"
)

// FormatValidationResultSimple provides a simple formatter without complex styling
func FormatValidationResultSimple(result *ValidationResult) string {
	if len(result.Errors) == 0 && len(result.Warnings) == 0 {
		return "✓ Configuration is valid\n"
	}
	
	var output strings.Builder
	
	// Format errors
	for i, err := range result.Errors {
		if i > 0 {
			output.WriteString("\n")
		}
		output.WriteString(formatErrorSimple(err))
	}
	
	// Format warnings  
	for i, warning := range result.Warnings {
		if len(result.Errors) > 0 || i > 0 {
			output.WriteString("\n")
		}
		output.WriteString(formatWarningSimple(warning))
	}
	
	// Summary
	if len(result.Errors) > 0 {
		output.WriteString(fmt.Sprintf("\nError: could not validate configuration due to %d errors\n", len(result.Errors)))
	}
	
	return output.String()
}

func formatErrorSimple(err ValidationError) string {
	var output strings.Builder
	
	// Error header
	output.WriteString(fmt.Sprintf("error: %s\n", err.Title))
	
	// Location if available
	if err.File != "" && err.Line > 0 {
		output.WriteString(fmt.Sprintf("  --> %s:%d:%d\n", err.File, err.Line, err.Column))
	} else if err.File != "" {
		output.WriteString(fmt.Sprintf("  --> %s\n", err.File))
	}
	
	// Field and value
	if err.Field != "" {
		output.WriteString("   |\n")
		if err.Value != "" {
			output.WriteString(fmt.Sprintf("   | %s: %s\n", err.Field, err.Value))
		} else {
			output.WriteString(fmt.Sprintf("   | %s\n", err.Field))
		}
		output.WriteString("   |\n")
	}
	
	// Help section
	if err.Help != "" {
		output.WriteString(fmt.Sprintf("   = help: %s\n", err.Help))
	}
	
	// Note section
	if err.Note != "" {
		output.WriteString(fmt.Sprintf("   = note: %s\n", err.Note))
	}
	
	// Suggestion section
	if err.Suggestion != "" {
		output.WriteString(fmt.Sprintf("   = suggestion: %s\n", err.Suggestion))
	}
	
	return output.String()
}

func formatWarningSimple(warning ValidationError) string {
	var output strings.Builder
	
	// Warning header
	output.WriteString(fmt.Sprintf("warning: %s\n", warning.Title))
	
	// Location if available
	if warning.File != "" && warning.Line > 0 {
		output.WriteString(fmt.Sprintf("  --> %s:%d:%d\n", warning.File, warning.Line, warning.Column))
	}
	
	// Field and value
	if warning.Field != "" {
		output.WriteString("   |\n")
		if warning.Value != "" {
			output.WriteString(fmt.Sprintf("   | %s: %s\n", warning.Field, warning.Value))
		}
		output.WriteString("   |\n")
	}
	
	// Help section
	if warning.Help != "" {
		output.WriteString(fmt.Sprintf("   = help: %s\n", warning.Help))
	}
	
	return output.String()
}

// FormatQuickFixSimple provides simple quick fix suggestions
func FormatQuickFixSimple(result *ValidationResult) string {
	if len(result.Errors) == 0 {
		return ""
	}
	
	var output strings.Builder
	output.WriteString("\nQuick fixes:\n")
	
	for i, err := range result.Errors {
		if i >= 3 {
			remaining := len(result.Errors) - 3
			output.WriteString(fmt.Sprintf("  ... and %d more errors\n", remaining))
			break
		}
		
		fix := getQuickFixSimple(err)
		if fix != "" {
			output.WriteString(fmt.Sprintf("  • %s\n", fix))
		}
	}
	
	return output.String()
}

func getQuickFixSimple(err ValidationError) string {
	switch {
	case strings.Contains(err.Title, "missing version"):
		return "Add 'version: \"1.0\"' to the top of your config file"
	case strings.Contains(err.Title, "source file not found"):
		return fmt.Sprintf("Create missing file: %s", err.Value)
	case strings.Contains(err.Title, "invalid file mode"):
		return fmt.Sprintf("Change mode to \"644\" in %s", err.Field)
	case strings.Contains(err.Title, "empty package name"):
		return "Remove empty package entries from your lists"
	case strings.Contains(err.Title, "invalid dconf path"):
		return fmt.Sprintf("Add '/' prefix to dconf path: %s", err.Value)
	default:
		return err.Help
	}
}