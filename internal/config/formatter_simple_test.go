package config

import (
	"strings"
	"testing"
)

func TestFormatValidationResultSimple_NoErrors(t *testing.T) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}
	
	output := FormatValidationResultSimple(result)
	
	expected := "✓ Configuration is valid\n"
	if output != expected {
		t.Errorf("expected '%s', got '%s'", expected, output)
	}
}

func TestFormatValidationResultSimple_WithErrors(t *testing.T) {
	result := &ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{
				Type:    "error",
				Title:   "missing version field",
				File:    "configr.yaml",
				Line:    1,
				Column:  1,
				Field:   "version",
				Message: "configuration version is required",
				Help:    "add 'version: \"1.0\"' to your configuration",
				Note:    "version helps ensure compatibility",
			},
		},
		Warnings: []ValidationError{},
	}
	
	output := FormatValidationResultSimple(result)
	
	// Check that all parts are present
	if !strings.Contains(output, "error: missing version field") {
		t.Error("output should contain error title")
	}
	
	if !strings.Contains(output, "configr.yaml:1:1") {
		t.Error("output should contain file location")
	}
	
	if !strings.Contains(output, "help: add 'version: \"1.0\"'") {
		t.Error("output should contain help text")
	}
	
	if !strings.Contains(output, "note: version helps ensure compatibility") {
		t.Error("output should contain note")
	}
	
	if !strings.Contains(output, "Error: could not validate configuration due to 1 errors") {
		t.Error("output should contain error summary")
	}
}

func TestFormatValidationResultSimple_WithWarnings(t *testing.T) {
	result := &ValidationResult{
		Valid: true,
		Errors: []ValidationError{},
		Warnings: []ValidationError{
			{
				Type:  "warning",
				Title: "overly permissive mode",
				Field: "files.test.mode",
				Value: "777",
				Help:  "consider using '644' for better security",
			},
		},
	}
	
	output := FormatValidationResultSimple(result)
	
	if !strings.Contains(output, "warning: overly permissive mode") {
		t.Error("output should contain warning title")
	}
	
	if !strings.Contains(output, "files.test.mode: 777") {
		t.Error("output should contain field and value")
	}
	
	if !strings.Contains(output, "help: consider using '644'") {
		t.Error("output should contain help text")
	}
}

func TestFormatValidationResultSimple_ErrorsAndWarnings(t *testing.T) {
	result := &ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{
				Type:  "error",
				Title: "source file not found",
				Field: "files.vimrc.source",
				Value: "dotfiles/vimrc",
			},
		},
		Warnings: []ValidationError{
			{
				Type:  "warning",
				Title: "overly permissive mode",
				Field: "files.test.mode",
			},
		},
	}
	
	output := FormatValidationResultSimple(result)
	
	// Should contain both errors and warnings
	if !strings.Contains(output, "error: source file not found") {
		t.Error("output should contain error")
	}
	
	if !strings.Contains(output, "warning: overly permissive mode") {
		t.Error("output should contain warning")
	}
	
	// Should have error summary
	if !strings.Contains(output, "Error: could not validate configuration due to 1 errors") {
		t.Error("output should contain error summary even with warnings present")
	}
}

func TestFormatQuickFixSimple_NoErrors(t *testing.T) {
	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}
	
	output := FormatQuickFixSimple(result)
	
	if output != "" {
		t.Errorf("expected empty output for no errors, got '%s'", output)
	}
}

func TestFormatQuickFixSimple_WithErrors(t *testing.T) {
	result := &ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{
				Title: "missing version field",
				Field: "version",
			},
			{
				Title: "source file not found",
				Field: "files.vimrc.source",
				Value: "dotfiles/vimrc",
			},
		},
	}
	
	output := FormatQuickFixSimple(result)
	
	if !strings.Contains(output, "Quick fixes:") {
		t.Error("output should contain quick fixes header")
	}
	
	if !strings.Contains(output, "Add 'version: \"1.0\"' to the top") {
		t.Error("output should contain version fix suggestion")
	}
	
	if !strings.Contains(output, "Create missing file: dotfiles/vimrc") {
		t.Error("output should contain file creation suggestion")
	}
}

func TestFormatQuickFixSimple_ManyErrors(t *testing.T) {
	// Test that only first 3 errors are shown with "... and X more errors"
	result := &ValidationResult{
		Valid: false,
		Errors: []ValidationError{
			{Title: "error 1"},
			{Title: "error 2"},
			{Title: "error 3"},
			{Title: "error 4"},
			{Title: "error 5"},
		},
	}
	
	output := FormatQuickFixSimple(result)
	
	if !strings.Contains(output, "... and 2 more errors") {
		t.Error("output should show remaining error count when more than 3 errors")
	}
}

func TestGetQuickFixSimple(t *testing.T) {
	tests := []struct {
		name        string
		error       ValidationError
		expectedFix string
	}{
		{
			name: "missing version",
			error: ValidationError{
				Title: "missing version field",
			},
			expectedFix: "Add 'version: \"1.0\"' to the top of your config file",
		},
		{
			name: "source file not found",
			error: ValidationError{
				Title: "source file not found",
				Value: "dotfiles/vimrc",
			},
			expectedFix: "Create missing file: dotfiles/vimrc",
		},
		{
			name: "invalid file mode",
			error: ValidationError{
				Title: "invalid file mode",
				Field: "files.test.mode",
			},
			expectedFix: "Change mode to \"644\" in files.test.mode",
		},
		{
			name: "empty package name",
			error: ValidationError{
				Title: "empty package name",
			},
			expectedFix: "Remove empty package entries from your lists",
		},
		{
			name: "invalid dconf path",
			error: ValidationError{
				Title: "invalid dconf path",
				Value: "org/gnome/test",
			},
			expectedFix: "Add '/' prefix to dconf path: org/gnome/test",
		},
		{
			name: "unknown error with help",
			error: ValidationError{
				Title: "unknown error type",
				Help:  "this is the help text",
			},
			expectedFix: "this is the help text",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fix := getQuickFixSimple(tt.error)
			if fix != tt.expectedFix {
				t.Errorf("expected fix '%s', got '%s'", tt.expectedFix, fix)
			}
		})
	}
}