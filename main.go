package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
)

type ValidationStats struct {
	TotalFiles   int
	InvalidFiles int
	Errors       int
}

type FieldInfo struct {
	Name string
	Line int
}

func main() {
	folder, requiredFields, validFields, allowUppercaseFields, allowSpaceFields, pattern, err := parseArguments()
	if err != nil {
		fmt.Printf("%v\n\n", err)
		fmt.Println(
			"Usage: gitcheckdiff" +
				" -folder=<folder>" +
				" -required=<fields>" +
				" -valid=<fields>" +
				" [-allowUppercase=<fields>]" +
				" [-allowSpace=<fields>]" +
				" [-pattern=<glob pattern>]")
		fmt.Println()
		flag.PrintDefaults()
		os.Exit(1)
	}

	files, err := getFiles(folder)
	if err != nil {
		fmt.Printf("Error getting files: %v\n", err)
		os.Exit(1)
	}
	if len(files) == 0 {
		fmt.Printf("Error: didn't find any files in folder '%s'\n", folder)
		os.Exit(1)
	}

	stats, err := checkFiles(files, requiredFields, validFields, allowUppercaseFields, allowSpaceFields, pattern)
	if err != nil {
		fmt.Printf("Validation of modified files failed: %v\n", err)
	}

	fmt.Println("Validation Statistics:")
	fmt.Printf("  Total files:     %d\n", stats.TotalFiles)
	fmt.Printf("  Valid files:     %d\n", stats.TotalFiles-stats.InvalidFiles)
	fmt.Printf("  Invalid files:   %d\n", stats.InvalidFiles)
	fmt.Printf("  Total errors:    %d\n", stats.Errors)

	if err != nil {
		os.Exit(1)
	}

	fmt.Println("Done.")
}

func getFiles(folder string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(folder, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking folder: %w", err)
	}
	return files, nil
}

func checkFiles(files, requiredFields, validFields, allowUppercaseFields, allowSpaceFields []string, pattern string) (ValidationStats, error) {
	stats := ValidationStats{TotalFiles: 0, InvalidFiles: 0, Errors: 0}

	for _, filename := range files {
		stats.TotalFiles++

		if pattern != "" {
			match, err := filepath.Match(pattern, filepath.Base(filename))
			if err != nil {
				fmt.Printf("Error matching pattern '%s' against '%s': %v\n", pattern, filename, err)
				stats.Errors++
				stats.InvalidFiles++
				continue
			}
			if !match {
				fmt.Printf("Filename '%s' does not match pattern '%s'\n", filename, pattern)
				stats.Errors++
				stats.InvalidFiles++
				continue
			}
		}

		data, err := os.ReadFile(filename)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("%s: File does not exist.\n", filename)
			} else {
				fmt.Printf("%v\n", err)
			}
			stats.Errors++
			stats.InvalidFiles++
			continue
		}

		errorCount, err := validateYaml(data, requiredFields, validFields, allowUppercaseFields, allowSpaceFields, filename)
		if err != nil {
			stats.InvalidFiles++
		}
		stats.Errors += errorCount
	}
	if stats.Errors > 0 {
		return stats, fmt.Errorf("see above errors")
	}

	return stats, nil
}

func validateYaml(data []byte, requiredFields, validFields, allowUppercaseFields, allowSpaceFields []string, filename string) (int, error) {
	errorCount := checkTrailingWhitespaces(data, filename)

	var node ast.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		fmt.Printf("%s: Error parsing file as yaml: %v\n", filename, err)
		return errorCount + 1, fmt.Errorf("")
	}

	errorCount += checkFieldValues(node, allowUppercaseFields, allowSpaceFields, filename)

	fields := extractFieldNames(node)

	errorCount += checkRequiredFields(fields, requiredFields, filename)
	errorCount += checkValidFields(fields, validFields, filename)

	if errorCount > 0 {
		return errorCount, fmt.Errorf("")
	}

	return 0, nil
}

func checkFieldValues(node ast.Node, allowUppercaseFields, allowSpaceFields []string, filename string) int {
	errorCount := 0

	allowUppercaseMap := make(map[string]bool)
	allowSpaceMap := make(map[string]bool)
	for _, field := range allowUppercaseFields {
		if field != "" {
			allowUppercaseMap[field] = true
		}
	}
	for _, field := range allowSpaceFields {
		if field != "" {
			allowSpaceMap[field] = true
		}
	}

	if node != nil {
		t := node.GetToken()
		if t != nil {
			for {
				if t.Prev == nil {
					break
				} else {
					t = t.Prev
				}
			}

			for {
				if t.Prev != nil && t.Prev.Value == ":" && t.Value != "" && t.Value != "-" {
					fieldName := ""
					if t.Prev.Prev != nil {
						fieldName = t.Prev.Prev.Value
					}

					if strings.Contains(t.Value, " ") && !allowSpaceMap[fieldName] {
						fmt.Printf("%s: Field value '%s' contains spaces at line %d\n", filename, t.Value, t.Position.Line)
						errorCount++
					}

					if !allowUppercaseMap[fieldName] {
						for _, ch := range t.Value {
							if ch >= 'A' && ch <= 'Z' {
								fmt.Printf("%s: Field value '%s' contains uppercase letters at line %d\n", filename, t.Value, t.Position.Line)
								errorCount++
								break
							}
						}
					}
				}
				t = t.Next
				if t == nil {
					break
				}
			}
		}
	}

	return errorCount
}

