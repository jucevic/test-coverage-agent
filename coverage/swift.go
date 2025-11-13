package coverage

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// SwiftAnalyzer implements coverage analysis for Swift projects
type SwiftAnalyzer struct{}

// DetectLanguage checks if this is a Swift project
func (s *SwiftAnalyzer) DetectLanguage(projectPath string) bool {
	// Check for Package.swift (SPM)
	if fileExists(filepath.Join(projectPath, "Package.swift")) {
		return true
	}

	// Check for .xcodeproj or .xcworkspace
	files, _ := os.ReadDir(projectPath)
	for _, file := range files {
		name := file.Name()
		if strings.HasSuffix(name, ".xcodeproj") || strings.HasSuffix(name, ".xcworkspace") {
			return true
		}
	}

	// Check for .swift files
	count := countFilesWithExtension(projectPath, []string{".swift"})
	return count > 0
}

// GetLanguageName returns "Swift"
func (s *SwiftAnalyzer) GetLanguageName() string {
	return "Swift"
}

// RunCoverage executes swift test with coverage
func (s *SwiftAnalyzer) RunCoverage(projectPath string) (*CoverageReport, error) {
	report := &CoverageReport{
		FileCoverage:   make(map[string]float64),
		UncoveredFiles: []string{},
		UncoveredLines: make(map[string][]int),
		Language:       "Swift",
	}

	// Run swift test with coverage enabled
	cmd := exec.Command("swift", "test", "--enable-code-coverage")
	cmd.Dir = projectPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run() // Ignore error, tests might fail

	// Export coverage data
	cmd = exec.Command("xcrun", "llvm-cov", "export",
		".build/debug/<PackageName>PackageTests.xctest/Contents/MacOS/<PackageName>PackageTests",
		"-instr-profile=.build/debug/codecov/default.profdata",
		"-format=text")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		// Try alternative approach for SPM
		return s.runCoverageAlternative(projectPath, report)
	}

	// Parse coverage output
	if err := s.parseCoverageOutput(string(output), report); err != nil {
		return nil, err
	}

	return report, nil
}

// runCoverageAlternative tries an alternative coverage method
func (s *SwiftAnalyzer) runCoverageAlternative(projectPath string, report *CoverageReport) (*CoverageReport, error) {
	// For Xcode projects, try xcodebuild
	cmd := exec.Command("xcodebuild", "test",
		"-scheme", "YourScheme",
		"-enableCodeCoverage", "YES")
	cmd.Dir = projectPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run()

	// Parse xcodebuild output for coverage info
	output := stdout.String()
	s.parseCoverageOutput(output, report)

	return report, nil
}

// parseCoverageOutput parses coverage information from output
func (s *SwiftAnalyzer) parseCoverageOutput(output string, report *CoverageReport) error {
	// Parse coverage percentage
	re := regexp.MustCompile(`(\d+\.?\d*)%\s+coverage`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		fmt.Sscanf(matches[1], "%f", &report.TotalCoverage)
	}

	return nil
}

// GetTestFilePath returns the test file path for a Swift source file
func (s *SwiftAnalyzer) GetTestFilePath(sourceFile string) string {
	// Swift convention: Foo.swift -> FooTests.swift in Tests directory
	dir := filepath.Dir(sourceFile)
	base := filepath.Base(sourceFile)
	name := strings.TrimSuffix(base, ".swift")

	// Check if there's a Tests directory
	testsDir := filepath.Join(filepath.Dir(dir), "Tests")
	if !fileExists(testsDir) {
		testsDir = filepath.Join(dir, "Tests")
	}

	return filepath.Join(testsDir, name+"Tests.swift")
}

// GetSourceFileForTest returns the source file for a Swift test file
func (s *SwiftAnalyzer) GetSourceFileForTest(testFile string) string {
	base := filepath.Base(testFile)
	name := strings.TrimSuffix(base, "Tests.swift") + ".swift"

	// Look in Sources directory
	projectRoot := filepath.Dir(filepath.Dir(testFile))
	sourcesDir := filepath.Join(projectRoot, "Sources")

	return filepath.Join(sourcesDir, name)
}

// RunTests runs tests for a specific test file
func (s *SwiftAnalyzer) RunTests(projectPath string, testFile string) (bool, string, error) {
	// For SPM projects
	cmd := exec.Command("swift", "test")
	cmd.Dir = projectPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	return err == nil, output, nil
}

// ValidateTestFile validates that a test file compiles and runs
func (s *SwiftAnalyzer) ValidateTestFile(projectPath string, testFile string) (bool, string, error) {
	// Try to build first
	cmd := exec.Command("swift", "build", "--build-tests")
	cmd.Dir = projectPath

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return false, "Compilation failed: " + stderr.String(), nil
	}

	return s.RunTests(projectPath, testFile)
}
