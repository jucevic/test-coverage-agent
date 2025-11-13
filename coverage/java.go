package coverage

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// JavaAnalyzer implements coverage analysis for Java projects
type JavaAnalyzer struct{}

// DetectLanguage checks if this is a Java project
func (j *JavaAnalyzer) DetectLanguage(projectPath string) bool {
	// Check for common Java build files
	indicators := []string{"pom.xml", "build.gradle", "build.gradle.kts"}
	for _, indicator := range indicators {
		if fileExists(filepath.Join(projectPath, indicator)) {
			return true
		}
	}

	// Check for .java files
	count := countFilesWithExtension(projectPath, []string{".java"})
	return count > 0
}

// GetLanguageName returns "Java"
func (j *JavaAnalyzer) GetLanguageName() string {
	return "Java"
}

// RunCoverage executes tests with JaCoCo coverage
func (j *JavaAnalyzer) RunCoverage(projectPath string) (*CoverageReport, error) {
	report := &CoverageReport{
		FileCoverage:   make(map[string]float64),
		UncoveredFiles: []string{},
		UncoveredLines: make(map[string][]int),
		Language:       "Java",
	}

	// Determine build tool
	isMaven := fileExists(filepath.Join(projectPath, "pom.xml"))
	isGradle := fileExists(filepath.Join(projectPath, "build.gradle")) ||
		fileExists(filepath.Join(projectPath, "build.gradle.kts"))

	var cmd *exec.Cmd

	if isMaven {
		// Run Maven with JaCoCo
		cmd = exec.Command("mvn", "clean", "test", "jacoco:report")
	} else if isGradle {
		// Run Gradle with JaCoCo
		cmd = exec.Command("./gradlew", "test", "jacocoTestReport")
		if !fileExists(filepath.Join(projectPath, "gradlew")) {
			cmd = exec.Command("gradle", "test", "jacocoTestReport")
		}
	} else {
		return nil, fmt.Errorf("no supported build tool found (Maven or Gradle)")
	}

	cmd.Dir = projectPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run() // Ignore error, tests might fail

	// Parse JaCoCo XML report
	var reportPath string
	if isMaven {
		reportPath = filepath.Join(projectPath, "target", "site", "jacoco", "jacoco.xml")
	} else {
		reportPath = filepath.Join(projectPath, "build", "reports", "jacoco", "test", "jacocoTestReport.xml")
	}

	if fileExists(reportPath) {
		if err := j.parseJaCoCoXML(reportPath, report); err != nil {
			return nil, fmt.Errorf("failed to parse JaCoCo report: %w", err)
		}
	}

	return report, nil
}

// parseJaCoCoXML parses JaCoCo XML coverage report
func (j *JavaAnalyzer) parseJaCoCoXML(filename string, report *CoverageReport) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	type Counter struct {
		Type    string `xml:"type,attr"`
		Missed  int    `xml:"missed,attr"`
		Covered int    `xml:"covered,attr"`
	}

	type SourceFile struct {
		Name     string    `xml:"name,attr"`
		Counters []Counter `xml:"counter"`
		Lines    []struct {
			Number int `xml:"nr,attr"`
			Hits   int `xml:"ci,attr"`
		} `xml:"line"`
	}

	type Package struct {
		Name        string       `xml:"name,attr"`
		SourceFiles []SourceFile `xml:"sourcefile"`
	}

	var jacocoReport struct {
		Packages []Package `xml:"package"`
		Counters []Counter `xml:"counter"`
	}

	if err := xml.Unmarshal(data, &jacocoReport); err != nil {
		return err
	}

	// Calculate total coverage from counters
	for _, counter := range jacocoReport.Counters {
		if counter.Type == "LINE" {
			total := counter.Covered + counter.Missed
			if total > 0 {
				report.TotalCoverage = (float64(counter.Covered) / float64(total)) * 100
			}
		}
	}

	// Parse per-file coverage
	for _, pkg := range jacocoReport.Packages {
		for _, sourceFile := range pkg.SourceFiles {
			fullPath := filepath.Join(pkg.Name, sourceFile.Name)

			// Calculate file coverage
			var covered, total int
			var uncovered []int

			for _, line := range sourceFile.Lines {
				total++
				if line.Hits > 0 {
					covered++
				} else {
					uncovered = append(uncovered, line.Number)
				}
			}

			if total > 0 {
				fileCoverage := (float64(covered) / float64(total)) * 100
				report.FileCoverage[fullPath] = fileCoverage

				if len(uncovered) > 0 {
					report.UncoveredFiles = append(report.UncoveredFiles, fullPath)
					report.UncoveredLines[fullPath] = uncovered
				}
			}
		}
	}

	return nil
}

