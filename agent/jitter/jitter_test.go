package jitter

import (
	"math"
	"testing"
	"time"

	"sOPown3d/shared"
)

// TestNewJitterCalculator_ValidConfig tests creating a jitter calculator with valid config
func TestNewJitterCalculator_ValidConfig(t *testing.T) {
	config := shared.JitterConfig{
		MinSeconds: 1.0,
		MaxSeconds: 5.0,
	}

	jc, err := NewJitterCalculator(config)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if jc == nil {
		t.Fatal("Expected non-nil JitterCalculator")
	}

	// Check that mean is calculated correctly (midpoint)
	expectedMean := (config.MinSeconds + config.MaxSeconds) / 2.0
	if jc.mean != expectedMean {
		t.Errorf("Expected mean=%.2f, got: %.2f", expectedMean, jc.mean)
	}

	// Check that stdDev is calculated correctly (range / 6)
	expectedStdDev := (config.MaxSeconds - config.MinSeconds) / 6.0
	if math.Abs(jc.stdDev-expectedStdDev) > 0.001 {
		t.Errorf("Expected stdDev=%.4f, got: %.4f", expectedStdDev, jc.stdDev)
	}
}

// TestNewJitterCalculator_InvalidConfig tests validation of invalid configs
func TestNewJitterCalculator_InvalidConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      shared.JitterConfig
		expectError bool
	}{
		{
			name: "Negative min",
			config: shared.JitterConfig{
				MinSeconds: -1.0,
				MaxSeconds: 5.0,
			},
			expectError: true,
		},
		{
			name: "Zero min",
			config: shared.JitterConfig{
				MinSeconds: 0.0,
				MaxSeconds: 5.0,
			},
			expectError: true,
		},
		{
			name: "Max less than min",
			config: shared.JitterConfig{
				MinSeconds: 10.0,
				MaxSeconds: 5.0,
			},
			expectError: true,
		},
		{
			name: "Max equal to min",
			config: shared.JitterConfig{
				MinSeconds: 5.0,
				MaxSeconds: 5.0,
			},
			expectError: true,
		},
		{
			name: "Valid config",
			config: shared.JitterConfig{
				MinSeconds: 1.0,
				MaxSeconds: 10.0,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jc, err := NewJitterCalculator(tt.config)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				if jc != nil {
					t.Errorf("Expected nil JitterCalculator on error, got non-nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if jc == nil {
					t.Errorf("Expected non-nil JitterCalculator")
				}
			}
		})
	}
}

// TestNext_WithinBounds tests that all generated values are within [min, max]
func TestNext_WithinBounds(t *testing.T) {
	config := shared.JitterConfig{
		MinSeconds: 1.0,
		MaxSeconds: 5.0,
	}

	jc, err := NewJitterCalculator(config)
	if err != nil {
		t.Fatalf("Failed to create jitter calculator: %v", err)
	}

	// Generate 1000 samples and verify all are within bounds
	numSamples := 1000
	for i := 0; i < numSamples; i++ {
		jitter := jc.Next()
		jitterSeconds := jitter.Seconds()

		if jitterSeconds < config.MinSeconds {
			t.Errorf("Sample %d: jitter %.4fs is below min %.4fs", i, jitterSeconds, config.MinSeconds)
		}
		if jitterSeconds > config.MaxSeconds {
			t.Errorf("Sample %d: jitter %.4fs is above max %.4fs", i, jitterSeconds, config.MaxSeconds)
		}
	}
}

// TestNext_GaussianDistribution tests that values follow Gaussian distribution
func TestNext_GaussianDistribution(t *testing.T) {
	config := shared.JitterConfig{
		MinSeconds: 5.0,
		MaxSeconds: 15.0,
	}

	jc, err := NewJitterCalculator(config)
	if err != nil {
		t.Fatalf("Failed to create jitter calculator: %v", err)
	}

	// Generate many samples
	numSamples := 10000
	samples := make([]float64, numSamples)
	var sum float64

	for i := 0; i < numSamples; i++ {
		jitter := jc.Next()
		jitterSeconds := jitter.Seconds()
		samples[i] = jitterSeconds
		sum += jitterSeconds
	}

	// Calculate sample mean
	sampleMean := sum / float64(numSamples)
	expectedMean := (config.MinSeconds + config.MaxSeconds) / 2.0

	// Mean should be close to expected (within 5%)
	meanDiff := math.Abs(sampleMean - expectedMean)
	maxMeanDiff := expectedMean * 0.05 // 5% tolerance
	if meanDiff > maxMeanDiff {
		t.Errorf("Sample mean %.4f differs too much from expected mean %.4f (diff: %.4f, max allowed: %.4f)",
			sampleMean, expectedMean, meanDiff, maxMeanDiff)
	}

	// Calculate sample standard deviation
	var sumSquaredDiff float64
	for _, sample := range samples {
		diff := sample - sampleMean
		sumSquaredDiff += diff * diff
	}
	sampleStdDev := math.Sqrt(sumSquaredDiff / float64(numSamples))

	// For Gaussian distribution, most values should be within 2 standard deviations
	// Count how many samples fall within 2σ of the mean
	within2Sigma := 0
	for _, sample := range samples {
		if math.Abs(sample-sampleMean) <= 2*sampleStdDev {
			within2Sigma++
		}
	}

	// In a normal distribution, ~95% of values fall within 2σ
	// We'll accept 90-98% as reasonable (accounting for clamping at boundaries)
	percentage := float64(within2Sigma) / float64(numSamples) * 100
	if percentage < 90.0 || percentage > 98.0 {
		t.Errorf("Expected 90-98%% of samples within 2σ, got %.2f%%", percentage)
	}

	t.Logf("Distribution stats: mean=%.4f (expected: %.4f), stdDev=%.4f, within2σ=%.2f%%",
		sampleMean, expectedMean, sampleStdDev, percentage)
}

