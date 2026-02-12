package jitter

import (
	"fmt"
	"math"
	"math/rand"
	"sOPown3d/pkg/shared"
	"time"
)

// JitterCalculator calculates random sleep durations using Gaussian distribution
type JitterCalculator struct {
	min    time.Duration
	max    time.Duration
	mean   float64
	stdDev float64
	rng    *rand.Rand
}

// NewJitterCalculator creates a new jitter calculator with Gaussian distribution
func NewJitterCalculator(config shared.JitterConfig) (*JitterCalculator, error) {
	// Validate config
	if config.MinSeconds <= 0 {
		return nil, fmt.Errorf("MinSeconds must be positive, got: %.2f", config.MinSeconds)
	}
	if config.MaxSeconds <= config.MinSeconds {
		return nil, fmt.Errorf("MaxSeconds (%.2f) must be greater than MinSeconds (%.2f)", config.MaxSeconds, config.MinSeconds)
	}

	// Calculate mean and standard deviation for Gaussian distribution
	// Mean is the midpoint between min and max
	mean := (config.MinSeconds + config.MaxSeconds) / 2.0

	// Standard deviation is set so that ~99.7% of values fall within [min, max]
	// Using 3-sigma rule: range = 6 * stdDev, so stdDev = range / 6
	stdDev := (config.MaxSeconds - config.MinSeconds) / 6.0

	// Create random number generator with current time as seed
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	return &JitterCalculator{
		min:    time.Duration(config.MinSeconds * float64(time.Second)),
		max:    time.Duration(config.MaxSeconds * float64(time.Second)),
		mean:   mean,
		stdDev: stdDev,
		rng:    rng,
	}, nil
}

// Next generates the next jitter duration using Gaussian distribution
// Values follow a normal distribution centered around the mean,
// but are clamped to stay within [min, max] range
func (jc *JitterCalculator) Next() time.Duration {
	// Generate Gaussian random value (mean=0, stdDev=1)
	gaussianValue := jc.rng.NormFloat64()

	// Scale and shift to our distribution
	// jitter = mean + (gaussianValue * stdDev)
	jitterSeconds := jc.mean + (gaussianValue * jc.stdDev)

	// Clamp to [min, max] range to ensure we never go outside bounds
	jitterSeconds = math.Max(jitterSeconds, float64(jc.min)/float64(time.Second))
	jitterSeconds = math.Min(jitterSeconds, float64(jc.max)/float64(time.Second))

	return time.Duration(jitterSeconds * float64(time.Second))
}

// GetStats returns statistical information about the jitter configuration
func (jc *JitterCalculator) GetStats() string {
	return fmt.Sprintf(
		"Jitter Config: min=%.2fs, max=%.2fs, mean=%.2fs, stdDev=%.2fs",
		float64(jc.min)/float64(time.Second),
		float64(jc.max)/float64(time.Second),
		jc.mean,
		jc.stdDev,
	)
}
