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

// TypeScriptAnalyzer implements coverage analysis for TypeScript/JavaScript projects
type TypeScriptAnalyzer struct{}

// DetectLanguage checks if this is a TypeScript/JavaScript project
func (t *TypeScriptAnalyzer) DetectLanguage(projectPath string) bool {
	// Check for package.json
	if fileExists(filepath.Join(projectPath, "package.json")) {
		return true
	}

	// Check for tsconfig.json
	if fileExists(filepath.Join(projectPath, "tsconfig.json")) {
		return true
	}

	// Check for .ts or .js files
	count := countFilesWithExtension(projectPath, []string{".ts", ".tsx", ".js", ".jsx"})
	return count > 0
}

// GetLanguageName returns "TypeScript"
func (t *TypeScriptAnalyzer) GetLanguageName() string {
	return "TypeScript"
}

// RunCoverage executes Jest with coverage
func (t *TypeScriptAnalyzer) RunCoverage(projectPath string) (*CoverageReport, error) {
	report := &CoverageReport{
		FileCoverage:   make(map[string]float64),
		UncoveredFiles: []string{},
		UncoveredLines: make(map[string][]int),
		Language:       "TypeScript",
	}

	// Run Jest with coverage
	cmd := exec.Command("npm", "test", "--", "--coverage", "--coverageReporters=json", "--coverageReporters=text")
	cmd.Dir = projectPath

	// Check if using yarn
	if fileExists(filepath.Join(projectPath, "yarn.lock")) {
		cmd = exec.Command("yarn", "test", "--coverage", "--coverageReporters=json", "--coverageReporters=text")
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run() // Ignore error, tests might fail

	// Parse coverage-final.json
	coverageFile := filepath.Join(projectPath, "coverage", "coverage-final.json")
	defer os.RemoveAll(filepath.Join(projectPath, "coverage")) // Cleanup

	if fileExists(coverageFile) {
		if err := t.parseCoverageJSON(coverageFile, report); err != nil {
			return nil, fmt.Errorf("failed to parse coverage: %w", err)
		}
	}

	return report, nil
}

// parseCoverageJSON parses Jest coverage-final.json format
func (t *TypeScriptAnalyzer) parseCoverageJSON(filename string, report *CoverageReport) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var coverage map[string]struct {
		Lines struct {
			Total   int            `json:"total"`
			Covered int            `json:"covered"`
			Pct     float64        `json:"pct"`
			Details map[string]int `json:"details"` // line number -> hits
		} `json:"lines"`
		Statements struct {
			Total   int     `json:"total"`
			Covered int     `json:"covered"`
			Pct     float64 `json:"pct"`
		} `json:"statements"`
	}

	if err := json.Unmarshal(data, &coverage); err != nil {
		return err
	}

	var totalLines, totalCovered int

	for filename, fileCov := range coverage {
		// Skip node_modules
		if strings.Contains(filename, "node_modules") {
			continue
		}

		report.FileCoverage[filename] = fileCov.Lines.Pct

		// Find uncovered lines
		var uncovered []int
		for lineStr, hits := range fileCov.Lines.Details {
			if hits == 0 {
				var lineNum int
				fmt.Sscanf(lineStr, "%d", &lineNum)
				uncovered = append(uncovered, lineNum)
			}
		}

		if len(uncovered) > 0 {
			report.UncoveredFiles = append(report.UncoveredFiles, filename)
			report.UncoveredLines[filename] = uncovered
		}

		totalLines += fileCov.Lines.Total
		totalCovered += fileCov.Lines.Covered
	}

	if totalLines > 0 {
		report.TotalCoverage = (float64(totalCovered) / float64(totalLines)) * 100
	}

	return nil
}

// GetTestFilePath returns the test file path for a TypeScript source file
func (t *TypeScriptAnalyzer) GetTestFilePath(sourceFile string) string {
	// Common patterns: foo.ts -> foo.test.ts or foo.spec.ts
	ext := filepath.Ext(sourceFile)
	base := strings.TrimSuffix(sourceFile, ext)

	// Prefer .test.ts pattern
	return base + ".test" + ext
}

// GetSourceFileForTest returns the source file for a TypeScript test file
func (t *TypeScriptAnalyzer) GetSourceFileForTest(testFile string) string {
	// Remove .test or .spec from filename
	if strings.Contains(testFile, ".test.") {
		return strings.Replace(testFile, ".test.", ".", 1)
	}
	if strings.Contains(testFile, ".spec.") {
		return strings.Replace(testFile, ".spec.", ".", 1)
	}
	return testFile
}

// RunTests runs tests for a specific test file
func (t *TypeScriptAnalyzer) RunTests(projectPath string, testFile string) (bool, string, error) {
	// Determine package manager
	cmd := exec.Command("npm", "test", "--", testFile)
	if fileExists(filepath.Join(projectPath, "yarn.lock")) {
		cmd = exec.Command("yarn", "test", testFile)
	}

	cmd.Dir = projectPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	return err == nil, output, nil
}

// ValidateTestFile validates that a test file runs successfully
func (t *TypeScriptAnalyzer) ValidateTestFile(projectPath string, testFile string) (bool, string, error) {
	return t.RunTests(projectPath, testFile)
}
