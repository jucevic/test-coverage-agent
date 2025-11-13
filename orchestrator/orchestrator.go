package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/tablev/test-coverage-agent/claude"
	"github.com/tablev/test-coverage-agent/config"
	"github.com/tablev/test-coverage-agent/coverage"
	"github.com/tablev/test-coverage-agent/git"
	"github.com/tablev/test-coverage-agent/testgen"
)

// Orchestrator manages the test generation workflow
type Orchestrator struct {
	config    *config.Config
	state     *config.State
	analyzer  coverage.Analyzer
	generator *testgen.Generator
	validator *testgen.Validator
	gitMgr    *git.Manager
}

// New creates a new orchestrator
func New(cfg *config.Config) (*Orchestrator, error) {
	// Detect project language
	analyzer, err := coverage.DetectProjectLanguage(cfg.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to detect project language: %w", err)
	}

	fmt.Printf("Detected language: %s\n", analyzer.GetLanguageName())

	// Create state
	state := config.NewState(cfg.ProjectPath, cfg.TargetCoverage, analyzer.GetLanguageName())

	// Create components
	generator := testgen.NewGenerator(cfg.ClaudeAPIKey, analyzer)
	validator := testgen.NewValidator(analyzer)
	gitMgr := git.NewManager(cfg.ProjectPath)

	return &Orchestrator{
		config:    cfg,
		state:     state,
		analyzer:  analyzer,
		generator: generator,
		validator: validator,
		gitMgr:    gitMgr,
	}, nil
}

// LoadState loads a previous state
func (o *Orchestrator) LoadState() error {
	state, err := config.LoadState(o.config.StateFile)
	if err != nil {
		return err
	}

	o.state = state
	return nil
}

// SaveState saves the current state
func (o *Orchestrator) SaveState() error {
	return o.state.SaveState(o.config.StateFile)
}

// Run executes the main orchestration loop
func (o *Orchestrator) Run(ctx context.Context) error {
	// Create a git branch for this session if git is available
	if o.gitMgr.IsEnabled() {
		branchName := fmt.Sprintf("test-coverage-agent-%s", time.Now().Format("20060102-150405"))
		if err := o.gitMgr.CreateBranchForSession(branchName); err != nil {
			fmt.Printf("Warning: Could not create git branch: %v\n", err)
		} else {
			fmt.Printf("Created git branch: %s\n", branchName)
		}
	}

	// Run initial coverage analysis to show starting point
	fmt.Println("\nAnalyzing current test coverage...")
	initialReport, err := o.analyzer.RunCoverage(o.config.ProjectPath)
	if err != nil {
		return fmt.Errorf("failed to run initial coverage analysis: %w", err)
	}

	o.state.AddCoverageSnapshot(initialReport.TotalCoverage)
	fmt.Printf("\n✓ Initial Coverage: %.2f%%\n", initialReport.TotalCoverage)
	fmt.Printf("  Target Coverage:  %.2f%%\n", o.config.TargetCoverage)

	coverageGap := o.config.TargetCoverage - initialReport.TotalCoverage
	if coverageGap > 0 {
		fmt.Printf("  Coverage Gap:     %.2f%% (need to add)\n", coverageGap)

		// Check if test generation is needed
		if !o.state.NeedsTestGeneration() {
			fmt.Printf("\n✅ Coverage already meets target!\n")
			fmt.Printf("No test generation needed at this time.\n")
			return o.SaveState()
		}
		fmt.Printf("\n🔧 Test generation needed (coverage below %.2f%% threshold)\n",
			o.config.TargetCoverage)
	} else {
		fmt.Printf("\n🎉 Target coverage already achieved!\n")
		fmt.Printf("Current coverage (%.2f%%) meets or exceeds target (%.2f%%)\n",
			initialReport.TotalCoverage, o.config.TargetCoverage)
		return o.SaveState()
	}
	fmt.Println()

	// Main loop
	for o.state.CurrentIteration < o.config.MaxIterations {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			fmt.Println("\nStopping and saving state...")
			return o.SaveState()
		default:
		}

		// Check for rate limiting
		if shouldWait, waitDuration := o.state.ShouldWaitForRateLimit(); shouldWait {
			fmt.Printf("\nRate limit reached. Waiting until %v (%v)...\n",
				o.state.RateLimitResetTime.Format(time.RFC3339),
				waitDuration)

			// Save state before waiting
			if err := o.SaveState(); err != nil {
				return fmt.Errorf("failed to save state: %w", err)
			}

			// Wait with context support
			timer := time.NewTimer(waitDuration)
			select {
			case <-ctx.Done():
				timer.Stop()
				return o.SaveState()
			case <-timer.C:
				fmt.Println("Rate limit reset. Resuming...")
			}
		}

		o.state.CurrentIteration++
		fmt.Printf("\n=== Iteration %d ===\n", o.state.CurrentIteration)

		// Run coverage analysis
		report, err := o.analyzer.RunCoverage(o.config.ProjectPath)
		if err != nil {
			return fmt.Errorf("failed to run coverage analysis: %w", err)
		}

		o.state.AddCoverageSnapshot(report.TotalCoverage)
		fmt.Printf("Current Coverage: %.2f%% / Target: %.2f%%\n",
			report.TotalCoverage, o.config.TargetCoverage)

		// Check if we've reached the target
		if report.TotalCoverage >= o.config.TargetCoverage {
			fmt.Printf("\n🎉 Target coverage of %.2f%% achieved!\n", o.config.TargetCoverage)
			fmt.Printf("Final coverage: %.2f%%\n", report.TotalCoverage)
			fmt.Printf("Tests generated: %d\n", len(o.state.GeneratedTests))
			fmt.Printf("Tests fixed: %d\n", len(o.state.FixedTests))
			return o.SaveState()
		}

		// Find files that need coverage improvement
		workItems := o.prioritizeWorkItems(report)
		if len(workItems) == 0 {
			fmt.Println("No more files to improve coverage for.")
			return o.SaveState()
		}

		// Process the highest priority file
		workItem := workItems[0]
		fmt.Printf("\nProcessing: %s (current coverage: %.2f%%)\n",
			workItem.SourceFile, workItem.CurrentCoverage)

		// Process the file
		if err := o.processFile(ctx, workItem, report); err != nil {
			// Check if it's a rate limit error
			if rateLimitErr, ok := err.(*claude.RateLimitError); ok {
				fmt.Printf("Rate limit hit: %v\n", rateLimitErr)
				o.state.SetRateLimitReset(rateLimitErr.ResetTime)

				// Save state and continue to next iteration (which will wait)
				if saveErr := o.SaveState(); saveErr != nil {
					return fmt.Errorf("failed to save state: %w", saveErr)
				}
				continue
			}

			// Other errors
			fmt.Printf("Error processing file: %v\n", err)
			o.state.MarkFileFailed(workItem.SourceFile, err.Error())
		}

		// Save state after each iteration
		if err := o.SaveState(); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}

		// Print progress
		fmt.Printf("\n%s\n", o.state.GetProgress())
	}

	fmt.Printf("\nReached maximum iterations (%d)\n", o.config.MaxIterations)
	fmt.Printf("Final coverage: %.2f%% / Target: %.2f%%\n",
		o.state.CurrentCoverage, o.config.TargetCoverage)

	return o.SaveState()
}

