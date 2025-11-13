# Test Coverage Agent

An autonomous AI-powered tool that uses Claude API to automatically generate and fix tests for your projects until a target code coverage level is achieved.

## Features

- 🤖 **Autonomous Operation**: Runs without human intervention until target coverage is reached or manually stopped
- 🌍 **Multi-Language Support**: Go, Swift, Python, JavaScript/TypeScript, and Java
- 🔄 **Pause/Resume**: Handles API rate limits automatically and can resume from saved state
- 🧪 **Test Generation & Fixing**: Creates new test files and fixes broken existing tests
- ✅ **Test Validation**: Validates generated tests compile and pass before accepting them
- 📊 **Progress Tracking**: JSON-based state persistence with detailed progress reporting
- 🔒 **Git Integration**: Optional automatic commits for each successful test addition
- 🎯 **Smart Prioritization**: Focuses on files with lowest coverage first

## Installation

### Prerequisites

- Go 1.21 or later
- Claude API key (from Anthropic)
- Language-specific test tools installed:
  - **Go**: `go test` (built-in)
  - **Python**: `pytest`, `pytest-cov`
  - **JavaScript/TypeScript**: `jest` or test runner in `package.json`
  - **Java**: Maven or Gradle with JaCoCo plugin
  - **Swift**: Xcode or Swift Package Manager

### Build

```bash
cd test-coverage-agent
go mod tidy
go build -o test-coverage-agent
```

### Install globally (optional)

```bash
go install
```

## Usage

### Basic Usage

```bash
# Set your Claude API key
export ANTHROPIC_API_KEY="your-api-key-here"

# Run with default settings (80% coverage target)
./test-coverage-agent -project /path/to/your/project

# Specify custom target coverage
./test-coverage-agent -project /path/to/your/project -target 90.0

# Dry run to preview what would happen
./test-coverage-agent -project /path/to/your/project -dry-run
```

### Command Line Options

```
-project string
    Path to the project to analyze (default: current directory)

-target float
    Target code coverage percentage, 0-100 (default: 80.0)

-api-key string
    Claude API key (or set ANTHROPIC_API_KEY environment variable)

-config string
    Configuration file path (default: "coverage-agent.json")

-state string
    State file for pause/resume (default: ".coverage-agent-state.json")

-dry-run
    Preview actions without making changes (default: false)

-resume
    Resume from previous state (default: false)

-max-iterations int
    Maximum number of test generation iterations (default: 100)
```

### Resume After Rate Limit

When the tool hits API rate limits, it automatically saves state and waits. You can also manually stop it with `Ctrl+C` and resume later:

```bash
# Stop the agent (Ctrl+C)
# State is automatically saved to .coverage-agent-state.json

# Resume later
./test-coverage-agent -project /path/to/your/project -resume
```

## How It Works

1. **Language Detection**: Automatically detects the project language
2. **Coverage Analysis**: Runs language-specific coverage tools
3. **Prioritization**: Identifies files with lowest coverage
4. **Test Generation**: Uses Claude API to generate comprehensive tests
5. **Validation**: Compiles and runs tests to ensure they work
6. **Auto-Fix**: If tests fail, attempts to fix them automatically
7. **Git Commit**: Optionally commits successful tests
8. **Iteration**: Repeats until target coverage or max iterations reached

## State File Format

The state file (`.coverage-agent-state.json`) contains:

```json
{
  "current_iteration": 5,
  "current_coverage": 75.5,
  "target_coverage": 80.0,
  "processed_files": {
    "main.go": true,
    "handler.go": true
  },
  "generated_tests": [
    "main_test.go",
    "handler_test.go"
  ],
  "coverage_history": [
    {
      "timestamp": "2025-01-15T10:30:00Z",
      "coverage": 65.0,
      "iteration": 1
    }
  ],
  "language": "Go"
}
```

## Language-Specific Notes

### Go
- Uses `go test -coverprofile` for coverage
- Follows convention: `foo.go` → `foo_test.go`
- Requires `go.mod` in project root

### Python
- Uses `pytest --cov` for coverage
- Follows convention: `foo.py` → `test_foo.py`
- Requires `pytest` and `pytest-cov` installed

### JavaScript/TypeScript
- Uses Jest for testing and coverage
- Follows convention: `foo.ts` → `foo.test.ts`
- Requires `package.json` with test script

### Java
- Uses JaCoCo for coverage via Maven or Gradle
- Follows convention: `Foo.java` → `FooTest.java`
- Requires proper build configuration

### Swift
- Uses `swift test --enable-code-coverage`
- Follows convention: `Foo.swift` → `FooTests.swift`
- Requires `Package.swift` or Xcode project

## Rate Limiting

The tool handles Claude API rate limits automatically:

- Detects 429 (Too Many Requests) responses
- Saves current state
- Waits until rate limit reset time
- Resumes automatically
- Supports manual pause/resume with `Ctrl+C`

## Git Integration

If your project is a git repository, the tool will:

1. Create a new branch: `test-coverage-agent-YYYYMMDD-HHMMSS`
2. Commit each successful test addition
3. Include coverage gain in commit messages
4. Allow easy rollback if needed

Disable git integration by running outside a git repository.

## Examples

### Example 1: Go Project

```bash
export ANTHROPIC_API_KEY="sk-ant-..."
cd ~/my-go-project
test-coverage-agent -target 85
```

### Example 2: Python Project with Resume

```bash
# Initial run
test-coverage-agent -project ~/my-python-app -target 90

# Stopped by rate limit, resume later
test-coverage-agent -project ~/my-python-app -resume
```

### Example 3: Dry Run

