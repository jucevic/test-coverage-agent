package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config holds the application configuration
type Config struct {
	ProjectPath    string  `json:"project_path"`
	TargetCoverage float64 `json:"target_coverage"`
	StateFile      string  `json:"state_file"`
	DryRun         bool    `json:"dry_run"`
	MaxIterations  int     `json:"max_iterations"`
	ClaudeAPIKey   string  `json:"-"` // Don't serialize the API key
}

// State represents the persistent state for pause/resume functionality
type State struct {
	// Tracking
	CurrentIteration   int                `json:"current_iteration"`
	CurrentCoverage    float64            `json:"current_coverage"`
	TargetCoverage     float64            `json:"target_coverage"`
	ProcessedFiles     map[string]bool    `json:"processed_files"`     // Files we've attempted to improve
	FailedFiles        map[string]string  `json:"failed_files"`        // Files that failed with error message
	GeneratedTests     []string           `json:"generated_tests"`     // List of test files we created
	FixedTests         []string           `json:"fixed_tests"`         // List of test files we fixed
	CoverageHistory    []CoverageSnapshot `json:"coverage_history"`    // Historical coverage data

	// Rate limiting
	LastAPICall        time.Time          `json:"last_api_call"`
	APICallCount       int                `json:"api_call_count"`
	RateLimitResetTime time.Time          `json:"rate_limit_reset_time"`

	// Metadata
	ProjectPath        string             `json:"project_path"`
	StartedAt          time.Time          `json:"started_at"`
	LastUpdatedAt      time.Time          `json:"last_updated_at"`
	PausedAt           *time.Time         `json:"paused_at,omitempty"`
	Language           string             `json:"language"`
}

// CoverageSnapshot represents coverage at a point in time
type CoverageSnapshot struct {
	Timestamp  time.Time `json:"timestamp"`
	Coverage   float64   `json:"coverage"`
	Iteration  int       `json:"iteration"`
	FilesAdded int       `json:"files_added"`
}

// NewState creates a new state instance
func NewState(projectPath string, targetCoverage float64, language string) *State {
	return &State{
		CurrentIteration: 0,
		CurrentCoverage:  0.0,
		TargetCoverage:   targetCoverage,
		ProcessedFiles:   make(map[string]bool),
		FailedFiles:      make(map[string]string),
		GeneratedTests:   []string{},
		FixedTests:       []string{},
		CoverageHistory:  []CoverageSnapshot{},
		ProjectPath:      projectPath,
		StartedAt:        time.Now(),
		LastUpdatedAt:    time.Now(),
		Language:         language,
		APICallCount:     0,
	}
}

// SaveState saves the state to a JSON file
func (s *State) SaveState(filename string) error {
	s.LastUpdatedAt = time.Now()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// LoadState loads the state from a JSON file
func LoadState(filename string) (*State, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}

// AddCoverageSnapshot adds a new coverage measurement to history
func (s *State) AddCoverageSnapshot(coverage float64) {
	s.CoverageHistory = append(s.CoverageHistory, CoverageSnapshot{
		Timestamp:  time.Now(),
		Coverage:   coverage,
		Iteration:  s.CurrentIteration,
		FilesAdded: len(s.GeneratedTests) + len(s.FixedTests),
	})
	s.CurrentCoverage = coverage
}

// MarkFileProcesed marks a file as having been processed
func (s *State) MarkFileProcessed(filename string) {
	s.ProcessedFiles[filename] = true
}

// MarkFileFailed marks a file as having failed processing
func (s *State) MarkFileFailed(filename string, errorMsg string) {
	s.FailedFiles[filename] = errorMsg
}

// IsFileProcessed checks if a file has already been processed
func (s *State) IsFileProcessed(filename string) bool {
	return s.ProcessedFiles[filename]
}

// AddGeneratedTest records a newly generated test file
func (s *State) AddGeneratedTest(testFile string) {
	s.GeneratedTests = append(s.GeneratedTests, testFile)
}

// AddFixedTest records a test file that was fixed
func (s *State) AddFixedTest(testFile string) {
	s.FixedTests = append(s.FixedTests, testFile)
}

// RecordAPICall updates API call tracking for rate limit management
func (s *State) RecordAPICall() {
	s.LastAPICall = time.Now()
	s.APICallCount++
}

// SetRateLimitReset sets the time when rate limits will reset
func (s *State) SetRateLimitReset(resetTime time.Time) {
	s.RateLimitResetTime = resetTime
}

// ShouldWaitForRateLimit checks if we should wait due to rate limiting
func (s *State) ShouldWaitForRateLimit() (bool, time.Duration) {
	if s.RateLimitResetTime.IsZero() {
		return false, 0
	}

	now := time.Now()
	if now.Before(s.RateLimitResetTime) {
		return true, s.RateLimitResetTime.Sub(now)
	}

	return false, 0
}

// GetProgress returns a human-readable progress summary
func (s *State) GetProgress() string {
	return fmt.Sprintf(
		"Iteration: %d | Coverage: %.2f%% / %.2f%% | Generated: %d | Fixed: %d | Failed: %d",
		s.CurrentIteration,
		s.CurrentCoverage,
		s.TargetCoverage,
		len(s.GeneratedTests),
		len(s.FixedTests),
		len(s.FailedFiles),
	)
}

// NeedsTestGeneration checks if test generation is needed based on coverage threshold
// Returns true if coverage is below target (e.g., < 40% when target is 40%)
func (s *State) NeedsTestGeneration() bool {
	return s.CurrentCoverage < s.TargetCoverage
}
