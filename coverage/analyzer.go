package coverage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CoverageReport represents coverage information for a project
type CoverageReport struct {
	TotalCoverage  float64               `json:"total_coverage"`
	FileCoverage   map[string]float64    `json:"file_coverage"`
	UncoveredFiles []string              `json:"uncovered_files"`
	UncoveredLines map[string][]int      `json:"uncovered_lines"`
	Language       string                `json:"language"`
}

// Analyzer defines the interface for language-specific coverage analyzers
type Analyzer interface {
	// DetectLanguage checks if this analyzer can handle the project
	DetectLanguage(projectPath string) bool

	// GetLanguageName returns the name of the language
	GetLanguageName() string

	// RunCoverage executes coverage analysis and returns a report
	RunCoverage(projectPath string) (*CoverageReport, error)

	// GetTestFilePath returns the conventional test file path for a source file
	GetTestFilePath(sourceFile string) string

	// GetSourceFileForTest returns the source file path for a test file
	GetSourceFileForTest(testFile string) string

	// RunTests executes tests and returns success/failure and output
	RunTests(projectPath string, testFile string) (bool, string, error)

	// ValidateTestFile checks if a test file is valid (compiles, runs)
	ValidateTestFile(projectPath string, testFile string) (bool, string, error)
}

// DetectProjectLanguage determines the primary language of a project
func DetectProjectLanguage(projectPath string) (Analyzer, error) {
	analyzers := []Analyzer{
		&GoAnalyzer{},
		&SwiftAnalyzer{},
		&PythonAnalyzer{},
		&TypeScriptAnalyzer{},
		&JavaAnalyzer{},
	}

	for _, analyzer := range analyzers {
		if analyzer.DetectLanguage(projectPath) {
			return analyzer, nil
		}
	}

	return nil, fmt.Errorf("unable to detect project language in %s", projectPath)
}

// Helper function to check if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Helper function to find files with specific extensions
func findFilesWithExtension(projectPath string, extensions []string) ([]string, error) {
	var files []string

	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Skip common directories we don't want to search
			dirName := info.Name()
			if dirName == "node_modules" || dirName == "vendor" || dirName == ".git" ||
				dirName == "build" || dirName == "dist" || dirName == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}

		for _, ext := range extensions {
			if strings.HasSuffix(path, ext) {
				files = append(files, path)
				break
			}
		}

		return nil
	})

	return files, err
}

// Helper function to count files with specific extensions
func countFilesWithExtension(projectPath string, extensions []string) int {
	files, err := findFilesWithExtension(projectPath, extensions)
	if err != nil {
		return 0
	}
	return len(files)
}

// ParseCoveragePercentage extracts percentage from various formats
func ParseCoveragePercentage(input string) float64 {
	// Try to find percentage in format: XX.X% or XX%
	var percentage float64

	// For "go tool cover -func" output: "total:\t\t\t(statements)\t\t32.2%"
	// Extract the last field that contains a percentage
	fields := strings.Fields(input)
	for i := len(fields) - 1; i >= 0; i-- {
		field := fields[i]
		if strings.HasSuffix(field, "%") {
			// Remove the % and parse
			numStr := strings.TrimSuffix(field, "%")
			if val, err := strconv.ParseFloat(numStr, 64); err == nil {
				return val
			}
		}
	}

	// Common patterns: "coverage: 75.5%", "75.5%", "Total: 75.5%"
	if n, _ := fmt.Sscanf(input, "%f%%", &percentage); n == 1 {
		return percentage
	}
	if n, _ := fmt.Sscanf(input, "coverage: %f%%", &percentage); n == 1 {
		return percentage
	}
	if n, _ := fmt.Sscanf(input, "Total: %f%%", &percentage); n == 1 {
		return percentage
	}

	return 0.0
}