func checkTrailingWhitespaces(data []byte, filename string) int {
	errorCount := 0
	lines := strings.Split(string(data), "\n")
	for lineNum, line := range lines {
		if line != strings.TrimRight(line, " \t\r") {
			fmt.Printf("%s: File has trailing whitespace at line %d: '%s'\n", filename, lineNum+1, line)
			errorCount++
		}
	}
	return errorCount
}

func extractFieldNames(node ast.Node) []FieldInfo {
	var fields []FieldInfo

	if node != nil {
		t := node.GetToken()
		for {
			if t.Prev == nil {
				break
			} else {
				t = t.Prev
			}
		}

		for {
			if (t.Prev == nil && t.Value != "-" && t.Value != ":") ||
				(t.Value != "-" && t.Value != ":" &&
					t.Prev != nil && t.Prev.Value != "-" && t.Prev.Value != ":" &&
					t.Prev.Prev != nil && (t.Prev.Prev.Value == "-" || t.Prev.Prev.Value == ":")) {

				fields = append(fields, FieldInfo{Name: t.Value, Line: t.Position.Line})
			}
			t = t.Next
			if t == nil {
				break
			}
		}
	}

	return fields
}

func checkRequiredFields(fields []FieldInfo, requiredFields []string, filename string) int {
	errorCount := 0

	foundFields := map[string]bool{}
	for _, field := range fields {
		foundFields[field.Name] = true
	}

	for _, fieldname := range requiredFields {
		if strings.Contains(fieldname, "|") {
			alternatives := strings.Split(fieldname, "|")
			found := false
			for _, alt := range alternatives {
				if foundFields[alt] {
					found = true
					break
				}
			}
			if !found {
				fmt.Printf("%s: Missing required field (one of: %s)\n", filename, fieldname)
				errorCount++
			}
		} else {
			if !foundFields[fieldname] {
				fmt.Printf("%s: Missing field '%s'\n", filename, fieldname)
				errorCount++
			}
		}
	}
	return errorCount
}

func checkValidFields(fields []FieldInfo, validFields []string, filename string) int {
	errorCount := 0

	type validFieldGroup struct {
		alternatives []string
		found        string
	}
	var fieldGroups []validFieldGroup
	fieldToGroupIdx := make(map[string]int)

	for _, field := range validFields {
		if strings.Contains(field, "|") {
			alternatives := strings.Split(field, "|")
			for j := range alternatives {
				fieldToGroupIdx[alternatives[j]] = len(fieldGroups)
			}
			fieldGroups = append(fieldGroups, validFieldGroup{alternatives: alternatives})
		} else {
			fieldToGroupIdx[field] = -1
		}
	}

	for _, field := range fields {
		groupIdx, isValid := fieldToGroupIdx[field.Name]
		if !isValid {
			fmt.Printf("%s: Invalid field name '%s' at line %d\n", filename, field.Name, field.Line)
			errorCount++
		} else if groupIdx >= 0 {
			// Field is in a group; check if another alternative already seen
			if fieldGroups[groupIdx].found != "" && fieldGroups[groupIdx].found != field.Name {
				fmt.Printf("%s: Field '%s' conflicts with '%s' (only one allowed) at line %d\n", filename, field.Name, fieldGroups[groupIdx].found, field.Line)
				errorCount++
			}
			fieldGroups[groupIdx].found = field.Name
		}
	}
	return errorCount
}

func parseArguments() (string, []string, []string, []string, []string, string, error) {
	folder := flag.String("folder", "", "Location of yaml files")
	required := flag.String("required", "", "Comma-separated list of required fields in the input yaml")
	valid := flag.String("valid", "", "Comma-separated list of valid fields in the input yaml")
	allowUppercase := flag.String("allowUppercase", "", "Comma-separated list of fields whose values can contain uppercase letters")
	allowSpace := flag.String("allowSpace", "", "Comma-separated list of fields whose values can contain spaces")
	pattern := flag.String("pattern", "", "Glob pattern that files must match (e.g. '*.yaml')")

	flag.Parse()

	if *folder == "" {
		return "", nil, nil, nil, nil, "", fmt.Errorf("missing or empty parameter: -folder")
	}
	if !isFlagPassed("required") {
		return "", nil, nil, nil, nil, "", fmt.Errorf("missing or empty parameter: -required")
	}
	if !isFlagPassed("valid") {
		return "", nil, nil, nil, nil, "", fmt.Errorf("missing or empty parameter: -valid")
	}

	requiredFields := strings.Split(*required, ",")
	validFields := strings.Split(*valid, ",")
	allowUppercaseFields := strings.Split(*allowUppercase, ",")
	allowSpaceFields := strings.Split(*allowSpace, ",")

	return *folder, requiredFields, validFields, allowUppercaseFields, allowSpaceFields, *pattern, nil
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
