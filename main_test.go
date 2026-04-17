package main

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// captureOutput captures stdout output during function execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestValidateYaml_ValidYaml(t *testing.T) {
	data := []byte(`name: test
description: testrepo
owner: myorg`)

	output := captureOutput(func() {
		errorCount, err := validateYaml(data, []string{"owner"}, []string{"name", "description", "owner"}, []string{}, []string{}, "test.yaml")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if errorCount != 0 {
			t.Errorf("Expected errorCount=0, got %d", errorCount)
		}
	})

	if output != "" {
		t.Errorf("Expected no output, got: %s", output)
	}
}

func TestValidateYaml_MissingRequiredField(t *testing.T) {
	data := []byte(`name: test
description: testrepo`)

	output := captureOutput(func() {
		errorCount, _ := validateYaml(data, []string{"owner"}, []string{"name", "description", "owner"}, []string{}, []string{}, "test.yaml")
		if errorCount == 0 {
			t.Errorf("Expected errorCount>0, got %d", errorCount)
		}
	})

	if !bytes.Contains([]byte(output), []byte("Missing field 'owner'")) {
		t.Errorf("Expected 'Missing field' error, got: %s", output)
	}
}

func TestValidateYaml_InvalidFieldName(t *testing.T) {
	data := []byte(`name: test
invalid_field: value
owner: myorg`)

	output := captureOutput(func() {
		errorCount, _ := validateYaml(data, []string{"owner"}, []string{"name", "owner"}, []string{}, []string{}, "test.yaml")
		if errorCount == 0 {
			t.Errorf("Expected errorCount>0, got %d", errorCount)
		}
	})

	if !bytes.Contains([]byte(output), []byte("Invalid field name")) {
		t.Errorf("Expected 'Invalid field name' error, got: %s", output)
	}
}

func TestValidateYaml_RequiredFieldOR_FirstAlternative(t *testing.T) {
	data := []byte(`name: test
owner: myorg`)

	output := captureOutput(func() {
		errorCount, err := validateYaml(data, []string{"owner|maintainer"}, []string{"name", "owner", "maintainer"}, []string{}, []string{}, "test.yaml")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if errorCount != 0 {
			t.Errorf("Expected errorCount=0, got %d", errorCount)
		}
	})

	if output != "" {
		t.Errorf("Expected no output, got: %s", output)
	}
}

func TestValidateYaml_RequiredFieldOR_SecondAlternative(t *testing.T) {
	data := []byte(`name: test
maintainer: john`)

	output := captureOutput(func() {
		errorCount, err := validateYaml(data, []string{"owner|maintainer"}, []string{"name", "owner", "maintainer"}, []string{}, []string{}, "test.yaml")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if errorCount != 0 {
			t.Errorf("Expected errorCount=0, got %d", errorCount)
		}
	})

	if output != "" {
		t.Errorf("Expected no output, got: %s", output)
	}
}

func TestValidateYaml_RequiredFieldOR_MissingBoth(t *testing.T) {
	data := []byte(`name: test`)

	output := captureOutput(func() {
		errorCount, _ := validateYaml(data, []string{"owner|maintainer"}, []string{"name", "owner", "maintainer"}, []string{}, []string{}, "test.yaml")
		if errorCount == 0 {
			t.Errorf("Expected errorCount>0, got %d", errorCount)
		}
	})

	if !bytes.Contains([]byte(output), []byte("Missing required field (one of:")) {
		t.Errorf("Expected 'Missing required field' error, got: %s", output)
	}
}

func TestValidateYaml_ValidFieldGroup_FirstAlternative(t *testing.T) {
	data := []byte(`name: test
owner: myorg`)

	output := captureOutput(func() {
		errorCount, err := validateYaml(data, []string{"owner"}, []string{"name|names", "owner"}, []string{}, []string{}, "test.yaml")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if errorCount != 0 {
			t.Errorf("Expected errorCount=0, got %d", errorCount)
		}
	})

	if output != "" {
		t.Errorf("Expected no output, got: %s", output)
	}
}