```bash
# See what would happen without making changes
test-coverage-agent -project ~/my-project -dry-run -target 80
```

## Troubleshooting

### "Unable to detect project language"
- Ensure your project has language-specific files (e.g., `go.mod`, `package.json`)
- Check that you're pointing to the correct project directory

### "Failed to run coverage analysis"
- Verify language-specific tools are installed
- For Python: `pip install pytest pytest-cov`
- For JavaScript: `npm install` and check `package.json`
- For Java: Ensure JaCoCo plugin is configured

### "Test validation failed"
- Check the error output in the state file
- The tool attempts auto-fix, but some issues may need manual intervention
- Review generated test files for syntax or logic errors

### Rate limits hit frequently
- Consider using a higher tier API key
- Reduce `max-iterations` to process in smaller batches
- Use `resume` to continue in multiple sessions

## Best Practices

1. **Start with a clean git state**: Commit existing changes before running
2. **Use dry-run first**: Preview changes with `-dry-run`
3. **Set realistic targets**: 80-90% coverage is often more practical than 100%
4. **Review generated tests**: AI-generated tests should be reviewed for quality
5. **Incremental approach**: Run multiple sessions rather than one long session
6. **Keep dependencies updated**: Ensure test tools are current

## Architecture

```
test-coverage-agent/
├── main.go                  # CLI entry point
├── config/                  # Configuration and state management
│   └── config.go
├── coverage/                # Language-specific coverage analyzers
│   ├── analyzer.go          # Interface and common logic
│   ├── go.go               # Go analyzer
│   ├── python.go           # Python analyzer
│   ├── typescript.go       # TypeScript/JavaScript analyzer
│   ├── java.go             # Java analyzer
│   └── swift.go            # Swift analyzer
├── claude/                  # Claude API client
│   ├── client.go           # HTTP client with rate limiting
│   └── prompts.go          # Prompt templates
├── testgen/                 # Test generation and validation
│   ├── generator.go        # Test generation logic
│   └── validator.go        # Test validation logic
├── git/                     # Git integration
│   └── operations.go       # Git operations
└── orchestrator/            # Main orchestration logic
    └── orchestrator.go     # Workflow coordination
```

## Using in CI/CD (Any Project)

The test-coverage-agent works seamlessly in CI pipelines for **any Go project**, regardless of directory structure or location.

### GitHub Actions Setup

1. **Add the secret** (one-time per repository):
```bash
gh secret set ANTHROPIC_API_KEY
```

2. **Add workflow file** to `.github/workflows/test.yml`:

```yaml
name: Tests with Auto-Coverage

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Run tests
        run: |
          go test ./... -coverprofile=coverage.out -covermode=atomic
          go tool cover -func=coverage.out

      - name: Check coverage
        id: coverage
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "coverage=${COVERAGE}" >> $GITHUB_OUTPUT
          if (( $(echo "$COVERAGE < 40" | bc -l) )); then
            echo "needs_gen=true" >> $GITHUB_OUTPUT
          else
            echo "needs_gen=false" >> $GITHUB_OUTPUT
          fi

      - name: Auto-generate tests
        if: steps.coverage.outputs.needs_gen == 'true'
        run: |
          # Install tool
          git clone https://github.com/jucevic/test-coverage-agent.git /tmp/tca
          cd /tmp/tca && go build -o test-coverage-agent
          sudo mv test-coverage-agent /usr/local/bin/

          # Generate tests
          cd $GITHUB_WORKSPACE
          test-coverage-agent -project . -target 40 -max-iterations 5 \
            -api-key "${{ secrets.ANTHROPIC_API_KEY }}"

          # Commit if tests generated
          if ! git diff --quiet; then
            git config user.name "github-actions[bot]"
            git config user.email "github-actions[bot]@users.noreply.github.com"
            git add .
            git commit -m "chore: auto-generate tests for coverage"
            git push
          fi
```

### Works With Any Project Structure

The tool automatically detects and works with any Go project:

```bash
# Monorepo with multiple services
/project-root
  ├── services/
  │   ├── api/          # test-coverage-agent -project ./services/api
  │   ├── worker/       # test-coverage-agent -project ./services/worker
  │   └── admin/        # test-coverage-agent -project ./services/admin
  ├── libs/
  │   └── common/       # test-coverage-agent -project ./libs/common
  └── tools/

# Nested projects
/workspace
  ├── backend/          # test-coverage-agent -project ./backend
  ├── cli/              # test-coverage-agent -project ./cli
  └── sdk/              # test-coverage-agent -project ./sdk

# Any location on your system
test-coverage-agent -project ~/projects/my-app -target 50
test-coverage-agent -project /var/repos/service -target 60
```

### Automatic Threshold Feature

The tool includes smart threshold checking:

- **Coverage ≥ 40%**: Skips test generation (no API calls)
- **39% ≤ Coverage < 40%**: Skips (within 1% of target)
- **Coverage < 39%**: Automatically generates tests

This means you can safely run it in CI on every commit - it only acts when needed!

```bash
# Safe to run repeatedly - only generates when coverage drops
test-coverage-agent -project . -target 40

# Output when coverage is 39.5%:
# ✅ Coverage is within acceptable range
# No test generation needed at this time.

# Output when coverage is 35%:
# 🔧 Test generation needed (coverage below threshold)
```

## Contributing

Contributions are welcome! Areas for improvement:

- Additional language support (Rust, C++, etc.)
- Better test quality validation
- Improved error recovery
- UI/dashboard for progress monitoring
- More CI/CD platform examples (GitLab, CircleCI, etc.)

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Powered by [Claude](https://www.anthropic.com/claude) from Anthropic
- Inspired by the need for better automated testing tools