// TestNext_Uniqueness tests that consecutive calls produce different values
func TestNext_Uniqueness(t *testing.T) {
	config := shared.JitterConfig{
		MinSeconds: 1.0,
		MaxSeconds: 10.0,
	}

	jc, err := NewJitterCalculator(config)
	if err != nil {
		t.Fatalf("Failed to create jitter calculator: %v", err)
	}

	// Generate several samples and count unique values
	numSamples := 100
	seen := make(map[time.Duration]bool)

	for i := 0; i < numSamples; i++ {
		jitter := jc.Next()
		seen[jitter] = true
	}

	// With proper randomization, we should get many unique values
	// Let's require at least 90% uniqueness
	minUniquePercentage := 90.0
	actualPercentage := float64(len(seen)) / float64(numSamples) * 100

	if actualPercentage < minUniquePercentage {
		t.Errorf("Expected at least %.0f%% unique values, got %.2f%% (%d unique out of %d)",
			minUniquePercentage, actualPercentage, len(seen), numSamples)
	}

	t.Logf("Generated %d unique values out of %d samples (%.2f%%)",
		len(seen), numSamples, actualPercentage)
}

// TestNext_TimeDuration tests that Next returns proper time.Duration
func TestNext_TimeDuration(t *testing.T) {
	config := shared.JitterConfig{
		MinSeconds: 1.0,
		MaxSeconds: 2.0,
	}

	jc, err := NewJitterCalculator(config)
	if err != nil {
		t.Fatalf("Failed to create jitter calculator: %v", err)
	}

	jitter := jc.Next()

	// Verify it's a valid time.Duration
	if jitter <= 0 {
		t.Errorf("Expected positive duration, got: %v", jitter)
	}

	// Verify it's in the expected range
	expectedMin := time.Duration(config.MinSeconds * float64(time.Second))
	expectedMax := time.Duration(config.MaxSeconds * float64(time.Second))

	if jitter < expectedMin {
		t.Errorf("Jitter %v is less than minimum %v", jitter, expectedMin)
	}
	if jitter > expectedMax {
		t.Errorf("Jitter %v is greater than maximum %v", jitter, expectedMax)
	}
}

// TestGetStats tests the statistics string output
func TestGetStats(t *testing.T) {
	config := shared.JitterConfig{
		MinSeconds: 5.0,
		MaxSeconds: 15.0,
	}

	jc, err := NewJitterCalculator(config)
	if err != nil {
		t.Fatalf("Failed to create jitter calculator: %v", err)
	}

	stats := jc.GetStats()

	// Verify the string contains expected values
	expectedSubstrings := []string{
		"5.00s",  // min
		"15.00s", // max
		"10.00s", // mean
		"1.67s",  // stdDev
	}

	for _, substr := range expectedSubstrings {
		if !contains(stats, substr) {
			t.Errorf("Expected stats string to contain '%s', got: %s", substr, stats)
		}
	}

	t.Logf("Stats output: %s", stats)
}

// TestEdgeCases_SmallRange tests jitter with very small ranges
func TestEdgeCases_SmallRange(t *testing.T) {
	config := shared.JitterConfig{
		MinSeconds: 0.1,
		MaxSeconds: 0.2,
	}

	jc, err := NewJitterCalculator(config)
	if err != nil {
		t.Fatalf("Failed to create jitter calculator: %v", err)
	}

	// Generate samples and verify they're all within bounds
	for i := 0; i < 100; i++ {
		jitter := jc.Next()
		jitterSeconds := jitter.Seconds()

		if jitterSeconds < config.MinSeconds || jitterSeconds > config.MaxSeconds {
			t.Errorf("Sample %d: jitter %.4fs is out of bounds [%.4f, %.4f]",
				i, jitterSeconds, config.MinSeconds, config.MaxSeconds)
		}
	}
}

// TestEdgeCases_LargeRange tests jitter with very large ranges
func TestEdgeCases_LargeRange(t *testing.T) {
	config := shared.JitterConfig{
		MinSeconds: 1.0,
		MaxSeconds: 300.0, // 5 minutes
	}

	jc, err := NewJitterCalculator(config)
	if err != nil {
		t.Fatalf("Failed to create jitter calculator: %v", err)
	}

	// Generate samples and verify they're all within bounds
	for i := 0; i < 100; i++ {
		jitter := jc.Next()
		jitterSeconds := jitter.Seconds()

		if jitterSeconds < config.MinSeconds || jitterSeconds > config.MaxSeconds {
			t.Errorf("Sample %d: jitter %.4fs is out of bounds [%.4f, %.4f]",
				i, jitterSeconds, config.MinSeconds, config.MaxSeconds)
		}
	}
}

// Benchmark_Next benchmarks the performance of Next()
func BenchmarkNext(b *testing.B) {
	config := shared.JitterConfig{
		MinSeconds: 1.0,
		MaxSeconds: 10.0,
	}

	jc, err := NewJitterCalculator(config)
	if err != nil {
		b.Fatalf("Failed to create jitter calculator: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = jc.Next()
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
