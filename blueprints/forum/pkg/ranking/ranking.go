// Package ranking provides sorting algorithms for forum content.
package ranking

import (
	"math"
	"time"
)

// HotScore calculates the "hot" ranking score based on Reddit's algorithm.
// Higher scores mean the content should appear higher in the feed.
// The algorithm combines score (upvotes - downvotes) with age.
func HotScore(score int, createdAt time.Time) float64 {
	// Logarithmic order of magnitude
	order := math.Log10(math.Max(math.Abs(float64(score)), 1))

	// Sign: 1 for positive, -1 for negative, 0 for zero
	sign := 0.0
	if score > 0 {
		sign = 1
	} else if score < 0 {
		sign = -1
	}

	// Seconds since epoch (newer posts get higher base score)
	epoch := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	seconds := createdAt.Sub(epoch).Seconds()

	// Combine: log10(score) + time/45000
	// 45000 seconds = 12.5 hours (half-life for decay)
	return sign*order + seconds/45000
}

// BestScore calculates Wilson score confidence interval.
// This balances high scores with number of votes.
// A post with 10/10 upvotes scores higher than 100/110.
func BestScore(upvotes, downvotes int) float64 {
	n := float64(upvotes + downvotes)
	if n == 0 {
		return 0
	}

	z := 1.96 // 95% confidence
	phat := float64(upvotes) / n

	// Wilson score lower bound
	numerator := phat + z*z/(2*n) - z*math.Sqrt((phat*(1-phat)+z*z/(4*n))/n)
	denominator := 1 + z*z/n

	return numerator / denominator
}

// ControversialScore calculates controversy score.
// Posts with balanced upvotes/downvotes (divisive) score higher.
// High magnitude + balanced ratio = most controversial.
func ControversialScore(upvotes, downvotes int) float64 {
	total := float64(upvotes + downvotes)
	if total == 0 {
		return 0
	}

	// Balance factor: posts near 50/50 split score highest
	balance := math.Min(float64(upvotes), float64(downvotes))

	// Magnitude: more votes = more controversial
	magnitude := float64(upvotes + downvotes)

	return balance * magnitude
}

// RisingScore calculates the "rising" score.
// New posts with growing scores rank higher.
func RisingScore(score int, createdAt time.Time) float64 {
	age := time.Since(createdAt).Hours()

	// Only consider posts less than 12 hours old
	if age > 12 {
		return 0
	}

	// Score divided by age^1.5 (decay factor)
	return float64(score) / math.Pow(age+2, 1.5)
}
