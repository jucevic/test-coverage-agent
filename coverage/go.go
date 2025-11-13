package coverage

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// GoAnalyzer implements coverage analysis for Go projects
type GoAnalyzer struct{}

// DetectLanguage checks if this is a Go project
func (g *GoAnalyzer) DetectLanguage(projectPath string) bool {
	// Check for go.mod file
	if fileExists(filepath.Join(projectPath, "go.mod")) {
		return true
	}

	// Check for .go files
	count := countFilesWithExtension(projectPath, []string{".go"})
	return count > 0
}

// GetLanguageName returns "Go"
func (g *GoAnalyzer) GetLanguageName() string {
	return "Go"
}

// RunCoverage executes go test with coverage
func (g *GoAnalyzer) RunCoverage(projectPath string) (*CoverageReport, error) {
	coverageFile := filepath.Join(projectPath, "coverage.out")

	// Run tests with coverage (use atomic for consistency with CI)
	cmd := exec.Command("go", "test", "./...", "-coverprofile="+coverageFile, "-covermode=atomic")
	cmd.Dir = projectPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Tests might fail, but we can still get coverage info if the file exists
		// Check if coverage file was generated despite test failures
		if !fileExists(coverageFile) {
			return nil, fmt.Errorf("tests failed and no coverage file generated: %w\nStderr: %s", err, stderr.String())
		}
	}

	// Parse coverage file
	report := &CoverageReport{
		FileCoverage:   make(map[string]float64),
		UncoveredFiles: []string{},
		UncoveredLines: make(map[string][]int),
		Language:       "Go",
	}

	// Read coverage file
	if fileExists(coverageFile) {
		if err := g.parseCoverageFile(coverageFile, report); err != nil {
			return nil, fmt.Errorf("failed to parse coverage: %w", err)
		}
	}

	// Get total coverage using go tool cover
	if fileExists(coverageFile) {
		cmd = exec.Command("go", "tool", "cover", "-func="+coverageFile)
		cmd.Dir = projectPath

		output, err := cmd.Output()
		if err == nil {
			// Parse total from last line: "total:  (statements)  XX.X%"
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "total:") {
					report.TotalCoverage = ParseCoveragePercentage(line)
					break
				}
			}
		}
	}

	return report, nil
}

// parseCoverageFile parses a Go coverage file
func (g *GoAnalyzer) parseCoverageFile(filename string, report *CoverageReport) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	fileStats := make(map[string]*fileCoverageStats)

	// Skip first line (mode: ...)
	for _, line := range lines[1:] {
		if line == "" {
			continue
		}

		// Format: filename:startLine.startCol,endLine.endCol numStmt count
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}

		// Extract filename and line range
		fileAndRange := parts[0]
		colonIdx := strings.LastIndex(fileAndRange, ":")
		if colonIdx == -1 {
			continue
		}

		filePath := fileAndRange[:colonIdx]

		// Convert module path to relative path
		// Coverage output has paths like: github.com/tablev/hls5/internal/handlers/file.go
		// We need to strip the module prefix and get: internal/handlers/file.go
		if strings.Contains(filePath, "/") {
			parts := strings.Split(filePath, "/")
			// Find where the actual project path starts (after module name)
			// Typically after the 3rd component (github.com/user/repo)
			if len(parts) > 3 {
				filePath = strings.Join(parts[3:], "/")
			}
		}
		lineRange := fileAndRange[colonIdx+1:]

		// Parse count (last field)
		count, _ := strconv.Atoi(parts[2])

		// Parse line numbers
		rangeParts := strings.Split(lineRange, ",")
		if len(rangeParts) != 2 {
			continue
		}

		startParts := strings.Split(rangeParts[0], ".")
		endParts := strings.Split(rangeParts[1], ".")
		if len(startParts) < 1 || len(endParts) < 1 {
			continue
		}

		startLine, _ := strconv.Atoi(startParts[0])
		endLine, _ := strconv.Atoi(endParts[0])

		// Initialize stats for file if needed
		if _, exists := fileStats[filePath]; !exists {
			fileStats[filePath] = &fileCoverageStats{
				covered:   0,
				total:     0,
				uncovered: []int{},
			}
		}

		stats := fileStats[filePath]

		// Count lines
		for line := startLine; line <= endLine; line++ {
			stats.total++
			if count > 0 {
				stats.covered++
			} else {
				stats.uncovered = append(stats.uncovered, line)
			}
		}
	}

	// Convert stats to report
	for filename, stats := range fileStats {
		if stats.total > 0 {
			coverage := (float64(stats.covered) / float64(stats.total)) * 100
			report.FileCoverage[filename] = coverage

			if coverage < 100 {
				report.UncoveredFiles = append(report.UncoveredFiles, filename)
				report.UncoveredLines[filename] = stats.uncovered
			}
		}
	}

	return nil
}

type fileCoverageStats struct {
	covered   int
	total     int
	uncovered []int
}

// GetTestFilePath returns the test file path for a Go source file
func (g *GoAnalyzer) GetTestFilePath(sourceFile string) string {
	// Go convention: foo.go -> foo_test.go
	ext := filepath.Ext(sourceFile)
	if ext == ".go" {
		base := strings.TrimSuffix(sourceFile, ext)
		return base + "_test.go"
	}
	return sourceFile + "_test.go"
}

// GetSourceFileForTest returns the source file for a Go test file
func (g *GoAnalyzer) GetSourceFileForTest(testFile string) string {
	// Remove _test.go suffix
	if strings.HasSuffix(testFile, "_test.go") {
		return strings.TrimSuffix(testFile, "_test.go") + ".go"
	}
	return testFile
}

// RunTests runs tests for a specific test file
func (g *GoAnalyzer) RunTests(projectPath string, testFile string) (bool, string, error) {
	// Get the package directory
	testDir := filepath.Dir(testFile)

	cmd := exec.Command("go", "test", "-v", "./"+testDir)
	cmd.Dir = projectPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	return err == nil, output, nil
}

// ValidateTestFile validates that a test file compiles and runs
func (g *GoAnalyzer) ValidateTestFile(projectPath string, testFile string) (bool, string, error) {
	// First, try to build
	testDir := filepath.Dir(testFile)
	cmd := exec.Command("go", "build", "./"+testDir)
	cmd.Dir = projectPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return false, "Compilation failed: " + stderr.String(), nil
	}

	// Then run tests
	return g.RunTests(projectPath, testFile)
}
