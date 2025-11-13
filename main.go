package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/tablev/test-coverage-agent/config"
	"github.com/tablev/test-coverage-agent/orchestrator"
)

func main() {
	// CLI flags
	var (
		projectPath    = flag.String("project", ".", "Path to the project to analyze")
		targetCoverage = flag.Float64("target", 80.0, "Target code coverage percentage (0-100)")
		stateFile      = flag.String("state", ".coverage-agent-state.json", "State file for pause/resume")
		dryRun         = flag.Bool("dry-run", false, "Preview actions without making changes")
		resume         = flag.Bool("resume", false, "Resume from previous state")
		maxIterations  = flag.Int("max-iterations", 100, "Maximum number of test generation iterations")
		claudeAPIKey   = flag.String("api-key", "", "Claude API key (or set ANTHROPIC_API_KEY env var)")
	)

	flag.Parse()

	// Validate inputs
	if *targetCoverage < 0 || *targetCoverage > 100 {
		fmt.Fprintf(os.Stderr, "Error: target coverage must be between 0 and 100\n")
		os.Exit(1)
	}

	// Get API key from flag or environment
	apiKey := *claudeAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "Error: Claude API key required (use -api-key flag or ANTHROPIC_API_KEY env var)\n")
		os.Exit(1)
	}

	// Load or create configuration
	cfg := &config.Config{
		ProjectPath:    *projectPath,
		TargetCoverage: *targetCoverage,
		StateFile:      *stateFile,
		DryRun:         *dryRun,
		MaxIterations:  *maxIterations,
		ClaudeAPIKey:   apiKey,
	}

	// Create orchestrator
	orch, err := orchestrator.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing orchestrator: %v\n", err)
		os.Exit(1)
	}

	// Load state if resuming
	if *resume {
		if err := orch.LoadState(); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading state: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Resuming from previous state...")
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n\nReceived interrupt signal. Saving state and shutting down gracefully...")
		cancel()
	}()

	// Run the orchestrator
	fmt.Printf("Starting Test Coverage Agent\n")
	fmt.Printf("Project: %s\n", cfg.ProjectPath)
	fmt.Printf("Target Coverage: %.2f%%\n", cfg.TargetCoverage)
	fmt.Printf("Max Iterations: %d\n", cfg.MaxIterations)
	if cfg.DryRun {
		fmt.Println("DRY RUN MODE - No changes will be made")
	}
	fmt.Println("Press Ctrl+C to pause and save state")
	fmt.Println("=====================================\n")

	if err := orch.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "\nError during execution: %v\n", err)

		// Save state before exiting
		if saveErr := orch.SaveState(); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Error saving state: %v\n", saveErr)
		}

		os.Exit(1)
	}

	fmt.Println("\n=====================================")
	fmt.Println("Test Coverage Agent completed successfully!")
}
