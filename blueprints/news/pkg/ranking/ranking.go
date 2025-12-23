package ranking

import (
	"math"
	"time"
)

const (
	// gravity controls how fast scores decay with time
	// Higher values = faster decay
	gravity = 1.8

	// hourDecay is added to age to prevent division by zero
	// and smooth initial ranking
	hourDecay = 2.0
)

// HotScore calculates the HN-style ranking score.
// Formula: score / (age + 2) ^ gravity
// Higher scores and newer items rank higher.
func HotScore(points int64, createdAt time.Time) float64 {
	if points <= 0 {
		points = 1
	}
	age := time.Since(createdAt).Hours()
	if age < 0 {
		age = 0
	}
	return float64(points) / math.Pow(age+hourDecay, gravity)
}

// ControversyScore calculates how controversial an item is.
// Items with roughly equal up and down votes are more controversial.
func ControversyScore(upvotes, downvotes int64) float64 {
	if upvotes <= 0 || downvotes <= 0 {
		return 0
	}
	total := float64(upvotes + downvotes)
	ratio := float64(upvotes) / float64(downvotes)
	if ratio < 1 {
		ratio = 1 / ratio
	}
	return total / ratio
}

// ShouldRecalculate determines if hot score needs update.
func ShouldRecalculate(score int64, lastScore int64, createdAt time.Time, lastCalc time.Time) bool {
	// Always recalculate if score changed
	if score != lastScore {
		return true
	}

	age := time.Since(createdAt)

	// For items < 24h old, recalculate every 10 minutes
	if age < 24*time.Hour {
		return time.Since(lastCalc) > 10*time.Minute
	}

	// For items < 7 days old, recalculate every hour
	if age < 7*24*time.Hour {
		return time.Since(lastCalc) > time.Hour
	}

	// For older items, don't bother recalculating
	return false
}