// WorkItem represents a file that needs test coverage
type WorkItem struct {
	SourceFile      string
	TestFile        string
	CurrentCoverage float64
	UncoveredLines  []int
	Priority        int
	Exists          bool
}

// prioritizeWorkItems creates a prioritized list of files to work on
func (o *Orchestrator) prioritizeWorkItems(report *coverage.CoverageReport) []WorkItem {
	var items []WorkItem

	for _, sourceFile := range report.UncoveredFiles {
		// Skip if already processed
		if o.state.IsFileProcessed(sourceFile) {
			continue
		}

		// Skip if in failed files
		if _, failed := o.state.FailedFiles[sourceFile]; failed {
			continue
		}

		testFile := o.analyzer.GetTestFilePath(sourceFile)
		uncoveredLines := report.UncoveredLines[sourceFile]
		currentCoverage := report.FileCoverage[sourceFile]

		// Calculate priority (lower coverage = higher priority)
		priority := int(100 - currentCoverage)

		// Check if test file exists
		testExists := fileExists(testFile)

		items = append(items, WorkItem{
			SourceFile:      sourceFile,
			TestFile:        testFile,
			CurrentCoverage: currentCoverage,
			UncoveredLines:  uncoveredLines,
			Priority:        priority,
			Exists:          testExists,
		})
	}

	// Sort by priority (higher priority first)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Priority > items[j].Priority
	})

	return items
}

// processFile processes a single file (generate or improve tests)
func (o *Orchestrator) processFile(ctx context.Context, item WorkItem, report *coverage.CoverageReport) error {
	// Check for cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	var testFile string
	var err error

	if !item.Exists {
		// Generate new test
		fmt.Println("  Generating new test file...")
		if !o.config.DryRun {
			o.state.RecordAPICall()
			testFile, err = o.generator.GenerateTestForFile(
				o.config.ProjectPath,
				item.SourceFile,
				item.UncoveredLines,
			)
			if err != nil {
				return fmt.Errorf("failed to generate test: %w", err)
			}
			o.state.AddGeneratedTest(testFile)
		} else {
			fmt.Println("  [DRY RUN] Would generate test file")
			testFile = item.TestFile
		}
	} else {
		// Improve existing test
		fmt.Println("  Improving existing test file...")
		if !o.config.DryRun {
			o.state.RecordAPICall()
			testFile, err = o.generator.ImproveExistingTest(
				o.config.ProjectPath,
				item.SourceFile,
				item.TestFile,
				item.UncoveredLines,
			)
			if err != nil {
				return fmt.Errorf("failed to improve test: %w", err)
			}
			o.state.AddFixedTest(testFile)
		} else {
			fmt.Println("  [DRY RUN] Would improve test file")
			testFile = item.TestFile
		}
	}

	// Validate the test
	if !o.config.DryRun {
		fmt.Println("  Validating test...")
		result, err := o.validator.ValidateAndRetry(
			o.config.ProjectPath,
			testFile,
			o.generator,
			2, // max 2 retries
		)

		if err != nil {
			return fmt.Errorf("validation error: %w", err)
		}

		if !result.Success {
			fmt.Printf("  ❌ Test validation failed: %s\n", result.ErrorMessage)
			o.state.MarkFileFailed(item.SourceFile, result.ErrorMessage)
			return nil
		}

		fmt.Println("  ✅ Test validation successful")

		// Commit to git if enabled
		if o.gitMgr.IsEnabled() {
			fmt.Println("  Committing to git...")
			coverageGain := 0.0 // We'd need to re-run coverage to know this
			if err := o.gitMgr.CreateSafetyCommit(testFile, coverageGain); err != nil {
				fmt.Printf("  Warning: Failed to commit: %v\n", err)
			}
		}
	}

	// Mark file as processed
	o.state.MarkFileProcessed(item.SourceFile)

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	absPath := path
	if !filepath.IsAbs(path) {
		var err error
		absPath, err = filepath.Abs(path)
		if err != nil {
			return false
		}
	}
	_, err := os.Stat(absPath)
	return err == nil
}
