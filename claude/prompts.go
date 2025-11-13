package claude

import (
	"fmt"
	"strings"
)

// GenerateTestPrompt creates a prompt for generating tests for uncovered code
func GenerateTestPrompt(language, sourceFile, sourceCode, uncoveredLines string) string {
	return fmt.Sprintf(`You are an expert %s test engineer. I need you to write comprehensive unit tests for the following source code.

Language: %s
Source File: %s

SOURCE CODE:
%s

UNCOVERED LINES (need tests):
%s

Please generate complete, runnable unit tests that:
1. Cover all the uncovered lines mentioned above
2. Follow %s best practices and idioms
3. Include edge cases and error conditions
4. Are properly structured and well-documented
5. Use appropriate testing frameworks for %s

Provide ONLY the complete test file code, without any explanations or markdown formatting.
The test file should be ready to save and run immediately.`,
		language, language, sourceFile, sourceCode, uncoveredLines, language, language)
}

// FixBrokenTestPrompt creates a prompt for fixing broken tests
func FixBrokenTestPrompt(language, testFile, testCode, errorOutput string) string {
	return fmt.Sprintf(`You are an expert %s test engineer. The following test file is failing and needs to be fixed.

Language: %s
Test File: %s

CURRENT TEST CODE:
%s

TEST FAILURE OUTPUT:
%s

Please fix the test code so that:
1. All tests pass successfully
2. The tests still provide meaningful coverage
3. The fixes address the root cause, not just symptoms
4. The code follows %s best practices

Provide ONLY the complete fixed test file code, without any explanations or markdown formatting.
The test file should be ready to save and run immediately.`,
		language, language, testFile, testCode, errorOutput, language)
}

// AnalyzeUncoveredCodePrompt creates a prompt for understanding what tests are needed
func AnalyzeUncoveredCodePrompt(language, sourceFile, sourceCode, coverageReport string) string {
	return fmt.Sprintf(`You are an expert %s code coverage analyst. Please analyze the following source code and coverage report.

Language: %s
Source File: %s

SOURCE CODE:
%s

COVERAGE REPORT:
%s

Please provide a concise analysis in this format:
1. List the specific functions/methods that need test coverage
2. For each function, briefly describe what test scenarios are needed
3. Identify any edge cases or error conditions that should be tested

Keep your response focused and actionable. Format as a numbered list.`,
		language, language, sourceFile, sourceCode, coverageReport)
}

// ImproveTestCoveragePrompt creates a prompt for improving existing tests
func ImproveTestCoveragePrompt(language, sourceFile, sourceCode, existingTests, coverageGaps string) string {
	return fmt.Sprintf(`You are an expert %s test engineer. I have existing tests that need to be improved to cover more code.

Language: %s
Source File: %s

SOURCE CODE:
%s

EXISTING TESTS:
%s

COVERAGE GAPS (uncovered code):
%s

Please enhance the existing tests to:
1. Cover all the gaps mentioned above
2. Maintain all existing test functionality
3. Add new test cases for uncovered scenarios
4. Follow %s testing best practices

Provide ONLY the complete enhanced test file code, without any explanations or markdown formatting.`,
		language, language, sourceFile, sourceCode, existingTests, coverageGaps, language)
}

// ExtractCodeFromResponse attempts to extract code from Claude's response
// Claude sometimes adds markdown formatting, so we need to clean it up
func ExtractCodeFromResponse(response string) string {
	// Remove markdown code blocks
	response = strings.TrimSpace(response)

	// Check for markdown code fences
	if strings.HasPrefix(response, "```") {
		lines := strings.Split(response, "\n")
		if len(lines) > 2 {
			// Remove first line (```language) and last line (```)
			if strings.HasPrefix(lines[0], "```") {
				lines = lines[1:]
			}
			if strings.HasSuffix(lines[len(lines)-1], "```") {
				lines = lines[:len(lines)-1]
			}
			response = strings.Join(lines, "\n")
		}
	}

	return strings.TrimSpace(response)
}
