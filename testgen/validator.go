package testgen

import (
	"fmt"
	"os"
	"strings"

	"github.com/tablev/test-coverage-agent/coverage"
)

// Validator validates generated tests
type Validator struct {
	analyzer coverage.Analyzer
}

// NewValidator creates a new test validator
func NewValidator(analyzer coverage.Analyzer) *Validator {
	return &Validator{
		analyzer: analyzer,
	}
}

// ValidationResult represents the result of test validation
type ValidationResult struct {
	Success        bool
	CompilationOK  bool
	TestsPassed    bool
	Output         string
	ErrorMessage   string
	FailedTests    []string
	CoverageGained float64
}

// ValidateTest validates a test file
func (v *Validator) ValidateTest(projectPath, testFile string) (*ValidationResult, error) {
	result := &ValidationResult{
		Success:       false,
		CompilationOK: false,
		TestsPassed:   false,
	}

	// Validate the test file (compile and run)
	success, output, err := v.analyzer.ValidateTestFile(projectPath, testFile)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	result.Output = output

	// Check compilation
	if strings.Contains(output, "compilation failed") ||
		strings.Contains(output, "build failed") ||
		strings.Contains(output, "error:") && strings.Contains(output, "cannot find symbol") {
		result.CompilationOK = false
		result.ErrorMessage = "Compilation failed"
		return result, nil
	}

	result.CompilationOK = true

	// Check if tests passed
	if success {
		result.TestsPassed = true
		result.Success = true
	} else {
		result.TestsPassed = false
		result.ErrorMessage = "Tests failed"
		result.FailedTests = v.extractFailedTests(output)
	}

	return result, nil
}

// ValidateAndRetry validates a test and retries if it fails
func (v *Validator) ValidateAndRetry(projectPath, testFile string, generator *Generator, maxRetries int) (*ValidationResult, error) {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := v.ValidateTest(projectPath, testFile)
		if err != nil {
			return nil, err
		}

		if result.Success {
			return result, nil
		}

		// If not the last attempt, try to fix
		if attempt < maxRetries {
			fmt.Printf("  Test validation failed (attempt %d/%d), attempting to fix...\n", attempt+1, maxRetries+1)

			// Try to fix the test
			_, err := generator.FixBrokenTest(projectPath, testFile, result.Output)
			if err != nil {
				return result, fmt.Errorf("failed to fix test: %w", err)
			}
		}
	}

	// Return last result if all retries failed
	result, err := v.ValidateTest(projectPath, testFile)
	return result, err
}

// extractFailedTests extracts names of failed tests from output
func (v *Validator) extractFailedTests(output string) []string {
	var failed []string

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Common patterns for failed tests
		if strings.Contains(line, "FAIL:") ||
			strings.Contains(line, "FAILED") ||
			strings.Contains(line, "✗") ||
			strings.Contains(line, "Failed:") {
			// Extract test name
			parts := strings.Fields(line)
			if len(parts) > 1 {
				failed = append(failed, parts[1])
			}
		}
	}

	return failed
}

// MeasureCoverageImprovement measures how much coverage improved after adding a test
func (v *Validator) MeasureCoverageImprovement(projectPath string, beforeCoverage, afterCoverage float64) float64 {
	improvement := afterCoverage - beforeCoverage
	if improvement < 0 {
		improvement = 0
	}
	return improvement
}

// IsTestFileValid performs a quick check if a test file looks valid
func (v *Validator) IsTestFileValid(testFile string, language string) bool {
	// Basic sanity checks based on language
	content, err := v.readFile(testFile)
	if err != nil {
		return false
	}

	switch language {
	case "Go":
		return strings.Contains(content, "func Test") &&
			strings.Contains(content, "package ") &&
			strings.Contains(content, "testing")

	case "Python":
		return (strings.Contains(content, "def test_") ||
			strings.Contains(content, "class Test")) &&
			(strings.Contains(content, "import pytest") ||
				strings.Contains(content, "import unittest"))

	case "TypeScript", "JavaScript":
		return (strings.Contains(content, "test(") ||
			strings.Contains(content, "it(") ||
			strings.Contains(content, "describe(")) &&
			(strings.Contains(content, "expect(") ||
				strings.Contains(content, "assert"))

	case "Java":
		return strings.Contains(content, "@Test") &&
			(strings.Contains(content, "import org.junit") ||
				strings.Contains(content, "import org.testng"))

	case "Swift":
		return strings.Contains(content, "XCTestCase") &&
			strings.Contains(content, "func test")

	default:
		return true // Assume valid if we don't know the language
	}
}

// readFile is a helper to read file contents
func (v *Validator) readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
