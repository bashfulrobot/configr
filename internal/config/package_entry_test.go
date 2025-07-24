package config

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPackageEntry_UnmarshalYAML_SimpleString(t *testing.T) {
	yamlData := `- git`
	
	var packages []PackageEntry
	err := yaml.Unmarshal([]byte(yamlData), &packages)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	
	if len(packages) != 1 {
		t.Errorf("expected 1 package, got %d", len(packages))
	}
	
	pkg := packages[0]
	if pkg.Name != "git" {
		t.Errorf("expected name 'git', got '%s'", pkg.Name)
	}
	
	if len(pkg.Flags) != 0 {
		t.Errorf("expected no flags for simple string, got %v", pkg.Flags)
	}
}

func TestPackageEntry_UnmarshalYAML_ComplexObject(t *testing.T) {
	yamlData := `
- "docker.io":
    flags: ["-y", "--install-suggests"]
`
	
	var packages []PackageEntry
	err := yaml.Unmarshal([]byte(yamlData), &packages)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	
	if len(packages) != 1 {
		t.Errorf("expected 1 package, got %d", len(packages))
	}
	
	pkg := packages[0]
	if pkg.Name != "docker.io" {
		t.Errorf("expected name 'docker.io', got '%s'", pkg.Name)
	}
	
	expectedFlags := []string{"-y", "--install-suggests"}
	if !reflect.DeepEqual(pkg.Flags, expectedFlags) {
		t.Errorf("expected flags %v, got %v", expectedFlags, pkg.Flags)
	}
}

func TestPackageEntry_UnmarshalYAML_MixedFormats(t *testing.T) {
	yamlData := `
- git
- curl
- "docker.io":
    flags: ["-y", "--install-suggests"]
- vim
`
	
	var packages []PackageEntry
	err := yaml.Unmarshal([]byte(yamlData), &packages)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	
	if len(packages) != 4 {
		t.Errorf("expected 4 packages, got %d", len(packages))
	}
	
	// Test simple packages
	expectedSimple := []string{"git", "curl", "vim"}
	simpleIndices := []int{0, 1, 3}
	
	for i, idx := range simpleIndices {
		pkg := packages[idx]
		if pkg.Name != expectedSimple[i] {
			t.Errorf("expected package %d name '%s', got '%s'", idx, expectedSimple[i], pkg.Name)
		}
		if len(pkg.Flags) != 0 {
			t.Errorf("expected no flags for simple package %d, got %v", idx, pkg.Flags)
		}
	}
	
	// Test complex package
	complexPkg := packages[2]
	if complexPkg.Name != "docker.io" {
		t.Errorf("expected complex package name 'docker.io', got '%s'", complexPkg.Name)
	}
	
	expectedFlags := []string{"-y", "--install-suggests"}
	if !reflect.DeepEqual(complexPkg.Flags, expectedFlags) {
		t.Errorf("expected complex package flags %v, got %v", expectedFlags, complexPkg.Flags)
	}
}

func TestPackageEntry_UnmarshalYAML_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		yamlData     string
		expectedName string
		shouldError  bool
	}{
		{
			name:         "number as string",
			yamlData:     `- 123`,
			expectedName: "123",
			shouldError:  false,
		},
		{
			name:         "boolean as string", 
			yamlData:     `- true`,
			expectedName: "true",
			shouldError:  false,
		},
		{
			name: "multiple keys in object - should error",
			yamlData: `
- "pkg1":
    flags: ["--flag1"]
  "pkg2":
    flags: ["--flag2"]
`,
			shouldError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var packages []PackageEntry
			err := yaml.Unmarshal([]byte(tt.yamlData), &packages)
			
			if tt.shouldError {
				if err == nil {
					t.Errorf("expected error for %s, but unmarshaling succeeded", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %s: %v", tt.name, err)
				}
				if len(packages) > 0 && packages[0].Name != tt.expectedName {
					t.Errorf("expected name '%s', got '%s'", tt.expectedName, packages[0].Name)
				}
			}
		})
	}
}

func TestPackageEntry_MarshalYAML_SimplePackage(t *testing.T) {
	pkg := PackageEntry{
		Name:  "git",
		Flags: []string{},
	}
	
	data, err := yaml.Marshal(pkg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	
	// Should marshal as simple string
	expected := "git\n"
	if string(data) != expected {
		t.Errorf("expected '%s', got '%s'", expected, string(data))
	}
}

func TestPackageEntry_MarshalYAML_ComplexPackage(t *testing.T) {
	pkg := PackageEntry{
		Name:  "docker.io",
		Flags: []string{"-y", "--install-suggests"},
	}
	
	data, err := yaml.Marshal(pkg)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	
	// Should marshal as object with flags
	var result map[string]interface{}
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	
	if dockerConfig, ok := result["docker.io"].(map[string]interface{}); ok {
		if flags, ok := dockerConfig["flags"].([]interface{}); ok {
			if len(flags) != 2 {
				t.Errorf("expected 2 flags, got %d", len(flags))
			}
		} else {
			t.Error("expected flags field in marshaled complex package")
		}
	} else {
		t.Error("expected docker.io key in marshaled complex package")
	}
}

func TestPackageEntry_RoundTrip(t *testing.T) {
	// Test that we can unmarshal and then marshal back to equivalent YAML
	original := []PackageEntry{
		{Name: "git"},
		{Name: "curl"},
		{Name: "docker.io", Flags: []string{"-y", "--install-suggests"}},
	}
	
	// Marshal to YAML
	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	
	// Unmarshal back
	var unmarshaled []PackageEntry
	err = yaml.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	
	// Compare
	if len(unmarshaled) != len(original) {
		t.Errorf("expected %d packages after round trip, got %d", len(original), len(unmarshaled))
	}
	
	for i, pkg := range unmarshaled {
		if pkg.Name != original[i].Name {
			t.Errorf("package %d name mismatch: expected '%s', got '%s'", i, original[i].Name, pkg.Name)
		}
		
		// Handle empty slice vs nil slice comparison
		originalFlags := original[i].Flags
		if originalFlags == nil {
			originalFlags = []string{}
		}
		pkgFlags := pkg.Flags
		if pkgFlags == nil {
			pkgFlags = []string{}
		}
		
		if !reflect.DeepEqual(pkgFlags, originalFlags) {
			t.Errorf("package %d flags mismatch: expected %v, got %v", i, originalFlags, pkgFlags)
		}
	}
}