func TestValidateYaml_ValidFieldGroup_SecondAlternative(t *testing.T) {
	data := []byte(`names:
  - repo1
  - repo2
owner: myorg`)

	output := captureOutput(func() {
		errorCount, err := validateYaml(data, []string{"owner"}, []string{"name|names", "owner"}, []string{}, []string{}, "test.yaml")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if errorCount != 0 {
			t.Errorf("Expected errorCount=0, got %d", errorCount)
		}
	})

	if output != "" {
		t.Errorf("Expected no output, got: %s", output)
	}
}

func TestValidateYaml_ValidFieldGroup_BothAlternatives(t *testing.T) {
	data := []byte(`name: test
names:
  - repo1
owner: myorg`)

	output := captureOutput(func() {
		errorCount, _ := validateYaml(data, []string{"owner"}, []string{"name|names", "owner"}, []string{}, []string{}, "test.yaml")
		if errorCount == 0 {
			t.Errorf("Expected errorCount>0, got %d", errorCount)
		}
	})

	if !bytes.Contains([]byte(output), []byte("conflicts with")) {
		t.Errorf("Expected 'conflicts with' error, got: %s", output)
	}
}

func TestValidateYaml_MultipleRequiredFields(t *testing.T) {
	data := []byte(`name: test
owner: myorg
maintainer: john`)

	output := captureOutput(func() {
		errorCount, err := validateYaml(data, []string{"name", "owner"}, []string{"name", "owner", "maintainer"}, []string{}, []string{}, "test.yaml")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if errorCount != 0 {
			t.Errorf("Expected errorCount=0, got %d", errorCount)
		}
	})

	if output != "" {
		t.Errorf("Expected no output, got: %s", output)
	}
}

func TestValidateYaml_InvalidYaml(t *testing.T) {
	data := []byte(`{invalid yaml: [`)

	output := captureOutput(func() {
		errorCount, _ := validateYaml(data, []string{"owner"}, []string{"owner"}, []string{}, []string{}, "test.yaml")
		if errorCount == 0 {
			t.Errorf("Expected errorCount>0, got %d", errorCount)
		}
	})

	if !bytes.Contains([]byte(output), []byte("Error parsing file as yaml")) {
		t.Errorf("Expected 'Error parsing' message, got: %s", output)
	}
}

func TestValidateYaml_ComplexScenario(t *testing.T) {
	// Valid yaml with one from each group and all required (with OR logic)
	data := []byte(`name: test
owner: myorg
layout: standard`)

	output := captureOutput(func() {
		errorCount, err := validateYaml(
			data,
			[]string{"owner|maintainer", "name"},
			[]string{"name|names", "owner", "maintainer", "layout", "description"},
			[]string{},
			[]string{},
			"test.yaml",
		)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if errorCount != 0 {
			t.Errorf("Expected errorCount=0, got %d", errorCount)
		}
	})

	if output != "" {
		t.Errorf("Expected no output, got: %s", output)
	}
}

func TestValidateYaml_EmptyYaml(t *testing.T) {
	data := []byte(``)

	output := captureOutput(func() {
		errorCount, _ := validateYaml(data, []string{"owner"}, []string{"owner"}, []string{}, []string{}, "test.yaml")
		if errorCount == 0 {
			t.Errorf("Expected errorCount>0, got %d", errorCount)
		}
	})

	if !bytes.Contains([]byte(output), []byte("Missing field 'owner'")) {
		t.Errorf("Expected 'Missing field' error for empty yaml, got: %s", output)
	}
}
func TestCheckFiles_Stats(t *testing.T) {
	// This test verifies stats are correctly tracked
	// Create temporary test files
	tempDir := t.TempDir()

	// Valid yaml file
	validFile := tempDir + "/valid.yaml"
	os.WriteFile(validFile, []byte(`name: test
owner: myorg`), 0600)

	// Invalid yaml file (missing required field)
	invalidFile := tempDir + "/invalid.yaml"
	os.WriteFile(invalidFile, []byte(`name: test`), 0600)

	// Suppress output during test
	output := captureOutput(func() {
		_, _ = checkFiles([]string{validFile, invalidFile}, []string{"owner"}, []string{"name", "owner"}, []string{}, []string{}, "")
	})

	// Just verify the function runs, checking actual stats in next test
	if output == "" {
		t.Errorf("Expected some output from validation errors")
	}
}