// GetTestFilePath returns the test file path for a Java source file
func (j *JavaAnalyzer) GetTestFilePath(sourceFile string) string {
	// Java convention: Foo.java in src/main/java -> FooTest.java in src/test/java
	if strings.Contains(sourceFile, "/main/") {
		testFile := strings.Replace(sourceFile, "/main/", "/test/", 1)
		base := filepath.Base(testFile)
		name := strings.TrimSuffix(base, ".java")
		dir := filepath.Dir(testFile)
		return filepath.Join(dir, name+"Test.java")
	}

	// Simple case: Foo.java -> FooTest.java
	base := filepath.Base(sourceFile)
	name := strings.TrimSuffix(base, ".java")
	dir := filepath.Dir(sourceFile)
	return filepath.Join(dir, name+"Test.java")
}

// GetSourceFileForTest returns the source file for a Java test file
func (j *JavaAnalyzer) GetSourceFileForTest(testFile string) string {
	// Remove Test suffix and swap test/main directories
	if strings.Contains(testFile, "/test/") {
		sourceFile := strings.Replace(testFile, "/test/", "/main/", 1)
		base := filepath.Base(sourceFile)
		if strings.HasSuffix(base, "Test.java") {
			name := strings.TrimSuffix(base, "Test.java") + ".java"
			return filepath.Join(filepath.Dir(sourceFile), name)
		}
		return sourceFile
	}

	base := filepath.Base(testFile)
	if strings.HasSuffix(base, "Test.java") {
		name := strings.TrimSuffix(base, "Test.java") + ".java"
		return filepath.Join(filepath.Dir(testFile), name)
	}

	return testFile
}

// RunTests runs tests for a specific test file
func (j *JavaAnalyzer) RunTests(projectPath string, testFile string) (bool, string, error) {
	// Determine build tool
	isMaven := fileExists(filepath.Join(projectPath, "pom.xml"))

	var cmd *exec.Cmd

	if isMaven {
		// Extract test class name
		className := j.getClassName(testFile)
		cmd = exec.Command("mvn", "test", "-Dtest="+className)
	} else {
		// Gradle
		className := j.getClassName(testFile)
		cmd = exec.Command("./gradlew", "test", "--tests", className)
		if !fileExists(filepath.Join(projectPath, "gradlew")) {
			cmd = exec.Command("gradle", "test", "--tests", className)
		}
	}

	cmd.Dir = projectPath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	return err == nil, output, nil
}

// getClassName extracts the fully qualified class name from a file path
func (j *JavaAnalyzer) getClassName(testFile string) string {
	// Extract package and class name from file path
	// Example: src/test/java/com/example/FooTest.java -> com.example.FooTest

	parts := strings.Split(testFile, "/")
	var packageParts []string
	inPackage := false

	for _, part := range parts {
		if part == "java" {
			inPackage = true
			continue
		}
		if inPackage {
			if strings.HasSuffix(part, ".java") {
				part = strings.TrimSuffix(part, ".java")
			}
			packageParts = append(packageParts, part)
		}
	}

	return strings.Join(packageParts, ".")
}

// ValidateTestFile validates that a test file compiles and runs
func (j *JavaAnalyzer) ValidateTestFile(projectPath string, testFile string) (bool, string, error) {
	// Java requires compilation before running, which is handled by the build tool
	return j.RunTests(projectPath, testFile)
}
