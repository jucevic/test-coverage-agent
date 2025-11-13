package testgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tablev/test-coverage-agent/claude"
	"github.com/tablev/test-coverage-agent/coverage"
)

// Generator handles test generation using Claude API
type Generator struct {
	claudeClient *claude.Client
	analyzer     coverage.Analyzer
}

// NewGenerator creates a new test generator
func NewGenerator(apiKey string, analyzer coverage.Analyzer) *Generator {
	return &Generator{
		claudeClient: claude.NewClient(apiKey),
		analyzer:     analyzer,
	}
}

// GenerateTestForFile generates a test file for an uncovered source file
func (g *Generator) GenerateTestForFile(projectPath, sourceFile string, uncoveredLines []int) (string, error) {
	// Read source file
	sourceCode, err := os.ReadFile(sourceFile)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %w", err)
	}

	// Format uncovered lines
	uncoveredLinesStr := g.formatUncoveredLines(uncoveredLines)

	// Generate prompt
	language := g.analyzer.GetLanguageName()
	relativeSourceFile, _ := filepath.Rel(projectPath, sourceFile)
	prompt := claude.GenerateTestPrompt(language, relativeSourceFile, string(sourceCode), uncoveredLinesStr)

	// Call Claude API
	response, err := g.claudeClient.SendMessage(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate test: %w", err)
	}

	// Extract code from response
	testCode := claude.ExtractCodeFromResponse(response)

	// Get test file path
	testFilePath := g.analyzer.GetTestFilePath(sourceFile)

	// Ensure directory exists
	testDir := filepath.Dir(testFilePath)
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create test directory: %w", err)
	}

	// Write test file
	if err := os.WriteFile(testFilePath, []byte(testCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write test file: %w", err)
	}

	return testFilePath, nil
}

// FixBrokenTest attempts to fix a failing test file
func (g *Generator) FixBrokenTest(projectPath, testFile, errorOutput string) (string, error) {
	// Read test file
	testCode, err := os.ReadFile(testFile)
	if err != nil {
		return "", fmt.Errorf("failed to read test file: %w", err)
	}

	// Generate prompt
	language := g.analyzer.GetLanguageName()
	relativeTestFile, _ := filepath.Rel(projectPath, testFile)
	prompt := claude.FixBrokenTestPrompt(language, relativeTestFile, string(testCode), errorOutput)

	// Call Claude API
	response, err := g.claudeClient.SendMessage(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to fix test: %w", err)
	}

	// Extract code from response
	fixedTestCode := claude.ExtractCodeFromResponse(response)

	// Write fixed test file
	if err := os.WriteFile(testFile, []byte(fixedTestCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write fixed test file: %w", err)
	}

	return testFile, nil
}

// ImproveExistingTest enhances an existing test to cover more code
func (g *Generator) ImproveExistingTest(projectPath, sourceFile, testFile string, uncoveredLines []int) (string, error) {
	// Read source and test files
	sourceCode, err := os.ReadFile(sourceFile)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %w", err)
	}

	existingTests, err := os.ReadFile(testFile)
	if err != nil {
		return "", fmt.Errorf("failed to read test file: %w", err)
	}

	// Format uncovered lines
	uncoveredLinesStr := g.formatUncoveredLines(uncoveredLines)

	// Generate prompt
	language := g.analyzer.GetLanguageName()
	relativeSourceFile, _ := filepath.Rel(projectPath, sourceFile)
	prompt := claude.ImproveTestCoveragePrompt(
		language,
		relativeSourceFile,
		string(sourceCode),
		string(existingTests),
		uncoveredLinesStr,
	)

	// Call Claude API
	response, err := g.claudeClient.SendMessage(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to improve test: %w", err)
	}

	// Extract code from response
	improvedTestCode := claude.ExtractCodeFromResponse(response)

	// Write improved test file
	if err := os.WriteFile(testFile, []byte(improvedTestCode), 0644); err != nil {
		return "", fmt.Errorf("failed to write improved test file: %w", err)
	}

	return testFile, nil
}

// formatUncoveredLines formats line numbers for the prompt
func (g *Generator) formatUncoveredLines(lines []int) string {
	if len(lines) == 0 {
		return "None"
	}

	// Group consecutive lines into ranges
	var ranges []string
	start := lines[0]
	end := lines[0]

	for i := 1; i < len(lines); i++ {
		if lines[i] == end+1 {
			end = lines[i]
		} else {
			if start == end {
				ranges = append(ranges, fmt.Sprintf("Line %d", start))
			} else {
				ranges = append(ranges, fmt.Sprintf("Lines %d-%d", start, end))
			}
			start = lines[i]
			end = lines[i]
		}
	}

	// Add last range
	if start == end {
		ranges = append(ranges, fmt.Sprintf("Line %d", start))
	} else {
		ranges = append(ranges, fmt.Sprintf("Lines %d-%d", start, end))
	}

	return strings.Join(ranges, ", ")
}
