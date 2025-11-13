package coverage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PythonAnalyzer implements coverage analysis for Python projects
type PythonAnalyzer struct{}

// DetectLanguage checks if this is a Python project
func (p *PythonAnalyzer) DetectLanguage(projectPath string) bool {
	// Check for common Python files
	indicators := []string{"setup.py", "pyproject.toml", "requirements.txt", "Pipfile"}
	for _, indicator := range indicators {
		if fileExists(filepath.Join(projectPath, indicator)) {
			return true
		}
	}

	// Check for .py files
	count := countFilesWithExtension(projectPath, []string{".py"})
	return count > 0
}

// GetLanguageName returns "Python"
func (p *PythonAnalyzer) GetLanguageName() string {
	return "Python"
}

// RunCoverage executes pytest with coverage
func (p *PythonAnalyzer) RunCoverage(projectPath string) (*CoverageReport, error) {
	// Try to use pytest-cov if available, fall back to coverage.py
	report := &CoverageReport{
		FileCoverage:   make(map[string]float64),
		UncoveredFiles: []string{},
		UncoveredLines: make(map[string][]int),
		Language:       "Python",
	}

	// Run pytest with coverage
	cmd := exec.Command("pytest", "--cov=.", "--cov-report=json", "--cov-report=term")
	cmd.Dir = projectPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run() // Ignore error, tests might fail but we can still get coverage

	// Parse JSON coverage report
	coverageFile := filepath.Join(projectPath, "coverage.json")
	defer os.Remove(coverageFile)

	if fileExists(coverageFile) {
		if err := p.parseCoverageJSON(coverageFile, report); err != nil {
			return nil, fmt.Errorf("failed to parse coverage: %w", err)
		}
	} else {
		// Try alternative: coverage run + coverage json
		cmd = exec.Command("coverage", "run", "-m", "pytest")
		cmd.Dir = projectPath
		cmd.Run()

		cmd = exec.Command("coverage", "json")
		cmd.Dir = projectPath
		if err := cmd.Run(); err == nil {
			if fileExists(coverageFile) {
				p.parseCoverageJSON(coverageFile, report)
			}
		}
	}

	return report, nil
}

// parseCoverageJSON parses Python coverage.json format
func (p *PythonAnalyzer) parseCoverageJSON(filename string, report *CoverageReport) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var coverage struct {
		Totals struct {
			PercentCovered float64 `json:"percent_covered"`
		} `json:"totals"`
		Files map[string]struct {
			Summary struct {
				PercentCovered float64 `json:"percent_covered"`
			} `json:"summary"`
			MissingLines []int `json:"missing_lines"`
		} `json:"files"`
	}

	if err := json.Unmarshal(data, &coverage); err != nil {
		return err
	}

	report.TotalCoverage = coverage.Totals.PercentCovered

	for filename, fileCov := range coverage.Files {
		report.FileCoverage[filename] = fileCov.Summary.PercentCovered
		if len(fileCov.MissingLines) > 0 {
			report.UncoveredFiles = append(report.UncoveredFiles, filename)
			report.UncoveredLines[filename] = fileCov.MissingLines
		}
	}

	return nil
}

// GetTestFilePath returns the test file path for a Python source file
func (p *PythonAnalyzer) GetTestFilePath(sourceFile string) string {
	// Python convention: foo.py -> test_foo.py or foo_test.py
	dir := filepath.Dir(sourceFile)
	base := filepath.Base(sourceFile)
	name := strings.TrimSuffix(base, ".py")

	// Prefer test_foo.py pattern
	return filepath.Join(dir, "test_"+name+".py")
}

// GetSourceFileForTest returns the source file for a Python test file
func (p *PythonAnalyzer) GetSourceFileForTest(testFile string) string {
	base := filepath.Base(testFile)
	dir := filepath.Dir(testFile)

	// Remove test_ prefix or _test suffix
	if strings.HasPrefix(base, "test_") {
		name := strings.TrimPrefix(base, "test_")
		return filepath.Join(dir, name)
	}
	if strings.HasSuffix(base, "_test.py") {
		name := strings.TrimSuffix(base, "_test.py") + ".py"
		return filepath.Join(dir, name)
	}

	return testFile
}

// RunTests runs tests for a specific test file
func (p *PythonAnalyzer) RunTests(projectPath string, testFile string) (bool, string, error) {
	cmd := exec.Command("pytest", "-v", testFile)
	cmd.Dir = projectPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	return err == nil, output, nil
}

// ValidateTestFile validates that a test file runs successfully
func (p *PythonAnalyzer) ValidateTestFile(projectPath string, testFile string) (bool, string, error) {
	// Python doesn't have a separate compile step, just run the tests
	return p.RunTests(projectPath, testFile)
}
