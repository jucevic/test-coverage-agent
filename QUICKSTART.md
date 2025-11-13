# Quick Start Guide

Get up and running with Test Coverage Agent in 5 minutes!

## Step 1: Install

```bash
cd test-coverage-agent
go build -o test-coverage-agent
```

Or install globally:

```bash
go install
```

## Step 2: Set API Key

Get your Claude API key from [Anthropic Console](https://console.anthropic.com/) and set it:

```bash
export ANTHROPIC_API_KEY="sk-ant-your-key-here"
```

## Step 3: Run on Your Project

### For Go Projects

```bash
./test-coverage-agent -project /path/to/your/go/project -target 85
```

### For Python Projects

Make sure pytest and pytest-cov are installed:

```bash
pip install pytest pytest-cov
./test-coverage-agent -project /path/to/your/python/project -target 80
```

### For JavaScript/TypeScript Projects

Ensure Jest is configured in your `package.json`:

```bash
./test-coverage-agent -project /path/to/your/js/project -target 80
```

### For Java Projects

Ensure JaCoCo is configured in `pom.xml` or `build.gradle`:

```bash
./test-coverage-agent -project /path/to/your/java/project -target 75
```

### For Swift Projects

```bash
./test-coverage-agent -project /path/to/your/swift/project -target 80
```

## Step 4: Monitor Progress

The tool will output progress as it runs:

```
Starting Test Coverage Agent
Project: /Users/me/my-project
Target Coverage: 85.00%
Max Iterations: 100
Press Ctrl+C to pause and save state
=====================================

Detected language: Go

=== Iteration 1 ===
Current Coverage: 65.50% / Target: 85.00%

Processing: handlers/user.go (current coverage: 45.00%)
  Generating new test file...
  Validating test...
  ✅ Test validation successful
  Committing to git...

Iteration: 1 | Coverage: 67.20% / 85.00% | Generated: 1 | Fixed: 0 | Failed: 0
```

## Step 5: Handle Rate Limits

If you hit API rate limits:

1. The tool will automatically wait and resume
2. Or you can stop with `Ctrl+C` and resume later:

```bash
./test-coverage-agent -project /path/to/your/project -resume
```

## Step 6: Review Generated Tests

Generated test files will be created following your language's conventions:

- **Go**: `foo.go` → `foo_test.go`
- **Python**: `foo.py` → `test_foo.py`
- **TypeScript**: `foo.ts` → `foo.test.ts`
- **Java**: `Foo.java` → `FooTest.java`
- **Swift**: `Foo.swift` → `FooTests.swift`

## Common Options

### Dry Run (Preview Mode)

See what would happen without making changes:

```bash
./test-coverage-agent -project . -dry-run
```

### Custom Target Coverage

```bash
./test-coverage-agent -project . -target 90
```

### Limit Iterations

```bash
./test-coverage-agent -project . -max-iterations 20
```

### Resume Previous Session

```bash
./test-coverage-agent -project . -resume
```

## Tips for Success

1. **Start Small**: Begin with a lower target (70-80%) for your first run
2. **Use Git**: Run in a git repository for automatic commits and easy rollback
3. **Review Tests**: AI-generated tests should be reviewed for quality
4. **Incremental**: Run multiple sessions rather than one very long session
5. **Clean State**: Ensure existing tests pass before running the agent

## What Happens Behind the Scenes

1. **Detection**: Identifies your project language
2. **Analysis**: Runs coverage tools to find gaps
3. **Prioritization**: Focuses on files with lowest coverage
4. **Generation**: Uses Claude to generate comprehensive tests
5. **Validation**: Ensures tests compile and pass
6. **Iteration**: Repeats until target reached

## Troubleshooting

### Tool can't detect language

Ensure you have language-specific files:
- Go: `go.mod`
- Python: `setup.py`, `pyproject.toml`, or `requirements.txt`
- JavaScript/TypeScript: `package.json`
- Java: `pom.xml` or `build.gradle`
- Swift: `Package.swift`

### Coverage tools not found

Install language-specific tools:

```bash
# Python
pip install pytest pytest-cov

# JavaScript/TypeScript (in your project)
npm install --save-dev jest

# Java - add JaCoCo to pom.xml or build.gradle
```

### Tests fail validation

- Check the error in the state file: `.coverage-agent-state.json`
- The tool will attempt auto-fix
- Some issues may require manual intervention

## Next Steps

- Read the full [README.md](README.md) for detailed documentation
- Check the [Architecture section](README.md#architecture) to understand how it works
- Contribute improvements or report issues on GitHub

## Example Session

```bash
# Set API key
export ANTHROPIC_API_KEY="sk-ant-..."

# Run agent
./test-coverage-agent -project ~/my-go-app -target 85

# Output shows progress...
# Stop with Ctrl+C if needed

# Resume later
./test-coverage-agent -project ~/my-go-app -resume

# Final output when done:
# 🎉 Target coverage of 85.00% achieved!
# Final coverage: 87.30%
# Tests generated: 15
# Tests fixed: 3
```

Happy testing! 🧪