func TestCheckFiles_StatsCount(t *testing.T) {
	// Test that stats are correctly counted
	tempDir := t.TempDir()

	// Valid yaml file
	validFile := tempDir + "/valid.yaml"
	os.WriteFile(validFile, []byte(`name: test
owner: myorg`), 0600)

	// Invalid yaml file (missing required field)
	invalidFile := tempDir + "/invalid.yaml"
	os.WriteFile(invalidFile, []byte(`name: test`), 0600)

	stats, err := captureOutputWithStats(func() (ValidationStats, error) {
		return checkFiles([]string{validFile, invalidFile}, []string{"owner"}, []string{"name", "owner"}, []string{}, []string{}, "")
	})

	if stats.TotalFiles != 2 {
		t.Errorf("Expected Total=2, got %d", stats.TotalFiles)
	}
	if stats.InvalidFiles != 1 {
		t.Errorf("Expected Invalid=1, got %d", stats.InvalidFiles)
	}
	if stats.Errors == 0 {
		t.Errorf("Expected Errors>0, got %d", stats.Errors)
	}
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestValidateYaml_TrailingWhitespace(t *testing.T) {
	data := []byte(`name: test space
owner: myorg`)

	output := captureOutput(func() {
		errorCount, _ := validateYaml(data, []string{"owner"}, []string{"name", "owner"}, []string{}, []string{}, "test.yaml")
		if errorCount == 0 {
			t.Errorf("Expected errorCount>0, got %d", errorCount)
		}
	})

	if !bytes.Contains([]byte(output), []byte("contains spaces")) {
		t.Errorf("Expected 'contains spaces' error, got: %s", output)
	}
}

func TestValidateYaml_NoTrailingWhitespace(t *testing.T) {
	data := []byte(`name: test
owner: myorg`)

	output := captureOutput(func() {
		errorCount, err := validateYaml(data, []string{"owner"}, []string{"name", "owner"}, []string{}, []string{}, "test.yaml")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if errorCount != 0 {
			t.Errorf("Expected errorCount=0, got %d", errorCount)
		}
	})

	if output != "" {
		t.Errorf("Expected no output, got: %s", output)
	}
}

// Helper function to capture output and stats
func captureOutputWithStats(f func() (ValidationStats, error)) (ValidationStats, error) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	stats, err := f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	return stats, err
}

func TestValidateYaml_AllLowercaseValues(t *testing.T) {
	data := []byte(`name: myrepo
owner: acmeorg
layout: standard`)

	output := captureOutput(func() {
		errorCount, err := validateYaml(data, []string{"owner"}, []string{"name", "owner", "layout"}, []string{}, []string{}, "test.yaml")
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if errorCount != 0 {
			t.Errorf("Expected errorCount=0 for all lowercase values, got %d", errorCount)
		}
	})

	if bytes.Contains([]byte(output), []byte("uppercase")) {
		t.Errorf("Expected no uppercase error for lowercase values, got: %s", output)
	}
}

func TestValidateYaml_UppercaseFieldValues(t *testing.T) {
	data := []byte(`name: MyRepo
owner: AcmeOrg
layout: standard`)

	output := captureOutput(func() {
		errorCount, _ := validateYaml(data, []string{"owner"}, []string{"name", "owner", "layout"}, []string{}, []string{}, "test.yaml")
		if errorCount == 0 {
			t.Errorf("Expected errorCount>0 for uppercase values, got %d", errorCount)
		}
	})

	if !bytes.Contains([]byte(output), []byte("uppercase")) {
		t.Errorf("Expected 'uppercase' error message, got: %s", output)
	}
}
